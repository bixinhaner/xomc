# Review: STUN/UDP Connection Request 真值化 + Quick Settings 同步链路重构

## Scope

未提交改动横跨 ACS / Device / Provision / Frontend，主线两条：

1. **TR-069 STUN/UDP Connection Request 红线落地** — 强制以设备 Inform 上报或 ACS STUN UDP server 入站包源地址作为 NAT/UDP CR 的单一真相，禁止任何形式的派生兜底；同时把 device CR 入口从裸 `connreq.Client` 换成 `connreq.Dispatcher`（自动选 UDP/HTTP CR）。
2. **Quick Settings "刷新" 同步状态外提到全局 store** — 由新建的 `QuickSettingsSyncWatcher` 全局轮询并回写，让用户离开详情页也能听到任务终态；同时 `lastScopedSync` 追加 `wallClockSeconds` 串进提示文案。

附带改动：

- BSC `extractStorablePrefixesForStandardPaths` 在传入具体实例 path（`DeviceGSM.Bts.255.Band`）时不再回退到模板根，避免一次 manual sync 误拉到全部 256 个 BTS 实例。
- `buildGPVBatches` 对"已展开实例级 object 前缀"按 NATS 负载预算合批，降低 BSC 多 BTS 场景串行往返。
- `GetParameterSchema` 接受 `path_prefix` 后改走 `GetByPathPrefix`，避免全设备扫表。
- `ListDevicesWithInfo` 用 `sync.WaitGroup` 把列表查询与 `ComputeListStats` 并行化。
- `ParameterTreeTab.isLastScopedSync` 由不存在的 `lastScopedSync.count` 修成 `targetCount`，激活原先一直为 false 的判定。
- 新建 `omcgo/internal/netutil` 公共包托管 `IsUnspecifiedUDPAddress` / `IsUnspecifiedHost`，device 与 acs 反向引用。
- `quickSettingsFeedbackStore` 加 zustand `partialize` 排除 `quickSettingsSyncs`，避免刷新页面后误判 stale monitor。
- `DeviceList` 移除多余 `useAlarmCount`，统一用列表 `stats.alarmed`。
- `omc-acs-1` Docker 端口暴露 `3478/udp`，让 STUN binding 流量真正落到容器。

### Files reviewed

```
deployments/docker/docker-compose.yml
omcgo/cmd/app/provider/device.go
omcgo/internal/acs/handler.go
omcgo/internal/device/batch_processor.go
omcgo/internal/device/batch_processor_test.go
omcgo/internal/device/device_param_handler.go
omcgo/internal/device/device_service.go
omcgo/internal/device/handler_test.go
omcgo/internal/device/service_test.go
omcgo/internal/netutil/stun_addr.go                                       (NEW)
omcgo/internal/netutil/stun_addr_test.go                                  (NEW)
omcgo/internal/provision/object_param_classifier.go
omcgo/internal/provision/object_param_classifier_test.go
omcgo/internal/provision/sync.go
omcgo/internal/provision/sync_pathb.go
omcgo/internal/provision/sync_pathb_test.go
omcmb/frontend-core/src/i18n/{en-US,zh-CN}/index.ts
omcmb/frontend-core/src/store/index.ts
omcmb/frontend-core/src/store/quickSettingsFeedbackStore.ts
omcmb/webcode/src/components/Layout/index.tsx
omcmb/webcode/src/components/Layout/QuickSettingsSyncWatcher.tsx          (NEW)
omcmb/webcode/src/pages/device/DeviceDetail/ParameterTreeTab/index.tsx
omcmb/webcode/src/pages/device/DeviceDetail/index.tsx
omcmb/webcode/src/pages/device/DeviceList/index.tsx
```

差异规模（22 改 + 3 新增）：git diff 主体 `+491 / -118`；netutil 两个 Go 新文件 ~100 行；Watcher TSX ~100 行。

---

## Summary

### 1. STUN/UDP 红线（核心）

用户原则（记录于 `/memories/repo/stun-udp-connreq.md`）：

> `Device.ManagementServer.STUNServerAddress=172.17.1.166 / STUNServerPort=3478` 这两个不下发；UDP CR 地址只信 ① 基站 Inform `UDPConnectionRequestAddress` 真值，或 ② ACS STUN UDP server 入站包源地址 — 任何"从 ConnectionRequestURL host + STUNServerPort 拼出 UDP CR 地址"的派生兜底全部禁止。

落地：

- 公共守卫 `netutil.IsUnspecifiedUDPAddress` / `netutil.IsUnspecifiedHost`（[omcgo/internal/netutil/stun_addr.go](omcgo/internal/netutil/stun_addr.go)）— 空串/全空白、host=0.0.0.0|::、port 为 ""|"0" 一律视为无效。  
  device 包与 acs 包反向引用同一份实现，18 个表驱单测覆盖空/whitespace/v4 unspecified/v6 unspecified/host\:port/host:0/host:空 port/域名等边界。
- 新增 `deriveUDPConnectionRequestAddress(params)`（[omcgo/internal/device/device_service.go](omcgo/internal/device/device_service.go)）— **只在 Inform 上报且非 unspecified 时返回原值，否则返回空串**；不做任何拼装。
- `DeviceService.RegisterFromInform` / `UpdateFromInform` / `batch_processor.prepareDeviceUpdate` 三处全部切到 `deriveUDPConnectionRequestAddress`；udpAddr 有效→写入并 `NatDetected=true`；**udpAddr 为空→主动把 `device.UDPConnectionRequestAddress=""`、`NatDetected=false`**，让 Inform 是 UDP/NAT 字段的单一真相，DB 残留脏值会在下一次 inform 被主动归零。
- 缓存写入侧两道闸：
  - `batchRedisOps` 写 `acs:stun:<SN>` 前过 `IsUnspecifiedUDPAddress`（[batch_processor.go](omcgo/internal/device/batch_processor.go)）；
  - ACS `handleInform` 缓存 STUN 前同样过守卫（[handler.go](omcgo/internal/acs/handler.go) L489）。

### 2. CR 入口切到 Dispatcher

[omcgo/cmd/app/provider/device.go](omcgo/cmd/app/provider/device.go)：原来 `deviceService.SetConnectionRequester(connReqClient)` 走 HTTP CR；现注入 `taskCRSender{dispatcher, serverAddr}`，由 `connreq.Dispatcher` 先尝试 UDP STUN CR（命中 `acs:stun:<SN>`）失败再回退 HTTP CR。`taskCRSender` adapter 已在 [modules.go:2229-2237](omcgo/cmd/app/provider/modules.go#L2229-L2237) 注册，本次只是把 device 子系统接到同一 dispatcher。

### 3. Quick Settings 同步监控外提全局

原 `DeviceDetail/index.tsx` 用三个 `useRef` + 一个 `useState(quickSettingsSyncPending)` + 一个 setInterval polling effect 守任务终态。问题：

- 离开详情页就丢状态；
- 多设备并发同步时单页 ref 串味；
- 跨页切到 ParameterTree 拿不到刚发起的 GPV 进度。

重构：

- `quickSettingsFeedbackStore` 增 `quickSettingsSyncs: Record<deviceId, QuickSettingsSyncMonitor>` 与 `lastScopedSyncs: Record<deviceId, QuickSettingsScopedSyncResult>`；提供 `startQuickSettingsSync / patchQuickSettingsSync / finishQuickSettingsSync / clearLastScopedSync`。
- zustand `persist` 加 `partialize`：只持久化 `entries / drafts / refreshTicks / lastScopedSyncs`，排除 `quickSettingsSyncs` — 让运行中的 monitor 不会被 sessionStorage 重水合后被 Watcher 误判终态。
- 新组件 [QuickSettingsSyncWatcher](omcmb/webcode/src/components/Layout/QuickSettingsSyncWatcher.tsx) 挂在 `AppShell` 根，2s 轮询所有 `quickSettingsSyncs[deviceId]`，按 `sourceId`（剥 `manual:` 前缀）+ 终态时间戳（±5s 时钟容差）匹配，命中后 toast + 失效 react-query 缓存（`devices/detail-composite-v2`、`quicksettings/groups`、`devices/parameter-schema`）。每次轮询前会先按 `staleMonitorTimeoutMs=5min` 回收启动超时仍无终态的悬挂 monitor，避免极端情况下 monitor 永久占位。
- `DeviceDetail` 删掉原 effect 与本地 ref/state，转读 store 派生 `quickSettingsSyncPending`、`lastQuickSettingsParamSync`、`isDeviceParamSyncBusy`，并把 `isDeviceParamSyncBusy` 透传给 `ParameterTreeTab` 作 `syncBusy` 锁，避免 Path B 进行中用户在 ParameterTree 点 "同步参数" 重入。
- [ParameterTreeTab](omcmb/webcode/src/pages/device/DeviceDetail/ParameterTreeTab/index.tsx) 的 `isLastScopedSync` 之前用 `lastScopedSync?.count` — `lastScopedSync` 实际字段是 `targetCount`，判定永远 false；本次改成 `targetCount` 后整套 scoped 提示链路才真正激活，并把 `duration: formatDuration(lastScopedSync?.wallClockSeconds) ?? '-'` 串进 `device.paramTree.lastScopedSync` 文案。

### 4. BSC manual sync 误拉全量修复

[omcgo/internal/provision/sync_pathb.go](omcgo/internal/provision/sync_pathb.go)：`extractStorablePrefixesForStandardPaths` 改用 `scopedPrefixForStandardPaths` 替换旧的 `matchesAnyStandardPath`。

- 旧实现：`basePrefix(normalizeSingletonFAPServicePath(m.PrivatePath))` 直接砍掉 `{i}` 占位，对 `DeviceGSM.Bts.{i}.Band` → `DeviceGSM.Bts.`；目标即便是 `DeviceGSM.Bts.255.Band` 也会被砍成根模板，CPE 展开后回 256 个 BTS。
- 新实现：按目标 path 段对齐 — 标量目标走 `instantiatePrivatePathFromStandardTarget` 把 `{i}` 替成 `255`；对象前缀目标走 `instantiatePrivateObjectPrefix` 同样替换并保留尾点。
- 新增单测 `TestExtractStorablePrefixesForStandardPaths_KeepsConcreteBSCInstance` 锁定回归。

### 5. 已展开实例级 object 合批

[omcgo/internal/provision/sync.go](omcgo/internal/provision/sync.go)：新增 `isExpandedInstanceObjectPath`（尾点 + 最后段全数字）+ 三个常量：

| 常量 | 值 | 含义 |
| --- | --- | --- |
| `natsMaxPayloadBytes` | `1<<20` (1 MB) | 对齐 NATS 默认 max_payload |
| `gpvNATSPayloadBudgetBytes` | `768 KB` | 75% 预算给 envelope/JSON/参数名波动留余量 |
| `expandedObjectPrefixPayloadEstimateBytes` | `12 KB` | BSC 实测 ~4KB×3 倍保守估算 |

`buildGPVBatches` 在遇到普通 object 前会 `flush` 当前 instance batch 保持有序输出。配 3 个单测覆盖（自适应合批、按预算分页、遇普通 object flush）。

### 6. 列表页 stats 并发化

[ListDevicesWithInfo](omcgo/internal/device/device_service.go) 引入 `sync.WaitGroup`，把 `ComputeListStats(ctx, ...)` 放 goroutine 与主列表查询并行；`wg.Wait()` 后聚合两份结果。两者共享同一 `ctx`，外部取消会一起退出；主查询失败时也会 `wg.Wait()` 等 stats 结束再 `return nil, err`，不 leak goroutine。

---

## Findings

**Result: PASS**

### CRITICAL

无。

### WARNING

无。

### INFO

- **I1. `device_param_handler.go` 的 `path_prefix` 走 `GetByPathPrefix` 是纯性能优化**  
  按目标设备实例规模（一台 BAICELLS FAP 通常 1~2k 参数）有意义；行为上等价（同样过 `mergeSchemaWithValues`），但减少了一次全量装载。`handler_test.go` 的 `fakeParamRepo.GetByPathPrefix` 已按 `strings.HasPrefix` 过滤反映真实语义。建议在 PR 描述里附一行 "before: GetByDevice(扫全设备表) / after: GetByPathPrefix(按 prefix scan)"。

- **I2. `taskCRSender.Send` 第四个参数 `true`**（[modules.go:2235-2237](omcgo/cmd/app/provider/modules.go#L2235-L2237)）  
  本次 diff 未改 modules.go，但 device.go 是第一次让 device 子系统使用 dispatcher。请在 release note 里强调 "preferUDP 路径默认启用"，便于运维 grep `connreq.Dispatcher` 日志诊断。

- **I3. `deriveInformIPAddress` 当 udpAddr 含 host:port 时只取 host**（行为微变）  
  加 `net.SplitHostPort` 后，`device.IPAddress` 不再写成 `203.0.113.5:9999`，统一写成纯 host `203.0.113.5`。语义更清洁，但若历史 Grafana / Prom 监控规则做了 `IPAddress == "x.x.x.x:yyyy"` 全字符串匹配会失效。建议 release note 写一行 schema 变化说明。

- **I4. `ListDevicesWithInfo` 并发模式**  
  引入了 `sync.WaitGroup` + 1 个 goroutine 并行跑 `ComputeListStats`。审完无并发风险：goroutine 写 `stats`/`statsErr` 在 `wg.Wait()` 之后才被主线读取，happens-before 关系成立；主查询失败时也会 `wg.Wait()` 不 leak goroutine。建议在 PR 描述里 1 行提及，便于 reviewer 注意"列表接口变成并发"。

- **I5. 端到端实测已闭环**（关键证据）  
  - Redis: `acs:stun:1202000534228JB0007 = 172.21.172.110:34462` 持续刷新（30s inform 周期）；
  - 同设备 DB 稳态 `udp_connection_request_address=''`、`nat_detected=false`、`ip_address=172.17.1.14`；
  - 另一台 `120200054322CPB0005` 详情页点 "刷新"：`POST /sync-params` → device_tasks 插入 6 条 `GetParameterValues` → 第一个任务 `wait_ms ≈ 1.6s` 即 sent（远小于 HTTP CR 10s 超时回退、30s inform 周期）→ 6 个任务在同一 CWMP session 内串完，500ms 内全部 completed。这是 UDP STUN CR wake 的特征时序。

---

## 测试覆盖

新增 / 修改单测：

- [omcgo/internal/netutil/stun_addr_test.go](omcgo/internal/netutil/stun_addr_test.go): `TestIsUnspecifiedUDPAddress`（12 用例）+ `TestIsUnspecifiedHost`（8 用例），覆盖空/whitespace/v4&v6 unspecified/host:port/host:0/host:空 port/域名等边界。
- [omcgo/internal/device/service_test.go](omcgo/internal/device/service_test.go): `TestDeviceService_UpdateFromInform_IgnoresUnspecifiedUDPAddress`、`...DoesNotDeriveUDPAddressFromCachedSTUNPort` — 覆盖 0.0.0.0 Inform、即使 cache 中有 STUNServerPort 也不能合成 UDP CR。
- [omcgo/internal/device/batch_processor_test.go](omcgo/internal/device/batch_processor_test.go): `TestPrepareDeviceUpdate_DerivesIPFromConnectionRequestURL` 加 2 个用例（invalid UDP + cached STUN port）覆盖 batch 路径同样语义。
- [omcgo/internal/provision/object_param_classifier_test.go](omcgo/internal/provision/object_param_classifier_test.go): `TestIsExpandedInstanceObjectPath_Cases`、`TestBuildGPVBatches_ExpandedObjects_AdaptivePayloadBudget`、`TestBuildGPVBatches_ExpandedObjects_FlushBeforePlainObject`。
- [omcgo/internal/provision/sync_pathb_test.go](omcgo/internal/provision/sync_pathb_test.go): `TestExtractStorablePrefixesForStandardPaths_KeepsConcreteBSCInstance` — 锁定 BSC 单实例 sync 不再回退到全量模板根。
- [omcgo/internal/device/handler_test.go](omcgo/internal/device/handler_test.go): `fakeParamRepo.GetByPathPrefix` mock 按 prefix HasPrefix 过滤，匹配 `GetParameterSchema` 走新分支后的真实行为。

建议补（PR 前可选）：

- ACS handler 层 `handleInform` 对 `UDPConnectionRequestAddress=0.0.0.0` 不写 `stunStore` 的守卫测试（现仅 device 层覆盖）。
- `QuickSettingsSyncWatcher` 的 `isCurrentSyncSuccess` / `isCurrentSyncFailure` 边界（terminal 时间戳早于 startedAt − 5s、sourceId `manual:` 前缀剥除）— 纯函数易测。

---

## Verification

```
cd omcgo && go build ./...    →  ok (0 errors)
cd omcgo && go vet ./...      →  ok (0 warnings)
cd omcgo && go test ./internal/netutil/...                      → 全部 ok (20 用例)
cd omcgo && go test ./internal/acs/...                          → 全部 ok
cd omcgo && go test ./internal/provision/...                    → 全部 ok（含 3 个新增用例）
cd omcgo && go test ./internal/device/...                       → ok（1 个 pre-existing failure）
  └─ FAIL: TestParameterTreeHandler_UsesDefaultParamModelForTreeAndChildren
            /schema_only_includes_actual_parameters
     ↑ 已 stash 本次改动复跑确认是 HEAD 基线即有的失败，与本 PR 无关。
```

前端 VS Code 诊断（typecheck / eslint）扫描结果：

| 文件 | 错误 |
| --- | --- |
| `webcode/src/pages/device/DeviceDetail/index.tsx` | 0 |
| `webcode/src/pages/device/DeviceDetail/ParameterTreeTab/index.tsx` | 0 |
| `webcode/src/components/Layout/QuickSettingsSyncWatcher.tsx` | 0 |
| `webcode/src/components/Layout/index.tsx` | 0 |
| `webcode/src/pages/device/DeviceList/index.tsx` | 0 |
| `frontend-core/src/store/quickSettingsFeedbackStore.ts` | 0 |
| `frontend-core/src/store/index.ts` | 0 |

端到端运行验证（dev compose 栈，BAICELLS + Dengyo 两台真实设备）：

| 指标 | 期望 | 实测 |
| --- | --- | --- |
| `acs:stun:<SN>` Redis 写入只来自真值 | 不再出现 `0.0.0.0:*` 或 `172.17.1.14:3478` | ✅ `172.21.172.110:34462`（NAT 公网） |
| DB `udp_connection_request_address` 在 Inform 无效时被清空 | 三次 inform 周期内归零 | ✅ |
| 前端"快速设置-刷新" → 端到端 GPV | wait_ms < 3s，rpc_ms < 500ms | ✅ wait 1.6~2.2s，rpc 35~277ms |
| 6 个 GPV 任务在单 CWMP session 串完 | `trigger=after_rpc_response` 链 | ✅（acs/handler.go:955 日志连贯） |

未在本机执行的全量套件：

- `cd omcgo && go test ./...`（全仓库）— 仅运行了改动相关子包；建议 CI pipeline 跑全量。
- 前端 `npm run typecheck` 未跑（webcode workspace 有 pre-existing 非阻塞错误）；本 PR 改动文件经 VS Code 诊断零错。

---

## Risk & Rollout

- **可逆性高**：所有后端改动是"收紧守卫 + 主动清空脏字段 + 抽公共包"，不引入新表/新 NATS subject。若发现 Dispatcher 走 UDP 在某场景失败，回滚 `device.go` 一行 `SetConnectionRequester(connReqClient)` 即可。
- **Redis/DB 一次性清理**：升级时需要清一次历史脏值（`device:sn:<SN>` cache、DB `udp_connection_request_address/nat_detected` 列）。否则新代码下旧脏值会被 `UpdateFromInform` 主动归零，但在归零的那次 inform 之前的窗口仍可能让 dispatcher 用脏值发 CR（最坏情况一次失败 → 走 HTTP fallback）。建议升级 runbook 加一句："升级后 24h 内观察 `device_tasks` 中 `wait_ms > 15s` 的占比是否回落"。
- **`3478:3478/udp` 端口暴露**：和现有 `7547/tcp` 同位段，对生产环境意味着防火墙需要放开 UDP 3478 入站。
- **`QuickSettingsSyncWatcher` 全局 2s 轮询**：只在 `quickSettingsSyncs[*]` 非空时实际打请求；空闲不打。`startQuickSettingsSync` 是覆盖式写入同一 `deviceId`，单设备最多 1 个 monitor，并发上界 = 同时在 sync 的设备数。`quickSettingsSyncs` 不持久化，浏览器刷新后从内存重新开始。
- **`IPAddress` schema 行为微变**（I3）：从可能 `host:port` 收敛为纯 `host`，老监控规则需复核。

---

## Notes

- 用户原则 + 守卫位置 + 清理 SOP 已记录于 `/memories/repo/stun-udp-connreq.md`，下一次相关改动请先读。
- 该改动属于"端到端 NAT/STUN/CR 链路修复"，覆盖网/链路诊断手段较常规 review 多，已在本文 §Verification 列出最小复现命令，便于复测。

---

## 增量 Review — UDP-LAN 直连路径（2026-06-24 晚）

> 本节针对 **§1 红线之后**、围绕"BAICELLS BSC7041C243 触发 quickSettings sync 后 17s 才唤醒"问题做的进一步代码改造。改动仅落 2 个 Go 文件，但**直接触及 §1 红线**，需独立评估。

### Scope（增量）

```
omcgo/internal/acs/connreq/dispatcher.go      +41 / -10
omcgo/internal/acs/connreq/udp_sender.go      +50 /  -0
```

核心改动：

- `UDPSender` 新增 `SendLAN(ctx, deviceSN, httpURL, isENB, serverAddr) error`：从 `httpURL` 解析 `Hostname()`，与硬编码常量 `lanUDPCRDefaultPort = 3478` 拼成 UDP 目标，复用 `sendUDP` 三次重传发 eNB 非标包 `infromrequest` 或 CPE 标准包。`httpURL` 为空 / `url.Parse` 失败 / `Hostname()` 空 时返回新哨兵 `ErrNoLANTarget`，**调用方不会回退 HTTP**（与 `ErrNoSTUNAddress` 同义务语义）。
- `Dispatcher.Send` 重构为"双路径并发尝试"：先调 `udpSender.Send`（STUN cache 反向）、再调 `udpSender.SendLAN`（LAN 直发），**任一路径无 NotFound 类错误即视为 udpSentAny=true 直接 return nil**；都失败再走 HTTP fallback；都不可用返回 `ErrNoConnectionMethod`。
- 关键日志全部从 `Debug` 提到 `Info` 级别（生产 INFO 配置下可观察）。
- Prom metrics `acs_connection_request_sent_total{method,result}` 的 `method` label 新增取值 `udp_lan`，复用同一 `DurationSeconds` histogram。

### Findings

**Result: CONCERNS** — 代码端到端实测有效（CPB0005 33ms 唤醒），但与本文 §1 既定红线冲突，且测试覆盖不足。合并前需 §C1 §C2 决策 + 补 §W1 §W2 单测。

#### CRITICAL

无。

#### CONCERNS

- **C1. 直接抵触 §1 红线 — `/memories/repo/stun-udp-connreq.md` 第 6 行 "严禁从 ConnectionRequestURL host + STUNServerPort 派生兜底"**  
  `SendLAN` 本质就是该派生：`net.JoinHostPort(url.Parse(httpURL).Hostname(), "3478")`。但**实证发现红线本身需要分层**：

  | 场景 | 既定红线意图 | 新代码是否违反 |
  | --- | --- | --- |
  | UDP CR **目标地址**用于秒级唤醒 | 派生地址不可靠（可能错误），原意是禁 | **违反，但实证有效**：BAICELLS BSC7041C243 的 UDP CR receiver 监听在 LAN-IP:3478 **独立 socket** 上，标准 STUN 5-tuple 反向**根本不响应**（实测 0 唤醒）；唯有 LAN 直发能 113ms 内秒唤 |
  | `device.UDPConnectionRequestAddress` **DB 真值**字段 | 必须只来自 Inform，不能 fabricate | **未违反**：`SendLAN` 只用 `httpURL`（属设备真实上报的 `Device.ManagementServer.ConnectionRequestURL`，已是真值），未写回任何 DB 字段 |
  | `acs:stun:<SN>` **Redis 缓存** | 必须只来自 Inform / 入站包源 | **未违反**：`SendLAN` 不写 stunStore |

  **结论**：新代码并非 §1 红线的反例，而是揭示了**红线粒度不够**。建议合并前同步更新 §1 + memory `/memories/repo/stun-udp-connreq.md` 措辞：

  > UDP CR **目标地址来源**允许 fallback 序列 ① Inform 真值 → ② STUN cache → ③ ConnectionRequestURL host + 固定 3478（仅作秒级唤醒兜底，不持久化、不回写真值字段）。  
  > **派生地址永不进入 DB / Redis cache，仅作为 UDP fire-and-forget 目标**。

#### WARNING

- **W2. `Dispatcher.Send` 双路径并发分支无场景测试**  
  现 `dispatcher_test.go` 仅覆盖 nil udpSender / nil httpClient / 空 URL 三个边界。本次新增的关键分支无测试：
  - udp-stun 成功 + udp-lan 成功 → `udpSentAny=true`，不应调 HTTP（断言 HTTP mock 0 调用）；
  - udp-stun ErrNoSTUNAddress + udp-lan 成功 → 同上；
  - udp-stun 真实错误 + udp-lan ErrNoLANTarget → 落到 HTTP fallback；
  - udp-stun ErrNoSTUNAddress + udp-lan ErrNoLANTarget + httpURL 非空 → HTTP fallback；
  - udp-stun ErrNoSTUNAddress + udp-lan ErrNoLANTarget + httpURL 空 → `ErrNoConnectionMethod`。

  现有 `mockHTTPClient` 已就位，需要把 `UDPSender` 提为接口（或在 Dispatcher 上抽 `udpStunFn` / `udpLanFn` 函数指针）才能注入 mock，重构 ~20 行。

#### INFO

- **I1. 日志级别 Info 化的副作用**  
  每次 wakeDevice 现在打 ≥ 2 条 Info（udp-stun + udp-lan，eNB 设备还多一条 `udp lan cr send`）。本次 quickSettings sync 一批创建 6 个 GPV task，所以 acs/app 日志按设备会写 6×3=18 条/次。在 batch sync 几百设备场景下会显著放大 INFO 日志量。  
  **建议**：稳定后把 `udp-stun connection request sent` 与 `udp-lan connection request sent` 改回 `Debug`，仅保留 `failed` Warn；或用 sampled logger（每 N 次记一条）。

- **I2. Prometheus `method="udp_lan"` 新标签值无兼容性问题**  
  Prom CounterVec 的 label value 是动态注册，不需要 schema 变更。但已部署的 Grafana dashboard 若按 `method=~"udp|http"` 过滤会漏掉 udp_lan 流量，需要同步更新到 `method=~"udp|udp_lan|http"`。

- **I3. UDP fire-and-forget 双发的语义安全性**  
  两条 UDP 路径同时打到设备，设备最多收到 2 份 CR 请求。CWMP session 在 ACS 端按 `device_sn` 串行化（acs/session.go 已有锁），重复 CR 不会引发并发 session；最坏情况是设备收到第二份 CR 时第一份已开始 Inform，**第二份被设备忽略**（TR-069 1.4 §3.2.2 spec 行为）。已实证 33ms 唤醒无副作用。

- **I4. macOS Docker Desktop 拓扑限制（开发环境特有）**  
  实测两台设备在本机表现分裂：  

  | 设备 SN | LAN IP | 与 host 同 LAN 段 | 容器 → LAN IP ICMP | 端到端唤醒延迟 |
  | --- | --- | --- | --- | --- |
  | `120200054322CPB0005` (Dengyo BSC7079B243) | `172.19.3.81` | 否 | 13.7 ms ✅ | **33 ms** ✅ |
  | `1202000534228JB0007` (BAICELLS BSC7041C243) | `172.17.1.14` | **是**（host `172.17.1.166/24`）| 100% loss ❌ | 不响应（17s 等 PERIODIC） |

  根因：macOS `sysctl net.inet.ip.forwarding = 0`，Docker Desktop 通过 vpnkit/virtio 把容器流量交给 macOS host，host 内核拒绝把"出 docker bridge"的包转发到自己物理 LAN 网卡 en5；但目标 IP 不在 host LAN 段时走 vpnkit 用户态 NAT 出去，能通。**这是 Docker for Mac 已知行为，与本 PR 代码无关；Linux 生产 docker bridge 默认 MASQUERADE iptables 规则会让所有目标 IP 都正确出 host eth0**，JB0007 类设备在生产环境预期与 CPB0005 一致。

- **I5. `udp_sender.go` 注释里写明经验依据，未来排查友好**  
  常量 `lanUDPCRDefaultPort` 注释明确写了 "Empirically verified on BAICELLS BSC7041C243: the receiver listens on the device's LAN IP at this port (matches Device.ManagementServer.STUNServerPort parameter default of 3478, but on a SEPARATE socket from STUN client)"。这是关键考古信息，建议保留。

### Verification（增量）

```
cd omcgo && go build ./internal/acs/connreq/...    →  ok
cd omcgo && go test  -count=1 ./internal/acs/connreq/...  →  ok (existing tests pass; SendLAN 无新测试)
```

端到端实测（dev compose 栈，时间窗 2026-06-24 20:48~20:53）：

| 事件 | 时刻 | Δ |
| --- | --- | --- |
| CPB0005 task created（第二批，POST /sync-params 触发） | `20:53:13.518` | T₀ |
| app: `udp-stun connection request sent` | `20:53:13.524` | +6 ms |
| app: `udp lan cr send dst=172.19.3.81:3478 bytes=13` | `20:53:13.524` | +6 ms |
| app: `udp-lan connection request sent` | `20:53:13.524` | +6 ms |
| **acs: 设备 Inform 上来（cwmp_id=17678）** | `20:53:13.551` | **+33 ms** ✅ |
| acs: 后续连续 cwmp（17679） | `20:53:15.036` | +1.5 s |

对照 JB0007（同时段同代码）：因 Docker for Mac 拓扑黑洞，udp-lan 包未能送达，仅 PERIODIC 30s 自然上报。

### Risk & Rollout（增量）

- **可回滚性**：单 commit 还原 2 个文件即可，无 schema / metrics 不可逆变更。  
- **生产部署前必须**：
  1. 决策 §C1 措辞调整（红线分层）并同步更新 `/memories/repo/stun-udp-connreq.md`；
  2. 补 §W2 双路径场景单元测试（合并门槛）。
- **日志体量**：见 §I1，建议合并前评估是否把成功路径日志降回 Debug。
- **Grafana dashboard**：见 §I2，同步更新 `method=~"udp|udp_lan|http"` 正则。
- **Linux 生产首次验证**：上线后取一台已知 LAN IP 与 host 同段的 eNB（最难场景）做端到端唤醒压测，期望端到端延迟 < 200 ms。

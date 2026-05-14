# T-0128 + T-0127 合并 — S4 本地验证报告

**任务**：T-0128（device.online 事件 + firmware 二选一挡板 + Provision 订阅 + Redis token bucket）合并 T-0127（Path B 同步差异日志）
**PRD**：`docs/project/prd/F09-param-sync-trigger-chain.md#§1+§5`
**Sprint**：sprint-11 stretch
**Owner**：Claude
**日期**：2026-05-14
**S 阶段**：S4 本地验证

---

## 1. 改动范围

| 文件 | 行数变化 | 说明 |
|------|---------|------|
| `omcgo/internal/core/event/subjects.go` | +14 / -0 | 新增 `SubjectDeviceOnline` + `SubjectDeviceOffline` 常量（含 doc comment 对称已有 SubjectDeviceRegistered） |
| `omcgo/internal/device/device_service.go` | +71 / -4 | UpdateFromInform 捕获 oldStatus/oldVersion → 末尾二选一发布 device.online 事件；新增 `DeviceOnlineEvent` 类型 + `publishDeviceOnlineEvent` helper |
| `omcgo/internal/device/service_test.go` | +143 / -1 | 4 个新测试：OfflineToActive_PublishesOnlineEvent / FirmwareChangedSuppressesOnlineEvent / ActiveStaysActive_NoOnlineEvent / NilEventBus_NoCrash |
| `omcgo/internal/provision/engine.go` | +91 / -1 | 新增 redisClient 字段 + SetRedisClient setter；Subscribe 追加 SubjectDeviceOnline 订阅；新增 HandleDeviceOnline 方法（Redis token bucket + StartPathBSync 调用） |
| `omcgo/internal/provision/engine_test.go` | +109 / -1 | 4 个新测试：RedisTokenBucketSkipsRepeat / DeviceNotFound_NoOp / NilSyncService_NoOp / RedisDown_StillProceeds |
| `omcgo/internal/provision/sync.go` | +27 / -0 | 新增 redisClient 字段 + SetRedisClient setter；新增 `pathBOptions` 结构体 + `WithReason` functional option |
| `omcgo/internal/provision/sync_pathb.go` | +119 / -2 | StartPathBSync 加 variadic opts + 写 Redis reason；HandleSyncResultPathB 后置插入差集计算；新增 `snapshotStandardPaths` + `logPathBSyncDiff` helpers |
| `omcgo/internal/provision/sync_pathb_test.go` | +213 / -1 | 9 个新测试：LogsMissingPaths_InfoLevel / SkipsOnFirstSync / NoMissing_NoLog / WarnLevel_WhenSetTooSmall / SamplingTruncatesTo20 / ReasonUnknownWhenRedisKeyAbsent / SnapshotStandardPaths × 3 |
| `omcgo/cmd/app/provider/modules.go` | +5 / -1 | DI wiring：provisionEngine.SetRedisClient(c.Redis) + syncSvc.SetRedisClient(c.Redis) |

**总改动**：净 +789 / -11，其中生产代码 ~310 行 / 测试 ~470 行。**符合 S 工作量估算**。

---

## 2. S4 出口门检查

| 门 | 命令 / 检查 | 结果 |
|----|-----------|------|
| `go build ./...` 通过 | `go build ./...` | ✅ PASS（无输出） |
| `go test ./...` 全绿 | `go test ./... -count=1` | ✅ PASS — 70+ 包全过；**唯一失败 TestDownloadHandler 在 main baseline 已 FAIL**（git stash 验证，与本任务无关） |
| 前端 typecheck | N/A — 本任务无前端改动 | N/A |
| `golangci-lint run` | 本地未安装 lint | ⚠️ Skipped — 留 CI 执行（dev-pipeline §B4 容忍） |
| 迁移 up/down 演练 | N/A — 本任务无 schema 变更 | N/A |
| 新端点 E/R ≥ 1 | N/A — 本任务无新 REST 端点（T-0126 才有手动同步端点） | N/A |
| 新 metric/log grep ≥ 1 | grep 验证每个新名称 | ✅ 全部命中（详见 §3） |
| 累计型依赖核销 | N/A — 本任务无累计型 deps | N/A |

---

## 3. 观测埋点 grep 验证

```bash
grep -rn "firmware_changed_suppresses_online" omcgo/internal/ → 命中 2（device_service.go × 2: log + doc comment）
grep -rn "device.online published" omcgo/internal/ → 命中 1（device_service.go:872 log）
grep -rn "param_sync_missing" omcgo/internal/ → 命中 8（sync_pathb.go × 2 prod log，sync_pathb_test.go × 6 测试断言）
grep -rn "provision:online_sync" omcgo/internal/ omcgo/cmd/ → 命中 4（engine.go log + key fmt + doc，modules.go 注释）
grep -rn "provision:syncreason" omcgo/internal/ → 命中 5（sync.go doc + sync_pathb.go × 2 + tests × 2）
```

**全 grep 命中** ✅，可观测性钩子全部到位。

---

## 4. 测试统计

| 测试函数 | 包 | 行为覆盖 | 结果 |
|---------|-----|---------|------|
| `TestUpdateFromInform_OfflineToActive_PublishesOnlineEvent` | device | offline→active + swVersion 不变 → 发 device.online + 载荷四字段 | ✅ PASS |
| `TestUpdateFromInform_FirmwareChangedSuppressesOnlineEvent` | device | offline→active + swVersion 变化 → 挡板抑制（无 device.online 事件） | ✅ PASS |
| `TestUpdateFromInform_ActiveStaysActive_NoOnlineEvent` | device | active→active → 不发事件 | ✅ PASS |
| `TestUpdateFromInform_NilEventBus_NoCrash` | device | eventBus=nil → 不 panic | ✅ PASS |
| `TestHandleDeviceOnline_RedisTokenBucketSkipsRepeat` | provision | 同设备 60s 内第二次调用 → 不进 device lookup | ✅ PASS |
| `TestHandleDeviceOnline_DeviceNotFound_NoOp` | provision | device 不存在 → 无 panic 无 error | ✅ PASS |
| `TestHandleDeviceOnline_NilSyncService_NoOp` | provision | syncService=nil → 早返不调 GetByID | ✅ PASS |
| `TestHandleDeviceOnline_RedisDown_StillProceeds` | provision | Redis 不可达 → SetNX 失败但继续推进 lookup（容忍 Redis 抖动） | ✅ PASS（2.09s — 重试退避） |
| `TestLogPathBSyncDiff_LogsMissingPaths_InfoLevel` | provision | B=4 / A=3 / missing=1 + reason="device_online" → Info 级日志 4 字段 | ✅ PASS |
| `TestLogPathBSyncDiff_SkipsOnFirstSync` | provision | B 空（首次同步）→ 不输出日志 | ✅ PASS |
| `TestLogPathBSyncDiff_NoMissing_NoLog` | provision | 所有旧 path 都重新上报 → 不输出日志 | ✅ PASS |
| `TestLogPathBSyncDiff_WarnLevel_WhenSetTooSmall` | provision | current < prev/2 → 升级 Warn 级别 | ✅ PASS |
| `TestLogPathBSyncDiff_SamplingTruncatesTo20` | provision | 50 个 missing → sample 截断到 20 条 + missing_count=50 | ✅ PASS |
| `TestLogPathBSyncDiff_ReasonUnknownWhenRedisKeyAbsent` | provision | Redis 无 reason key → 降级 reason="unknown" 仍输出 | ✅ PASS |
| `TestSnapshotStandardPaths_ReturnsNilOnError` | provision | GetByDevice 失败 → 返 nil 静默跳过 | ✅ PASS |
| `TestSnapshotStandardPaths_ReturnsEmptyOnNoRows` | provision | 无现有 standardPath → 返 nil（等价首次同步） | ✅ PASS |
| `TestSnapshotStandardPaths_BuildsMap` | provision | 3 个 path → 正确构建 map[string]struct{} | ✅ PASS |

**合计 17 个新测试 / 0 个 FAIL** ✅

---

## 5. 验收标准映射（PRD §3）

| AC | 测试覆盖 |
|----|---------|
| AC-1（offline→active 触发 Path B + Redis token bucket + 60s 重试跳过）| TestUpdateFromInform_OfflineToActive_PublishesOnlineEvent + TestHandleDeviceOnline_RedisTokenBucketSkipsRepeat |
| AC-2（firmware 变化挡板抑制 online 事件 + 日志记录）| TestUpdateFromInform_FirmwareChangedSuppressesOnlineEvent |
| AC-3（Path B 完成时差异日志 6 字段）| TestLogPathBSyncDiff_LogsMissingPaths_InfoLevel |
| AC-4（首次同步跳过差异日志 / B 空时无输出）| TestLogPathBSyncDiff_SkipsOnFirstSync |
| AC-5（同设备短时间多次 Inform 不重复入队）| TestHandleDeviceOnline_RedisTokenBucketSkipsRepeat（单元测试覆盖；端到端 cpe_simulator E2E 留 T-0125 完成后联动补） |

---

## 6. Out of Scope / 诚实碎片

1. **`provision_online_throttle_total` Prometheus counter 未实施** — S2 备忘列出但生产代码未加（provision 包无现有 Prometheus DI 链路）。token bucket 命中率可通过 grep `"device.online throttled by token bucket"` 日志计数获得。**Follow-up**：T-0128-x 加 metrics 包注册（小改 ~10 行）
2. **`SubjectDeviceFirmwareChanged` 事件本身未发布** — T-0128 仅在 UpdateFromInform 内做挡板（log "firmware_changed_suppresses_online" 不发 online），真正的 firmware.changed 事件发布留到 T-0125 实施。**临时降级**：firmware 变化设备短期内不触发 Path B 同步（与 pre-T-0128 行为相同，不引入回归）
3. **`offline_detector.go:133` 字面量未迁移到 `SubjectDeviceOffline` 常量** — S2 备忘指出"顺手补但不强迁旧代码"。常量已加，迁移留 follow-up（一行替换，零业务变化）
4. **E2E 用例未补到 `scripts/e2e_verify.sh`** — 本任务依赖 cpe_simulator 模拟 11min 离线 + 上线流程，跨大型脚本改动；留到 T-0125 firmware 路径完成时一起补（设计方案 §8.2 计划）
5. **golangci-lint 本地未跑** — 本机未装；CI / S5 阶段补
6. **TestDownloadHandler pre-existing failure** — main baseline 已 FAIL（git stash 验证），与本任务无关，**不阻塞本次提交**

---

## 7. 关键 ULTRATHINK 决策回顾

| 决策 | 方案 | 理由 |
|------|------|------|
| Reason 传递 | Redis 临时映射 `provision:syncreason:{deviceID}` TTL=10min | 不破坏 sourceID 作为 task_id 追溯的契约；不引入新组件；异步回流端可读 |
| Redis 注入 | Setter 模式 `SetRedisClient` 不改构造函数 | 保 T-0098/既有调用方不破；nil-safe 降级 |
| `pathBOptions` Functional option | `WithReason("...")` | 4 个现有 caller 不动；新调用方按需附加；Go-idiomatic |
| firmware 变化挡板 | log "firmware_changed_suppresses_online" 不发任何事件 | T-0125 实施前不引入"firmware-change 触发 online sync"的临时错误行为；与 pre-T-0128 行为一致 |
| `SubjectDeviceOffline` 补常量但不迁旧代码 | 添加常量 + 文档注释 + 保留 offline_detector.go:133 字面量 | 控制本任务范围；为后续清理留好钩子 |
| 差异日志 `missing_paths_sample` 截断到前 20 | 不再加查询端点 | 设计方案 §5.5 明示；运维需全量时另开端点（不在本方案） |

---

**S4 出口门：✅ PASS（lint 留 CI；TestDownloadHandler pre-existing 不阻塞）**

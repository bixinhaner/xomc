# T-0126 — S4 本地验证报告

**任务**：T-0126（F09 参数同步触发链 §4 — 手动同步端点 + 前端按钮）— **F09 触发链 5/5 收官**
**PRD**：`docs/project/prd/F09-param-sync-trigger-chain.md#§4` + `## 设计备忘（T-0126）`
**Sprint**：sprint-11 stretch
**Owner**：Claude
**日期**：2026-05-14
**Est**：S

---

## 1. 改动范围

| 文件 | 行数变化 | 说明 |
|------|---------|------|
| `omcgo/internal/provision/sync.go` | +10 | `StartManualSync` wrapper（避免 device 包 import provision.PathBOption 循环依赖） |
| `omcgo/internal/device/param_sync_starter.go` | **新增 25 行** | `ParamSyncStarter` narrow interface（消费者驱动，1 方法） |
| `omcgo/internal/device/device_service.go` | +50 / -40 | 删旧 `TriggerParamSync`（Path A）；新增 `SyncDeviceParamsManual` + paramSyncStarter 字段 + setter |
| `omcgo/internal/device/device_handler.go` | +50 / -16 | 删旧 `TriggerParamSync` handler + `/param-sync` 路由；新增 `SyncDeviceParams` handler + `/sync-params` 路由（含 force 字段 / sourceID 生成 / 503 fallback） |
| `omcgo/internal/device/device_param_handler.go` | +5 / -65 | 删旧 `TriggerSync` handler + `/parameters/sync` 路由（注释说明迁移到新端点）|
| `omcgo/cmd/app/provider/modules.go` | +4 | DI wiring：`deviceService.SetParamSyncStarter(syncSvc)` |
| `omcgo/internal/device/handler_test.go` | +135 | 7 新 testcase：Success / NoBody_OK / NotFound / InvalidUUID / PathBUnavailable_503 / NilStarter_500 / RoutesOldEndpointGone |
| `omcmb/frontend-core/src/services/api/deviceApi.ts` | +30 | `syncDeviceParams(id, options?)` API method（含 snake_case → camelCase 字段映射） |
| `omcmb/frontend-core/src/hooks/api/useDevices.ts` | +18 | `useSyncDeviceParams` Hook（mutation + invalidate device-parameters cache） |
| `omcmb/frontend-core/src/services/api/deviceParameterApi.ts` | +2 / -7 | 删 `syncParameters` 方法（迁移注释） |
| `omcmb/frontend-core/src/hooks/api/useDeviceParameters.ts` | +2 / -18 | 删 `useSyncParameters` Hook + 删 unused `ParameterSyncOptions` 导入 |
| `omcmb/frontend-core/src/mock/services/deviceParameterService.ts` | +24 / -41 | 删 mock syncParameters；简化 discoverParameters 不再 fallback 调；删 unused `ParameterSyncOptions` 导入 |
| `omcmb/webcode/src/pages/device/DeviceDetail/ParameterTreeTab/index.tsx` | +7 / -6 | 切换 `useSyncParameters` → `useSyncDeviceParams`；toast 文案含 sourceId |

**总改动**：净 +362 / -193，生产代码 ~170 行 / 测试 ~135 行 / 前端 ~60 行。**符合 S 工作量**（旧端点删除 ~190 行抵消新增）。

---

## 2. S4 出口门

| 门 | 结果 |
|----|------|
| `go build ./...` | ✅ PASS |
| `go test ./...` 全包 | ✅ PASS（仅 pre-existing TestDownloadHandler 与本任务无关） |
| 前端 `npx tsc --noEmit` | ✅ PASS（无新增类型错） |
| 前端 `npm run lint` | ✅ 改动 6 文件 0 新错（pre-existing 225 baseline 错与本任务无关；frontend-core 在 webcode eslint base path 外但本任务 ParameterTreeTab/index.tsx 0 错） |
| 迁移 N/A | 本任务无 schema 变更 |
| 新端点 E/R ≥ 1 | ✅ 新端点 `POST /devices/:id/sync-params` + 7 handler 单测 + 1 路由覆盖测试（`RoutesOldEndpointGone`） |
| 累计型依赖 | N/A |
| `golangci-lint` | ⚠️ 本地未装留 CI |

---

## 3. 观测埋点 grep 验证

```bash
grep -rn "SyncDeviceParams\|StartManualSync\|ParamSyncStarter" omcgo/internal/ → 后端定义 + 实现 + 注入 + 7 测试
grep -rn "manual sync requested" omcgo/internal/ → device_service.go info log
grep -rn "/sync-params" omcgo/internal/ → device_handler.go 路由注册 + 端点注释
grep -rn "syncDeviceParams\|useSyncDeviceParams" omcmb/ → 前端 API + Hook + ParameterTreeTab 调用
```

全 grep 命中 ✅

---

## 4. 测试统计（新增）

| 测试函数 | 验证 | 结果 |
|---------|------|------|
| `TestHandler_SyncDeviceParams_Success` | 202 响应含 status/source_id/device_id/serial_number；source_id 前缀 manual:；ParamSyncStarter 调用一次 | ✅ |
| `TestHandler_SyncDeviceParams_NoBody_OK` | body 为空（force 字段缺失）→ 仍 202 | ✅ |
| `TestHandler_SyncDeviceParams_NotFound` | device 不存在 → 404 | ✅ |
| `TestHandler_SyncDeviceParams_InvalidUUID` | URL id 非 UUID → 400 | ✅ |
| `TestHandler_SyncDeviceParams_PathBUnavailable_503` | starter 返 used=false（MappingSet 缺失）→ 503 + message 含 "path-b sync unavailable" | ✅ |
| `TestHandler_SyncDeviceParams_NilStarter_500` | DI 未注入 starter → 500（防 NPE） | ✅ |
| `TestHandler_SyncDeviceParams_RoutesOldEndpointGone` | 旧 `/param-sync` 路由返 404 验证已下线 | ✅ |

**合计 7 个新测试 / 0 个 FAIL** ✅

---

## 5. 出口门 5 项检查

- [x] 接口契约明确（ParamSyncStarter narrow interface + StartManualSync wrapper + 端点 contract）
- [x] 迁移草案 N/A
- [x] Carrier 差异点：无新增（手动同步走 Path B 自动适配 Carrier MappingSet）
- [x] 观测埋点列出（manual sync requested log + sourceID prefix + 前端 toast）
- [x] 待定点 0 个

---

## 6. Out of Scope / 诚实碎片

1. **`force` 字段当前 no-op** — 设计方案 §4.1 提到该字段但当前无 manual-side 节流可绕过（manual 端点天然不走 Redis token bucket）；保留字段供未来扩展（例如 PeriodicSyncer 跳过窗口绕过）
2. **响应不返单一 taskID** — StartPathBSync 内部 enqueueGPVPrefixes 创建多个 batch GPV task，无单一 taskID。响应返 `source_id="manual:UUID"` 作 correlation 标识
3. **前端 SyncStatusBar 与新端点不联动** — 前端旧 `useSyncStatus` (`GET /parameters/sync-status`) 是 discovery flow 进度条；新 Path B 全量同步用 toast 提示 sourceId 不显示进度条（Path B 完成时间分钟级用户感知可接受；进度由日志追踪）
4. **前端 lint pre-existing 225 baseline 错** — 与本任务无关（main baseline 已存在）
5. **E2E 用例补到 `scripts/e2e_verify.sh`** — 留 F09 触发链整体集成验证（依赖 cpe_simulator + Path B 全链路）后续单列任务
6. **golangci-lint** — 本地未装留 CI
7. **TestDownloadHandler pre-existing failure** — 与本任务无关，git stash 验证基线已 FAIL

---

## 7. 关键决策回顾

| 决策 | 方案 | 理由 |
|------|------|------|
| 升级范围 (b) | 删旧端点 `/param-sync` + `/parameters/sync` + 新加 `/sync-params` | URL 与设计方案 §4.1 一致；旧 Path A 整体下线；前端只改 1 行 import |
| 循环依赖 | 消费者驱动 ParamSyncStarter narrow interface（device 包定义，provision.SyncService 自然满足）+ wrapper `StartManualSync` | 不暴露 provision.PathBOption 类型给 device 包；与 T-0124 ParamSyncWriter 风格统一 |
| 响应不返单一 taskID | `source_id="manual:UUID"` 作 correlation | StartPathBSync 多 batch 入队天然无单一 ID；source_id 通过 Redis hint 传递 reason="manual" 让差异日志可追溯 |
| Path B 不可用 503 + Fallback | 返 503 + 明确 message vs 自动回退 Path A | Path A 已下线；MappingSet 缺失说明设备 productClass 未配置，需运维侧处理而非自动降级 |
| 旧 `/parameters/sync` 路由整体删除 | 不保留 deprecated | 前端唯一调用方已切换；保留只增加端点表面积 |
| `TriggerDiscover` / `GetSyncStatus` 保留 | 与 Path B 全量同步并存 | discovery flow 是独立功能；SyncStatusBar 仍由 discovery 用 |

---

## 8. F09 参数同步触发链 5/5 全收官 🎉

| Task | 触发源 | reason 标签 | 状态 | Commit |
|------|--------|-----------|------|--------|
| T-0123 | device.online 事件 + Redis token bucket | "device_online" | ✅ done | `d1eea063` |
| T-0124 | PeriodicSyncer ticker + PG advisory lock | "periodic" | ✅ done | `9f071c33` |
| T-0125 | device.firmware.changed 事件 + Redis 串行锁 | "firmware_changed" | ✅ done | `b4790172` |
| **T-0126** | **手动端点 `POST /devices/:id/sync-params`** | **"manual"** | ✅ **done（本任务）** | 待 commit |
| T-0127 | Path B 差异日志（共用 reason 通道） | — | ✅ done | `d1eea063`（与 T-0123 合并） |

**5 触发源全部用 Path B 全量同步** → 统一 last_param_sync_at 回写口径（T-0124）+ 统一差异日志（T-0127）+ 统一 Translator 翻译（T-0098 基础设施）。

**F09 参数同步触发链补强完整闭环**：
- 已存在设备 offline→active 自动刷新（T-0123）
- 周期性兜底漂移检测（T-0124）
- 固件升级后重新交集避免 Translator 降级（T-0125）
- 一键全量对账运维入口（T-0126）
- 跨触发源差异追溯日志（T-0127）

设计方案 9 漏点中 5 个全部交付；剩 §6 Value Change 真机测试（需真机环境，单列任务）。

---

**S4 出口门：✅ PASS**

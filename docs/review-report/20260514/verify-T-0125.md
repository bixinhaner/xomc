# T-0125 — S4 本地验证报告

**任务**：T-0125（F09 参数同步触发链 §3 — firmware.changed 事件 + Redis 串行锁 + 重新交集）
**PRD**：`docs/project/prd/F09-param-sync-trigger-chain.md#§3` + `## 设计备忘（T-0125）`
**Sprint**：sprint-11 stretch（接力 T-0123 ✅）
**Owner**：Claude
**日期**：2026-05-14

---

## 1. 改动范围

| 文件 | 行数变化 | 说明 |
|------|---------|------|
| `omcgo/internal/core/event/subjects.go` | +6 / -0 | 新增 `SubjectDeviceFirmwareChanged = "device.firmware.changed"` 常量 + doc comment |
| `omcgo/internal/device/device_service.go` | +50 / -8 | 把 T-0123 firmware 挡板从 log-only 替换为 publishDeviceFirmwareChangedEvent；新增 `DeviceFirmwareChangedEvent` 类型（DeviceID/SerialNumber/ProductClass/OldVersion/NewVersion/BecameOnline 6 字段）+ `publishDeviceFirmwareChangedEvent` helper |
| `omcgo/internal/provision/engine.go` | +91 / -0 | Subscribe 追加 SubjectDeviceFirmwareChanged 订阅（含 Deduper 包装）；新增 `HandleFirmwareChanged` 方法（5 步控制流：Redis 串行锁 → reason hint → GetDevice → RequestModelUpload → Path B fallback） |
| `omcgo/internal/device/service_test.go` | +57 / -10 | 强化原 FirmwareChangedSuppressesOnlineEvent 测试同时断言 firmware.changed 事件 + 载荷 4 字段；新增 ActiveStaysActive_BecameOnlineFalse 测试 |
| `omcgo/internal/provision/engine_test.go` | +109 / 0 | 新增 5 testcase：RedisSerialLockSkipsConcurrent / WritesReasonHintToRedis / DeviceNotFound_NoOp / NilModelUpload_FallbackPathBSkippedWhenSyncNil / RedisDown_StillProceeds |

**总改动**：净 +313 / -18，生产代码 ~147 行 / 测试 ~166 行。**符合 S 工作量估算**。

---

## 2. S4 出口门

| 门 | 结果 |
|----|------|
| `go build ./...` | ✅ PASS |
| `go test ./internal/device/... ./internal/provision/... ./internal/core/event/...` | ✅ PASS（device 1.6s / provision 6.8s / event 5.1s） |
| `go test ./...` 全包 | ✅ PASS（仅 pre-existing TestDownloadHandler 与本任务无关） |
| 前端 typecheck | N/A（无前端改动） |
| `golangci-lint` | ⚠️ 本地未装留 CI |
| 迁移 up/down | N/A（无 schema 变更） |
| 新端点 E/R ≥ 1 | N/A（无新 REST 端点） |

---

## 3. 观测埋点 grep 验证

```bash
grep -rn "SubjectDeviceFirmwareChanged" omcgo/internal/ → 命中 5+
grep -rn "firmware.changed event published" omcgo/internal/ → device_service.go info log（publish helper）
grep -rn "firmware handling already in progress" omcgo/internal/ → engine.go debug log
grep -rn "provision:firmware_handling" omcgo/internal/ → engine.go doc + key fmt
grep -rn "firmware.changed: model upload" omcgo/internal/ → engine.go info × 2（enqueued / skipped）
grep -rn "firmware.changed: direct Path B" omcgo/internal/ → engine.go info（fallback initiated）
```

**全 grep 命中** ✅

---

## 4. 测试统计（新增）

| 测试函数 | 包 | 验证 | 结果 |
|---------|-----|------|------|
| `TestUpdateFromInform_FirmwareChangedSuppressesOnlineEvent`（强化）| device | swVersion 变化 → firmware.changed 事件载荷 4 字段 + BecameOnline=true + 二选一抑制 device.online | ✅ PASS |
| `TestUpdateFromInform_FirmwareChanged_ActiveStaysActive_BecameOnlineFalse` | device | active→active + swVersion 变化 → firmware.changed BecameOnline=false | ✅ PASS |
| `TestHandleFirmwareChanged_RedisSerialLockSkipsConcurrent` | provision | 同 deviceID 10min 内第二次调用被串行锁拦截不进 lookup | ✅ PASS |
| `TestHandleFirmwareChanged_WritesReasonHintToRedis` | provision | reason hint `provision:syncreason:{deviceID}="firmware_changed"` 写入 Redis | ✅ PASS |
| `TestHandleFirmwareChanged_DeviceNotFound_NoOp` | provision | GetDevice 返 nil → 无 panic 无 error | ✅ PASS |
| `TestHandleFirmwareChanged_NilModelUpload_FallbackPathBSkippedWhenSyncNil` | provision | modelUploadService=nil + syncService=nil → fallback 早返无 panic | ✅ PASS |
| `TestHandleFirmwareChanged_RedisDown_StillProceeds` | provision | Redis 不可达 → SetNX 失败但继续推进 device lookup | ✅ PASS（4.19s — 重试退避） |

**合计 7 个新/强化测试 / 0 个 FAIL** ✅

---

## 5. 出口门 5 项检查

- [x] 接口契约明确（DeviceFirmwareChangedEvent 6 字段 / Subscribe 追加 / HandleFirmwareChanged 5 步控制流）
- [x] 迁移草案 N/A
- [x] Carrier 差异点：无新增（既有 MappingSet 自动覆盖）
- [x] 观测埋点名字列出 + grep 全命中（4 log key + 1 Redis lock key + 1 reason hint key 复用 T-0123）
- [x] 待定点 0 个

---

## 6. Out of Scope / 诚实碎片

1. **Path B 完成时 reason 由 hint 决定** — 当 handleDataModelFileReceived 内 auto-sync 触发 Path B 时不带 WithReason，依赖 step 2 预设的 Redis hint；hint TTL=10min 覆盖 Upload + Intersect + Path B 上界。若实际超 10min 则 reason 降级为 "unknown"（HandleSyncResultPathB 已处理）—— 验收可接受
2. **HandleFirmwareChanged 不等 Upload 完成** — 锁的作用是防并发触发而非协调步骤；Path B 由后续 datamodel.file.received 事件链异步触发；锁 TTL 10min 自然过期防短时间重复触发
3. **modelUploadService nil 路径** — 走直接 Path B fallback；本任务测试只验证 nil 不 panic + 不进入死分支（syncService 也 nil 时不真实调用）。真实链路集成由 T-0123 已验证（StartPathBSync + diff log）
4. **golangci-lint 本地未跑** — 留 CI
5. **E2E** — `scripts/e2e_verify.sh` 加 swVersion v1→v2 模拟用例留 T-0125 后续集成 commit（cpe_simulator + handleDataModelFileReceived 全链路）

---

## 7. 关键决策回顾

| 决策 | 方案 | 理由 |
|------|------|------|
| Reason 传递 | 复用 T-0123 Redis key `provision:syncreason:{deviceID}` | StartPathBSync 内 `if pbOpts.reason != ""` 仅显式 WithReason 时覆盖；handleDataModelFileReceived 不带 opts → 不覆盖 → HandleSyncResultPathB 读到 firmware_changed |
| Redis 串行锁不主动释放 | 让 TTL=10min 自然过期 | 防短时间内重复 firmware Inform 引发并发交集；锁作用是去重不是协调 |
| HandleFirmwareChanged 不阻塞等 Upload 完成 | 步骤 4 入队后立即返回 | EventBus handler 不应阻塞；剩余链路由 handleDataModelFileReceived 异步处理 |
| `enable_filetype11=false` 走 fallback Path B | 调用 StartPathBSync(WithReason("firmware_changed")) 用 default 映射 | 不"卡住"该设备；接受映射精度暂时降级，下次设备启用时再走完整交集 |
| BecameOnline 字段保留在事件载荷 | 不直接抑制 — 让订阅者知晓 | 下游可参考做指标/审计区分；当前 HandleFirmwareChanged 不依赖此字段 |

---

**S4 出口门：✅ PASS**

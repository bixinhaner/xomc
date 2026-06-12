# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-12 |
| 提交 | c0fa92cb（staged diff id，commit 见 git log） |
| 作者 | renpengfei |
| 范围 | software / ufte / acs-transfercfg / frontend |
| 变更文件数 | 12（10 改 + 2 新增） |
| 新增行数 | +497 |
| 删除行数 | -45 |

## 变更概要

升级/回退任务并发控制双层化：

1. **任务内并发数可配**：创建升级类任务（升级 / PATCH / FPGA）时可指定任务内设备并发执行数（1-100，默认 20），UFTE `CreateTaskRequest.concurrency` 透传 `software.BatchUpgradeRequest.Concurrency`；前端文件传输中心建任务抽屉新增「升级并发数」输入。
2. **系统级（跨任务）并发上限闸**：新增 `fairSlotPool`（`global_concurrency.go`），所有升级/回退任务合计同时执行的设备数封顶（默认 100），空闲槽位按任务轮转（round-robin）分配，防多任务叠加打爆固件下载带宽 / ACS。上限运行时取自 sys_config `acs_transfer.maxGlobalUpgradeConcurrency`（系统设置 → ACS 传输配置新增表单项，30s 缓存生效）。
3. **槽位语义 = 设备服务器侧 IO 阶段**：`waitSubTaskSlotRelease` 以 10s 轮询子任务状态，进入 rebooting/verifying/completed/failed/terminated 即释放槽位（TC 后安装重启不占服务器资源）；2h 兜底超时防 reaper 失效时槽位泄漏。派发循环移入后台 goroutine（槽位长占后 sem 满载，不能阻塞 HTTP 调用方），急停时补齐 done 计数。

## 审查发现

### CRITICAL

无。

### WARNING

无。

### INFO

1. **后端未钳制 `req.Concurrency` 上限** — 前端 InputNumber max=100，但直连 API 可传更大值；系统级闸（默认 100）实际兜底总量，风险可控。建议后续在 service 层 clamp 到全局上限。
2. **槽位等待为轮询实现** — 每个在飞设备一个 goroutine 以 10s 间隔查 DB，最坏并发 = 全局上限（默认 100 → ~10 QPS 点查），负载可忽略；若运维把上限调到数千需重新评估（可改事件订阅）。
3. **回退任务内并发仍固定 5** — `startRollbackExecution` 未开放参数，但已接入系统级闸；与升级共用闸语义一致，设计如此。

## 验证

- `go build ./...` ✓
- `go test -race ./internal/software/ ./internal/ufte/ ./internal/acs/transfercfg/` 全部 ok ✓
- `fairSlotPool` 4 个单测：基本占用/释放、跨任务轮转公平（A1→B1→A2 而非 FIFO）、SetLimit 动态放行、ctx 取消不泄漏槽位 ✓
- 前端链路核对：系统设置保存走通用 KV 批量 upsert（`maxGlobalUpgradeConcurrency` 落 `acs_transfer` category，与后端 `transfercfg.KeyMaxGlobalUpgradeConcurrency` 同 key）✓；建任务 payload `...values` 展开包含 `concurrency` ✓

## 结论

**PASS** — 无 CRITICAL / WARNING，3 条 INFO 为后续可选优化。

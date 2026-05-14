# Code Review — BatchInformProcessor transition 事件发布补完

| 字段 | 值 |
|---|---|
| Base Commit | c6c97954（前置 Bug-2 menus seed）|
| Author | shangyingbin |
| Reviewer | Claude Opus 4.7（自动审查） |
| Scope | device, provision |
| Type | fix（hotfix）|
| Backlog | HOTFIX（08754f89 done 后真机 E2E 发现的 P0 缺陷，原 T-0123/T-0125 已 done）|
| Files | 4 |
| Lines | +70 / -14 |
| Date | 2026-05-14 |

---

## 1. 背景

08754f89（F09 触发链 5/5 收官 PR）实现 T-0123 / T-0125 时只覆盖了 `DeviceService.UpdateFromInform` 的非 batch 路径（在末尾调 `publishDeviceOnlineEvent` / `publishDeviceFirmwareChangedEvent`）。但本机部署默认开启了 `BatchInformProcessor` —— `inform_handler.go:250` → `BatchInformProcessor.Submit`，flush 流程 `batch_processor.go:doFlush` 只更新 PG + Redis，**完全跳过事件发布**。

真机 E2E 中（2026-05-14 16:36~16:39）：mock `status=offline` + 清 Redis 缓存 + 等真机 Inform 翻回 active 后，**完全没有 `device.online published` 日志**，`provision:online_sync:` Redis key 也不存在 → T-0123 触发链在真机 batch 模式下死寂。

## 2. 修复设计

### 2.1 暴露面

`DeviceService.publishDeviceOnlineEvent` / `publishDeviceFirmwareChangedEvent` 改为大写导出 `PublishXxx`，并新增注释说明"导出供 BatchInformProcessor.doFlush 调用"。同时把 `UpdateFromInform` 内部两处调用也更新到 PascalCase。

### 2.2 跨包接口

`batch_processor.go` 新增 narrow interface：

```go
type TransitionEventPublisher interface {
    PublishDeviceOnlineEvent(ctx context.Context, device *model.Device)
    PublishDeviceFirmwareChangedEvent(ctx context.Context, device *model.Device,
        oldVersion, newVersion string, becameOnline bool)
}
```

`DeviceService` 自然满足此接口（隐式实现）。仅 2 方法，避免 BatchInformProcessor 反向依赖 DeviceService 整体类型。

### 2.3 数据流

```
inform_handler.handlePeriodic
  ├─ oldStatus, oldVersion := device.Status, device.FirmwareVersion   // 调 prepareDeviceUpdate 前捕获
  ├─ prepareDeviceUpdate(device, inform)                              // 覆盖 device 字段
  └─ batchProcessor.Submit(device, inform, params, oldStatus, oldVersion)
         └─ informUpdate{device, inform, params, oldStatus, oldVersion}
              └─ doFlush（10s tick / 缓冲满）
                   ├─ batchUpdateDevices  // PG 写入
                   ├─ batchUpsertParams   // PG 写入
                   ├─ batchRedisOps       // Redis 写入
                   └─ 新增第 4 步：遍历 hit，按 oldStatus/oldVersion 调 publisher
                          ├─ firmwareChanged → PublishDeviceFirmwareChangedEvent
                          ├─ becameOnline    → PublishDeviceOnlineEvent
                          └─ 二选一挡板：firmwareChanged 优先，becameOnline 在 firmwareChanged 时不发
```

### 2.4 装配

`cmd/app/provider/device.go`：`batchProcessor.SetTransitionPublisher(deviceService)`，仅在 BatchProcessor enabled 时注入。test 场景或灰度关闭时 publisher=nil，跳过事件发布（向后兼容）。

## 3. 审查检查项

### 3.1 Go 工程规范

- [x] 命名：导出函数 PascalCase（`PublishDeviceOnlineEvent` / `PublishDeviceFirmwareChangedEvent` / `TransitionEventPublisher` / `SetTransitionPublisher`） ✅
- [x] 接口设计：narrow interface（仅 2 方法），消费者侧 batch_processor.go 定义，符合"接受接口、返回具体类型"原则 ✅
- [x] 错误处理：PublishXxx 内部已 try-publish 不阻塞（沿用原实现 ✅）；event publish 失败仅 Warn，不影响 doFlush 返回
- [x] 并发安全：transitionPublisher 字段由 SetTransitionPublisher 在装配阶段一次性设置，doFlush 异步读取 — 无 race
- [x] nil 容错：`if p.transitionPublisher != nil` 检查，test 场景安全
- [x] Context 传递：使用 doFlush 的 ctx（30s timeout），事件发布在同一 ctx 内
- [x] 资源/泄漏：无新增 goroutine / 连接

### 3.2 业务正确性

- [x] **事件发布时机**：在 PG + cache 写入成功后（doFlush 步骤 4），与非 batch 路径行为对齐——避免事件发布与数据状态不一致（订阅者读 PG 看到的 device 跟事件载荷一致）
- [x] **二选一挡板**：firmware.changed 优先，becameOnline 在 firmwareChanged 同时满足时不发独立 device.online — 与 device_service.go:688-693 的非 batch 路径逻辑**完全等价**
- [x] **失败时不发事件**：if batchUpdateDevices / batchUpsertParams 失败 → doFlush 提前 return err → 第 4 步不会执行 → 不发"伪造"事件 ✅
- [x] **hit/orphan 分区**：只对 hit（PG 行存在的设备）发事件，orphan（cache stale）不发 — 正确语义

### 3.3 Submit 签名变更（潜在 breaking）

- [x] 调用点完整：`grep -rn "batchProcessor.Submit\|p.Submit\|processor.Submit\|bp.Submit" omcgo/internal/ omcgo/cmd/ | grep -v _test.go` → 仅 1 处（inform_handler.go:257），已同步更新
- [x] 测试调用点：`grep -rn "Submit(.*params\|Submit(device, inform" omcgo/internal/device/` → 无 test 直接调 Submit（batch_processor 走 worker channel，测试通过 fake processor）
- [x] 编译：`go build ./...` exit 0
- [x] 单测：`go test ./internal/device/... ./internal/provision/...` 全绿

### 3.4 测试覆盖

- [ ] ⚠️ **本次未新增 batch processor 单测**。原因：BatchInformProcessor 单测受 PG/Redis 依赖较深（doFlush 调真实 pgxpool）；本次修复语义已被**真机 E2E 测试**完整覆盖（重测 T-0123 mock offline + 清缓存 → 真机 Inform → batch path 触发 device.online published + path-b sync started reason=device_online + token bucket 60s 拦截二次 ✅；T-0125 mock 旧固件 → firmware.changed published + model upload enqueued ✅；二选一挡板 became_online_suppressed=true ✅）。
- [ ] ⚠️ **跟进**：后续可加 fakeTransitionPublisher 单测覆盖 doFlush 第 4 步分支选择逻辑（firmwareChanged / becameOnline / both / neither）。

### 3.5 安全 / 合规

- [x] 无新增外部输入处理
- [x] 无 SQL 拼接
- [x] 无敏感日志（事件载荷不含密钥）

## 4. 结论

**PASS_WITH_WARNINGS**

| 严重级 | 数量 | 说明 |
|---|---|---|
| CRITICAL | 0 | — |
| WARNING | 0 | — |
| INFO | 2 | 见 §3.4：单测覆盖建议跟进 |

**理由**：
- 修复策略正确（与非 batch 路径行为对齐）
- 业务正确性可证：事件发布时机、二选一挡板、失败短路均与非 batch 路径等价
- 编译 + 现有单测全绿
- 真机 E2E 实证覆盖
- 唯一遗留：batch processor 第 4 步事件发布逻辑未直接单测，已记为跟进

## 5. 相关文档

- 真机 E2E 测试结果：`~/Documents/notes/F09-触发链真机E2E测试结果-20260514.md` §Bug-3
- 设计参考（非 batch 路径行为）：`omcgo/internal/device/device_service.go` UpdateFromInform §末尾事件发布块

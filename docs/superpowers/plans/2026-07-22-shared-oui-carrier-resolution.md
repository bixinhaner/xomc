# Shared OUI Carrier Resolution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 防止共享厂商 OUI 被固定归入 CMCC，同时保留唯一 OUI 和未知 OUI 的现有兼容行为。

**Architecture:** CarrierRegistry 根据 OUI 收集候选运营商，再用 ProductClass 缩小候选；唯一候选成功，多个候选返回显式歧义。Inform 自动注册消费完整设备身份，CSV 预登记在缺少 ProductClass 时只接受唯一 OUI 候选。

**Tech Stack:** Go、CarrierRegistry、TR-069 Inform、table-driven tests、testify。

## Global Constraints

- 不按注册顺序选择共享 OUI。
- 不自动修改已入库设备 carrier。
- 显式 carrier 始终优先。
- 未知 OUI 保留部署默认 carrier 兼容性。

---

### Task 1: 注册表身份解析契约

**Files:**
- Modify: `omcgo/internal/core/carrier/registry.go`
- Modify: `omcgo/internal/core/carrier/registry_test.go`

**Interfaces:**
- Produces: `ResolveByIdentity(oui, productClass string) (model.CarrierCode, error)`
- Produces: 可由上层识别的共享 OUI 歧义错误。

- [x] 先增加共享 OUI 分别被 CMCC/CTCC ProductClass 唯一解析、缺失 ProductClass 返回歧义、独占 OUI 正常解析的失败测试。
- [x] 运行 `go test -count=1 ./internal/core/carrier`，确认测试因现有首项选择逻辑失败。
- [x] 实现候选收集和唯一匹配，不改变注册顺序用于其他 Registry 行为。
- [x] 重跑聚焦测试并确认通过。

### Task 2: Inform 自动注册路径

**Files:**
- Modify: `omcgo/internal/device/inform_handler.go`
- Modify: `omcgo/internal/device/inform_handler_test.go`

**Interfaces:**
- Consumes: `CarrierRegistry.ResolveByIdentity`。
- Produces: `resolveCarrier(oui, productClass string) (model.CarrierCode, error)`。

- [x] 先增加 CTCC 共享 OUI 组合不会落入 CMCC，以及歧义身份不会回退默认 carrier 的失败测试。
- [x] 运行对应 device 测试确认红灯原因正确。
- [x] 将 bootstrap、reboot auto-register、periodic auto-register 等所有新建设备入口传入 ProductClass 并处理歧义错误。
- [x] 保持已存在设备更新路径不重新识别 carrier。
- [x] 重跑 device 聚焦测试并确认通过。

### Task 3: CSV 预登记歧义处理

**Files:**
- Modify: `omcgo/internal/device/device_service.go`
- Modify: `omcgo/internal/device/device_handler_batch_preregister.go`
- Test: 相应 batch preregister 测试文件。

**Interfaces:**
- Consumes: `CarrierRegistry.ResolveByIdentity(oui, "")`。
- Produces: 共享 OUI 未显式 carrier 时返回可定位的 `carrier_required` 行错误。

- [x] 先增加共享 OUI 未填 carrier 失败、显式 CTCC 成功的测试。
- [x] 运行测试确认现有实现错误选择 CMCC。
- [x] 最小修改服务与 handler，保留显式 carrier 优先级。
- [x] 重跑聚焦测试。

### Task 4: 完整验证与提交

**Files:**
- Verify only。

- [x] 运行 `gofmt`。
- [x] 运行 `go test -count=1 ./internal/core/carrier ./internal/device`。
- [x] 运行 `go test ./...`。
- [x] 运行 `go build ./...`。
- [x] 检查 diff 只包含共享 OUI 修复、测试和本文档。
- [x] 使用 Conventional Commit 提交，随后按用户流程推送并创建 MR。

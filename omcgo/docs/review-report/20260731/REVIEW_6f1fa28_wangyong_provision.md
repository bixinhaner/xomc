# 代码审查报告：BOOT 触发可靠全量参数同步

- 日期：2026-07-31
- 基线：`6f1fa2811`
- 范围：`provider`、`paramsync`、`provision`、设备在线事件
- 结论：PASS

## 变更概述

- 将开发和生产参数同步路由切换为 `durable`。
- 将 `device.online` 和固件变化伴随上线的恢复动作直接提交到可靠的全量参数同步生命周期。
- 删除注册/上线场景的旧 Path B MAC 定向入口，避免两条同步链路并存。
- BOOT/设备上线请求遇到自动退避时持久化为 `queued`，待调度器到期继续执行。
- 提交失败或设备查询失败时释放短时节流键，使消息重投可以再次提交；稳定事件 ID 用于幂等。

## 审查结果

### CRITICAL

无。

### WARNING

无。

### INFO

- 保留对历史 `sync-gpv-partial-*` 任务结果的识别，仅用于兼容已经在队列中的旧任务；本次变更不再创建该类任务。
- 生产路由切换为 durable 会扩大到所有启用设备，已通过完整后端测试、构建和真实设备 BOOT 验证覆盖主链路。

## 检查项

- SQL：继续使用 Squirrel 构建更新语句，无字符串拼接 SQL。
- 错误处理：新增错误均带设备/操作上下文并使用 `%w` 包装底层错误。
- 幂等与重投：使用事件 ID 构造稳定幂等键；提交前失败不会被 Redis 节流或事件去重永久吞掉。
- 终态：自动退避的 BOOT 请求保持非终态并可延迟调度；进入运行态时清理旧结果码、错误和完成时间。
- 路由：非 durable 模式 fail-closed，不回退旧 Path B。
- 测试：覆盖直接 durable 提交、退避排队、消息重投、节流释放、固件变化伴随上线及旧入口移除。

## 验证

- `git diff --check`：通过。
- `cd omcgo && go test ./...`：通过。
- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test -count=1 ./cmd/app/provider ./internal/paramsync ./internal/provision`：通过。
- 设备 `1202000534228JB0007` 上报 `1 BOOT` 后触发 `device_online` 全量同步：最终 `27/27` 任务完成，失败 `0`，请求和运行状态均为 `succeeded`。

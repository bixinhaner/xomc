# Issue #353 PM recovery 无效任务版本兜底真实验收

## 测试目标

证明主库中不存在任务版本、但 TSDB 和 Redis 仍有运行态时，worker 会将窗口隔离为终态并清理 Redis，且不会创建历史回放 consumer 或在下一轮重复恢复。

## 测试环境

- 日期：2026-08-20
- 环境：本机 OMC Docker 全栈
- 分支：`fix/353-pm-recovery-invalid-version`
- 基线提交：`9b9a1c36f784a7a12d1904366d6677676814b6ae`
- 数据库：清空主库、TSDB 与 Redis 后，由当前分支基线迁移重新创建

## 测试数据

- task_id：`35300000-0000-4000-8000-000000000001`
- task_version_id：`35300000-0000-4000-8000-000000000002`
- entity_key：`ISSUE353_ORPHAN`
- 窗口：`[2026-08-18T00:00:00Z, 2026-08-18T01:00:00Z)`
- 初始窗口状态：`open`
- Redis 哨兵：窗口 meta/seen/slots，以及版本级 definitions key

## 前置条件

- 主库目标 task_version_id 数量：`0`
- TSDB 目标窗口：`open|||false`
- Redis 目标版本 key：`4` 个

## 验收结果

- worker 启动恢复后，窗口状态为 `orphaned`。
- `recovery_original_status` 为 `open`。
- `recovery_terminal_reason` 为 `task version missing from primary database`。
- `recovery_terminal_at` 与 `runtime_cleaned_at` 均已写入。
- 目标版本的窗口级 key 和版本 definitions key 数量为 `0`。
- worker 汇总日志只记录一次窗口隔离和一次运行态清理；隔离 `1` 个窗口，清理 `1` 个窗口，清理耗时约 `2.5ms`。
- 日志中没有 `batch replay consumer`、逐窗口 `restore PM aggregation window` 错误或目标 task_version_id 的回放记录。
- 跨过下一轮 recovery 周期后，目标版本 active window 数量仍为 `0`，终态与清理标记保持不变。

## 自动化验证

- `go test ./internal/pm/stream -count=1`：通过。
- `go test ./internal/pm/... ./cmd/worker ./cmd/migrate -count=1`：通过。
- `go build ./...`：通过。

## PM 入口矩阵

本次不修改 PM 统计口径、页面聚合查询、定时聚合计算、自定义聚合计算、导出或启用指标旁路。验收仅覆盖 streaming recovery、TSDB 窗口终态与 Redis 运行态，因此页面浏览器验证不适用。

## 清理方式

验收完成后删除 task_version_id 为 `35300000-0000-4000-8000-000000000002` 的 TSDB 测试窗口。Redis 哨兵已由被测 recovery 自动清理。

# Issue #354 PM 聚合任务生命周期闭环真实验收

## 测试目标

证明 PM streaming task 在部署/reconcile、规则变化、source 删除、运行态收敛和同 ID source 恢复时保持稳定 task_id、不可变版本历史和最终一致性。

## 测试环境

- 日期：2026-08-20
- 环境：本机 OMC Docker 全栈
- 分支：`fix/354-pm-task-lifecycle-closure`
- 基线：包含 MR !666、!667 的 `main`
- 主库：清空后由当前分支 schema/seed 重新创建

## 测试数据

- 测试设备：`ISSUE354_DEVICE`，device_id `35400000-0000-4000-8000-000000000010`。
- source/streaming task_id：`35400000-0000-4000-8000-000000000001`。
- 初始 counter：`C000000001`；变更 counter：`C000000002`。
- TSDB active window：version 2、device 维度、hourly。
- Redis：active window meta 哨兵。
- 历史结果：`ISSUE354_RESULT`。

## 验收结果

### 稳定 ID 与无变化 reconcile

- seed 中 12 个内置 continuous source task 全部存在同 ID streaming task：`12/12`，缺失 `0`。
- 自建 source 首次启动 reconcile 后，streaming task_id 与 source ID 完全相同。
- 首次生成 version 1；不改规则重启 worker 后 current_version_id 和版本总数均不变。
- `source_updated_at` 与 source task 的 `updated_at` 一致，无漂移任务不会重复解析规则和成员。

### 规则变化与版本历史

- metric_paths 从 `C000000001` 改为 `C000000002` 后，task_id 不变。
- 新建 version 2 并成为 current version。
- version 1 写入 `effective_to`，version 2 保持打开。
- 退役发生在版本尚未生效时，`effective_to` 使用 `GREATEST(effective_from, retire_at)`，两代版本均满足 `effective_to >= effective_from`。

### source 删除与运行态收敛

- 直接删除 source 定义、未发送 PM control event，worker 启动 DB reconcile 仍发现 missing source。
- streaming task 只软退役：task 行和两代版本均保留，task `enabled=false`、`deleted_at` 已写入。
- version 2 的 TSDB active window 收敛为 `retired`，原状态为 `open`，原因是 `task was deleted`。
- 对应 Redis meta 清理完成，未创建 batch replay consumer。
- `ISSUE354_RESULT` 历史结果仍为 1 条，没有被任务删除或退役动作物理删除。

### 同 ID source 恢复

- 使用原 source task_id 恢复定义后，没有创建新的 task ID。
- 原 streaming task 清除 deleted_at 并重新启用。
- current_version_id 指向新建 version 3；version 1、2 继续保留且已关闭。
- version 3 打开，`source_updated_at` 再次与 source 定义一致。

## 自动化验证

- 废弃内置任务 SQL 断言只执行 UPDATE 软退役，不允许 DELETE。
- missing-source 查询使用同事务 `NOT EXISTS pm_tasks`、`LIMIT`、`FOR UPDATE SKIP LOCKED`。
- lifecycle drift 查询覆盖 streaming task 缺失、升级后 source 标记缺失、enabled 不一致和 deleted task 恢复；不使用会被运行进度频繁刷新的 `pm_tasks.updated_at` 做周期重算门禁。
- planned_end_at 与 canceled 状态的 enabled 判定均有测试。
- 退役时间早于未来 effective_from 的边界有回归测试。
- `go test ./internal/pm/... ./cmd/worker ./cmd/migrate -count=1`：通过。
- `go build ./...`：通过。

## 控制事件丢失兜底

本次删除 source 定义时没有发布 PM control event。worker 启动后依靠主库 reconcile 软退役任务；snapshot/recovery 再按数据库状态收敛 TSDB 和 Redis，证明 NATS control event 不是唯一保障。

## 真实页面保存验证

内置 Browser surface 不可用后，按 browser-control 规则降级到可见系统 Chrome + Playwright，并保留最终页面现场。

- 本机主库重建后通过 `/license` 真实页面安装项目规定的新系统样例 `omc.lic`：`POST /api/v1/system-license` 返回 `201`，随后 GET 返回 `200`，无 console/page error。
- 页面扫描确认 URL `/performance/pm-adhoc`，导航包含性能仪表盘、自定义聚合、设备性能查看、指标查询；页面分内置任务和自建任务区域。
- 内置任务 `0184dddd-0003-4000-8000-000000000003`：
  - 加入 `KGSM0110` 后，`PUT /api/v1/pm/adhoc/tasks/:id` 返回 `200`，`metric_paths` 为原 3 项加 `KGSM0110`。
  - 通过同一 UI 移除 `KGSM0110` 并恢复原 3 项，第二次 PUT 返回 `200`，页面提示“指标已更新”。
- 自建 continuous 任务：
  - 通过 5 步向导创建 `ISSUE354_UI_TASK`，选择 network 维度、LTE、4 个固定粒度和 `C000020013`；POST 返回 `201`。
  - 通过编辑向导追加 `C000020014`；PUT 返回 `200`，页面提示“新版本将从下一个完整聚合窗口生效”。
  - source task ID 与 streaming task ID 相同；数据库从 version 1 生成 version 2，version 1 写入 `effective_to`。
  - 通过页面取消任务后 DELETE 返回 `200`；再删除定义，`DELETE /definition` 返回 `200`。
  - 定义删除后 source 行为 0，stream task 软退役，取消形成的第 3 代 disabled version 与前两代版本全部保留并关闭。
- 所有页面动作均无 console error 和 page error；测试任务、临时设备及其 stream 历史随后已清理。

## PM 入口矩阵

本次不修改 PM 指标统计口径、页面聚合查询、定时聚合计算、自定义聚合结果读取、导出或启用指标旁路。入口矩阵中与本次直接相关的“自建任务创建/编辑保存”和“内置任务编辑指标保存”已通过真实浏览器验证；其余查询/导出入口不受代码改动影响。

## 清理方式

验收完成后删除 ID/名称/metric_id 带 `ISSUE354` 的 source task、streaming task、设备、TSDB window/result 和 Redis 哨兵。

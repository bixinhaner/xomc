# Issue 78 历史告警固定 01:08 清理设计

## 目标

将 `alarms_history` 的 TimescaleDB retention policy 固定为每天 `01:08 Asia/Shanghai` 执行，让配置历史告警保留天数后的下一次后台清理时间可预测。

## 问题边界

Issue 78 已确认不是保留天数未保存，而是配置保存只会重建异步 retention policy，截图时后台任务尚未执行。页面展示 `raised_at`，retention 按超表分区字段 `time` 删除完整过期 chunk；本次不改变这两个既有口径。

## 方案

1. 运行时重建 `alarms_history` retention policy 时，继续使用当前配置的 `drop_after`，同时显式传入：
   - `schedule_interval => INTERVAL '1 day'`
   - `initial_start => TIMESTAMPTZ '2000-01-01 01:08:00+08'`
   - `timezone => 'Asia/Shanghai'`
2. 新增 TSDB 增量迁移，仅定位 `policy_retention + public.alarms_history` 对应的唯一 job，并用 `alter_job` 固定相同相位，使既有数据库不必等待再次保存配置。
3. Down 迁移只恢复该 job 的非固定日周期调度，不重建 policy、不改变 job `config`。

## 明确不做

- 不调用 `run_job`、`drop_chunks` 或逐行 `DELETE`，配置保存后不立即清理。
- 不修改 `drop_after`、保留天数范围、7 天 chunk interval。
- 不修改 `raised_at`、`cleared_at`、`time` 的业务含义。
- 不修改历史告警查询接口或前端。
- 不修改任何其他 retention、compression、continuous aggregate 或应用定时任务。

## 验证标准

- 单元测试证明运行时建 policy SQL 包含固定日周期、01:08 和 Asia/Shanghai。
- 迁移前后 `alarms_history` job 的 `config` 完全一致，仅调度属性变化。
- 迁移后该 job 为 fixed schedule，下一次执行对齐北京时间 01:08。
- Up/Down 均可在 disposable TimescaleDB 上执行。
- `go build ./...` 与历史告警 retention 相关测试通过。


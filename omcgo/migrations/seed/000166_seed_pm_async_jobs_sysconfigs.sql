-- +goose Up
-- +goose StatementBegin
-- T-0164 收尾 G7-Gap-5 + G8-Gap-3：sys_configs 暴露 adhoc retention + asyncjob 阈值
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.7 G7 / §4.8 G8
-- 实施 plan：docs/project/plan-T-0164-followup-gaps.md §G7-Gap-5, §G8-Gap-3
--
-- 4 个新 key：
--   - pm.adhoc.retention_days   — adhoc 聚合结果保留期（默认 365 / 1 年）
--   - asyncjob.heartbeat_interval_seconds — 心跳间隔（默认 30s）
--   - asyncjob.zombie_threshold_seconds  — 僵尸任务阈值（默认 300s = 5min）
--   - asyncjob.sweeper_interval_seconds  — Sweeper 扫描间隔（默认 60s）
--
-- 运行时通过 SysConfigSaved 事件 hook 通知 asyncjob.Sweeper Reload。

INSERT INTO sys_configs (category, key, value, value_type, description, is_public)
VALUES
  ('pm.adhoc', 'retention_days', '365', 'int', 'PM adhoc 聚合任务结果保留天数（默认 365d）', FALSE),
  ('asyncjob', 'heartbeat_interval_seconds', '30',  'int', '异步任务心跳上报间隔（秒）', FALSE),
  ('asyncjob', 'zombie_threshold_seconds',  '300', 'int', '心跳超过此值视为僵尸任务（秒）', FALSE),
  ('asyncjob', 'sweeper_interval_seconds',  '60',  'int', 'Sweeper 扫描僵尸任务的间隔（秒）', FALSE)
ON CONFLICT (category, key) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM sys_configs WHERE category IN ('pm.adhoc', 'asyncjob')
  AND key IN ('retention_days', 'heartbeat_interval_seconds', 'zombie_threshold_seconds', 'sweeper_interval_seconds');
-- +goose StatementEnd

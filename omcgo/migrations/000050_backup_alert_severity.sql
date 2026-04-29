-- +goose Up
-- T-0084 / R-102 followup: BackupPolicy.AlertSeverity column
--
-- 新增 alert_severity 列让运维分级响应：warning / major / critical。
-- 关闭 policy_alarm_publisher.go + policy_storage_alarm.go 两处 severity TODO。
-- DEFAULT 'major' 让既有行 backfill 与之前硬编码常量行为一致 — 升级零运维感知。
-- 单字段同时控制 backup_task_failed (T-0073) + backup_storage_threshold_exceeded (T-0082)
-- 两类告警的严重度（PRD §2.1 决策：单字段 vs 双字段；选 A 单字段）。

ALTER TABLE backup_policies
    ADD COLUMN IF NOT EXISTS alert_severity VARCHAR(16) NOT NULL DEFAULT 'major'
    CHECK (alert_severity IN ('warning', 'major', 'critical'));

-- +goose Down
ALTER TABLE backup_policies DROP COLUMN IF EXISTS alert_severity;

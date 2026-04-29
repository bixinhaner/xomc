-- +goose Up
-- T-0071 / R-102 followup: backup_policies — 单实例策略持久化层
--
-- 本表持久化 19 个备份策略字段（保留 / 自动清理 / 压缩 / 存储 / 加密 / 告警 6 类）。
-- 应用层保证单实例语义（GET/Upsert ORDER BY updated_at DESC LIMIT 1 / WHERE id=...），
-- DB 不加 unique 约束，留 future per-tenant 维度扩展空间。
--
-- ⚠ Enforcement boundary: 本任务仅交付持久化层。各类的实际生效（cleanup cron / 压缩 /
-- 加密 / 磁盘监控 / 告警发送）由后续 followup 任务（T-0073/0074/0075）逐项完成。
-- 详见 `docs/project/prd/T-0071-backup-policy-persistence.md` §2.

CREATE TABLE IF NOT EXISTS backup_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 保留策略
    retention_days INT NOT NULL DEFAULT 30
        CHECK (retention_days BETWEEN 1 AND 3650),
    max_backup_count INT NOT NULL DEFAULT 100
        CHECK (max_backup_count >= 1),
    min_backup_count INT NOT NULL DEFAULT 3
        CHECK (min_backup_count >= 1),

    -- 自动清理（enforcement = T-0073）
    auto_cleanup BOOLEAN NOT NULL DEFAULT true,
    cleanup_time VARCHAR(8) NOT NULL DEFAULT '03:00',
    cleanup_day_of_week INT NOT NULL DEFAULT -1
        CHECK (cleanup_day_of_week BETWEEN -1 AND 6),
    keep_last_n INT NOT NULL DEFAULT 5
        CHECK (keep_last_n >= 1),

    -- 压缩（enforcement = T-0074）
    enable_compression BOOLEAN NOT NULL DEFAULT true,
    compression_level INT NOT NULL DEFAULT 6
        CHECK (compression_level BETWEEN 1 AND 9),
    compression_format VARCHAR(16) NOT NULL DEFAULT 'gzip'
        CHECK (compression_format IN ('gzip', 'bzip2', 'lz4', 'zstd')),

    -- 存储
    storage_backend VARCHAR(16) NOT NULL DEFAULT 'local'
        CHECK (storage_backend IN ('local', 'ftp', 'sftp', 'nfs')),
    ftp_config_id UUID REFERENCES ftp_configs(id) ON DELETE SET NULL,
    local_path TEXT NOT NULL DEFAULT '/var/backup/omc',
    max_storage_gb INT NOT NULL DEFAULT 500
        CHECK (max_storage_gb >= 1),

    -- 加密（enforcement = T-0075，安全敏感需 SecOps 评审）
    enable_encryption BOOLEAN NOT NULL DEFAULT false,
    encryption_algorithm VARCHAR(32) NOT NULL DEFAULT 'AES-256-GCM',

    -- 告警（enforcement = T-0073，集成 F04 邮件通道 T-0007 已就绪）
    alert_on_failure BOOLEAN NOT NULL DEFAULT true,
    alert_email VARCHAR(256) NOT NULL DEFAULT '',
    alert_threshold_percent INT NOT NULL DEFAULT 80
        CHECK (alert_threshold_percent BETWEEN 50 AND 95),

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS backup_policies;

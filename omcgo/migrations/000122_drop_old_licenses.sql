-- F06 System License 重构 P1 Step 5 — DROP 老 multi-license 模型表。
--
-- PRD: docs/project/prd/F06-system-license-redesign.md §8.1
-- 决策: 2026-05-18 用户拍板"方案 A 完全替代，现有 licenses 数据 DROP 不要"
--
-- 现状（删除前）：
--   - licenses              (migration 000006 创建, 000043 加 enforcement 列)
--   - license_logs          (migration 000073 创建, 000074 加 auto_revoke)
--
-- 新模型已在 migration 000121 落地（system_license + system_license_history），
-- 应用层全部切到 singleton 链路，老表已 0 caller。

-- +goose Up

-- 主表 + 审计表用 CASCADE 一并清理（依赖索引 / 触发器 / 历史外键全 cascade）。
DROP TABLE IF EXISTS license_logs CASCADE;
DROP TABLE IF EXISTS licenses     CASCADE;

-- +goose Down

-- Down 仅恢复表骨架（无 enforcement 扩展列），数据无法找回（PRD §11 fresh start）。
-- 若需要完整回滚到 000043 / 000074 状态，请按版本号逐个 goose down 而非依赖本 Down 段。
CREATE TABLE IF NOT EXISTS licenses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_name VARCHAR(200) NOT NULL,
    license_code VARCHAR(200) NOT NULL UNIQUE,
    product_name VARCHAR(200) NOT NULL,
    license_type VARCHAR(50)  NOT NULL,
    status       VARCHAR(50)  NOT NULL DEFAULT 'pending',
    max_devices  INTEGER      NOT NULL DEFAULT 0,
    used_devices INTEGER      NOT NULL DEFAULT 0,
    features     JSONB        NOT NULL DEFAULT '[]',
    issue_date   TIMESTAMPTZ  NOT NULL,
    expiry_date  TIMESTAMPTZ,
    licensor     VARCHAR(200),
    device_type  VARCHAR(50),
    region       VARCHAR(100),
    notes        TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS license_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id    UUID,
    log_type      VARCHAR(50) NOT NULL,
    actor_user_id UUID,
    result        VARCHAR(20) NOT NULL,
    details       JSONB       NOT NULL DEFAULT '{}',
    client_ip     VARCHAR(100),
    user_agent    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

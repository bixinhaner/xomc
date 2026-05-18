-- +goose Up
-- ============================================================
-- 000121_system_license.sql
-- F06 System License 重构 P1 Step 1 — 新增 system_license + history 双表
--
-- PRD: docs/project/prd/F06-system-license-redesign.md §3 数据模型
--
-- 业务模型转换：multi-license (licenses 表) → singleton system_license。
-- 老 licenses + license_logs 表本 migration **不动**，等 Step 5 真 DROP
-- （这样老链路在过渡期仍可工作，便于回滚）。
--
-- singleton 约束实现：partial UNIQUE INDEX ON (is_current) WHERE is_current=true
--   → 最多 1 行 is_current=true；is_current=false 可有多行（历史副本）
-- ============================================================

-- ---------- system_license: 当前生效 license（最多 1 行 is_current=true）----------
CREATE TABLE IF NOT EXISTS system_license (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id          VARCHAR(100) NOT NULL UNIQUE,    -- 厂商签发全局 ID（如 NO2022-03-14002）
    license_type        VARCHAR(50)  NOT NULL,           -- Commercial / Trial / Evaluation / Internal
    issuer              VARCHAR(200),                    -- 签发方（Baicells OEM）
    licensee            VARCHAR(200),                    -- 授权对象（客户公司 / 部署 ID）
    issued_at           TIMESTAMPTZ  NOT NULL,
    expiry_date         TIMESTAMPTZ,                     -- NULL = 永久
    devices_support     JSONB NOT NULL DEFAULT '{}'::jsonb,
        -- {"eNB":10000,"gNB":10000,"CPE":10000,"WCG":1000,"UPS":1000}
    feature_list        JSONB NOT NULL DEFAULT '{}'::jsonb,
        -- 三级嵌套，详见 PRD §4
    raw_content         TEXT  NOT NULL,                  -- 原始 license 文件全文（审计 + 重新验签）
    signature           TEXT,                            -- base64 RSA-PSS 签名（可空：未签场景）
    signature_key_id    VARCHAR(128),                    -- OEM 公钥 SHA-256 fingerprint
    signature_status    VARCHAR(20) NOT NULL,            -- verified / unverified / invalid
    uploaded_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by_user_id UUID,                            -- 谁 Update 的（弱引用，不加 FK）
    is_current          BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_signature_status CHECK (signature_status IN ('verified', 'unverified', 'invalid')),
    CONSTRAINT chk_license_type     CHECK (license_type IN ('Commercial', 'Trial', 'Evaluation', 'Internal'))
);

-- singleton 约束：partial unique index — 最多 1 行 is_current=true
CREATE UNIQUE INDEX IF NOT EXISTS uq_system_license_current
    ON system_license(is_current) WHERE is_current = true;

CREATE INDEX IF NOT EXISTS idx_system_license_expiry
    ON system_license(expiry_date) WHERE is_current = true;
CREATE INDEX IF NOT EXISTS idx_system_license_uploaded
    ON system_license(uploaded_at DESC);

CREATE TRIGGER trigger_system_license_updated_at
    BEFORE UPDATE ON system_license FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE  system_license IS 'Singleton system-wide license; 最多 1 行 is_current=true，由 partial unique 约束';
COMMENT ON COLUMN system_license.license_id  IS '厂商签发的全局唯一 ID，例如 NO2022-03-14002';
COMMENT ON COLUMN system_license.devices_support IS '设备类型 → 配额 map，例如 {"eNB":10000,"gNB":10000}';
COMMENT ON COLUMN system_license.feature_list IS '三级嵌套功能矩阵，详见 PRD §4';
COMMENT ON COLUMN system_license.raw_content IS '原始 license 文件全文，用于审计与重新验签';

-- ---------- system_license_history: 每次 Update 留档 ----------
CREATE TABLE IF NOT EXISTS system_license_history (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id           VARCHAR(100) NOT NULL,           -- 不加 UNIQUE：同 license_id 可历史多次替换记录
    license_type         VARCHAR(50)  NOT NULL,
    issuer               VARCHAR(200),
    licensee             VARCHAR(200),
    issued_at            TIMESTAMPTZ NOT NULL,
    expiry_date          TIMESTAMPTZ,
    devices_support      JSONB NOT NULL,
    feature_list         JSONB NOT NULL,
    raw_content          TEXT  NOT NULL,
    signature            TEXT,
    signature_key_id     VARCHAR(128),
    signature_status     VARCHAR(20) NOT NULL,
    uploaded_at          TIMESTAMPTZ NOT NULL,             -- 当时上传时间
    uploaded_by_user_id  UUID,
    replaced_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- 被新 license 替换的时间
    replaced_by_id       UUID,                                -- 替换它的新 system_license.id；ON DELETE SET NULL
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_replaced_by FOREIGN KEY (replaced_by_id) REFERENCES system_license(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_system_license_history_license_id
    ON system_license_history(license_id);
CREATE INDEX IF NOT EXISTS idx_system_license_history_uploaded
    ON system_license_history(uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_license_history_replaced
    ON system_license_history(replaced_at DESC);

COMMENT ON TABLE system_license_history IS '每次 Update 把旧的 current license 拷贝进本表，留作合规审计';

-- ============================================================
-- 注：老 licenses / license_logs 表本 migration **不动**，本步骤是 P1 Step 1
--     "新增不删"。Step 5（前端切换完成 + 单 sprint 验证后）才 DROP 老表。
-- ============================================================

-- +goose Down
DROP INDEX IF EXISTS idx_system_license_history_replaced;
DROP INDEX IF EXISTS idx_system_license_history_uploaded;
DROP INDEX IF EXISTS idx_system_license_history_license_id;
DROP TABLE IF EXISTS system_license_history;

DROP TRIGGER IF EXISTS trigger_system_license_updated_at ON system_license;
DROP INDEX IF EXISTS idx_system_license_uploaded;
DROP INDEX IF EXISTS idx_system_license_expiry;
DROP INDEX IF EXISTS uq_system_license_current;
DROP TABLE IF EXISTS system_license;

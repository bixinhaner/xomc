-- +goose Up
-- KPI Indicator Management: 17 tables for indicator lifecycle management

-- ============================================================
-- Table: indicator_unit (单位参考表)
-- ============================================================
CREATE TABLE IF NOT EXISTS indicator_unit (
    id          VARCHAR(50)  NOT NULL,
    en_name     VARCHAR(100),
    cn_name     VARCHAR(100),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT pk_indicator_unit PRIMARY KEY (id)
);

-- ============================================================
-- Table: indicator_group_enb (指标功能集树)
-- ============================================================
CREATE TABLE IF NOT EXISTS indicator_group_enb (
    id              VARCHAR(32)  NOT NULL,
    en_name         VARCHAR(200),
    operator_code   VARCHAR(100),
    is_build_in     CHAR(1)      NOT NULL DEFAULT '0',
    description     TEXT,
    parent_id       VARCHAR(32)  NOT NULL,
    cn_name         VARCHAR(200),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_indicator_group_enb PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_indicator_group_enb_parent_id ON indicator_group_enb(parent_id);

-- ============================================================
-- Table: indicator_group_gsm (指标功能集树)
-- ============================================================
CREATE TABLE IF NOT EXISTS indicator_group_gsm (
    id              VARCHAR(32)  NOT NULL,
    en_name         VARCHAR(200),
    operator_code   VARCHAR(100),
    is_build_in     CHAR(1)      NOT NULL DEFAULT '0',
    description     TEXT,
    parent_id       VARCHAR(32)  NOT NULL,
    cn_name         VARCHAR(200),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_indicator_group_gsm PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_indicator_group_gsm_parent_id ON indicator_group_gsm(parent_id);

-- ============================================================
-- Table: indicator_group_gnb (指标功能集树)
-- ============================================================
CREATE TABLE IF NOT EXISTS indicator_group_gnb (
    id              VARCHAR(32)  NOT NULL,
    en_name         VARCHAR(200),
    operator_code   VARCHAR(100),
    is_build_in     CHAR(1)      NOT NULL DEFAULT '0',
    description     TEXT,
    parent_id       VARCHAR(32)  NOT NULL,
    cn_name         VARCHAR(200),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_indicator_group_gnb PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_indicator_group_gnb_parent_id ON indicator_group_gnb(parent_id);

-- ============================================================
-- Table: perf_indicators_enb
-- ============================================================
CREATE TABLE IF NOT EXISTS perf_indicators_enb (
    id                  VARCHAR(20)  NOT NULL,
    en_name             VARCHAR(200) NOT NULL,
    cn_name             VARCHAR(200) NOT NULL,
    en_description      TEXT,
    cn_description      TEXT,
    group_id            VARCHAR(32)  NOT NULL,
    operator_code       VARCHAR(100),
    data_type           VARCHAR(20),
    unit_id             VARCHAR(50),
    updator             VARCHAR(64),
    is_build_in         CHAR(1)      NOT NULL DEFAULT '0',
    is_counter          CHAR(1)      NOT NULL DEFAULT '1',
    arithmetic          TEXT,
    statis_type         VARCHAR(20),
    calculating_status  VARCHAR(20),
    product_types       TEXT,
    indicator_level     VARCHAR(20),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_perf_indicators_enb PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_enb_group_id ON perf_indicators_enb(group_id);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_enb_is_build_in ON perf_indicators_enb(is_build_in);

-- ============================================================
-- Table: perf_indicators_gsm
-- ============================================================
CREATE TABLE IF NOT EXISTS perf_indicators_gsm (
    id                  VARCHAR(20)  NOT NULL,
    en_name             VARCHAR(200) NOT NULL,
    cn_name             VARCHAR(200) NOT NULL,
    en_description      TEXT,
    cn_description      TEXT,
    group_id            VARCHAR(32)  NOT NULL,
    operator_code       VARCHAR(100),
    data_type           VARCHAR(20),
    unit_id             VARCHAR(50),
    updator             VARCHAR(64),
    is_build_in         CHAR(1)      NOT NULL DEFAULT '0',
    is_counter          CHAR(1)      NOT NULL DEFAULT '1',
    arithmetic          TEXT,
    statis_type         VARCHAR(20),
    calculating_status  VARCHAR(20),
    product_types       TEXT,
    indicator_level     VARCHAR(20),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_perf_indicators_gsm PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_gsm_group_id ON perf_indicators_gsm(group_id);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_gsm_is_build_in ON perf_indicators_gsm(is_build_in);

-- ============================================================
-- Table: perf_indicators_gnb
-- ============================================================
CREATE TABLE IF NOT EXISTS perf_indicators_gnb (
    id                  VARCHAR(20)  NOT NULL,
    en_name             VARCHAR(200) NOT NULL,
    cn_name             VARCHAR(200) NOT NULL,
    en_description      TEXT,
    cn_description      TEXT,
    group_id            VARCHAR(32)  NOT NULL,
    operator_code       VARCHAR(100),
    data_type           VARCHAR(20),
    unit_id             VARCHAR(50),
    updator             VARCHAR(64),
    is_build_in         CHAR(1)      NOT NULL DEFAULT '0',
    is_counter          CHAR(1)      NOT NULL DEFAULT '1',
    arithmetic          TEXT,
    statis_type         VARCHAR(20),
    calculating_status  VARCHAR(20),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_perf_indicators_gnb PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_gnb_group_id ON perf_indicators_gnb(group_id);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_gnb_is_build_in ON perf_indicators_gnb(is_build_in);

-- ============================================================
-- Table: rela_platform_indicator_formula_enb
-- ============================================================
CREATE TABLE IF NOT EXISTS rela_platform_indicator_formula_enb (
    id              UUID         NOT NULL DEFAULT gen_random_uuid(),
    platform_name   VARCHAR(64)  NOT NULL,
    indicator_id    VARCHAR(20)  NOT NULL,
    formula         TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_rela_platform_indicator_formula_enb PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_rela_platform_indicator_formula_enb_platform_indicator ON rela_platform_indicator_formula_enb(platform_name, indicator_id);

-- ============================================================
-- Table: rela_platform_indicator_formula_gsm
-- ============================================================
CREATE TABLE IF NOT EXISTS rela_platform_indicator_formula_gsm (
    id              UUID         NOT NULL DEFAULT gen_random_uuid(),
    platform_name   VARCHAR(64)  NOT NULL,
    indicator_id    VARCHAR(20)  NOT NULL,
    formula         TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_rela_platform_indicator_formula_gsm PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_rela_platform_indicator_formula_gsm_platform_indicator ON rela_platform_indicator_formula_gsm(platform_name, indicator_id);

-- ============================================================
-- Table: rela_platform_indicator_formula_gnb
-- ============================================================
CREATE TABLE IF NOT EXISTS rela_platform_indicator_formula_gnb (
    id              UUID         NOT NULL DEFAULT gen_random_uuid(),
    platform_name   VARCHAR(64)  NOT NULL,
    indicator_id    VARCHAR(20)  NOT NULL,
    formula         TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_rela_platform_indicator_formula_gnb PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_rela_platform_indicator_formula_gnb_platform_indicator ON rela_platform_indicator_formula_gnb(platform_name, indicator_id);

-- ============================================================
-- Table: enabled_pm_indicators_enb
-- ============================================================
CREATE TABLE IF NOT EXISTS enabled_pm_indicators_enb (
    operator_code   VARCHAR(100) NOT NULL,
    indicator_id    VARCHAR(20)  NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_enabled_pm_indicators_enb PRIMARY KEY (operator_code, indicator_id)
);

-- ============================================================
-- Table: enabled_pm_indicators_gsm
-- ============================================================
CREATE TABLE IF NOT EXISTS enabled_pm_indicators_gsm (
    operator_code   VARCHAR(100) NOT NULL,
    indicator_id    VARCHAR(20)  NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_enabled_pm_indicators_gsm PRIMARY KEY (operator_code, indicator_id)
);

-- ============================================================
-- Table: enabled_pm_indicators_gnb
-- ============================================================
CREATE TABLE IF NOT EXISTS enabled_pm_indicators_gnb (
    operator_code   VARCHAR(100) NOT NULL,
    indicator_id    VARCHAR(20)  NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_enabled_pm_indicators_gnb PRIMARY KEY (operator_code, indicator_id)
);

-- ============================================================
-- Table: perf_template_rel_arithmetic (模板-指标关联)
-- ============================================================
CREATE TABLE IF NOT EXISTS perf_template_rel_arithmetic (
    id              UUID         NOT NULL DEFAULT gen_random_uuid(),
    temp_id         VARCHAR(32)  NOT NULL,
    indicator_id    VARCHAR(20)  NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_perf_template_rel_arithmetic PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_perf_template_rel_arithmetic_temp_id ON perf_template_rel_arithmetic(temp_id);

-- ============================================================
-- Table: perf_cust_name (指标自定义名称)
-- ============================================================
CREATE TABLE IF NOT EXISTS perf_cust_name (
    operator_code   VARCHAR(100) NOT NULL,
    perf_id         VARCHAR(20)  NOT NULL,
    cust_name       VARCHAR(200),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_perf_cust_name PRIMARY KEY (operator_code, perf_id)
);

-- ============================================================
-- Table: indicator_threshold (指标门限)
-- ============================================================
CREATE TABLE IF NOT EXISTS indicator_threshold (
    id                  UUID         NOT NULL DEFAULT gen_random_uuid(),
    indicator_id        VARCHAR(20),
    threshold_period    VARCHAR(10),
    threshold_color     VARCHAR(10),
    threshold_low       VARCHAR(20),
    threshold_high      VARCHAR(20),
    threshold_level     VARCHAR(10),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_indicator_threshold PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_indicator_threshold_indicator_id ON indicator_threshold(indicator_id);

-- ============================================================
-- Table: perf_alarm_threshold (告警模板阈值)
-- ============================================================
CREATE TABLE IF NOT EXISTS perf_alarm_threshold (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    temp_id          VARCHAR(32),
    indicator_id     VARCHAR(20),
    comparison       VARCHAR(8),
    threshold_value  VARCHAR(20),
    comparison2      VARCHAR(8),
    threshold_value2 VARCHAR(20),
    operation        VARCHAR(50),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
,    CONSTRAINT pk_perf_alarm_threshold PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_perf_alarm_threshold_temp_id ON perf_alarm_threshold(temp_id);

-- ============================================================
-- Triggers: updated_at auto-update
-- ============================================================
CREATE TRIGGER trigger_indicator_unit_updated_at
    BEFORE UPDATE ON indicator_unit
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_indicator_group_enb_updated_at
    BEFORE UPDATE ON indicator_group_enb
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_indicator_group_gsm_updated_at
    BEFORE UPDATE ON indicator_group_gsm
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_indicator_group_gnb_updated_at
    BEFORE UPDATE ON indicator_group_gnb
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_perf_indicators_enb_updated_at
    BEFORE UPDATE ON perf_indicators_enb
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_perf_indicators_gsm_updated_at
    BEFORE UPDATE ON perf_indicators_gsm
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_perf_indicators_gnb_updated_at
    BEFORE UPDATE ON perf_indicators_gnb
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_rela_platform_indicator_formula_enb_updated_at
    BEFORE UPDATE ON rela_platform_indicator_formula_enb
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_rela_platform_indicator_formula_gsm_updated_at
    BEFORE UPDATE ON rela_platform_indicator_formula_gsm
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_rela_platform_indicator_formula_gnb_updated_at
    BEFORE UPDATE ON rela_platform_indicator_formula_gnb
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_enabled_pm_indicators_enb_updated_at
    BEFORE UPDATE ON enabled_pm_indicators_enb
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_enabled_pm_indicators_gsm_updated_at
    BEFORE UPDATE ON enabled_pm_indicators_gsm
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_enabled_pm_indicators_gnb_updated_at
    BEFORE UPDATE ON enabled_pm_indicators_gnb
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_perf_template_rel_arithmetic_updated_at
    BEFORE UPDATE ON perf_template_rel_arithmetic
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_perf_cust_name_updated_at
    BEFORE UPDATE ON perf_cust_name
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_indicator_threshold_updated_at
    BEFORE UPDATE ON indicator_threshold
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_perf_alarm_threshold_updated_at
    BEFORE UPDATE ON perf_alarm_threshold
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
-- Drop in reverse dependency order
DROP TABLE IF EXISTS perf_alarm_threshold CASCADE;
DROP TABLE IF EXISTS indicator_threshold CASCADE;
DROP TABLE IF EXISTS perf_cust_name CASCADE;
DROP TABLE IF EXISTS perf_template_rel_arithmetic CASCADE;
DROP TABLE IF EXISTS enabled_pm_indicators_gnb CASCADE;
DROP TABLE IF EXISTS enabled_pm_indicators_gsm CASCADE;
DROP TABLE IF EXISTS enabled_pm_indicators_enb CASCADE;
DROP TABLE IF EXISTS rela_platform_indicator_formula_gnb CASCADE;
DROP TABLE IF EXISTS rela_platform_indicator_formula_gsm CASCADE;
DROP TABLE IF EXISTS rela_platform_indicator_formula_enb CASCADE;
DROP TABLE IF EXISTS perf_indicators_gnb CASCADE;
DROP TABLE IF EXISTS perf_indicators_gsm CASCADE;
DROP TABLE IF EXISTS perf_indicators_enb CASCADE;
DROP TABLE IF EXISTS indicator_group_gnb CASCADE;
DROP TABLE IF EXISTS indicator_group_gsm CASCADE;
DROP TABLE IF EXISTS indicator_group_enb CASCADE;
DROP TABLE IF EXISTS indicator_unit CASCADE;

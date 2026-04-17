-- +goose Up
-- ============================================================
-- 000022_mml_param_library.sql
-- MML 参数库表结构重构
-- 
-- 设计目标:
-- 1. 清晰定义 param_group 和 param 的关联关系(显式外键)
-- 2. 保留老版本所有业务字段和功能
-- 3. 使用 PostgreSQL 高级特性(JSONB、数组、生成列、LTREE)
-- 4. 完整的审计追踪(created_at, updated_at, created_by)
-- 5. 提供老数据迁移脚本和兼容视图
-- ============================================================

-- 启用 LTREE 扩展(用于树形路径查询)
CREATE EXTENSION IF NOT EXISTS ltree;

-- ============================================================
-- 1. 参数版本管理表
-- ============================================================
CREATE TABLE mml_param_versions (
    version_code            VARCHAR(50) PRIMARY KEY,
    version_name            VARCHAR(200) NOT NULL,
    description             TEXT,
    release_date            DATE,
    product_models          VARCHAR(200)[],
    software_versions       VARCHAR(100)[],
    is_active               BOOLEAN NOT NULL DEFAULT true,
    is_deprecated           BOOLEAN NOT NULL DEFAULT false,
    group_count             INT NOT NULL DEFAULT 0,
    param_count             INT NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 初始化老版本数据
INSERT INTO mml_param_versions (version_code, version_name, description, product_models) VALUES
('QB1.0', 'Qcells B1.0', '早期小站版本', ARRAY['Qcells-B100']),
('CA2.0', 'Celleagle A2.0', '中期版本', ARRAY['Celleagle-A200']),
('436Q1.0', '436 Q1.0', '特定型号版本', ARRAY['Baicells-436']),
('MLN1.0', 'Multi-mode LTE N1.0', '最新多模版本', ARRAY['Baicells-Neo']),
('MLQ1.0', 'Multi-mode LTE Q1.0', '多模LTE版本', ARRAY['Baicells-Neo']),
('BLX1.0', 'Baicells LTE X1.0', 'LTE版本', ARRAY['Baicells-BLX']),
('BAIBLQ1.0', 'Baicells BL Q1.0', 'Baicells BL版本', ARRAY['Baicells-BLQ']),
('BaiBNX1.0', 'Baicells N X1.0', 'NR版本', ARRAY['Baicells-NX']),
('CR4.0', 'CR 4.0', 'CR版本', ARRAY['Baicells-CR']),
('EA4.0', 'EA 4.0', 'EA版本', ARRAY['Baicells-EA']),
('EA4.0DUAL', 'EA 4.0 Dual', 'EA双模版本', ARRAY['Baicells-EA-Dual']),
('QC3.1', 'Qcells C3.1', 'Qcells版本', ARRAY['Qcells-C300']),
('QC4.2', 'Qcells C4.2', 'Qcells版本', ARRAY['Qcells-C400']),
('QC4.2T', 'Qcells C4.2T', 'Qcells T版本', ARRAY['Qcells-C400T']),
('BSC1.0', 'BSC 1.0', 'BSC版本', ARRAY['Baicells-BSC']),
('BTS1.0', 'BTS 1.0', 'BTS版本', ARRAY['Baicells-BTS']),
('DXDF1.0', 'DXDF 1.0', 'DXDF版本', ARRAY['Baicells-DXDF']),
('NBIOT1.0', 'NB-IoT 1.0', '物联网版本', ARRAY['Baicells-NB']),
('Nova430', 'Nova 430', 'Nova 430版本', ARRAY['Baicells-Nova430']),
('Nova430i', 'Nova 430i', 'Nova 430i版本', ARRAY['Baicells-Nova430i']),
('BaiBLN_3.0.3', 'Baicells BLN 3.0.3', 'BLN版本', ARRAY['Baicells-BLN']),
('BaiBLQ_3.0.2', 'Baicells BLQ 3.0.2', 'BLQ版本', ARRAY['Baicells-BLQ3']),
('ENB_DEFAULT_098', 'eNB Default 098', 'TR098默认', ARRAY['Baicells-eNB']),
('ENB_DEFAULT_181', 'eNB Default 181', 'TR181默认', ARRAY['Baicells-eNB']);

-- ============================================================
-- 2. 参数分组表
-- ============================================================
CREATE TABLE mml_param_groups (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_code              VARCHAR(100) NOT NULL,
    group_name_zh           VARCHAR(200) NOT NULL,
    group_name_en           VARCHAR(200),
    parent_id               UUID REFERENCES mml_param_groups(id) ON DELETE CASCADE,
    path                    LTREE,
    level                   INT NOT NULL DEFAULT 0,
    is_listable             BOOLEAN NOT NULL DEFAULT false,
    is_modifiable           BOOLEAN NOT NULL DEFAULT false,
    is_addable              BOOLEAN NOT NULL DEFAULT false,
    is_removable            BOOLEAN NOT NULL DEFAULT false,
    add_object_path         VARCHAR(500),
    delete_object_path      VARCHAR(500),
    param_version           VARCHAR(50) NOT NULL REFERENCES mml_param_versions(version_code),
    platform_support        VARCHAR(10)[],
    mobile_support          BOOLEAN NOT NULL DEFAULT true,
    broadband_support       BOOLEAN NOT NULL DEFAULT true,
    cell_number             INT NOT NULL DEFAULT 1,
    cell_index_location     INT NOT NULL DEFAULT 0,
    require_second_confirm  BOOLEAN NOT NULL DEFAULT false,
    confirm_message_zh      TEXT,
    confirm_message_en      TEXT,
    display_order           INT NOT NULL DEFAULT 0,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    created_by              VARCHAR(100),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT uq_group_version_code UNIQUE (param_version, group_code),
    CONSTRAINT chk_level CHECK (level >= 0)
);

-- 索引
CREATE INDEX idx_mml_param_groups_parent ON mml_param_groups(parent_id);
CREATE INDEX idx_mml_param_groups_version ON mml_param_groups(param_version);
CREATE INDEX idx_mml_param_groups_path ON mml_param_groups USING GIST(path);
CREATE INDEX idx_mml_param_groups_active ON mml_param_groups(param_version, is_active) WHERE is_active = true;
CREATE INDEX idx_mml_param_groups_deleted ON mml_param_groups(deleted_at) WHERE deleted_at IS NOT NULL;

-- 自动更新 updated_at
DROP TRIGGER IF EXISTS trigger_mml_param_groups_updated_at ON mml_param_groups;
CREATE TRIGGER trigger_mml_param_groups_updated_at
    BEFORE UPDATE ON mml_param_groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 3. 参数定义表
-- ============================================================
CREATE TABLE mml_params (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    param_code              VARCHAR(200) NOT NULL,
    param_name_zh           VARCHAR(500) NOT NULL,
    param_name_en           VARCHAR(500),
    tr069_path              VARCHAR(1000) NOT NULL,
    tr069_path_parts        TEXT[] GENERATED ALWAYS AS (string_to_array(tr069_path, '.')) STORED,
    value_type              VARCHAR(50) NOT NULL,
    value_constraint        JSONB,
    default_value           TEXT,
    js_regex                VARCHAR(500),
    is_writable             BOOLEAN NOT NULL DEFAULT false,
    is_listable             BOOLEAN NOT NULL DEFAULT false,
    is_modifiable           BOOLEAN NOT NULL DEFAULT false,
    is_addable              BOOLEAN NOT NULL DEFAULT false,
    is_removable            BOOLEAN NOT NULL DEFAULT false,
    is_leaf                 BOOLEAN NOT NULL DEFAULT true,
    is_dynamic              BOOLEAN NOT NULL DEFAULT false,
    display_order           INT NOT NULL DEFAULT 0,
    param_version           VARCHAR(50) NOT NULL REFERENCES mml_param_versions(version_code),
    software_version        VARCHAR(100),
    platform_support        VARCHAR(10)[],
    mobile_support          BOOLEAN NOT NULL DEFAULT true,
    broadband_support       BOOLEAN NOT NULL DEFAULT true,
    memo                    TEXT,
    explanation_zh          TEXT,
    explanation_en          TEXT,
    title_zh                VARCHAR(1000),
    title_en                VARCHAR(1000),
    require_second_confirm  BOOLEAN NOT NULL DEFAULT false,
    confirm_message_zh      TEXT,
    confirm_message_en      TEXT,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    created_by              VARCHAR(100),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT uq_param_version_path UNIQUE (param_version, tr069_path),
    CONSTRAINT chk_value_type CHECK (value_type IN ('string', 'enum', 'unsignedInt', 'unsignedIntList', 
                                                     'stringList', 'boolean', 'uniqueInt', 'int'))
);

-- 索引
CREATE INDEX idx_mml_params_version ON mml_params(param_version);
CREATE INDEX idx_mml_params_path ON mml_params(tr069_path);
CREATE INDEX idx_mml_params_path_parts ON mml_params USING GIN(tr069_path_parts);
CREATE INDEX idx_mml_params_code ON mml_params(param_code);
CREATE INDEX idx_mml_params_active ON mml_params(param_version, is_active) WHERE is_active = true;
CREATE INDEX idx_mml_params_deleted ON mml_params(deleted_at) WHERE deleted_at IS NOT NULL;

-- 自动更新 updated_at
DROP TRIGGER IF EXISTS trigger_mml_params_updated_at ON mml_params;
CREATE TRIGGER trigger_mml_params_updated_at
    BEFORE UPDATE ON mml_params
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 4. 分组-参数关联表(核心新增 - 显式外键关系)
-- ============================================================
CREATE TABLE mml_group_param_rel (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id                UUID NOT NULL REFERENCES mml_param_groups(id) ON DELETE CASCADE,
    param_id                UUID NOT NULL REFERENCES mml_params(id) ON DELETE CASCADE,
    sort_order              INT NOT NULL DEFAULT 0,
    matched_by              VARCHAR(20) NOT NULL DEFAULT 'manual',
    match_rule              TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              VARCHAR(100),
    CONSTRAINT uq_group_param UNIQUE (group_id, param_id)
);

-- 索引
CREATE INDEX idx_mml_group_param_group ON mml_group_param_rel(group_id);
CREATE INDEX idx_mml_group_param_param ON mml_group_param_rel(param_id);
CREATE INDEX idx_mml_group_param_sort ON mml_group_param_rel(group_id, sort_order);

-- ============================================================
-- 5. 辅助函数
-- ============================================================

-- 解析老 V_TYPE 字符串到 JSONB
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION parse_v_type(v_type_str TEXT)
RETURNS TABLE(value_type VARCHAR(50), value_constraint JSONB) AS $$
DECLARE
    result_type VARCHAR(50);
    result_constraint JSONB;
    parts TEXT[];
    main_part TEXT;
    range_part TEXT;
    enum_labels TEXT[];
    enum_values TEXT[];
    min_val TEXT;
    max_val TEXT;
    max_len INT;
BEGIN
    IF v_type_str IS NULL OR v_type_str = '' THEN
        RETURN QUERY SELECT 'string'::VARCHAR(50), '{"type": "string"}'::JSONB;
        RETURN;
    END IF;

    -- 解析 enum 类型: enum-{label1,label2}-{value1,value2}
    IF v_type_str LIKE 'enum-%' THEN
        parts := regexp_match(v_type_str, '^enum-\{([^}]*)\}-\{([^}]*)\}$');
        IF parts IS NOT NULL THEN
            enum_labels := string_to_array(parts[1], ',');
            enum_values := string_to_array(parts[2], ',');
            result_type := 'enum';
            result_constraint := jsonb_build_object(
                'type', 'enum',
                'labels', enum_labels,
                'values', enum_values
            );
            RETURN QUERY SELECT result_type, result_constraint;
            RETURN;
        END IF;
    END IF;

    -- 解析带范围的类型: type-[min:max]
    IF v_type_str LIKE '%-[%' THEN
        parts := regexp_match(v_type_str, '^([^-]+)-\[([^:]*):([^\]]*)\]$');
        IF parts IS NOT NULL THEN
            main_part := parts[1];
            min_val := parts[2];
            max_val := parts[3];

            result_type := main_part;
            result_constraint := jsonb_build_object('type', main_part);

            IF min_val != '' THEN
                result_constraint := result_constraint || jsonb_build_object('min', min_val::INT);
            END IF;
            IF max_val != '' THEN
                result_constraint := result_constraint || jsonb_build_object('max', max_val::INT);
            END IF;

            RETURN QUERY SELECT result_type, result_constraint;
            RETURN;
        END IF;
    END IF;

    -- 解析 string 带长度: string-[0:256]
    IF v_type_str LIKE 'string-%' THEN
        parts := regexp_match(v_type_str, '^string-\[([^:]*):([^\]]*)\]$');
        IF parts IS NOT NULL THEN
            min_val := parts[1];
            max_val := parts[2];

            result_type := 'string';
            result_constraint := jsonb_build_object('type', 'string');

            IF min_val != '' THEN
                result_constraint := result_constraint || jsonb_build_object('min_length', min_val::INT);
            END IF;
            IF max_val != '' THEN
                result_constraint := result_constraint || jsonb_build_object('max_length', max_val::INT);
            END IF;

            RETURN QUERY SELECT result_type, result_constraint;
            RETURN;
        END IF;
    END IF;

    -- 简单类型(无约束)
    result_type := v_type_str;
    result_constraint := jsonb_build_object('type', v_type_str);
    RETURN QUERY SELECT result_type, result_constraint;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
-- +goose StatementEnd

-- 反向构建 V_TYPE 字符串(用于兼容视图)
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION build_v_type_string(v_type VARCHAR(50), v_constraint JSONB)
RETURNS TEXT AS $$
DECLARE
    result TEXT;
    labels TEXT[];
    values TEXT[];
    min_val INT;
    max_val INT;
    min_len INT;
    max_len INT;
BEGIN
    IF v_constraint IS NULL THEN
        RETURN v_type;
    END IF;

    -- enum 类型
    IF v_type = 'enum' THEN
        labels := v_constraint->>'labels';
        values := v_constraint->>'values';
        IF labels IS NOT NULL AND values IS NOT NULL THEN
            result := 'enum-{' || array_to_string(labels, ',') || '}-{' || array_to_string(values, ',') || '}';
            RETURN result;
        END IF;
    END IF;

    -- 带范围的数值类型
    IF v_type IN ('unsignedInt', 'int', 'uniqueInt') THEN
        min_val := (v_constraint->>'min')::INT;
        max_val := (v_constraint->>'max')::INT;
        result := v_type || '-[';
        IF min_val IS NOT NULL THEN
            result := result || min_val;
        END IF;
        result := result || ':';
        IF max_val IS NOT NULL THEN
            result := result || max_val;
        END IF;
        result := result || ']';
        RETURN result;
    END IF;

    -- string 带长度
    IF v_type = 'string' THEN
        min_len := (v_constraint->>'min_length')::INT;
        max_len := (v_constraint->>'max_length')::INT;
        result := 'string-[';
        IF min_len IS NOT NULL THEN
            result := result || min_len;
        END IF;
        result := result || ':';
        IF max_len IS NOT NULL THEN
            result := result || max_len;
        END IF;
        result := result || ']';
        RETURN result;
    END IF;

    -- 默认返回类型
    RETURN v_type;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
-- +goose StatementEnd

-- +goose Down
DROP VIEW IF EXISTS v_params_legacy;
DROP VIEW IF EXISTS v_param_groups_legacy;
DROP FUNCTION IF EXISTS build_v_type_string;
DROP FUNCTION IF EXISTS parse_v_type;
DROP TABLE IF EXISTS mml_group_param_rel CASCADE;
DROP TABLE IF EXISTS mml_params CASCADE;
DROP TABLE IF EXISTS mml_param_groups CASCADE;
DROP TABLE IF EXISTS mml_param_versions CASCADE;

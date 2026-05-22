-- ============================================================
-- 000154_drop_mml_params.sql
-- 删除 mml_params 老业务参数库表 + 关联表 + STANDARD anchor 残留
--
-- 背景：v2.3 catalog 单源化（seed/000152 + seed/000155）后，MML Console 的
-- sub_field 元数据已通过 mml_command_sub_fields.standard_path_id FK 直接 JOIN
-- standard_params（脏迁移 000113 完成），不再依赖 mml_params。
--
-- 用户决策（2026-05-22）：方案 Y 单表统一
--   · 不扩 standard_params schema（不加 default_value/js_regex/explanation_i18n 等业务字段）
--   · 不做数据迁移（mml_params 4304 行直接抛弃 — 472 行带 default_value、
--     141 行带 js_regex、16 行带 explanation_zh，业务可接受丢失）
--   · 直接 DROP mml_params + mml_group_param_rel 表
--
-- 配套已下线（同批 commit）：
--   · internal/mml/param_handler.go / param_service.go / param_pg_repository.go（4 个 /mml/param-versions/* 端点）
--   · internal/mml/admin_repository.go 的 AdminParamRepository / PgAdminParamRepository
--   · internal/mml/admin_service.go 的 CreateParam/UpdateParam/DeleteParam/ListParams/ListParamReferences
--   · internal/mml/admin_handler.go 的 /admin/params/* 4 端点 + /admin/params/:id/references
--   · internal/mml/xml_import_service.go / xml_import_helpers.go（XML 批量导入 mml_params）
--   · internal/config/parammodel/mmlstandardloader/loader.go 的 upsertParams（已 stub no-op）
--   · cmd/omcctl/mml.go 的 import-standard-xml 子命令
--   · 前端 ParamsTab.tsx / ParamReferencesDrawer.tsx / mmlAdminApi.ts 参数 CRUD section
--
-- DROP 顺序：
--   1. mml_group_param_rel（FK → mml_params(id) 与 mml_command_groups(id)）
--   2. mml_params
--   3. 顺手清掉 mml_param_versions 的 STANDARD anchor（仅 mml_params 使用，
--      mml_command_groups 走 cmcc-td-lte-v2.3）
--
-- 不可逆：mml_params 4304 行 + mml_group_param_rel 全量删除。Down 仅重建空表
--        schema 以便回滚到旧代码（数据无法恢复）。
-- ============================================================

-- +goose Up

-- 1. 关联表：mml_group_param_rel（FK to mml_params(id) ON DELETE CASCADE 推测）
DROP TABLE IF EXISTS mml_group_param_rel;

-- 2. 主表：mml_params（4304 行，全量删除）
DROP TABLE IF EXISTS mml_params;

-- 3. mml_param_versions 老业务参数库 anchor 行（STANDARD + 22 个产品版本号）
--   全部仅 mml_params 使用过；mml_command_groups 走 cmcc-td-lte-v2.3，
--   standard_commands 空表；删除后只剩 cmcc-td-lte-v2.3 单一活跃版本。
DELETE FROM mml_param_versions
 WHERE version_code <> 'cmcc-td-lte-v2.3';


-- +goose Down

-- Down 仅恢复空 schema 用于代码回滚兼容；4304 + N 行业务数据无法恢复。
-- 完整 schema 应从 migrations/000022 / 000023 / 000026 / 000035 / 000090 等
-- 老迁移合成。这里 minimal 重建供 down 测试链路顺利完成。

CREATE TABLE IF NOT EXISTS mml_params (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    param_code VARCHAR(200) NOT NULL,
    param_name_zh VARCHAR(500) NOT NULL,
    param_name_en VARCHAR(500),
    tr069_path VARCHAR(1000) NOT NULL,
    value_type VARCHAR(50) NOT NULL,
    value_constraint jsonb,
    default_value text,
    js_regex VARCHAR(500),
    is_leaf BOOLEAN NOT NULL DEFAULT true,
    display_order INT NOT NULL DEFAULT 0,
    param_version VARCHAR(50) NOT NULL,
    explanation_zh text,
    explanation_en text,
    title_zh VARCHAR(1000),
    title_en VARCHAR(1000),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    name_i18n jsonb NOT NULL DEFAULT '{}'::jsonb,
    explanation_i18n jsonb NOT NULL DEFAULT '{}'::jsonb,
    access_type VARCHAR(20) NOT NULL DEFAULT 'READ_ONLY',
    is_object BOOLEAN NOT NULL DEFAULT false,
    supports_add BOOLEAN NOT NULL DEFAULT false,
    supports_delete BOOLEAN NOT NULL DEFAULT false,
    change_applies VARCHAR(20) NOT NULL DEFAULT 'Immediate',
    constraint_text_i18n jsonb NOT NULL DEFAULT '{}'::jsonb,
    catalog_protected BOOLEAN NOT NULL DEFAULT false,
    source VARCHAR(20) NOT NULL DEFAULT 'admin'
);

CREATE TABLE IF NOT EXISTS mml_group_param_rel (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id uuid NOT NULL,
    param_id uuid NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    matched_by VARCHAR(20) NOT NULL DEFAULT 'manual',
    match_rule text,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO mml_param_versions (id, version_code, version_name, description, source, is_active, is_deprecated)
VALUES (gen_random_uuid(), 'STANDARD', 'TR-069 Standard Model',
        '由 standard-model.xml 派生，loader 自动维护', 'standard', true, false)
ON CONFLICT (version_code) DO NOTHING;

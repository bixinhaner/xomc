-- +goose Up
-- ============================================================
-- 000061_dict_detail_parent_id.sql
-- 字典明细树形增强（PRD docs/prd/system/data-dictionary.md v0.2 §10）
-- ============================================================

-- 新增 parent_id (FK self) + level
ALTER TABLE sys_dictionary_details
    ADD COLUMN parent_id BIGINT NULL REFERENCES sys_dictionary_details(id) ON DELETE CASCADE,
    ADD COLUMN level     INT NOT NULL DEFAULT 0;

-- parent_id 查询索引
CREATE INDEX idx_dict_detail_parent
    ON sys_dictionary_details(parent_id)
    WHERE parent_id IS NOT NULL;

-- 一次性清理：历史 seed 重复行（seed/000001_seed_data.sql 双载导致，
-- gender/int/...等字典每条 detail 都有 (sys_dictionary_id, value) 重复 2 份）。
-- 加新唯一索引前必须先把重复行软删，否则 CREATE UNIQUE INDEX 失败。
-- 策略：每组 (sys_dictionary_id, value) 仅保留 min(id)，其它软删。
UPDATE sys_dictionary_details
SET deleted_at = NOW(), updated_at = NOW()
WHERE deleted_at IS NULL
  AND id NOT IN (
    SELECT MIN(id) FROM sys_dictionary_details
    WHERE deleted_at IS NULL
    GROUP BY sys_dictionary_id, value
  );

-- 同字典 / 顶层项 value 唯一（Q4 决议）
CREATE UNIQUE INDEX uniq_dict_detail_top_value
    ON sys_dictionary_details(sys_dictionary_id, value)
    WHERE parent_id IS NULL AND deleted_at IS NULL;

-- 同字典 / 同父 子项 value 唯一；跨分支可重名（Q4 决议）
CREATE UNIQUE INDEX uniq_dict_detail_child_value
    ON sys_dictionary_details(sys_dictionary_id, parent_id, value)
    WHERE parent_id IS NOT NULL AND deleted_at IS NULL;

COMMENT ON COLUMN sys_dictionary_details.parent_id IS '父明细 ID；NULL = 顶层；非 NULL = 子项（自引用 FK，删父级联子）';
COMMENT ON COLUMN sys_dictionary_details.level     IS '层级冗余：0=顶层 / 1=一级子 / 2=二级子，最大深度 3 层（应用层校验）';

-- +goose Down
DROP INDEX IF EXISTS uniq_dict_detail_child_value;
DROP INDEX IF EXISTS uniq_dict_detail_top_value;
DROP INDEX IF EXISTS idx_dict_detail_parent;
ALTER TABLE sys_dictionary_details
    DROP COLUMN IF EXISTS level,
    DROP COLUMN IF EXISTS parent_id;

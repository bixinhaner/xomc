-- +goose Up
-- ufte_task_types 加 sort_order 列控制子 tab 显示顺序。
--
-- 背景：原 OrderBy("built_in DESC", "category_label ASC", "display_name ASC") 让
-- 中文 displayName 沉底 —— "4G 基站软件升级"(基=U+57FA) 排到 "4G FPGA 升级"(F)
-- 和 "4G Patch 增量升级"(P) 之后，用户期待主升级在首位。
--
-- 改造：sort_order 数字小的排前。内置模板按业务重要性显式赋值（10/20/30...，
-- 留空隙便于以后插队）；自定义模板默认 0，会排在内置之前（如需调整再在「模板
-- 配置」UI 拖拽）。
--
-- 后端 internal/ufte/repository.go 的 OrderBy 同步改成
--   ORDER BY sort_order ASC, built_in DESC, display_name ASC

ALTER TABLE ufte_task_types ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 100;

-- 内置模板按业务序赋值（数字小靠前）：
-- 升级类 10-19，回退 20，日志 30-39，配置备份 40-49，配置恢复 50
UPDATE ufte_task_types SET sort_order = 10 WHERE type_code IN ('ENB_IMG_UPGRADE', 'GNB_IMG_UPGRADE');
UPDATE ufte_task_types SET sort_order = 15 WHERE type_code = 'ENB_PATCH_UPGRADE';
UPDATE ufte_task_types SET sort_order = 18 WHERE type_code = 'ENB_FPGA_UPGRADE';
UPDATE ufte_task_types SET sort_order = 20 WHERE type_code = 'VERSION_ROLLBACK';
UPDATE ufte_task_types SET sort_order = 30 WHERE type_code = 'RUNTIME_LOG_COLLECT';
UPDATE ufte_task_types SET sort_order = 35 WHERE type_code = 'FAULT_LOG_COLLECT';
UPDATE ufte_task_types SET sort_order = 40 WHERE type_code = 'CONFIG_BACKUP_XML';
UPDATE ufte_task_types SET sort_order = 45 WHERE type_code = 'CONFIG_BACKUP_NV';
UPDATE ufte_task_types SET sort_order = 50 WHERE type_code = 'CONFIG_RESTORE';

CREATE INDEX IF NOT EXISTS idx_ufte_task_types_sort ON ufte_task_types (category, sort_order, display_name);


-- +goose Down
DROP INDEX IF EXISTS idx_ufte_task_types_sort;
ALTER TABLE ufte_task_types DROP COLUMN IF EXISTS sort_order;

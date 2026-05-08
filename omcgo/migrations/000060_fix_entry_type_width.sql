-- T-0098-P1-06 followup — 修复 entry_type 列宽
--
-- 背景：P1-03（迁移 000058）将 param_mappings / discovered_param_mappings / standard_params
-- 三表的 entry_type 设为 VARCHAR(8)，与 CHECK 约束允许的 'parameter'（9 chars）冲突。
-- P1-06 Loader 跑实际数据时触发 SQLSTATE 22001 "value too long for type character varying(8)"。
--
-- 修复：扩列到 VARCHAR(16)，CHECK 约束不变（IN ('object','parameter')）。
-- VARCHAR(16) 给后续可能的取值留余量（如未来可能引入的 'subscriber'、'event' 等），
-- 不影响存储成本（PG 9.x+ varchar 仅存实际字节）。

-- +goose Up
ALTER TABLE param_mappings              ALTER COLUMN entry_type TYPE VARCHAR(16);
ALTER TABLE discovered_param_mappings   ALTER COLUMN entry_type TYPE VARCHAR(16);
ALTER TABLE standard_params             ALTER COLUMN entry_type TYPE VARCHAR(16);

-- +goose Down
-- 缩列前必须清空：'parameter' 9 chars 不可逆装入 VARCHAR(8)。
-- 此 down 数据销毁可接受 — 数据由 Loader 从 XML 重建，且 058 down 全表 DROP；本 60 down
-- 仅是反向兼容路径；真实回滚一般是 60 → 59（连 058 整层 DROP）。
TRUNCATE standard_params, discovered_param_mappings, param_mappings;
ALTER TABLE standard_params             ALTER COLUMN entry_type TYPE VARCHAR(8);
ALTER TABLE discovered_param_mappings   ALTER COLUMN entry_type TYPE VARCHAR(8);
ALTER TABLE param_mappings              ALTER COLUMN entry_type TYPE VARCHAR(8);

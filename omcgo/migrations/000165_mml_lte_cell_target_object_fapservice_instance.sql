-- 000165_mml_lte_cell_target_object_fapservice_instance.sql
--
-- 修复 mml_commands.target_object 缺 FAPService 实例号段（LTE_CELL 命令）。
--
-- 背景（与 000163 CARRIER 完全同模式）：
--   1. 原 target_object 形如
--      `Device.Services.FAPService.CellConfig.LTE.RAN.NeighborList.LTECell.`
--      —— 缺 FAPService 实例号（`.1.` / `.2.`），CPE 直接拒（fault 9005
--      "Invalid Object Name"）
--   2. BLQ param_mappings 表中 LTECell 路径形如
--      `Device.Services.FAPService.2.CellConfig.LTE.RAN.NeighborList.LTECell.{i}`
--      （固定 `.2.` 因 BLQ 设备的 BaiBLQ 数据模型在单 slot 下走 FAPService.2）
--   3. fanout.translateObjectName (commit 9d89e52a) 现在已对 ADD/RMV 的
--      object_name 走 ParamRegistry → Translator.ToPrivate；只有 target_object
--      含可命中的 standardPath 才能翻译到 privatePath
--
-- 本迁移：把 LTE_CELL 的 target_object 改为含 `.2.` 的形态，与 BLQ mapping
-- 对齐。这样在 BaiBLQ 设备上：
--   1. console_executor.go::buildStatementCommandEntry (ADD/RMV 分支) 不带占位符
--      → substituteInstanceSelectors no-op → 输出
--      `...FAPService.2.CellConfig.LTE.RAN.NeighborList.LTECell.`
--   2. fanout.translateObjectName 查表命中 (identity mapping) → 下发 SOAP
--      用同名 path
--   3. CPE 接受 → AddObject/DeleteObject 成功
--
-- 与 000163 区别：仅命令字典行不同（CARRIER vs LTE_CELL），其余逻辑完全一致。
--
-- 范围说明：QA v2 §3 类 A 共 4 例 9005 失败（A1/R1 即 A1_MEASURE_CTRL；
-- L1a/L1b 即 LST DEVICE_INFO 字段缺失）：其中 A1_MEASURE_CTRL / A2-B2 等
-- 共 54 个 ADD/RMV 命令的 standardPath 在 BLQ param_mappings 完全无映射，
-- 不在本迁移范围；待后续 X3（discovered_access）+ 字典治理 PRD 处理。

-- +goose Up

UPDATE mml_commands
   SET target_object = 'Device.Services.FAPService.2.CellConfig.LTE.RAN.NeighborList.LTECell.',
       updated_at    = NOW()
 WHERE logical_code = 'LTE_CELL'
   AND operation_type IN ('ADD', 'RMV')
   AND target_object = 'Device.Services.FAPService.CellConfig.LTE.RAN.NeighborList.LTECell.';

-- +goose Down

UPDATE mml_commands
   SET target_object = 'Device.Services.FAPService.CellConfig.LTE.RAN.NeighborList.LTECell.',
       updated_at    = NOW()
 WHERE logical_code = 'LTE_CELL'
   AND operation_type IN ('ADD', 'RMV')
   AND target_object = 'Device.Services.FAPService.2.CellConfig.LTE.RAN.NeighborList.LTECell.';

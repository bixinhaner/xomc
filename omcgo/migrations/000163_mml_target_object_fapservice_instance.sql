-- 000163_mml_target_object_fapservice_instance.sql
--
-- 修复 mml_commands.target_object 缺 FAPService 实例号段。
--
-- 背景：
--   1. 原 target_object 形如 `Device.Services.FAPService.CellConfig.LTE...Carrier.`
--      —— 缺 FAPService 实例号（`.1.` / `.2.`），CPE 直接拒绝（fault 9005
--      "Invalid Object Name"）。
--   2. BLQ param_mappings 使用 `Device.Services.FAPService.2.CellConfig...Carrier.{i}`
--      形态（固定 .2. 因 BLQ 设备的 BaiBLQ 数据模型在单 slot 下走 FAPService.2）。
--   3. fanout.translateObjectName (v1.2 commit) 现在已对 ADD/RMV 的 object_name 走
--      ParamRegistry → Translator.ToPrivate；只有 target_object 含可命中的 standardPath
--      才能翻译到 privatePath。
--
-- 本迁移：把 CARRIER 命令的 target_object 改为含 `.2.` 的形态，与 BLQ mapping 表
-- 对齐。这样在 BaiBLQ 设备上：
--   1. console_executor.go::buildStatementCommandEntry (ADD/RMV 分支) 不带占位符 →
--      substituteInstanceSelectors no-op → 输出 `...FAPService.2.CellConfig...Carrier.`
--   2. fanout.translateObjectName 查表命中 (identity mapping) → 下发 SOAP 用同名 path
--   3. CPE 接受 → AddObject/DeleteObject 成功
--
-- 注意：
--   - A1_MEASURE_CTRL 命令 mappings 表完全没有该路径（数据缺失，超 P0 scope），不修
--   - 仅改 CARRIER 是 v1.2 测试报告 §6.P0 的最小验证集；其余命令字典审计走单独 PRD
--   - 多 FAPService 实例（slot=1/2 都用）的设备未来需要把 `.2.` 改回 `.{i}.`
--     并在前端引导用户选择实例号 —— 这是数据模型的 product-level 决策

-- +goose Up

-- CARRIER ADD/RMV 的 target_object 补 .2. 实例号段
UPDATE mml_commands
   SET target_object = 'Device.Services.FAPService.2.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.',
       updated_at    = NOW()
 WHERE logical_code = 'CARRIER'
   AND operation_type IN ('ADD', 'RMV')
   AND target_object = 'Device.Services.FAPService.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.';

-- +goose Down

-- 回滚：恢复原 standardPath 形态（缺 FAPService 实例号）。
-- 注：回滚后 ADD/RMV 仍然必失败（CPE 9005），与本迁移上线前行为一致。
UPDATE mml_commands
   SET target_object = 'Device.Services.FAPService.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.',
       updated_at    = NOW()
 WHERE logical_code = 'CARRIER'
   AND operation_type IN ('ADD', 'RMV')
   AND target_object = 'Device.Services.FAPService.2.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.';

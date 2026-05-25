-- +goose Up
-- ============================================================
-- 000182_device_info_add_lac.sql
-- 给 device_info 表补 LAC 列（GSM 位置区码），与已存在的 tac 列对齐。
--
-- 背景：device_groups 已有 LAC / TAC 匹配模式定义（migrations/000023 + matcher.go），
-- TAC 列也早就在 device_info 表里。但 LAC 列从未建过 —— pg_device_lister.go:19
-- 历史注释还停在"三表均无 lac/tac 列"的旧认知。
--
-- 现状（截至 2026-05-25）：
--   - matcher 的 MatchingModeLAC case 100% 走 `if req.LAC == nil { return false }`
--   - ListAllForRuleEval SQL 不读 lac/tac → DeviceForMatch.LAC/TAC 永远 nil
--   - 用户在 UI 选"匹配方式 = LAC"会得到一个永远为空的分组
--
-- 本迁移只补字段；写入逻辑（ACS Inform 解析 Device.DeviceInfo.BTS.CurrentLac）
-- 和异步重匹配事件（device.attributes.changed）在 Go 侧补齐。
-- ============================================================

ALTER TABLE device_info
    ADD COLUMN IF NOT EXISTS lac VARCHAR(16);

COMMENT ON COLUMN device_info.lac IS
    'GSM 位置区码（Location Area Code），由 ACS Inform 解析 Device.DeviceInfo.BTS.CurrentLac 写入。可空 —— LTE-only 设备无此值。';

COMMENT ON COLUMN device_info.tac IS
    'LTE 跟踪区码（Tracking Area Code），由 ACS Inform 解析 Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC 写入。可空 —— GSM-only 设备无此值。';

-- +goose Down
ALTER TABLE device_info DROP COLUMN IF EXISTS lac;

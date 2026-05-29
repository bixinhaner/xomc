-- T-0179 P1: alarm_definitions 加 loaded_from 列,记录 Loader 来源 XML 文件名。
--
-- 背景:
--   - T-0179 alarm-library 页面 drill-down 重设计需要按 ne_type 聚合行(数量、来源文件)
--   - 现 alarm_definitions 只有 ne_type 列,无法区分同 ne_type 下来自不同 XML 文件的告警
--   - 聚合 API `GET /api/v1/alarm-definitions/ne-types` 需要按 ne_type + loaded_from 双键统计
--
-- 行为:
--   - 新增 loaded_from VARCHAR(256) 可空列,Loader.batchUpsertAlarms 入库时写入 filepath.Base(path)
--   - 历史数据 loaded_from = NULL,通过聚合 API 的 COALESCE 逻辑显示为"未知 XML 来源"
--     (无需回填,下次 Loader.Reload 会自动写入)
--
-- 幂等:ADD COLUMN IF NOT EXISTS
--
-- Down:DROP COLUMN IF EXISTS(回滚到 T-0179 之前行为)

-- +goose Up

ALTER TABLE alarm_definitions ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256);

COMMENT ON COLUMN alarm_definitions.loaded_from IS 'T-0179 Loader 来源 XML 文件名(如 "ENB.xml");NULL 表示历史数据未回填,Reload 后会自动写入';

-- +goose Down

ALTER TABLE alarm_definitions DROP COLUMN IF EXISTS loaded_from;

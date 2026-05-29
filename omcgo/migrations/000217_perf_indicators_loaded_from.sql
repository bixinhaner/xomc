-- T-0180 P1.2: perf_indicators_{enb,gsm,gnb} 加 loaded_from 列。
--
-- 背景:
--   - T-0180 KPI 指标库页面重设计需要按 (tech, loaded_from) 聚合一级表
--     (M 内置 / N 自定义 列),以及二级表显"来源"Tag
--   - Loader 入库时写 filepath.Rel(XMLBaseDir, absPath) | ToSlash 形如
--     "indicator-library/enb/ALL.xml" 或 "indicator-library-custom/enb/MY.xml"
--   - 与 T-0178 param_models.loaded_from / T-0179 alarm_definitions.loaded_from 字段口径一致
--
-- 行为:
--   - 新增 loaded_from VARCHAR(256) 可空列(三张表) + 各加一个索引(支持 "管理 XML 文件" Modal 按 loaded_from GROUP BY)
--   - 历史数据 loaded_from = NULL → ClassifySource 返 SourceUnknown → IsDeletable=false(保守拒删)
--     无需回填,下次 Loader.Reload 自动写入正确前缀
--
-- 幂等:ADD COLUMN IF NOT EXISTS + CREATE INDEX IF NOT EXISTS
--
-- Down:DROP INDEX + DROP COLUMN(回滚到 T-0180 之前行为)

-- +goose Up

ALTER TABLE perf_indicators_enb ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256);
ALTER TABLE perf_indicators_gsm ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256);
ALTER TABLE perf_indicators_gnb ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256);

CREATE INDEX IF NOT EXISTS idx_perf_indicators_enb_loaded_from ON perf_indicators_enb(loaded_from);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_gsm_loaded_from ON perf_indicators_gsm(loaded_from);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_gnb_loaded_from ON perf_indicators_gnb(loaded_from);

COMMENT ON COLUMN perf_indicators_enb.loaded_from IS 'T-0180 Loader 来源 XML 相对路径(含前缀,如 "indicator-library/enb/ALL.xml" 或 "indicator-library-custom/enb/MY.xml");NULL 表示历史数据未回填,Reload 后会自动写入';
COMMENT ON COLUMN perf_indicators_gsm.loaded_from IS 'T-0180 Loader 来源 XML 相对路径(含前缀);NULL 表示历史数据未回填,Reload 后会自动写入';
COMMENT ON COLUMN perf_indicators_gnb.loaded_from IS 'T-0180 Loader 来源 XML 相对路径(含前缀);NULL 表示历史数据未回填,Reload 后会自动写入';

-- +goose Down

DROP INDEX IF EXISTS idx_perf_indicators_enb_loaded_from;
DROP INDEX IF EXISTS idx_perf_indicators_gsm_loaded_from;
DROP INDEX IF EXISTS idx_perf_indicators_gnb_loaded_from;

ALTER TABLE perf_indicators_enb DROP COLUMN IF EXISTS loaded_from;
ALTER TABLE perf_indicators_gsm DROP COLUMN IF EXISTS loaded_from;
ALTER TABLE perf_indicators_gnb DROP COLUMN IF EXISTS loaded_from;

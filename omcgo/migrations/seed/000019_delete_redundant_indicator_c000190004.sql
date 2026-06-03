-- +goose Up
-- PM 原始计数指标落库编号化 P1:删冗余编号 C000190004(种子阶段)。
-- 标准名 L.UL.Interference.Avg 错挂两条编号(C000090117 正主 / C000190004 冗余),
-- 从源头消歧义:保留 C000090117(MAC 组成套系列),删 C000190004。
--
-- 为何放在 seed 阶段而非 schema 阶段:本项目 migrate 执行模型是「先全部 schema(migrations/),
-- 再全部 seed(migrations/seed/,depends_on schema 完成)」。冗余行由 seed/000001_init_seed.sql
-- 无条件 INSERT 插入(enabled_pm_indicators_enb 1 行 / perf_indicators_enb 1 行 /
-- rela_platform_indicator_formula_enb 3 行,共 5 处)。已 applied 的 seed/000001 按 §5.5.11
-- 铁律 4 不回头改,故在其之后(本文件号 000019 > 000001)显式 DELETE 把插回的行删净,
-- 保证新库重建路径上 C000190004 最终在 DB 消失(放 schema 阶段会被随后 seed 插回,无效)。
--
-- 运行 XML(data/indicator-library/enb/{BLQ,MLN,MLQ}.xml)已删该 <indicator> 行,
-- 但装载器重载不会自动清掉这些残留:
--   - 指标行走 UPSERT(INSERT ON CONFLICT),不删 XML 里已消失的孤儿行 → 残留;
--   - 启用桶走"插启用 + 删禁用",XML 删掉后该 id 既不在启用集也不在禁用集 → 残留;
--   - 仅平台公式表每次 reload 走 TRUNCATE 全量重写会自清(但此处一并删更确定)。
-- 故显式 DELETE 三张表。仅 enb(C 编码是 LTE 指标,gsm/gnb 不含)。
DELETE FROM rela_platform_indicator_formula_enb WHERE indicator_id = 'C000190004';
DELETE FROM enabled_pm_indicators_enb WHERE indicator_id = 'C000190004';
DELETE FROM perf_indicators_enb WHERE id = 'C000190004';

-- +goose Down
-- 数据删除不可逆:被删的 C000190004 指标行 / 平台公式 / 启用桶无法恢复。
-- 该编号本就是冗余错挂,正主 C000090117 仍在,无需也无法回滚,Down 段留空注释。

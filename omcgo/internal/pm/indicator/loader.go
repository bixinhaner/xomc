package indicator

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/dictloader"
	"github.com/omcgo/omcgo/internal/core/reliability"
)

// LoaderName 是 dictloader.Registry 中的注册名（T-0098 P1-06）。
const LoaderName = "indicator"

// defaultGroupID 是 P1-06 baseline 用的占位 indicator_group_*.id。
// 真正的多级分组由 Phase 2/3 handler/UI 维护；本 baseline 不解析 XML 中的功能集树（XML 当前未携带）。
const defaultGroupID = "default"

// Loader 实现 dictloader.Loader（T-0098 P1-06）。
//
// 加载语义（设计 §2.6 + 实施计划 §1.2 注："{enabled} / 多文件 OR 合并 / operator_code 桶刷新" 在 P2-09 增强）：
//  1. 扫描 enb/*.xml + GSM.xml + GNB.xml
//  2. （单位已下线）指标单位改由数据字典 type='indicator_unit' 管理，初始化走
//     migrations/seed/000007；Loader 不再写 indicator_unit 表
//  3. 插入 indicator_group_{enb,gsm,gnb} 占位 group（id="default"，satisfy NOT NULL FK）
//  4. UPSERT perf_indicators_{enb,gsm,gnb}（按 id 唯一）
//  5. 重写 rela_platform_indicator_formula_{enb,gsm,gnb} — 每 (platform_name, indicator_id) 一行
//
// 不在 P1-06 范围（P2-09 接力）：
//   - enabled 属性 OR 合并语义（多文件取 OR）→ enabled_pm_indicators_* 写入
//   - operator_code='default' 桶刷新（仅刷新 default 行，不动其他 operator）
type Loader struct {
	pool   *pgxpool.Pool
	cfg    appconfig.IndicatorLoaderConfig
	base   string
	logger *zap.Logger

	// cacheBumper 在每次成功加载后调用，使依赖指标库的下游缓存（KPI 路由缓存）失效。
	// 解析侧采集白名单 / KPI 公式都从 KPI 路由派生（pm/kpi/router），而路由按 product 缓存
	// 于 Redis L2（24h TTL，version-bump 失效）。若指标库新增/改动指标后不 bump 路由 cache
	// version，已缓存的旧路由会继续生效——新增计数器（如 ISSUE-389 的「统计时长」）的 report_key
	// 不在旧路由白名单里 → 落库被当孤儿丢弃 → 派生 KPI 永远无分母。
	// 为避免 indicator → router 的反向 import 环（router 已 import indicator），此处用裸回调，
	// 由 provider/worker 接线层注入「调 RedisCache.BumpVersion」的闭包。nil 时跳过（无 Redis 部署）。
	cacheBumper func(context.Context) error
}

func NewLoader(pool *pgxpool.Pool, cfg appconfig.IndicatorLoaderConfig, baseDir string, logger *zap.Logger) *Loader {
	if cfg.BaseDirectory == "" {
		cfg.BaseDirectory = "indicator-library"
	}
	if cfg.EnbSubdir == "" {
		cfg.EnbSubdir = "enb"
	}
	if cfg.GsmFile == "" {
		cfg.GsmFile = "GSM.xml"
	}
	if cfg.GnbFile == "" {
		cfg.GnbFile = "GNB.xml"
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Loader{pool: pool, cfg: cfg, base: baseDir, logger: logger.Named(LoaderName)}
}

// WithCacheBumper 注入加载成功后的下游缓存失效回调（典型实现：bump KPI 路由 cache_version）。
// 返回 *Loader 便于链式调用。bumper 为 nil 时为 no-op（不改变现有行为）。
func (l *Loader) WithCacheBumper(bumper func(context.Context) error) *Loader {
	l.cacheBumper = bumper
	return l
}

func (l *Loader) Name() string      { return LoaderName }
func (l *Loader) Directory() string { return l.cfg.BaseDirectory }

func (l *Loader) LoadOnce(ctx context.Context) (dictloader.Report, error) { return l.run(ctx) }
func (l *Loader) Reload(ctx context.Context) (dictloader.Report, error)   { return l.run(ctx) }

// xmlIndicatorModel 解析 indicator 单文件 XML（设计 §2.6）。
//
// LoadedFrom 是 T-0180 P1.2 在 Loader 内存中追加的来源标记(xml:"-" 不参与 XML 解析),
// 形如 "indicator-library/enb/ALL.xml"(单目录 + sidecar:builtin/custom 同住,custom 旁有 .custom 标记),
// 由 resolveENBSources / resolveSingleTechSources 在文件扫描阶段写入,
// 由 flattenDocsByDeviceType 透传到 indicatorRecord 最终落 perf_indicators_*.loaded_from 列。
type xmlIndicatorModel struct {
	XMLName        xml.Name       `xml:"indicatorModel"`
	Platform       string         `xml:"platform,attr"`
	IndicatorCount int            `xml:"indicatorCount,attr"`
	Indicators     []xmlIndicator `xml:"indicators>indicator"`
	LoadedFrom     string         `xml:"-"`
}

type xmlIndicator struct {
	ID             string `xml:"id,attr"`
	EnName         string `xml:"enName,attr"`
	ReportKey      string `xml:"reportKey,attr"`
	CnName         string `xml:"cnName,attr"`
	IsBuildIn      string `xml:"isBuildIn,attr"` // "1" / "0"
	IsCounter      string `xml:"isCounter,attr"` // "1" / "0"
	DataType       string `xml:"dataType,attr"`
	UnitID         string `xml:"unitId,attr"`
	StatisType     string `xml:"statisType,attr"`
	Arithmetic     string `xml:"arithmetic,attr"`
	IndicatorLevel string `xml:"indicatorLevel,attr"` // ENB/GSM only
	Formula        string `xml:"formula,attr"`
	Enabled        string `xml:"enabled,attr"` // 缺省视 "true"（P2-09 接力 OR 合并）
	GroupID        string `xml:"groupId,attr"` // 功能集 group_id；缺省回退 default 占位组
}

func (l *Loader) run(ctx context.Context) (dictloader.Report, error) {
	rep := dictloader.NewReport(LoaderName)
	defer rep.Finish()

	// 三库 XML 导入重构(2026-06-04 单目录):三制式单目录树扫描,sidecar 判来源。
	// ENB 多文件:indicator-library/enb/*.xml(builtin + custom 同住,sidecar 已跳过)
	enbSources, err := resolveENBSources(
		l.base,
		filepath.Join(l.cfg.BaseDirectory, l.cfg.EnbSubdir),
	)
	if err != nil {
		rep.AddError("enb", "resolve", err)
		return rep, fmt.Errorf("resolve enb sources: %w", err)
	}
	enbDocs := l.parseDocs(enbSources, &rep, "" /*platformFallback inferred per-file*/)

	// GSM:indicator-library/ 根级文件(出厂 GSM.xml 按文件名、自定义上传按 XML
	// deviceType 属性分类)。目录调整(2026-06-05):取消 gsm/ 子目录。
	gsmSources, err := resolveRootTechSources(l.base, l.cfg.BaseDirectory, "gsm")
	if err != nil {
		rep.AddError("gsm", "resolve", err)
		return rep, fmt.Errorf("resolve gsm sources: %w", err)
	}
	gsmDocs := l.parseDocs(gsmSources, &rep, "BSC")

	// GNB:同 GSM(根级,deviceType=GNB)
	gnbSources, err := resolveRootTechSources(l.base, l.cfg.BaseDirectory, "gnb")
	if err != nil {
		rep.AddError("gnb", "resolve", err)
		return rep, fmt.Errorf("resolve gnb sources: %w", err)
	}
	gnbDocs := l.parseDocs(gnbSources, &rep, "BaiBNQ")

	// 3) 单事务写入 4 类表
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return rep, fmt.Errorf("begin tx: %w", err)
	}
	defer reliability.RollbackTx(ctx, tx, l.logger, "indicator.Loader.run")

	// 3a) units 已下线:指标单位改由数据字典(sys_dictionaries type='indicator_unit')
	//     统一管理,初始化走 migrations/seed/000007;Loader 不再写 indicator_unit 表。

	// 3b) groups — 三个 device type 各一个占位
	if err := ensureDefaultGroups(ctx, tx); err != nil {
		return rep, fmt.Errorf("ensure default groups: %w", err)
	}

	// 3c) ENB indicators + formulas — 多文件 dedupe 取 first-seen;formula 每 platform 一行
	// T-0180 P1.2: flattenDocsByDeviceType 现返回 indicatorRecord(含 LoadedFrom),供 flushIndicators 写 loaded_from 列
	enbRecords, enbFormulas := flattenDocsByDeviceType(enbDocs, false /*hasIndicatorLevel=true for ENB but not GNB*/)
	gsmRecords, gsmFormulas := flattenDocsByDeviceType(gsmDocs, false)
	gnbRecords, gnbFormulas := flattenDocsByDeviceType(gnbDocs, true)

	if n, err := upsertIndicators(ctx, tx, "perf_indicators_enb", enbRecords, false); err != nil {
		return rep, err
	} else {
		rep.RowsAffected += n
	}
	if n, err := upsertIndicators(ctx, tx, "perf_indicators_gsm", gsmRecords, false); err != nil {
		return rep, err
	} else {
		rep.RowsAffected += n
	}
	if n, err := upsertIndicators(ctx, tx, "perf_indicators_gnb", gnbRecords, true); err != nil {
		return rep, err
	} else {
		rep.RowsAffected += n
	}

	if n, err := rewriteFormulas(ctx, tx, "rela_platform_indicator_formula_enb", enbFormulas); err != nil {
		return rep, err
	} else {
		rep.RowsAffected += n
	}
	if n, err := rewriteFormulas(ctx, tx, "rela_platform_indicator_formula_gsm", gsmFormulas); err != nil {
		return rep, err
	} else {
		rep.RowsAffected += n
	}
	if n, err := rewriteFormulas(ctx, tx, "rela_platform_indicator_formula_gnb", gnbFormulas); err != nil {
		return rep, err
	} else {
		rep.RowsAffected += n
	}

	// T-0098 P2-09：enabled 属性 OR 合并 + operator_code='default' 桶刷新（设计 §2.6 + 实施计划 §1.2）
	enbEnabled := aggregateEnabledOR(enbDocs)
	gsmEnabled := aggregateEnabledOR(gsmDocs)
	gnbEnabled := aggregateEnabledOR(gnbDocs)
	if n, err := refreshDefaultEnabledBucket(ctx, tx, "enabled_pm_indicators_enb", enbEnabled); err != nil {
		return rep, fmt.Errorf("refresh enabled_pm_indicators_enb: %w", err)
	} else {
		rep.RowsAffected += n
	}
	if n, err := refreshDefaultEnabledBucket(ctx, tx, "enabled_pm_indicators_gsm", gsmEnabled); err != nil {
		return rep, fmt.Errorf("refresh enabled_pm_indicators_gsm: %w", err)
	} else {
		rep.RowsAffected += n
	}
	if n, err := refreshDefaultEnabledBucket(ctx, tx, "enabled_pm_indicators_gnb", gnbEnabled); err != nil {
		return rep, fmt.Errorf("refresh enabled_pm_indicators_gnb: %w", err)
	} else {
		rep.RowsAffected += n
	}

	if err := tx.Commit(ctx); err != nil {
		return rep, fmt.Errorf("commit indicator tx: %w", err)
	}

	l.logger.Info("indicators loaded",
		zap.Int("enb_indicators", len(enbRecords)), zap.Int("enb_formulas", len(enbFormulas)),
		zap.Int("gsm_indicators", len(gsmRecords)), zap.Int("gsm_formulas", len(gsmFormulas)),
		zap.Int("gnb_indicators", len(gnbRecords)), zap.Int("gnb_formulas", len(gnbFormulas)))

	if rep.HasErrors() {
		return rep, rep.FirstError()
	}

	// 加载成功 → 使依赖指标库的 KPI 路由缓存失效（ISSUE-389：新增「统计时长」计数器后，
	// 旧路由白名单不含其 report_key，不 bump 则注入的统计时长落库被丢弃）。
	l.bumpDownstreamCache(ctx)

	return rep, nil
}

// bumpDownstreamCache 调用注入的 cacheBumper 使下游 KPI 路由缓存失效。
// bump 失败不回滚加载（指标库本身已一致），仅告警——下游路由最坏沿用 24h TTL 兜底失效。
// cacheBumper 为 nil（无 Redis 部署 / 测试场景）时为 no-op。
func (l *Loader) bumpDownstreamCache(ctx context.Context) {
	if l.cacheBumper == nil {
		return
	}
	if err := l.cacheBumper(ctx); err != nil {
		l.logger.Warn("indicator load succeeded but KPI route cache bump failed; stale routes will expire via TTL",
			zap.Error(err))
		return
	}
	l.logger.Info("KPI route cache version bumped after indicator load (downstream routes will rebuild)")
}

// parseDocs 是 T-0180 P1.2 引入的内聚 helper:把 resolveXxxSources 返回的 []fileSource
// 一次性解析为 []xmlIndicatorModel,每份 doc 写入 LoadedFrom + Platform fallback。
//
// platformFallback 用于 GSM("BSC")/ GNB("BaiBNQ") 的常量 fallback;ENB 传 ""
// 表示按文件名(去 .xml + 不含目录路径)推断 platform。
//
// 解析失败仅记录 rep.AddError 并 FilesSkipped++,不阻塞其他文件,保留与旧 Loader 容错语义一致。
func (l *Loader) parseDocs(srcs []fileSource, rep *dictloader.Report, platformFallback string) []xmlIndicatorModel {
	docs := make([]xmlIndicatorModel, 0, len(srcs))
	for _, src := range srcs {
		rep.FilesScanned++
		var doc xmlIndicatorModel
		if err := readXML(src.AbsPath, &doc); err != nil {
			rep.AddError(src.LoadedFrom, "parse", err)
			rep.FilesSkipped++
			continue
		}
		doc.LoadedFrom = src.LoadedFrom
		if doc.Platform == "" {
			if platformFallback != "" {
				doc.Platform = platformFallback
			} else {
				base := filepath.Base(src.LoadedFrom)
				doc.Platform = strings.TrimSuffix(base, filepath.Ext(base))
			}
		}
		docs = append(docs, doc)
		rep.FilesLoaded++
	}
	return docs
}

func readXML(path string, out interface{}) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := xml.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return nil
}

// indicatorRecord 是 T-0180 P1.2 引入的 dedupe 单元:
// 把 xmlIndicator 与其来源 LoadedFrom 配对,供 flushIndicators 写 loaded_from 列。
type indicatorRecord struct {
	Ind        xmlIndicator
	LoadedFrom string
}

// flattenDocsByDeviceType 跨多文件去重 indicator(按 id),并产生 formula 列表。
// 后处理(dedupe)策略:first-seen 胜出(与 alarm 模式一致);formula 全量保留(设计 §2.6 多平台公式)。
//
// T-0180 P1.2: 返回 map[string]indicatorRecord(替代旧 map[string]xmlIndicator),
// 让 LoadedFrom 与 indicator 同生命周期,first-seen 的来源被锁定;后续同 id 的 doc
// 即使 LoadedFrom 不同也不覆盖(保留首个文件作为权威源)。
func flattenDocsByDeviceType(docs []xmlIndicatorModel, isGnb bool) (map[string]indicatorRecord, []formulaRow) {
	records := make(map[string]indicatorRecord, 256)
	formulas := make([]formulaRow, 0, 1024)
	for _, doc := range docs {
		platform := doc.Platform
		for _, ind := range doc.Indicators {
			if ind.ID == "" {
				continue
			}
			if _, exists := records[ind.ID]; !exists {
				records[ind.ID] = indicatorRecord{Ind: ind, LoadedFrom: doc.LoadedFrom}
			}
			if ind.Formula == "" {
				continue
			}
			formulas = append(formulas, formulaRow{
				PlatformName: platform,
				IndicatorID:  ind.ID,
				Formula:      ind.Formula,
				LoadedFrom:   doc.LoadedFrom,
			})
		}
	}
	_ = isGnb // GNB 处理差异由 upsertIndicators 接管(无 indicator_level 列)
	return records, formulas
}

type formulaRow struct {
	PlatformName string
	IndicatorID  string
	Formula      string
	// LoadedFrom 是声明该公式/平台的 XML 文件相对路径(含 builtin/custom 前缀)。
	// 一个 XML 文件即一个平台,故同一 platform_name 的所有 formula 行此值单值;
	// SummaryByTech 按平台聚合时取它得「该平台的加载源」,并由前缀派生 source。
	LoadedFrom string
}

func ensureDefaultGroups(ctx context.Context, tx pgx.Tx) error {
	for _, dt := range []string{"enb", "gsm", "gnb"} {
		table := "indicator_group_" + dt
		// id=default, parent_id="" — 占位组（NOT NULL parent_id 取空字符串而非 NULL）
		q := fmt.Sprintf(`INSERT INTO %s (id, en_name, cn_name, parent_id, is_build_in, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING`, table)
		if _, err := tx.Exec(ctx, q, defaultGroupID, "Default Group", "默认功能集", "", "1", "T-0098 P1-06 占位组；UI 阶段补全分组层级"); err != nil {
			return fmt.Errorf("insert default %s: %w", table, err)
		}
	}
	return nil
}

func upsertIndicators(ctx context.Context, tx pgx.Tx, table string, records map[string]indicatorRecord, isGnb bool) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}
	const chunkSize = 200
	var batch []indicatorRecord
	for _, rec := range records {
		batch = append(batch, rec)
		if len(batch) >= chunkSize {
			if err := flushIndicators(ctx, tx, table, batch, isGnb); err != nil {
				return 0, err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := flushIndicators(ctx, tx, table, batch, isGnb); err != nil {
			return 0, err
		}
	}
	return len(records), nil
}

// indicatorColumns 返回 perf_indicators_* 的列清单(列顺序与 indicatorRowValues 严格对齐)。
//
// T-0180 P1.2: loaded_from 始终在 indicator_level 之前(非 GNB 时 indicator_level 尾列追加)。
// PM-P1: report_key 紧跟 loaded_from(reportKey 与 loaded_from 同侧、同款 nullIfEmpty 落库)。
func indicatorInsertColumns(isGnb bool) []string {
	cols := []string{
		"id", "en_name", "cn_name", "group_id",
		"data_type", "unit_id", "is_build_in", "is_counter",
		"arithmetic", "statis_type",
		"loaded_from", "report_key",
	}
	if !isGnb {
		cols = append(cols, "indicator_level")
	}
	return cols
}

// indicatorRowValues 把一条 indicatorRecord 展平为与 indicatorColumns 顺序对齐的 values。
// 抽成纯函数便于单测验证列↔值对位(尤其 report_key 与 en_name 各就各位)。
func indicatorInsertRow(rec indicatorRecord, isGnb bool) []interface{} {
	ind := rec.Ind
	enName := ind.EnName
	if enName == "" {
		enName = ind.ID // NOT NULL fallback
	}
	cnName := ind.CnName
	if cnName == "" {
		cnName = ind.ID
	}
	isBuiltIn := "1"
	if ind.IsBuildIn == "0" {
		isBuiltIn = "0"
	}
	isCounter := "1"
	if ind.IsCounter == "0" {
		isCounter = "0"
	}
	groupID := defaultGroupID
	if g := strings.TrimSpace(ind.GroupID); g != "" {
		groupID = g
	}
	row := []interface{}{
		ind.ID, enName, cnName, groupID,
		// issue #67 §4：XML dataType 中英混杂枚举 → 受控码（int/real/float）后落库。
		nullIfEmpty(normalizeDataType(ind.DataType)), nullIfEmpty(ind.UnitID), isBuiltIn, isCounter,
		nullIfEmpty(ind.Arithmetic), nullIfEmpty(ind.StatisType),
		nullIfEmpty(rec.LoadedFrom), nullIfEmpty(ind.ReportKey),
	}
	if !isGnb {
		row = append(row, nullIfEmpty(ind.IndicatorLevel))
	}
	return row
}

func flushIndicators(ctx context.Context, tx pgx.Tx, table string, batch []indicatorRecord, isGnb bool) error {
	cols := indicatorInsertColumns(isGnb)
	ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Insert(table).Columns(cols...)
	for _, rec := range batch {
		ib = ib.Values(indicatorInsertRow(rec, isGnb)...)
	}
	// group_id 跟随 XML 的 GroupID（缺省 default）；XML 是真相源，重启始终对齐 XML。
	updateClause := `ON CONFLICT (id) DO UPDATE SET
	    en_name      = EXCLUDED.en_name,
	    cn_name      = EXCLUDED.cn_name,
	    group_id     = EXCLUDED.group_id,
	    data_type    = EXCLUDED.data_type,
	    unit_id      = EXCLUDED.unit_id,
	    is_build_in  = EXCLUDED.is_build_in,
	    is_counter   = EXCLUDED.is_counter,
	    arithmetic   = EXCLUDED.arithmetic,
	    statis_type  = EXCLUDED.statis_type,
	    loaded_from  = EXCLUDED.loaded_from,
	    report_key   = EXCLUDED.report_key`
	if !isGnb {
		updateClause += `, indicator_level = EXCLUDED.indicator_level`
	}
	ib = ib.Suffix(updateClause)
	sqlStr, args, err := ib.ToSql()
	if err != nil {
		return fmt.Errorf("build %s sql: %w", table, err)
	}
	if _, err := tx.Exec(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("upsert %s: %w", table, err)
	}
	return nil
}

// formulaKey 标识一条公式归属的 (平台, 指标)，用于 reload 时判断该键是否已被
// 用户自定义公式覆盖。
type formulaKey struct {
	platform  string
	indicator string
}

// excludeCustomOverridden 过滤掉 (platform, indicator) 已存在自定义公式的 builtin 公式。
// 自定义公式（loaded_from IS NULL，经 UI 编辑写入）优先：reload 重灌 builtin 时跳过
// 这些键，避免与保留下来的自定义行重复，并尊重用户「编辑即替换」的意图（#98）。
func excludeCustomOverridden(formulas []formulaRow, customKeys map[formulaKey]struct{}) []formulaRow {
	if len(customKeys) == 0 {
		return formulas
	}
	kept := make([]formulaRow, 0, len(formulas))
	for _, f := range formulas {
		if _, overridden := customKeys[formulaKey{f.PlatformName, f.IndicatorID}]; overridden {
			continue
		}
		kept = append(kept, f)
	}
	return kept
}

// selectCustomFormulaKeys 读取表中自定义公式（loaded_from IS NULL = UI 写入）的
// (平台, 指标) 集合，供 reload 保留自定义行 + builtin 去重。
func selectCustomFormulaKeys(ctx context.Context, tx pgx.Tx, table string) (map[formulaKey]struct{}, error) {
	rows, err := tx.Query(ctx, fmt.Sprintf(
		"SELECT DISTINCT platform_name, indicator_id FROM %s WHERE loaded_from IS NULL", table))
	if err != nil {
		return nil, fmt.Errorf("select custom keys %s: %w", table, err)
	}
	defer rows.Close()
	keys := make(map[formulaKey]struct{})
	for rows.Next() {
		var k formulaKey
		if err := rows.Scan(&k.platform, &k.indicator); err != nil {
			return nil, fmt.Errorf("scan custom key %s: %w", table, err)
		}
		keys[k] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate custom keys %s: %w", table, err)
	}
	return keys, nil
}

// rewriteFormulas: 只重写 builtin 公式（loaded_from 非空），保留用户经 UI 编辑写入的
// 自定义公式（loaded_from IS NULL）。修复 #98：原实现 TRUNCATE 整表会连带清空自定义
// 公式。自定义公式优先——builtin 重插时跳过已被自定义覆盖的 (平台, 指标)。
func rewriteFormulas(ctx context.Context, tx pgx.Tx, table string, formulas []formulaRow) (int, error) {
	// 先读自定义公式键（loaded_from IS NULL），用于保留自定义行 + builtin 去重。
	customKeys, err := selectCustomFormulaKeys(ctx, tx, table)
	if err != nil {
		return 0, err
	}
	// 只删 builtin 行（loaded_from 非空），保留自定义行。
	if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE loaded_from IS NOT NULL", table)); err != nil {
		return 0, fmt.Errorf("delete builtin %s: %w", table, err)
	}
	toInsert := excludeCustomOverridden(formulas, customKeys)
	if len(toInsert) == 0 {
		return 0, nil
	}
	const chunkSize = 200
	for i := 0; i < len(toInsert); i += chunkSize {
		end := i + chunkSize
		if end > len(toInsert) {
			end = len(toInsert)
		}
		ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Insert(table).Columns("platform_name", "indicator_id", "formula", "loaded_from")
		for _, f := range toInsert[i:end] {
			ib = ib.Values(f.PlatformName, f.IndicatorID, f.Formula, f.LoadedFrom)
		}
		sqlStr, args, err := ib.ToSql()
		if err != nil {
			return 0, fmt.Errorf("build %s sql: %w", table, err)
		}
		if _, err := tx.Exec(ctx, sqlStr, args...); err != nil {
			return 0, fmt.Errorf("insert %s: %w", table, err)
		}
	}
	return len(toInsert), nil
}

// aggregateEnabledOR 计算 indicator_id → 有效 enabled 标志（多文件 OR 合并；设计 §2.6 / P2-09）。
//
// 语义：
//   - 同一 indicator_id 在任意一个文件标 enabled="true"（或缺省）→ true
//   - 仅当所有出现处都明确 enabled="false" / "0" → false
//   - 缺省 / 空字符串 / "true" / 其他 → true（与 XML 历史宽松约定一致）
func aggregateEnabledOR(docs []xmlIndicatorModel) map[string]bool {
	out := make(map[string]bool, 256)
	for _, doc := range docs {
		for _, ind := range doc.Indicators {
			if ind.ID == "" {
				continue
			}
			cur, exists := out[ind.ID]
			if !exists {
				out[ind.ID] = parseEnabledFlag(ind.Enabled)
				continue
			}
			// OR 合并：cur 已 true → 保持；否则取本次解析值
			if !cur {
				out[ind.ID] = parseEnabledFlag(ind.Enabled)
			}
		}
	}
	return out
}

// parseEnabledFlag 把 XML 字符串 enabled 属性解析为 bool。
//   - "" / "true" / "1" / "TRUE" / "T" → true
//   - "false" / "0" / "FALSE" / "F"   → false
//   - 其它 → true（容错，与 XML 现网约定一致）
func parseEnabledFlag(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "false", "0", "f", "no", "n":
		return false
	}
	return true
}

// refreshDefaultEnabledBucket 仅刷新 operator_code='default' 桶；不动其他 operator。
//
// 实现：
//  1. INSERT enabled=true 集合（ON CONFLICT DO NOTHING）
//  2. DELETE enabled=false 集合 WHERE operator_code='default' AND indicator_id IN (...)
//
// 返回 (insertedOrDeletedRows, err)。
//
// 注意：本函数不预先 TRUNCATE — 这是设计 §2.6 "桶刷新"语义的关键：
// 运营商在 UI 配的非 default 桶（cmcc / ctcc / cucc 等）必须保留。
func refreshDefaultEnabledBucket(ctx context.Context, tx pgx.Tx, table string, enabledMap map[string]bool) (int, error) {
	if len(enabledMap) == 0 {
		return 0, nil
	}
	enabledIDs := make([]string, 0, len(enabledMap))
	disabledIDs := make([]string, 0, len(enabledMap))
	for id, en := range enabledMap {
		if en {
			enabledIDs = append(enabledIDs, id)
		} else {
			disabledIDs = append(disabledIDs, id)
		}
	}

	rowsAffected := 0

	// 1. 插入启用集（ON CONFLICT 防重）
	if len(enabledIDs) > 0 {
		ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Insert(table).Columns("operator_code", "indicator_id")
		for _, id := range enabledIDs {
			ib = ib.Values("default", id)
		}
		ib = ib.Suffix("ON CONFLICT (operator_code, indicator_id) DO NOTHING")
		insSQL, args, err := ib.ToSql()
		if err != nil {
			return 0, fmt.Errorf("build insert %s default-bucket: %w", table, err)
		}
		tag, err := tx.Exec(ctx, insSQL, args...)
		if err != nil {
			return 0, fmt.Errorf("exec insert %s default-bucket: %w", table, err)
		}
		rowsAffected += int(tag.RowsAffected())
	}

	// 2. 删除禁用集 — 仅限 operator_code='default'
	if len(disabledIDs) > 0 {
		delSQL, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Delete(table).
			Where(sq.Eq{"operator_code": "default"}).
			Where(sq.Eq{"indicator_id": disabledIDs}).
			ToSql()
		if err != nil {
			return 0, fmt.Errorf("build delete %s default-bucket: %w", table, err)
		}
		tag, err := tx.Exec(ctx, delSQL, args...)
		if err != nil {
			return 0, fmt.Errorf("exec delete %s default-bucket: %w", table, err)
		}
		rowsAffected += int(tag.RowsAffected())
	}

	return rowsAffected, nil
}

func nullIfEmpty(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

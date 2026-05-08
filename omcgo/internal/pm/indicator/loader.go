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
)

// LoaderName 是 dictloader.Registry 中的注册名（T-0098 P1-06）。
const LoaderName = "indicator"

// defaultGroupID 是 P1-06 baseline 用的占位 indicator_group_*.id。
// 真正的多级分组由 Phase 2/3 handler/UI 维护；本 baseline 不解析 XML 中的功能集树（XML 当前未携带）。
const defaultGroupID = "default"

// Loader 实现 dictloader.Loader（T-0098 P1-06）。
//
// 加载语义（设计 §2.6 + 实施计划 §1.2 注："{enabled} / 多文件 OR 合并 / operator_code 桶刷新" 在 P2-09 增强）：
//   1. 扫描 enb/*.xml + GSM.xml + GNB.xml
//   2. 收集 distinct unitId → 插入 indicator_unit（按 id UPSERT）
//   3. 插入 indicator_group_{enb,gsm,gnb} 占位 group（id="default"，satisfy NOT NULL FK）
//   4. UPSERT perf_indicators_{enb,gsm,gnb}（按 id 唯一）
//   5. 重写 rela_platform_indicator_formula_{enb,gsm,gnb} — 每 (platform_name, indicator_id) 一行
//
// 不在 P1-06 范围（P2-09 接力）：
//   - enabled 属性 OR 合并语义（多文件取 OR）→ enabled_pm_indicators_* 写入
//   - operator_code='default' 桶刷新（仅刷新 default 行，不动其他 operator）
type Loader struct {
	pool   *pgxpool.Pool
	cfg    appconfig.IndicatorLoaderConfig
	base   string
	logger *zap.Logger
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

func (l *Loader) Name() string      { return LoaderName }
func (l *Loader) Directory() string { return l.cfg.BaseDirectory }

func (l *Loader) LoadOnce(ctx context.Context) (dictloader.Report, error) { return l.run(ctx) }
func (l *Loader) Reload(ctx context.Context) (dictloader.Report, error)   { return l.run(ctx) }

// xmlIndicatorModel 解析 indicator 单文件 XML（设计 §2.6）。
type xmlIndicatorModel struct {
	XMLName        xml.Name       `xml:"indicatorModel"`
	Platform       string         `xml:"platform,attr"`
	IndicatorCount int            `xml:"indicatorCount,attr"`
	Indicators     []xmlIndicator `xml:"indicators>indicator"`
}

type xmlIndicator struct {
	ID              string `xml:"id,attr"`
	EnName          string `xml:"enName,attr"`
	ReportKey       string `xml:"reportKey,attr"`
	CnName          string `xml:"cnName,attr"`
	IsBuildIn       string `xml:"isBuildIn,attr"`  // "1" / "0"
	IsCounter       string `xml:"isCounter,attr"`  // "1" / "0"
	DataType        string `xml:"dataType,attr"`
	UnitID          string `xml:"unitId,attr"`
	StatisType      string `xml:"statisType,attr"`
	Arithmetic      string `xml:"arithmetic,attr"`
	IndicatorLevel  string `xml:"indicatorLevel,attr"` // ENB/GSM only
	Formula         string `xml:"formula,attr"`
	Enabled         string `xml:"enabled,attr"`        // 缺省视 "true"（P2-09 接力 OR 合并）
}

func (l *Loader) run(ctx context.Context) (dictloader.Report, error) {
	rep := dictloader.NewReport(LoaderName)
	defer rep.Finish()

	// 1) ENB 子目录：所有 *.xml
	enbDir := filepath.Join(l.base, l.cfg.BaseDirectory, l.cfg.EnbSubdir)
	enbFiles, err := scanXMLFiles(enbDir)
	if err != nil {
		rep.AddError(l.cfg.EnbSubdir, "scan", err)
		return rep, fmt.Errorf("scan enb dir: %w", err)
	}
	enbDocs := make([]xmlIndicatorModel, 0, len(enbFiles))
	for _, f := range enbFiles {
		rep.FilesScanned++
		var doc xmlIndicatorModel
		if err := readXML(filepath.Join(enbDir, f), &doc); err != nil {
			rep.AddError(f, "parse", err)
			rep.FilesSkipped++
			continue
		}
		// 文件名（去 .xml）作为 platform_name fallback；XML 里 platform 属性优先
		if doc.Platform == "" {
			doc.Platform = strings.TrimSuffix(f, filepath.Ext(f))
		}
		enbDocs = append(enbDocs, doc)
		rep.FilesLoaded++
	}

	// 2) GSM / GNB 单文件
	var gsmDoc, gnbDoc xmlIndicatorModel
	gsmPath := filepath.Join(l.base, l.cfg.BaseDirectory, l.cfg.GsmFile)
	rep.FilesScanned++
	if err := readXML(gsmPath, &gsmDoc); err != nil {
		rep.AddError(l.cfg.GsmFile, "parse", err)
		rep.FilesSkipped++
	} else {
		if gsmDoc.Platform == "" {
			gsmDoc.Platform = "BSC"
		}
		rep.FilesLoaded++
	}
	gnbPath := filepath.Join(l.base, l.cfg.BaseDirectory, l.cfg.GnbFile)
	rep.FilesScanned++
	if err := readXML(gnbPath, &gnbDoc); err != nil {
		rep.AddError(l.cfg.GnbFile, "parse", err)
		rep.FilesSkipped++
	} else {
		if gnbDoc.Platform == "" {
			gnbDoc.Platform = "BaiBNQ"
		}
		rep.FilesLoaded++
	}

	// 3) 单事务写入 4 类表
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return rep, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 3a) units — 收集所有 distinct unitId
	allDocs := append([]xmlIndicatorModel{}, enbDocs...)
	allDocs = append(allDocs, gsmDoc, gnbDoc)
	unitsLoaded, err := upsertUnits(ctx, tx, allDocs)
	if err != nil {
		return rep, fmt.Errorf("upsert indicator_unit: %w", err)
	}
	rep.RowsAffected += unitsLoaded

	// 3b) groups — 三个 device type 各一个占位
	if err := ensureDefaultGroups(ctx, tx); err != nil {
		return rep, fmt.Errorf("ensure default groups: %w", err)
	}

	// 3c) ENB indicators + formulas — 多文件 dedupe 取最后一份；formula 每 platform 一行
	enbIndicators, enbFormulas := flattenDocsByDeviceType(enbDocs, false /*hasIndicatorLevel=true for ENB but not GNB*/)
	gsmIndicators, gsmFormulas := flattenDocsByDeviceType([]xmlIndicatorModel{gsmDoc}, false)
	gnbIndicators, gnbFormulas := flattenDocsByDeviceType([]xmlIndicatorModel{gnbDoc}, true)

	if n, err := upsertIndicators(ctx, tx, "perf_indicators_enb", enbIndicators, false); err != nil {
		return rep, err
	} else {
		rep.RowsAffected += n
	}
	if n, err := upsertIndicators(ctx, tx, "perf_indicators_gsm", gsmIndicators, false); err != nil {
		return rep, err
	} else {
		rep.RowsAffected += n
	}
	if n, err := upsertIndicators(ctx, tx, "perf_indicators_gnb", gnbIndicators, true); err != nil {
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
	gsmEnabled := aggregateEnabledOR([]xmlIndicatorModel{gsmDoc})
	gnbEnabled := aggregateEnabledOR([]xmlIndicatorModel{gnbDoc})
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
		zap.Int("enb_indicators", len(enbIndicators)), zap.Int("enb_formulas", len(enbFormulas)),
		zap.Int("gsm_indicators", len(gsmIndicators)), zap.Int("gsm_formulas", len(gsmFormulas)),
		zap.Int("gnb_indicators", len(gnbIndicators)), zap.Int("gnb_formulas", len(gnbFormulas)),
		zap.Int("units", unitsLoaded))

	if rep.HasErrors() {
		return rep, rep.FirstError()
	}
	return rep, nil
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

func scanXMLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if !strings.EqualFold(filepath.Ext(e.Name()), ".xml") {
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

// flattenDocsByDeviceType 跨多文件去重 indicator（按 id），并产生 formula 列表。
// 后处理（dedupe）策略：first-seen 胜出（与 alarm 模式一致）；formula 全量保留（设计 §2.6 多平台公式）。
func flattenDocsByDeviceType(docs []xmlIndicatorModel, isGnb bool) (map[string]xmlIndicator, []formulaRow) {
	indicators := make(map[string]xmlIndicator, 256)
	formulas := make([]formulaRow, 0, 1024)
	for _, doc := range docs {
		platform := doc.Platform
		for _, ind := range doc.Indicators {
			if ind.ID == "" {
				continue
			}
			if _, exists := indicators[ind.ID]; !exists {
				indicators[ind.ID] = ind
			}
			if ind.Formula == "" {
				continue
			}
			formulas = append(formulas, formulaRow{
				PlatformName: platform,
				IndicatorID:  ind.ID,
				Formula:      ind.Formula,
			})
		}
	}
	_ = isGnb // GNB 处理差异由 upsertIndicators 接管（无 indicator_level 列）
	return indicators, formulas
}

type formulaRow struct {
	PlatformName string
	IndicatorID  string
	Formula      string
}

func upsertUnits(ctx context.Context, tx pgx.Tx, docs []xmlIndicatorModel) (int, error) {
	uniq := make(map[string]struct{}, 32)
	for _, d := range docs {
		for _, i := range d.Indicators {
			if u := strings.TrimSpace(i.UnitID); u != "" {
				uniq[u] = struct{}{}
			}
		}
	}
	if len(uniq) == 0 {
		return 0, nil
	}
	ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Insert("indicator_unit").Columns("id", "en_name", "cn_name")
	for u := range uniq {
		ib = ib.Values(u, u, u) // P1-06 baseline：en/cn name 占位等同 id；UI 工作时再补
	}
	ib = ib.Suffix("ON CONFLICT (id) DO NOTHING")
	sqlStr, args, err := ib.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build indicator_unit sql: %w", err)
	}
	if _, err := tx.Exec(ctx, sqlStr, args...); err != nil {
		return 0, err
	}
	return len(uniq), nil
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

func upsertIndicators(ctx context.Context, tx pgx.Tx, table string, indicators map[string]xmlIndicator, isGnb bool) (int, error) {
	if len(indicators) == 0 {
		return 0, nil
	}
	const chunkSize = 200
	var batch []xmlIndicator
	for _, ind := range indicators {
		batch = append(batch, ind)
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
	return len(indicators), nil
}

func flushIndicators(ctx context.Context, tx pgx.Tx, table string, batch []xmlIndicator, isGnb bool) error {
	cols := []string{
		"id", "en_name", "cn_name", "group_id",
		"data_type", "unit_id", "is_build_in", "is_counter",
		"arithmetic", "statis_type",
	}
	if !isGnb {
		cols = append(cols, "indicator_level")
	}
	ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Insert(table).Columns(cols...)
	for _, ind := range batch {
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
		row := []interface{}{
			ind.ID, enName, cnName, defaultGroupID,
			nullIfEmpty(ind.DataType), nullIfEmpty(ind.UnitID), isBuiltIn, isCounter,
			nullIfEmpty(ind.Arithmetic), nullIfEmpty(ind.StatisType),
		}
		if !isGnb {
			row = append(row, nullIfEmpty(ind.IndicatorLevel))
		}
		ib = ib.Values(row...)
	}
	updateClause := `ON CONFLICT (id) DO UPDATE SET
	    en_name      = EXCLUDED.en_name,
	    cn_name      = EXCLUDED.cn_name,
	    group_id     = EXCLUDED.group_id,
	    data_type    = EXCLUDED.data_type,
	    unit_id      = EXCLUDED.unit_id,
	    is_build_in  = EXCLUDED.is_build_in,
	    is_counter   = EXCLUDED.is_counter,
	    arithmetic   = EXCLUDED.arithmetic,
	    statis_type  = EXCLUDED.statis_type`
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

// rewriteFormulas: TRUNCATE then batch INSERT — formula 每次 reload 全量重写（设计 §2.6 加载流程）
func rewriteFormulas(ctx context.Context, tx pgx.Tx, table string, formulas []formulaRow) (int, error) {
	if _, err := tx.Exec(ctx, fmt.Sprintf("TRUNCATE %s RESTART IDENTITY", table)); err != nil {
		return 0, fmt.Errorf("truncate %s: %w", table, err)
	}
	if len(formulas) == 0 {
		return 0, nil
	}
	const chunkSize = 200
	for i := 0; i < len(formulas); i += chunkSize {
		end := i + chunkSize
		if end > len(formulas) {
			end = len(formulas)
		}
		ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Insert(table).Columns("platform_name", "indicator_id", "formula")
		for _, f := range formulas[i:end] {
			ib = ib.Values(f.PlatformName, f.IndicatorID, f.Formula)
		}
		sqlStr, args, err := ib.ToSql()
		if err != nil {
			return 0, fmt.Errorf("build %s sql: %w", table, err)
		}
		if _, err := tx.Exec(ctx, sqlStr, args...); err != nil {
			return 0, fmt.Errorf("insert %s: %w", table, err)
		}
	}
	return len(formulas), nil
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
//   1. INSERT enabled=true 集合（ON CONFLICT DO NOTHING）
//   2. DELETE enabled=false 集合 WHERE operator_code='default' AND indicator_id IN (...)
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

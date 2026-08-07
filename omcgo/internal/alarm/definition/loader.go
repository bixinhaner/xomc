package definition

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/dictloader"
)

const LoaderName = "alarm-definition"

// Loader 实现 dictloader.Loader（T-0098 P1-06）。
//
// 加载语义（设计 §3.4）：
//  1. 扫描 {XMLBaseDir}/{Directory}/*.xml — 文件名（去 .xml）即 ne_type
//  2. 解析每个 alarmModel.alarms → 校验 severity 字符串属于 4 行种子（P1-04 已写）
//  3. UPSERT alarm_definitions（按 identifier 唯一）；severity_id 由 severity name → severity_levels lookup 解析
//  4. 跨文件 identifier 全局唯一性检查（同一 identifier 在不同 ne_type 中冲突时跳过 + ERROR）
type Loader struct {
	pool   *pgxpool.Pool
	cfg    appconfig.AlarmDefinitionLoaderConfig
	base   string
	logger *zap.Logger
}

func NewLoader(pool *pgxpool.Pool, cfg appconfig.AlarmDefinitionLoaderConfig, baseDir string, logger *zap.Logger) *Loader {
	if cfg.Directory == "" {
		cfg.Directory = BuiltinDirSubdir
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Loader{pool: pool, cfg: cfg, base: baseDir, logger: logger.Named(LoaderName)}
}

func (l *Loader) Name() string      { return LoaderName }
func (l *Loader) Directory() string { return l.cfg.Directory }

func (l *Loader) LoadOnce(ctx context.Context) (dictloader.Report, error) { return l.run(ctx) }
func (l *Loader) Reload(ctx context.Context) (dictloader.Report, error)   { return l.run(ctx) }

func (l *Loader) run(ctx context.Context) (dictloader.Report, error) {
	rep := dictloader.NewReport(LoaderName)
	defer rep.Finish()

	// 三库 XML 导入重构(2026-06-04 单目录):仅扫 alarm-definitions/,跳过 sidecar。
	sources, err := l.resolveSources()
	if err != nil {
		return rep, err
	}

	severityMap, err := l.fetchSeverityMap(ctx)
	if err != nil {
		return rep, fmt.Errorf("fetch severity_levels: %w", err)
	}
	if len(severityMap) == 0 {
		return rep, fmt.Errorf("alarm_severity_levels empty (P1-04 seed not loaded?)")
	}

	// 跨文件 identifier 去重：记录来源和实际入库字段，用于识别一次性的 ENB→GSM
	// 内置定义迁移兼容；其余重复 identifier 仍然硬失败。
	seenIdentifier := make(map[string]seenAlarmIdentifier, 512)

	for _, src := range sources {
		rep.FilesScanned++
		name := filepath.Base(src.AbsPath)
		rows, err := l.loadAlarmFile(ctx, src.AbsPath, src.LoadedFrom, severityMap, seenIdentifier)
		if err != nil {
			rep.AddError(name, "parse-or-persist", err)
			rep.FilesSkipped++
			l.logger.Error("alarm file failed", zap.String("file", name), zap.Error(err))
			continue
		}
		rep.FilesLoaded++
		rep.RowsAffected += rows
	}

	if rep.HasErrors() {
		return rep, rep.FirstError()
	}
	return rep, nil
}

// alarmFileSource 配对一个待加载 XML 的绝对路径与其 loaded_from 列值(含目录前缀)。
type alarmFileSource struct {
	AbsPath    string // os.ReadFile 用
	LoadedFrom string // "alarm-definitions/ENB.xml"
}

// resolveSources 扫描单目录 cfg.Directory（alarm-definitions/，builtin + custom XML 同住）,
// 跳过隐藏文件、非 .xml 文件与 sidecar 标记文件(X.xml.custom)。
// LoadedFrom 始终带目录前缀,供 ClassifySource / IsDeletable 据 sidecar 判定来源。
// 按 basename 字典序排序,保证跨文件 identifier first-seen 去重的确定性。
func (l *Loader) resolveSources() ([]alarmFileSource, error) {
	dir := filepath.Join(l.base, l.cfg.Directory)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 目录可缺(空库 / 本地裸跑),静默返回空
		}
		return nil, fmt.Errorf("read alarm dir %s: %w", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		// 跳过 sidecar 标记文件(X.xml.custom);其 ext 非 .xml,EqualFold 也会排除,显式跳更清晰。
		if strings.HasSuffix(e.Name(), CustomMarkerSuffix) {
			continue
		}
		if !strings.EqualFold(filepath.Ext(e.Name()), ".xml") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	out := make([]alarmFileSource, 0, len(names))
	for _, name := range names {
		out = append(out, alarmFileSource{
			AbsPath:    filepath.Join(dir, name),
			LoadedFrom: filepath.ToSlash(filepath.Join(l.cfg.Directory, name)),
		})
	}
	return out, nil
}

func (l *Loader) loadAlarmFile(ctx context.Context, path, loadedFrom string, severityMap map[string]string, seen map[string]seenAlarmIdentifier) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	var doc xmlAlarmModel
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return 0, fmt.Errorf("xml unmarshal %s: %w", path, err)
	}
	neType := doc.NeType
	if neType == "" {
		// 回退：从文件名推断 ne_type（设计 §3.4 加载流程 1）
		neType = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		neType = strings.ToUpper(neType)
	}
	if err := validateAlarmIdentifiers(doc.Alarms, seen, filepath.Base(path), neType); err != nil {
		return 0, err
	}
	for _, alarm := range doc.Alarms {
		if first, ok := seen[strings.TrimSpace(alarm.Identifier)]; ok &&
			isLegacyGSMReclassificationDuplicate(alarm, first, filepath.Base(path), neType) {
			l.logger.Warn("accept legacy ENB to GSM alarm reclassification",
				zap.String("identifier", alarm.Identifier),
				zap.String("from", first.Filename),
				zap.String("to", filepath.Base(path)))
		}
	}

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := batchUpsertAlarms(ctx, tx, neType, loadedFrom, doc.Alarms, severityMap)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit alarm tx (%s): %w", neType, err)
	}
	for _, a := range doc.Alarms {
		if a.Identifier != "" {
			seen[a.Identifier] = seenAlarmIdentifier{
				Filename: filepath.Base(path),
				NeType:   neType,
				Alarm:    a,
			}
		}
	}
	l.logger.Info("alarm file loaded",
		zap.String("ne_type", neType),
		zap.String("file", filepath.Base(path)),
		zap.Int("rows", rows))
	return rows, nil
}

// validateAlarmIdentifiers 在写库前检查同一 XML 和跨 XML 的 identifier 冲突。
// identifier 是 alarm_definitions 的全局唯一键，Registry 也仅按 identifier 查询；
// 继续跳过重复项会让新导入的库没有任何可展示行却误报导入成功。
type seenAlarmIdentifier struct {
	Filename string
	NeType   string
	Alarm    xmlAlarm
}

var gsmReclassifiedAlarmIdentifiers = map[string]struct{}{
	"60001": {},
	"60002": {},
	"60003": {},
	"60004": {},
	"60005": {},
}

// isLegacyGSMReclassificationDuplicate 只兼容 #189 的一次性数据归属迁移：
// 升级时安装器会保留运维修改过的旧 ENB.xml，同时补入新版 GSM.xml。若旧 ENB 中
// 仍保留原样的 60001-60005，则允许后加载的 GSM.xml 通过并由 UPSERT 将最终归属改为
// GSM。只要文件对、identifier 或任一实际入库字段不匹配，仍按真实跨库冲突拒绝。
func isLegacyGSMReclassificationDuplicate(alarm xmlAlarm, first seenAlarmIdentifier, filename, neType string) bool {
	if !strings.EqualFold(first.Filename, "ENB.xml") || !strings.EqualFold(filename, "GSM.xml") {
		return false
	}
	if first.NeType != "ENB" || neType != "GSM" {
		return false
	}
	if _, ok := gsmReclassifiedAlarmIdentifiers[strings.TrimSpace(alarm.Identifier)]; !ok {
		return false
	}
	return first.Alarm == alarm
}

func validateAlarmIdentifiers(alarms []xmlAlarm, seen map[string]seenAlarmIdentifier, filename, neType string) error {
	if len(alarms) == 0 {
		return fmt.Errorf("alarm XML %s contains no alarm definitions", filename)
	}

	local := make(map[string]struct{}, len(alarms))
	for _, a := range alarms {
		identifier := strings.TrimSpace(a.Identifier)
		if identifier == "" {
			return fmt.Errorf("alarm XML %s contains an alarm without identifier", filename)
		}
		if first, ok := seen[identifier]; ok &&
			!isLegacyGSMReclassificationDuplicate(a, first, filename, neType) {
			return fmt.Errorf("alarm identifier %q in %s conflicts with %s", identifier, filename, first.Filename)
		}
		if _, ok := local[identifier]; ok {
			return fmt.Errorf("alarm identifier %q is duplicated in %s", identifier, filename)
		}
		local[identifier] = struct{}{}
	}
	return nil
}

func batchUpsertAlarms(ctx context.Context, tx pgx.Tx, neType, loadedFrom string, alarms []xmlAlarm, severityMap map[string]string) (int, error) {
	const chunkSize = 200
	rows := 0
	pending := make([]xmlAlarm, 0, chunkSize)

	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Insert("alarm_definitions").Columns(
			"identifier", "ne_type", "cn_name", "en_name",
			"severity_id", "event_type",
			"cn_probable_cause", "en_probable_cause", "cn_suggestion", "en_suggestion", "is_show",
			"loaded_from",
		)
		for _, a := range pending {
			sevID, ok := severityMap[a.Severity]
			if !ok {
				return fmt.Errorf("alarm %s: severity %q not in alarm_severity_levels (4 seed rows: Critical/Major/Minor/Warning)", a.Identifier, a.Severity)
			}
			var eventType interface{}
			if v := strings.TrimSpace(a.EventType); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					eventType = n
				}
			}
			isShow := !strings.EqualFold(strings.TrimSpace(a.IsShow), "N") // 缺省 / 非 "N" → true
			ib = ib.Values(
				a.Identifier, neType,
				nullIfEmpty(a.CnName), nullIfEmpty(a.EnName),
				sevID, eventType,
				nullIfEmpty(a.CnProbableCause), nullIfEmpty(a.EnProbableCause),
				nullIfEmpty(a.CnSuggestion), nullIfEmpty(a.EnSuggestion),
				isShow,
				nullIfEmpty(loadedFrom),
			)
		}
		// ON CONFLICT DO UPDATE 保 idempotent；冲突 identifier 同 file 内不可能（XML 单文件唯一），跨文件已被 seen 拦截
		ib = ib.Suffix(`ON CONFLICT (identifier) DO UPDATE
		SET ne_type           = EXCLUDED.ne_type,
		    cn_name           = EXCLUDED.cn_name,
		    en_name           = EXCLUDED.en_name,
		    severity_id       = EXCLUDED.severity_id,
		    event_type        = EXCLUDED.event_type,
		    cn_probable_cause = EXCLUDED.cn_probable_cause,
		    en_probable_cause = EXCLUDED.en_probable_cause,
		    cn_suggestion     = EXCLUDED.cn_suggestion,
		    en_suggestion     = EXCLUDED.en_suggestion,
		    is_show           = EXCLUDED.is_show,
		    loaded_from       = EXCLUDED.loaded_from`)
		sqlStr, args, err := ib.ToSql()
		if err != nil {
			return fmt.Errorf("build sql alarm_definitions: %w", err)
		}
		if _, err := tx.Exec(ctx, sqlStr, args...); err != nil {
			return fmt.Errorf("batch upsert alarm_definitions: %w", err)
		}
		rows += len(pending)
		pending = pending[:0]
		return nil
	}

	for _, a := range alarms {
		pending = append(pending, a)
		if len(pending) >= chunkSize {
			if err := flush(); err != nil {
				return rows, err
			}
		}
	}
	if err := flush(); err != nil {
		return rows, err
	}
	return rows, nil
}

func (l *Loader) fetchSeverityMap(ctx context.Context) (map[string]string, error) {
	rows, err := l.pool.Query(ctx, `SELECT name, id::text FROM alarm_severity_levels`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string, 4)
	for rows.Next() {
		var name, id string
		if err := rows.Scan(&name, &id); err != nil {
			return nil, err
		}
		out[name] = id
	}
	return out, rows.Err()
}

func nullIfEmpty(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

package definition

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
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
//   1. 扫描 {XMLBaseDir}/{Directory}/*.xml — 文件名（去 .xml）即 ne_type
//   2. 解析每个 alarmModel.alarms → 校验 severity 字符串属于 4 行种子（P1-04 已写）
//   3. UPSERT alarm_definitions（按 identifier 唯一）；severity_id 由 severity name → severity_levels lookup 解析
//   4. 跨文件 identifier 全局唯一性检查（同一 identifier 在不同 ne_type 中冲突时跳过 + ERROR）
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
	if cfg.CustomDirectory == "" {
		cfg.CustomDirectory = CustomDirSubdir
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

	// T-0180 对标:builtin + custom 双目录合并扫描;同名文件按 CustomOverrides 决定胜负。
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

	// 跨文件 identifier 去重：first-seen 胜出，重复 identifier 记录 error 并跳过
	seenIdentifier := make(map[string]string, 512)

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
	LoadedFrom string // "alarm-definitions/ENB.xml" 或 "alarm-definitions-custom/MY.xml"
}

// resolveSources 合并扫描 builtin（cfg.Directory）与 custom（cfg.CustomDirectory）两目录,
// 按 basename 去重:CustomOverridesEnabled() 决定同名时谁胜出(默认 custom 胜出)。
// custom 目录允许不存在(尚无任何上传)。LoadedFrom 始终带目录前缀,供 ClassifySource 判定。
func (l *Loader) resolveSources() ([]alarmFileSource, error) {
	byName := make(map[string]alarmFileSource, 16)
	order := make([]string, 0, 16)

	scan := func(subdir string) error {
		dir := filepath.Join(l.base, subdir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // builtin 必有(已 ensure),custom 可缺
			}
			return fmt.Errorf("read alarm dir %s: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if !strings.EqualFold(filepath.Ext(e.Name()), ".xml") {
				continue
			}
			name := e.Name()
			if _, ok := byName[name]; !ok {
				order = append(order, name)
			}
			byName[name] = alarmFileSource{
				AbsPath:    filepath.Join(dir, name),
				LoadedFrom: filepath.ToSlash(filepath.Join(subdir, name)),
			}
		}
		return nil
	}

	// 后扫描者覆盖先扫描者:customWins → 先 builtin 后 custom;否则相反。
	first, second := l.cfg.Directory, l.cfg.CustomDirectory
	if !l.cfg.CustomOverridesEnabled() {
		first, second = l.cfg.CustomDirectory, l.cfg.Directory
	}
	if err := scan(first); err != nil {
		return nil, err
	}
	if err := scan(second); err != nil {
		return nil, err
	}

	out := make([]alarmFileSource, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
	}
	return out, nil
}

func (l *Loader) loadAlarmFile(ctx context.Context, path, loadedFrom string, severityMap map[string]string, seen map[string]string) (int, error) {
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

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := batchUpsertAlarms(ctx, tx, neType, loadedFrom, doc.Alarms, severityMap, seen, l.logger)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit alarm tx (%s): %w", neType, err)
	}
	l.logger.Info("alarm file loaded",
		zap.String("ne_type", neType),
		zap.String("file", filepath.Base(path)),
		zap.Int("rows", rows))
	return rows, nil
}

func batchUpsertAlarms(ctx context.Context, tx pgx.Tx, neType, loadedFrom string, alarms []xmlAlarm, severityMap map[string]string, seen map[string]string, logger *zap.Logger) (int, error) {
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
			"cn_probable_cause", "en_probable_cause",
			"cn_suggestion", "en_suggestion", "is_show",
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
		if a.Identifier == "" {
			continue
		}
		if firstFile, ok := seen[a.Identifier]; ok {
			logger.Error("duplicate alarm identifier across files; keep first-seen",
				zap.String("identifier", a.Identifier),
				zap.String("first_seen_in", firstFile),
				zap.String("ne_type_now", neType))
			continue
		}
		seen[a.Identifier] = neType + ".xml"
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

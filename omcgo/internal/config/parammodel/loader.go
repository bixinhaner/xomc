package parammodel

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

// LoaderName 是 dictloader.Registry 中的注册名（也用于日志域）。
const LoaderName = "param-model"

// Loader 实现 dictloader.Loader（T-0098 P1-06）。
//
// 加载语义（设计 §1.7）：
//   1. 扫描 ParamModelFiles 白名单 → 每个 XML 一次事务（DELETE param_mappings + UPSERT param_models + 批量 INSERT param_mappings）
//   2. 加载 StandardModelFile → standard_params 全量重写
//
// 注意：本 P1-06 baseline 不实现 `{i}` 占位符校验、跨 XML 字段冲突合并（设计 §1.7.1 / §1.9）；
// 这两项放在 Phase 2 P2-02 ParamRegistry / Intersect。
type Loader struct {
	pool   *pgxpool.Pool
	cfg    appconfig.ParamModelLoaderConfig
	base   string
	logger *zap.Logger
}

// NewLoader 构造 ParamModel Loader。
// baseDir 来自 DictLoaderConfig.XMLBaseDir。空白 cfg 字段使用默认值。
func NewLoader(pool *pgxpool.Pool, cfg appconfig.ParamModelLoaderConfig, baseDir string, logger *zap.Logger) *Loader {
	if cfg.Directory == "" {
		cfg.Directory = "param-mappings"
	}
	if cfg.StandardModelFile == "" {
		cfg.StandardModelFile = "standard-model.xml"
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Loader{pool: pool, cfg: cfg, base: baseDir, logger: logger.Named(LoaderName)}
}

// Name implements dictloader.Loader.
func (l *Loader) Name() string { return LoaderName }

// Directory implements dictloader.Loader.
func (l *Loader) Directory() string { return l.cfg.Directory }

// LoadOnce implements dictloader.Loader.
// 启动期入口：依次加载 paramModel + standardModel。
func (l *Loader) LoadOnce(ctx context.Context) (dictloader.Report, error) {
	return l.run(ctx)
}

// Reload implements dictloader.Loader. 当前与 LoadOnce 等价（UPSERT 语义天然幂等）。
func (l *Loader) Reload(ctx context.Context) (dictloader.Report, error) {
	return l.run(ctx)
}

func (l *Loader) run(ctx context.Context) (dictloader.Report, error) {
	rep := dictloader.NewReport(LoaderName)
	defer rep.Finish()

	dir := filepath.Join(l.base, l.cfg.Directory)

	// Pass 1: 9 个 paramModel.xml
	files := l.cfg.ParamModelFiles
	if len(files) == 0 {
		// 缺省白名单：所有非保留文件。Scanner 已剔除 dotfiles 与子目录；
		// 我们再排除 standard-model.xml / products.xml / 旧 routing。
		all, err := scanXMLFiles(dir)
		if err != nil {
			return rep, fmt.Errorf("scan param-mappings: %w", err)
		}
		files = pruneReserved(all, []string{
			l.cfg.StandardModelFile,
			"products.xml",
			"product-name-routing.xml",
			"param-model-routing.xml",
		})
	}

	for _, name := range files {
		path := filepath.Join(dir, name)
		rep.FilesScanned++
		rows, err := l.loadParamModelFile(ctx, path)
		if err != nil {
			rep.AddError(name, "parse-or-persist", err)
			rep.FilesSkipped++
			l.logger.Error("param-model file failed", zap.String("file", name), zap.Error(err))
			continue
		}
		rep.FilesLoaded++
		rep.RowsAffected += rows
	}

	// Pass 2: standard-model.xml（单文件）
	stdPath := filepath.Join(dir, l.cfg.StandardModelFile)
	rep.FilesScanned++
	if rows, err := l.loadStandardModelFile(ctx, stdPath); err != nil {
		rep.AddError(l.cfg.StandardModelFile, "parse-or-persist", err)
		rep.FilesSkipped++
		l.logger.Error("standard-model file failed", zap.String("file", l.cfg.StandardModelFile), zap.Error(err))
	} else {
		rep.FilesLoaded++
		rep.RowsAffected += rows
	}

	if rep.HasErrors() {
		return rep, rep.FirstError()
	}
	return rep, nil
}

// loadParamModelFile 单文件加载：upsert param_models + 重写 param_mappings。
func (l *Loader) loadParamModelFile(ctx context.Context, path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	var doc xmlParameterModel
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return 0, fmt.Errorf("xml unmarshal %s: %w", path, err)
	}
	if doc.ParamModel == "" {
		return 0, fmt.Errorf("paramModel attribute empty in %s", path)
	}

	loadedFrom := filepath.Base(path)
	totalObjects := len(doc.Objects)
	totalParams := len(doc.Params)
	totalEntries := doc.TotalEntries
	if totalEntries == 0 {
		totalEntries = totalObjects + totalParams
	}

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// UPSERT param_models（按 name 唯一）
	const upsertModel = `INSERT INTO param_models (name, total_entries, total_objects, total_params, is_active, loaded_from)
	VALUES ($1, $2, $3, $4, TRUE, $5)
	ON CONFLICT (name) DO UPDATE
	SET total_entries = EXCLUDED.total_entries,
	    total_objects = EXCLUDED.total_objects,
	    total_params  = EXCLUDED.total_params,
	    is_active     = TRUE,
	    loaded_from   = EXCLUDED.loaded_from
	RETURNING id`
	var modelID string
	if err := tx.QueryRow(ctx, upsertModel, doc.ParamModel, totalEntries, totalObjects, totalParams, loadedFrom).Scan(&modelID); err != nil {
		return 0, fmt.Errorf("upsert param_models %q: %w", doc.ParamModel, err)
	}

	// 重写 param_mappings：先按 model 删旧，再批量插入
	if _, err := tx.Exec(ctx, "DELETE FROM param_mappings WHERE param_model_id = $1", modelID); err != nil {
		return 0, fmt.Errorf("clear param_mappings for %q: %w", doc.ParamModel, err)
	}

	rows := 0
	if len(doc.Objects) > 0 {
		if n, err := batchInsertMappings(ctx, tx, modelID, doc.Objects, "object"); err != nil {
			return 0, err
		} else {
			rows += n
		}
	}
	if len(doc.Params) > 0 {
		if n, err := batchInsertMappings(ctx, tx, modelID, doc.Params, "parameter"); err != nil {
			return 0, err
		} else {
			rows += n
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	rows++ // count the param_models row
	l.logger.Info("param-model loaded",
		zap.String("model", doc.ParamModel),
		zap.Int("objects", totalObjects),
		zap.Int("params", totalParams),
		zap.String("file", loadedFrom))
	return rows, nil
}

// batchInsertMappings 把 entries 批量插入 param_mappings；entryType 固定。
// 单批次 ~200 行（pgx CopyFrom 不适用因含 NULL 与 CHECK，回退批量 INSERT VALUES）。
func batchInsertMappings(ctx context.Context, tx pgx.Tx, modelID string, entries []xmlParamEntry, entryType string) (int, error) {
	const chunkSize = 200
	total := 0
	for i := 0; i < len(entries); i += chunkSize {
		end := i + chunkSize
		if end > len(entries) {
			end = len(entries)
		}
		ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Insert("param_mappings").Columns(
			"param_model_id", "standard_path", "private_path", "entry_type",
			"access", "data_type", "change_applies",
			"min_value", "max_value", "is_storable", "is_active",
		)
		for _, e := range entries[i:end] {
			std := e.StandardPath
			if std == "" {
				std = e.Name // 容错：少数 XML 可能未给 standardPath，回退 name
			}
			ib = ib.Values(
				modelID,
				std,
				e.Name,
				entryType,
				nullIfEmpty(e.Access),
				nullIfEmpty(e.DataType),
				nullIfEmpty(e.ChangeApplies),
				parseNullableInt(e.Min),
				parseNullableInt(e.Max),
				!strings.EqualFold(strings.TrimSpace(e.Store), "false"), // 缺省 / 任意非 "false" → true
				true,
			)
		}
		sqlStr, args, err := ib.ToSql()
		if err != nil {
			return total, fmt.Errorf("build sql param_mappings (%s, batch %d-%d): %w", entryType, i, end, err)
		}
		if _, err := tx.Exec(ctx, sqlStr, args...); err != nil {
			return total, fmt.Errorf("batch insert param_mappings (%s, batch %d-%d): %w", entryType, i, end, err)
		}
		total += end - i
	}
	return total, nil
}

// loadStandardModelFile 全量重写 standard_params 表。
func (l *Loader) loadStandardModelFile(ctx context.Context, path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	var doc xmlStandardModel
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return 0, fmt.Errorf("xml unmarshal %s: %w", path, err)
	}

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `TRUNCATE standard_params RESTART IDENTITY`); err != nil {
		return 0, fmt.Errorf("truncate standard_params: %w", err)
	}

	rows := 0
	if n, err := batchInsertStandardEntries(ctx, tx, doc.Objects, "object"); err != nil {
		return 0, err
	} else {
		rows += n
	}
	if n, err := batchInsertStandardEntries(ctx, tx, doc.Params, "parameter"); err != nil {
		return 0, err
	} else {
		rows += n
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit standard tx: %w", err)
	}
	l.logger.Info("standard-model loaded", zap.Int("objects", len(doc.Objects)), zap.Int("params", len(doc.Params)))
	return rows, nil
}

func batchInsertStandardEntries(ctx context.Context, tx pgx.Tx, entries []xmlStandardEntry, entryType string) (int, error) {
	const chunkSize = 200
	total := 0
	for i := 0; i < len(entries); i += chunkSize {
		end := i + chunkSize
		if end > len(entries) {
			end = len(entries)
		}
		ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Insert("standard_params").Columns(
			"standard_path", "entry_type", "access", "data_type",
			"change_applies", "min_value", "max_value",
		)
		for _, e := range entries[i:end] {
			ib = ib.Values(
				e.StandardPath,
				entryType,
				nullIfEmpty(e.Access),
				nullIfEmpty(e.DataType),
				nullIfEmpty(e.ChangeApplies),
				parseNullableInt(e.Min),
				parseNullableInt(e.Max),
			)
		}
		sqlStr, args, err := ib.ToSql()
		if err != nil {
			return total, fmt.Errorf("build sql standard_params (%s, batch %d-%d): %w", entryType, i, end, err)
		}
		if _, err := tx.Exec(ctx, sqlStr, args...); err != nil {
			// 容错：standard_params 唯一索引在 standard_path 上，重复条目（多文件可能产生）跳过
			if strings.Contains(err.Error(), "duplicate key") {
				continue
			}
			return total, fmt.Errorf("batch insert standard_params (%s, batch %d-%d): %w", entryType, i, end, err)
		}
		total += end - i
	}
	return total, nil
}

// ── 通用工具 ──────────────────────────────────────────────────────────

func nullIfEmpty(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func parseNullableInt(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return n
}

// scanXMLFiles 列举目录下所有 *.xml 的 basename。
func scanXMLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.EqualFold(filepath.Ext(name), ".xml") {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}

func pruneReserved(all, reserved []string) []string {
	skip := make(map[string]struct{}, len(reserved))
	for _, r := range reserved {
		skip[r] = struct{}{}
	}
	out := make([]string, 0, len(all))
	for _, n := range all {
		if _, ok := skip[n]; ok {
			continue
		}
		out = append(out, n)
	}
	return out
}

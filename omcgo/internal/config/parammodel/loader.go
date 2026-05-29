package parammodel

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
	"github.com/google/uuid"
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
// 加载语义（设计 §1.7 + T-0178 §9.1 分层目录）：
//  1. 扫描 builtin (cfg.Directory) + custom (Loader.customDir) → mergeFileLists
//     → 每个 XML 一次事务（DELETE param_mappings + UPSERT param_models + 批量 INSERT param_mappings）
//  2. 加载 StandardModelFile → standard_params 全量重写（仅 builtin 目录）
//
// 注意：本 P1-06 baseline 不实现 `{i}` 占位符校验、跨 XML 字段冲突合并（设计 §1.7.1 / §1.9）；
// 这两项放在 Phase 2 P2-02 ParamRegistry / Intersect。
type Loader struct {
	pool   *pgxpool.Pool
	cfg    appconfig.ParamModelLoaderConfig
	base   string
	logger *zap.Logger

	// T-0178 分层目录: customDir 是 host bind mount 持久化目录,custom XML 落地处。
	// 默认值 "param-mappings-custom" 由 NewLoader 注入。后续 appconfig 字段补齐后,
	// NewLoader 改从 cfg 读;现阶段保持 struct 字段以避免改跨包结构体。
	customDir       string
	customOverrides bool
}

// NewLoader 构造 ParamModel Loader。
// baseDir 来自 DictLoaderConfig.XMLBaseDir。空白 cfg 字段使用默认值。
func NewLoader(pool *pgxpool.Pool, cfg appconfig.ParamModelLoaderConfig, baseDir string, logger *zap.Logger) *Loader {
	if cfg.Directory == "" {
		cfg.Directory = "param-mappings"
	}
	if cfg.CustomDirectory == "" {
		// yaml 不写 custom_directory → 走默认值;允许用户改为非默认目录但极少需要
		cfg.CustomDirectory = CustomDirSubdir
	}
	if cfg.StandardModelFile == "" {
		cfg.StandardModelFile = "standard-model.xml"
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Loader{
		pool:            pool,
		cfg:             cfg,
		base:            baseDir,
		logger:          logger.Named(LoaderName),
		customDir:       cfg.CustomDirectory,          // T-0178 host 持久化目录(子路径)
		customOverrides: cfg.CustomOverridesEnabled(), // T-0178 决策 1: 默认 custom 胜出
	}
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

	builtinDir := filepath.Join(l.base, l.cfg.Directory)
	customSub := l.customDir
	if customSub == "" {
		// 直接构造 Loader 字面量(测试 / 未来 cfg 改造)绕过 NewLoader 时的兜底
		customSub = CustomDirSubdir
	}
	customDir := filepath.Join(l.base, customSub)

	reserved := []string{
		l.cfg.StandardModelFile,
		"products.xml",
		"product-name-routing.xml",
		"param-model-routing.xml",
	}

	// Pass 1: builtin + custom 双目录解析 + 同名合并(T-0178 §9.3)
	files, shadowedCustom, warnings, err := resolveLoaderFiles(
		builtinDir, customDir,
		l.cfg.ParamModelFiles,
		reserved,
		l.customOverrides,
	)
	if err != nil {
		return rep, fmt.Errorf("resolve param-model files: %w", err)
	}
	for _, w := range warnings {
		// custom dir 非 ENOENT 异常(权限/类型错误)走 WARN 不阻塞 builtin 加载
		l.logger.Warn("param-model file resolve warning", zap.String("detail", w))
	}
	// T-0178 R-NEW-T0178-8: customOverrides=false 时同名 custom 文件被 builtin 压制
	// 但 host 上文件还在,运维容易困惑"上传了为何不生效"。启动期 WARN 出清单,
	// 让 Loki/grep 一查即知;UI 端的提示由后续任务跟进。
	if !l.customOverrides && len(shadowedCustom) > 0 {
		l.logger.Warn(
			"custom param-model XML files are SUPPRESSED by builtin (custom_overrides_builtin=false)",
			zap.Strings("suppressed_files", shadowedCustom),
			zap.String("custom_dir", customDir),
			zap.String("hint", "either delete the shadowed custom files or set custom_overrides_builtin=true to let custom win"),
		)
	}

	for _, absPath := range files {
		rep.FilesScanned++
		rows, err := l.loadParamModelFile(ctx, absPath)
		if err != nil {
			rep.AddError(filepath.Base(absPath), "parse-or-persist", err)
			rep.FilesSkipped++
			l.logger.Error("param-model file failed",
				zap.String("file", absPath),
				zap.Error(err))
			continue
		}
		rep.FilesLoaded++
		rep.RowsAffected += rows
	}

	// Pass 2: standard-model.xml(单文件;仅 builtin 目录,custom 不允许覆盖 standard)
	stdPath := filepath.Join(builtinDir, l.cfg.StandardModelFile)
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

	// T-0178: loaded_from 写"<dir>/<file>" 相对 base 的 slash 路径
	// (源代码契约:source.go 的 ClassifySource / IsDeletable 据此判定 builtin/custom)。
	// 退化情况(base/path 跨盘等)由 resolveLoadedFrom 兜底为裸 basename。
	loadedFrom := resolveLoadedFrom(l.base, path)
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

	// §1.9 对账：param_mappings 已被本事务整体替换，需同步 discovered_param_mappings。
	// 除 DELETE / UPDATE 外，还要为已有子参数但缺失父 object 的 discovered 集合补插 object 行。
	// 失败仅记 WARN，不阻断 reload（DB 已落地）。
	delN, updN, insN, recErr := reconcileDiscoveredForModel(ctx, tx, modelID)
	if recErr != nil {
		l.logger.Warn("reconcile discovered_param_mappings failed",
			zap.String("model", doc.ParamModel),
			zap.Error(recErr))
	} else if delN > 0 || updN > 0 || insN > 0 {
		l.logger.Info("reconciled discovered_param_mappings",
			zap.String("model", doc.ParamModel),
			zap.Int64("deleted", delN),
			zap.Int64("updated", updN),
			zap.Int64("inserted", insN))
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

// reconcileDiscoveredForModel 实现设计 §1.9 对账：
//
//   - DELETE：discovered 中 private_path 在新 param_mappings 已不存在 → 删
//   - UPDATE：private_path 仍在，但 standard_path / 元属性变了 →
//     按 product.device_attrs_override 决定每个属性取设备值还是新默认值；
//     is_storable 始终跟随默认（不参与设备覆盖）。
//
// 范围限定：仅处理 products.param_model_id = modelID 引用的产品。
// 调用时机：每个 paramModel 重写 param_mappings 后、commit 前。
func reconcileDiscoveredForModel(ctx context.Context, tx pgx.Tx, modelID string) (deleted, updated, inserted int64, err error) {
	const delSQL = `
DELETE FROM discovered_param_mappings d
USING products pr
WHERE d.product_id = pr.id
  AND pr.param_model_id = $1
  AND NOT EXISTS (
    SELECT 1 FROM param_mappings pm
    WHERE pm.param_model_id = $1
      AND pm.private_path = d.private_path
  )`
	delTag, err := tx.Exec(ctx, delSQL, modelID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("reconcile DELETE: %w", err)
	}
	deleted = delTag.RowsAffected()

	const updSQL = `
UPDATE discovered_param_mappings d
SET standard_path = pm.standard_path,
    access = CASE WHEN COALESCE((pr.device_attrs_override->>'access')::boolean, false)
                  THEN d.access ELSE pm.access END,
    data_type = CASE WHEN COALESCE((pr.device_attrs_override->>'data_type')::boolean, false)
                     THEN d.data_type ELSE pm.data_type END,
    change_applies = CASE WHEN COALESCE((pr.device_attrs_override->>'change_applies')::boolean, false)
                          THEN d.change_applies ELSE pm.change_applies END,
    min_value = CASE WHEN COALESCE((pr.device_attrs_override->>'min_value')::boolean, false)
                     THEN d.min_value ELSE pm.min_value END,
    max_value = CASE WHEN COALESCE((pr.device_attrs_override->>'max_value')::boolean, false)
                     THEN d.max_value ELSE pm.max_value END,
    is_storable = pm.is_storable,
    is_supported = pm.is_supported,
    updated_at = now()
FROM param_mappings pm, products pr
WHERE d.product_id = pr.id
  AND pr.param_model_id = $1
  AND pm.param_model_id = $1
  AND d.private_path = pm.private_path
  AND (
    d.standard_path IS DISTINCT FROM pm.standard_path
    OR d.is_storable IS DISTINCT FROM pm.is_storable
    OR d.is_supported IS DISTINCT FROM pm.is_supported
    OR (NOT COALESCE((pr.device_attrs_override->>'access')::boolean, false)
        AND d.access IS DISTINCT FROM pm.access)
    OR (NOT COALESCE((pr.device_attrs_override->>'data_type')::boolean, false)
        AND d.data_type IS DISTINCT FROM pm.data_type)
    OR (NOT COALESCE((pr.device_attrs_override->>'change_applies')::boolean, false)
        AND d.change_applies IS DISTINCT FROM pm.change_applies)
    OR (NOT COALESCE((pr.device_attrs_override->>'min_value')::boolean, false)
        AND d.min_value IS DISTINCT FROM pm.min_value)
    OR (NOT COALESCE((pr.device_attrs_override->>'max_value')::boolean, false)
        AND d.max_value IS DISTINCT FROM pm.max_value)
  )`
	updTag, err := tx.Exec(ctx, updSQL, modelID)
	if err != nil {
		return deleted, 0, 0, fmt.Errorf("reconcile UPDATE: %w", err)
	}
	updated = updTag.RowsAffected()

	objectMappings, discoveredRows, err := loadDiscoveredObjectReconcileRows(ctx, tx, modelID)
	if err != nil {
		return deleted, updated, 0, err
	}
	pending := planDiscoveredObjectInserts(objectMappings, discoveredRows)
	inserted, err = batchInsertDiscoveredObjects(ctx, tx, pending)
	if err != nil {
		return deleted, updated, 0, err
	}
	return deleted, updated, inserted, nil
}

type discoveredMappingRef struct {
	ProductID       uuid.UUID
	SoftwareVersion string
	PrivatePath     string
}

type discoveredObjectInsert struct {
	ProductID       uuid.UUID
	SoftwareVersion string
	Mapping         ParamMapping
}

func loadDiscoveredObjectReconcileRows(ctx context.Context, tx pgx.Tx, modelID string) ([]ParamMapping, []discoveredMappingRef, error) {
	objectRows, err := tx.Query(ctx, `
SELECT standard_path, private_path, entry_type, access, data_type, change_applies,
       min_value, max_value, enum_values, enum_labels, mirror_with,
       is_storable, is_active, is_supported
FROM param_mappings
WHERE param_model_id = $1
  AND entry_type = 'object'`, modelID)
	if err != nil {
		return nil, nil, fmt.Errorf("reconcile INSERT load objects: %w", err)
	}
	defer objectRows.Close()

	objectMappings := make([]ParamMapping, 0)
	for objectRows.Next() {
		var mapping ParamMapping
		if err := objectRows.Scan(
			&mapping.StandardPath,
			&mapping.PrivatePath,
			&mapping.EntryType,
			&mapping.Access,
			&mapping.DataType,
			&mapping.ChangeApplies,
			&mapping.MinValue,
			&mapping.MaxValue,
			&mapping.EnumValues,
			&mapping.EnumLabels,
			&mapping.MirrorWith,
			&mapping.IsStorable,
			&mapping.IsActive,
			&mapping.IsSupported,
		); err != nil {
			return nil, nil, fmt.Errorf("reconcile INSERT scan object: %w", err)
		}
		objectMappings = append(objectMappings, mapping)
	}
	if err := objectRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("reconcile INSERT iterate objects: %w", err)
	}

	discoveredQueryRows, err := tx.Query(ctx, `
SELECT d.product_id, d.software_version, d.private_path
FROM discovered_param_mappings d
JOIN products pr
  ON pr.id = d.product_id
WHERE pr.param_model_id = $1`, modelID)
	if err != nil {
		return nil, nil, fmt.Errorf("reconcile INSERT load discovered: %w", err)
	}
	defer discoveredQueryRows.Close()

	discoveredRows := make([]discoveredMappingRef, 0)
	for discoveredQueryRows.Next() {
		var row discoveredMappingRef
		if err := discoveredQueryRows.Scan(&row.ProductID, &row.SoftwareVersion, &row.PrivatePath); err != nil {
			return nil, nil, fmt.Errorf("reconcile INSERT scan discovered: %w", err)
		}
		discoveredRows = append(discoveredRows, row)
	}
	if err := discoveredQueryRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("reconcile INSERT iterate discovered: %w", err)
	}

	return objectMappings, discoveredRows, nil
}

func planDiscoveredObjectInserts(objectMappings []ParamMapping, discoveredRows []discoveredMappingRef) []discoveredObjectInsert {
	if len(objectMappings) == 0 || len(discoveredRows) == 0 {
		return nil
	}

	existing := make(map[string]struct{}, len(discoveredRows))
	for _, row := range discoveredRows {
		existing[discoveredInsertKey(row.ProductID, row.SoftwareVersion, row.PrivatePath)] = struct{}{}
	}

	pending := make(map[string]discoveredObjectInsert)
	for _, row := range discoveredRows {
		normalized := normalizeInstancePath(row.PrivatePath)
		for _, mapping := range objectMappings {
			if !strings.HasPrefix(normalized, mapping.PrivatePath) {
				continue
			}
			key := discoveredInsertKey(row.ProductID, row.SoftwareVersion, mapping.PrivatePath)
			if _, ok := existing[key]; ok {
				continue
			}
			if _, ok := pending[key]; ok {
				continue
			}
			pending[key] = discoveredObjectInsert{
				ProductID:       row.ProductID,
				SoftwareVersion: row.SoftwareVersion,
				Mapping:         mapping,
			}
		}
	}

	if len(pending) == 0 {
		return nil
	}

	keys := make([]string, 0, len(pending))
	for key := range pending {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]discoveredObjectInsert, 0, len(keys))
	for _, key := range keys {
		out = append(out, pending[key])
	}
	return out
}

func discoveredInsertKey(productID uuid.UUID, softwareVersion, privatePath string) string {
	return productID.String() + "|" + softwareVersion + "|" + privatePath
}

func batchInsertDiscoveredObjects(ctx context.Context, tx pgx.Tx, pending []discoveredObjectInsert) (int64, error) {
	if len(pending) == 0 {
		return 0, nil
	}

	const chunkSize = 200
	var total int64
	for i := 0; i < len(pending); i += chunkSize {
		end := i + chunkSize
		if end > len(pending) {
			end = len(pending)
		}

		ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Insert("discovered_param_mappings").Columns(
			"product_id", "software_version", "standard_path", "private_path",
			"entry_type", "access", "data_type", "change_applies",
			"min_value", "max_value", "enum_values", "enum_labels", "mirror_with",
			"is_storable", "is_active", "is_supported",
		)
		for _, ins := range pending[i:end] {
			mapping := ins.Mapping
			ib = ib.Values(
				ins.ProductID,
				ins.SoftwareVersion,
				mapping.StandardPath,
				mapping.PrivatePath,
				mapping.EntryType,
				mapping.Access,
				mapping.DataType,
				mapping.ChangeApplies,
				mapping.MinValue,
				mapping.MaxValue,
				mapping.EnumValues,
				mapping.EnumLabels,
				mapping.MirrorWith,
				mapping.IsStorable,
				mapping.IsActive,
				mapping.IsSupported,
			)
		}
		sqlStr, args, err := ib.ToSql()
		if err != nil {
			return total, fmt.Errorf("reconcile INSERT build SQL: %w", err)
		}
		tag, err := tx.Exec(ctx, sqlStr, args...)
		if err != nil {
			return total, fmt.Errorf("reconcile INSERT exec: %w", err)
		}
		total += tag.RowsAffected()
	}
	return total, nil
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
			"min_value", "max_value",
			"enum_values", "enum_labels", // T-0158
			"mirror_with", // T-0159
			"is_storable", "is_active", "is_supported",
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
				nullIfEmpty(e.EnumValues), // T-0158: 枚举值 CSV
				nullIfEmpty(e.EnumLabels), // T-0158: 枚举标签 CSV
				nullIfEmpty(e.MirrorWith), // T-0159: 交叉镜像目标 standardPath
				!strings.EqualFold(strings.TrimSpace(e.Store), "false"), // 缺省 / 任意非 "false" → true
				true, // is_active
				!strings.EqualFold(strings.TrimSpace(e.Supported), "false"), // T-0103: 缺省 / 任意非 "false" → true
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

	// TRUNCATE 不行：mml_command_sub_fields.standard_path_id 反向 FK 引用
	// standard_params（migration 000113），即使 ON DELETE RESTRICT 也会被
	// CLAUDE.md §5.5.9 拒绝。改为 ON CONFLICT UPSERT — 同 standard_path 行
	// 元属性可能因 XML 演化更新；不存在的行新增；不再清空既有 sub_field
	// 引用的目标行（FK 完整性保留）。删除被 XML 移除的 path 留给 admin UI
	// 手工处理（T-0132 Catalog 管理已支持）。
	// 旧逻辑 (TRUNCATE) 移除，新逻辑由下方 batchInsertStandardEntries 的
	// ON CONFLICT 子句承接。

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
		// UPSERT 替代 TRUNCATE 路径（详见 loadStandardModel 注释）。
		// standard_params.standard_path UNIQUE → ON CONFLICT 触发更新元属性。
		sqlStr += ` ON CONFLICT (standard_path) DO UPDATE SET
            entry_type     = EXCLUDED.entry_type,
            access         = EXCLUDED.access,
            data_type      = EXCLUDED.data_type,
            change_applies = EXCLUDED.change_applies,
            min_value      = EXCLUDED.min_value,
            max_value      = EXCLUDED.max_value,
            updated_at     = NOW()`
		if _, err := tx.Exec(ctx, sqlStr, args...); err != nil {
			return total, fmt.Errorf("batch upsert standard_params (%s, batch %d-%d): %w", entryType, i, end, err)
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

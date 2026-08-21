package parammodel

import (
	"context"
	"encoding/xml"
	"errors"
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
	"github.com/omcgo/omcgo/internal/core/reliability"
)

// LoaderName 是 dictloader.Registry 中的注册名（也用于日志域）。
const LoaderName = "param-model"

// Loader 实现 dictloader.Loader（T-0098 P1-06）。
//
// 加载语义（设计 §1.7；三库 XML 导入重构 2026-06-04 单目录）：
//  1. 扫描单目录 cfg.Directory（param-mappings/，builtin + custom XML 同住，跳过 sidecar）
//     → 每个 XML 一次事务（DELETE param_mappings + UPSERT param_models + 批量 INSERT param_mappings）
//  2. 加载 StandardModelFile → standard_params 全量重写
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
	return &Loader{
		pool:   pool,
		cfg:    cfg,
		base:   baseDir,
		logger: logger.Named(LoaderName),
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

	reserved := []string{
		l.cfg.StandardModelFile,
		"products.xml",
		"product-name-routing.xml",
		"param-model-routing.xml",
	}

	// Pass 1: 单目录扫描(builtin + custom XML 同住,sidecar 文件已在 scanXMLFiles 跳过)
	files, err := resolveLoaderFiles(builtinDir, l.cfg.ParamModelFiles, reserved)
	if err != nil {
		return rep, fmt.Errorf("resolve param-model files: %w", err)
	}

	loadedModels := make(map[string]struct{}, len(files))
	for _, absPath := range files {
		rep.FilesScanned++
		rows, modelName, err := l.loadParamModelFile(ctx, absPath)
		if err != nil {
			rep.AddError(filepath.Base(absPath), "parse-or-persist", err)
			rep.FilesSkipped++
			l.logger.Error("param-model file failed",
				zap.String("file", absPath),
				zap.Error(err))
			continue
		}
		if modelName != "" {
			loadedModels[modelName] = struct{}{}
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
	pruned, err := l.pruneRetiredBuiltinParamModels(ctx, loadedModels)
	if err != nil {
		return rep, err
	}
	rep.RowsAffected += int(pruned)
	return rep, nil
}

// loadParamModelFile 单文件加载：upsert param_models + 重写 param_mappings。
func (l *Loader) loadParamModelFile(ctx context.Context, path string) (int, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, "", fmt.Errorf("read %s: %w", path, err)
	}
	var doc xmlParameterModel
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return 0, "", fmt.Errorf("xml unmarshal %s: %w", path, err)
	}
	if doc.ParamModel == "" {
		return 0, "", fmt.Errorf("paramModel attribute empty in %s", path)
	}

	// loaded_from 写 "param-mappings/<file>" 相对 base 的 slash 路径。
	// 来源(builtin/custom)由同目录 sidecar(X.xml.custom)判定,见 source.go。
	// 退化情况(base/path 跨盘等)由 resolveLoadedFrom 兜底为裸 basename。
	loadedFrom := resolveLoadedFrom(l.base, path)
	totalObjects := len(doc.Objects)
	totalParams := len(doc.Params)
	// XML totalEntries 可能陈旧，且去重及保留 custom 覆盖都会改变实际落库数量。
	// 事务末尾会从 param_mappings 重算；此处只给新行一个不依赖 XML 元数据的初值。
	totalEntries := totalObjects + totalParams

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, "", fmt.Errorf("begin tx: %w", err)
	}
	defer reliability.RollbackTx(ctx, tx, l.logger, "loadParamModelFile")

	// UPSERT param_models（按 name 唯一）
	const upsertModel = `INSERT INTO param_models (name, total_entries, total_objects, total_params, is_active, loaded_from)
	VALUES ($1, $2, $3, $4, TRUE, $5)
	ON CONFLICT (name) DO UPDATE
	SET total_entries = EXCLUDED.total_entries,
	    total_objects = EXCLUDED.total_objects,
	    total_params  = EXCLUDED.total_params,
	    loaded_from   = EXCLUDED.loaded_from
	RETURNING id`
	var modelID string
	if err := tx.QueryRow(ctx, upsertModel, doc.ParamModel, totalEntries, totalObjects, totalParams, loadedFrom).Scan(&modelID); err != nil {
		return 0, "", fmt.Errorf("upsert param_models %q: %w", doc.ParamModel, err)
	}

	// 重写 param_mappings：先按 model 删旧 builtin 行，再批量插入。
	// T-PMSRC：只删 source='builtin'，保留管理员经 UI 维护的 source='custom' 覆盖项,
	// 实现"重新加载 XML 时不覆盖自定义数据"。custom 与 builtin 同 private_path 时,
	// batchInsertMappings 的 ON CONFLICT DO NOTHING 让 builtin 跳过 → custom 胜出。
	if _, err := tx.Exec(ctx, "DELETE FROM param_mappings WHERE param_model_id = $1 AND source = 'builtin'", modelID); err != nil {
		return 0, "", fmt.Errorf("clear param_mappings for %q: %w", doc.ParamModel, err)
	}

	// 2026-05-29 防御:同 XML 文件内 private_path 重复(撞新约束
	// uniq_param_mappings_model_private,见 migration 000219)。批量 INSERT 任意
	// 一条撞约束整批回滚 → paramModel reload 失败 → dictload module init failed
	// → app 拒启动。
	// 注:standard_path 允许多 private_path 别名(BM.xml 的 X_COM_EUTRAULEarfcn →
	// EUTRACarrierARFCN 就是有意为之,Translator 接受 last-wins);约束改 private_path
	// 后,业务上"私有路径在 paramModel 内必须唯一"也是真要求。
	// 策略:first-seen 胜出 + WARN 日志,跨 objects/params 两批次共享去重集。
	dedupSet := make(map[string]struct{}, len(doc.Objects)+len(doc.Params))
	dedupObjects := dedupByPrivatePath(doc.Objects, dedupSet, path, "object", l.logger)
	dedupParams := dedupByPrivatePath(doc.Params, dedupSet, path, "parameter", l.logger)

	rows := 0
	if len(dedupObjects) > 0 {
		if n, err := batchInsertMappings(ctx, tx, modelID, dedupObjects, "object"); err != nil {
			return 0, "", err
		} else {
			rows += n
		}
	}
	if len(dedupParams) > 0 {
		if n, err := batchInsertMappings(ctx, tx, modelID, dedupParams, "parameter"); err != nil {
			return 0, "", err
		} else {
			rows += n
		}
	}

	// param_mappings 是统计唯一真值源。重载会保留 custom 行，且 XML 内部可能去重，
	// 因此必须在同一事务内按最终有效行重算，不能信任 XML 的 totalEntries。
	const recountModel = `UPDATE param_models pm
	SET total_entries = stats.total_entries,
	    total_objects = stats.total_objects,
	    total_params  = stats.total_params
	FROM (
	    SELECT COUNT(*)::int AS total_entries,
	           COUNT(*) FILTER (WHERE entry_type = 'object')::int AS total_objects,
	           COUNT(*) FILTER (WHERE entry_type = 'parameter')::int AS total_params
	      FROM param_mappings
	     WHERE param_model_id = $1 AND is_active = TRUE
	) stats
	WHERE pm.id = $1
	RETURNING pm.total_entries, pm.total_objects, pm.total_params`
	if err := tx.QueryRow(ctx, recountModel, modelID).Scan(&totalEntries, &totalObjects, &totalParams); err != nil {
		return 0, "", fmt.Errorf("recount param_model %q: %w", doc.ParamModel, err)
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
		return 0, "", fmt.Errorf("commit tx: %w", err)
	}

	rows++ // count the param_models row
	l.logger.Info("param-model loaded",
		zap.String("model", doc.ParamModel),
		zap.Int("objects", totalObjects),
		zap.Int("params", totalParams),
		zap.String("file", loadedFrom))
	return rows, doc.ParamModel, nil
}

func (l *Loader) pruneRetiredBuiltinParamModels(ctx context.Context, loadedModels map[string]struct{}) (int64, error) {
	var total int64
	for _, name := range []string{"UPS"} {
		if _, loaded := loadedModels[name]; loaded {
			continue
		}
		pruned, err := l.pruneRetiredBuiltinParamModel(ctx, name)
		if err != nil {
			return total, err
		}
		total += pruned
	}
	return total, nil
}

func (l *Loader) pruneRetiredBuiltinParamModel(ctx context.Context, name string) (int64, error) {
	expectedLoadedFrom := l.cfg.Directory + "/" + name + ".xml"

	var id uuid.UUID
	var loadedFrom string
	err := l.pool.QueryRow(ctx,
		`SELECT id, COALESCE(loaded_from, '') FROM param_models WHERE name = $1`,
		name,
	).Scan(&id, &loadedFrom)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("lookup retired param_model %q: %w", name, err)
	}
	if loadedFrom != expectedLoadedFrom || IsCustom(l.base, loadedFrom) {
		return 0, nil
	}

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin retired param_model prune: %w", err)
	}
	defer reliability.RollbackTx(ctx, tx, l.logger, "pruneRetiredBuiltinParamModel")

	if _, err := tx.Exec(ctx, `
UPDATE devices
   SET param_model_id = NULL
 WHERE param_model_id = $1
   AND COALESCE(product_class, '') LIKE 'UPS%'`, id); err != nil {
		return 0, fmt.Errorf("clear retired UPS device param_model_id: %w", err)
	}

	tag, err := tx.Exec(ctx, `DELETE FROM param_models WHERE id = $1`, id)
	if err != nil {
		return 0, fmt.Errorf("delete retired param_model %q: %w", name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit retired param_model prune: %w", err)
	}
	if tag.RowsAffected() > 0 {
		l.logger.Info("retired builtin param-model pruned",
			zap.String("param_model", name),
			zap.String("loaded_from", loadedFrom))
	}
	return tag.RowsAffected(), nil
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
    default_value = pm.default_value,
    validation_pattern = pm.validation_pattern,
    enum_values = pm.enum_values,
    enum_labels = pm.enum_labels,
    mirror_with = pm.mirror_with,
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
    OR d.default_value IS DISTINCT FROM pm.default_value
    OR d.validation_pattern IS DISTINCT FROM pm.validation_pattern
    OR d.enum_values IS DISTINCT FROM pm.enum_values
    OR d.enum_labels IS DISTINCT FROM pm.enum_labels
    OR d.mirror_with IS DISTINCT FROM pm.mirror_with
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
       min_value, max_value, default_value, validation_pattern,
       enum_values, enum_labels, mirror_with,
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
		// 2026-05-29 bug fix:`access` / `data_type` / `change_applies` 三列在 DB 是
		// NULL-able(见 information_schema)。entry_type='object' 行的 data_type 全部
		// 为 NULL(对象本无数据类型);而 ParamMapping 的对应字段是 plain string,
		// 直接 Scan 会 "cannot scan NULL into *string" 报错,导致整个 reload XML 终止。
		// 仿 pg_handler_repo.go::getMappingByID 模式:用 *string 收 + strDeref 转回。
		var mapping ParamMapping
		var access, dataType, changeApplies *string
		if err := objectRows.Scan(
			&mapping.StandardPath,
			&mapping.PrivatePath,
			&mapping.EntryType,
			&access,
			&dataType,
			&changeApplies,
			&mapping.MinValue,
			&mapping.MaxValue,
			&mapping.DefaultValue,
			&mapping.ValidationPattern,
			&mapping.EnumValues,
			&mapping.EnumLabels,
			&mapping.MirrorWith,
			&mapping.IsStorable,
			&mapping.IsActive,
			&mapping.IsSupported,
		); err != nil {
			return nil, nil, fmt.Errorf("reconcile INSERT scan object: %w", err)
		}
		mapping.Access = strDeref(access)
		mapping.DataType = strDeref(dataType)
		mapping.ChangeApplies = strDeref(changeApplies)
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
			"min_value", "max_value", "default_value", "validation_pattern",
			"enum_values", "enum_labels", "mirror_with",
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
				mapping.DefaultValue,
				mapping.ValidationPattern,
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

// dedupByPrivatePath 同 XML 文件内按 private_path(xmlParamEntry.Name)去重
// (first-seen 胜出),防止 batch INSERT 撞 uniq_param_mappings_model_private
// 约束(migration 000219)让整批回滚。
// 注:standard_path 不参与去重 — 允许多个 private 别名映射到同一 standard
// (BM.xml 的 X_COM_EUTRAULEarfcn → EUTRACarrierARFCN 即此设计)。
// 同一 private_path 在 objects 与 params 之间也会去重(共享 seen set)。
// 重复行写 WARN 日志,包含文件路径 / entryType / Name / standardPath,运维可对照修 XML。
func dedupByPrivatePath(entries []xmlParamEntry, seen map[string]struct{}, path, entryType string, logger *zap.Logger) []xmlParamEntry {
	out := make([]xmlParamEntry, 0, len(entries))
	for _, e := range entries {
		// private_path = e.Name(xml `name` 属性);唯一标识 paramModel 内的物理路径
		priv := e.Name
		if priv == "" {
			// 极端兜底:name 缺失时退到 standardPath,跟 batchInsertMappings 的反向 fallback 同
			priv = e.StandardPath
		}
		if _, dup := seen[priv]; dup {
			logger.Warn("duplicate privatePath in XML, skipping (first-seen kept)",
				zap.String("file", path),
				zap.String("entry_type", entryType),
				zap.String("private_path", priv),
				zap.String("skipped_standard_path", e.StandardPath))
			continue
		}
		seen[priv] = struct{}{}
		out = append(out, e)
	}
	return out
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
			"default_value", "validation_pattern",
			"enum_values", "enum_labels", // T-0158
			"mirror_with", // T-0159
			"is_storable", "is_active", "is_supported",
			"source", // T-PMSRC：XML 加载行恒为 builtin
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
				nullIfEmpty(e.DefaultValue),
				nullIfEmpty(e.ValidationPattern),
				nullIfEmpty(normalizeEnumCSV(e.EnumValues)),             // T-0158: 枚举值 CSV
				nullIfEmpty(normalizeEnumCSV(e.EnumLabels)),             // T-0158: 枚举标签 CSV
				nullIfEmpty(e.MirrorWith),                               // T-0159: 交叉镜像目标 standardPath
				!strings.EqualFold(strings.TrimSpace(e.Store), "false"), // 缺省 / 任意非 "false" → true
				true, // is_active
				!strings.EqualFold(strings.TrimSpace(e.Supported), "false"), // T-0103: 缺省 / 任意非 "false" → true
				"builtin", // T-PMSRC source
			)
		}
		// T-PMSRC：同 private_path 已被 custom 覆盖项占用时跳过该 builtin 行(custom 胜出),
		// 避免撞 uniq_param_mappings_model_private 导致整批回滚 → reload 失败 → app 拒启动。
		ib = ib.Suffix("ON CONFLICT (param_model_id, private_path) DO NOTHING")
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

func normalizeEnumCSV(value string) string {
	parts := strings.Split(strings.ReplaceAll(value, "，", ","), ",")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			normalized = append(normalized, part)
		}
	}
	return strings.Join(normalized, ",")
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
	defer reliability.RollbackTx(ctx, tx, l.logger, "loadStandardModelFile")

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
		// 跳过 sidecar 标记文件(X.xml.custom)。其 ext 是 .custom 而非 .xml,
		// 下面的 EqualFold 过滤已能排除,这里显式 continue 以求清晰 + 安全。
		if strings.HasSuffix(name, CustomMarkerSuffix) {
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

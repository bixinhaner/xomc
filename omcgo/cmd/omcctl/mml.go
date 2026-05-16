package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/omcgo/omcgo/internal/config/parammodel/mmlstandardloader"
)

// contextWithTimeout 返回 (context.Context, cancel) 并设定超时。
// omcctl 一次性命令使用，外部捕获 SIGINT 不重要（cobra 会终止进程）。
func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// openPgPool 打开 pgxpool（小池子，omcctl 单次命令用 4 个连接足够）。
func openPgPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 4
	cfg.MinConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// newMMLCmd 注册 `omcctl mml` 父命令 + 子命令。
//
// 目前仅含一个子命令：`import-standard-xml`，把 standard-model.xml 翻译为
// migrations/seed/ 下的 SQL 文件（一次性 DB 导入；运行期 mmlstandardloader
// 启动期注册由 T-0123-P0 deps.go 同步下线）。
//
// 设计依据：docs/design/mml-restore-old-interaction-plan-20260514.md v2 §M.3
func newMMLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mml",
		Short: "MML catalog management commands",
		Long:  "MML 命令字典工具：standard-model.xml → seed SQL 一次性导入",
	}
	cmd.AddCommand(newMMLImportCmd())
	cmd.AddCommand(newMMLMigrateDeviceParamsCmd())
	return cmd
}

// newMMLMigrateDeviceParamsCmd 注册 `omcctl mml migrate-device-params` 子命令。
//
// 整改方案 Stage 2（用户决策 2026-05-16）：device_parameters.parameter_path
// 历史值可能是 privatePath（旧 Inform 直接入库），新写入都是 standardPath
// （rpc_response_subscriber 翻译后入库 + Path B 同步）。本命令把历史 private
// 行翻译为 standard，按 (device_id, parameter_path) 行级 UPDATE。
//
// 路径：device.product_class → products.product_class regex 匹配 → product.id
//      → param_mappings.private_path == dp.parameter_path → standard_path
//
// 翻译失败的行（未匹配到 product / 未在 param_mappings 中找到 private_path）
// 保持不变，由运维通过 admin UI 补 discovered_param_mappings 后重跑命令。
//
// Flags：
//   --dsn       连接串（必填）
//   --dry-run   仅扫描+报告，不写入（默认）
//   --apply     真执行 UPDATE
//   --batch     批大小（默认 500，超大表分批避免长事务）
func newMMLMigrateDeviceParamsCmd() *cobra.Command {
	var (
		dsn     string
		dryRun  bool
		apply   bool
		batch   int
	)
	c := &cobra.Command{
		Use:   "migrate-device-params",
		Short: "Translate legacy privatePath rows in device_parameters to standardPath",
		Long: `把 device_parameters 表中历史 privatePath 行翻译为 standardPath。

读 products + param_mappings 现有数据，按 (device.product_class regex 匹配
product) + (private_path 精确匹配) 重写 parameter_path。

默认 --dry-run 只报告会改多少行，--apply 才真改。--batch 控制每次 UPDATE
的行数上限（默认 500）。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMMLMigrateDeviceParams(dsn, dryRun, apply, batch)
		},
	}
	c.Flags().StringVar(&dsn, "dsn", "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable", "PostgreSQL DSN")
	c.Flags().BoolVar(&dryRun, "dry-run", true, "Only count affected rows; default true")
	c.Flags().BoolVar(&apply, "apply", false, "Actually execute UPDATE (overrides --dry-run)")
	c.Flags().IntVar(&batch, "batch", 500, "Max rows per UPDATE statement")
	return c
}

func runMMLMigrateDeviceParams(dsn string, dryRun, apply bool, batch int) error {
	if apply {
		dryRun = false
	}
	ctx, cancel := contextWithTimeout(5 * time.Minute)
	defer cancel()

	pool, err := openPgPool(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect pg: %w", err)
	}
	defer pool.Close()

	// 1. 扫描可翻译候选数
	// 注：products 用 product_class_patterns 表存正则（pattern 一对多）。
	const candidateSQL = `
SELECT COUNT(*) FROM device_parameters dp
JOIN devices d ON d.id = dp.device_id
JOIN product_class_patterns pcp ON d.product_class ~ pcp.product_class AND pcp.is_active
JOIN products p ON p.id = pcp.product_id
JOIN param_mappings pm ON pm.param_model_id = p.param_model_id
WHERE pm.private_path = dp.parameter_path
  AND dp.parameter_path <> pm.standard_path`
	var candidates int64
	if err := pool.QueryRow(ctx, candidateSQL).Scan(&candidates); err != nil {
		return fmt.Errorf("count candidates: %w", err)
	}
	fmt.Printf("Found %d device_parameters rows translatable (privatePath → standardPath)\n", candidates)

	if dryRun {
		fmt.Println("Dry-run mode; no rows modified. Pass --apply to execute.")
		return nil
	}

	// 2. 真执行 UPDATE（分批，避免长事务锁太多行）
	const updateSQL = `
UPDATE device_parameters dp
SET parameter_path = sub.standard_path,
    last_updated_at = NOW()
FROM (
    SELECT dp.device_id, dp.parameter_path AS old_path, pm.standard_path
    FROM device_parameters dp
    JOIN devices d ON d.id = dp.device_id
    JOIN product_class_patterns pcp ON d.product_class ~ pcp.product_class AND pcp.is_active
    JOIN products p ON p.id = pcp.product_id
    JOIN param_mappings pm ON pm.param_model_id = p.param_model_id
    WHERE pm.private_path = dp.parameter_path
      AND dp.parameter_path <> pm.standard_path
    LIMIT $1
) sub
WHERE dp.device_id = sub.device_id
  AND dp.parameter_path = sub.old_path`
	totalUpdated := int64(0)
	for {
		ct, err := pool.Exec(ctx, updateSQL, batch)
		if err != nil {
			return fmt.Errorf("update batch: %w", err)
		}
		n := ct.RowsAffected()
		totalUpdated += n
		fmt.Printf("  batch updated %d rows (cumulative %d)\n", n, totalUpdated)
		if n == 0 {
			break
		}
	}
	fmt.Printf("Done. Total %d rows translated to standardPath.\n", totalUpdated)
	return nil
}

func newMMLImportCmd() *cobra.Command {
	var (
		xmlPath     string
		outPath     string
		versionCode string
		dryRun      bool
	)
	c := &cobra.Command{
		Use:   "import-standard-xml",
		Short: "Convert standard-model.xml to migrations/seed SQL (T-0123-P0)",
		Long: `把 TR-069 standard-model.xml 翻译为 migrations/seed/ 下的 INSERT SQL，
一次性导入 mml_params 表（catalog_protected=true / source='standard'）。

- 使用既有 mml_params.uq_param_version_path UNIQUE (param_version, tr069_path) 作 ON CONFLICT 锚点
- ON CONFLICT DO UPDATE WHERE catalog_protected=true（仅刷新 standard 行；admin 改过的行保留）
- 不写 is_writable 列（GENERATED 派生，migration 000095 Q1=B 决议）
- name_i18n: 末段 PascalCase 拆词 + zh 词典翻译
- constraint_text_i18n: 综合 type/min/max 渲染双语提示`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMMLImport(xmlPath, outPath, versionCode, dryRun)
		},
	}
	c.Flags().StringVar(&xmlPath, "xml", "omcgo/data/param-mappings/standard-model.xml", "Path to standard-model.xml input")
	c.Flags().StringVar(&outPath, "out", "omcgo/migrations/seed/000096_mml_standard_params_import.sql", "Output seed SQL file path")
	c.Flags().StringVar(&versionCode, "version-code", "STANDARD", "mml_param_versions.version_code")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "Parse XML and report counts only; do not write SQL")
	return c
}

// runMMLImport 主流程。
func runMMLImport(xmlPath, outPath, versionCode string, dryRun bool) error {
	params, objects, err := mmlstandardloader.ParseStandardXMLFile(xmlPath)
	if err != nil {
		return fmt.Errorf("parse standard-model.xml: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[omcctl mml import] parsed %d params + %d objects from %s\n",
		len(params), len(objects), xmlPath)

	rows, err := buildImportRows(params, objects, versionCode)
	if err != nil {
		return fmt.Errorf("build rows: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[omcctl mml import] generated %d rows (after dedup) → %s\n", len(rows), outPath)

	if dryRun {
		fmt.Fprintf(os.Stderr, "[omcctl mml import] --dry-run set; not writing %s\n", outPath)
		return nil
	}

	sqlBytes, err := renderImportSQL(rows, versionCode)
	if err != nil {
		return fmt.Errorf("render SQL: %w", err)
	}
	if err := os.WriteFile(outPath, sqlBytes, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	fmt.Fprintf(os.Stderr, "[omcctl mml import] wrote %d bytes to %s\n", len(sqlBytes), outPath)
	return nil
}

// importRow 单条 INSERT 行的中间数据。
type importRow struct {
	ParamCode          string
	Tr069Path          string
	ValueType          string
	AccessType         string
	IsObject           bool
	SupportsAdd        bool
	SupportsDelete    bool
	ChangeApplies      string
	NameI18n           map[string]string
	ConstraintTextI18n map[string]string
}

// buildImportRows 合并 params + objects → []importRow（按 tr069_path 升序）。
//
// 冲突处理：
//   - hash8 碰撞兜底：检测到同 param_code 但不同 tr069_path 则升 12 字符 hash 重试
//   - 同 tr069_path 重复：以 params 优先（XML 中正常无重复）
func buildImportRows(params []mmlstandardloader.ParamSpec, objects []mmlstandardloader.ObjectSpec, versionCode string) ([]importRow, error) {
	seenPath := make(map[string]struct{}, len(params)+len(objects))
	codeUsed := make(map[string]string, len(params)+len(objects)) // code → first path using it
	rows := make([]importRow, 0, len(params)+len(objects))

	addRow := func(p, valueType, access, change string, isObject bool, min, max *int64, maxLen *int) {
		if _, dup := seenPath[p]; dup {
			return
		}
		seenPath[p] = struct{}{}

		code := buildParamCode(p, codeUsed)
		codeUsed[code] = p

		access = normalizeAccess(access)
		change = normalizeChangeApplies(change)
		supportsAdd := isObject && access == mmlstandardloader.AccessReadWrite
		supportsDelete := supportsAdd

		rows = append(rows, importRow{
			ParamCode:          code,
			Tr069Path:          p,
			ValueType:          mapValueTypeToDB(valueType), // CHECK 白名单：string/int/unsignedInt/boolean/...
			AccessType:         access,
			IsObject:           isObject,
			SupportsAdd:        supportsAdd,
			SupportsDelete:     supportsDelete,
			ChangeApplies:      change,
			NameI18n:           renderNameI18n(p),
			ConstraintTextI18n: renderConstraintTextI18n(valueType, min, max, maxLen),
		})
	}

	for _, p := range params {
		addRow(p.StandardPath, p.Type, p.Access, p.ChangeApplies, false, p.Min, p.Max, p.MaxLen)
	}
	for _, o := range objects {
		addRow(o.StandardPath, "OBJECT", o.Access, o.ChangeApplies, true, nil, nil, nil)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Tr069Path < rows[j].Tr069Path
	})
	return rows, nil
}

// buildParamCode 派生 param_code。
//
// 规则：UPPER(path 末段) + "_" + hash8(fullPath)。
// 碰撞兜底：如果 hash8 与已有 code 撞，升级到 12 字符 hash。
func buildParamCode(p string, codeUsed map[string]string) string {
	leaf := lastSegment(p)
	leaf = strings.ToUpper(sanitizeCodePart(leaf))
	if leaf == "" {
		leaf = "PATH"
	}

	full := sha256.Sum256([]byte(p))
	short := hex.EncodeToString(full[:])[:8]
	code := leaf + "_" + short

	if existPath, conflict := codeUsed[code]; conflict && existPath != p {
		// 升 12 字符 hash 兜底（F2 自审）
		code = leaf + "_" + hex.EncodeToString(full[:])[:12]
	}
	return code
}

func lastSegment(p string) string {
	t := strings.TrimSuffix(p, ".")
	if i := strings.LastIndex(t, "."); i >= 0 {
		return t[i+1:]
	}
	return t
}

// sanitizeCodePart 把 path 末段清洗为 [A-Z0-9_] 集合内字符。
func sanitizeCodePart(s string) string {
	s = strings.ReplaceAll(s, "{i}", "")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z',
			r >= 'a' && r <= 'z',
			r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// mapValueTypeToDB 把 XML 大写 type 翻译为 mml_params.value_type CHECK 白名单中的小写值。
// CHECK 约束（migration 000022）：'string' / 'enum' / 'unsignedInt' / 'unsignedIntList' /
// 'stringList' / 'boolean' / 'uniqueInt' / 'int'。
//
// 未知类型 fail-safe 归 'string'，避免 INSERT 触发 CHECK violation。
// OBJECT 路径（object 容器）也归 'string'，由 is_object=true 列区分。
func mapValueTypeToDB(xmlType string) string {
	switch strings.ToUpper(xmlType) {
	case "INT":
		return "int"
	case "U_INT", "UINT", "UNSIGNEDINT":
		return "unsignedInt"
	case "BOOLEAN", "BOOL":
		return "boolean"
	default:
		// STRING / DATE_TIME / OBJECT / 未知 → string 兜底
		return "string"
	}
}

func normalizeAccess(a string) string {
	switch strings.ToUpper(a) {
	case "READ_WRITE", "WRITE_ONLY":
		return strings.ToUpper(a)
	default:
		return "READ_ONLY"
	}
}

func normalizeChangeApplies(c string) string {
	switch strings.EqualFold(c, "OnReboot") {
	case true:
		return "OnReboot"
	default:
		return "Immediate"
	}
}

// renderNameI18n 根据 path 末段生成 {"en-US":"...","zh-CN":"..."}。
// - en-US：末段 PascalCase 拆词
// - zh-CN：逐词查 zhDictionary，全部命中才返回；否则留空（前端 "🟡 需翻译" badge 兜底）
func renderNameI18n(p string) map[string]string {
	leaf := lastSegment(p)
	leaf = strings.ReplaceAll(leaf, "{i}", "")
	leaf = strings.TrimSpace(leaf)
	if leaf == "" {
		return map[string]string{}
	}
	en := splitPascalCase(leaf)
	zh := translateZh(en)
	out := map[string]string{"en-US": en}
	if zh != "" {
		out["zh-CN"] = zh
	}
	return out
}

// splitPascalCase 把 "SoftwareVersion" → "Software Version"；"X_COM_MODULE_TYPE" → "X COM Module Type"。
func splitPascalCase(s string) string {
	// 先把下划线转空格
	s = strings.ReplaceAll(s, "_", " ")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// 前一个字符是小写或数字 → 插空格（避免 "SoftwareVersion"→"S Oftware Version"）
			prev := rune(s[i-1])
			if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') {
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
	}
	out := b.String()
	// 合并连续空格
	for strings.Contains(out, "  ") {
		out = strings.ReplaceAll(out, "  ", " ")
	}
	return strings.TrimSpace(out)
}

// renderConstraintTextI18n 综合 type/min/max/maxLen 渲染双语提示。
func renderConstraintTextI18n(valueType string, min, max *int64, maxLen *int) map[string]string {
	t := strings.ToUpper(valueType)
	switch t {
	case "INT", "U_INT", "UNSIGNEDINT":
		if min != nil && max != nil {
			return map[string]string{
				"en-US": fmt.Sprintf("Integer, range: %d-%d", *min, *max),
				"zh-CN": fmt.Sprintf("整数，范围：%d-%d", *min, *max),
			}
		}
		if max != nil {
			return map[string]string{
				"en-US": fmt.Sprintf("Integer, max: %d", *max),
				"zh-CN": fmt.Sprintf("整数，上限：%d", *max),
			}
		}
		return map[string]string{
			"en-US": "Integer",
			"zh-CN": "整数",
		}
	case "BOOLEAN":
		return map[string]string{
			"en-US": "Boolean (true/false)",
			"zh-CN": "布尔值（true/false）",
		}
	case "STRING":
		if maxLen != nil {
			return map[string]string{
				"en-US": fmt.Sprintf("String, max length: %d", *maxLen),
				"zh-CN": fmt.Sprintf("字符串，最大长度：%d", *maxLen),
			}
		}
		return map[string]string{
			"en-US": "String",
			"zh-CN": "字符串",
		}
	case "DATE_TIME", "DATETIME":
		return map[string]string{
			"en-US": "DateTime (ISO 8601)",
			"zh-CN": "日期时间（ISO 8601）",
		}
	case "OBJECT":
		return map[string]string{
			"en-US": "TR-069 object container",
			"zh-CN": "TR-069 对象容器",
		}
	default:
		return map[string]string{
			"en-US": t,
		}
	}
}

// renderImportSQL 把所有行渲染成单一 goose-compatible SQL 文件。
//
// 形态：
//
//	-- 文件头注释（版本来源 / 生成时间 / 行数）
//	-- +goose Up
//	INSERT INTO mml_params (...) VALUES
//	  (...),
//	  (...),
//	  ...
//	ON CONFLICT (param_version, tr069_path) DO UPDATE SET ...
//	WHERE mml_params.catalog_protected = true;
//
//	-- +goose Down
//	DELETE FROM mml_params WHERE source = 'standard' AND param_version = 'STANDARD';
func renderImportSQL(rows []importRow, versionCode string) ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "-- 由 omcctl mml import-standard-xml 生成（T-0123-P0 §M.3）\n")
	fmt.Fprintf(&b, "-- 生成时间：%s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "-- 行数：%d\n", len(rows))
	fmt.Fprintf(&b, "-- param_version：%s\n", versionCode)
	fmt.Fprintf(&b, "-- 后续 admin UI 改过的 catalog_protected=false 行不会被本 seed re-import 覆盖（Q2=C 决议）\n\n")

	fmt.Fprintln(&b, "-- +goose Up")
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "-- 1. 确保 mml_param_versions.STANDARD 行存在（000090 TRUNCATE 后由 Loader 重填的路径已下线，")
	fmt.Fprintln(&b, "--    本 seed 自包含 STANDARD 版本 INSERT，幂等）。")
	fmt.Fprintf(&b, "INSERT INTO mml_param_versions (version_code, version_name, description, is_active, source)\n")
	fmt.Fprintf(&b, "VALUES (%s, 'TR-069 Standard Model', 'Standard model imported from standard-model.xml by omcctl', true, 'standard')\n",
		sqlString(versionCode))
	fmt.Fprintln(&b, "ON CONFLICT (version_code) DO NOTHING;")
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "-- 2. 批量 INSERT mml_params（is_writable 列由 GENERATED ALWAYS STORED 自动派生，不显式写入）")
	fmt.Fprintln(&b, "INSERT INTO mml_params (")
	fmt.Fprintln(&b, "    id, param_version, param_code, param_name_zh, param_name_en, tr069_path, value_type,")
	fmt.Fprintln(&b, "    access_type, is_object, supports_add, supports_delete, change_applies,")
	fmt.Fprintln(&b, "    name_i18n, explanation_i18n, constraint_text_i18n,")
	fmt.Fprintln(&b, "    catalog_protected, source, display_order")
	fmt.Fprintln(&b, ") VALUES")

	for i, r := range rows {
		nameI18nJSON, err := json.Marshal(r.NameI18n)
		if err != nil {
			return nil, fmt.Errorf("marshal name_i18n %s: %w", r.Tr069Path, err)
		}
		constraintI18nJSON, err := json.Marshal(r.ConstraintTextI18n)
		if err != nil {
			return nil, fmt.Errorf("marshal constraint_text_i18n %s: %w", r.Tr069Path, err)
		}
		comma := ","
		if i == len(rows)-1 {
			comma = ""
		}
		// param_name_zh NOT NULL（migration 000022）；优先 zh-CN，回退 en-US，最终用 paramCode 兜底
		nameZh := r.NameI18n["zh-CN"]
		nameEn := r.NameI18n["en-US"]
		if nameZh == "" {
			nameZh = nameEn
		}
		if nameZh == "" {
			nameZh = r.ParamCode
		}
		fmt.Fprintf(&b,
			"    (gen_random_uuid(), %s, %s, %s, %s, %s, %s, %s, %t, %t, %t, %s, %s::jsonb, %s::jsonb, %s::jsonb, true, 'standard', %d)%s\n",
			sqlString(versionCode),
			sqlString(r.ParamCode),
			sqlString(nameZh),
			sqlString(nameEn),
			sqlString(r.Tr069Path),
			sqlString(r.ValueType),
			sqlString(r.AccessType),
			r.IsObject, r.SupportsAdd, r.SupportsDelete,
			sqlString(r.ChangeApplies),
			sqlString(string(nameI18nJSON)),
			sqlString("{}"),
			sqlString(string(constraintI18nJSON)),
			i,
			comma,
		)
	}

	fmt.Fprintln(&b, "ON CONFLICT (param_version, tr069_path) DO UPDATE SET")
	fmt.Fprintln(&b, "    access_type          = EXCLUDED.access_type,")
	fmt.Fprintln(&b, "    is_object            = EXCLUDED.is_object,")
	fmt.Fprintln(&b, "    supports_add         = EXCLUDED.supports_add,")
	fmt.Fprintln(&b, "    supports_delete      = EXCLUDED.supports_delete,")
	fmt.Fprintln(&b, "    change_applies       = EXCLUDED.change_applies,")
	fmt.Fprintln(&b, "    constraint_text_i18n = EXCLUDED.constraint_text_i18n,")
	fmt.Fprintln(&b, "    name_i18n            = EXCLUDED.name_i18n,")
	fmt.Fprintln(&b, "    value_type           = EXCLUDED.value_type")
	fmt.Fprintln(&b, "WHERE mml_params.catalog_protected = true;")

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "-- +goose Down")
	fmt.Fprintf(&b, "DELETE FROM mml_params WHERE source = 'standard' AND param_version = %s;\n", sqlString(versionCode))
	fmt.Fprintf(&b, "DELETE FROM mml_param_versions WHERE version_code = %s AND source = 'standard';\n", sqlString(versionCode))

	return []byte(b.String()), nil
}

// sqlString PostgreSQL 字符串字面值（单引号转义）。
func sqlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"

	"github.com/omcgo/omcgo/internal/config/parammodel/mmlstandardloader"
	"github.com/omcgo/omcgo/internal/task"
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
// mml_params 表已下线后保留两个子命令：
//   - `import-standard-params`  把 standard-model.xml 翻译为 standard_params seed SQL
//   - `migrate-device-params`   把历史 privatePath 翻译为 standardPath（device_parameters）
//
// 历史 `import-standard-xml` 子命令（写 mml_params）已随 v2.3 catalog 单源化下线。
func newMMLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mml",
		Short: "MML catalog management commands",
		Long:  "MML 命令字典工具：standard_params seed + device_parameters 翻译",
	}
	cmd.AddCommand(newMMLMigrateDeviceParamsCmd())
	cmd.AddCommand(newMMLImportStandardParamsCmd())
	cmd.AddCommand(newMMLImportSpecMDCmd()) // T-0169: spec md → seed SQL + JSON catalog
	cmd.AddCommand(newMMLResetScriptsCmd())
	return cmd
}

// newMMLResetScriptsCmd registers the source-scoped Redis cleanup command.
// PostgreSQL cleanup is handled by the corresponding migration; this command
// only touches queued MML tasks in Redis.
func newMMLResetScriptsCmd() *cobra.Command {
	var (
		redisAddr string
		dryRun    bool
		apply     bool
		confirm   string
	)
	c := &cobra.Command{
		Use:   "reset-script-data",
		Short: "Safely purge queued MML task data from Redis",
		Long: `扫描并清理 Redis 中 source=mml 的任务队列项。

默认仅 dry-run 统计，不修改 Redis。执行删除必须同时指定
--apply --confirm DELETE-MML-RUNTIME；命令绝不使用 KEYS/FLUSHDB。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMMLResetScripts(redisAddr, dryRun, apply, confirm)
		},
	}
	c.Flags().StringVar(&redisAddr, "redis-addr", "localhost:6379", "Redis address")
	c.Flags().BoolVar(&dryRun, "dry-run", true, "Only report matching tasks; default true")
	c.Flags().BoolVar(&apply, "apply", false, "Actually delete matching tasks")
	c.Flags().StringVar(&confirm, "confirm", "", "Required confirmation for --apply: DELETE-MML-RUNTIME")
	return c
}

func runMMLResetScripts(redisAddr string, dryRun, apply bool, confirm string) error {
	if apply {
		if confirm != "DELETE-MML-RUNTIME" {
			return fmt.Errorf("--apply requires --confirm DELETE-MML-RUNTIME")
		}
		dryRun = false
	}
	// A caller that omits --apply always stays in dry-run mode, even if it
	// explicitly passes --dry-run=false; deletion requires the explicit guard.
	if !apply {
		dryRun = true
	}

	result, err := executeMMLResetScripts(redisAddr, dryRun, apply)
	if err != nil {
		return err
	}
	fmt.Printf("matched=%d deleted=%d skipped=%d errors=%d\n", result.Matched, result.Deleted, result.Skipped, result.Errors)
	if result.Errors > 0 {
		return fmt.Errorf("MML Redis purge completed with %d errors", result.Errors)
	}
	return nil
}

func executeMMLResetScripts(redisAddr string, dryRun, apply bool) (task.PurgeBySourceResult, error) {
	if apply {
		dryRun = false
	}
	ctx, cancel := contextWithTimeout(5 * time.Minute)
	defer cancel()

	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		return task.PurgeBySourceResult{}, fmt.Errorf("ping redis: %w", err)
	}
	queue := task.NewRedisTaskQueue(client)
	return queue.PurgeBySource(ctx, task.TaskSourceMML, dryRun)
}

// newMMLImportStandardParamsCmd 注册 `omcctl mml import-standard-params` 子命令。
//
// 把 standard-model.xml 翻译为 standard_params 表的 INSERT SQL 片段。
//
// 背景：standard_params 运行期由 parammodel Loader 填充；但 seed/000111 在迁移期
// 用 `SELECT FROM standard_params` 生成 mml_command_sub_fields —— 迁移先于
// Loader，那时该表还空 → 0 sub_fields。本命令产出 seed 片段，供新迁移在
// sub_field 生成之前先把 standard_params 种进去。
func newMMLImportStandardParamsCmd() *cobra.Command {
	var (
		xmlPath string
		outPath string
	)
	c := &cobra.Command{
		Use:   "import-standard-params",
		Short: "Convert standard-model.xml to standard_params seed SQL fragment",
		Long: `把 TR-069 standard-model.xml 翻译为 standard_params 表的 INSERT SQL 片段。

列与 parammodel Loader.batchInsertStandardEntries 完全一致：
  standard_path / entry_type / access / data_type / change_applies / min_value / max_value
ON CONFLICT (standard_path) DO UPDATE —— 与运行期 Loader 同锚点，幂等、互不冲突。
输出为纯 SQL 片段（无 goose 标记），由调用者拼进迁移文件。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStandardParamsImport(xmlPath, outPath)
		},
	}
	c.Flags().StringVar(&xmlPath, "xml", "omcgo/data/param-mappings/standard-model.xml", "Path to standard-model.xml input")
	c.Flags().StringVar(&outPath, "out", "omcgo/migrations/seed/.standard_params_rows.sql", "Output SQL fragment path")
	return c
}

// runStandardParamsImport 解析 XML → 生成 standard_params 批量 INSERT SQL 片段。
func runStandardParamsImport(xmlPath, outPath string) error {
	params, objects, err := mmlstandardloader.ParseStandardXMLFile(xmlPath)
	if err != nil {
		return fmt.Errorf("parse standard-model.xml: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[omcctl mml import-standard-params] parsed %d params + %d objects from %s\n",
		len(params), len(objects), xmlPath)

	// standard_params.standard_path UNIQUE —— 同 path 在一条 INSERT 内出现两次会让
	// ON CONFLICT DO UPDATE 报错，故先按 path 去重。objects 先放、params 后放，
	// 同 path 重复（XML 正常无）以 params 为准（与 Loader「objects 批 + params 批」
	// 先后顺序一致）。
	type spRow struct {
		path, entryType, access, dataType, change string
		min, max                                  *int64
	}
	order := make([]string, 0, len(params)+len(objects))
	byPath := make(map[string]spRow, len(params)+len(objects))
	put := func(r spRow) {
		if _, seen := byPath[r.path]; !seen {
			order = append(order, r.path)
		}
		byPath[r.path] = r
	}
	for _, o := range objects {
		put(spRow{path: o.StandardPath, entryType: "object", access: o.Access, change: o.ChangeApplies})
	}
	for _, p := range params {
		put(spRow{path: p.StandardPath, entryType: "parameter", access: p.Access, dataType: p.Type, change: p.ChangeApplies, min: p.Min, max: p.Max})
	}

	var b strings.Builder
	fmt.Fprintf(&b, "-- 由 omcctl mml import-standard-params 生成\n")
	fmt.Fprintf(&b, "-- 生成时间：%s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "-- standard_params 行数：%d（列与 parammodel Loader 一致）\n\n", len(order))

	const chunk = 200
	for i := 0; i < len(order); i += chunk {
		end := i + chunk
		if end > len(order) {
			end = len(order)
		}
		fmt.Fprintln(&b, "INSERT INTO standard_params (standard_path, entry_type, access, data_type, change_applies, min_value, max_value) VALUES")
		for j := i; j < end; j++ {
			r := byPath[order[j]]
			comma := ","
			if j == end-1 {
				comma = ""
			}
			fmt.Fprintf(&b, "    (%s, %s, %s, %s, %s, %s, %s)%s\n",
				sqlString(r.path), sqlString(r.entryType),
				sqlStringOrNull(r.access), sqlStringOrNull(r.dataType), sqlStringOrNull(r.change),
				sqlIntOrNull(r.min), sqlIntOrNull(r.max), comma)
		}
		b.WriteString("ON CONFLICT (standard_path) DO UPDATE SET\n" +
			"    entry_type     = EXCLUDED.entry_type,\n" +
			"    access         = EXCLUDED.access,\n" +
			"    data_type      = EXCLUDED.data_type,\n" +
			"    change_applies = EXCLUDED.change_applies,\n" +
			"    min_value      = EXCLUDED.min_value,\n" +
			"    max_value      = EXCLUDED.max_value,\n" +
			"    updated_at     = NOW();\n\n")
	}

	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	fmt.Fprintf(os.Stderr, "[omcctl mml import-standard-params] wrote %d standard_params rows → %s\n", len(order), outPath)
	return nil
}

// sqlStringOrNull 渲染 SQL 文本字面量；空串 → NULL（对齐 Loader.nullIfEmpty）。
func sqlStringOrNull(s string) string {
	if s == "" {
		return "NULL"
	}
	return sqlString(s)
}

// sqlIntOrNull 渲染可空整数字面量；nil → NULL。
func sqlIntOrNull(p *int64) string {
	if p == nil {
		return "NULL"
	}
	return strconv.FormatInt(*p, 10)
}

// newMMLMigrateDeviceParamsCmd 注册 `omcctl mml migrate-device-params` 子命令。
//
// 整改方案 Stage 2（用户决策 2026-05-16）：device_parameters.parameter_path
// 历史值可能是 privatePath（旧 Inform 直接入库），新写入都是 standardPath
// （rpc_response_subscriber 翻译后入库 + Path B 同步）。本命令把历史 private
// 行翻译为 standard，按 (device_id, parameter_path) 行级 UPDATE。
//
// 路径：device.product_class → products.product_class regex 匹配 → product.id
//
//	→ param_mappings.private_path == dp.parameter_path → standard_path
//
// 翻译失败的行（未匹配到 product / 未在 param_mappings 中找到 private_path）
// 保持不变，由运维通过 admin UI 补 discovered_param_mappings 后重跑命令。
//
// Flags：
//
//	--dsn       连接串（必填）
//	--dry-run   仅扫描+报告，不写入（默认）
//	--apply     真执行 UPDATE
//	--batch     批大小（默认 500，超大表分批避免长事务）
func newMMLMigrateDeviceParamsCmd() *cobra.Command {
	var (
		dsn    string
		dryRun bool
		apply  bool
		batch  int
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

// sqlString PostgreSQL 字符串字面值（单引号转义）。
func sqlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

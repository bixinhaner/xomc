package indicator

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FileRepository 抽象 T-0180 P1.3+ XML 文件层级的 DB 操作。
//
// 接口故意小(2 方法):便于 file_handler_test.go 用纯内存 stub 替换,
// 不依赖 testcontainers/pg dockertest。
//
// 实现:PgFileRepository(同包 PG 实现);测试用 mockFileRepository(_test 文件内)。
type FileRepository interface {
	// CountByLoadedFrom 统计指定 tech(enb/gsm/gnb)表中 loaded_from = ? 的行数。
	// 用于 DELETE 前的存在性校验(0 → 404)。
	CountByLoadedFrom(ctx context.Context, tech, loadedFrom string) (int, error)

	// DeleteByLoadedFrom 在单事务中级联删除该 loaded_from 关联的:
	//   - rela_platform_indicator_formula_<tech>(被删指标的公式)
	//   - enabled_pm_indicators_<tech>(启用记录)
	//   - perf_indicators_<tech>(主表)
	// 不删 indicator_group_<tech>(分组可能被其他文件引用,见 PRD §5 非目标 #5)
	//
	// 返回 perf_indicators_<tech> 主表删除的行数(供日志与响应体)。
	DeleteByLoadedFrom(ctx context.Context, tech, loadedFrom string) (int, error)

	// SummaryByTech 聚合三制式所有 (loaded_from, platform) 组合,每个 (制式, 平台)
	// 一行(2026-05-29 用户决策从"三行/制式"调为"N 行/(制式,平台)粒度")。
	//
	// JOIN perf_indicators_<tech> ⨝ rela_platform_indicator_formula_<tech> GROUP BY
	// (loaded_from, platform_name);只返带 builtin/custom 前缀的 loaded_from(过滤 unknown);
	// INNER JOIN 排除无 platform 的 indicator(罕见,通常 Loader 入库即建公式)。
	SummaryByTech(ctx context.Context) ([]PlatformSummary, error)

	// ListFilesByTech 按 (loaded_from) GROUP BY 列出指定 tech 下所有 XML 文件的
	// 指标计数。NULL loaded_from(历史数据未回填)归到 ""(由调用方决定如何展示)。
	ListFilesByTech(ctx context.Context, tech string) ([]FileGroup, error)

	// DeleteOrphansBefore 删除 perf_indicators_<tech> 中 updated_at < before 的行
	// 与级联的 rela_platform_indicator_formula_<tech> + enabled_pm_indicators_<tech>。
	//
	// 使用场景:ImportDirectory ?mode=reload 期间,先记录 start := time.Now(),
	// 然后触发 Loader.ReloadOne(UPSERT 触发 BEFORE UPDATE trigger 自动刷 updated_at),
	// 再调本方法删除 updated_at < start 的指标 — 即"Reload 未覆盖到的孤儿"。
	//
	// 单事务三步级联(formula → enabled → main,逆依赖顺序);
	// 返回 perf_indicators_<tech> 主表删除行数(供日志与响应体)。
	DeleteOrphansBefore(ctx context.Context, tech string, before time.Time) (int, error)
}

// PlatformSummary 是 /indicators/summary 端点单行 — (制式, 平台) 二元组粒度。
//
// 2026-05-29 用户决策:从"三行/制式"(TechSummary)调为"N 行/(制式, 平台)"
//   - 每个 (tech, platform_name) 唯一一行
//   - LoadedFrom 是该 (tech, platform) 对应的 XML 文件 rel 路径(理论上 1:1 映射)
//   - Indicators 是该文件入库的指标计数
type PlatformSummary struct {
	Tech       string `json:"tech"`        // enb / gsm / gnb
	Platform   string `json:"platform"`    // 平台名(从 rela_platform_indicator_formula_*.platform_name)
	LoadedFrom string `json:"loaded_from"` // XML 文件相对路径(含前缀),如 "indicator-library/enb/ALL.xml"
	Indicators int    `json:"indicators"`  // 该 (loaded_from, platform_name) 的指标计数
}

// FileGroup 是 /indicators/files?tech= 单行 — DB 聚合视角。
// 物理目录扫描的合并由 FileHandler.ListFiles 完成(uploaded-but-not-loaded 场景)。
type FileGroup struct {
	LoadedFrom string `json:"loaded_from"`
	Count      int    `json:"count"`
}

// PgFileRepository 是 FileRepository 的 PostgreSQL 实现。
type PgFileRepository struct {
	pool *pgxpool.Pool
}

func NewPgFileRepository(pool *pgxpool.Pool) *PgFileRepository {
	return &PgFileRepository{pool: pool}
}

// allowedTech 是 DeleteByLoadedFrom / CountByLoadedFrom 接受的合法 tech 值集合。
// 字符串硬编码进 SQL 表名,因此必须强校验防 SQL 注入。
var allowedTech = map[string]struct{}{
	"enb": {},
	"gsm": {},
	"gnb": {},
}

func validateTech(tech string) error {
	if _, ok := allowedTech[tech]; !ok {
		return fmt.Errorf("invalid tech %q: must be one of enb/gsm/gnb", tech)
	}
	return nil
}

// CountByLoadedFrom 实现 FileRepository。
func (r *PgFileRepository) CountByLoadedFrom(ctx context.Context, tech, loadedFrom string) (int, error) {
	if err := validateTech(tech); err != nil {
		return 0, err
	}
	// tech 经白名单校验,可拼接;loadedFrom 走参数化绑定
	sqlStr := fmt.Sprintf(`SELECT COUNT(*) FROM perf_indicators_%s WHERE loaded_from = $1`, tech)
	var n int
	if err := r.pool.QueryRow(ctx, sqlStr, loadedFrom).Scan(&n); err != nil {
		return 0, fmt.Errorf("count perf_indicators_%s by loaded_from: %w", tech, err)
	}
	return n, nil
}

// DeleteByLoadedFrom 实现 FileRepository — 单事务三步级联。
//
// 顺序:formula → enabled → 主表(逆依赖方向,避免外键悬挂);
// 三步均按 indicator_id IN (SELECT id FROM perf_indicators_<tech> WHERE loaded_from = $1)
// 子查询命中,即使 enabled / formula 数为 0 也无副作用。
//
// 返回主表 perf_indicators_<tech> 实际删除的行数(以 RowsAffected 计)。
func (r *PgFileRepository) DeleteByLoadedFrom(ctx context.Context, tech, loadedFrom string) (int, error) {
	if err := validateTech(tech); err != nil {
		return 0, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	formulaSQL := fmt.Sprintf(`
DELETE FROM rela_platform_indicator_formula_%s
 WHERE indicator_id IN (
   SELECT id FROM perf_indicators_%s WHERE loaded_from = $1
 )`, tech, tech)
	if _, err := tx.Exec(ctx, formulaSQL, loadedFrom); err != nil {
		return 0, fmt.Errorf("cascade delete formula (%s): %w", tech, err)
	}

	enabledSQL := fmt.Sprintf(`
DELETE FROM enabled_pm_indicators_%s
 WHERE indicator_id IN (
   SELECT id FROM perf_indicators_%s WHERE loaded_from = $1
 )`, tech, tech)
	if _, err := tx.Exec(ctx, enabledSQL, loadedFrom); err != nil {
		return 0, fmt.Errorf("cascade delete enabled (%s): %w", tech, err)
	}

	mainSQL := fmt.Sprintf(`DELETE FROM perf_indicators_%s WHERE loaded_from = $1`, tech)
	tag, err := tx.Exec(ctx, mainSQL, loadedFrom)
	if err != nil {
		return 0, fmt.Errorf("delete perf_indicators_%s: %w", tech, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit delete tx: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// SummaryByTech 实现 FileRepository — 按 (tech, loaded_from, platform_name) GROUP BY,
// 每 (制式, 平台) 一行(2026-05-29 用户决策粒度调整)。
//
// JOIN 策略:
//   - INNER JOIN perf_indicators_<tech> 与 rela_platform_indicator_formula_<tech>
//   - 排除 loaded_from 不带 builtin/custom 前缀的"未知"行
//   - GROUP BY (loaded_from, platform_name) → 一文件一平台 = 一行
//   - 实践中每 XML <indicatorModel platform="X"> 单一 platform,但允许多平台也兼容
//
// 顺序:tech 升序(enb/gsm/gnb)→ loaded_from 升序 → platform 升序,UX 稳定。
func (r *PgFileRepository) SummaryByTech(ctx context.Context) ([]PlatformSummary, error) {
	out := make([]PlatformSummary, 0, 16)
	for _, tech := range []string{"enb", "gsm", "gnb"} {
		sqlStr := fmt.Sprintf(`
SELECT pi.loaded_from,
       rf.platform_name,
       COUNT(DISTINCT pi.id) AS indicators
  FROM perf_indicators_%s pi
 INNER JOIN rela_platform_indicator_formula_%s rf ON rf.indicator_id = pi.id
 WHERE pi.loaded_from LIKE 'indicator-library/%%'
    OR pi.loaded_from LIKE 'indicator-library-custom/%%'
 GROUP BY pi.loaded_from, rf.platform_name
 ORDER BY pi.loaded_from, rf.platform_name`, tech, tech)

		rows, err := r.pool.Query(ctx, sqlStr)
		if err != nil {
			return nil, fmt.Errorf("summary platforms (%s): %w", tech, err)
		}
		for rows.Next() {
			row := PlatformSummary{Tech: tech}
			if err := rows.Scan(&row.LoadedFrom, &row.Platform, &row.Indicators); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan platform summary row (%s): %w", tech, err)
			}
			out = append(out, row)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate platform summary (%s): %w", tech, err)
		}
	}
	return out, nil
}

// DeleteOrphansBefore 实现 FileRepository(T-0180 P1.5)。
//
// 单事务三步级联(逆依赖顺序);全部基于 perf_indicators_<tech>.updated_at < $1 子查询,
// 该列由 trigger_perf_indicators_<tech>_updated_at(migrations/000035 §Triggers)
// 在 BEFORE UPDATE 时自动刷新 — Loader.Reload 走 ON CONFLICT DO UPDATE 路径必触发。
func (r *PgFileRepository) DeleteOrphansBefore(ctx context.Context, tech string, before time.Time) (int, error) {
	if err := validateTech(tech); err != nil {
		return 0, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	formulaSQL := fmt.Sprintf(`
DELETE FROM rela_platform_indicator_formula_%s
 WHERE indicator_id IN (
   SELECT id FROM perf_indicators_%s WHERE updated_at < $1
 )`, tech, tech)
	if _, err := tx.Exec(ctx, formulaSQL, before); err != nil {
		return 0, fmt.Errorf("cascade delete orphan formula (%s): %w", tech, err)
	}

	enabledSQL := fmt.Sprintf(`
DELETE FROM enabled_pm_indicators_%s
 WHERE indicator_id IN (
   SELECT id FROM perf_indicators_%s WHERE updated_at < $1
 )`, tech, tech)
	if _, err := tx.Exec(ctx, enabledSQL, before); err != nil {
		return 0, fmt.Errorf("cascade delete orphan enabled (%s): %w", tech, err)
	}

	mainSQL := fmt.Sprintf(`DELETE FROM perf_indicators_%s WHERE updated_at < $1`, tech)
	tag, err := tx.Exec(ctx, mainSQL, before)
	if err != nil {
		return 0, fmt.Errorf("delete orphan perf_indicators_%s: %w", tech, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit orphan delete tx (%s): %w", tech, err)
	}
	return int(tag.RowsAffected()), nil
}

// ListFilesByTech 实现 FileRepository — 按 loaded_from GROUP BY 单查询。
// NULL loaded_from 通过 COALESCE 归到 ""(调用方决定展示策略)。
func (r *PgFileRepository) ListFilesByTech(ctx context.Context, tech string) ([]FileGroup, error) {
	if err := validateTech(tech); err != nil {
		return nil, err
	}
	sqlStr := fmt.Sprintf(`
SELECT COALESCE(loaded_from, '') AS lf, COUNT(*) AS c
  FROM perf_indicators_%s
 GROUP BY COALESCE(loaded_from, '')
 ORDER BY lf`, tech)
	rows, err := r.pool.Query(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("list files by tech (%s): %w", tech, err)
	}
	defer rows.Close()
	out := make([]FileGroup, 0, 8)
	for rows.Next() {
		var fg FileGroup
		if err := rows.Scan(&fg.LoadedFrom, &fg.Count); err != nil {
			return nil, fmt.Errorf("scan file group (%s): %w", tech, err)
		}
		out = append(out, fg)
	}
	return out, rows.Err()
}

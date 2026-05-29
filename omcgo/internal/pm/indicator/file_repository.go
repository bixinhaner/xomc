package indicator

import (
	"context"
	"fmt"

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

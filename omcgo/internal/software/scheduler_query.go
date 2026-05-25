package software

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// listDueScheduledFromTable 给 software / transfer/repo 两边复用：扫描指定物理表里
// "create_status=timing AND status=pending AND scheduled_at <= before" 的行，
// 按 scheduled_at 升序返回（最早到期的优先触发）。
//
// 与 idx_<table>_due_scheduled partial index 配套；命中索引后实际全表扫描代价 ≈ 0。
// limit > 0 时切分 LIMIT，避免一次性把大批量任务拉进内存。
func listDueScheduledFromTable(
	ctx context.Context,
	pool *pgxpool.Pool,
	table string,
	before time.Time,
	limit int,
) ([]*UpgradeTask, error) {
	builder := storage.Psql.Select(taskColumns...).
		From(table).
		Where(sq.Eq{"create_status": CreateStatusTiming}).
		Where(sq.Eq{"status": TaskPending}).
		Where(sq.LtOrEq{"scheduled_at": before}).
		Where("scheduled_at IS NOT NULL").
		OrderBy("scheduled_at ASC")

	if limit > 0 {
		builder = builder.Limit(uint64(limit))
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list due scheduled SQL (%s): %w", table, err)
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query due scheduled (%s): %w", table, err)
	}
	defer rows.Close()

	var out []*UpgradeTask
	for rows.Next() {
		t, err := scanUpgradeTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan due scheduled (%s): %w", table, err)
		}
		out = append(out, t)
	}
	return out, nil
}

// updateCreateStatusGuardedOn 把指定表中 id 对应行的 create_status: from → to，
// 仅在 status=pending 时生效，配合 scheduler 的抢占式锁。RowsAffected=1 返回 true。
func updateCreateStatusGuardedOn(
	ctx context.Context,
	pool *pgxpool.Pool,
	table string,
	id uuid.UUID,
	from, to string,
) (bool, error) {
	q, args, err := storage.Psql.Update(table).
		Set("create_status", to).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"create_status": from}).
		Where(sq.Eq{"status": string(TaskPending)}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build update create_status SQL (%s): %w", table, err)
	}
	tag, err := pool.Exec(ctx, q, args...)
	if err != nil {
		return false, fmt.Errorf("exec update create_status (%s): %w", table, err)
	}
	return tag.RowsAffected() == 1, nil
}

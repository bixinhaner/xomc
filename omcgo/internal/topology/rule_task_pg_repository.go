package topology

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sq "github.com/Masterminds/squirrel"
)

// PgRuleTaskRepository 规则任务仓储的 PostgreSQL 实现
type PgRuleTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgRuleTaskRepository 创建规则任务仓储
func NewPgRuleTaskRepository(pool *pgxpool.Pool) *PgRuleTaskRepository {
	return &PgRuleTaskRepository{pool: pool}
}

// Create 创建任务
func (r *PgRuleTaskRepository) Create(ctx context.Context, task *RuleTask) error {
	query, args, err := sq.Insert("device_rule_tasks").
		Columns("rule_id", "status", "created_by").
		Values(task.RuleID, task.Status, task.CreatedBy).
		Suffix("RETURNING id, created_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert task SQL: %w", err)
	}

	err = r.pool.QueryRow(ctx, query, args...).Scan(&task.ID, &task.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取任务
func (r *PgRuleTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*RuleTask, error) {
	query, args, err := sq.Select(
		"t.id", "t.rule_id", "t.status", "t.total_devices", "t.matched_count", "t.failed_count",
		"t.started_at", "t.completed_at", "t.error_message", "t.created_by", "t.created_at",
		"r.name as rule_name",
	).
		From("device_rule_tasks t").
		LeftJoin("device_rules r ON t.rule_id = r.id").
		Where(sq.Eq{"t.id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get task SQL: %w", err)
	}

	var task RuleTask
	var ruleName *string
	var startedAt, completedAt *time.Time
	var errorMessage, createdBy *string

	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&task.ID, &task.RuleID, &task.Status, &task.TotalDevices, &task.MatchedCount, &task.FailedCount,
		&startedAt, &completedAt, &errorMessage, &createdBy, &task.CreatedAt,
		&ruleName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("scan task: %w", err)
	}

	task.StartedAt = startedAt
	task.CompletedAt = completedAt
	if errorMessage != nil {
		task.ErrorMessage = *errorMessage
	}
	if createdBy != nil {
		task.CreatedBy = *createdBy
	}
	if ruleName != nil {
		task.RuleName = *ruleName
	}

	// 计算进度
	if task.TotalDevices > 0 {
		task.Progress = (task.MatchedCount + task.FailedCount) * 100 / task.TotalDevices
	}

	return &task, nil
}

// Update 更新任务
func (r *PgRuleTaskRepository) Update(ctx context.Context, task *RuleTask) error {
	query, args, err := sq.Update("device_rule_tasks").
		Set("status", task.Status).
		Set("total_devices", task.TotalDevices).
		Set("matched_count", task.MatchedCount).
		Set("failed_count", task.FailedCount).
		Set("started_at", task.StartedAt).
		Set("completed_at", task.CompletedAt).
		Set("error_message", task.ErrorMessage).
		Where(sq.Eq{"id": task.ID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update task SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// ListByRule 获取规则的任务列表
func (r *PgRuleTaskRepository) ListByRule(ctx context.Context, ruleID uuid.UUID, limit int) ([]RuleTask, error) {
	query, args, err := sq.Select(
		"t.id", "t.rule_id", "t.status", "t.total_devices", "t.matched_count", "t.failed_count",
		"t.started_at", "t.completed_at", "t.error_message", "t.created_by", "t.created_at",
		"r.name as rule_name",
	).
		From("device_rule_tasks t").
		LeftJoin("device_rules r ON t.rule_id = r.id").
		Where(sq.Eq{"t.rule_id": ruleID}).
		OrderBy("t.created_at DESC").
		Limit(uint64(limit)).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list tasks SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []RuleTask
	for rows.Next() {
		var task RuleTask
		var ruleName *string
		var startedAt, completedAt *time.Time
		var errorMessage, createdBy *string

		err := rows.Scan(
			&task.ID, &task.RuleID, &task.Status, &task.TotalDevices, &task.MatchedCount, &task.FailedCount,
			&startedAt, &completedAt, &errorMessage, &createdBy, &task.CreatedAt,
			&ruleName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		task.StartedAt = startedAt
		task.CompletedAt = completedAt
		if errorMessage != nil {
			task.ErrorMessage = *errorMessage
		}
		if createdBy != nil {
			task.CreatedBy = *createdBy
		}
		if ruleName != nil {
			task.RuleName = *ruleName
		}

		// 计算进度
		if task.TotalDevices > 0 {
			task.Progress = (task.MatchedCount + task.FailedCount) * 100 / task.TotalDevices
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetLatestByRule 获取规则的最新任务
func (r *PgRuleTaskRepository) GetLatestByRule(ctx context.Context, ruleID uuid.UUID) (*RuleTask, error) {
	tasks, err := r.ListByRule(ctx, ruleID, 1)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	return &tasks[0], nil
}

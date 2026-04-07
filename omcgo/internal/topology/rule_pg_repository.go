package topology

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	sq "github.com/Masterminds/squirrel"
)

// PgDeviceRuleRepository 设备规则的 PostgreSQL 实现
type PgDeviceRuleRepository struct {
	pool *pgxpool.Pool
}

// NewPgDeviceRuleRepository 创建设备规则仓储
func NewPgDeviceRuleRepository(pool *pgxpool.Pool) *PgDeviceRuleRepository {
	return &PgDeviceRuleRepository{pool: pool}
}

// Create 创建设备规则
func (r *PgDeviceRuleRepository) Create(ctx context.Context, rule *DeviceRule) error {
	// 序列化 name_rule_list
	var nameRuleListJSON []byte
	if len(rule.NameRuleList) > 0 {
		var err error
		nameRuleListJSON, err = json.Marshal(rule.NameRuleList)
		if err != nil {
			return fmt.Errorf("marshal name_rule_list: %w", err)
		}
	}

	query, args, err := sq.Insert("device_rules").
		Columns(
			"name", "priority", "target_group_id", "enabled",
			"matching_mode", "name_rule_list", "lac_list", "tac_list",
			"description", "operators", "created_by", "updated_by",
		).
		Values(
			rule.Name, rule.Priority, rule.TargetGroupID, rule.Enabled,
			rule.MatchingMode, nameRuleListJSON, rule.LACList, rule.TACList,
			rule.Description, rule.Operators, rule.CreatedBy, rule.UpdatedBy,
		).
		Suffix("RETURNING id, created_at, updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert SQL: %w", err)
	}

	err = r.pool.QueryRow(ctx, query, args...).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert device rule: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取设备规则
func (r *PgDeviceRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*DeviceRule, error) {
	query, args, err := sq.Select(
		"dr.id", "dr.name", "dr.priority", "dr.target_group_id", "dr.enabled",
		"dr.matching_mode", "dr.name_rule_list", "dr.lac_list", "dr.tac_list",
		"dr.description", "dr.operators", "dr.created_by", "dr.updated_by", "dr.created_at", "dr.updated_at",
		"COALESCE(dg.name, '') as target_group_name",
	).
		From("device_rules dr").
		LeftJoin("device_groups dg ON dr.target_group_id = dg.id").
		Where(sq.Eq{"dr.id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select SQL: %w", err)
	}

	var rule DeviceRule
	var matchingMode *string
	var nameRuleListJSON []byte
	var targetGroupID *uuid.UUID
	var createdBy, updatedBy, description, operators *string

	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&rule.ID, &rule.Name, &rule.Priority, &targetGroupID, &rule.Enabled,
		&matchingMode, &nameRuleListJSON, &rule.LACList, &rule.TACList,
		&description, &operators, &createdBy, &updatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		&rule.TargetGroupName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("device rule not found")
		}
		return nil, fmt.Errorf("scan device rule: %w", err)
	}

	rule.TargetGroupID = targetGroupID
	if matchingMode != nil {
		rule.MatchingMode = MatchingMode(*matchingMode)
	}
	if createdBy != nil {
		rule.CreatedBy = *createdBy
	}
	if updatedBy != nil {
		rule.UpdatedBy = *updatedBy
	}
	if description != nil {
		rule.Description = *description
	}
	if operators != nil {
		rule.Operators = *operators
	}

	// 解析 name_rule_list
	if len(nameRuleListJSON) > 0 {
		if err := json.Unmarshal(nameRuleListJSON, &rule.NameRuleList); err != nil {
			return nil, fmt.Errorf("unmarshal name_rule_list: %w", err)
		}
	}

	return &rule, nil
}

// Update 更新设备规则
func (r *PgDeviceRuleRepository) Update(ctx context.Context, rule *DeviceRule) error {
	rule.UpdatedAt = time.Now()

	// 序列化 name_rule_list
	var nameRuleListJSON []byte
	if len(rule.NameRuleList) > 0 {
		var err error
		nameRuleListJSON, err = json.Marshal(rule.NameRuleList)
		if err != nil {
			return fmt.Errorf("marshal name_rule_list: %w", err)
		}
	}

	query, args, err := sq.Update("device_rules").
		Set("name", rule.Name).
		Set("priority", rule.Priority).
		Set("target_group_id", rule.TargetGroupID).
		Set("enabled", rule.Enabled).
		Set("matching_mode", rule.MatchingMode).
		Set("name_rule_list", nullableJSONB(nameRuleListJSON)).
		Set("lac_list", pq.Array(rule.LACList)).
		Set("tac_list", pq.Array(rule.TACList)).
		Set("description", rule.Description).
		Set("operators", rule.Operators).
		Set("updated_by", rule.UpdatedBy).
		Set("updated_at", rule.UpdatedAt).
		Where(sq.Eq{"id": rule.ID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("device rule not found")
	}

	return nil
}

// Delete 删除设备规则
func (r *PgDeviceRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := sq.Delete("device_rules").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete device rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("device rule not found")
	}

	return nil
}

// List 获取规则列表
func (r *PgDeviceRuleRepository) List(ctx context.Context, req *RuleListRequest) ([]DeviceRule, int64, error) {
	// 构建过滤条件
	conditions := sq.And{}
	if req.Enabled != nil {
		conditions = append(conditions, sq.Eq{"dr.enabled": *req.Enabled})
	}
	if req.Name != "" {
		conditions = append(conditions, sq.ILike{"dr.name": "%" + req.Name + "%"})
	}

	// 查询总数
	countQuery, countArgs, err := sq.Select("COUNT(*)").
		From("device_rules dr").
		Where(conditions).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count SQL: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count device rules: %w", err)
	}

	// 构建列表查询
	offset := (req.Page - 1) * req.PageSize
	query, args, err := sq.Select(
		"dr.id", "dr.name", "dr.priority", "dr.target_group_id", "dr.enabled",
		"dr.matching_mode", "dr.name_rule_list", "dr.lac_list", "dr.tac_list",
		"dr.description", "dr.operators", "dr.created_by", "dr.updated_by", "dr.created_at", "dr.updated_at",
		"COALESCE(dg.name, '') as target_group_name",
	).
		From("device_rules dr").
		LeftJoin("device_groups dg ON dr.target_group_id = dg.id").
		Where(conditions).
		OrderByClause("dr.priority ASC").
		Limit(uint64(req.PageSize)).Offset(uint64(offset)).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query device rules: %w", err)
	}
	defer rows.Close()

	var rules []DeviceRule
	for rows.Next() {
		var rule DeviceRule
		var matchingMode *string
		var nameRuleListJSON []byte
		var targetGroupID *uuid.UUID
		var createdBy, updatedBy, description, operators *string

		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Priority, &targetGroupID, &rule.Enabled,
			&matchingMode, &nameRuleListJSON, &rule.LACList, &rule.TACList,
			&description, &operators, &createdBy, &updatedBy, &rule.CreatedAt, &rule.UpdatedAt,
			&rule.TargetGroupName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan device rule: %w", err)
		}

		rule.TargetGroupID = targetGroupID
		if matchingMode != nil {
			rule.MatchingMode = MatchingMode(*matchingMode)
		}
		if createdBy != nil {
			rule.CreatedBy = *createdBy
		}
		if updatedBy != nil {
			rule.UpdatedBy = *updatedBy
		}
		if description != nil {
			rule.Description = *description
		}
		if operators != nil {
			rule.Operators = *operators
		}

		// 解析 name_rule_list
		if len(nameRuleListJSON) > 0 {
			if err := json.Unmarshal(nameRuleListJSON, &rule.NameRuleList); err != nil {
				return nil, 0, fmt.Errorf("unmarshal name_rule_list: %w", err)
			}
		}

		rules = append(rules, rule)
	}

	return rules, total, nil
}

// GetAll 获取所有规则
func (r *PgDeviceRuleRepository) GetAll(ctx context.Context) ([]DeviceRule, error) {
	query, args, err := sq.Select(
		"id", "name", "priority", "target_group_id", "enabled",
		"matching_mode", "name_rule_list", "lac_list", "tac_list",
		"description", "operators", "created_by", "updated_by", "created_at", "updated_at",
	).
		From("device_rules").
		OrderBy("priority ASC").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get all SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query all device rules: %w", err)
	}
	defer rows.Close()

	var rules []DeviceRule
	for rows.Next() {
		var rule DeviceRule
		var matchingMode *string
		var nameRuleListJSON []byte
		var targetGroupID *uuid.UUID
		var createdBy, updatedBy, description, operators *string

		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Priority, &targetGroupID, &rule.Enabled,
			&matchingMode, &nameRuleListJSON, &rule.LACList, &rule.TACList,
			&description, &operators, &createdBy, &updatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan device rule: %w", err)
		}

		rule.TargetGroupID = targetGroupID
		if matchingMode != nil {
			rule.MatchingMode = MatchingMode(*matchingMode)
		}
		if createdBy != nil {
			rule.CreatedBy = *createdBy
		}
		if updatedBy != nil {
			rule.UpdatedBy = *updatedBy
		}
		if description != nil {
			rule.Description = *description
		}
		if operators != nil {
			rule.Operators = *operators
		}

		if len(nameRuleListJSON) > 0 {
			if err := json.Unmarshal(nameRuleListJSON, &rule.NameRuleList); err != nil {
				return nil, fmt.Errorf("unmarshal name_rule_list: %w", err)
			}
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// GetEnabledByPriority 获取所有启用的规则，按优先级排序
func (r *PgDeviceRuleRepository) GetEnabledByPriority(ctx context.Context) ([]DeviceRule, error) {
	query, args, err := sq.Select(
		"id", "name", "priority", "target_group_id", "enabled",
		"matching_mode", "name_rule_list", "lac_list", "tac_list",
		"description", "operators", "created_by", "updated_by", "created_at", "updated_at",
	).
		From("device_rules").
		Where(sq.Eq{"enabled": true}).
		OrderBy("priority ASC").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get enabled SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query enabled device rules: %w", err)
	}
	defer rows.Close()

	var rules []DeviceRule
	for rows.Next() {
		var rule DeviceRule
		var matchingMode *string
		var nameRuleListJSON []byte
		var targetGroupID *uuid.UUID
		var createdBy, updatedBy, description, operators *string

		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Priority, &targetGroupID, &rule.Enabled,
			&matchingMode, &nameRuleListJSON, &rule.LACList, &rule.TACList,
			&description, &operators, &createdBy, &updatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan device rule: %w", err)
		}

		rule.TargetGroupID = targetGroupID
		if matchingMode != nil {
			rule.MatchingMode = MatchingMode(*matchingMode)
		}
		if createdBy != nil {
			rule.CreatedBy = *createdBy
		}
		if updatedBy != nil {
			rule.UpdatedBy = *updatedBy
		}
		if description != nil {
			rule.Description = *description
		}
		if operators != nil {
			rule.Operators = *operators
		}

		if len(nameRuleListJSON) > 0 {
			if err := json.Unmarshal(nameRuleListJSON, &rule.NameRuleList); err != nil {
				return nil, fmt.Errorf("unmarshal name_rule_list: %w", err)
			}
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// ExistsByPriority 检查优先级是否已存在
func (r *PgDeviceRuleRepository) ExistsByPriority(ctx context.Context, priority int, excludeID *uuid.UUID) (bool, error) {
	query := sq.Select("1").
		From("device_rules").
		Where(sq.Eq{"priority": priority, "enabled": true}).
		PlaceholderFormat(sq.Dollar)

	if excludeID != nil {
		query = query.Where(sq.NotEq{"id": *excludeID})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return false, fmt.Errorf("build exists SQL: %w", err)
	}

	var exists int
	err = r.pool.QueryRow(ctx, sql, args...).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check priority exists: %w", err)
	}

	return true, nil
}

// GetNextPriority 获取下一个可用的优先级
func (r *PgDeviceRuleRepository) GetNextPriority(ctx context.Context) (int, error) {
	query, args, err := sq.Select("COALESCE(MAX(priority), 0) + 1").
		From("device_rules").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build next priority SQL: %w", err)
	}

	var nextPriority int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&nextPriority); err != nil {
		return 0, fmt.Errorf("get next priority: %w", err)
	}

	return nextPriority, nil
}

// BatchUpdatePriority 批量更新优先级
// 使用单条 UPDATE ... CASE WHEN 语句原子更新，避免唯一约束冲突
func (r *PgDeviceRuleRepository) BatchUpdatePriority(ctx context.Context, items []RuleSortItem) error {
	if len(items) == 0 {
		return nil
	}

	// 构建: UPDATE device_rules SET priority = CASE WHEN id=$1 THEN $2 WHEN id=$3 THEN $4 ... END, updated_at = $n WHERE id IN ($1,$3,...)
	args := make([]interface{}, 0, len(items)*2+1)
	caseParts := make([]string, 0, len(items))
	inParts := make([]string, 0, len(items))

	for i, item := range items {
		id, err := uuid.Parse(item.ID)
		if err != nil {
			return fmt.Errorf("parse rule ID: %w", err)
		}

		idxID := i*2 + 1
		idxVal := i*2 + 2
		caseParts = append(caseParts, fmt.Sprintf("WHEN id = $%d THEN $%d", idxID, idxVal))
		inParts = append(inParts, fmt.Sprintf("$%d", idxID))
		args = append(args, id, item.Priority)
	}

	// 添加 updated_at 参数
	updatedAtIdx := len(items)*2 + 1
	args = append(args, time.Now())

	// 构建完整 SQL
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString("UPDATE device_rules SET priority = CASE ")
	for _, part := range caseParts {
		sqlBuilder.WriteString(part)
		sqlBuilder.WriteString(" ")
	}
	sqlBuilder.WriteString("END, updated_at = $")
	sqlBuilder.WriteString(strconv.Itoa(updatedAtIdx))
	sqlBuilder.WriteString(" WHERE id IN (")
	sqlBuilder.WriteString(strings.Join(inParts, ", "))
	sqlBuilder.WriteString(")")

	_, err := r.pool.Exec(ctx, sqlBuilder.String(), args...)
	if err != nil {
		return fmt.Errorf("batch update priority: %w", err)
	}

	return nil
}

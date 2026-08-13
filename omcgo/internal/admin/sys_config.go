package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// SysConfig represents a system configuration entry.
type SysConfig struct {
	ID          uuid.UUID `json:"id"`
	Category    string    `json:"category"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	ValueType   string    `json:"value_type"`
	Description string    `json:"desc"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateSysConfigRequest is the input for creating a config.
type CreateSysConfigRequest struct {
	Category    string `json:"category" binding:"required"`
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	Description string `json:"desc"`
	IsPublic    *bool  `json:"is_public"`
}

// UpdateSysConfigRequest is the input for updating a config.
type UpdateSysConfigRequest struct {
	Value       *string `json:"value"`
	Description *string `json:"desc"`
	IsPublic    *bool   `json:"is_public"`
}

// BatchItem 是单次批量 upsert 的 KV 项。value_type 可选，缺省 'string'。
type BatchItem struct {
	Key       string `json:"key" binding:"required"`
	Value     string `json:"value"`
	ValueType string `json:"value_type"`
}

// BatchUpdateSysConfigRequest 按 category 批量 upsert 配置（PRD config.md §5.2）。
type BatchUpdateSysConfigRequest struct {
	Category string      `json:"category" binding:"required"`
	Items    []BatchItem `json:"items" binding:"required,min=1"`
}

const (
	ConfigApplyStatusPending  = "pending"
	ConfigApplyStatusApplying = "applying"
	ConfigApplyStatusApplied  = "applied"
	ConfigApplyStatusFailed   = "failed"

	// ConfigApplySuccessScopeRuntimeApplied 表示目标成功后，运行态策略已经由目标系统应用。
	ConfigApplySuccessScopeRuntimeApplied = "runtime_applied"
	// ConfigApplySuccessScopeEventDelivered 只表示可靠消息已经发布，不代表所有消费实例已加载。
	ConfigApplySuccessScopeEventDelivered = "event_delivered"

	ConfigApplyObservationApplied       = "applied"
	ConfigApplyObservationFailed        = "failed"
	ConfigApplyObservationClaimFailed   = "claim_failed"
	ConfigApplyObservationExecuteFailed = "execute_failed"

	defaultApplyLeaseDuration      = time.Minute
	defaultApplyLeaseRenewInterval = 20 * time.Second
)

// ConfigApplyTarget 描述一个配置批次中独立可观察的运行态应用目标。
// ExpectedValue 与 ActualValue 只供受控后端存储和诊断使用，HTTP 响应不返回它们，
// 防止配置值（尤其是凭据）经状态接口泄露。
type ConfigApplyTarget struct {
	Target        string         `json:"target"`
	Status        string         `json:"status"`
	SuccessScope  string         `json:"success_scope"`
	Attempts      int            `json:"attempts"`
	AppliedAt     *time.Time     `json:"applied_at,omitempty"`
	LastError     string         `json:"last_error,omitempty"`
	ExpectedValue map[string]any `json:"-"`
	ActualValue   map[string]any `json:"-"`
}

// ConfigApplyBatch 表示一次已持久化的配置意图及其运行态应用进度。
type ConfigApplyBatch struct {
	ID            uuid.UUID           `json:"id"`
	Category      string              `json:"category"`
	ConfigVersion int64               `json:"config_version"`
	Status        string              `json:"status"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	Targets       []ConfigApplyTarget `json:"targets"`
}

// BatchUpsertResult 把数据库写入结果和持久化应用状态绑定，避免 HTTP 把“写入成功”误报为“已生效”。
type BatchUpsertResult struct {
	Updated int              `json:"updated"`
	Batch   ConfigApplyBatch `json:"batch"`
}

// --- SysConfigRepository ---

// SysConfigRepository defines the persistence interface for system configs.
type SysConfigRepository interface {
	Create(ctx context.Context, cfg *SysConfig) error
	GetByID(ctx context.Context, id uuid.UUID) (*SysConfig, error)
	GetByKey(ctx context.Context, category, key string) (*SysConfig, error)
	List(ctx context.Context, category string, publicOnly bool) ([]SysConfig, error)
	Update(ctx context.Context, cfg *SysConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
	BatchUpsert(ctx context.Context, category string, items []BatchItem) (int, error)
}

// SysConfigApplyRepository 是支持持久化应用状态的增强接口。保留 SysConfigRepository
// 可让历史调用方（例如 agent config writer）继续只依赖原有 BatchUpsert 契约。
type SysConfigApplyRepository interface {
	SysConfigRepository
	BatchUpsertWithApply(ctx context.Context, category string, items []BatchItem, targets []ConfigApplyTarget) (BatchUpsertResult, error)
	GetApplyBatch(ctx context.Context, id uuid.UUID) (*ConfigApplyBatch, error)
	ClaimApplyTarget(ctx context.Context, batchID uuid.UUID, target string) (*ConfigApplyWork, error)
	ClaimNextApplyTarget(ctx context.Context) (*ConfigApplyWork, error)
	RenewApplyTargetLease(ctx context.Context, work ConfigApplyWork, leaseExpiresAt time.Time) (bool, error)
	CompleteApplyTarget(ctx context.Context, work ConfigApplyWork, actualValue map[string]any, applyErr error) error
}

// SysConfigAtomicCategoryValidationRepository optionally lets a repository run
// cross-key validation and the complete batch write in one serialized storage
// transaction. Repositories without this capability keep the legacy service
// validation path for compatibility.
type SysConfigAtomicCategoryValidationRepository interface {
	BatchUpsertWithApplyValidated(
		ctx context.Context,
		category string,
		items []BatchItem,
		targets []ConfigApplyTarget,
		validator SysConfigCategoryValidator,
	) (BatchUpsertResult, error)
}

// ConfigApplyWork 是执行器独占领取的一项配置应用任务。
type ConfigApplyWork struct {
	Batch          ConfigApplyBatch
	Target         ConfigApplyTarget
	LeaseToken     uuid.UUID
	LeaseExpiresAt time.Time
}

// ConfigApplyHandler 执行一个目标的外部运行态更新。actualValue 只持久化给受控诊断，
// 不通过状态 HTTP API 暴露；返回 error 表示意图已保存但尚未生效。
type ConfigApplyHandler func(ctx context.Context, work ConfigApplyWork) (actualValue map[string]any, err error)

// ConfigApplyObservation 是一次配置应用尝试的低基数、可观测结果。BatchID 仅用于日志关联，
// Prometheus 指标只使用 category/target/result，避免把批次 ID 变成高基数标签。
type ConfigApplyObservation struct {
	BatchID  uuid.UUID
	Category string
	Target   string
	Attempts int
	Result   string
	Err      error
}

type ConfigApplyObserver func(ConfigApplyObservation)

// PgSysConfigRepository implements SysConfigRepository using PostgreSQL.
type PgSysConfigRepository struct {
	pool *pgxpool.Pool
}

var _ SysConfigRepository = (*PgSysConfigRepository)(nil)

// NewPgSysConfigRepository creates a new PgSysConfigRepository.
func NewPgSysConfigRepository(pool *pgxpool.Pool) *PgSysConfigRepository {
	return &PgSysConfigRepository{pool: pool}
}

func (r *PgSysConfigRepository) Create(ctx context.Context, cfg *SysConfig) error {
	now := time.Now()
	query, args, err := storage.Psql.Insert("sys_configs").
		Columns("category", "key", "value", "value_type", "description", "is_public", "created_at", "updated_at").
		Values(cfg.Category, cfg.Key, cfg.Value, cfg.ValueType, cfg.Description, cfg.IsPublic, now, now).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert config SQL: %w", err)
	}
	return r.pool.QueryRow(ctx, query, args...).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)
}

func (r *PgSysConfigRepository) GetByID(ctx context.Context, id uuid.UUID) (*SysConfig, error) {
	query, args, err := storage.Psql.Select("id", "category", "key", "value", "value_type", "description", "is_public", "created_at", "updated_at").
		From("sys_configs").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get config SQL: %w", err)
	}
	var c SysConfig
	err = r.pool.QueryRow(ctx, query, args...).Scan(&c.ID, &c.Category, &c.Key, &c.Value, &c.ValueType, &c.Description, &c.IsPublic, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get config: %w", err)
	}
	return &c, nil
}

func (r *PgSysConfigRepository) GetByKey(ctx context.Context, category, key string) (*SysConfig, error) {
	query, args, err := storage.Psql.Select("id", "category", "key", "value", "value_type", "description", "is_public", "created_at", "updated_at").
		From("sys_configs").Where(sq.Eq{"category": category, "key": key}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get config by key SQL: %w", err)
	}
	var c SysConfig
	err = r.pool.QueryRow(ctx, query, args...).Scan(&c.ID, &c.Category, &c.Key, &c.Value, &c.ValueType, &c.Description, &c.IsPublic, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get config by key: %w", err)
	}
	return &c, nil
}

func (r *PgSysConfigRepository) List(ctx context.Context, category string, publicOnly bool) ([]SysConfig, error) {
	builder := storage.Psql.Select("id", "category", "key", "value", "value_type", "description", "is_public", "created_at", "updated_at").
		From("sys_configs").OrderBy("category ASC", "key ASC")

	where := sq.And{}
	if category != "" {
		where = append(where, sq.Eq{"category": category})
	}
	if publicOnly {
		where = append(where, sq.Eq{"is_public": true})
	}
	if len(where) > 0 {
		builder = builder.Where(where)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list configs SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list configs: %w", err)
	}
	defer rows.Close()

	var items []SysConfig
	for rows.Next() {
		var c SysConfig
		if err := rows.Scan(&c.ID, &c.Category, &c.Key, &c.Value, &c.ValueType, &c.Description, &c.IsPublic, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan config: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *PgSysConfigRepository) Update(ctx context.Context, cfg *SysConfig) error {
	now := time.Now()
	builder := storage.Psql.Update("sys_configs").Set("updated_at", now)
	if cfg.Value != "" {
		builder = builder.Set("value", cfg.Value)
	}
	builder = builder.Set("description", cfg.Description)
	builder = builder.Set("is_public", cfg.IsPublic)

	query, args, err := builder.Where(sq.Eq{"id": cfg.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build update config SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update config: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgSysConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM sys_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete config: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// BatchUpsert 在单次事务内批量 upsert sys_configs 的 (category, key) → value 映射。
// 借助 DDL 中的 UNIQUE(category, key) 约束做 ON CONFLICT 升级。返回成功条数。
func (r *PgSysConfigRepository) BatchUpsert(ctx context.Context, category string, items []BatchItem) (int, error) {
	result, err := r.BatchUpsertWithApply(ctx, category, items, []ConfigApplyTarget{{Target: "runtime_cache", Status: ConfigApplyStatusApplied}})
	return result.Updated, err
}

// BatchUpsertWithApply 在同一 PostgreSQL 事务中保存 KV、递增分类版本并创建应用批次。
// 调用方传入的 target 状态代表“需要外部应用”或“仅本地缓存已失效”；后续执行器只会推进
// pending/failed 目标，因而写入提交与应用意图不可分离。
func (r *PgSysConfigRepository) BatchUpsertWithApply(ctx context.Context, category string, items []BatchItem, targets []ConfigApplyTarget) (BatchUpsertResult, error) {
	return r.batchUpsertWithApply(ctx, category, items, targets, nil)
}

// BatchUpsertWithApplyValidated serializes writers for one category on its
// config_apply_versions row, validates the merged final state, and persists the
// KV values plus apply intent in the same PostgreSQL transaction.
func (r *PgSysConfigRepository) BatchUpsertWithApplyValidated(
	ctx context.Context,
	category string,
	items []BatchItem,
	targets []ConfigApplyTarget,
	validator SysConfigCategoryValidator,
) (BatchUpsertResult, error) {
	return r.batchUpsertWithApply(ctx, category, items, targets, validator)
}

func (r *PgSysConfigRepository) batchUpsertWithApply(
	ctx context.Context,
	category string,
	items []BatchItem,
	targets []ConfigApplyTarget,
	validator SysConfigCategoryValidator,
) (BatchUpsertResult, error) {
	if len(items) == 0 {
		return BatchUpsertResult{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return BatchUpsertResult{}, fmt.Errorf("begin batch upsert tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `
INSERT INTO config_apply_versions (category, config_version, updated_at)
VALUES ($1, 0, NOW())
ON CONFLICT (category) DO NOTHING
`, category); err != nil {
		return BatchUpsertResult{}, fmt.Errorf("ensure config apply version for %s: %w", category, err)
	}
	var lockedVersion int64
	if err := tx.QueryRow(ctx, `
SELECT config_version
FROM config_apply_versions
WHERE category = $1
FOR UPDATE
`, category).Scan(&lockedVersion); err != nil {
		return BatchUpsertResult{}, fmt.Errorf("lock config category %s: %w", category, err)
	}

	if validator != nil {
		rows, err := tx.Query(ctx, `
SELECT key, value
FROM sys_configs
WHERE category = $1
`, category)
		if err != nil {
			return BatchUpsertResult{}, fmt.Errorf("list locked sys_config category %s: %w", category, err)
		}
		values := make(map[string]string, len(items))
		for rows.Next() {
			var key, value string
			if err := rows.Scan(&key, &value); err != nil {
				rows.Close()
				return BatchUpsertResult{}, fmt.Errorf("scan locked sys_config category %s: %w", category, err)
			}
			values[key] = value
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return BatchUpsertResult{}, fmt.Errorf("iterate locked sys_config category %s: %w", category, err)
		}
		rows.Close()
		for _, item := range items {
			values[item.Key] = item.Value
		}
		if err := validator(values); err != nil {
			return BatchUpsertResult{}, err
		}
	}

	count := 0
	for _, item := range items {
		valueType := item.ValueType
		if valueType == "" {
			valueType = "string"
		}
		isPublic := isPublicSysConfig(category, item.Key)
		// 仅 update value、派生后的 public 分类和 updated_at；value_type / desc
		// 不随通用批量保存改写。public 分类由代码白名单全量纠正历史脏标记。
		_, err := tx.Exec(ctx, `
INSERT INTO sys_configs (category, key, value, value_type, is_public)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (category, key) DO UPDATE
SET value = EXCLUDED.value,
    is_public = EXCLUDED.is_public,
    updated_at = NOW()
`, category, item.Key, item.Value, valueType, isPublic)
		if err != nil {
			return BatchUpsertResult{}, fmt.Errorf("upsert %s.%s: %w", category, item.Key, err)
		}
		count++
	}

	var configVersion int64
	if err := tx.QueryRow(ctx, `
UPDATE config_apply_versions
SET config_version = config_version + 1,
    updated_at = NOW()
WHERE category = $1
RETURNING config_version
`, category).Scan(&configVersion); err != nil {
		return BatchUpsertResult{}, fmt.Errorf("increment config apply version for %s: %w", category, err)
	}

	if len(targets) == 0 {
		targets = []ConfigApplyTarget{{Target: "runtime_cache", Status: ConfigApplyStatusApplied}}
	}
	batchStatus := summarizeConfigApplyStatus(targets)
	var batch ConfigApplyBatch
	if err := tx.QueryRow(ctx, `
INSERT INTO config_apply_batches (category, config_version, status)
VALUES ($1, $2, $3)
RETURNING id, created_at, updated_at
`, category, configVersion, batchStatus).Scan(&batch.ID, &batch.CreatedAt, &batch.UpdatedAt); err != nil {
		return BatchUpsertResult{}, fmt.Errorf("create config apply batch for %s: %w", category, err)
	}
	batch.Category = category
	batch.ConfigVersion = configVersion
	batch.Status = batchStatus
	batch.Targets = make([]ConfigApplyTarget, 0, len(targets))
	expectedValue, err := json.Marshal(batchItemsToApplyState(category, items))
	if err != nil {
		return BatchUpsertResult{}, fmt.Errorf("marshal config apply expected value: %w", err)
	}
	for _, target := range targets {
		if target.Target == "" {
			return BatchUpsertResult{}, fmt.Errorf("create config apply target: target is required")
		}
		if target.SuccessScope == "" {
			target.SuccessScope = configApplySuccessScope(target.Target)
		}
		if target.Status == "" {
			target.Status = ConfigApplyStatusPending
		}
		var appliedAt *time.Time
		if target.Status == ConfigApplyStatusApplied {
			now := time.Now().UTC()
			appliedAt = &now
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO config_apply_targets (batch_id, category, target, status, attempts, applied_at, last_error, expected_value, actual_value)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, '{}'::jsonb)
`, batch.ID, category, target.Target, target.Status, target.Attempts, appliedAt, target.LastError, expectedValue); err != nil {
			return BatchUpsertResult{}, fmt.Errorf("create config apply target %s: %w", target.Target, err)
		}
		target.AppliedAt = appliedAt
		batch.Targets = append(batch.Targets, target)
	}
	if err := tx.Commit(ctx); err != nil {
		return BatchUpsertResult{}, fmt.Errorf("commit batch upsert: %w", err)
	}
	return BatchUpsertResult{Updated: count, Batch: batch}, nil
}

func batchItemsToApplyState(category string, items []BatchItem) map[string]string {
	values := make(map[string]string, len(items))
	for _, item := range items {
		if isSecretSysConfig(category, item.Key) {
			values[item.Key] = "[REDACTED]"
			continue
		}
		values[item.Key] = item.Value
	}
	return values
}

// GetApplyBatch 读取批次及其目标状态。expected/actual 不映射到 HTTP DTO，避免值泄露。
func (r *PgSysConfigRepository) GetApplyBatch(ctx context.Context, id uuid.UUID) (*ConfigApplyBatch, error) {
	batch := &ConfigApplyBatch{}
	err := r.pool.QueryRow(ctx, `
SELECT id, category, config_version, status, created_at, updated_at
FROM config_apply_batches WHERE id = $1
`, id).Scan(&batch.ID, &batch.Category, &batch.ConfigVersion, &batch.Status, &batch.CreatedAt, &batch.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get config apply batch: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
SELECT target, status, attempts, applied_at, last_error
FROM config_apply_targets WHERE batch_id = $1 ORDER BY target ASC
`, id)
	if err != nil {
		return nil, fmt.Errorf("list config apply targets: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var target ConfigApplyTarget
		if err := rows.Scan(&target.Target, &target.Status, &target.Attempts, &target.AppliedAt, &target.LastError); err != nil {
			return nil, fmt.Errorf("scan config apply target: %w", err)
		}
		target.SuccessScope = configApplySuccessScope(target.Target)
		batch.Targets = append(batch.Targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate config apply targets: %w", err)
	}
	return batch, nil
}

// ClaimApplyTarget 使用行锁把一个指定目标从 pending/failed 原子变为 applying。
// 并发 app 实例只有一个可以获得任务；没任务时返回 (nil, nil)。
func (r *PgSysConfigRepository) ClaimApplyTarget(ctx context.Context, batchID uuid.UUID, target string) (*ConfigApplyWork, error) {
	return r.claimApplyTarget(ctx, `
WITH candidate AS (
    SELECT t.id FROM config_apply_targets AS t
    WHERE t.batch_id = $1 AND t.target = $2 AND t.status IN ('pending', 'failed')
      AND NOT EXISTS (
          SELECT 1 FROM config_apply_targets AS active
          WHERE active.category = t.category AND active.target = t.target
            AND active.status = 'applying'
      )
    FOR UPDATE SKIP LOCKED
)
UPDATE config_apply_targets AS t
SET status = 'applying', attempts = t.attempts + 1, updated_at = NOW(), last_error = '',
    lease_token = gen_random_uuid(), lease_expires_at = NOW() + INTERVAL '1 minute'
FROM candidate, config_apply_batches AS b
WHERE t.id = candidate.id AND b.id = t.batch_id
RETURNING b.id, b.category, b.config_version, b.status, b.created_at, b.updated_at,
          t.target, t.status, t.attempts, t.applied_at, t.last_error, t.lease_token, t.lease_expires_at
`, batchID, target)
}

// ClaimNextApplyTarget 从 due 的 pending/failed 任务中领取一项。指数退避由 attempts
// 决定，最多 5 分钟；FOR UPDATE SKIP LOCKED 使多实例可以安全并行扫描。
func (r *PgSysConfigRepository) ClaimNextApplyTarget(ctx context.Context) (*ConfigApplyWork, error) {
	return r.claimApplyTarget(ctx, `
WITH candidate AS (
    SELECT t.id FROM config_apply_targets AS t
    WHERE (
        t.status = 'applying'
        AND (t.lease_expires_at IS NULL OR t.lease_expires_at <= NOW())
    ) OR (
        t.status IN ('pending', 'failed')
        AND t.updated_at <= NOW() - (LEAST(POWER(2, t.attempts), 300) * INTERVAL '1 second')
        AND NOT EXISTS (
            SELECT 1 FROM config_apply_targets AS active
            WHERE active.category = t.category AND active.target = t.target
              AND active.status = 'applying'
        )
    )
    ORDER BY CASE WHEN t.status = 'applying' THEN 0 ELSE 1 END, t.updated_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE config_apply_targets AS t
SET status = 'applying', attempts = t.attempts + 1, updated_at = NOW(), last_error = '',
    lease_token = gen_random_uuid(), lease_expires_at = NOW() + INTERVAL '1 minute'
FROM candidate, config_apply_batches AS b
WHERE t.id = candidate.id AND b.id = t.batch_id
RETURNING b.id, b.category, b.config_version, b.status, b.created_at, b.updated_at,
          t.target, t.status, t.attempts, t.applied_at, t.last_error, t.lease_token, t.lease_expires_at
`)
}

func (r *PgSysConfigRepository) claimApplyTarget(ctx context.Context, query string, args ...any) (*ConfigApplyWork, error) {
	var work ConfigApplyWork
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&work.Batch.ID, &work.Batch.Category, &work.Batch.ConfigVersion, &work.Batch.Status, &work.Batch.CreatedAt, &work.Batch.UpdatedAt,
		&work.Target.Target, &work.Target.Status, &work.Target.Attempts, &work.Target.AppliedAt, &work.Target.LastError, &work.LeaseToken, &work.LeaseExpiresAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uq_config_apply_target_running" {
			return nil, nil
		}
		return nil, fmt.Errorf("claim config apply target: %w", err)
	}
	return &work, nil
}

func (r *PgSysConfigRepository) RenewApplyTargetLease(ctx context.Context, work ConfigApplyWork, leaseExpiresAt time.Time) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
UPDATE config_apply_targets
SET lease_expires_at = $4, updated_at = NOW()
WHERE batch_id = $1 AND target = $2 AND status = 'applying' AND lease_token = $3
`, work.Batch.ID, work.Target.Target, work.LeaseToken, leaseExpiresAt)
	if err != nil {
		return false, fmt.Errorf("renew config apply target %s lease: %w", work.Target.Target, err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PgSysConfigRepository) CompleteApplyTarget(ctx context.Context, work ConfigApplyWork, actualValue map[string]any, applyErr error) error {
	actual, err := json.Marshal(actualValue)
	if err != nil {
		return fmt.Errorf("marshal config apply actual value: %w", err)
	}
	status := ConfigApplyStatusApplied
	lastError := ""
	var appliedAt *time.Time
	if applyErr != nil {
		status = ConfigApplyStatusFailed
		lastError = applyErr.Error()
	} else {
		now := time.Now().UTC()
		appliedAt = &now
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin complete config apply target: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
UPDATE config_apply_targets
SET status = $3, applied_at = $4, last_error = $5, actual_value = $6::jsonb,
    lease_token = NULL, lease_expires_at = NULL, updated_at = NOW()
WHERE batch_id = $1 AND target = $2 AND status = 'applying' AND lease_token = $7
`, work.Batch.ID, work.Target.Target, status, appliedAt, lastError, actual, work.LeaseToken)
	if err != nil {
		return fmt.Errorf("complete config apply target %s: %w", work.Target.Target, err)
	}
	// The lease may have expired and been reclaimed while a slow external call
	// was still running. In that case its result is stale and must not alter the
	// newer worker's status or batch summary.
	if tag.RowsAffected() == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE config_apply_batches AS b
SET status = CASE
    WHEN EXISTS (SELECT 1 FROM config_apply_targets t WHERE t.batch_id = b.id AND t.status = 'failed') THEN 'failed'
    WHEN EXISTS (SELECT 1 FROM config_apply_targets t WHERE t.batch_id = b.id AND t.status IN ('pending', 'applying')) THEN 'pending'
    ELSE 'applied'
END,
updated_at = NOW()
WHERE b.id = $1
`, work.Batch.ID); err != nil {
		return fmt.Errorf("summarize config apply batch: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit config apply target completion: %w", err)
	}
	return nil
}

func summarizeConfigApplyStatus(targets []ConfigApplyTarget) string {
	for _, target := range targets {
		if target.Status == ConfigApplyStatusFailed {
			return ConfigApplyStatusFailed
		}
		if target.Status == ConfigApplyStatusPending || target.Status == ConfigApplyStatusApplying {
			return ConfigApplyStatusPending
		}
	}
	return ConfigApplyStatusApplied
}

// --- SysConfigService ---

// SysConfigSavedHook 是 BatchUpsert 提交成功后触发的回调。
// 典型用途：让 in-memory policy cache（如 SecurityPolicy / PeriodicSyncPolicy）
// 在配置保存后立刻失效，避免等 30s TTL 自然过期。
//
// 多个 hook 按注册顺序串行调用；任一 hook panic 不影响其它 hook（recover）。
// hook 收到的 category 是本次保存的 category 名，回调侧自行判定是否相关。
type SysConfigSavedHook func(ctx context.Context, category string)

// SysConfigValidator 是 BatchUpsert 提交前 (category, key) 维度的值校验回调。
// 仅在进程启动 wiring 阶段经 RegisterValidator 注册。校验失败返非 nil error，
// BatchUpsert 立即中止（整批不落库），错误包 commonerrors.ErrInvalidInput 让
// handler.HTTPStatusFromError 自动映射成 HTTP 400。
//
// 典型用途：minio public_endpoint 等业务语义强的 KV 在 sys_config 通用 KV 模型上
// 做格式守门（如 host[:port] 无 scheme，issue #548 切片 2 D 后端）。
type SysConfigValidator func(value string) error

// SysConfigCategoryValidator validates the final value set for one category.
// The service merges persisted values with the submitted batch before calling
// it, so cross-key contracts cannot be bypassed by a partial API request.
type SysConfigCategoryValidator func(values map[string]string) error

// validatorKey 用 (category, key) 锁定一个 validator。
type validatorKey struct {
	Category string
	Key      string
}

// SysConfigService provides business logic for system configuration.
type SysConfigService struct {
	repo                    SysConfigRepository
	hooks                   []SysConfigSavedHook
	validators              map[validatorKey]SysConfigValidator
	categoryValidators      map[string]SysConfigCategoryValidator
	appliers                map[string]map[string]ConfigApplyHandler
	applyObserver           ConfigApplyObserver
	applyLeaseDuration      time.Duration
	applyLeaseRenewInterval time.Duration
}

// defaultConfigApplyTargets 覆盖已知会驱动外部运行态的分类。即使应用进程在
// 启动接线尚未完成前接到请求，也必须先留下 pending 意图，不能把它伪装成 applied。
var defaultConfigApplyTargets = map[string][]string{
	"acs_transfer":    {"acs_transfer_event_delivery"},
	"minio.retention": {"minio_raw_file_lifecycle"},
	"pm.retention":    {"pm_retention"},
	"storage":         {"alarm_history_retention", "minio_presign_endpoint"},
}

// NewSysConfigService creates a new SysConfigService.
func NewSysConfigService(repo SysConfigRepository) *SysConfigService {
	return &SysConfigService{
		repo:                    repo,
		applyLeaseDuration:      defaultApplyLeaseDuration,
		applyLeaseRenewInterval: defaultApplyLeaseRenewInterval,
	}
}

// RegisterSavedHook 注册一个 BatchUpsert 后回调。仅在进程启动 wiring 阶段调用，
// 无并发保护（运行期不再修改 hooks 切片）。
func (s *SysConfigService) RegisterSavedHook(h SysConfigSavedHook) {
	if h == nil {
		return
	}
	s.hooks = append(s.hooks, h)
}

// RegisterApplyHandler 注册一个会产生外部副作用的目标。仅 wiring 阶段调用；
// 后续保存会先持久化 pending 目标，再由即时/后台执行器确认 applied 或 failed。
func (s *SysConfigService) RegisterApplyHandler(category, target string, handler ConfigApplyHandler) {
	if category == "" || target == "" || handler == nil {
		return
	}
	if s.appliers == nil {
		s.appliers = make(map[string]map[string]ConfigApplyHandler)
	}
	if s.appliers[category] == nil {
		s.appliers[category] = make(map[string]ConfigApplyHandler)
	}
	s.appliers[category][target] = handler
}

// SetApplyObserver 注入配置应用结果观察器。观察器只负责日志/指标，不参与业务状态推进；
// 即使观察器异常也不能中断配置应用。
func (s *SysConfigService) SetApplyObserver(observer ConfigApplyObserver) {
	s.applyObserver = observer
}

// RegisterValidator 注册 (category, key) 的值校验器。仅在进程启动 wiring 阶段调用。
// 同一 (category, key) 重复注册以后注册的为准（不报错，便于调试期反复注入）。
// fn 为 nil 等价于"删除"该 key 的校验。
func (s *SysConfigService) RegisterValidator(category, key string, fn SysConfigValidator) {
	if s.validators == nil {
		s.validators = make(map[validatorKey]SysConfigValidator)
	}
	k := validatorKey{Category: category, Key: key}
	if fn == nil {
		delete(s.validators, k)
		return
	}
	s.validators[k] = fn
}

// RegisterCategoryValidator registers a cross-key validator for a category.
// A nil function removes the validator.
func (s *SysConfigService) RegisterCategoryValidator(category string, fn SysConfigCategoryValidator) {
	if s.categoryValidators == nil {
		s.categoryValidators = make(map[string]SysConfigCategoryValidator)
	}
	if fn == nil {
		delete(s.categoryValidators, category)
		return
	}
	s.categoryValidators[category] = fn
}

// Create creates a new config entry.
func (s *SysConfigService) Create(context.Context, CreateSysConfigRequest) (*SysConfig, error) {
	return nil, directSysConfigMutationError()
}

// Get returns a config by ID.
func (s *SysConfigService) Get(ctx context.Context, id uuid.UUID) (*SysConfig, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns configs, optionally filtered by category.
func (s *SysConfigService) List(ctx context.Context, category string, publicOnly bool) ([]SysConfig, error) {
	return s.repo.List(ctx, category, publicOnly)
}

// ListPublic 返回无需认证即可读取的配置项。
// 公开范围只由代码白名单决定，不信任数据库中的历史 is_public 标记。
func (s *SysConfigService) ListPublic(ctx context.Context, category string) ([]SysConfig, error) {
	items, err := s.repo.List(ctx, category, false)
	if err != nil {
		return nil, err
	}
	publicItems := make([]SysConfig, 0, len(items))
	for _, item := range items {
		if isPublicSysConfig(item.Category, item.Key) {
			publicItems = append(publicItems, item)
		}
	}
	return publicItems, nil
}

// Update updates a config.
func (s *SysConfigService) Update(context.Context, uuid.UUID, UpdateSysConfigRequest) (*SysConfig, error) {
	return nil, directSysConfigMutationError()
}

// Delete deletes a config.
func (s *SysConfigService) Delete(context.Context, uuid.UUID) error {
	return directSysConfigMutationError()
}

func directSysConfigMutationError() error {
	return fmt.Errorf("%w: direct sys_config mutation is disabled; use category batch update", commonerrors.ErrInvalidInput)
}

// BatchUpsert 按 PRD config.md §5.2 把整个 category 的 KV 批量写入。
// 先跑已注册的 SysConfigValidator（任一失败包 ErrInvalidInput → handler 翻 HTTP 400，整批不落库），
// 再 repo.BatchUpsert，commit 成功后顺序触发 SysConfigSavedHook（hook 异常被 recover 隔离）。
func (s *SysConfigService) BatchUpsert(ctx context.Context, req BatchUpdateSysConfigRequest) (int, error) {
	result, err := s.BatchUpsertWithResult(ctx, req)
	return result.Updated, err
}

// BatchUpsertWithResult 返回已提交的配置意图及其应用状态。没有外部应用器的分类会
// 创建 runtime_cache 目标并在保存后触发本地 cache invalidation，因此状态为 applied。
func (s *SysConfigService) BatchUpsertWithResult(ctx context.Context, req BatchUpdateSysConfigRequest) (BatchUpsertResult, error) {
	req.Items = preserveBlankSecrets(req.Category, req.Items)
	if len(req.Items) == 0 {
		return BatchUpsertResult{}, fmt.Errorf("%w: no writable sys_config items", commonerrors.ErrInvalidInput)
	}
	if err := s.runValidators(req.Category, req.Items); err != nil {
		return BatchUpsertResult{}, err
	}
	targets := s.configApplyTargetsForCategory(req.Category)
	if validator, ok := s.categoryValidators[req.Category]; ok {
		if atomicRepo, atomicOK := s.repo.(SysConfigAtomicCategoryValidationRepository); atomicOK {
			result, err := atomicRepo.BatchUpsertWithApplyValidated(
				ctx,
				req.Category,
				req.Items,
				targets,
				func(values map[string]string) error {
					if err := validator(values); err != nil {
						return fmt.Errorf("%w: sys_config %s: %v", commonerrors.ErrInvalidInput, req.Category, err)
					}
					return nil
				},
			)
			if err != nil {
				return BatchUpsertResult{}, err
			}
			s.fireSavedHooks(ctx, req.Category)
			return s.refreshCommittedApplyBatch(ctx, result), nil
		}
		if err := s.runCategoryValidator(ctx, req.Category, req.Items); err != nil {
			return BatchUpsertResult{}, err
		}
	}
	applyRepo, ok := s.repo.(SysConfigApplyRepository)
	if !ok {
		n, err := s.repo.BatchUpsert(ctx, req.Category, req.Items)
		if err != nil {
			return BatchUpsertResult{}, err
		}
		s.fireSavedHooks(ctx, req.Category)
		now := time.Now().UTC()
		return BatchUpsertResult{Updated: n, Batch: ConfigApplyBatch{
			ID: uuid.New(), Category: req.Category, ConfigVersion: 0, Status: summarizeConfigApplyStatus(targets),
			CreatedAt: now, UpdatedAt: now,
			Targets: targets,
		}}, nil
	}
	result, err := applyRepo.BatchUpsertWithApply(ctx, req.Category, req.Items, targets)
	if err != nil {
		return BatchUpsertResult{}, err
	}
	s.fireSavedHooks(ctx, req.Category)
	return s.refreshCommittedApplyBatch(ctx, result), nil
}

func (s *SysConfigService) refreshCommittedApplyBatch(ctx context.Context, result BatchUpsertResult) BatchUpsertResult {
	if result.Batch.Status == ConfigApplyStatusPending {
		// The intent and retry target are already committed. Immediate execution
		// is best effort: returning an HTTP failure here would hide the durable
		// batch ID and encourage the caller to create a duplicate version.
		if applyRepo, ok := s.repo.(SysConfigApplyRepository); ok {
			_ = s.ApplyBatch(ctx, result.Batch.ID)
			if refreshed, err := applyRepo.GetApplyBatch(ctx, result.Batch.ID); err == nil {
				result.Batch = *refreshed
			}
		}
	}
	return result
}

func (s *SysConfigService) configApplyTargetsForCategory(category string) []ConfigApplyTarget {
	registered := s.appliers[category]
	defaults := defaultConfigApplyTargets[category]
	if len(registered) > 0 || len(defaults) > 0 {
		nameSet := make(map[string]struct{}, len(registered)+len(defaults))
		for _, target := range defaults {
			nameSet[target] = struct{}{}
		}
		for target := range registered {
			nameSet[target] = struct{}{}
		}
		names := make([]string, 0, len(nameSet))
		for target := range nameSet {
			names = append(names, target)
		}
		sort.Strings(names)
		targets := make([]ConfigApplyTarget, 0, len(names))
		for _, target := range names {
			targets = append(targets, ConfigApplyTarget{
				Target:       target,
				Status:       ConfigApplyStatusPending,
				SuccessScope: configApplySuccessScope(target),
			})
		}
		return targets
	}
	now := time.Now().UTC()
	return []ConfigApplyTarget{{
		Target: "runtime_cache", Status: ConfigApplyStatusApplied,
		SuccessScope: ConfigApplySuccessScopeRuntimeApplied, AppliedAt: &now,
	}}
}

func configApplySuccessScope(target string) string {
	if target == "acs_transfer_event_delivery" {
		return ConfigApplySuccessScopeEventDelivered
	}
	return ConfigApplySuccessScopeRuntimeApplied
}

// ApplyBatch 立即执行一个保存后的 batch。处理器失败会转为持久化 failed 状态，而不是
// 让 HTTP 保存失败并误导调用方“配置没有保存”。
func (s *SysConfigService) ApplyBatch(ctx context.Context, id uuid.UUID) error {
	applyRepo, ok := s.repo.(SysConfigApplyRepository)
	if !ok {
		return nil
	}
	batch, err := applyRepo.GetApplyBatch(ctx, id)
	if err != nil {
		return err
	}
	for _, target := range batch.Targets {
		if target.Status != ConfigApplyStatusPending && target.Status != ConfigApplyStatusFailed {
			continue
		}
		work, err := applyRepo.ClaimApplyTarget(ctx, id, target.Target)
		if err != nil {
			return err
		}
		if work == nil {
			continue
		}
		if err := s.executeApplyWork(ctx, applyRepo, *work); err != nil {
			return err
		}
	}
	return nil
}

// RetryPendingApplyOnce 供后台 ticker 调用。无 due 任务返回 (false, nil)。
func (s *SysConfigService) RetryPendingApplyOnce(ctx context.Context) (bool, error) {
	applyRepo, ok := s.repo.(SysConfigApplyRepository)
	if !ok {
		return false, nil
	}
	work, err := applyRepo.ClaimNextApplyTarget(ctx)
	if err != nil {
		s.observeApply(ConfigApplyObservation{Result: ConfigApplyObservationClaimFailed, Err: err})
		return false, err
	}
	if work == nil {
		return false, nil
	}
	return true, s.executeApplyWork(ctx, applyRepo, *work)
}

func (s *SysConfigService) executeApplyWork(ctx context.Context, repo SysConfigApplyRepository, work ConfigApplyWork) error {
	handler := s.appliers[work.Batch.Category][work.Target.Target]
	if handler == nil {
		applyErr := fmt.Errorf("no config apply handler registered for %s/%s", work.Batch.Category, work.Target.Target)
		completeErr := repo.CompleteApplyTarget(ctx, work, nil, applyErr)
		if completeErr != nil {
			s.observeApplyWork(work, ConfigApplyObservationExecuteFailed, completeErr)
			return completeErr
		}
		s.observeApplyWork(work, ConfigApplyObservationFailed, applyErr)
		return nil
	}

	leaseDuration := s.applyLeaseDuration
	if leaseDuration <= 0 {
		leaseDuration = defaultApplyLeaseDuration
	}
	renewInterval := s.applyLeaseRenewInterval
	if renewInterval <= 0 || renewInterval >= leaseDuration {
		renewInterval = leaseDuration / 3
	}
	if renewInterval <= 0 {
		renewInterval = defaultApplyLeaseRenewInterval
	}

	type applyResult struct {
		actual map[string]any
		err    error
	}
	handlerCtx, cancelHandler := context.WithCancel(ctx)
	defer cancelHandler()
	resultCh := make(chan applyResult, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				resultCh <- applyResult{err: fmt.Errorf("config apply handler panic for %s/%s: %v", work.Batch.Category, work.Target.Target, recovered)}
			}
		}()
		actual, err := handler(handlerCtx, work)
		resultCh <- applyResult{actual: actual, err: err}
	}()

	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()
	for {
		select {
		case result := <-resultCh:
			if err := repo.CompleteApplyTarget(ctx, work, result.actual, result.err); err != nil {
				s.observeApplyWork(work, ConfigApplyObservationExecuteFailed, err)
				return err
			}
			if result.err != nil {
				s.observeApplyWork(work, ConfigApplyObservationFailed, result.err)
			} else {
				s.observeApplyWork(work, ConfigApplyObservationApplied, nil)
			}
			return nil
		case <-ctx.Done():
			err := fmt.Errorf("config apply context ended for %s/%s: %w", work.Batch.Category, work.Target.Target, ctx.Err())
			s.observeApplyWork(work, ConfigApplyObservationExecuteFailed, err)
			return err
		case <-ticker.C:
			leaseExpiresAt := time.Now().UTC().Add(leaseDuration)
			renewed, err := repo.RenewApplyTargetLease(ctx, work, leaseExpiresAt)
			if err != nil {
				cancelHandler()
				err = fmt.Errorf("renew config apply lease for %s/%s: %w", work.Batch.Category, work.Target.Target, err)
				s.observeApplyWork(work, ConfigApplyObservationExecuteFailed, err)
				return err
			}
			if !renewed {
				cancelHandler()
				err := fmt.Errorf("config apply lease lost for %s/%s", work.Batch.Category, work.Target.Target)
				s.observeApplyWork(work, ConfigApplyObservationExecuteFailed, err)
				return err
			}
		}
	}
}

func (s *SysConfigService) observeApplyWork(work ConfigApplyWork, result string, err error) {
	s.observeApply(ConfigApplyObservation{
		BatchID:  work.Batch.ID,
		Category: work.Batch.Category,
		Target:   work.Target.Target,
		Attempts: work.Target.Attempts,
		Result:   result,
		Err:      err,
	})
}

func (s *SysConfigService) observeApply(observation ConfigApplyObservation) {
	if s.applyObserver == nil {
		return
	}
	defer func() { _ = recover() }()
	s.applyObserver(observation)
}

// StartApplyRetry 启动单进程重试循环并返回停止函数。数据库级 SKIP LOCKED 保证多实例
// 同时运行时同一目标只会被一台实例领取。
func (s *SysConfigService) StartApplyRetry(interval time.Duration) func() {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_, _ = s.RetryPendingApplyOnce(context.Background())
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}

// GetApplyBatch 查询已经持久化的应用状态。只有生产 Pg repository 支持此能力；
// 旧的窄仓储适配器不会暴露伪造状态。
func (s *SysConfigService) GetApplyBatch(ctx context.Context, id uuid.UUID) (*ConfigApplyBatch, error) {
	applyRepo, ok := s.repo.(SysConfigApplyRepository)
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	return applyRepo.GetApplyBatch(ctx, id)
}

// runValidators 对每个 BatchItem 找 (category, key) 匹配 validator 并执行。
// 找不到 validator 视作"通用 KV，无需校验"，直接通过。任一失败立即返错（短路）。
func (s *SysConfigService) runValidators(category string, items []BatchItem) error {
	if len(s.validators) == 0 {
		return nil
	}
	for _, item := range items {
		v, ok := s.validators[validatorKey{Category: category, Key: item.Key}]
		if !ok {
			continue
		}
		if err := v(item.Value); err != nil {
			// 包 ErrInvalidInput 让 HTTPStatusFromError 映射成 400。
			return fmt.Errorf("%w: sys_config %s.%s: %v", commonerrors.ErrInvalidInput, category, item.Key, err)
		}
	}
	return nil
}

func (s *SysConfigService) runCategoryValidator(ctx context.Context, category string, items []BatchItem) error {
	validator, ok := s.categoryValidators[category]
	if !ok {
		return nil
	}

	current, err := s.repo.List(ctx, category, false)
	if err != nil {
		return fmt.Errorf("list sys_config category %s for validation: %w", category, err)
	}
	values := make(map[string]string, len(current)+len(items))
	for _, item := range current {
		values[item.Key] = item.Value
	}
	for _, item := range items {
		values[item.Key] = item.Value
	}
	if err := validator(values); err != nil {
		return fmt.Errorf("%w: sys_config %s: %v", commonerrors.ErrInvalidInput, category, err)
	}
	return nil
}

// fireSavedHooks 串行调用注册的 SavedHook。单 hook panic 不影响后续 hook 与 caller。
func (s *SysConfigService) fireSavedHooks(ctx context.Context, category string) {
	for _, h := range s.hooks {
		func(hook SysConfigSavedHook) {
			defer func() {
				_ = recover() // hook 失败仅丢弃，保存动作已落库
			}()
			hook(ctx, category)
		}(h)
	}
}

package admin

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	if len(items) == 0 {
		return 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin batch upsert tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	count := 0
	for _, item := range items {
		valueType := item.ValueType
		if valueType == "" {
			valueType = "string"
		}
		// 仅 update value + updated_at；value_type / desc / is_public 仅在新增时落库，
		// 已存在的行不会被覆盖到这些维度（防止业务侧误传 value_type 把字段语义破坏）。
		_, err := tx.Exec(ctx, `
INSERT INTO sys_configs (category, key, value, value_type)
VALUES ($1, $2, $3, $4)
ON CONFLICT (category, key) DO UPDATE
SET value = EXCLUDED.value, updated_at = NOW()
`, category, item.Key, item.Value, valueType)
		if err != nil {
			return count, fmt.Errorf("upsert %s.%s: %w", category, item.Key, err)
		}
		count++
	}
	if err := tx.Commit(ctx); err != nil {
		return count, fmt.Errorf("commit batch upsert: %w", err)
	}
	return count, nil
}

// --- SysConfigService ---

// SysConfigSavedHook 是 BatchUpsert 提交成功后触发的回调。
// 典型用途：让 in-memory policy cache（如 SecurityPolicy / PeriodicSyncPolicy）
// 在配置保存后立刻失效，避免等 30s TTL 自然过期。
//
// 多个 hook 按注册顺序串行调用；任一 hook panic 不影响其它 hook（recover）。
// hook 收到的 category 是本次保存的 category 名，回调侧自行判定是否相关。
type SysConfigSavedHook func(ctx context.Context, category string)

// SysConfigService provides business logic for system configuration.
type SysConfigService struct {
	repo  SysConfigRepository
	hooks []SysConfigSavedHook
}

// NewSysConfigService creates a new SysConfigService.
func NewSysConfigService(repo SysConfigRepository) *SysConfigService {
	return &SysConfigService{repo: repo}
}

// RegisterSavedHook 注册一个 BatchUpsert 后回调。仅在进程启动 wiring 阶段调用，
// 无并发保护（运行期不再修改 hooks 切片）。
func (s *SysConfigService) RegisterSavedHook(h SysConfigSavedHook) {
	if h == nil {
		return
	}
	s.hooks = append(s.hooks, h)
}

// Create creates a new config entry.
func (s *SysConfigService) Create(ctx context.Context, req CreateSysConfigRequest) (*SysConfig, error) {
	cfg := &SysConfig{
		Category:    req.Category,
		Key:         req.Key,
		Value:       req.Value,
		ValueType:   req.ValueType,
		Description: req.Description,
		IsPublic:    false,
	}
	if cfg.ValueType == "" {
		cfg.ValueType = "string"
	}
	if req.IsPublic != nil {
		cfg.IsPublic = *req.IsPublic
	}

	if err := s.repo.Create(ctx, cfg); err != nil {
		return nil, fmt.Errorf("create config: %w", err)
	}
	return cfg, nil
}

// Get returns a config by ID.
func (s *SysConfigService) Get(ctx context.Context, id uuid.UUID) (*SysConfig, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns configs, optionally filtered by category.
func (s *SysConfigService) List(ctx context.Context, category string, publicOnly bool) ([]SysConfig, error) {
	return s.repo.List(ctx, category, publicOnly)
}

// Update updates a config.
func (s *SysConfigService) Update(ctx context.Context, id uuid.UUID, req UpdateSysConfigRequest) (*SysConfig, error) {
	cfg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get config for update: %w", err)
	}
	if req.Value != nil {
		cfg.Value = *req.Value
	}
	if req.Description != nil {
		cfg.Description = *req.Description
	}
	if req.IsPublic != nil {
		cfg.IsPublic = *req.IsPublic
	}
	if err := s.repo.Update(ctx, cfg); err != nil {
		return nil, fmt.Errorf("update config: %w", err)
	}
	return cfg, nil
}

// Delete deletes a config.
func (s *SysConfigService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// BatchUpsert 按 PRD config.md §5.2 把整个 category 的 KV 批量写入，commit 成功后
// 顺序触发已注册的 SysConfigSavedHook（hook 异常被 recover 隔离，不阻塞接口返回）。
func (s *SysConfigService) BatchUpsert(ctx context.Context, req BatchUpdateSysConfigRequest) (int, error) {
	n, err := s.repo.BatchUpsert(ctx, req.Category, req.Items)
	if err != nil {
		return n, err
	}
	s.fireSavedHooks(ctx, req.Category)
	return n, nil
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

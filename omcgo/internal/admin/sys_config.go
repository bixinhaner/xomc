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

// --- SysConfigRepository ---

// SysConfigRepository defines the persistence interface for system configs.
type SysConfigRepository interface {
	Create(ctx context.Context, cfg *SysConfig) error
	GetByID(ctx context.Context, id uuid.UUID) (*SysConfig, error)
	GetByKey(ctx context.Context, category, key string) (*SysConfig, error)
	List(ctx context.Context, category string, publicOnly bool) ([]SysConfig, error)
	Update(ctx context.Context, cfg *SysConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
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

// --- SysConfigService ---

// SysConfigService provides business logic for system configuration.
type SysConfigService struct {
	repo SysConfigRepository
}

// NewSysConfigService creates a new SysConfigService.
func NewSysConfigService(repo SysConfigRepository) *SysConfigService {
	return &SysConfigService{repo: repo}
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

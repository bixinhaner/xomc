package northbound

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// ServerRepository 北向 OSS 主备服务器持久层。
type ServerRepository interface {
	List(ctx context.Context) ([]Server, error)
	GetByRole(ctx context.Context, role ServerRole) (*Server, error)
	// SetActive 在事务里把指定 role 设为 active、其他设为 inactive。
	// 实现要保证 partial unique index `is_active = TRUE` 不冲突：
	//   1) UPDATE northbound_servers SET is_active = FALSE WHERE is_active = TRUE
	//   2) UPDATE northbound_servers SET is_active = TRUE  WHERE role = $1
	// 两步同事务。
	SetActive(ctx context.Context, role ServerRole) error
	// Update 修改指定 role 的连接配置（host / port / description）。
	// 不动 is_active 字段；切换激活组走 SetActive。role 不存在返回 ErrNotFound。
	Update(ctx context.Context, role ServerRole, host string, port int, description string) error
}

// PgServerRepository 实现 ServerRepository 基于 PostgreSQL。
type PgServerRepository struct {
	pool *pgxpool.Pool
}

var _ ServerRepository = (*PgServerRepository)(nil)

// NewPgServerRepository 创建仓库实例。
func NewPgServerRepository(pool *pgxpool.Pool) *PgServerRepository {
	return &PgServerRepository{pool: pool}
}

var serverColumns = []string{
	"id", "role", "host", "port", "description", "is_active", "created_at", "updated_at",
}

func scanServer(row pgx.Row) (*Server, error) {
	var s Server
	if err := row.Scan(
		&s.ID, &s.Role, &s.Host, &s.Port, &s.Description, &s.IsActive,
		&s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PgServerRepository) List(ctx context.Context) ([]Server, error) {
	query, args, err := storage.Psql.
		Select(serverColumns...).
		From("northbound_servers").
		OrderBy("role ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list northbound_servers SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query northbound_servers: %w", err)
	}
	defer rows.Close()

	servers := make([]Server, 0, 2)
	for rows.Next() {
		s, err := scanServer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan northbound_server row: %w", err)
		}
		servers = append(servers, *s)
	}
	return servers, rows.Err()
}

func (r *PgServerRepository) GetByRole(ctx context.Context, role ServerRole) (*Server, error) {
	query, args, err := storage.Psql.
		Select(serverColumns...).
		From("northbound_servers").
		Where(sq.Eq{"role": role}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get northbound_server SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	s, err := scanServer(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get northbound_server by role: %w", err)
	}
	return s, nil
}

// SetActive 把指定 role 设为 active、其它设为 inactive。
//
// 事务内顺序至关重要：先把所有 is_active=TRUE 改 FALSE，再把目标 role 改 TRUE。
// 反过来会触发 partial unique index `WHERE is_active = TRUE` 冲突。
func (r *PgServerRepository) SetActive(ctx context.Context, role ServerRole) error {
	if !role.IsValid() {
		return fmt.Errorf("invalid role %q: %w", role, commonerrors.ErrInvalidInput)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Step 1: 把所有当前激活的设为 false（partial unique index 防止跨步骤间冲突）
	if _, err := tx.Exec(ctx,
		`UPDATE northbound_servers SET is_active = FALSE WHERE is_active = TRUE`,
	); err != nil {
		return fmt.Errorf("clear active flag: %w", err)
	}

	// Step 2: 设置目标 role 为 active
	tag, err := tx.Exec(ctx,
		`UPDATE northbound_servers SET is_active = TRUE WHERE role = $1`,
		role,
	)
	if err != nil {
		return fmt.Errorf("set active by role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit set-active tx: %w", err)
	}
	return nil
}

// Update 修改 host / port / description；不影响 is_active。
func (r *PgServerRepository) Update(ctx context.Context, role ServerRole, host string, port int, description string) error {
	if !role.IsValid() {
		return fmt.Errorf("invalid role %q: %w", role, commonerrors.ErrInvalidInput)
	}

	query, args, err := storage.Psql.
		Update("northbound_servers").
		Set("host", host).
		Set("port", port).
		Set("description", description).
		Where(sq.Eq{"role": role}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update northbound_server SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update northbound_server: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

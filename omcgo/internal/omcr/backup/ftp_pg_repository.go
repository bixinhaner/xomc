package backup

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
)

// ---- column list ----

var ftpConfigColumns = []string{
	"id", "config_name", "host", "port", "username",
	"password_encrypted", "protocol", "remote_path",
	"passive", "enabled", "created_at", "updated_at",
}

// ======================================================================
// PgFTPConfigRepository
// ======================================================================

var _ FTPConfigRepository = (*PgFTPConfigRepository)(nil)

// PgFTPConfigRepository is a PostgreSQL implementation of FTPConfigRepository.
type PgFTPConfigRepository struct {
	pool *pgxpool.Pool
}

// NewPgFTPConfigRepository creates a new PgFTPConfigRepository.
func NewPgFTPConfigRepository(pool *pgxpool.Pool) *PgFTPConfigRepository {
	return &PgFTPConfigRepository{pool: pool}
}

func (r *PgFTPConfigRepository) Create(ctx context.Context, config *FTPConfig) error {
	query, args, err := psql.Insert("ftp_configs").
		Columns("config_name", "host", "port", "username",
			"password_encrypted", "protocol", "remote_path",
			"passive", "enabled").
		Values(config.ConfigName, config.Host, config.Port, config.Username,
			config.PasswordEncrypted, config.Protocol, config.RemotePath,
			config.Passive, config.Enabled).
		Suffix("RETURNING " + joinColumns(ftpConfigColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert ftp_config SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanFTPConfig(row)
	if err != nil {
		return fmt.Errorf("create ftp_config: %w", err)
	}
	*config = *created
	return nil
}

func (r *PgFTPConfigRepository) GetByID(ctx context.Context, id uuid.UUID) (*FTPConfig, error) {
	query, args, err := psql.Select(ftpConfigColumns...).
		From("ftp_configs").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get ftp_config SQL: %w", err)
	}

	config, err := scanFTPConfig(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get ftp_config: %w", err)
	}
	return config, nil
}

func (r *PgFTPConfigRepository) Update(ctx context.Context, config *FTPConfig) error {
	query, args, err := psql.Update("ftp_configs").
		Set("config_name", config.ConfigName).
		Set("host", config.Host).
		Set("port", config.Port).
		Set("username", config.Username).
		Set("password_encrypted", config.PasswordEncrypted).
		Set("protocol", config.Protocol).
		Set("remote_path", config.RemotePath).
		Set("passive", config.Passive).
		Set("enabled", config.Enabled).
		Where(sq.Eq{"id": config.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update ftp_config SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update ftp_config: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgFTPConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Delete("ftp_configs").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete ftp_config SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete ftp_config: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgFTPConfigRepository) List(ctx context.Context, filter FTPConfigFilter) (*model.ListResponse[FTPConfig], error) {
	base := psql.Select(ftpConfigColumns...).From("ftp_configs")
	countBase := psql.Select("COUNT(*)").From("ftp_configs")

	if filter.Enabled != nil {
		base = base.Where(sq.Eq{"enabled": *filter.Enabled})
		countBase = countBase.Where(sq.Eq{"enabled": *filter.Enabled})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count ftp_config SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count ftp_configs: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list ftp_config SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list ftp_configs: %w", err)
	}
	defer rows.Close()

	var items []FTPConfig
	for rows.Next() {
		config, err := scanFTPConfigRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ftp_config row: %w", err)
		}
		items = append(items, *config)
	}

	if items == nil {
		items = []FTPConfig{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- FTP config scanning helpers ----

func scanFTPConfig(row pgx.Row) (*FTPConfig, error) {
	var c FTPConfig
	err := row.Scan(
		&c.ID, &c.ConfigName, &c.Host, &c.Port, &c.Username,
		&c.PasswordEncrypted, &c.Protocol, &c.RemotePath,
		&c.Passive, &c.Enabled, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanFTPConfigRow(rows pgx.Rows) (*FTPConfig, error) {
	var c FTPConfig
	err := rows.Scan(
		&c.ID, &c.ConfigName, &c.Host, &c.Port, &c.Username,
		&c.PasswordEncrypted, &c.Protocol, &c.RemotePath,
		&c.Passive, &c.Enabled, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

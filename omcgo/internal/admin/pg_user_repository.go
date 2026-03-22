package admin

import (
	"context"
	"fmt"
	"math"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var userColumns = []string{
	"id", "username", "password_hash", "display_name", "email",
	"carrier", "status", "failed_login_attempts", "locked_until",
	"last_failed_login_at", "last_login_at", "created_at", "updated_at",
}

// PgUserRepository implements UserRepository using PostgreSQL.
type PgUserRepository struct {
	pool *pgxpool.Pool
}

var _ UserRepository = (*PgUserRepository)(nil)

// NewPgUserRepository creates a new PgUserRepository.
func NewPgUserRepository(pool *pgxpool.Pool) *PgUserRepository {
	return &PgUserRepository{pool: pool}
}

func (r *PgUserRepository) Create(ctx context.Context, user *User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	query, args, err := psql.Insert("users").
		Columns(userColumns...).
		Values(
			user.ID, user.Username, user.PasswordHash, user.DisplayName,
			nullableString(user.Email), nullableCarrier(user.Carrier),
			user.Status, user.FailedLoginAttempts, nullableTime(user.LockedUntil),
			nullableTime(user.LastFailedLoginAt), nullableTime(user.LastLoginAt),
			user.CreatedAt, user.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert user SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *PgUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query, args, err := psql.Select(userColumns...).
		From("users").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get user SQL: %w", err)
	}

	user, err := scanUser(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *PgUserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	query, args, err := psql.Select(userColumns...).
		From("users").
		Where(sq.Eq{"username": username}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get user by username SQL: %w", err)
	}

	user, err := scanUser(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *PgUserRepository) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()

	query, args, err := psql.Update("users").
		Set("display_name", user.DisplayName).
		Set("email", nullableString(user.Email)).
		Set("carrier", nullableCarrier(user.Carrier)).
		Set("status", user.Status).
		Set("updated_at", user.UpdatedAt).
		Where(sq.Eq{"id": user.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update user SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query, args, err := psql.Update("users").
		Set("password_hash", passwordHash).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update password SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Delete("users").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete user SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgUserRepository) List(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error) {
	base := psql.Select(userColumns...).From("users")
	countBase := psql.Select("COUNT(*)").From("users")

	if filter.Carrier != nil {
		base = base.Where(sq.Eq{"carrier": *filter.Carrier})
		countBase = countBase.Where(sq.Eq{"carrier": *filter.Carrier})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.Search != nil && *filter.Search != "" {
		like := "%" + *filter.Search + "%"
		cond := sq.Or{
			sq.ILike{"username": like},
			sq.ILike{"display_name": like},
			sq.ILike{"email": like},
		}
		base = base.Where(cond)
		countBase = countBase.Where(cond)
	}

	// Count
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count users SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	// Pagination
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir != "" {
		sortDir = filter.SortDir
	}

	query, args, err := base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list users SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}

	return &model.ListResponse[User]{
		Items:      users,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	}, nil
}

func (r *PgUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Update("users").
		Set("last_login_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update last login SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	var email, carrier *string
	err := row.Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &email,
		&carrier, &u.Status, &u.FailedLoginAttempts, &u.LockedUntil,
		&u.LastFailedLoginAt, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	if email != nil {
		u.Email = *email
	}
	if carrier != nil {
		cc := model.CarrierCode(*carrier)
		u.Carrier = &cc
	}
	return &u, nil
}

func scanUserFromRows(rows pgx.Rows) (*User, error) {
	var u User
	var email, carrier *string
	err := rows.Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &email,
		&carrier, &u.Status, &u.FailedLoginAttempts, &u.LockedUntil,
		&u.LastFailedLoginAt, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan user row: %w", err)
	}
	if email != nil {
		u.Email = *email
	}
	if carrier != nil {
		cc := model.CarrierCode(*carrier)
		u.Carrier = &cc
	}
	return &u, nil
}

// UpdateLoginSecurity updates the failed login attempt counter and lock fields.
func (r *PgUserRepository) UpdateLoginSecurity(ctx context.Context, id uuid.UUID, failedAttempts int, lockedUntil *time.Time) error {
	now := time.Now()
	query, args, err := psql.Update("users").
		Set("failed_login_attempts", failedAttempts).
		Set("locked_until", nullableTime(lockedUntil)).
		Set("last_failed_login_at", now).
		Set("updated_at", now).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update login security SQL: %w", err)
	}
	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update login security: %w", err)
	}
	return nil
}

// ResetLoginSecurity clears failed login attempts and lock on successful login.
func (r *PgUserRepository) ResetLoginSecurity(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Update("users").
		Set("failed_login_attempts", 0).
		Set("locked_until", nil).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build reset login security SQL: %w", err)
	}
	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("reset login security: %w", err)
	}
	return nil
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableCarrier(c *model.CarrierCode) interface{} {
	if c == nil {
		return nil
	}
	return string(*c)
}

func nullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

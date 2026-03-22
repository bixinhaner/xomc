package nedirect

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// ---- column lists ----

var sessionColumns = []string{
	"id", "device_id", "device_sn", "user_id", "username",
	"status", "ip_address", "connected_at", "last_active_at",
	"disconnect_at", "created_at", "updated_at",
}

var commandColumns = []string{
	"id", "session_id", "device_sn", "command", "status",
	"response", "error_msg", "sent_at", "respond_at", "created_at",
}

// ======================================================================
// PgSessionRepository
// ======================================================================

var _ SessionRepository = (*PgSessionRepository)(nil)

// PgSessionRepository is a PostgreSQL implementation of SessionRepository.
type PgSessionRepository struct {
	pool *pgxpool.Pool
}

// NewPgSessionRepository creates a new PgSessionRepository.
func NewPgSessionRepository(pool *pgxpool.Pool) *PgSessionRepository {
	return &PgSessionRepository{pool: pool}
}

func (r *PgSessionRepository) Create(ctx context.Context, session *Session) error {
	query, args, err := psql.Insert("nedirect_sessions").
		Columns("device_id", "device_sn", "user_id", "username",
			"status", "ip_address", "connected_at", "last_active_at").
		Values(session.DeviceID, session.DeviceSN, session.UserID, session.Username,
			session.Status, session.IPAddress, session.ConnectedAt, session.LastActiveAt).
		Suffix("RETURNING " + joinColumns(sessionColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert nedirect_session SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanSession(row)
	if err != nil {
		return fmt.Errorf("create nedirect_session: %w", err)
	}
	*session = *created
	return nil
}

func (r *PgSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	query, args, err := psql.Select(sessionColumns...).
		From("nedirect_sessions").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get nedirect_session SQL: %w", err)
	}

	session, err := scanSession(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get nedirect_session: %w", err)
	}
	return session, nil
}

func (r *PgSessionRepository) GetActiveByDeviceAndUser(ctx context.Context, deviceSN, userID string) (*Session, error) {
	query, args, err := psql.Select(sessionColumns...).
		From("nedirect_sessions").
		Where(sq.Eq{
			"device_sn": deviceSN,
			"user_id":   userID,
			"status":    SessionActive,
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get active session SQL: %w", err)
	}

	session, err := scanSession(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get active session: %w", err)
	}
	return session, nil
}

func (r *PgSessionRepository) Update(ctx context.Context, session *Session) error {
	query, args, err := psql.Update("nedirect_sessions").
		Set("status", session.Status).
		Set("last_active_at", session.LastActiveAt).
		Set("disconnect_at", session.DisconnectAt).
		Where(sq.Eq{"id": session.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update nedirect_session SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update nedirect_session: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgSessionRepository) List(ctx context.Context, filter SessionFilter) (*model.ListResponse[Session], error) {
	base := psql.Select(sessionColumns...).From("nedirect_sessions")
	countBase := psql.Select("COUNT(*)").From("nedirect_sessions")

	if filter.DeviceSN != nil {
		base = base.Where(sq.Eq{"device_sn": *filter.DeviceSN})
		countBase = countBase.Where(sq.Eq{"device_sn": *filter.DeviceSN})
	}
	if filter.UserID != nil {
		base = base.Where(sq.Eq{"user_id": *filter.UserID})
		countBase = countBase.Where(sq.Eq{"user_id": *filter.UserID})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}

	// Count
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count nedirect_session SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count nedirect_sessions: %w", err)
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
		return nil, fmt.Errorf("build list nedirect_session SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list nedirect_sessions: %w", err)
	}
	defer rows.Close()

	var items []Session
	for rows.Next() {
		s, err := scanSessionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan nedirect_session row: %w", err)
		}
		items = append(items, *s)
	}
	if items == nil {
		items = []Session{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgSessionRepository) ListActiveByDevice(ctx context.Context, deviceSN string) ([]Session, error) {
	query, args, err := psql.Select(sessionColumns...).
		From("nedirect_sessions").
		Where(sq.Eq{"device_sn": deviceSN, "status": SessionActive}).
		OrderBy("connected_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list active sessions SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list active sessions: %w", err)
	}
	defer rows.Close()

	var items []Session
	for rows.Next() {
		s, err := scanSessionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan session row: %w", err)
		}
		items = append(items, *s)
	}
	if items == nil {
		items = []Session{}
	}
	return items, nil
}

func (r *PgSessionRepository) CloseExpiredSessions(ctx context.Context, timeoutMinutes int) (int64, error) {
	query, args, err := psql.Update("nedirect_sessions").
		Set("status", SessionTimedOut).
		Set("disconnect_at", sq.Expr("NOW()")).
		Where(sq.Eq{"status": SessionActive}).
		Where(sq.Expr("last_active_at < NOW() - INTERVAL '1 minute' * ?", timeoutMinutes)).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build close expired sessions SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("close expired sessions: %w", err)
	}
	return result.RowsAffected(), nil
}

// ---- session scanning helpers ----

func scanSession(row pgx.Row) (*Session, error) {
	var s Session
	err := row.Scan(
		&s.ID, &s.DeviceID, &s.DeviceSN, &s.UserID, &s.Username,
		&s.Status, &s.IPAddress, &s.ConnectedAt, &s.LastActiveAt,
		&s.DisconnectAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func scanSessionRow(rows pgx.Rows) (*Session, error) {
	var s Session
	err := rows.Scan(
		&s.ID, &s.DeviceID, &s.DeviceSN, &s.UserID, &s.Username,
		&s.Status, &s.IPAddress, &s.ConnectedAt, &s.LastActiveAt,
		&s.DisconnectAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ======================================================================
// PgCommandRepository
// ======================================================================

var _ CommandRepository = (*PgCommandRepository)(nil)

// PgCommandRepository is a PostgreSQL implementation of CommandRepository.
type PgCommandRepository struct {
	pool *pgxpool.Pool
}

// NewPgCommandRepository creates a new PgCommandRepository.
func NewPgCommandRepository(pool *pgxpool.Pool) *PgCommandRepository {
	return &PgCommandRepository{pool: pool}
}

func (r *PgCommandRepository) Create(ctx context.Context, cmd *Command) error {
	query, args, err := psql.Insert("nedirect_commands").
		Columns("session_id", "device_sn", "command", "status", "sent_at").
		Values(cmd.SessionID, cmd.DeviceSN, cmd.CommandStr, cmd.Status, cmd.SentAt).
		Suffix("RETURNING " + joinColumns(commandColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert nedirect_command SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanCommand(row)
	if err != nil {
		return fmt.Errorf("create nedirect_command: %w", err)
	}
	*cmd = *created
	return nil
}

func (r *PgCommandRepository) GetByID(ctx context.Context, id uuid.UUID) (*Command, error) {
	query, args, err := psql.Select(commandColumns...).
		From("nedirect_commands").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get nedirect_command SQL: %w", err)
	}

	cmd, err := scanCommand(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get nedirect_command: %w", err)
	}
	return cmd, nil
}

func (r *PgCommandRepository) Update(ctx context.Context, cmd *Command) error {
	query, args, err := psql.Update("nedirect_commands").
		Set("status", cmd.Status).
		Set("response", cmd.Response).
		Set("error_msg", cmd.ErrorMsg).
		Set("respond_at", cmd.RespondAt).
		Where(sq.Eq{"id": cmd.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update nedirect_command SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update nedirect_command: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgCommandRepository) List(ctx context.Context, filter CommandFilter) (*model.ListResponse[Command], error) {
	base := psql.Select(commandColumns...).From("nedirect_commands")
	countBase := psql.Select("COUNT(*)").From("nedirect_commands")

	if filter.SessionID != nil {
		base = base.Where(sq.Eq{"session_id": *filter.SessionID})
		countBase = countBase.Where(sq.Eq{"session_id": *filter.SessionID})
	}
	if filter.DeviceSN != nil {
		base = base.Where(sq.Eq{"device_sn": *filter.DeviceSN})
		countBase = countBase.Where(sq.Eq{"device_sn": *filter.DeviceSN})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}

	// Count
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count nedirect_command SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count nedirect_commands: %w", err)
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
		return nil, fmt.Errorf("build list nedirect_command SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list nedirect_commands: %w", err)
	}
	defer rows.Close()

	var items []Command
	for rows.Next() {
		cmd, err := scanCommandRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan nedirect_command row: %w", err)
		}
		items = append(items, *cmd)
	}
	if items == nil {
		items = []Command{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- command scanning helpers ----

func scanCommand(row pgx.Row) (*Command, error) {
	var c Command
	err := row.Scan(
		&c.ID, &c.SessionID, &c.DeviceSN, &c.CommandStr, &c.Status,
		&c.Response, &c.ErrorMsg, &c.SentAt, &c.RespondAt, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanCommandRow(rows pgx.Rows) (*Command, error) {
	var c Command
	err := rows.Scan(
		&c.ID, &c.SessionID, &c.DeviceSN, &c.CommandStr, &c.Status,
		&c.Response, &c.ErrorMsg, &c.SentAt, &c.RespondAt, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ---- shared helpers ----

func joinColumns(cols []string) string {
	result := ""
	for i, c := range cols {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}

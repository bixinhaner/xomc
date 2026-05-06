package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/admin/sqlc"
	"github.com/omcgo/omcgo/internal/core/model"
)

// PgUserRepositoryV2 implements UserRepository using sqlc-generated queries.
// This is an alternative to PgUserRepository that provides compile-time type safety
// for SQL queries. To activate, change the constructor call in cmd/app/provider/admin.go.
type PgUserRepositoryV2 struct {
	q *adminsqlc.Queries
}

// NewPgUserRepositoryV2 creates a new sqlc-backed UserRepository.
func NewPgUserRepositoryV2(pool *pgxpool.Pool) *PgUserRepositoryV2 {
	return &PgUserRepositoryV2{
		q: adminsqlc.New(pool),
	}
}

func (r *PgUserRepositoryV2) Create(ctx context.Context, user *User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	source := user.Source
	if source == "" {
		source = UserSourceAdmin
	}
	_, err := r.q.CreateUser(ctx, adminsqlc.CreateUserParams{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		DisplayName:  toPgText(user.DisplayName),
		Email:        toPgText(user.Email),
		Phone:        toPgText(user.Phone),
		Carrier:      carrierToPgText(user.Carrier),
		Status:       string(user.Status),
		Source:       string(source),
	})
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *PgUserRepositoryV2) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return sqlcUserToAdmin(u), nil
}

func (r *PgUserRepositoryV2) GetByUsername(ctx context.Context, username string) (*User, error) {
	u, err := r.q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return sqlcUserToAdmin(u), nil
}

func (r *PgUserRepositoryV2) Update(ctx context.Context, user *User) error {
	_, err := r.q.UpdateUser(ctx, adminsqlc.UpdateUserParams{
		ID:          user.ID,
		DisplayName: toPgText(user.DisplayName),
		Email:       toPgText(user.Email),
		Phone:       toPgText(user.Phone),
		Carrier:     carrierToPgText(user.Carrier),
		Status:      string(user.Status),
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *PgUserRepositoryV2) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return r.q.UpdatePassword(ctx, adminsqlc.UpdatePasswordParams{
		ID:           id,
		PasswordHash: passwordHash,
	})
}

func (r *PgUserRepositoryV2) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteUser(ctx, id)
}

func (r *PgUserRepositoryV2) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	return r.q.UpdateLastLogin(ctx, id)
}

func (r *PgUserRepositoryV2) List(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error) {
	carrier := carrierToPgText(filter.Carrier)
	status := ""
	if filter.Status != nil {
		status = string(*filter.Status)
	}
	search := pgtype.Text{}
	if filter.Search != nil {
		search = toPgText(*filter.Search)
	}

	total, err := r.q.CountUsers(ctx, adminsqlc.CountUsersParams{
		Carrier: carrier,
		Status:  status,
		Column3: search,
	})
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	rows, err := r.q.ListUsers(ctx, adminsqlc.ListUsersParams{
		Carrier: carrier,
		Status:  status,
		Column3: search,
		Limit:   int32(filter.Limit()),
		Offset:  int32(filter.Offset()),
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]User, 0, len(rows))
	for _, u := range rows {
		users = append(users, *sqlcUserToAdmin(u))
	}

	return model.NewListResponse(users, total, filter.Page, filter.PageSize), nil
}

// Helper functions for type conversion between domain models and sqlc models.

func toPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func carrierToPgText(c *model.CarrierCode) pgtype.Text {
	if c == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: string(*c), Valid: true}
}

func sqlcUserToAdmin(u adminsqlc.User) *User {
	user := &User{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Status:       UserStatus(u.Status),
		Source:       UserSource(u.Source),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
	if u.DisplayName.Valid {
		user.DisplayName = u.DisplayName.String
	}
	if u.Email.Valid {
		user.Email = u.Email.String
	}
	if u.Phone.Valid {
		user.Phone = u.Phone.String
	}
	if u.Carrier.Valid {
		c := model.CarrierCode(u.Carrier.String)
		user.Carrier = &c
	}
	if u.LastLoginAt.Valid {
		t := u.LastLoginAt.Time
		user.LastLoginAt = &t
	}
	return user
}

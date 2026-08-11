package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type ContactGroup struct {
	ID          uuid.UUID            `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	IsDefault   bool                 `json:"is_default"`
	Revision    int64                `json:"revision"`
	Archived    bool                 `json:"archived"`
	CreatedBy   string               `json:"created_by"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	Members     []ContactGroupMember `json:"members"`
}

type ContactGroupMember struct {
	ID                uuid.UUID `json:"id"`
	TargetType        string    `json:"target_type"`
	TargetID          string    `json:"target_id,omitempty"`
	AddressConfigured bool      `json:"address_configured"`
	ChannelLimit      []string  `json:"channel_limit"`
}

type ContactGroupMemberInput struct {
	TargetType           string   `json:"target_type"`
	TargetID             string   `json:"target_id,omitempty"`
	AddressCiphertext    []byte   `json:"address_ciphertext,omitempty"`
	AddressKeyVersion    int      `json:"address_key_version,omitempty"`
	RecipientFingerprint []byte   `json:"recipient_fingerprint,omitempty"`
	ChannelLimit         []string `json:"channel_limit"`
}

type ContactGroupInput struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	IsDefault   bool                      `json:"is_default"`
	Members     []ContactGroupMemberInput `json:"members"`
}

type ContactGroupRepository interface {
	List(context.Context) ([]ContactGroup, error)
	Get(context.Context, uuid.UUID) (*ContactGroup, error)
	Create(context.Context, ContactGroupInput, string) (*ContactGroup, error)
	Update(context.Context, uuid.UUID, int64, ContactGroupInput, string) (*ContactGroup, error)
	ListMembers(context.Context, uuid.UUID) ([]RecipientTarget, error)
}

var _ ContactGroupRepository = (*PgContactGroupRepository)(nil)

type PgContactGroupRepository struct{ db storage.DB }

func NewPgContactGroupRepository(pool *pgxpool.Pool) *PgContactGroupRepository {
	return &PgContactGroupRepository{db: storage.NewPoolDB(pool)}
}

func (r *PgContactGroupRepository) List(ctx context.Context) ([]ContactGroup, error) {
	query, args, err := storage.Psql.Select(
		"id", "name", "description", "is_default", "revision", "archived", "created_by", "created_at", "updated_at",
	).From("notification_contact_groups").OrderBy("is_default DESC", "name", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification contact group list: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notification contact groups: %w", err)
	}
	defer rows.Close()
	groups := make([]ContactGroup, 0)
	for rows.Next() {
		group, scanErr := scanContactGroup(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan notification contact group: %w", scanErr)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification contact groups: %w", err)
	}
	return groups, nil
}

func (r *PgContactGroupRepository) Get(ctx context.Context, id uuid.UUID) (*ContactGroup, error) {
	return getContactGroup(ctx, r.db, id, false)
}

func (r *PgContactGroupRepository) Create(ctx context.Context, input ContactGroupInput, actor string) (*ContactGroup, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin notification contact group create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id, now := uuid.New(), time.Now().UTC()
	if input.IsDefault {
		if err := clearDefaultContactGroup(ctx, tx, now, uuid.Nil); err != nil {
			return nil, err
		}
	}
	query, args, err := storage.Psql.Insert("notification_contact_groups").Columns(
		"id", "name", "description", "is_default", "revision", "archived", "created_by", "created_at", "updated_at",
	).Values(id, input.Name, input.Description, input.IsDefault, 1, false, actor, now, now).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification contact group create: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return nil, mapContactGroupWriteError("create notification contact group", err)
	}
	if err := insertContactGroupMembers(ctx, tx, id, input.Members, now); err != nil {
		return nil, err
	}
	return commitAndReadContactGroup(ctx, tx, id, "notification contact group create")
}

func (r *PgContactGroupRepository) Update(
	ctx context.Context,
	id uuid.UUID,
	expectedRevision int64,
	input ContactGroupInput,
	actor string,
) (*ContactGroup, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin notification contact group update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	group, err := getContactGroup(ctx, tx, id, true)
	if err != nil {
		return nil, err
	}
	if group.Revision != expectedRevision {
		return nil, ErrRevisionMismatch
	}
	now := time.Now().UTC()
	if input.IsDefault {
		if err := clearDefaultContactGroup(ctx, tx, now, id); err != nil {
			return nil, err
		}
	}
	query, args, err := storage.Psql.Update("notification_contact_groups").
		Set("name", input.Name).Set("description", input.Description).Set("is_default", input.IsDefault).
		Set("revision", expectedRevision+1).Set("updated_at", now).
		Where(sq.Eq{"id": id, "revision": expectedRevision}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification contact group update: %w", err)
	}
	if tag, execErr := tx.Exec(ctx, query, args...); execErr != nil {
		return nil, mapContactGroupWriteError("update notification contact group", execErr)
	} else if tag.RowsAffected() != 1 {
		return nil, ErrRevisionMismatch
	}
	deleteQuery, deleteArgs, err := storage.Psql.Delete("notification_contact_group_members").Where(sq.Eq{"contact_group_id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification contact group member replacement: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteQuery, deleteArgs...); err != nil {
		return nil, fmt.Errorf("replace notification contact group members: %w", err)
	}
	if err := insertContactGroupMembers(ctx, tx, id, input.Members, now); err != nil {
		return nil, err
	}
	_ = actor
	return commitAndReadContactGroup(ctx, tx, id, "notification contact group update")
}

func (r *PgContactGroupRepository) ListMembers(ctx context.Context, id uuid.UUID) ([]RecipientTarget, error) {
	group, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if group.Archived {
		return nil, commonerrors.ErrNotFound
	}
	query, args, err := storage.Psql.Select(
		"target_type", "target_id", "address_ciphertext", "address_key_version", "recipient_fingerprint", "channel_limit",
	).From("notification_contact_group_members").Where(sq.Eq{"contact_group_id": id}).OrderBy("created_at", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification contact group member list: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notification contact group members: %w", err)
	}
	defer rows.Close()
	targets := make([]RecipientTarget, 0)
	for rows.Next() {
		var target RecipientTarget
		var targetID *string
		var keyVersion *int
		if err := rows.Scan(
			&target.TargetType, &targetID, &target.AddressCiphertext, &keyVersion,
			&target.RecipientFingerprint, &target.ChannelLimit,
		); err != nil {
			return nil, fmt.Errorf("scan notification contact group member target: %w", err)
		}
		if targetID != nil {
			target.TargetID = *targetID
		}
		if keyVersion != nil {
			target.AddressKeyVersion = *keyVersion
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification contact group member targets: %w", err)
	}
	return targets, nil
}

func getContactGroup(ctx context.Context, queryer ruleQueryer, id uuid.UUID, lock bool) (*ContactGroup, error) {
	builder := storage.Psql.Select(
		"id", "name", "description", "is_default", "revision", "archived", "created_by", "created_at", "updated_at",
	).From("notification_contact_groups").Where(sq.Eq{"id": id})
	if lock {
		builder = builder.Suffix("FOR UPDATE")
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification contact group lookup: %w", err)
	}
	group, err := scanContactGroup(queryer.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, commonerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get notification contact group: %w", err)
	}
	membersQuery, membersArgs, err := storage.Psql.Select(
		"id", "target_type", "target_id", "address_ciphertext IS NOT NULL", "channel_limit",
	).From("notification_contact_group_members").Where(sq.Eq{"contact_group_id": id}).OrderBy("created_at", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification contact group members lookup: %w", err)
	}
	rows, err := queryer.Query(ctx, membersQuery, membersArgs...)
	if err != nil {
		return nil, fmt.Errorf("get notification contact group members: %w", err)
	}
	defer rows.Close()
	group.Members = make([]ContactGroupMember, 0)
	for rows.Next() {
		var member ContactGroupMember
		var targetID *string
		if err := rows.Scan(&member.ID, &member.TargetType, &targetID, &member.AddressConfigured, &member.ChannelLimit); err != nil {
			return nil, fmt.Errorf("scan notification contact group member: %w", err)
		}
		if targetID != nil {
			member.TargetID = *targetID
		}
		group.Members = append(group.Members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification contact group members: %w", err)
	}
	return &group, nil
}

func scanContactGroup(row interface{ Scan(...any) error }) (ContactGroup, error) {
	var group ContactGroup
	err := row.Scan(
		&group.ID, &group.Name, &group.Description, &group.IsDefault, &group.Revision,
		&group.Archived, &group.CreatedBy, &group.CreatedAt, &group.UpdatedAt,
	)
	return group, err
}

func insertContactGroupMembers(ctx context.Context, queryer ruleQueryer, groupID uuid.UUID, members []ContactGroupMemberInput, now time.Time) error {
	for _, member := range members {
		var targetID any
		if member.TargetID != "" {
			targetID = member.TargetID
		}
		var ciphertext, fingerprint any
		var keyVersion any
		if len(member.AddressCiphertext) > 0 {
			ciphertext, keyVersion, fingerprint = member.AddressCiphertext, member.AddressKeyVersion, member.RecipientFingerprint
		}
		query, args, err := storage.Psql.Insert("notification_contact_group_members").Columns(
			"id", "contact_group_id", "target_type", "target_id", "address_ciphertext",
			"address_key_version", "recipient_fingerprint", "channel_limit", "created_at",
		).Values(
			uuid.New(), groupID, member.TargetType, targetID, ciphertext,
			keyVersion, fingerprint, member.ChannelLimit, now,
		).ToSql()
		if err != nil {
			return fmt.Errorf("build notification contact group member insert: %w", err)
		}
		if _, err := queryer.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert notification contact group member: %w", err)
		}
	}
	return nil
}

func clearDefaultContactGroup(ctx context.Context, queryer ruleQueryer, now time.Time, excludedID uuid.UUID) error {
	builder := storage.Psql.Update("notification_contact_groups").
		Set("is_default", false).Set("revision", sq.Expr("revision + 1")).Set("updated_at", now).
		Where(sq.Eq{"is_default": true})
	if excludedID != uuid.Nil {
		builder = builder.Where(sq.NotEq{"id": excludedID})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build notification default contact group clear: %w", err)
	}
	if _, err := queryer.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("clear notification default contact group: %w", err)
	}
	return nil
}

func commitAndReadContactGroup(ctx context.Context, tx pgx.Tx, id uuid.UUID, operation string) (*ContactGroup, error) {
	group, err := getContactGroup(ctx, tx, id, false)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit %s: %w", operation, err)
	}
	return group, nil
}

func mapContactGroupWriteError(context string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return commonerrors.ErrAlreadyExists
	}
	return fmt.Errorf("%s: %w", context, err)
}

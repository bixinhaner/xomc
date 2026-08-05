package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var _ TemplateManagementRepository = (*PgTemplateManagementRepository)(nil)

type PgTemplateManagementRepository struct{ db storage.DB }

func NewPgTemplateManagementRepository(pool *pgxpool.Pool) *PgTemplateManagementRepository {
	return &PgTemplateManagementRepository{db: storage.NewPoolDB(pool)}
}

var managedTemplateColumns = []string{
	"id", "name", "channel", "language", "revision", "current_draft_version_id",
	"current_published_version_id", "archived", "created_by", "created_at", "updated_at",
}

func (r *PgTemplateManagementRepository) List(ctx context.Context) ([]ManagedTemplate, error) {
	query, args, err := storage.Psql.Select(managedTemplateColumns...).From("notification_templates").
		Where(sq.Eq{"archived": false}).OrderBy("channel", "name", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build versioned notification template list: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list versioned notification templates: %w", err)
	}
	defer rows.Close()
	items := make([]ManagedTemplate, 0)
	for rows.Next() {
		item, scanErr := scanManagedTemplate(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan versioned notification template: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate versioned notification templates: %w", err)
	}
	return items, nil
}

func (r *PgTemplateManagementRepository) Get(ctx context.Context, id uuid.UUID) (*ManagedTemplate, error) {
	return getManagedTemplate(ctx, r.db, id, false)
}

func (r *PgTemplateManagementRepository) Create(ctx context.Context, input ManagedTemplateInput, actor string) (*ManagedTemplate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin versioned notification template create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	templateID, versionID, now := uuid.New(), uuid.New(), time.Now().UTC()
	query, args, err := storage.Psql.Insert("notification_templates").Columns(
		"id", "name", "channel", "language", "subject", "body", "variables", "enabled",
		"revision", "current_draft_version_id", "archived", "created_by", "created_at", "updated_at",
	).Values(
		templateID, input.Name, input.Channel, input.Language, input.Subject, input.TextBody, input.Variables,
		false, 1, versionID, false, actor, now, now,
	).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build versioned notification template create: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return nil, mapRuleWriteError("create versioned notification template", err)
	}
	if err := insertManagedTemplateVersion(ctx, tx, versionID, templateID, 1, input, actor, now); err != nil {
		return nil, err
	}
	return commitAndReadManagedTemplate(ctx, tx, templateID, "versioned notification template create")
}

func (r *PgTemplateManagementRepository) UpdateDraft(ctx context.Context, id uuid.UUID, expectedRevision int64, input ManagedTemplateInput, actor string) (*ManagedTemplate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin versioned notification template update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := getManagedTemplate(ctx, tx, id, true)
	if err != nil {
		return nil, err
	}
	if item.Revision != expectedRevision {
		return nil, ErrRevisionMismatch
	}
	if item.Archived {
		return nil, commonerrors.ErrInvalidInput
	}
	var nextVersion int64
	versionQuery, versionArgs, err := storage.Psql.Select("COALESCE(MAX(version_no), 0) + 1").
		From("notification_template_versions").Where(sq.Eq{"template_id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build next notification template version: %w", err)
	}
	if err := tx.QueryRow(ctx, versionQuery, versionArgs...).Scan(&nextVersion); err != nil {
		return nil, fmt.Errorf("read next notification template version: %w", err)
	}
	versionID, now := uuid.New(), time.Now().UTC()
	if err := insertManagedTemplateVersion(ctx, tx, versionID, id, nextVersion, input, actor, now); err != nil {
		return nil, err
	}
	query, args, err := storage.Psql.Update("notification_templates").
		Set("name", input.Name).Set("channel", input.Channel).Set("language", input.Language).
		Set("subject", input.Subject).Set("body", input.TextBody).Set("variables", input.Variables).
		Set("current_draft_version_id", versionID).Set("revision", expectedRevision+1).Set("updated_at", now).
		Where(sq.Eq{"id": id, "revision": expectedRevision}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build versioned notification template pointer update: %w", err)
	}
	if tag, execErr := tx.Exec(ctx, query, args...); execErr != nil {
		return nil, mapRuleWriteError("update versioned notification template", execErr)
	} else if tag.RowsAffected() != 1 {
		return nil, ErrRevisionMismatch
	}
	return commitAndReadManagedTemplate(ctx, tx, id, "versioned notification template update")
}

func (r *PgTemplateManagementRepository) Publish(ctx context.Context, id uuid.UUID, expectedRevision int64, actor string) (*ManagedTemplate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin notification template publish: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := getManagedTemplate(ctx, tx, id, true)
	if err != nil {
		return nil, err
	}
	if item.Revision != expectedRevision {
		return nil, ErrRevisionMismatch
	}
	if item.Archived || item.CurrentDraftVersionID == nil ||
		(item.CurrentPublishedVersionID != nil && *item.CurrentPublishedVersionID == *item.CurrentDraftVersionID) {
		return nil, ErrDraftUnavailable
	}
	now := time.Now().UTC()
	query, args, err := storage.Psql.Update("notification_template_versions").Set("published_at", now).
		Where(sq.Eq{"id": *item.CurrentDraftVersionID, "template_id": id, "published_at": nil}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification template version publish: %w", err)
	}
	if tag, execErr := tx.Exec(ctx, query, args...); execErr != nil {
		return nil, fmt.Errorf("publish notification template version: %w", execErr)
	} else if tag.RowsAffected() != 1 {
		return nil, ErrDraftUnavailable
	}
	update, updateArgs, err := storage.Psql.Update("notification_templates").
		Set("current_published_version_id", *item.CurrentDraftVersionID).Set("enabled", true).
		Set("revision", expectedRevision+1).Set("updated_at", now).
		Where(sq.Eq{"id": id, "revision": expectedRevision}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification template published pointer update: %w", err)
	}
	if tag, execErr := tx.Exec(ctx, update, updateArgs...); execErr != nil {
		return nil, fmt.Errorf("update notification template published pointer: %w", execErr)
	} else if tag.RowsAffected() != 1 {
		return nil, ErrRevisionMismatch
	}
	_ = actor
	return commitAndReadManagedTemplate(ctx, tx, id, "notification template publish")
}

func insertManagedTemplateVersion(ctx context.Context, queryer ruleQueryer, id, templateID uuid.UUID, versionNo int64, input ManagedTemplateInput, actor string, now time.Time) error {
	query, args, err := storage.Psql.Insert("notification_template_versions").Columns(
		"id", "template_id", "version_no", "channel", "language", "subject", "text_body", "html_body",
		"variables", "created_by", "change_reason", "created_at",
	).Values(
		id, templateID, versionNo, input.Channel, input.Language, input.Subject, input.TextBody, input.HTMLBody,
		input.Variables, actor, input.ChangeReason, now,
	).ToSql()
	if err != nil {
		return fmt.Errorf("build notification template version insert: %w", err)
	}
	if _, err := queryer.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert notification template version: %w", err)
	}
	return nil
}

func getManagedTemplate(ctx context.Context, queryer ruleQueryer, id uuid.UUID, lock bool) (*ManagedTemplate, error) {
	builder := storage.Psql.Select(managedTemplateColumns...).From("notification_templates").Where(sq.Eq{"id": id})
	if lock {
		builder = builder.Suffix("FOR UPDATE")
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build versioned notification template lookup: %w", err)
	}
	item, err := scanManagedTemplate(queryer.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, commonerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get versioned notification template: %w", err)
	}
	versions := make(map[uuid.UUID]*ManagedTemplateVersion, 2)
	for _, reference := range []*uuid.UUID{item.CurrentDraftVersionID, item.CurrentPublishedVersionID} {
		if reference == nil {
			continue
		}
		version, exists := versions[*reference]
		if !exists {
			version, err = getManagedTemplateVersion(ctx, queryer, *reference)
			if err != nil {
				return nil, err
			}
			versions[*reference] = version
		}
		if item.CurrentDraftVersionID != nil && *reference == *item.CurrentDraftVersionID {
			item.Draft = version
		}
		if item.CurrentPublishedVersionID != nil && *reference == *item.CurrentPublishedVersionID {
			item.Published = version
		}
	}
	return &item, nil
}

func getManagedTemplateVersion(ctx context.Context, queryer ruleQueryer, id uuid.UUID) (*ManagedTemplateVersion, error) {
	query, args, err := storage.Psql.Select(
		"id", "template_id", "version_no", "channel", "language", "subject", "text_body", "html_body",
		"variables", "created_by", "change_reason", "created_at", "published_at",
	).From("notification_template_versions").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification template version lookup: %w", err)
	}
	var version ManagedTemplateVersion
	if err := queryer.QueryRow(ctx, query, args...).Scan(
		&version.ID, &version.TemplateID, &version.VersionNo, &version.Channel, &version.Language,
		&version.Subject, &version.TextBody, &version.HTMLBody, &version.Variables, &version.CreatedBy,
		&version.ChangeReason, &version.CreatedAt, &version.PublishedAt,
	); err != nil {
		return nil, fmt.Errorf("get notification template version: %w", err)
	}
	return &version, nil
}

func scanManagedTemplate(row interface{ Scan(...any) error }) (ManagedTemplate, error) {
	var item ManagedTemplate
	err := row.Scan(
		&item.ID, &item.Name, &item.Channel, &item.Language, &item.Revision, &item.CurrentDraftVersionID,
		&item.CurrentPublishedVersionID, &item.Archived, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func commitAndReadManagedTemplate(ctx context.Context, tx pgx.Tx, id uuid.UUID, operation string) (*ManagedTemplate, error) {
	item, err := getManagedTemplate(ctx, tx, id, false)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit %s: %w", operation, err)
	}
	return item, nil
}

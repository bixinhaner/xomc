package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var _ RuleRepository = (*PgRuleRepository)(nil)

type PgRuleRepository struct{ db storage.DB }

func NewPgRuleRepository(pool *pgxpool.Pool) *PgRuleRepository {
	return &PgRuleRepository{db: storage.NewPoolDB(pool)}
}

func newPgRuleRepository(db storage.DB) *PgRuleRepository { return &PgRuleRepository{db: db} }

type ruleQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

var ruleColumns = []string{
	"id", "name", "revision", "current_draft_version_id", "current_published_version_id",
	"current_enabled_version_id", "priority", "archived", "created_by", "created_at", "updated_at",
}

func (r *PgRuleRepository) List(ctx context.Context) ([]NotificationRule, error) {
	query, args, err := storage.Psql.Select(ruleColumns...).From("notification_rules").OrderBy("priority", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification rule list: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notification rules: %w", err)
	}
	defer rows.Close()
	rules := make([]NotificationRule, 0)
	for rows.Next() {
		rule, scanErr := scanNotificationRule(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan notification rule: %w", scanErr)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification rules: %w", err)
	}
	return rules, nil
}

func (r *PgRuleRepository) Get(ctx context.Context, id uuid.UUID) (*NotificationRule, error) {
	rule, err := getNotificationRule(ctx, r.db, id, false)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *PgRuleRepository) Create(ctx context.Context, input RuleDraftInput, actor string) (*NotificationRule, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin notification rule create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now, ruleID, versionID := time.Now().UTC(), uuid.New(), uuid.New()
	conditions, err := json.Marshal(input.MatchConditions)
	if err != nil {
		return nil, fmt.Errorf("marshal notification rule conditions: %w", err)
	}
	query, args, err := storage.Psql.Insert("notification_rules").Columns(
		"id", "name", "revision", "current_draft_version_id", "priority", "archived", "created_by", "created_at", "updated_at",
	).Values(ruleID, input.Name, 1, versionID, input.Priority, false, actor, now, now).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification rule create: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return nil, mapRuleWriteError("create notification rule", err)
	}
	if err := insertNotificationRuleVersion(ctx, tx, NotificationRuleVersion{
		ID: versionID, RuleID: ruleID, VersionNo: 1, MatchConditions: input.MatchConditions,
		Policy: input.Policy, CreatedBy: actor, ChangeReason: input.ChangeReason, CreatedAt: now,
	}, conditions); err != nil {
		return nil, err
	}
	if err := insertRuleVersionBindings(ctx, tx, versionID, input, now); err != nil {
		return nil, err
	}
	rule, err := getNotificationRule(ctx, tx, ruleID, false)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit notification rule create: %w", err)
	}
	return rule, nil
}

func (r *PgRuleRepository) UpdateDraft(
	ctx context.Context,
	id uuid.UUID,
	expectedRevision int64,
	input RuleDraftInput,
	actor string,
) (*NotificationRule, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin notification rule draft update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rule, err := getNotificationRule(ctx, tx, id, true)
	if err != nil {
		return nil, err
	}
	if rule.Revision != expectedRevision {
		return nil, ErrRevisionMismatch
	}
	if rule.Archived {
		return nil, commonerrors.ErrInvalidInput
	}
	var nextVersion int64
	versionQuery, versionArgs, err := storage.Psql.Select("COALESCE(MAX(version_no), 0) + 1").
		From("notification_rule_versions").Where(sq.Eq{"rule_id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build next notification rule version: %w", err)
	}
	if err := tx.QueryRow(ctx, versionQuery, versionArgs...).Scan(&nextVersion); err != nil {
		return nil, fmt.Errorf("read next notification rule version: %w", err)
	}
	conditions, err := json.Marshal(input.MatchConditions)
	if err != nil {
		return nil, fmt.Errorf("marshal notification rule conditions: %w", err)
	}
	versionID, now := uuid.New(), time.Now().UTC()
	if err := insertNotificationRuleVersion(ctx, tx, NotificationRuleVersion{
		ID: versionID, RuleID: id, VersionNo: nextVersion, MatchConditions: input.MatchConditions,
		Policy: input.Policy, CreatedBy: actor, ChangeReason: input.ChangeReason, CreatedAt: now,
	}, conditions); err != nil {
		return nil, err
	}
	if err := insertRuleVersionBindings(ctx, tx, versionID, input, now); err != nil {
		return nil, err
	}
	updateQuery, updateArgs, err := storage.Psql.Update("notification_rules").
		Set("name", input.Name).Set("priority", input.Priority).Set("current_draft_version_id", versionID).
		Set("revision", expectedRevision+1).Set("updated_at", now).Where(sq.Eq{"id": id, "revision": expectedRevision}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification rule draft pointer update: %w", err)
	}
	if tag, execErr := tx.Exec(ctx, updateQuery, updateArgs...); execErr != nil {
		return nil, mapRuleWriteError("update notification rule draft", execErr)
	} else if tag.RowsAffected() != 1 {
		return nil, ErrRevisionMismatch
	}
	return commitAndReadRule(ctx, tx, id, "notification rule draft update")
}

func (r *PgRuleRepository) Publish(ctx context.Context, id uuid.UUID, expectedRevision int64, actor string) (*NotificationRule, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin notification rule publish: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rule, err := getNotificationRule(ctx, tx, id, true)
	if err != nil {
		return nil, err
	}
	if rule.Revision != expectedRevision {
		return nil, ErrRevisionMismatch
	}
	if rule.Archived || rule.CurrentDraftVersionID == nil {
		return nil, ErrDraftUnavailable
	}
	if rule.CurrentPublishedVersionID != nil && *rule.CurrentPublishedVersionID == *rule.CurrentDraftVersionID {
		return nil, ErrDraftUnavailable
	}
	if err := validateRuleVersionBindings(ctx, tx, *rule.CurrentDraftVersionID, false); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	publishQuery, publishArgs, err := storage.Psql.Update("notification_rule_versions").
		Set("published_at", now).
		Where(sq.Eq{"id": *rule.CurrentDraftVersionID, "rule_id": id, "published_at": nil}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification rule version publish: %w", err)
	}
	if tag, execErr := tx.Exec(ctx, publishQuery, publishArgs...); execErr != nil {
		return nil, fmt.Errorf("publish notification rule version: %w", execErr)
	} else if tag.RowsAffected() != 1 {
		return nil, ErrDraftUnavailable
	}
	updateQuery, updateArgs, err := storage.Psql.Update("notification_rules").
		Set("current_published_version_id", *rule.CurrentDraftVersionID).
		Set("revision", expectedRevision+1).Set("updated_at", now).
		Where(sq.Eq{"id": id, "revision": expectedRevision}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification rule publish pointer update: %w", err)
	}
	if tag, execErr := tx.Exec(ctx, updateQuery, updateArgs...); execErr != nil {
		return nil, fmt.Errorf("update notification rule publish pointer: %w", execErr)
	} else if tag.RowsAffected() != 1 {
		return nil, ErrRevisionMismatch
	}
	_ = actor // publication actor is already retained on the immutable draft version.
	return commitAndReadRule(ctx, tx, id, "notification rule publish")
}

func (r *PgRuleRepository) Enable(ctx context.Context, id uuid.UUID, expectedRevision int64, versionID uuid.UUID) (*NotificationRule, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin notification rule enable: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rule, err := getNotificationRule(ctx, tx, id, true)
	if err != nil {
		return nil, err
	}
	if rule.Revision != expectedRevision {
		return nil, ErrRevisionMismatch
	}
	if rule.Archived {
		return nil, commonerrors.ErrInvalidInput
	}
	var publishedAt *time.Time
	versionQuery, versionArgs, err := storage.Psql.Select("published_at").From("notification_rule_versions").
		Where(sq.Eq{"id": versionID, "rule_id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build enabled notification rule version lookup: %w", err)
	}
	if err := tx.QueryRow(ctx, versionQuery, versionArgs...).Scan(&publishedAt); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVersionUnpublished
	} else if err != nil {
		return nil, fmt.Errorf("read enabled notification rule version: %w", err)
	}
	if publishedAt == nil {
		return nil, ErrVersionUnpublished
	}
	if err := validateRuleVersionBindings(ctx, tx, versionID, true); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	query, args, err := storage.Psql.Update("notification_rules").
		Set("current_enabled_version_id", versionID).Set("revision", expectedRevision+1).Set("updated_at", now).
		Where(sq.Eq{"id": id, "revision": expectedRevision}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification rule enable: %w", err)
	}
	if tag, execErr := tx.Exec(ctx, query, args...); execErr != nil {
		return nil, fmt.Errorf("enable notification rule: %w", execErr)
	} else if tag.RowsAffected() != 1 {
		return nil, ErrRevisionMismatch
	}
	return commitAndReadRule(ctx, tx, id, "notification rule enable")
}

func (r *PgRuleRepository) Archive(ctx context.Context, id uuid.UUID, expectedRevision int64) (*NotificationRule, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin notification rule archive: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rule, err := getNotificationRule(ctx, tx, id, true)
	if err != nil {
		return nil, err
	}
	if rule.Revision != expectedRevision {
		return nil, ErrRevisionMismatch
	}
	now := time.Now().UTC()
	query, args, err := storage.Psql.Update("notification_rules").
		Set("archived", true).Set("current_enabled_version_id", nil).
		Set("revision", expectedRevision+1).Set("updated_at", now).
		Where(sq.Eq{"id": id, "revision": expectedRevision}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification rule archive: %w", err)
	}
	if tag, execErr := tx.Exec(ctx, query, args...); execErr != nil {
		return nil, fmt.Errorf("archive notification rule: %w", execErr)
	} else if tag.RowsAffected() != 1 {
		return nil, ErrRevisionMismatch
	}
	return commitAndReadRule(ctx, tx, id, "notification rule archive")
}

func getNotificationRule(ctx context.Context, queryer ruleQueryer, id uuid.UUID, lock bool) (*NotificationRule, error) {
	builder := storage.Psql.Select(ruleColumns...).From("notification_rules").Where(sq.Eq{"id": id})
	if lock {
		builder = builder.Suffix("FOR UPDATE")
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification rule lookup: %w", err)
	}
	rule, err := scanNotificationRule(queryer.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, commonerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get notification rule: %w", err)
	}
	versions := make(map[uuid.UUID]*NotificationRuleVersion, 3)
	for _, ref := range []*uuid.UUID{rule.CurrentDraftVersionID, rule.CurrentPublishedVersionID, rule.CurrentEnabledVersionID} {
		if ref == nil {
			continue
		}
		version, exists := versions[*ref]
		if !exists {
			version, err = getNotificationRuleVersion(ctx, queryer, *ref)
			if err != nil {
				return nil, err
			}
			versions[*ref] = version
		}
		switch {
		case rule.CurrentDraftVersionID != nil && *ref == *rule.CurrentDraftVersionID:
			rule.Draft = version
		}
		if rule.CurrentPublishedVersionID != nil && *ref == *rule.CurrentPublishedVersionID {
			rule.Published = version
		}
		if rule.CurrentEnabledVersionID != nil && *ref == *rule.CurrentEnabledVersionID {
			rule.Enabled = version
		}
	}
	return &rule, nil
}

func getNotificationRuleVersion(ctx context.Context, queryer ruleQueryer, id uuid.UUID) (*NotificationRuleVersion, error) {
	query, args, err := storage.Psql.Select(
		"id", "rule_id", "version_no", "match_conditions", "policy", "created_by", "change_reason", "created_at", "published_at",
	).From("notification_rule_versions").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification rule version lookup: %w", err)
	}
	var version NotificationRuleVersion
	var conditions []byte
	if err := queryer.QueryRow(ctx, query, args...).Scan(
		&version.ID, &version.RuleID, &version.VersionNo, &conditions, &version.Policy,
		&version.CreatedBy, &version.ChangeReason, &version.CreatedAt, &version.PublishedAt,
	); err != nil {
		return nil, fmt.Errorf("get notification rule version: %w", err)
	}
	if err := json.Unmarshal(conditions, &version.MatchConditions); err != nil {
		return nil, fmt.Errorf("decode notification rule match conditions: %w", err)
	}
	if err := hydrateRuleVersionBindings(ctx, queryer, &version); err != nil {
		return nil, err
	}
	return &version, nil
}

func scanNotificationRule(row interface{ Scan(...any) error }) (NotificationRule, error) {
	var rule NotificationRule
	err := row.Scan(
		&rule.ID, &rule.Name, &rule.Revision, &rule.CurrentDraftVersionID,
		&rule.CurrentPublishedVersionID, &rule.CurrentEnabledVersionID, &rule.Priority,
		&rule.Archived, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)
	return rule, err
}

func insertNotificationRuleVersion(
	ctx context.Context,
	queryer ruleQueryer,
	version NotificationRuleVersion,
	conditions []byte,
) error {
	query, args, err := storage.Psql.Insert("notification_rule_versions").Columns(
		"id", "rule_id", "version_no", "match_conditions", "policy", "created_by", "change_reason", "created_at", "published_at",
	).Values(
		version.ID, version.RuleID, version.VersionNo, conditions, version.Policy, version.CreatedBy,
		version.ChangeReason, version.CreatedAt, version.PublishedAt,
	).ToSql()
	if err != nil {
		return fmt.Errorf("build notification rule version insert: %w", err)
	}
	if _, err := queryer.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert notification rule version: %w", err)
	}
	return nil
}

func insertRuleVersionBindings(ctx context.Context, queryer ruleQueryer, versionID uuid.UUID, input RuleDraftInput, now time.Time) error {
	for _, recipient := range input.Recipients {
		var targetID any
		if recipient.TargetID != "" {
			targetID = recipient.TargetID
		}
		var ciphertext, keyVersion, fingerprint any
		if len(recipient.AddressCiphertext) > 0 {
			ciphertext, keyVersion, fingerprint = recipient.AddressCiphertext, recipient.AddressKeyVersion, recipient.RecipientFingerprint
		}
		query, args, err := storage.Psql.Insert("notification_rule_recipients").Columns(
			"id", "rule_version_id", "target_type", "target_id", "address_ciphertext",
			"address_key_version", "recipient_fingerprint", "channel_limit", "created_at",
		).Values(
			uuid.New(), versionID, recipient.TargetType, targetID, ciphertext,
			keyVersion, fingerprint, recipient.ChannelLimit, now,
		).ToSql()
		if err != nil {
			return fmt.Errorf("build notification rule recipient insert: %w", err)
		}
		if _, err := queryer.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert notification rule recipient: %w", err)
		}
	}
	for _, channel := range input.Channels {
		query, args, err := storage.Psql.Insert("notification_rule_channels").Columns(
			"id", "rule_version_id", "channel", "channel_config_id", "raised_template_version_id",
			"escalated_template_version_id", "cleared_template_version_id", "policy", "created_at",
		).Values(
			uuid.New(), versionID, channel.Channel, channel.ChannelConfigID, channel.RaisedTemplateVersionID,
			channel.EscalatedTemplateVersionID, channel.ClearedTemplateVersionID, channel.Policy, now,
		).ToSql()
		if err != nil {
			return fmt.Errorf("build notification rule channel insert: %w", err)
		}
		if _, err := queryer.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert notification rule channel: %w", err)
		}
	}
	return nil
}

func hydrateRuleVersionBindings(ctx context.Context, queryer ruleQueryer, version *NotificationRuleVersion) error {
	recipientQuery, recipientArgs, err := storage.Psql.Select(
		"id", "target_type", "target_id", "address_ciphertext IS NOT NULL", "channel_limit",
	).From("notification_rule_recipients").Where(sq.Eq{"rule_version_id": version.ID}).OrderBy("created_at", "id").ToSql()
	if err != nil {
		return fmt.Errorf("build notification rule recipients lookup: %w", err)
	}
	rows, err := queryer.Query(ctx, recipientQuery, recipientArgs...)
	if err != nil {
		return fmt.Errorf("get notification rule recipients: %w", err)
	}
	version.Recipients = make([]RuleRecipient, 0)
	for rows.Next() {
		var recipient RuleRecipient
		var targetID *string
		if err := rows.Scan(&recipient.ID, &recipient.TargetType, &targetID, &recipient.AddressConfigured, &recipient.ChannelLimit); err != nil {
			rows.Close()
			return fmt.Errorf("scan notification rule recipient: %w", err)
		}
		if targetID != nil {
			recipient.TargetID = *targetID
		}
		version.Recipients = append(version.Recipients, recipient)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate notification rule recipients: %w", err)
	}
	rows.Close()

	channelQuery, channelArgs, err := storage.Psql.Select(
		"id", "channel", "channel_config_id", "raised_template_version_id",
		"escalated_template_version_id", "cleared_template_version_id", "policy",
	).From("notification_rule_channels").Where(sq.Eq{"rule_version_id": version.ID}).OrderBy("channel", "id").ToSql()
	if err != nil {
		return fmt.Errorf("build notification rule channels lookup: %w", err)
	}
	rows, err = queryer.Query(ctx, channelQuery, channelArgs...)
	if err != nil {
		return fmt.Errorf("get notification rule channels: %w", err)
	}
	defer rows.Close()
	version.Channels = make([]RuleChannel, 0)
	for rows.Next() {
		var channel RuleChannel
		if err := rows.Scan(
			&channel.ID, &channel.Channel, &channel.ChannelConfigID, &channel.RaisedTemplateVersionID,
			&channel.EscalatedTemplateVersionID, &channel.ClearedTemplateVersionID, &channel.Policy,
		); err != nil {
			return fmt.Errorf("scan notification rule channel: %w", err)
		}
		version.Channels = append(version.Channels, channel)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate notification rule channels: %w", err)
	}
	return nil
}

func validateRuleVersionBindings(ctx context.Context, queryer ruleQueryer, versionID uuid.UUID, requireEnabled bool) error {
	var recipientCount int
	countQuery, countArgs, err := storage.Psql.Select("COUNT(*)").From("notification_rule_recipients").
		Where(sq.Eq{"rule_version_id": versionID}).ToSql()
	if err != nil {
		return fmt.Errorf("build notification rule recipient count: %w", err)
	}
	if err := queryer.QueryRow(ctx, countQuery, countArgs...).Scan(&recipientCount); err != nil {
		return fmt.Errorf("count notification rule recipients: %w", err)
	}
	query, args, err := storage.Psql.Select(
		"rc.channel", "cfg.channel", "cfg.enabled",
		"raised.channel", "raised.published_at",
		"escalated.channel", "escalated.published_at",
		"cleared.channel", "cleared.published_at",
	).From("notification_rule_channels rc").
		Join("notification_channel_configs cfg ON cfg.id = rc.channel_config_id").
		LeftJoin("notification_template_versions raised ON raised.id = rc.raised_template_version_id").
		LeftJoin("notification_template_versions escalated ON escalated.id = rc.escalated_template_version_id").
		LeftJoin("notification_template_versions cleared ON cleared.id = rc.cleared_template_version_id").
		Where(sq.Eq{"rc.rule_version_id": versionID}).ToSql()
	if err != nil {
		return fmt.Errorf("build notification rule channel validation: %w", err)
	}
	rows, err := queryer.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("validate notification rule channels: %w", err)
	}
	defer rows.Close()
	channelCount := 0
	for rows.Next() {
		channelCount++
		var channel, configChannel string
		var enabled bool
		var raisedChannel, escalatedChannel, clearedChannel *string
		var raisedPublishedAt, escalatedPublishedAt, clearedPublishedAt *time.Time
		if err := rows.Scan(
			&channel, &configChannel, &enabled, &raisedChannel, &raisedPublishedAt,
			&escalatedChannel, &escalatedPublishedAt, &clearedChannel, &clearedPublishedAt,
		); err != nil {
			return fmt.Errorf("scan notification rule channel validation: %w", err)
		}
		if channel != configChannel || raisedChannel == nil || raisedPublishedAt == nil || !templateChannelMatches(channel, *raisedChannel) ||
			(escalatedChannel != nil && (escalatedPublishedAt == nil || !templateChannelMatches(channel, *escalatedChannel))) ||
			(clearedChannel != nil && (clearedPublishedAt == nil || !templateChannelMatches(channel, *clearedChannel))) {
			return ErrVersionUnpublished
		}
		if requireEnabled && !enabled {
			return ErrChannelDisabled
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate notification rule channel validation: %w", err)
	}
	if recipientCount == 0 || channelCount == 0 {
		return ErrRuleIncomplete
	}
	return nil
}

func templateChannelMatches(ruleChannel, templateChannel string) bool {
	if ruleChannel == TemplateChannelEmail {
		return templateChannel == TemplateChannelEmail
	}
	return strings.HasPrefix(ruleChannel, "sms_") && templateChannel == TemplateChannelSMS
}

func commitAndReadRule(ctx context.Context, tx pgx.Tx, id uuid.UUID, operation string) (*NotificationRule, error) {
	rule, err := getNotificationRule(ctx, tx, id, false)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit %s: %w", operation, err)
	}
	return rule, nil
}

func mapRuleWriteError(context string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return commonerrors.ErrAlreadyExists
	}
	return fmt.Errorf("%s: %w", context, err)
}

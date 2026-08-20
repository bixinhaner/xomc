package deviceaccess

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var (
	ErrCurrentDecisionArchive  = errors.New("current effective decision cannot be archived")
	ErrDecisionArchiveConflict = errors.New("device access decision archive state conflict")
)

func (s *PgManagementStore) ListDecisions(
	ctx context.Context,
	filter ManagementFilter,
) ([]DecisionItem, int64, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := decisionFilters(filter)
	countBuilder := storage.Psql.Select("COUNT(*)").From("device_access_decisions decision").Where(where)
	countBuilder = applyAccessIdentityVisibility(
		countBuilder, "decision.device_id", "decision.carrier", "decision.serial_number", filter.VisibleGroups,
	)
	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build decision history count: %w", err)
	}
	var total int64
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count decision history: %w", err)
	}

	listBuilder := storage.Psql.Select(
		"decision.id", "decision.trigger_type", "decision.previous_state", "decision.new_state",
		"decision.decision", "decision.reason_code", "decision.policy_version_id",
		"decision.evidence_version", "decision.decision_version", "decision.matched_list_entry_id",
		"decision.matched_rule_id", "decision.occurred_at",
		"archive.id", "COALESCE(archive.archived_by::text, '')", "COALESCE(archive.reason, '')", "archive.archived_at",
	).From("device_access_decisions decision").
		LeftJoin("device_access_decision_archives archive ON archive.decision_id = decision.id AND archive.restored_at IS NULL").
		Where(where).OrderBy("decision.decision_version DESC", "decision.id DESC").
		Limit(uint64(pageSize)).Offset(uint64((page - 1) * pageSize))
	listBuilder = applyAccessIdentityVisibility(
		listBuilder, "decision.device_id", "decision.carrier", "decision.serial_number", filter.VisibleGroups,
	)
	query, args, err := listBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build decision history: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query decision history: %w", err)
	}
	defer rows.Close()
	items := make([]DecisionItem, 0, pageSize)
	for rows.Next() {
		var item DecisionItem
		var archiveID *uuid.UUID
		var archivedBy, archiveReason string
		var archivedAt *time.Time
		if err := rows.Scan(
			&item.ID, &item.TriggerType, &item.PreviousState, &item.NewState, &item.Decision,
			&item.ReasonCode, &item.PolicyVersionID, &item.EvidenceVersion, &item.DecisionVersion,
			&item.MatchedListEntry, &item.MatchedRule, &item.OccurredAt,
			&archiveID, &archivedBy, &archiveReason, &archivedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan decision history: %w", err)
		}
		if archiveID != nil && archivedAt != nil {
			item.Archive = &DecisionArchive{
				ID: archiveID.String(), DecisionID: item.ID.String(), ArchivedBy: archivedBy,
				Reason: archiveReason, ArchivedAt: *archivedAt,
			}
		}
		item.Checks, err = s.loadDecisionChecks(ctx, item.ID)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate decision history: %w", err)
	}
	return items, total, nil
}

func decisionFilters(filter ManagementFilter) sq.And {
	where := sq.And{sq.Eq{"decision.carrier": strings.TrimSpace(filter.Carrier)}}
	if value := strings.TrimSpace(filter.SerialNumber); value != "" {
		where = append(where, sq.Eq{"decision.serial_number": value})
	}
	if filter.Decision != "" {
		where = append(where, sq.Eq{"decision.decision": filter.Decision})
	}
	if filter.ReasonCode != "" {
		where = append(where, sq.Eq{"decision.reason_code": filter.ReasonCode})
	}
	if filter.PolicyVersionID != nil {
		where = append(where, sq.Eq{"decision.policy_version_id": *filter.PolicyVersionID})
	}
	if filter.MatchedRuleID != nil {
		where = append(where, sq.Eq{"decision.matched_rule_id": *filter.MatchedRuleID})
	}
	if filter.StartedAt != nil {
		where = append(where, sq.GtOrEq{"decision.occurred_at": *filter.StartedAt})
	}
	if filter.EndedAt != nil {
		where = append(where, sq.LtOrEq{"decision.occurred_at": *filter.EndedAt})
	}
	if filter.Dimension != "" {
		where = append(where, sq.Expr(`EXISTS (
			SELECT 1 FROM device_access_decision_checks dimension_check
			WHERE dimension_check.decision_id = decision.id AND dimension_check.check_type = ?
		)`, filter.Dimension))
	}
	switch strings.TrimSpace(filter.ArchiveStatus) {
	case "archived":
		where = append(where, sq.Expr(`EXISTS (
			SELECT 1 FROM device_access_decision_archives active_archive
			WHERE active_archive.decision_id = decision.id AND active_archive.restored_at IS NULL
		)`))
	case "all":
	default:
		where = append(where, sq.Expr(`NOT EXISTS (
			SELECT 1 FROM device_access_decision_archives active_archive
			WHERE active_archive.decision_id = decision.id AND active_archive.restored_at IS NULL
		)`))
	}
	return where
}

func (s *PgManagementStore) ArchiveDecision(
	ctx context.Context,
	carrier string,
	decisionID, actorID uuid.UUID,
	visibleGroups []uuid.UUID,
	reason string,
) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Errorf("archive decision: reason is required")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin decision archive: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	decisionVersion, currentVersion, err := lockDecisionForArchive(
		ctx, tx, carrier, decisionID, visibleGroups,
	)
	if err != nil {
		return err
	}
	if decisionVersion >= currentVersion {
		return ErrCurrentDecisionArchive
	}
	now := time.Now().UTC()
	query, args, err := storage.Psql.Insert("device_access_decision_archives").
		Columns("id", "decision_id", "archived_by", "reason", "archived_at").
		Values(uuid.New(), decisionID, actorID, reason, now).
		Suffix("ON CONFLICT (decision_id) WHERE restored_at IS NULL DO NOTHING").ToSql()
	if err != nil {
		return fmt.Errorf("build decision archive insert: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("archive decision: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrDecisionArchiveConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit decision archive: %w", err)
	}
	return nil
}

func (s *PgManagementStore) RestoreDecision(
	ctx context.Context,
	carrier string,
	decisionID, actorID uuid.UUID,
	visibleGroups []uuid.UUID,
) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin decision archive restore: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, _, err := lockDecisionForArchive(ctx, tx, carrier, decisionID, visibleGroups); err != nil {
		return err
	}
	now := time.Now().UTC()
	query, args, err := storage.Psql.Update("device_access_decision_archives").
		Set("restored_by", actorID).Set("restored_at", now).
		Where(sq.Eq{"decision_id": decisionID}).Where("restored_at IS NULL").ToSql()
	if err != nil {
		return fmt.Errorf("build decision archive restore: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("restore decision archive: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrDecisionArchiveConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit decision archive restore: %w", err)
	}
	return nil
}

func lockDecisionForArchive(
	ctx context.Context,
	tx pgx.Tx,
	carrier string,
	decisionID uuid.UUID,
	visibleGroups []uuid.UUID,
) (int64, int64, error) {
	builder := storage.Psql.Select("decision.decision_version", "state.decision_version").
		From("device_access_decisions decision").
		Join("device_access_states state ON state.carrier = decision.carrier AND state.serial_number = decision.serial_number").
		Where(sq.Eq{"decision.id": decisionID, "decision.carrier": strings.TrimSpace(carrier)})
	builder = applyAccessIdentityVisibility(
		builder, "decision.device_id", "decision.carrier", "decision.serial_number", visibleGroups,
	)
	query, args, err := builder.Suffix("FOR UPDATE OF decision").ToSql()
	if err != nil {
		return 0, 0, fmt.Errorf("build decision archive lock: %w", err)
	}
	var decisionVersion, currentVersion int64
	if err := tx.QueryRow(ctx, query, args...).Scan(&decisionVersion, &currentVersion); err != nil {
		return 0, 0, fmt.Errorf("lock decision for archive: %w", err)
	}
	return decisionVersion, currentVersion, nil
}

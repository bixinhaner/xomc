package deviceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/storage"
)

func (s *PgManagementStore) ListIdentitySnapshots(
	ctx context.Context,
	filter ManagementFilter,
) ([]IdentitySnapshot, int64, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := sq.And{sq.Eq{
		"snapshot.carrier":       strings.TrimSpace(filter.Carrier),
		"snapshot.serial_number": strings.TrimSpace(filter.SerialNumber),
	}}
	countBuilder := storage.Psql.Select("COUNT(*)").From("device_access_identity_snapshots snapshot").Where(where)
	countBuilder = applyAccessIdentityVisibility(
		countBuilder, "snapshot.device_id", "snapshot.carrier", "snapshot.serial_number", filter.VisibleGroups,
	)
	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build identity snapshot count: %w", err)
	}
	var total int64
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count identity snapshots: %w", err)
	}
	listBuilder := storage.Psql.Select(
		"snapshot.id::text", "snapshot.request_id", "COALESCE(snapshot.decision_id::text, '')",
		"COALESCE(snapshot.device_id::text, '')", "COALESCE(snapshot.candidate_id::text, '')",
		"snapshot.carrier", "snapshot.serial_number", "COALESCE(snapshot.device_code, '')",
		"COALESCE(snapshot.cloud_key, '')", "COALESCE(snapshot.oui, '')", "COALESCE(snapshot.product_class, '')",
		"snapshot.raw_remote_ip", "snapshot.observed_remote_ip", "COALESCE(snapshot.inform_event, '')",
		"snapshot.inform_time", "snapshot.identity_source", "snapshot.identity_status",
		"COALESCE(snapshot.identity_reason_code, '')", "snapshot.created_at",
	).From("device_access_identity_snapshots snapshot").Where(where).
		OrderBy("snapshot.inform_time DESC", "snapshot.id DESC").
		Limit(uint64(pageSize)).Offset(uint64((page - 1) * pageSize))
	listBuilder = applyAccessIdentityVisibility(
		listBuilder, "snapshot.device_id", "snapshot.carrier", "snapshot.serial_number", filter.VisibleGroups,
	)
	query, args, err := listBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build identity snapshot list: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query identity snapshots: %w", err)
	}
	defer rows.Close()
	items := make([]IdentitySnapshot, 0, pageSize)
	for rows.Next() {
		var item IdentitySnapshot
		if err := rows.Scan(
			&item.ID, &item.RequestID, &item.DecisionID, &item.DeviceID, &item.CandidateID,
			&item.Carrier, &item.SerialNumber, &item.DeviceCode, &item.CloudKey, &item.OUI, &item.ProductClass,
			&item.RawRemoteIP, &item.ObservedRemoteIP, &item.InformEvent, &item.InformTime,
			&item.IdentitySource, &item.IdentityStatus, &item.IdentityReasonCode, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan identity snapshot: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate identity snapshots: %w", err)
	}
	return items, total, nil
}

func (s *PgManagementStore) ListEvidence(
	ctx context.Context,
	filter ManagementFilter,
) ([]EvidenceItem, int64, error) {
	if err := s.ensureStateVisible(ctx, filter); err != nil {
		return nil, 0, err
	}
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := sq.Eq{
		"carrier":       strings.TrimSpace(filter.Carrier),
		"serial_number": strings.TrimSpace(filter.SerialNumber),
	}
	total, err := countRows(ctx, s.db, "device_access_evidence", where)
	if err != nil {
		return nil, 0, fmt.Errorf("count access evidence: %w", err)
	}
	query, args, err := storage.Psql.Select(
		"id", "evidence_version", "evidence_type", "evidence_status", "normalized_value", "source",
		"task_id", "observed_at", "expires_at",
	).From("device_access_evidence").Where(where).
		OrderBy("evidence_version DESC", "observed_at DESC", "id DESC").
		Limit(uint64(pageSize)).Offset(uint64((page - 1) * pageSize)).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build paged access evidence: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query paged access evidence: %w", err)
	}
	defer rows.Close()
	items := make([]EvidenceItem, 0, pageSize)
	for rows.Next() {
		var item EvidenceItem
		if err := rows.Scan(
			&item.ID, &item.EvidenceVersion, &item.EvidenceType, &item.EvidenceStatus,
			&item.NormalizedValue, &item.Source, &item.TaskID, &item.ObservedAt, &item.ExpiresAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan paged access evidence: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate paged access evidence: %w", err)
	}
	return items, total, nil
}

func (s *PgManagementStore) ListNotifications(
	ctx context.Context,
	filter ManagementFilter,
) ([]AccessNotificationItem, int64, error) {
	if err := s.ensureStateVisible(ctx, filter); err != nil {
		return nil, 0, err
	}
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := sq.And{
		sq.Eq{
			"decision.carrier":       strings.TrimSpace(filter.Carrier),
			"decision.serial_number": strings.TrimSpace(filter.SerialNumber),
		},
		sq.Eq{"history.source_type": "device_access_action"},
	}
	countBuilder := storage.Psql.Select("COUNT(*)").From("notification_history history").
		Join("device_access_actions action ON action.id = history.source_id").
		Join("device_access_decisions decision ON decision.id = action.decision_id").
		Where(where)
	countBuilder = applyAccessIdentityVisibility(
		countBuilder, "decision.device_id", "decision.carrier", "decision.serial_number", filter.VisibleGroups,
	)
	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build access notification count: %w", err)
	}
	var total int64
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count access notifications: %w", err)
	}
	listBuilder := storage.Psql.Select(
		"history.id", "history.channel", "history.recipients", "history.subject", "history.status",
		"COALESCE(history.error_message, '')", "history.source_type", "history.source_id", "history.event_id",
		"history.correlation_id", "history.retry_count", "history.sent_at", "history.created_at",
	).From("notification_history history").
		Join("device_access_actions action ON action.id = history.source_id").
		Join("device_access_decisions decision ON decision.id = action.decision_id").
		Where(where).OrderBy("history.created_at DESC", "history.id DESC").
		Limit(uint64(pageSize)).Offset(uint64((page - 1) * pageSize))
	listBuilder = applyAccessIdentityVisibility(
		listBuilder, "decision.device_id", "decision.carrier", "decision.serial_number", filter.VisibleGroups,
	)
	query, args, err := listBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build access notification list: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query access notifications: %w", err)
	}
	defer rows.Close()
	items := make([]AccessNotificationItem, 0, pageSize)
	for rows.Next() {
		var item AccessNotificationItem
		if err := rows.Scan(
			&item.ID, &item.Channel, &item.Recipients, &item.Subject, &item.Status,
			&item.ErrorMessage, &item.SourceType, &item.SourceID, &item.EventID,
			&item.CorrelationID, &item.RetryCount, &item.SentAt, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan access notification: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate access notifications: %w", err)
	}
	return items, total, nil
}

func (s *PgManagementStore) ListManualOperations(
	ctx context.Context,
	filter ManagementFilter,
) ([]AccessManualOperationItem, int64, error) {
	if err := s.ensureStateVisible(ctx, filter); err != nil {
		return nil, 0, err
	}
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	carrier := strings.TrimSpace(filter.Carrier)
	serialNumber := strings.TrimSpace(filter.SerialNumber)
	where := sq.And{
		sq.Eq{"resource": "device_access"},
		sq.Expr("details ->> 'carrier' = ?", carrier),
		sq.Or{
			sq.Eq{"resource_id": serialNumber},
			sq.Expr(`resource_id IN (
				SELECT decision.id::text FROM device_access_decisions decision
				WHERE decision.carrier = ? AND decision.serial_number = ?
			)`, carrier, serialNumber),
			sq.Expr(`resource_id IN (
				SELECT action.id::text FROM device_access_actions action
				JOIN device_access_decisions decision ON decision.id = action.decision_id
				WHERE decision.carrier = ? AND decision.serial_number = ?
			)`, carrier, serialNumber),
			sq.Expr(`resource_id IN (
				SELECT candidate.id::text FROM device_access_candidates candidate
				WHERE candidate.carrier = ? AND candidate.serial_number = ?
			)`, carrier, serialNumber),
			sq.Expr("jsonb_exists(details -> 'serial_numbers', ?)", serialNumber),
		},
	}
	total, err := countRows(ctx, s.db, "audit_logs", where)
	if err != nil {
		return nil, 0, fmt.Errorf("count access manual operations: %w", err)
	}
	query, args, err := storage.Psql.Select(
		"id::text", "COALESCE(user_id::text, '')", "username",
		"COALESCE(details ->> 'operation', action)", "COALESCE(resource_id, '')",
		"COALESCE(details, '{}'::jsonb)", "created_at",
	).From("audit_logs").Where(where).OrderBy("created_at DESC", "id DESC").
		Limit(uint64(pageSize)).Offset(uint64((page - 1) * pageSize)).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build access manual operation list: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query access manual operations: %w", err)
	}
	defer rows.Close()
	items := make([]AccessManualOperationItem, 0, pageSize)
	for rows.Next() {
		var item AccessManualOperationItem
		var details []byte
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Username, &item.Operation, &item.ResourceID, &details, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan access manual operation: %w", err)
		}
		item.Details = json.RawMessage(details)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate access manual operations: %w", err)
	}
	return items, total, nil
}

func (s *PgManagementStore) ensureStateVisible(ctx context.Context, filter ManagementFilter) error {
	builder := storage.Psql.Select("1").From("device_access_states state").Where(sq.Eq{
		"state.carrier":       strings.TrimSpace(filter.Carrier),
		"state.serial_number": strings.TrimSpace(filter.SerialNumber),
	})
	builder = applyAccessIdentityVisibility(
		builder, "state.device_id", "state.carrier", "state.serial_number", filter.VisibleGroups,
	)
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build access detail visibility check: %w", err)
	}
	var one int
	if err := s.db.QueryRow(ctx, query, args...).Scan(&one); err != nil {
		if err == pgx.ErrNoRows {
			return pgx.ErrNoRows
		}
		return fmt.Errorf("check access detail visibility: %w", err)
	}
	return nil
}

package deviceaccess

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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type AccessStateItem struct {
	Carrier           string          `json:"carrier"`
	SerialNumber      string          `json:"serial_number"`
	ProductName       string          `json:"product_name"`
	DeviceID          *uuid.UUID      `json:"device_id,omitempty"`
	CandidateID       *uuid.UUID      `json:"candidate_id,omitempty"`
	State             AccessState     `json:"state"`
	EffectiveDecision EffectiveAction `json:"effective_decision"`
	ReasonCode        ReasonCode      `json:"reason_code"`
	PolicyVersionID   *uuid.UUID      `json:"policy_version_id,omitempty"`
	EvidenceVersion   int64           `json:"evidence_version"`
	DecisionVersion   int64           `json:"decision_version"`
	NormalTasksFrozen bool            `json:"normal_tasks_frozen"`
	DecisionExpiresAt *time.Time      `json:"decision_expires_at,omitempty"`
	LastDecidedAt     time.Time       `json:"last_decided_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type PolicyVersionSummary struct {
	ID                       uuid.UUID           `json:"id"`
	PolicySetID              uuid.UUID           `json:"policy_set_id"`
	Name                     string              `json:"name"`
	Carrier                  string              `json:"carrier"`
	Version                  int64               `json:"version"`
	Status                   PolicyVersionStatus `json:"status"`
	DefaultAction            PolicyDefaultAction `json:"default_action"`
	FailureMode              FailureMode         `json:"failure_mode"`
	CollectionTimeoutSeconds int64               `json:"collection_timeout_seconds"`
	BypassProfileCount       int64               `json:"bypass_profile_count"`
	CreatedBy                *uuid.UUID          `json:"created_by,omitempty"`
	PublishedBy              *uuid.UUID          `json:"published_by,omitempty"`
	PublishedAt              *time.Time          `json:"published_at,omitempty"`
	CreatedAt                time.Time           `json:"created_at"`
}

type AccessListItem struct {
	ID            uuid.UUID       `json:"id"`
	Carrier       string          `json:"carrier"`
	EntryType     ListEntryType   `json:"entry_type"`
	IdentityType  IdentityType    `json:"identity_type"`
	IdentityValue string          `json:"identity_value"`
	ProductName   string          `json:"product_name"`
	ReasonCode    string          `json:"reason_code"`
	Reason        string          `json:"reason,omitempty"`
	ValidFrom     time.Time       `json:"valid_from"`
	ValidUntil    *time.Time      `json:"valid_until,omitempty"`
	Status        ListEntryStatus `json:"status"`
	CreatedBy     *uuid.UUID      `json:"created_by,omitempty"`
	ApprovedBy    *uuid.UUID      `json:"approved_by,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type CandidateItem struct {
	ID               uuid.UUID  `json:"id"`
	Carrier          string     `json:"carrier"`
	SerialNumber     string     `json:"serial_number"`
	OUI              string     `json:"oui"`
	ProductClass     string     `json:"product_class,omitempty"`
	SoftwareVersion  string     `json:"software_version,omitempty"`
	ObservedRemoteIP string     `json:"observed_remote_ip,omitempty"`
	DeviceID         *uuid.UUID `json:"device_id,omitempty"`
	FirstSeenAt      time.Time  `json:"first_seen_at"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	InformCount      int64      `json:"inform_count"`
	ReviewStatus     string     `json:"review_status"`
	ReviewedBy       *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ExpiresAt        time.Time  `json:"expires_at"`
}

type DecisionItem struct {
	ID               uuid.UUID           `json:"id"`
	TriggerType      string              `json:"trigger_type"`
	PreviousState    *AccessState        `json:"previous_state,omitempty"`
	NewState         AccessState         `json:"new_state"`
	Decision         EffectiveAction     `json:"decision"`
	ReasonCode       ReasonCode          `json:"reason_code"`
	PolicyVersionID  *uuid.UUID          `json:"policy_version_id,omitempty"`
	EvidenceVersion  int64               `json:"evidence_version"`
	DecisionVersion  int64               `json:"decision_version"`
	MatchedListEntry *uuid.UUID          `json:"matched_list_entry_id,omitempty"`
	MatchedRule      *uuid.UUID          `json:"matched_rule_id,omitempty"`
	OccurredAt       time.Time           `json:"occurred_at"`
	Checks           []DecisionCheckItem `json:"checks"`
	Archive          *DecisionArchive    `json:"archive,omitempty"`
}

type DecisionCheckItem struct {
	ID              uuid.UUID   `json:"id"`
	CheckType       string      `json:"check_type"`
	Result          CheckResult `json:"result"`
	ExpectedSummary string      `json:"expected_summary,omitempty"`
	ObservedSummary string      `json:"observed_summary,omitempty"`
	EvidenceSource  string      `json:"evidence_source,omitempty"`
	ObservedAt      *time.Time  `json:"observed_at,omitempty"`
	ReasonCode      string      `json:"reason_code,omitempty"`
}

type EvidenceItem struct {
	ID              uuid.UUID       `json:"id"`
	EvidenceVersion int64           `json:"evidence_version"`
	EvidenceType    ConditionType   `json:"evidence_type"`
	EvidenceStatus  EvidenceStatus  `json:"evidence_status"`
	NormalizedValue json.RawMessage `json:"normalized_value"`
	Source          string          `json:"source"`
	TaskID          *uuid.UUID      `json:"task_id,omitempty"`
	ObservedAt      time.Time       `json:"observed_at"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
}

type AccessNotificationItem struct {
	ID            uuid.UUID  `json:"id"`
	Channel       string     `json:"channel"`
	Recipients    []string   `json:"recipients"`
	Subject       string     `json:"subject"`
	Status        string     `json:"status"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	SourceType    string     `json:"source_type"`
	SourceID      *uuid.UUID `json:"source_id,omitempty"`
	EventID       *uuid.UUID `json:"event_id,omitempty"`
	CorrelationID string     `json:"correlation_id,omitempty"`
	RetryCount    int        `json:"retry_count"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type AccessDetail struct {
	State     AccessStateItem `json:"state"`
	Decisions []DecisionItem  `json:"decisions"`
	Evidence  []EvidenceItem  `json:"evidence"`
	Actions   []Action        `json:"actions"`
}

type AccessStateSummary struct {
	Accepted       int64 `json:"accepted"`
	Rejected       int64 `json:"rejected"`
	ReviewRequired int64 `json:"review_required"`
	Revoked        int64 `json:"revoked"`
	Total          int64 `json:"total"`
}

type ManagementFilter struct {
	Carrier         string
	SerialNumber    string
	ProductName     string
	State           AccessState
	Status          string
	EntryType       ListEntryType
	Decision        EffectiveAction
	ReasonCode      ReasonCode
	ActionStatus    ActionStatus
	Dimension       ConditionType
	PolicyVersionID *uuid.UUID
	MatchedRuleID   *uuid.UUID
	StartedAt       *time.Time
	EndedAt         *time.Time
	ArchiveStatus   string
	Page            int
	PageSize        int
	VisibleGroups   []uuid.UUID
}

type ManagementStore interface {
	ListStates(context.Context, ManagementFilter) ([]AccessStateItem, int64, error)
	SummarizeStates(context.Context, ManagementFilter) (AccessStateSummary, error)
	GetDetail(context.Context, ManagementFilter) (AccessDetail, error)
	ListDecisions(context.Context, ManagementFilter) ([]DecisionItem, int64, error)
	ListIdentitySnapshots(context.Context, ManagementFilter) ([]IdentitySnapshot, int64, error)
	ListEvidence(context.Context, ManagementFilter) ([]EvidenceItem, int64, error)
	ListNotifications(context.Context, ManagementFilter) ([]AccessNotificationItem, int64, error)
	ListManualOperations(context.Context, ManagementFilter) ([]AccessManualOperationItem, int64, error)
	ArchiveDecision(context.Context, string, uuid.UUID, uuid.UUID, []uuid.UUID, string) error
	RestoreDecision(context.Context, string, uuid.UUID, uuid.UUID, []uuid.UUID) error
	ListPolicyVersions(context.Context, ManagementFilter) ([]PolicyVersionSummary, int64, error)
	ListEntries(context.Context, ManagementFilter) ([]AccessListItem, int64, error)
	ListCandidates(context.Context, ManagementFilter) ([]CandidateItem, int64, error)
	ReviewCandidate(context.Context, string, uuid.UUID, uuid.UUID, []uuid.UUID, ListEntryType, string) error
}

// CandidateOwnershipRegistration is the minimum asset information required to
// turn an approved unknown candidate into a pre-registered OMC asset. The
// registration is written in the same transaction as the review/list/outbox
// changes so an approval cannot be partially applied.
type CandidateOwnershipRegistration struct {
	Carrier      string
	SerialNumber string
	CreatedBy    string
}

type CandidateOwnershipRegistrar interface {
	EnsureCandidateOwnership(context.Context, pgx.Tx, CandidateOwnershipRegistration) error
}

type PgManagementStore struct {
	db                 storage.DB
	actions            ActionStore
	ownershipRegistrar CandidateOwnershipRegistrar
}

const (
	accessStateProductNameSQL = "COALESCE(NULLIF(d.model_name, ''), '')"
	accessListProductNameSQL  = "COALESCE(NULLIF(d.model_name, ''), '')"
)

func NewPgManagementStore(pool *pgxpool.Pool, actions ActionStore) *PgManagementStore {
	return &PgManagementStore{db: storage.NewPoolDB(pool), actions: actions}
}

func (s *PgManagementStore) SetCandidateOwnershipRegistrar(registrar CandidateOwnershipRegistrar) {
	s.ownershipRegistrar = registrar
}

func (s *PgManagementStore) ListStates(ctx context.Context, filter ManagementFilter) ([]AccessStateItem, int64, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := stateFilters(filter)
	countBuilder := storage.Psql.Select("COUNT(*)").From("device_access_states s").Where(where)
	countBuilder = applyAccessIdentityVisibility(countBuilder, "s.device_id", "s.carrier", "s.serial_number", filter.VisibleGroups)
	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build access state count: %w", err)
	}
	var total int64
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count access states: %w", err)
	}
	listBuilder := storage.Psql.Select(
		"s.carrier", "s.serial_number", accessStateProductNameSQL, "s.device_id", "s.candidate_id", "s.state",
		"s.effective_decision", "s.reason_code", "s.policy_version_id", "s.evidence_version",
		"s.decision_version", "s.normal_tasks_frozen", "s.decision_expires_at", "s.last_decided_at", "s.updated_at",
	).From("device_access_states s").
		LeftJoin("devices d ON d.id = s.device_id").
		Where(where).
		OrderBy("s.updated_at DESC", "s.serial_number").Limit(uint64(pageSize)).
		Offset(uint64((page - 1) * pageSize))
	listBuilder = applyAccessIdentityVisibility(listBuilder, "s.device_id", "s.carrier", "s.serial_number", filter.VisibleGroups)
	query, args, err := listBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build access state list: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query access states: %w", err)
	}
	defer rows.Close()
	items := make([]AccessStateItem, 0, pageSize)
	for rows.Next() {
		var item AccessStateItem
		if err := rows.Scan(
			&item.Carrier, &item.SerialNumber, &item.ProductName, &item.DeviceID, &item.CandidateID, &item.State,
			&item.EffectiveDecision, &item.ReasonCode, &item.PolicyVersionID, &item.EvidenceVersion,
			&item.DecisionVersion, &item.NormalTasksFrozen, &item.DecisionExpiresAt, &item.LastDecidedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan access state: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate access states: %w", err)
	}
	return items, total, nil
}

func (s *PgManagementStore) SummarizeStates(ctx context.Context, filter ManagementFilter) (AccessStateSummary, error) {
	where := stateFilters(filter)
	builder := storage.Psql.Select(
		"COUNT(*) FILTER (WHERE s.state = 'accepted')",
		"COUNT(*) FILTER (WHERE s.effective_decision = 'reject')",
		"COUNT(*) FILTER (WHERE s.effective_decision = 'review')",
		"COUNT(*) FILTER (WHERE s.effective_decision = 'revoke')",
		"COUNT(*)",
	).From("device_access_states s").Where(where)
	builder = applyAccessIdentityVisibility(builder, "s.device_id", "s.carrier", "s.serial_number", filter.VisibleGroups)
	query, args, err := builder.ToSql()
	if err != nil {
		return AccessStateSummary{}, fmt.Errorf("build access state summary: %w", err)
	}
	var summary AccessStateSummary
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&summary.Accepted, &summary.Rejected, &summary.ReviewRequired, &summary.Revoked, &summary.Total,
	); err != nil {
		return AccessStateSummary{}, fmt.Errorf("summarize access states: %w", err)
	}
	return summary, nil
}

func stateFilters(filter ManagementFilter) sq.And {
	where := tenantFilters(filter, "s")
	if filter.Decision != "" {
		where = append(where, sq.Eq{"s.effective_decision": filter.Decision})
	}
	if filter.ReasonCode != "" {
		where = append(where, sq.Eq{"s.reason_code": filter.ReasonCode})
	}
	if filter.PolicyVersionID != nil {
		where = append(where, sq.Eq{"s.policy_version_id": *filter.PolicyVersionID})
	}
	if filter.StartedAt != nil {
		where = append(where, sq.GtOrEq{"s.last_decided_at": *filter.StartedAt})
	}
	if filter.EndedAt != nil {
		where = append(where, sq.LtOrEq{"s.last_decided_at": *filter.EndedAt})
	}
	if filter.MatchedRuleID != nil {
		where = append(where, sq.Expr(`EXISTS (
			SELECT 1 FROM device_access_decisions filtered_decision
			WHERE filtered_decision.carrier = s.carrier
			AND filtered_decision.serial_number = s.serial_number
			AND filtered_decision.decision_version = s.decision_version
			AND filtered_decision.matched_rule_id = ?
		)`, *filter.MatchedRuleID))
	}
	if filter.Dimension != "" {
		where = append(where, sq.Expr(`EXISTS (
			SELECT 1 FROM device_access_decisions dimension_decision
			JOIN device_access_decision_checks dimension_check ON dimension_check.decision_id = dimension_decision.id
			WHERE dimension_decision.carrier = s.carrier
			AND dimension_decision.serial_number = s.serial_number
			AND dimension_decision.decision_version = s.decision_version
			AND dimension_check.check_type = ?
		)`, filter.Dimension))
	}
	if filter.ActionStatus != "" {
		where = append(where, sq.Expr(`EXISTS (
			SELECT 1 FROM device_access_actions filtered_action
			JOIN device_access_decisions action_decision ON action_decision.id = filtered_action.decision_id
			WHERE action_decision.carrier = s.carrier
			AND action_decision.serial_number = s.serial_number
			AND filtered_action.status = ?
		)`, filter.ActionStatus))
	}
	return where
}

func (s *PgManagementStore) GetDetail(ctx context.Context, filter ManagementFilter) (AccessDetail, error) {
	carrier := strings.TrimSpace(filter.Carrier)
	serialNumber := strings.TrimSpace(filter.SerialNumber)
	builder := storage.Psql.Select(
		"s.carrier", "s.serial_number", accessStateProductNameSQL,
		"s.device_id", "s.candidate_id", "s.state", "s.effective_decision",
		"s.reason_code", "s.policy_version_id", "s.evidence_version", "s.decision_version",
		"s.normal_tasks_frozen", "s.decision_expires_at", "s.last_decided_at", "s.updated_at",
	).From("device_access_states s").
		LeftJoin("devices d ON d.id = s.device_id").
		Where(sq.Eq{"s.carrier": carrier, "s.serial_number": serialNumber})
	builder = applyAccessIdentityVisibility(builder, "s.device_id", "s.carrier", "s.serial_number", filter.VisibleGroups)
	query, args, err := builder.ToSql()
	if err != nil {
		return AccessDetail{}, fmt.Errorf("build access state detail: %w", err)
	}
	detail := AccessDetail{Decisions: []DecisionItem{}, Evidence: []EvidenceItem{}, Actions: []Action{}}
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&detail.State.Carrier, &detail.State.SerialNumber, &detail.State.ProductName, &detail.State.DeviceID, &detail.State.CandidateID,
		&detail.State.State, &detail.State.EffectiveDecision, &detail.State.ReasonCode, &detail.State.PolicyVersionID,
		&detail.State.EvidenceVersion, &detail.State.DecisionVersion, &detail.State.NormalTasksFrozen,
		&detail.State.DecisionExpiresAt,
		&detail.State.LastDecidedAt, &detail.State.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return AccessDetail{}, fmt.Errorf("get access state detail: %w", err)
		}
		return AccessDetail{}, fmt.Errorf("query access state detail: %w", err)
	}
	historyFilter := filter
	historyFilter.Page = 1
	historyFilter.PageSize = 50
	historyFilter.ArchiveStatus = "all"
	detail.Decisions, _, err = s.ListDecisions(ctx, historyFilter)
	if err != nil {
		return AccessDetail{}, fmt.Errorf("load access state decisions: %w", err)
	}
	detail.Evidence, err = s.loadEvidenceVersion(ctx, carrier, serialNumber, detail.State.EvidenceVersion)
	if err != nil {
		return AccessDetail{}, err
	}
	if s.actions != nil {
		detail.Actions, _, err = s.actions.List(ctx, ActionListFilter{
			Carrier: carrier, SerialNumber: serialNumber, Page: 1, PageSize: 100,
			VisibleGroups: filter.VisibleGroups,
		})
		if err != nil {
			return AccessDetail{}, fmt.Errorf("load access state actions: %w", err)
		}
	}
	return detail, nil
}

func (s *PgManagementStore) ListPolicyVersions(ctx context.Context, filter ManagementFilter) ([]PolicyVersionSummary, int64, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := sq.And{sq.Eq{"ps.carrier": strings.TrimSpace(filter.Carrier)}}
	from := "device_access_policy_versions pv JOIN device_access_policy_sets ps ON ps.id = pv.policy_set_id"
	total, err := countRows(ctx, s.db, from, where)
	if err != nil {
		return nil, 0, fmt.Errorf("count policy versions: %w", err)
	}
	query, args, err := storage.Psql.Select(
		"pv.id", "pv.policy_set_id", "ps.name", "ps.carrier", "pv.version", "pv.status",
		"pv.default_action", "pv.failure_mode", "pv.collection_timeout_seconds", "jsonb_array_length(pv.bypass_profiles)",
		"pv.created_by", "pv.published_by", "pv.published_at", "pv.created_at",
	).From(from).Where(where).OrderBy("pv.created_at DESC").Limit(uint64(pageSize)).
		Offset(uint64((page - 1) * pageSize)).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build policy version list: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query policy versions: %w", err)
	}
	defer rows.Close()
	items := make([]PolicyVersionSummary, 0, pageSize)
	for rows.Next() {
		var item PolicyVersionSummary
		if err := rows.Scan(&item.ID, &item.PolicySetID, &item.Name, &item.Carrier, &item.Version, &item.Status, &item.DefaultAction,
			&item.FailureMode, &item.CollectionTimeoutSeconds, &item.BypassProfileCount,
			&item.CreatedBy, &item.PublishedBy, &item.PublishedAt, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan policy version: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate policy versions: %w", err)
	}
	return items, total, nil
}

func (s *PgManagementStore) ListEntries(ctx context.Context, filter ManagementFilter) ([]AccessListItem, int64, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := sq.And{sq.Eq{"le.carrier": strings.TrimSpace(filter.Carrier)}}
	if filter.EntryType != "" {
		where = append(where, sq.Eq{"le.entry_type": filter.EntryType})
	}
	if filter.SerialNumber != "" {
		where = append(where, sq.ILike{"le.identity_value": "%" + strings.TrimSpace(filter.SerialNumber) + "%"})
	}
	if filter.ProductName != "" {
		where = append(where, sq.ILike{accessListProductNameSQL: "%" + strings.TrimSpace(filter.ProductName) + "%"})
	}
	if filter.Status != "" {
		where = append(where, sq.Eq{"le.status": filter.Status})
	}
	countBuilder := storage.Psql.Select("COUNT(*)").From("device_access_list_entries le").
		LeftJoin("devices d ON d.carrier = le.carrier AND d.serial_number = le.identity_value").Where(where)
	countBuilder = applyAccessIdentityVisibility(countBuilder, "d.id", "le.carrier", "le.identity_value", filter.VisibleGroups)
	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build access list count: %w", err)
	}
	var total int64
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count access list: %w", err)
	}
	listBuilder := storage.Psql.Select(
		"le.id", "le.carrier", "le.entry_type", "le.identity_type", "le.identity_value",
		accessListProductNameSQL, "le.reason_code",
		"COALESCE(le.reason, '')", "le.valid_from", "le.valid_until", "le.status", "le.created_by", "le.approved_by",
		"le.created_at", "le.updated_at",
	).From("device_access_list_entries le").
		LeftJoin("devices d ON d.carrier = le.carrier AND d.serial_number = le.identity_value").
		Where(where).OrderBy("le.updated_at DESC").Limit(uint64(pageSize)).Offset(uint64((page - 1) * pageSize))
	listBuilder = applyAccessIdentityVisibility(listBuilder, "d.id", "le.carrier", "le.identity_value", filter.VisibleGroups)
	query, args, err := listBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build access list query: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query access list: %w", err)
	}
	defer rows.Close()
	items := make([]AccessListItem, 0, pageSize)
	for rows.Next() {
		var item AccessListItem
		if err := rows.Scan(&item.ID, &item.Carrier, &item.EntryType, &item.IdentityType, &item.IdentityValue, &item.ProductName,
			&item.ReasonCode, &item.Reason, &item.ValidFrom, &item.ValidUntil, &item.Status,
			&item.CreatedBy, &item.ApprovedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan access list entry: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate access list: %w", err)
	}
	return items, total, nil
}

func (s *PgManagementStore) ListCandidates(ctx context.Context, filter ManagementFilter) ([]CandidateItem, int64, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := sq.And{sq.Eq{"c.carrier": strings.TrimSpace(filter.Carrier)}}
	if filter.Status != "" {
		where = append(where, sq.Eq{"c.review_status": filter.Status})
		if filter.Status == "pending" {
			where = append(where, sq.Expr("c.device_id IS NULL"))
		}
	}
	if filter.SerialNumber != "" {
		where = append(where, sq.ILike{"c.serial_number": "%" + strings.TrimSpace(filter.SerialNumber) + "%"})
	}
	countBuilder := storage.Psql.Select("COUNT(*)").From("device_access_candidates c").Where(where)
	countBuilder = applyAccessIdentityVisibility(countBuilder, "c.device_id", "c.carrier", "c.serial_number", filter.VisibleGroups)
	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build candidate count: %w", err)
	}
	var total int64
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count candidates: %w", err)
	}
	listBuilder := storage.Psql.Select(
		"c.id", "c.carrier", "c.serial_number", "c.oui", "COALESCE(c.product_class, '')", "COALESCE(c.software_version, '')",
		"COALESCE(host(c.observed_remote_ip), '')", "c.device_id", "c.first_seen_at", "c.last_seen_at", "c.inform_count",
		"c.review_status", "c.reviewed_by", "c.reviewed_at", "c.expires_at",
	).From("device_access_candidates c").Where(where).OrderBy("c.last_seen_at DESC").
		Limit(uint64(pageSize)).Offset(uint64((page - 1) * pageSize))
	listBuilder = applyAccessIdentityVisibility(listBuilder, "c.device_id", "c.carrier", "c.serial_number", filter.VisibleGroups)
	query, args, err := listBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build candidate query: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query candidates: %w", err)
	}
	defer rows.Close()
	items := make([]CandidateItem, 0, pageSize)
	for rows.Next() {
		var item CandidateItem
		if err := rows.Scan(&item.ID, &item.Carrier, &item.SerialNumber, &item.OUI, &item.ProductClass,
			&item.SoftwareVersion, &item.ObservedRemoteIP, &item.DeviceID, &item.FirstSeenAt, &item.LastSeenAt,
			&item.InformCount, &item.ReviewStatus, &item.ReviewedBy, &item.ReviewedAt, &item.ExpiresAt); err != nil {
			return nil, 0, fmt.Errorf("scan candidate: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate candidates: %w", err)
	}
	return items, total, nil
}

func (s *PgManagementStore) ReviewCandidate(
	ctx context.Context,
	carrier string,
	candidateID, reviewerID uuid.UUID,
	visibleGroups []uuid.UUID,
	outcome ListEntryType,
	reason string,
) error {
	if outcome != ListEntryTypeAllow && outcome != ListEntryTypeDeny {
		return fmt.Errorf("candidate review outcome must be allow or deny")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin candidate review: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	lockBuilder := storage.Psql.Select("c.serial_number", "c.device_id").From("device_access_candidates c").
		Where(sq.Eq{"c.id": candidateID, "c.carrier": carrier})
	lockBuilder = applyAccessIdentityVisibility(lockBuilder, "c.device_id", "c.carrier", "c.serial_number", visibleGroups)
	query, args, err := lockBuilder.Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return fmt.Errorf("build candidate review lock: %w", err)
	}
	var serialNumber string
	var deviceID *uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&serialNumber, &deviceID); err != nil {
		return fmt.Errorf("lock candidate review: %w", err)
	}
	if outcome == ListEntryTypeAllow && deviceID == nil {
		if s.ownershipRegistrar == nil {
			return errors.New("candidate ownership registrar is not configured")
		}
		if err := s.ownershipRegistrar.EnsureCandidateOwnership(ctx, tx, CandidateOwnershipRegistration{
			Carrier:      carrier,
			SerialNumber: serialNumber,
			CreatedBy:    reviewerID.String(),
		}); err != nil {
			return fmt.Errorf("register approved candidate ownership: %w", err)
		}
	}
	status := "approved"
	if outcome == ListEntryTypeDeny {
		status = "rejected"
	}
	now := time.Now().UTC()
	query, args, err = storage.Psql.Update("device_access_candidates").Set("review_status", status).
		Set("reviewed_by", reviewerID).Set("reviewed_at", now).Set("updated_at", now).
		Where(sq.Eq{"id": candidateID, "review_status": "pending"}).ToSql()
	if err != nil {
		return fmt.Errorf("build candidate review update: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update candidate review: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("candidate review: %w", ErrActionStateChanged)
	}
	query, args, err = storage.Psql.Insert("device_access_list_entries").
		Columns("carrier", "entry_type", "identity_type", "identity_value", "reason_code", "reason", "status", "created_by", "approved_by").
		Values(carrier, outcome, IdentityTypeSerialNumber, serialNumber, "candidate_review", nullableString(reason), ListEntryStatusActive, reviewerID, reviewerID).
		Suffix(`ON CONFLICT (carrier,entry_type,identity_type,identity_value) DO UPDATE SET
			reason_code = EXCLUDED.reason_code, reason = EXCLUDED.reason, status = EXCLUDED.status,
			approved_by = EXCLUDED.approved_by, updated_at = now()`).ToSql()
	if err != nil {
		return fmt.Errorf("build candidate review list entry: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert candidate review list entry: %w", err)
	}
	payload, err := json.Marshal(ReevaluationRequest{Carrier: carrier, SerialNumber: serialNumber, TriggerType: TriggerCandidateReviewed})
	if err != nil {
		return fmt.Errorf("encode candidate review reevaluation: %w", err)
	}
	outboxID := uuid.New()
	query, args, err = storage.Psql.Insert("device_access_outbox").Columns(
		"aggregate_type", "aggregate_id", "event_type", "event_key", "payload", "next_attempt_at", "created_at", "updated_at",
	).Values("device_access_reevaluation", outboxID, event.SubjectDeviceAccessReevaluationRequested,
		fmt.Sprintf("candidate-review:%s:%s", candidateID, outboxID), payload, now, now, now).ToSql()
	if err != nil {
		return fmt.Errorf("build candidate review outbox: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert candidate review outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit candidate review: %w", err)
	}
	return nil
}

func (s *PgManagementStore) loadDecisionChecks(ctx context.Context, decisionID uuid.UUID) ([]DecisionCheckItem, error) {
	query, args, err := storage.Psql.Select(
		"id", "check_type", "result", "COALESCE(expected_summary, '')", "COALESCE(observed_summary, '')",
		"COALESCE(evidence_source, '')", "observed_at", "COALESCE(reason_code, '')",
	).From("device_access_decision_checks").Where(sq.Eq{"decision_id": decisionID}).OrderBy("check_type").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build decision checks: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query decision checks: %w", err)
	}
	defer rows.Close()
	items := []DecisionCheckItem{}
	for rows.Next() {
		var item DecisionCheckItem
		if err := rows.Scan(&item.ID, &item.CheckType, &item.Result, &item.ExpectedSummary, &item.ObservedSummary,
			&item.EvidenceSource, &item.ObservedAt, &item.ReasonCode); err != nil {
			return nil, fmt.Errorf("scan decision check: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decision checks: %w", err)
	}
	return items, nil
}

func (s *PgManagementStore) loadEvidenceVersion(
	ctx context.Context,
	carrier, serialNumber string,
	version int64,
) ([]EvidenceItem, error) {
	query, args, err := storage.Psql.Select(
		"id", "evidence_version", "evidence_type", "evidence_status", "normalized_value", "source",
		"task_id", "observed_at", "expires_at",
	).From("device_access_evidence").Where(sq.Eq{
		"carrier": carrier, "serial_number": serialNumber, "evidence_version": version,
	}).OrderBy("evidence_type").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build access evidence query: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query access evidence: %w", err)
	}
	defer rows.Close()
	items := make([]EvidenceItem, 0)
	for rows.Next() {
		var item EvidenceItem
		if err := rows.Scan(
			&item.ID, &item.EvidenceVersion, &item.EvidenceType, &item.EvidenceStatus,
			&item.NormalizedValue, &item.Source, &item.TaskID, &item.ObservedAt, &item.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan access evidence: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate access evidence: %w", err)
	}
	return items, nil
}

// applyAccessIdentityVisibility applies the canonical device-group scope to an
// access row. A candidate without a formal device is visible to a restricted
// operator only when its pre-registration belongs to one of the visible groups.
func applyAccessIdentityVisibility(
	builder sq.SelectBuilder,
	deviceIDColumn, carrierColumn, serialNumberColumn string,
	visibleGroups []uuid.UUID,
) sq.SelectBuilder {
	if visibleGroups == nil {
		return builder
	}
	if len(visibleGroups) == 0 {
		return builder.Where(sq.Expr("FALSE"))
	}
	realGroups, includeUngrouped := authz.SplitVisibleGroups(visibleGroups)
	clauses := make(sq.Or, 0, 3)
	if len(realGroups) > 0 {
		deviceIDs := sq.Select("device_id").From("device_group_members").Where(sq.Eq{"group_id": realGroups})
		clauses = append(clauses, sq.Expr("? IN (?)", sq.Expr(deviceIDColumn), deviceIDs))
		registration := sq.Select("1").From("device_registrations scope_registration").
			Where(sq.Expr("scope_registration.carrier = ?", sq.Expr(carrierColumn))).
			Where(sq.Expr("scope_registration.serial_number = ?", sq.Expr(serialNumberColumn))).
			Where(sq.Eq{"scope_registration.group_id": realGroups})
		clauses = append(clauses, sq.Expr("? IS NULL AND EXISTS (?)", sq.Expr(deviceIDColumn), registration))
	}
	if includeUngrouped {
		clauses = append(clauses, sq.Expr(
			"? IS NOT NULL AND NOT EXISTS (SELECT 1 FROM device_group_members scope_membership WHERE scope_membership.device_id = ?)",
			sq.Expr(deviceIDColumn), sq.Expr(deviceIDColumn),
		))
	}
	if len(clauses) == 0 {
		return builder.Where(sq.Expr("FALSE"))
	}
	return builder.Where(clauses)
}

func tenantFilters(filter ManagementFilter, alias string) sq.And {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	where := sq.And{sq.Eq{prefix + "carrier": strings.TrimSpace(filter.Carrier)}}
	if filter.SerialNumber != "" {
		where = append(where, sq.ILike{prefix + "serial_number": "%" + strings.TrimSpace(filter.SerialNumber) + "%"})
	}
	if filter.State != "" {
		where = append(where, sq.Eq{prefix + "state": filter.State})
	}
	return where
}

func countRows(ctx context.Context, db storage.DB, from string, where sq.Sqlizer) (int64, error) {
	query, args, err := storage.Psql.Select("COUNT(*)").From(from).Where(where).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}
	var total int64
	if err := db.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("execute count query: %w", err)
	}
	return total, nil
}

var _ ManagementStore = (*PgManagementStore)(nil)

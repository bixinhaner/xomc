package agentbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrFindingConflict = errors.New("finding delivery payload conflicts with the existing projection")

type FindingRepository struct {
	pool *pgxpool.Pool
}

func NewFindingRepository(pool *pgxpool.Pool) *FindingRepository {
	return &FindingRepository{pool: pool}
}

func (r *FindingRepository) Accept(ctx context.Context, delivery FindingDelivery) (string, bool, bool, error) {
	raw, err := json.Marshal(delivery.Finding)
	if err != nil {
		return "", false, false, fmt.Errorf("marshal finding payload: %w", err)
	}
	sum := sha256.Sum256(raw)
	payloadHash := hex.EncodeToString(sum[:])
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", false, false, fmt.Errorf("begin finding projection: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var existingID, existingHash, existingStatus string
	lookupSQL, lookupArgs, err := psql.Select("id", "payload_hash", "status").From("agent_findings").
		Where(sq.Eq{"delivery_id": delivery.DeliveryID}).Limit(1).ToSql()
	if err != nil {
		return "", false, false, fmt.Errorf("build finding replay lookup: %w", err)
	}
	err = tx.QueryRow(ctx, lookupSQL, lookupArgs...).Scan(&existingID, &existingHash, &existingStatus)
	if err == nil {
		if existingHash != payloadHash {
			return "", false, false, ErrFindingConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return "", false, false, fmt.Errorf("commit finding replay: %w", err)
		}
		return existingID, true, existingStatus == "active", nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, false, fmt.Errorf("lookup finding replay: %w", err)
	}

	groupIDs, err := r.resolveScopes(ctx, tx, delivery.Finding.ResourceRefs)
	if err != nil {
		return "", false, false, err
	}
	status := "active"
	if len(groupIDs) == 0 {
		status = "quarantined"
	}
	facts, _ := json.Marshal(delivery.Finding.Facts)
	hypotheses, _ := json.Marshal(delivery.Finding.Hypotheses)
	details, _ := json.Marshal(delivery.Finding.Details)
	actions, _ := json.Marshal(delivery.Finding.SuggestedActions)
	presentation, _ := json.Marshal(delivery.Finding.Presentation)
	insertSQL, insertArgs, err := psql.Insert("agent_findings").Columns(
		"delivery_id", "remote_finding_id", "run_id", "scenario_key", "package_digest", "handbook_digest",
		"title", "summary", "severity", "confidence", "facts", "hypotheses", "details", "suggested_actions",
		"presentation", "payload_hash", "status", "trace_id", "expires_at",
	).Values(
		delivery.DeliveryID, delivery.FindingID, delivery.RunID, delivery.ScenarioKey, delivery.PackageDigest, delivery.HandbookDigest,
		delivery.Finding.Title, delivery.Finding.Summary, delivery.Finding.Severity, delivery.Finding.Confidence,
		facts, hypotheses, details, actions, presentation, payloadHash, status, delivery.TraceID, delivery.Finding.ExpiresAt,
	).Suffix("RETURNING id").ToSql()
	if err != nil {
		return "", false, false, fmt.Errorf("build finding projection insert: %w", err)
	}
	var localID string
	if err := tx.QueryRow(ctx, insertSQL, insertArgs...).Scan(&localID); err != nil {
		return "", false, false, fmt.Errorf("insert finding projection: %w", err)
	}
	for _, resource := range delivery.Finding.ResourceRefs {
		query, args, buildErr := psql.Insert("agent_finding_resources").
			Columns("finding_id", "resource_type", "resource_id", "resource_role", "label").
			Values(localID, resource.Type, resource.ID, resource.Role, resource.Label).
			Suffix("ON CONFLICT DO NOTHING").ToSql()
		if buildErr != nil {
			return "", false, false, fmt.Errorf("build finding resource insert: %w", buildErr)
		}
		if _, execErr := tx.Exec(ctx, query, args...); execErr != nil {
			return "", false, false, fmt.Errorf("insert finding resource: %w", execErr)
		}
	}
	for _, groupID := range groupIDs {
		query, args, buildErr := psql.Insert("agent_finding_scopes").Columns("finding_id", "group_id").
			Values(localID, groupID).Suffix("ON CONFLICT DO NOTHING").ToSql()
		if buildErr != nil {
			return "", false, false, fmt.Errorf("build finding scope insert: %w", buildErr)
		}
		if _, execErr := tx.Exec(ctx, query, args...); execErr != nil {
			return "", false, false, fmt.Errorf("insert finding scope: %w", execErr)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", false, false, fmt.Errorf("commit finding projection: %w", err)
	}
	return localID, false, len(groupIDs) > 0, nil
}

func (r *FindingRepository) resolveScopes(ctx context.Context, tx pgx.Tx, refs []ResourceRef) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]struct{}{}
	for _, ref := range refs {
		var query string
		var args []any
		var err error
		switch ref.Type {
		case "device":
			query, args, err = psql.Select("dgm.group_id").From("device_group_members dgm").
				Where(sq.Eq{"dgm.device_id": ref.ID}).ToSql()
		case "task":
			query, args, err = psql.Select("dgm.group_id").From("device_tasks t").
				Join("devices d ON d.serial_number = t.device_sn AND d.deleted_at IS NULL").
				Join("device_group_members dgm ON dgm.device_id = d.id").Where(sq.Eq{"t.id": ref.ID}).ToSql()
		default:
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("build finding scope resolution: %w", err)
		}
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("resolve finding scopes: %w", err)
		}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan finding scope: %w", err)
			}
			seen[id] = struct{}{}
		}
		rows.Close()
	}
	result := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result, nil
}

type FindingListOptions struct {
	UserID           uuid.UUID
	SuperAdmin       bool
	VisibleGroups    []uuid.UUID
	FindingID        string
	ResourceType     string
	ResourceID       string
	Limit            int
	Offset           int
	IncludeDismissed bool
}

func (r *FindingRepository) List(ctx context.Context, options FindingListOptions) ([]StoredFinding, int64, error) {
	base := psql.Select("f.id").From("agent_findings f").Where(sq.Eq{"f.status": "active"}).
		Where(sq.Or{sq.Expr("f.expires_at IS NULL"), sq.Gt{"f.expires_at": time.Now().UTC()}})
	base = applyFindingVisibility(base, options)
	if options.FindingID != "" {
		base = base.Where(sq.Eq{"f.id": options.FindingID})
	}
	if options.ResourceType != "" && options.ResourceID != "" {
		base = base.Join("agent_finding_resources filter_resource ON filter_resource.finding_id = f.id").
			Where(sq.Eq{"filter_resource.resource_type": options.ResourceType, "filter_resource.resource_id": options.ResourceID})
	}
	if !options.IncludeDismissed {
		base = base.Where(sq.Expr("NOT EXISTS (SELECT 1 FROM agent_finding_user_states ds WHERE ds.finding_id = f.id AND ds.user_id = ? AND ds.dismissed_at IS NOT NULL)", options.UserID))
	}
	countSQL, countArgs, err := base.RemoveColumns().Column("COUNT(DISTINCT f.id)").ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build finding count: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count findings: %w", err)
	}
	limit := options.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}
	queryBuilder := base.RemoveColumns().Columns(
		"DISTINCT f.id", "f.delivery_id", "f.remote_finding_id", "f.run_id", "f.scenario_key",
		"f.title", "f.summary", "f.severity", "f.confidence", "f.facts", "f.hypotheses", "f.details",
		"f.suggested_actions", "f.presentation", "f.created_at", "f.expires_at",
	).Column(
		"EXISTS (SELECT 1 FROM agent_finding_user_states rs WHERE rs.finding_id=f.id AND rs.user_id=? AND rs.read_at IS NOT NULL) AS is_read",
		options.UserID,
	).Column(
		"EXISTS (SELECT 1 FROM agent_finding_user_states xs WHERE xs.finding_id=f.id AND xs.user_id=? AND xs.dismissed_at IS NOT NULL) AS is_dismissed",
		options.UserID,
	).OrderBy("f.created_at DESC").Limit(uint64(limit)).Offset(uint64(max(options.Offset, 0)))
	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build finding list: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query findings: %w", err)
	}
	findings, err := pgx.CollectRows(rows, scanStoredFinding)
	if err != nil {
		return nil, 0, fmt.Errorf("scan findings: %w", err)
	}
	for i := range findings {
		findings[i].Resources, err = r.listResources(ctx, findings[i].ID)
		if err != nil {
			return nil, 0, err
		}
	}
	return findings, total, nil
}

func applyFindingVisibility(builder sq.SelectBuilder, options FindingListOptions) sq.SelectBuilder {
	if options.SuperAdmin {
		return builder
	}
	if len(options.VisibleGroups) == 0 {
		return builder.Where(sq.Expr("FALSE"))
	}
	return builder.Where(sq.Expr("EXISTS (SELECT 1 FROM agent_finding_scopes fs WHERE fs.finding_id=f.id AND fs.group_id = ANY(?))", options.VisibleGroups))
}

func scanStoredFinding(row pgx.CollectableRow) (StoredFinding, error) {
	var finding StoredFinding
	var facts, hypotheses, details, actions, presentation []byte
	err := row.Scan(&finding.ID, &finding.DeliveryID, &finding.RemoteFindingID, &finding.RunID, &finding.ScenarioKey,
		&finding.Title, &finding.Summary, &finding.Severity, &finding.Confidence, &facts, &hypotheses, &details,
		&actions, &presentation, &finding.CreatedAt, &finding.ExpiresAt, &finding.Read, &finding.Dismissed)
	if err != nil {
		return finding, err
	}
	_ = json.Unmarshal(facts, &finding.Facts)
	_ = json.Unmarshal(hypotheses, &finding.Hypotheses)
	_ = json.Unmarshal(details, &finding.Details)
	_ = json.Unmarshal(actions, &finding.SuggestedActions)
	_ = json.Unmarshal(presentation, &finding.Presentation)
	return finding, nil
}

func (r *FindingRepository) listResources(ctx context.Context, findingID string) ([]ResourceRef, error) {
	query, args, err := psql.Select("resource_type", "resource_id", "resource_role", "label").
		From("agent_finding_resources").Where(sq.Eq{"finding_id": findingID}).OrderBy("resource_role", "resource_type", "resource_id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build finding resource list: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query finding resources: %w", err)
	}
	resources, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ResourceRef, error) {
		var value ResourceRef
		err := row.Scan(&value.Type, &value.ID, &value.Role, &value.Label)
		return value, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan finding resources: %w", err)
	}
	return resources, nil
}

func (r *FindingRepository) SetUserState(ctx context.Context, findingID string, userID uuid.UUID, action string) error {
	updates := map[string]any{"updated_at": time.Now().UTC()}
	switch action {
	case "read":
		updates["read_at"] = time.Now().UTC()
	case "dismiss":
		updates["dismissed_at"] = time.Now().UTC()
	case "restore":
		updates["dismissed_at"] = nil
	default:
		return fmt.Errorf("unknown finding user-state action %q", action)
	}
	query, args, err := psql.Insert("agent_finding_user_states").Columns("finding_id", "user_id", "read_at", "dismissed_at").
		Values(findingID, userID, updates["read_at"], updates["dismissed_at"]).
		Suffix("ON CONFLICT (finding_id,user_id) DO UPDATE SET read_at=COALESCE(EXCLUDED.read_at,agent_finding_user_states.read_at), dismissed_at=EXCLUDED.dismissed_at, updated_at=now()").ToSql()
	if err != nil {
		return fmt.Errorf("build finding user state upsert: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update finding user state: %w", err)
	}
	return nil
}

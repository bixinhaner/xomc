package agentassistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var sql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
var ErrNotFound = errors.New("ASSISTANT_NOT_FOUND")
var ErrConflict = errors.New("ASSISTANT_REVISION_CONFLICT")
var ErrTrialRequired = errors.New("ASSISTANT_TRIAL_REQUIRED")

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type dbQuery interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

const assistantColumns = "a.id,a.owner_id,a.role_id,a.revision,a.state,a.published_revision,a.next_run_at,a.last_run_at,a.last_error,a.document,a.created_at,a.updated_at,v.definition,v.created_at"

func assistantSelect() sq.SelectBuilder {
	return sql.Select(assistantColumns).From("agent_assistants a").LeftJoin("agent_assistant_versions v ON v.assistant_id=a.id AND v.revision=a.published_revision")
}
func scanAssistant(row pgx.Row) (Assistant, error) {
	var a Assistant
	var document, published []byte
	err := row.Scan(&a.ID, &a.OwnerID, &a.RoleID, &a.Revision, &a.State, &a.PublishedRevision, &a.NextRunAt, &a.LastRunAt, &a.LastError, &document, &a.CreatedAt, &a.UpdatedAt, &published, &a.PublishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	if err != nil {
		return a, fmt.Errorf("scan assistant: %w", err)
	}
	var doc Assistant
	if err = json.Unmarshal(document, &doc); err != nil {
		return a, fmt.Errorf("decode assistant document: %w", err)
	}
	a.Definition = doc.Definition
	a.Messages = doc.Messages
	a.Readiness = doc.Readiness
	a.Questions = doc.Questions
	a.MissingCapabilities = doc.MissingCapabilities
	a.Locale = doc.Locale
	a.Timezone = doc.Timezone
	if len(published) > 0 {
		if err = json.Unmarshal(published, &a.PublishedDefinition); err != nil {
			return a, fmt.Errorf("decode published assistant: %w", err)
		}
	}
	return a, nil
}
func getAssistant(ctx context.Context, db dbQuery, owner, id string, lock bool) (Assistant, error) {
	q := assistantSelect().Where(sq.Eq{"a.id": id})
	if owner != "" {
		q = q.Where(sq.Eq{"a.owner_id": owner})
	}
	if lock {
		q = q.Suffix("FOR UPDATE OF a")
	}
	text, args, err := q.ToSql()
	if err != nil {
		return Assistant{}, fmt.Errorf("build assistant lookup: %w", err)
	}
	return scanAssistant(db.QueryRow(ctx, text, args...))
}
func (r *Repository) Get(ctx context.Context, owner, id string) (Assistant, error) {
	return getAssistant(ctx, r.pool, owner, id, false)
}
func (r *Repository) List(ctx context.Context, owner string) ([]Assistant, error) {
	q, args, err := assistantSelect().Where(sq.Eq{"a.owner_id": owner}).OrderBy("a.updated_at DESC").Limit(100).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build assistant list: %w", err)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list assistants: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Assistant, error) { return scanAssistant(row) })
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []Assistant{}
	}
	return out, nil
}
func (r *Repository) Create(ctx context.Context, owner, role, id, locale, timezone string) (Assistant, error) {
	a := Assistant{Messages: []Message{}, Readiness: "needs_input", Questions: []string{}, MissingCapabilities: []string{}, Locale: locale, Timezone: timezone}
	raw, err := json.Marshal(a)
	if err != nil {
		return a, fmt.Errorf("encode assistant draft: %w", err)
	}
	q, args, err := sql.Insert("agent_assistants").Columns("id", "owner_id", "role_id", "document").Values(id, owner, role, raw).Suffix("ON CONFLICT (id) DO NOTHING").ToSql()
	if err != nil {
		return a, fmt.Errorf("build assistant insert: %w", err)
	}
	if _, err = r.pool.Exec(ctx, q, args...); err != nil {
		return a, fmt.Errorf("create assistant: %w", err)
	}
	return r.Get(ctx, owner, id)
}
func (r *Repository) SaveDraft(ctx context.Context, a Assistant, expected int) (Assistant, error) {
	raw, err := json.Marshal(a)
	if err != nil {
		return a, fmt.Errorf("encode assistant draft: %w", err)
	}
	q, args, err := sql.Update("agent_assistants").Set("document", raw).Set("revision", expected+1).Set("role_id", a.RoleID).Set("updated_at", time.Now().UTC()).Where(sq.Eq{"id": a.ID, "owner_id": a.OwnerID, "revision": expected}).ToSql()
	if err != nil {
		return a, fmt.Errorf("build draft update: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return a, fmt.Errorf("save assistant draft: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return a, ErrConflict
	}
	return r.Get(ctx, a.OwnerID, a.ID)
}
func (r *Repository) Publish(ctx context.Context, owner, id string, revision int, principal Principal) (Assistant, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Assistant{}, fmt.Errorf("begin assistant publish: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	a, err := getAssistant(ctx, tx, owner, id, true)
	if err != nil {
		return a, err
	}
	if a.Revision != revision {
		return a, ErrConflict
	}
	if a.Definition == nil || a.Readiness != "ready" {
		return a, ErrTrialRequired
	}
	if principal.UserID != owner || principal.RoleID != a.RoleID || principal.ScopeDigest == "" {
		return a, fmt.Errorf("ASSISTANT_SCOPE_CHANGED")
	}
	q, args, err := sql.Select("status", "COALESCE(output->>'outcome','')", "principal->>'scopeDigest'").From("agent_assistant_runs").Where(sq.Eq{"assistant_id": id, "revision": revision, "kind": "trial"}).OrderBy("created_at DESC").Limit(1).ToSql()
	if err != nil {
		return a, fmt.Errorf("build trial gate: %w", err)
	}
	var trialStatus, trialOutcome string
	var trialScope *string
	if err = tx.QueryRow(ctx, q, args...).Scan(&trialStatus, &trialOutcome, &trialScope); errors.Is(err, pgx.ErrNoRows) {
		return a, ErrTrialRequired
	} else if err != nil {
		return a, fmt.Errorf("check assistant trial: %w", err)
	}
	if trialStatus != "COMPLETED" || (trialOutcome != "finding" && trialOutcome != "no_change") {
		return a, ErrTrialRequired
	}
	if trialScope == nil || *trialScope != principal.ScopeDigest {
		return a, fmt.Errorf("ASSISTANT_SCOPE_CHANGED")
	}
	if a.PublishedRevision != nil && *a.PublishedRevision == revision {
		return a, nil
	}
	raw, _ := json.Marshal(a.Definition)
	q, args, err = sql.Insert("agent_assistant_versions").Columns("assistant_id", "revision", "definition", "role_id", "locale", "timezone", "scope_digest").Values(id, revision, raw, a.RoleID, a.Locale, a.Timezone, principal.ScopeDigest).Suffix("ON CONFLICT (assistant_id,revision) DO NOTHING").ToSql()
	if err != nil {
		return a, fmt.Errorf("build assistant version: %w", err)
	}
	if _, err = tx.Exec(ctx, q, args...); err != nil {
		return a, fmt.Errorf("save assistant version: %w", err)
	}
	q, args, err = sql.Update("agent_assistants").Set("published_revision", revision).Set("state", "active").Set("next_run_at", NextRun(a.Definition.Trigger, time.Now())).Set("last_error", "").Set("updated_at", time.Now().UTC()).Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return a, fmt.Errorf("build assistant activation: %w", err)
	}
	if _, err = tx.Exec(ctx, q, args...); err != nil {
		return a, fmt.Errorf("activate assistant: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return a, fmt.Errorf("commit assistant publish: %w", err)
	}
	return r.Get(ctx, owner, id)
}
func (r *Repository) SetState(ctx context.Context, owner, id, state string) (Assistant, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Assistant{}, fmt.Errorf("begin assistant state change: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	a, err := getAssistant(ctx, tx, owner, id, true)
	if err != nil {
		return a, err
	}
	if a.PublishedDefinition == nil {
		return a, ErrTrialRequired
	}
	var next *time.Time
	if state == "active" {
		next = NextRun(a.PublishedDefinition.Trigger, time.Now())
	}
	q, args, err := sql.Update("agent_assistants").Set("state", state).Set("next_run_at", next).Set("last_error", "").Set("updated_at", time.Now().UTC()).Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return a, fmt.Errorf("build assistant state change: %w", err)
	}
	if _, err = tx.Exec(ctx, q, args...); err != nil {
		return a, fmt.Errorf("update assistant state: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return a, fmt.Errorf("commit assistant state: %w", err)
	}
	return r.Get(ctx, owner, id)
}
func (r *Repository) Published(ctx context.Context, eventType string, due bool) ([]Assistant, error) {
	q := assistantSelect().Where(sq.Eq{"a.state": "active"})
	if due {
		q = q.Where(sq.LtOrEq{"a.next_run_at": time.Now().UTC()})
	} else {
		q = q.Where(sq.Expr("v.definition->'trigger'->>'kind'='event' AND v.definition->'trigger'->>'eventType'=?", eventType))
	}
	if due {
		q = q.Limit(200)
	}
	text, args, err := q.OrderBy("a.next_run_at NULLS LAST", "a.id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build active assistant list: %w", err)
	}
	rows, err := r.pool.Query(ctx, text, args...)
	if err != nil {
		return nil, fmt.Errorf("list active assistants: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Assistant, error) { return scanAssistant(row) })
}
func (r *Repository) VersionPrincipal(ctx context.Context, id string, revision int) (Principal, string, string, error) {
	var p Principal
	var locale, tz string
	q, args, err := sql.Select("a.owner_id", "v.role_id", "v.locale", "v.timezone", "v.scope_digest").From("agent_assistants a").Join("agent_assistant_versions v ON v.assistant_id=a.id").Where(sq.Eq{"a.id": id, "v.revision": revision}).ToSql()
	if err != nil {
		return p, locale, tz, fmt.Errorf("build published principal: %w", err)
	}
	err = r.pool.QueryRow(ctx, q, args...).Scan(&p.UserID, &p.RoleID, &locale, &tz, &p.ScopeDigest)
	if err != nil {
		return p, locale, tz, fmt.Errorf("load published principal: %w", err)
	}
	if p.ScopeDigest == "" {
		return p, locale, tz, fmt.Errorf("ASSISTANT_SCOPE_CHANGED")
	}
	return p, locale, tz, nil
}

const runColumns = "id,assistant_id,owner_id,revision,kind,status,connector_id,request,principal,output,error_code,error_message,tools,attempts,lease_token,created_at,started_at,completed_at,read_at,notify_visible"

func scanRun(row pgx.Row) (Run, error) {
	var r Run
	var request, principal, output, tools []byte
	err := row.Scan(&r.ID, &r.AssistantID, &r.OwnerID, &r.Revision, &r.Kind, &r.Status, &r.ConnectorID, &request, &principal, &output, &r.ErrorCode, &r.ErrorMessage, &tools, &r.Attempts, &r.LeaseToken, &r.CreatedAt, &r.StartedAt, &r.CompletedAt, &r.ReadAt, &r.NotifyVisible)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, fmt.Errorf("scan assistant run: %w", err)
	}
	for _, pair := range []struct {
		raw    []byte
		target any
	}{{request, &r.Request}, {principal, &r.Principal}, {tools, &r.Tools}} {
		if err = json.Unmarshal(pair.raw, pair.target); err != nil {
			return r, fmt.Errorf("decode assistant run: %w", err)
		}
	}
	if len(output) > 0 {
		if err = json.Unmarshal(output, &r.Output); err != nil {
			return r, fmt.Errorf("decode assistant result: %w", err)
		}
	}
	return r, nil
}
func (r *Repository) GetRun(ctx context.Context, owner, id string) (Run, error) {
	q := sql.Select(runColumns).From("agent_assistant_runs").Where(sq.Eq{"id": id})
	if owner != "" {
		q = q.Where(sq.Eq{"owner_id": owner})
	}
	text, args, err := q.ToSql()
	if err != nil {
		return Run{}, fmt.Errorf("build assistant run lookup: %w", err)
	}
	return scanRun(r.pool.QueryRow(ctx, text, args...))
}
func (r *Repository) Runs(ctx context.Context, owner, id string) ([]Run, error) {
	// Ownership is checked even for an empty run list; other users cannot probe IDs.
	if _, err := r.Get(ctx, owner, id); err != nil {
		return nil, err
	}
	q, args, err := sql.Select(runColumns).From("agent_assistant_runs").Where(sq.Eq{"owner_id": owner, "assistant_id": id}).OrderBy("created_at DESC").Limit(50).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build run list: %w", err)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list assistant runs: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Run, error) { return scanRun(row) })
	if out == nil {
		out = []Run{}
	}
	return out, err
}

// Enqueue atomically binds a run to the locally owned definition. The assistant
// row serializes publish/pause/scheduling and per-assistant concurrency admission.
func (r *Repository) Enqueue(ctx context.Context, a Assistant, run Run, key string, scheduled bool) (Run, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return run, fmt.Errorf("begin assistant enqueue: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	current, err := getAssistant(ctx, tx, a.OwnerID, a.ID, true)
	if err != nil {
		return run, err
	}
	q, args, err := sql.Select(runColumns).From("agent_assistant_runs").Where(sq.Eq{"assistant_id": a.ID, "dedupe_key": key}).ToSql()
	if err != nil {
		return run, fmt.Errorf("build run dedupe: %w", err)
	}
	existing, findErr := scanRun(tx.QueryRow(ctx, q, args...))
	if findErr == nil {
		return existing, nil
	}
	if !errors.Is(findErr, ErrNotFound) {
		return run, findErr
	}
	if run.Kind == "trial" {
		if current.Revision != run.Revision {
			return run, ErrConflict
		}
	} else if current.State != "active" || current.PublishedRevision == nil || *current.PublishedRevision != run.Revision {
		return run, ErrConflict
	}
	q, args, err = sql.Select("COUNT(*)").From("agent_assistant_runs").Where(sq.Eq{"assistant_id": a.ID, "status": []string{"QUEUED", "RUNNING", "CANCELLING"}}).ToSql()
	if err != nil {
		return run, fmt.Errorf("build run admission: %w", err)
	}
	var active int
	if err = tx.QueryRow(ctx, q, args...).Scan(&active); err != nil {
		return run, fmt.Errorf("check active assistant runs: %w", err)
	}
	if active >= 1 {
		return run, fmt.Errorf("ASSISTANT_ALREADY_RUNNING")
	}
	request, _ := json.Marshal(run.Request)
	principal, _ := json.Marshal(run.Principal)
	q, args, err = sql.Insert("agent_assistant_runs").Columns("id", "assistant_id", "owner_id", "revision", "kind", "connector_id", "request", "principal", "dedupe_key").Values(run.ID, a.ID, a.OwnerID, run.Revision, run.Kind, run.ConnectorID, request, principal, key).ToSql()
	if err != nil {
		return run, fmt.Errorf("build assistant run insert: %w", err)
	}
	if _, err = tx.Exec(ctx, q, args...); err != nil {
		return run, fmt.Errorf("enqueue assistant run: %w", err)
	}
	u := sql.Update("agent_assistants").Set("last_run_at", time.Now().UTC()).Where(sq.Eq{"id": a.ID})
	if scheduled {
		u = u.Set("next_run_at", NextRun(run.Request.Definition.Trigger, time.Now()))
	}
	q, args, err = u.ToSql()
	if err != nil {
		return run, fmt.Errorf("build schedule advance: %w", err)
	}
	if _, err = tx.Exec(ctx, q, args...); err != nil {
		return run, fmt.Errorf("advance assistant schedule: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return run, fmt.Errorf("commit assistant enqueue: %w", err)
	}
	return r.GetRun(ctx, a.OwnerID, run.ID)
}

// Claim leases one local polling task. The remote run uses the same immutable
// run ID, so retries after a process/network failure never submit a new job.
func (r *Repository) Claim(ctx context.Context) (Run, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Run{}, fmt.Errorf("begin run claim: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	now := time.Now().UTC()
	q, args, err := sql.Select(runColumns).From("agent_assistant_runs").Where(sq.Eq{"status": []string{"QUEUED", "RUNNING", "CANCELLING"}}).Where(sq.LtOrEq{"next_poll_at": now}).Where(sq.Or{sq.Eq{"lease_until": nil}, sq.LtOrEq{"lease_until": now}}).OrderBy("next_poll_at", "created_at").Limit(1).Suffix("FOR UPDATE SKIP LOCKED").ToSql()
	if err != nil {
		return Run{}, fmt.Errorf("build run claim: %w", err)
	}
	run, err := scanRun(tx.QueryRow(ctx, q, args...))
	if err != nil {
		return run, err
	}
	token := uuid.NewString()
	q, args, err = sql.Update("agent_assistant_runs").Set("lease_token", token).Set("lease_until", now.Add(30*time.Second)).Set("attempts", run.Attempts+1).Where(sq.Eq{"id": run.ID}).ToSql()
	if err != nil {
		return run, fmt.Errorf("build run lease: %w", err)
	}
	if _, err = tx.Exec(ctx, q, args...); err != nil {
		return run, fmt.Errorf("claim assistant run: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return run, fmt.Errorf("commit run claim: %w", err)
	}
	run.LeaseToken = token
	run.Attempts++
	return run, nil
}
func (r *Repository) Progress(ctx context.Context, run Run, remote RemoteRun, pollAfter time.Duration) error {
	state := remote.Status
	if state != "COMPLETED" && state != "FAILED" && state != "CANCELLED" {
		state = "RUNNING"
	}
	if run.Status == "CANCELLING" && state == "RUNNING" {
		state = "CANCELLING"
	}
	now := time.Now().UTC()
	terminal := state == "COMPLETED" || state == "FAILED" || state == "CANCELLED"
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin run progress: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	notify := false
	if terminal {
		q, args, e := sql.Select("last_notice_at").From("agent_assistants").Where(sq.Eq{"id": run.AssistantID}).Suffix("FOR UPDATE").ToSql()
		if e != nil {
			return fmt.Errorf("build notification gate: %w", e)
		}
		var last *time.Time
		if e = tx.QueryRow(ctx, q, args...).Scan(&last); e != nil {
			return fmt.Errorf("read notification gate: %w", e)
		}
		outcome := ""
		if remote.Output != nil {
			outcome = remote.Output.Outcome
		}
		notify = ShouldNotify(run.Kind, state, outcome, run.Request.Definition, last, now)
	}
	output, _ := json.Marshal(remote.Output)
	tools, _ := json.Marshal(remote.Tools)
	code, message := "", ""
	if remote.Error != nil {
		code = remote.Error.Code
		message = remote.Error.Message
	}
	u := sql.Update("agent_assistant_runs").Set("status", state).Set("tools", tools).Set("error_code", code).Set("error_message", message).Set("lease_token", "").Set("lease_until", nil).Set("next_poll_at", now.Add(pollAfter))
	if remote.StartedAt != nil {
		u = u.Set("started_at", remote.StartedAt)
	}
	if state == "COMPLETED" {
		u = u.Set("output", output)
	}
	if terminal {
		u = u.Set("completed_at", now).Set("notify_visible", notify)
	}
	q, args, err := u.Where(sq.Eq{"id": run.ID, "lease_token": run.LeaseToken, "status": run.Status}).ToSql()
	if err != nil {
		return fmt.Errorf("build run progress: %w", err)
	}
	tag, err := tx.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("save run progress: %w", err)
	}
	if tag.RowsAffected() == 1 && notify {
		q, args, err = sql.Update("agent_assistants").Set("last_notice_at", now).Where(sq.Eq{"id": run.AssistantID}).ToSql()
		if err != nil {
			return fmt.Errorf("build notification timestamp: %w", err)
		}
		if _, err = tx.Exec(ctx, q, args...); err != nil {
			return fmt.Errorf("save notification timestamp: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit run progress: %w", err)
	}
	return nil
}
func (r *Repository) Cancel(ctx context.Context, owner, id string) (Run, error) {
	q, args, err := sql.Update("agent_assistant_runs").Set("status", "CANCELLING").Set("next_poll_at", time.Now().UTC()).Set("lease_until", nil).Set("lease_token", "").Where(sq.Eq{"id": id, "owner_id": owner, "status": []string{"QUEUED", "RUNNING"}}).ToSql()
	if err != nil {
		return Run{}, fmt.Errorf("build run cancel: %w", err)
	}
	if _, err = r.pool.Exec(ctx, q, args...); err != nil {
		return Run{}, fmt.Errorf("cancel assistant run: %w", err)
	}
	return r.GetRun(ctx, owner, id)
}
func (r *Repository) ReadRun(ctx context.Context, owner, id string) error {
	q, args, err := sql.Update("agent_assistant_runs").Set("read_at", time.Now().UTC()).Where(sq.Eq{"id": id, "owner_id": owner}).ToSql()
	if err != nil {
		return fmt.Errorf("build run read state: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("save run read state: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
func (r *Repository) Block(ctx context.Context, id string, revision int, cause error) error {
	q, args, err := sql.Update("agent_assistants").Set("state", "blocked").Set("next_run_at", nil).Set("last_error", cause.Error()).Where(sq.Eq{"id": id, "published_revision": revision, "state": "active"}).ToSql()
	if err != nil {
		return fmt.Errorf("build assistant block: %w", err)
	}
	if _, err = r.pool.Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("block assistant: %w", err)
	}
	return nil
}

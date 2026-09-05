package agentassistant

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/attention"
)

func integrationRepository(t *testing.T) (*Repository, string) {
	t.Helper()
	url := os.Getenv("ASSISTANT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set ASSISTANT_TEST_DATABASE_URL for PostgreSQL lifecycle tests")
	}
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	schema := "assistant_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	// The schema name is generated internally, never supplied by users.
	if _, err = adminPool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close(); _, _ = adminPool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); adminPool.Close() })
	if _, err = pool.Exec(ctx, `CREATE TABLE users(id UUID PRIMARY KEY, username TEXT NOT NULL, source TEXT NOT NULL, status TEXT NOT NULL);
 CREATE TABLE roles(id UUID PRIMARY KEY,name TEXT NOT NULL);
 CREATE TABLE user_roles(user_id UUID NOT NULL,role_id UUID NOT NULL);
 CREATE TABLE devices(id UUID PRIMARY KEY,deleted_at TIMESTAMPTZ);
 CREATE TABLE device_group_members(device_id UUID, group_id UUID);`); err != nil {
		t.Fatal(err)
	}
	baseline, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(baseline)
	start := strings.Index(text, "CREATE TABLE IF NOT EXISTS agent_assistants (")
	if start < 0 {
		t.Fatal("missing baseline DDL")
	}
	end := strings.Index(text[start:], "-- +omcgo MainReconcileEnd")
	if end < 0 {
		t.Fatal("missing baseline reconciliation boundary")
	}
	if _, err = pool.Exec(ctx, text[start:start+end]); err != nil {
		t.Fatalf("real baseline DDL: %v", err)
	}
	owner := uuid.NewString()
	if _, err = pool.Exec(ctx, "INSERT INTO users VALUES ($1,'creator',$2,'active')", owner, string(admin.UserSourceBuiltIn)); err != nil {
		t.Fatal(err)
	}
	return NewRepository(pool), owner
}
func mustDraft(t *testing.T, r *Repository, owner string) Assistant {
	t.Helper()
	ctx := context.Background()
	a, err := r.Create(ctx, owner, uuid.Nil.String(), uuid.NewString(), "en-US", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	d := testDefinition()
	a.Definition = &d
	a.Readiness = "ready"
	a, err = r.SaveDraft(ctx, a, a.Revision)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func mustEnqueue(t *testing.T, r *Repository, a Assistant, kind, key string) Run {
	t.Helper()
	run := Run{ID: uuid.NewString(), AssistantID: a.ID, OwnerID: a.OwnerID, Revision: a.Revision, Kind: kind, ConnectorID: "connector", Principal: Principal{UserID: a.OwnerID, RoleID: a.RoleID}}
	run.Principal.ScopeDigest = scopeDigest(&admin.Claims{UserID: uuid.MustParse(a.OwnerID), IsSuperAdmin: true}, nil)
	run.Request = ExecutionRequest{RunID: run.ID, AssistantID: a.ID, Revision: a.Revision, Definition: *a.Definition}
	got, err := r.Enqueue(context.Background(), a, run, key, false)
	if err != nil {
		t.Fatal(err)
	}
	return got
}
func completeNext(t *testing.T, r *Repository) Run {
	t.Helper()
	ctx := context.Background()
	run, err := r.Claim(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = r.Progress(ctx, run, RemoteRun{ID: run.ID, Status: "COMPLETED", Output: &Result{Outcome: "finding", Title: "Observed finding", Summary: "Real integration fixture", Facts: []Fact{}, Hypotheses: []string{}, NextSteps: []string{}}, Tools: []ToolProgress{}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.GetRun(ctx, run.OwnerID, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	return got
}
func TestPostgresOwnershipVersionTrialAndIdempotency(t *testing.T) {
	r, owner := integrationRepository(t)
	ctx := context.Background()
	a := mustDraft(t, r, owner)
	if _, err := r.Get(ctx, uuid.NewString(), a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner assistant: %v", err)
	}
	if _, err := r.Publish(ctx, owner, a.ID, a.Revision); !errors.Is(err, ErrTrialRequired) {
		t.Fatalf("fake trial accepted: %v", err)
	}
	run := mustEnqueue(t, r, a, "trial", "trial-key")
	again := mustEnqueue(t, r, a, "trial", "trial-key")
	if again.ID != run.ID {
		t.Fatal("idempotency failed")
	}
	if _, err := r.GetRun(ctx, uuid.NewString(), run.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner run: %v", err)
	}
	completeNext(t, r)
	published, err := r.Publish(ctx, owner, a.ID, a.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if published.State != "active" || published.PublishedRevision == nil {
		t.Fatal(published)
	}
	oldGoal := published.PublishedDefinition.Goal
	a = published
	a.Definition.Goal = "Revised goal"
	a, err = r.SaveDraft(ctx, a, a.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if a.PublishedDefinition.Goal != oldGoal || *a.PublishedRevision == a.Revision {
		t.Fatal("draft overwrote immutable version")
	}
	if _, err = r.SaveDraft(ctx, a, a.Revision-1); !errors.Is(err, ErrConflict) {
		t.Fatalf("lost update accepted: %v", err)
	}
	if _, err = r.Publish(ctx, owner, a.ID, a.Revision); !errors.Is(err, ErrTrialRequired) {
		t.Fatalf("stale trial published: %v", err)
	}
	if _, err = r.SetState(ctx, owner, a.ID, "paused"); err != nil {
		t.Fatal(err)
	}
	due, err := r.Published(ctx, "", true)
	if err != nil || len(due) != 0 {
		t.Fatalf("paused scheduled: %v %v", due, err)
	}
}
func TestPostgresCancellationFencingAndRecovery(t *testing.T) {
	r, owner := integrationRepository(t)
	ctx := context.Background()
	a := mustDraft(t, r, owner)
	mustEnqueue(t, r, a, "trial", "first")
	first, err := r.Claim(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Claim(ctx); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double claim: %v", err)
	}
	if _, err = r.pool.Exec(ctx, "UPDATE agent_assistant_runs SET lease_until=NOW()-INTERVAL '1 second' WHERE id=$1", first.ID); err != nil {
		t.Fatal(err)
	}
	second, err := r.Claim(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if second.LeaseToken == first.LeaseToken {
		t.Fatal("lease not fenced")
	}
	if err = r.Progress(ctx, first, RemoteRun{Status: "COMPLETED", Output: &Result{Outcome: "finding"}}, 0); err != nil {
		t.Fatal(err)
	}
	run, _ := r.GetRun(ctx, owner, first.ID)
	if run.Status == "COMPLETED" {
		t.Fatal("old lease changed terminal state")
	}
	if _, err = r.Cancel(ctx, owner, first.ID); err != nil {
		t.Fatal(err)
	}
	if err = r.Progress(ctx, second, RemoteRun{Status: "COMPLETED", Output: &Result{Outcome: "finding"}}, 0); err != nil {
		t.Fatal(err)
	}
	run, _ = r.GetRun(ctx, owner, first.ID)
	if run.Status != "CANCELLING" {
		t.Fatalf("cancel overwritten: %s", run.Status)
	}
	third, err := r.Claim(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Progress(ctx, third, RemoteRun{Status: "CANCELLED"}, 0); err != nil {
		t.Fatal(err)
	}
	run, _ = r.GetRun(ctx, owner, first.ID)
	if run.Status != "CANCELLED" || run.NotifyVisible {
		t.Fatal(run)
	}
}
func TestPostgresPrivateInboxAndNotificationCooldown(t *testing.T) {
	r, owner := integrationRepository(t)
	ctx := context.Background()
	a := mustDraft(t, r, owner)
	mustEnqueue(t, r, a, "trial", "first")
	completeNext(t, r)
	var err error
	a, err = r.Publish(ctx, owner, a.ID, a.Revision)
	if err != nil {
		t.Fatal(err)
	}
	mustEnqueue(t, r, a, "manual", "manual-1")
	first := completeNext(t, r)
	if !first.NotifyVisible {
		t.Fatal("missing first notification")
	}
	mustEnqueue(t, r, a, "manual", "manual-2")
	second := completeNext(t, r)
	if second.NotifyVisible {
		t.Fatal("cooldown ignored")
	}
	source := NewAttentionSource(r)
	scope := attention.Scope{UserID: uuid.MustParse(owner), IsSuperAdmin: true}
	inbox, err := source.ListWindow(ctx, scope, 0, 20)
	if err != nil || inbox.Total != 1 {
		t.Fatalf("inbox: %+v %v", inbox, err)
	}
	other := attention.Scope{UserID: uuid.New(), IsSuperAdmin: true}
	inbox, err = source.ListWindow(ctx, other, 0, 20)
	if err != nil || inbox.Total != 0 {
		t.Fatalf("superadmin not owner leaked: %+v %v", inbox, err)
	}
	if err = r.ReadRun(ctx, owner, first.ID); err != nil {
		t.Fatal(err)
	}
	inbox, err = source.ListWindow(ctx, scope, 0, 20)
	if err != nil || inbox.Total != 0 {
		t.Fatal(inbox, err)
	}
	// Role-based branch executes real Squirrel SQL, not just the superadmin path.
	scope.IsSuperAdmin = false
	scope.VisibleGroups = []uuid.UUID{uuid.New()}
	if _, err = source.ListWindow(ctx, scope, 0, 20); err != nil {
		t.Fatal(err)
	}
}
func TestPostgresCurrentPrincipalNeverInventsPrivileges(t *testing.T) {
	r, owner := integrationRepository(t)
	ctx := context.Background()
	role := uuid.NewString()
	_, err := r.pool.Exec(ctx, "UPDATE users SET source='local' WHERE id=$1", owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.CurrentPrincipal(ctx, Principal{UserID: owner, RoleID: role}); err == nil {
		t.Fatal("missing role accepted")
	}
	_, _ = r.pool.Exec(ctx, "INSERT INTO roles VALUES ($1,'viewer')", role)
	_, _ = r.pool.Exec(ctx, "INSERT INTO user_roles VALUES ($1,$2)", owner, role)
	c, err := r.CurrentPrincipal(ctx, Principal{UserID: owner, RoleID: role})
	if err != nil || c.IsSuperAdmin {
		t.Fatal(c, err)
	}
	_, _ = r.pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", owner)
	if _, err = r.CurrentPrincipal(ctx, Principal{UserID: owner, RoleID: role}); err == nil {
		t.Fatal("revoked account accepted")
	}
}

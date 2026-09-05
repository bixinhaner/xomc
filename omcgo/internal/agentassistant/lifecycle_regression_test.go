package agentassistant

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type lifecycleRemote struct {
	result    RemoteRun
	cancelled bool
}

func (r *lifecycleRemote) AssistantConnection(context.Context) (string, error) {
	return "connector", nil
}
func (r *lifecycleRemote) PlanAssistant(context.Context, PlanRequest) (PlanResponse, error) {
	return PlanResponse{}, fmt.Errorf("unused")
}
func (r *lifecycleRemote) SubmitAssistant(context.Context, string, ExecutionRequest) (RemoteRun, error) {
	return r.result, nil
}
func (r *lifecycleRemote) GetAssistantRun(context.Context, string, string) (RemoteRun, error) {
	return r.result, nil
}
func (r *lifecycleRemote) CancelAssistantRun(context.Context, string, string) error {
	r.cancelled = true
	return nil
}

type lifecycleAccess struct{ groups []uuid.UUID }

func (a *lifecycleAccess) GetUserVisibleGroupIDs(context.Context, uuid.UUID, bool) ([]uuid.UUID, error) {
	return a.groups, nil
}
func (a *lifecycleAccess) GetDeviceGroupIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return a.groups, nil
}
func (a *lifecycleAccess) CheckPermission(context.Context, uuid.UUID, string, string) (bool, error) {
	return true, nil
}
func (a *lifecycleAccess) AssistantCapabilityCatalog() []map[string]any {
	return []map[string]any{{"operationId": "get.devices.by_id", "path": "/api/v1/devices/:id", "deviceScoped": true}}
}
func (a *lifecycleAccess) HandbookMetadata() (map[string]any, error) {
	return map[string]any{"handbookDigest": "test"}, nil
}

func TestPostgresPublishedGrantCannotExpandBetweenRuns(t *testing.T) {
	r, owner := integrationRepository(t)
	ctx := context.Background()
	a := mustDraft(t, r, owner)
	mustEnqueue(t, r, a, "trial", "trial")
	completeNext(t, r)
	p := publishedPrincipal(owner, a.RoleID)
	changed := p
	changed.ScopeDigest = "different-grant"
	if _, err := r.Publish(ctx, owner, a.ID, a.Revision, changed); err == nil || !strings.Contains(err.Error(), "SCOPE_CHANGED") {
		t.Fatalf("trial grant change accepted: %v", err)
	}
	a, err := r.Publish(ctx, owner, a.ID, a.Revision, p)
	if err != nil {
		t.Fatal(err)
	}
	saved, _, _, err := r.VersionPrincipal(ctx, a.ID, *a.PublishedRevision)
	if err != nil || saved.ScopeDigest != p.ScopeDigest {
		t.Fatalf("grant not frozen: %+v %v", saved, err)
	}
	access := &lifecycleAccess{}
	s := NewService(r, &lifecycleRemote{}, access, access, access, access, nil)
	if _, err = s.enqueue(ctx, a, "manual", "before", nil, false); err != nil {
		t.Fatal(err)
	}
	completeNext(t, r)
	access.groups = []uuid.UUID{uuid.New()}
	if _, err = s.enqueue(ctx, a, "schedule", "after", nil, true); err == nil || !strings.Contains(err.Error(), "SCOPE_CHANGED") {
		t.Fatalf("new run silently rebound expanded grant: %v", err)
	}
	if _, err = r.pool.Exec(ctx, "UPDATE agent_assistant_versions SET scope_digest='' WHERE assistant_id=$1", a.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = r.VersionPrincipal(ctx, a.ID, *a.PublishedRevision); err == nil {
		t.Fatal("legacy unbound grant accepted")
	}
}

func TestPostgresCancellationUsesRemoteTerminalState(t *testing.T) {
	for _, status := range []string{"COMPLETED", "FAILED", "CANCELLED", "RUNNING"} {
		t.Run(status, func(t *testing.T) {
			r, owner := integrationRepository(t)
			ctx := context.Background()
			a := mustDraft(t, r, owner)
			run := mustEnqueue(t, r, a, "trial", "trial")
			if _, err := r.Cancel(ctx, owner, run.ID); err != nil {
				t.Fatal(err)
			}
			remote := &lifecycleRemote{result: RemoteRun{ID: run.ID, Status: status}}
			if status == "COMPLETED" {
				remote.result.Output = &Result{Outcome: "finding", Title: "Completed before cancellation"}
			}
			s := NewService(r, remote, nil, nil, nil, nil, nil)
			s.poll(ctx)
			got, err := r.GetRun(ctx, owner, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := status
			if status == "RUNNING" {
				want = "CANCELLING"
			}
			if !remote.cancelled || got.Status != want {
				t.Fatalf("cancel=%v state=%s want=%s", remote.cancelled, got.Status, want)
			}
			if status == "COMPLETED" && got.Output == nil {
				t.Fatal("completed result discarded")
			}
		})
	}
}

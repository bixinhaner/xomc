package backup

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
)

// pubPolicyGetter satisfies PolicyGetter for publisher tests.
type pubPolicyGetter struct {
	policy *BackupPolicy
	err    error
}

func (p *pubPolicyGetter) Get(_ context.Context) (*BackupPolicy, error) {
	if p.err != nil {
		return nil, p.err
	}
	cp := *p.policy
	return &cp, nil
}

// pubEventBus captures Publish calls for assertions.
type pubEventBus struct {
	mu         sync.Mutex
	calls      []pubCall
	publishErr error
}

type pubCall struct {
	subject string
	payload FailureAlarmPayload
}

func (b *pubEventBus) Publish(_ context.Context, subject string, evt event.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.publishErr != nil {
		return b.publishErr
	}
	var p FailureAlarmPayload
	if err := evt.DecodePayload(&p); err != nil {
		return err
	}
	b.calls = append(b.calls, pubCall{subject: subject, payload: p})
	return nil
}

func (b *pubEventBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return nil, nil
}
func (b *pubEventBus) QueueSubscribe(_, _ string, _ event.EventHandler) (event.Subscription, error) {
	return nil, nil
}
func (b *pubEventBus) Close() error { return nil }

// ---------------------------------------------------------------------------
// V1 — alertOnFailure=true publishes alarm.raised
// ---------------------------------------------------------------------------

func TestPublishFailureAlarm_AlertOnTruePublishes(t *testing.T) {
	policy := DefaultPolicy()
	policy.AlertOnFailure = true
	policy.AlertEmail = "ops@example.com"
	getter := &pubPolicyGetter{policy: policy}
	bus := &pubEventBus{}
	metrics := NewPolicyMetrics(nil)

	errMsg := "no devices were successfully queued"
	task := &BackupTask{
		ID:           uuid.New(),
		TargetIDs:    []string{"SN-1", "SN-2", "SN-3"},
		ErrorMessage: &errMsg,
	}

	err := PublishFailureAlarm(context.Background(), getter, bus, metrics, task)
	require.NoError(t, err)
	require.Len(t, bus.calls, 1)
	c := bus.calls[0]
	assert.Equal(t, event.SubjectAlarmRaised, c.subject)
	assert.Equal(t, "backup", c.payload.Source)
	assert.Equal(t, "major", c.payload.Severity)
	assert.Equal(t, "backup_task_failed", c.payload.Identifier)
	assert.Equal(t, task.ID.String(), c.payload.TaskID)
	assert.Equal(t, 3, c.payload.TargetCount)
	assert.Equal(t, errMsg, c.payload.ErrorMessage)
	assert.Equal(t, "ops@example.com", c.payload.AlertEmail)
}

// ---------------------------------------------------------------------------
// V2 — alertOnFailure=false short-circuits (no publish)
// ---------------------------------------------------------------------------

func TestPublishFailureAlarm_AlertOffSkips(t *testing.T) {
	policy := DefaultPolicy()
	policy.AlertOnFailure = false
	getter := &pubPolicyGetter{policy: policy}
	bus := &pubEventBus{}
	task := &BackupTask{ID: uuid.New(), TargetIDs: []string{"SN-1"}}

	err := PublishFailureAlarm(context.Background(), getter, bus, NewPolicyMetrics(nil), task)
	require.NoError(t, err)
	assert.Empty(t, bus.calls, "no event should be published when alert_on_failure=false")
}

// Bonus — policy fetch error propagates without panicking.
func TestPublishFailureAlarm_PolicyFetchError(t *testing.T) {
	getter := &pubPolicyGetter{err: errors.New("DB outage")}
	bus := &pubEventBus{}
	task := &BackupTask{ID: uuid.New(), TargetIDs: []string{"SN-1"}}

	err := PublishFailureAlarm(context.Background(), getter, bus, NewPolicyMetrics(nil), task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB outage")
	assert.Empty(t, bus.calls)
}

// Bonus — nil-task guard.
func TestPublishFailureAlarm_NilTask(t *testing.T) {
	getter := &pubPolicyGetter{policy: DefaultPolicy()}
	err := PublishFailureAlarm(context.Background(), getter, &pubEventBus{}, NewPolicyMetrics(nil), nil)
	require.Error(t, err)
}

// Review fix HIGH-1: nil PolicyGetter and nil EventBus are now defensively rejected.
func TestPublishFailureAlarm_NilPolicyService(t *testing.T) {
	task := &BackupTask{ID: uuid.New(), TargetIDs: []string{"SN-1"}}
	err := PublishFailureAlarm(context.Background(), nil, &pubEventBus{}, NewPolicyMetrics(nil), task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "policyService")
}

func TestPublishFailureAlarm_NilEventBus(t *testing.T) {
	task := &BackupTask{ID: uuid.New(), TargetIDs: []string{"SN-1"}}
	getter := &pubPolicyGetter{policy: DefaultPolicy()}
	err := PublishFailureAlarm(context.Background(), getter, nil, NewPolicyMetrics(nil), task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event bus")
}

// V5 (T-0084) — policy.AlertSeverity drives payload.Severity.
func TestPublishFailureAlarm_PolicyDrivenSeverity(t *testing.T) {
	cases := []struct {
		name     string
		severity string
		want     string
	}{
		{"warning passthrough", "warning", "warning"},
		{"major passthrough", "major", "major"},
		{"critical passthrough", "critical", "critical"},
		{"empty falls back to major (V8 fallback)", "", "major"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			policy := DefaultPolicy()
			policy.AlertOnFailure = true
			policy.AlertSeverity = tc.severity
			getter := &pubPolicyGetter{policy: policy}
			bus := &pubEventBus{}
			task := &BackupTask{ID: uuid.New(), TargetIDs: []string{"SN-1"}}

			err := PublishFailureAlarm(context.Background(), getter, bus, NewPolicyMetrics(nil), task)
			require.NoError(t, err)
			require.Len(t, bus.calls, 1)
			assert.Equal(t, tc.want, bus.calls[0].payload.Severity)
		})
	}
}

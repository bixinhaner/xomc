package task

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeOpenRepo struct {
	tasks []*Task
	err   error
}

func (f *fakeOpenRepo) ListOpenByDeviceAndMethods(_ context.Context, _ string, _ []string) ([]*Task, error) {
	return f.tasks, f.err
}

type fakeCompleter struct {
	completedIDs []string
	completedRes []json.RawMessage
	err          error
}

func (f *fakeCompleter) MarkTaskCompleted(_ context.Context, id string, res json.RawMessage) error {
	f.completedIDs = append(f.completedIDs, id)
	f.completedRes = append(f.completedRes, res)
	return f.err
}

func newRebootCompleteEvent(t *testing.T, sn string, events []string) event.Event {
	t.Helper()
	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, map[string]interface{}{
		"device_id": map[string]interface{}{
			"SerialNumber": sn,
		},
		"events": events,
	})
	require.NoError(t, err)
	return evt
}

func TestRebootCloser_IgnoresWithoutMReboot(t *testing.T) {
	repo := &fakeOpenRepo{tasks: []*Task{{ID: "t-1", Method: "Reboot", Status: TaskStatusSent}}}
	completer := &fakeCompleter{}
	closer := NewRebootCloser(repo, completer, zap.NewNop())

	// 纯 "1 BOOT" = 自主重启，不是 ACS 主动下发的 Reboot 回包
	evt := newRebootCompleteEvent(t, "SN-1", []string{"1 BOOT"})
	require.NoError(t, closer.handle(context.Background(), evt))

	assert.Empty(t, completer.completedIDs, "should not close tasks without M Reboot event code")
}

func TestRebootCloser_ClosesOpenRebootTask(t *testing.T) {
	repo := &fakeOpenRepo{tasks: []*Task{
		{ID: "t-reboot-1", Method: "Reboot", Status: TaskStatusSent},
		{ID: "t-reboot-2", Method: "FactoryReset", Status: TaskStatusPending},
	}}
	completer := &fakeCompleter{}
	closer := NewRebootCloser(repo, completer, zap.NewNop())

	evt := newRebootCompleteEvent(t, "SN-REBOOT-OK", []string{"1 BOOT", "M Reboot"})
	require.NoError(t, closer.handle(context.Background(), evt))

	assert.ElementsMatch(t, []string{"t-reboot-1", "t-reboot-2"}, completer.completedIDs)
	require.Len(t, completer.completedRes, 2)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(completer.completedRes[0], &payload))
	assert.Equal(t, "reboot_complete_inform", payload["closed_by"])
}

func TestRebootCloser_NoOpenTasks(t *testing.T) {
	repo := &fakeOpenRepo{tasks: nil}
	completer := &fakeCompleter{}
	closer := NewRebootCloser(repo, completer, zap.NewNop())

	evt := newRebootCompleteEvent(t, "SN-CLEAN", []string{"M Reboot"})
	require.NoError(t, closer.handle(context.Background(), evt))

	assert.Empty(t, completer.completedIDs)
}

func TestRebootCloser_EmptySerial(t *testing.T) {
	repo := &fakeOpenRepo{tasks: []*Task{{ID: "t-1", Method: "Reboot"}}}
	completer := &fakeCompleter{}
	closer := NewRebootCloser(repo, completer, zap.NewNop())

	evt := newRebootCompleteEvent(t, "", []string{"M Reboot"})
	require.NoError(t, closer.handle(context.Background(), evt))

	assert.Empty(t, completer.completedIDs, "empty SN should short-circuit before repo call")
}

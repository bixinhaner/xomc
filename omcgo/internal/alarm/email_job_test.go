package alarm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

type alarmJobEnqueuerStub struct {
	request asyncjob.InsertRequest
	calls   int
}

func (s *alarmJobEnqueuerStub) Insert(_ context.Context, request asyncjob.InsertRequest) (uuid.UUID, error) {
	s.calls++
	s.request = request
	return uuid.New(), nil
}

func TestAsyncEmailDispatcherSnapshotsAndEnqueues(t *testing.T) {
	t.Parallel()
	jobs := &alarmJobEnqueuerStub{}
	dispatcher := NewAsyncEmailDispatcher(jobs, NewEmailMetrics(nil))
	require.NoError(t, dispatcher.Dispatch(
		context.Background(), []string{"ops@example.com", "OPS@example.com"}, "告警", "处理建议: 检查链路",
	))
	require.Equal(t, 1, jobs.calls)
	require.Equal(t, AlarmEmailJobType, jobs.request.JobType)
	var payload alarmEmailJobPayload
	require.NoError(t, json.Unmarshal(jobs.request.Payload, &payload))
	require.Equal(t, []string{"ops@example.com"}, payload.Recipients)
	require.Contains(t, payload.Body, "处理建议")
	require.NotEmpty(t, payload.DedupKey)
}

func TestAsyncEmailDispatcherSnapshotsAlarmContext(t *testing.T) {
	t.Parallel()
	jobs := &alarmJobEnqueuerStub{}
	dispatcher := NewAsyncEmailDispatcher(jobs, NewEmailMetrics(nil))
	alarmID := uuid.New()
	ruleID := uuid.New()
	require.NoError(t, dispatcher.DispatchAlarm(context.Background(), AlarmEmailDispatchRequest{
		AlarmID: alarmID, RuleID: ruleID, Lifecycle: AlarmEmailLifecycleCleared,
		Recipients: []string{"ops@example.com"}, Subject: "告警通知/Alarm Notification", Body: "当前状态: cleared",
	}))

	var payload alarmEmailJobPayload
	require.NoError(t, json.Unmarshal(jobs.request.Payload, &payload))
	require.Equal(t, alarmID.String(), payload.AlarmID)
	require.Equal(t, ruleID.String(), payload.RuleID)
	require.Equal(t, AlarmEmailLifecycleCleared, payload.Lifecycle)
	require.NotEmpty(t, payload.DedupKey)
	require.Equal(t, alarmID.String()+":"+ruleID.String()+":"+AlarmEmailLifecycleCleared, alarmEmailBusinessID(payload))
}

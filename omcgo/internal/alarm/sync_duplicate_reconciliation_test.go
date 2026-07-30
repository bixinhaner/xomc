package alarm

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestComputeDiffReconcilesDuplicateLocalInstances(t *testing.T) {
	deviceRaisedAt := time.Date(2026, 7, 30, 16, 25, 5, 0, time.Local)
	old := syncTestAlarm("11109", deviceRaisedAt.Add(-30*time.Minute), "old")
	current := syncTestAlarm("11109", deviceRaisedAt.Add(4*time.Second), "current")
	remote := syncTestAlarm("11109", deviceRaisedAt, "remote")
	remote.Description = "updated by device"

	diff := ComputeDiff(
		[]*model.Alarm{remote},
		[]*model.Alarm{old, current},
	)

	require.Len(t, diff.ToUpdate, 1)
	assert.Same(t, current, diff.ToUpdate[0].Local)
	assert.Same(t, remote, diff.ToUpdate[0].Remote)
	require.Len(t, diff.ToClearDuplicates, 1)
	assert.Same(t, old, diff.ToClearDuplicates[0].Duplicate)
	assert.Same(t, current, diff.ToClearDuplicates[0].Keeper)
	assert.Empty(t, diff.ToClear)
}

func TestComputeDiffKeepsLatestLocalOnEqualRaisedAtDistance(t *testing.T) {
	deviceRaisedAt := time.Date(2026, 7, 30, 16, 25, 5, 0, time.Local)
	earlier := syncTestAlarm("11112", deviceRaisedAt.Add(-time.Second), "earlier")
	later := syncTestAlarm("11112", deviceRaisedAt.Add(time.Second), "later")
	remote := syncTestAlarm("11112", deviceRaisedAt, "remote")
	remote.Description = "updated by device"

	diff := ComputeDiff(
		[]*model.Alarm{remote},
		[]*model.Alarm{earlier, later},
	)

	require.Len(t, diff.ToUpdate, 1)
	assert.Same(t, later, diff.ToUpdate[0].Local)
	require.Len(t, diff.ToClearDuplicates, 1)
	assert.Same(t, earlier, diff.ToClearDuplicates[0].Duplicate)
	assert.Same(t, later, diff.ToClearDuplicates[0].Keeper)
}

func TestProcessSyncReconcilesIssue227SixActiveAlarmsToTwo(t *testing.T) {
	const deviceSN = "120200055922C8B0068"
	store := newMockAlarmStore()
	metrics := NewAlarmMetrics(prometheus.NewRegistry())
	engine := newTestEngine(store)
	engine.metrics = metrics
	processor := NewAlarmSyncProcessor(engine, store, nil, nil, zap.NewNop())
	deviceID := uuid.New()

	old11109 := issue227LocalAlarm(deviceID, deviceSN, "11109", "S1 Setup failure", model.AlarmMajor, time.Date(2026, 7, 30, 15, 55, 28, 0, time.Local))
	old11112 := issue227LocalAlarm(deviceID, deviceSN, "11112", "SCTP link failure", model.AlarmCritical, time.Date(2026, 7, 30, 15, 55, 28, 0, time.Local))
	gps := issue227LocalAlarm(deviceID, deviceSN, "11190", "GPS unavailable", model.AlarmMajor, time.Date(2026, 7, 30, 15, 59, 59, 0, time.Local))
	gps.AdditionalInfo = map[string]string{
		additionalInfoManagedObjectInstance: "Device.FaultMgmt.ExpeditedEvent.",
		additionalInfoAdditionalInformation: "Gps unavailable, gps antenna open.",
	}
	clock := issue227LocalAlarm(deviceID, deviceSN, "11189", "Clock source sync fail", model.AlarmMajor, time.Date(2026, 7, 30, 15, 59, 59, 0, time.Local))
	clock.AdditionalInfo = map[string]string{
		additionalInfoManagedObjectInstance: "Device.FaultMgmt.ExpeditedEvent.",
		additionalInfoAdditionalInformation: "Set Clock source sync fail Alarm",
	}
	current11109 := issue227LocalAlarm(deviceID, deviceSN, "11109", "S1 Setup failure", model.AlarmMajor, time.Date(2026, 7, 30, 16, 25, 9, 0, time.Local))
	current11112 := issue227LocalAlarm(deviceID, deviceSN, "11112", "SCTP link failure", model.AlarmCritical, time.Date(2026, 7, 30, 16, 25, 9, 0, time.Local))
	for _, alarm := range []*model.Alarm{old11109, old11112, gps, clock, current11109, current11112} {
		require.NoError(t, store.SaveActive(context.Background(), alarm))
	}

	params := issue227CurrentAlarmParams()
	result := processor.processSync(context.Background(), deviceSN, params)

	assert.Equal(t, 4, result.Cleared)
	assert.Zero(t, result.FailedClear)
	require.Len(t, store.active, 2)
	assert.Contains(t, store.active, current11109.ID)
	assert.Contains(t, store.active, current11112.ID)
	require.Len(t, store.history, 4)
	assert.Equal(t, float64(2), testutil.ToFloat64(
		metrics.ReconciliationTotal.WithLabelValues("sync_duplicate_cleared"),
	))

	historyByID := make(map[uuid.UUID]*model.Alarm, len(store.history))
	for _, alarm := range store.history {
		historyByID[alarm.ID] = alarm
	}
	for _, duplicate := range []*model.Alarm{old11109, old11112} {
		archived := historyByID[duplicate.ID]
		require.NotNil(t, archived)
		require.NotNil(t, archived.ClearedBy)
		assert.Equal(t, "system", *archived.ClearedBy)
		require.NotNil(t, archived.ClearNote)
		assert.Equal(t, "duplicate active alarm reconciled by full sync", *archived.ClearNote)
	}
}

func TestProcessSyncRestoresKeeperRedisKeyAfterDuplicateClear(t *testing.T) {
	const deviceSN = "SN-SYNC-REDIS"
	ctx := context.Background()
	store := newMockAlarmStore()
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	redisStore := NewRedisAlarmStore(client)
	engine := &AlarmEngine{store: store, redisStore: redisStore, logger: zap.NewNop()}
	processor := NewAlarmSyncProcessor(engine, store, nil, nil, zap.NewNop())
	deviceID := uuid.New()
	old := issue227LocalAlarm(deviceID, deviceSN, "11109", "S1 Setup failure", model.AlarmMajor, time.Date(2026, 7, 30, 15, 55, 28, 0, time.Local))
	keeper := issue227LocalAlarm(deviceID, deviceSN, "11109", "S1 Setup failure", model.AlarmMajor, time.Date(2026, 7, 30, 16, 25, 9, 0, time.Local))
	require.NoError(t, store.SaveActive(ctx, old))
	require.NoError(t, store.SaveActive(ctx, keeper))
	require.NoError(t, redisStore.Set(ctx, deviceSN, activeAlarmMatchKey(old), old.ID.String()))

	result := processor.processSync(ctx, deviceSN, issue227CurrentAlarmParams()[:8])

	require.Equal(t, 1, result.Cleared)
	value, err := redisStore.Get(ctx, deviceSN, activeAlarmMatchKey(keeper))
	require.NoError(t, err)
	assert.Equal(t, keeper.ID.String(), value)
}

func syncTestAlarm(identifier string, raisedAt time.Time, description string) *model.Alarm {
	return &model.Alarm{
		ID:              uuid.New(),
		AlarmIdentifier: identifier,
		Description:     description,
		Severity:        model.AlarmMajor,
		RaisedAt:        raisedAt,
		CreatedAt:       raisedAt,
		AdditionalInfo: map[string]string{
			additionalInfoManagedObjectInstance: "Device.FaultMgmt.ExpeditedEvent.",
			additionalInfoAdditionalText:        "LTE0",
			additionalInfoAdditionalInformation: "LTE0(73828545);same object",
		},
	}
}

func issue227LocalAlarm(
	deviceID uuid.UUID,
	deviceSN string,
	identifier string,
	description string,
	severity model.AlarmSeverity,
	raisedAt time.Time,
) *model.Alarm {
	alarm := syncTestAlarm(identifier, raisedAt, description)
	alarm.DeviceID = deviceID
	alarm.DeviceSN = deviceSN
	alarm.Carrier = model.CarrierCMCC
	alarm.Status = model.AlarmActive
	alarm.Severity = severity
	return alarm
}

func issue227CurrentAlarmParams() []tr069.ParameterValueStruct {
	return []tr069.ParameterValueStruct{
		{Name: "Device.FaultMgmt.CurrentAlarm.3.AlarmIdentifier", Value: "11109"},
		{Name: "Device.FaultMgmt.CurrentAlarm.3.AlarmRaisedTime", Value: "2026-07-30T16:25:05+08:00"},
		{Name: "Device.FaultMgmt.CurrentAlarm.3.EventType", Value: "Communications Alarm"},
		{Name: "Device.FaultMgmt.CurrentAlarm.3.ProbableCause", Value: "S1 Setup failure"},
		{Name: "Device.FaultMgmt.CurrentAlarm.3.SpecificProblem", Value: "S1 Setup failure"},
		{Name: "Device.FaultMgmt.CurrentAlarm.3.PerceivedSeverity", Value: "Major"},
		{Name: "Device.FaultMgmt.CurrentAlarm.3.AdditionalText", Value: "LTE0"},
		{Name: "Device.FaultMgmt.CurrentAlarm.3.AdditionalInformation", Value: "LTE0(73828545);same object"},
		{Name: "Device.FaultMgmt.CurrentAlarm.4.AlarmIdentifier", Value: "11112"},
		{Name: "Device.FaultMgmt.CurrentAlarm.4.AlarmRaisedTime", Value: "2026-07-30T16:25:05+08:00"},
		{Name: "Device.FaultMgmt.CurrentAlarm.4.EventType", Value: "Communications Alarm"},
		{Name: "Device.FaultMgmt.CurrentAlarm.4.ProbableCause", Value: "SCTP link failure"},
		{Name: "Device.FaultMgmt.CurrentAlarm.4.SpecificProblem", Value: "SCTP link failure"},
		{Name: "Device.FaultMgmt.CurrentAlarm.4.PerceivedSeverity", Value: "Critical"},
		{Name: "Device.FaultMgmt.CurrentAlarm.4.AdditionalText", Value: "LTE0"},
		{Name: "Device.FaultMgmt.CurrentAlarm.4.AdditionalInformation", Value: "LTE0(73828545);same object"},
	}
}

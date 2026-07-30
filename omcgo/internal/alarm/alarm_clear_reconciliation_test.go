package alarm

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAutoClearFallsBackToUniqueIdentifierWhenQualifierChanged(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()
	existing := &model.Alarm{
		ID:              uuid.New(),
		DeviceSN:        "TEST-GPS",
		DeviceID:        uuid.New(),
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "11190",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now().Add(-5 * time.Minute),
		Status:          model.AlarmActive,
		AdditionalInfo: map[string]string{
			"managed_object_instance": "Device.FaultMgmt.ExpeditedEvent.",
			"additional_information":  "Gps unavailable, gps antenna open.",
		},
	}
	require.NoError(t, store.SaveActive(ctx, existing))

	err := engine.AutoClear(ctx, &model.Alarm{
		DeviceSN:        existing.DeviceSN,
		AlarmIdentifier: existing.AlarmIdentifier,
		AdditionalInfo: map[string]string{
			"managed_object_instance": "Device.FaultMgmt.HistoryEvent.3.",
			"additional_information":  "clear GPS Unavailable Alarm",
		},
	})

	require.NoError(t, err)
	assert.Empty(t, store.active)
	require.Len(t, store.history, 1)
	assert.Equal(t, existing.ID, store.history[0].ID)
}

func TestAutoClearDoesNotGuessWhenIdentifierHasMultipleCandidates(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()
	for _, cell := range []string{"cell=1", "cell=2"} {
		require.NoError(t, engine.Process(ctx, &model.Alarm{
			DeviceSN:        "TEST-MULTI",
			DeviceID:        uuid.New(),
			Carrier:         model.CarrierCMCC,
			AlarmIdentifier: "11184",
			Severity:        model.AlarmMajor,
			RaisedAt:        time.Now(),
			AdditionalInfo:  map[string]string{"additional_information": cell},
		}))
	}

	err := engine.AutoClear(ctx, &model.Alarm{
		DeviceSN:        "TEST-MULTI",
		AlarmIdentifier: "11184",
	})

	require.NoError(t, err)
	assert.Len(t, store.active, 2)
	assert.Empty(t, store.history)
}

func TestAutoClearDeletesStableRedisKey(t *testing.T) {
	ctx := context.Background()
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	redisStore := NewRedisAlarmStore(client)
	store := newMockAlarmStore()
	engine := &AlarmEngine{store: store, redisStore: redisStore, logger: zap.NewNop()}
	existing := &model.Alarm{
		ID:              uuid.New(),
		DeviceSN:        "TEST-REDIS-CLEAR",
		DeviceID:        uuid.New(),
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "11184",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now(),
		Status:          model.AlarmActive,
		AdditionalInfo:  map[string]string{"managed_object_instance": "Device.Radio.1"},
	}
	require.NoError(t, store.SaveActive(ctx, existing))
	require.NoError(t, redisStore.Set(ctx, existing.DeviceSN, activeAlarmMatchKey(existing), existing.ID.String()))

	require.NoError(t, engine.AutoClear(ctx, existing))

	exists, err := redisStore.Exists(ctx, existing.DeviceSN, activeAlarmMatchKey(existing))
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestClearBySyncDeletesStableRedisKey(t *testing.T) {
	ctx := context.Background()
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	redisStore := NewRedisAlarmStore(client)
	store := newMockAlarmStore()
	engine := &AlarmEngine{store: store, redisStore: redisStore, logger: zap.NewNop()}
	existing := &model.Alarm{
		ID:              uuid.New(),
		DeviceSN:        "TEST-REDIS-SYNC",
		DeviceID:        uuid.New(),
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "11184",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now(),
		Status:          model.AlarmActive,
		AdditionalInfo:  map[string]string{"managed_object_instance": "Device.Radio.1"},
	}
	require.NoError(t, store.SaveActive(ctx, existing))
	require.NoError(t, redisStore.Set(ctx, existing.DeviceSN, activeAlarmMatchKey(existing), existing.ID.String()))

	require.NoError(t, engine.ClearBySync(ctx, existing))

	exists, err := redisStore.Exists(ctx, existing.DeviceSN, activeAlarmMatchKey(existing))
	require.NoError(t, err)
	assert.False(t, exists)
}

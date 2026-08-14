package alarm

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestFilterEngine_ProcessAlarm_LiveInReceiver
//
// 验证 W2 T-0011 关键 deliverable：FilterEngine 已接进 AlarmEngine.Process 主路径。
//
// 当 AlarmEngine 配置了 FilterEngine + 命中 ignore 规则时：
//   - alarm 不应入库（store.SaveActive 不被调用）
//   - Process 返回 nil（业务流程不阻塞）
//   - dispatcher 不被触发
//
// 当无命中规则时，AlarmEngine 走默认路径正常入库。
func TestFilterEngine_ProcessAlarm_LiveInReceiver(t *testing.T) {
	t.Run("ignore_rule_short_circuits_process", func(t *testing.T) {
		store := newCountingStore()
		dispatcher := &mockDispatcher{}

		repo := &mockFilterRuleRepo{rules: []AlarmFilterRule{
			{
				FilterType:       FilterTypeAlarmIdentifier,
				AlarmIdentifiers: []string{"NOISE_ALARM"},
				Action:           FilterActionIgnore,
				Name:             "ignore-noise",
			},
		}}
		filterEngine := NewFilterEngine(repo, store, dispatcher, nil, nil, zap.NewNop())

		engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())
		engine.SetFilterEngine(filterEngine)

		alarm := &model.Alarm{
			DeviceID:        uuid.New(),
			DeviceSN:        "SN-FILTER-IGNORE",
			AlarmIdentifier: "NOISE_ALARM",
			Severity:        model.AlarmMinor,
			RaisedAt:        time.Now(),
			Carrier:         model.CarrierCMCC,
		}

		err := engine.Process(context.Background(), alarm)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), atomic.LoadInt32(&store.saveActiveCount), "ignored alarm must not be persisted")
		assert.Empty(t, dispatcher.Calls(), "ignore action does not invoke dispatcher")
	})

	t.Run("no_match_falls_through_to_default_persistence", func(t *testing.T) {
		store := newCountingStore()
		dispatcher := &mockDispatcher{}

		repo := &mockFilterRuleRepo{rules: []AlarmFilterRule{
			{
				FilterType:       FilterTypeAlarmIdentifier,
				AlarmIdentifiers: []string{"OTHER_ALARM"},
				Action:           FilterActionIgnore,
				Name:             "ignore-other",
			},
		}}
		filterEngine := NewFilterEngine(repo, store, dispatcher, nil, nil, zap.NewNop())

		engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())
		engine.SetFilterEngine(filterEngine)

		alarm := &model.Alarm{
			DeviceID:        uuid.New(),
			DeviceSN:        "SN-FILTER-PASS",
			AlarmIdentifier: "DEVICE_OFFLINE",
			Severity:        model.AlarmMajor,
			RaisedAt:        time.Now(),
			Carrier:         model.CarrierCMCC,
		}

		err := engine.Process(context.Background(), alarm)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), atomic.LoadInt32(&store.saveActiveCount), "non-matching alarm should reach persistence")
	})

	t.Run("device_group_rule_short_circuits_process", func(t *testing.T) {
		store := newCountingStore()
		dispatcher := &mockDispatcher{}
		groupID := uuid.New()

		repo := &mockFilterRuleRepo{rules: []AlarmFilterRule{
			{
				FilterType:     FilterTypeDeviceGroup,
				DeviceGroupIDs: []uuid.UUID{groupID},
				Action:         FilterActionIgnore,
				Name:           "ignore-group",
			},
		}}
		filterEngine := NewFilterEngine(repo, store, dispatcher, nil, nil, zap.NewNop())
		filterEngine.SetDeviceGroupResolver(&mockDeviceGroupResolver{groupID: &groupID})

		engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())
		engine.SetFilterEngine(filterEngine)

		alarm := &model.Alarm{
			DeviceID:        uuid.New(),
			DeviceSN:        "SN-FILTER-GROUP",
			AlarmIdentifier: "NOISE_ALARM",
			Severity:        model.AlarmMinor,
			RaisedAt:        time.Now(),
			Carrier:         model.CarrierCMCC,
		}

		err := engine.Process(context.Background(), alarm)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), atomic.LoadInt32(&store.saveActiveCount), "group-matched ignored alarm must not be persisted")
		assert.Empty(t, dispatcher.Calls())
	})

	t.Run("nil_filter_engine_keeps_default_behavior", func(t *testing.T) {
		store := newCountingStore()
		engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())
		// 故意不调 SetFilterEngine：保持向后兼容路径

		alarm := &model.Alarm{
			DeviceID:        uuid.New(),
			DeviceSN:        "SN-NO-FILTER",
			AlarmIdentifier: "DEVICE_OFFLINE",
			Severity:        model.AlarmMajor,
			RaisedAt:        time.Now(),
			Carrier:         model.CarrierCMCC,
		}

		err := engine.Process(context.Background(), alarm)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), atomic.LoadInt32(&store.saveActiveCount))
	})

	t.Run("auto_ack_rule_still_persists_active_alarm", func(t *testing.T) {
		store := newMockAlarmStore()

		repo := &mockFilterRuleRepo{rules: []AlarmFilterRule{
			{
				FilterType:       FilterTypeAlarmIdentifier,
				AlarmIdentifiers: []string{"10001"},
				DeviceIDs:        []uuid.UUID{uuid.MustParse("11111111-1111-1111-1111-111111111111")},
				Action:           FilterActionAutoAcknowledge,
				Name:             "auto-ack-device-and-alarm",
			},
		}}
		filterEngine := NewFilterEngine(repo, store, nil, nil, nil, zap.NewNop())

		engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())
		engine.SetFilterEngine(filterEngine)

		alarm := &model.Alarm{
			ID:              uuid.New(),
			DeviceID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			DeviceSN:        "SN-AUTO-ACK",
			AlarmIdentifier: "10001",
			Severity:        model.AlarmMajor,
			RaisedAt:        time.Now(),
			Carrier:         model.CarrierCMCC,
		}

		err := engine.Process(context.Background(), alarm)
		assert.NoError(t, err)

		stored, getErr := store.GetActiveByID(context.Background(), alarm.ID)
		assert.NoError(t, getErr)
		assert.Equal(t, model.AlarmAcknowledged, stored.Status)
		assert.NotNil(t, stored.AcknowledgedAt)
		assert.NotNil(t, stored.AcknowledgedBy)
	})
}

func TestFilterEngine_NotifyEmail_CoversRaiseAndClearLifecycle(t *testing.T) {
	store := newMockAlarmStore()
	emailDispatcher := &mockEmailDispatcher{}
	filterEngine := NewFilterEngine(&mockFilterRuleRepo{rules: []AlarmFilterRule{
		{
			FilterType:       FilterTypeAlarmIdentifier,
			AlarmIdentifiers: []string{"DEVICE_OFFLINE"},
			Action:           FilterActionNotifyEmail,
			EmailRecipients:  []string{"noc@omc.local"},
			Name:             "email-offline",
		},
	}}, store, nil, nil, nil, zap.NewNop())
	filterEngine.SetEmailDispatcher(emailDispatcher)
	engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())
	engine.SetFilterEngine(filterEngine)

	alarm := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-LIFECYCLE",
		AlarmIdentifier: "DEVICE_OFFLINE",
		Severity:        model.AlarmMajor,
		Status:          model.AlarmActive,
		RaisedAt:        time.Now().Add(-time.Minute),
	}
	assert.NoError(t, engine.Process(context.Background(), alarm))
	assert.NoError(t, engine.Process(context.Background(), alarm), "duplicate device report must not enqueue another raised email")
	assert.Len(t, emailDispatcher.Calls(), 1)
	assert.NoError(t, engine.Clear(context.Background(), alarm.ID))

	calls := emailDispatcher.Calls()
	assert.Len(t, calls, 2)
	assert.Contains(t, calls[0].Body, "当前状态: active")
	assert.Contains(t, calls[1].Body, "当前状态: cleared")
	assert.Contains(t, calls[1].Body, "清除时间:")
}

// countingStore 仅记录 SaveActive 调用次数，其它方法回退到 mockStoreForEngine 行为。
type countingStore struct {
	mockStoreForEngine
	saveActiveCount int32
}

func newCountingStore() *countingStore {
	return &countingStore{}
}

func (s *countingStore) SaveActive(_ context.Context, _ *model.Alarm) error {
	atomic.AddInt32(&s.saveActiveCount, 1)
	return nil
}

// 其它方法继承 mockStoreForEngine 的 zero-value 实现（GetActive* 返回 nil, nil 等）

package alarm

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockFilterRuleRepo 模拟过滤规则仓储
type mockFilterRuleRepo struct {
	rules []AlarmFilterRule
}

func (m *mockFilterRuleRepo) Create(ctx context.Context, rule *AlarmFilterRule) error                                           { return nil }
func (m *mockFilterRuleRepo) GetByID(ctx context.Context, id uuid.UUID) (*AlarmFilterRule, error)                             { return nil, nil }
func (m *mockFilterRuleRepo) Update(ctx context.Context, rule *AlarmFilterRule) error                                       { return nil }
func (m *mockFilterRuleRepo) Delete(ctx context.Context, id uuid.UUID) error                                               { return nil }
func (m *mockFilterRuleRepo) List(ctx context.Context, filter AlarmFilterRuleFilter) (*model.ListResponse[AlarmFilterRule], error) { return nil, nil }
func (m *mockFilterRuleRepo) Toggle(ctx context.Context, id uuid.UUID) error                                               { return nil }
func (m *mockFilterRuleRepo) ListEnabled(ctx context.Context) ([]AlarmFilterRule, error)                                    { return m.rules, nil }

// mockStoreForEngine 模拟告警存储
type mockStoreForEngine struct{}

func (m *mockStoreForEngine) SaveActive(ctx context.Context, alarm *model.Alarm) error       { return nil }
func (m *mockStoreForEngine) GetActiveByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error) { return nil, nil }
func (m *mockStoreForEngine) GetActiveByDeviceAndIdentifier(ctx context.Context, deviceSN string, alarmIdentifier string) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockStoreForEngine) GetActiveByDeviceSN(ctx context.Context, deviceSN string) ([]*model.Alarm, error) {
	return nil, nil
}
func (m *mockStoreForEngine) UpdateActive(ctx context.Context, alarm *model.Alarm) error          { return nil }
func (m *mockStoreForEngine) RemoveActive(ctx context.Context, id uuid.UUID) error              { return nil }
func (m *mockStoreForEngine) ListActive(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return nil, nil
}
func (m *mockStoreForEngine) Archive(ctx context.Context, alarm *model.Alarm) error              { return nil }
func (m *mockStoreForEngine) ListHistory(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return nil, nil
}
func (m *mockStoreForEngine) Statistics(ctx context.Context, filter AlarmFilter) (*AlarmStatistics, error) { return nil, nil }
func (m *mockStoreForEngine) HistoryStatistics(ctx context.Context, filter AlarmFilter) (*AlarmStatistics, error) { return nil, nil }
func (m *mockStoreForEngine) BatchAcknowledge(ctx context.Context, ids []uuid.UUID, by string, note string) error { return nil }
func (m *mockStoreForEngine) BatchUnacknowledge(ctx context.Context, ids []uuid.UUID) error                { return nil }
func (m *mockStoreForEngine) BatchClear(ctx context.Context, ids []uuid.UUID, by string, note string) error       { return nil }
func (m *mockStoreForEngine) BatchHistoryAcknowledge(ctx context.Context, ids []uuid.UUID, by string, note string) error { return nil }
func (m *mockStoreForEngine) BatchHistoryUnacknowledge(ctx context.Context, ids []uuid.UUID) error                { return nil }
func (m *mockStoreForEngine) BatchHistoryDelete(ctx context.Context, ids []uuid.UUID) error                       { return nil }
func (m *mockStoreForEngine) MarkRead(ctx context.Context, id uuid.UUID) error                                    { return nil }

func TestMatch_IgnoreAction(t *testing.T) {
	engine := NewFilterEngine(&mockFilterRuleRepo{}, &mockStoreForEngine{}, zap.NewNop())
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmSource, AlarmSources: []string{"Device"}, Action: FilterActionIgnore}
	alarm := &model.Alarm{AlarmSource: strPtr("Device"), AlarmIdentifier: "CPU_OVERLOAD"}
	assert.True(t, engine.match(alarm, uuid.UUID{}, rule))
}

func TestMatch_AutoAcknowledge(t *testing.T) {
	engine := NewFilterEngine(&mockFilterRuleRepo{}, &mockStoreForEngine{}, zap.NewNop())
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"CPU_OVERLOAD"}, Action: FilterActionAutoAcknowledge}
	alarm := &model.Alarm{AlarmIdentifier: "CPU_OVERLOAD"}
	assert.True(t, engine.match(alarm, uuid.UUID{}, rule))
}

func TestMatch_AutoClear(t *testing.T) {
	engine := NewFilterEngine(&mockFilterRuleRepo{}, &mockStoreForEngine{}, zap.NewNop())
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"TEMP_HIGH"}, Action: FilterActionAutoClear}
	alarm := &model.Alarm{AlarmIdentifier: "TEMP_HIGH"}
	assert.True(t, engine.match(alarm, uuid.UUID{}, rule))
}

func TestMatch_NoMatch(t *testing.T) {
	engine := NewFilterEngine(&mockFilterRuleRepo{}, &mockStoreForEngine{}, zap.NewNop())
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"CPU_OVERLOAD"}, Action: FilterActionIgnore}
	alarm := &model.Alarm{AlarmIdentifier: "GPS_LOSS"}
	assert.False(t, engine.match(alarm, uuid.UUID{}, rule))
}

func TestProcessAlarm_Default(t *testing.T) {
	repo := &mockFilterRuleRepo{rules: []AlarmFilterRule{}}
	engine := NewFilterEngine(repo, &mockStoreForEngine{}, zap.NewNop())
	alarm := &model.Alarm{AlarmIdentifier: "DEVICE_OFFLINE"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.False(t, result.Handled)
	assert.Equal(t, FilterActionDefault, result.Action)
}

func TestProcessAlarm_IgnoreAction(t *testing.T) {
	repo := &mockFilterRuleRepo{rules: []AlarmFilterRule{
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"DEVICE_OFFLINE"}, Action: FilterActionIgnore},
	}}
	engine := NewFilterEngine(repo, &mockStoreForEngine{}, zap.NewNop())
	alarm := &model.Alarm{AlarmIdentifier: "DEVICE_OFFLINE"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionIgnore, result.Action)
}

func TestProcessAlarm_AutoAck(t *testing.T) {
	repo := &mockFilterRuleRepo{rules: []AlarmFilterRule{
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"CPU_OVERLOAD"}, Action: FilterActionAutoAcknowledge, Name: "auto-ack-cpu"},
	}}
	engine := NewFilterEngine(repo, &mockStoreForEngine{}, zap.NewNop())
	alarm := &model.Alarm{AlarmIdentifier: "CPU_OVERLOAD"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionAutoAcknowledge, result.Action)
	assert.Equal(t, model.AlarmAcknowledged, alarm.Status)
}

func TestProcessAlarm_AutoClear(t *testing.T) {
	repo := &mockFilterRuleRepo{rules: []AlarmFilterRule{
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"GPS_LOSS"}, Action: FilterActionAutoClear, Name: "auto-clear-gps"},
	}}
	engine := NewFilterEngine(repo, &mockStoreForEngine{}, zap.NewNop())
	alarm := &model.Alarm{AlarmIdentifier: "GPS_LOSS"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionAutoClear, result.Action)
}

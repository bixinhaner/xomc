package alarm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
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

// mockDispatcher 记录所有 Dispatch 调用，用于断言。
type mockDispatcher struct {
	mu      sync.Mutex
	calls   []dispatchCall
	failNext bool
}

type dispatchCall struct {
	URL     string
	Payload []byte
}

func (m *mockDispatcher) Dispatch(ctx context.Context, url string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, dispatchCall{URL: url, Payload: append([]byte(nil), payload...)})
	if m.failNext {
		m.failNext = false
		return assert.AnError
	}
	return nil
}

func (m *mockDispatcher) Calls() []dispatchCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dispatchCall, len(m.calls))
	copy(out, m.calls)
	return out
}

func newTestFilterEngine(rules []AlarmFilterRule, dispatcher WebhookDispatcher) *FilterEngine {
	repo := &mockFilterRuleRepo{rules: rules}
	return NewFilterEngine(repo, &mockStoreForEngine{}, dispatcher, nil, zap.NewNop())
}

func TestMatch_IgnoreAction(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmSource, AlarmSources: []string{"Device"}, Action: FilterActionIgnore}
	alarm := &model.Alarm{AlarmSource: strPtr("Device"), AlarmIdentifier: "CPU_OVERLOAD"}
	assert.True(t, engine.match(alarm, uuid.UUID{}, rule))
}

func TestMatch_AutoAcknowledge(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"CPU_OVERLOAD"}, Action: FilterActionAutoAcknowledge}
	alarm := &model.Alarm{AlarmIdentifier: "CPU_OVERLOAD"}
	assert.True(t, engine.match(alarm, uuid.UUID{}, rule))
}

func TestMatch_AutoClear(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"TEMP_HIGH"}, Action: FilterActionAutoClear}
	alarm := &model.Alarm{AlarmIdentifier: "TEMP_HIGH"}
	assert.True(t, engine.match(alarm, uuid.UUID{}, rule))
}

func TestMatch_NoMatch(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"CPU_OVERLOAD"}, Action: FilterActionIgnore}
	alarm := &model.Alarm{AlarmIdentifier: "GPS_LOSS"}
	assert.False(t, engine.match(alarm, uuid.UUID{}, rule))
}

func TestProcessAlarm_Default(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	alarm := &model.Alarm{AlarmIdentifier: "DEVICE_OFFLINE"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.False(t, result.Handled)
	assert.Equal(t, FilterActionDefault, result.Action)
}

func TestProcessAlarm_IgnoreAction(t *testing.T) {
	engine := newTestFilterEngine([]AlarmFilterRule{
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"DEVICE_OFFLINE"}, Action: FilterActionIgnore},
	}, nil)
	alarm := &model.Alarm{AlarmIdentifier: "DEVICE_OFFLINE"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionIgnore, result.Action)
}

func TestProcessAlarm_AutoAck(t *testing.T) {
	engine := newTestFilterEngine([]AlarmFilterRule{
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"CPU_OVERLOAD"}, Action: FilterActionAutoAcknowledge, Name: "auto-ack-cpu"},
	}, nil)
	alarm := &model.Alarm{AlarmIdentifier: "CPU_OVERLOAD"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionAutoAcknowledge, result.Action)
	assert.Equal(t, model.AlarmAcknowledged, alarm.Status)
}

func TestProcessAlarm_AutoClear(t *testing.T) {
	engine := newTestFilterEngine([]AlarmFilterRule{
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"GPS_LOSS"}, Action: FilterActionAutoClear, Name: "auto-clear-gps"},
	}, nil)
	alarm := &model.Alarm{AlarmIdentifier: "GPS_LOSS"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionAutoClear, result.Action)
}

// W1.5：notify_webhook 命中规则后应调用 dispatcher，URL/payload 与配置一致。
func TestProcessAlarm_NotifyWebhook_Dispatched(t *testing.T) {
	url := "https://example.test/hook"
	dispatcher := &mockDispatcher{}
	engine := newTestFilterEngine([]AlarmFilterRule{
		{
			FilterType:       FilterTypeAlarmIdentifier,
			AlarmIdentifiers: []string{"CPU_OVERLOAD"},
			Action:           FilterActionNotifyWebhook,
			WebhookURL:       &url,
			Name:             "webhook-cpu",
		},
	}, dispatcher)

	alarm := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-001",
		Severity:        model.AlarmCritical,
		AlarmIdentifier: "CPU_OVERLOAD",
		AlarmSource:     strPtr("Device"),
		Status:          model.AlarmActive,
	}

	result, err := engine.ProcessAlarm(context.Background(), alarm, alarm.DeviceID)
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionNotifyWebhook, result.Action)

	calls := dispatcher.Calls()
	assert.Len(t, calls, 1, "dispatcher should be invoked once")
	assert.Equal(t, url, calls[0].URL)

	var payload map[string]interface{}
	assert.NoError(t, json.Unmarshal(calls[0].Payload, &payload))
	assert.Equal(t, "CPU_OVERLOAD", payload["alarm_identifier"])
	assert.Equal(t, "Device", payload["alarm_source"])
	assert.Equal(t, alarm.DeviceID.String(), payload["device_id"])
	assert.Equal(t, "SN-001", payload["device_sn"])
}

// W1.5 兜底：webhook_url 缺失时不应触发 dispatcher，仍返回 Handled=true。
func TestProcessAlarm_NotifyWebhook_MissingURL_Skipped(t *testing.T) {
	dispatcher := &mockDispatcher{}
	engine := newTestFilterEngine([]AlarmFilterRule{
		{
			FilterType:       FilterTypeAlarmIdentifier,
			AlarmIdentifiers: []string{"CPU_OVERLOAD"},
			Action:           FilterActionNotifyWebhook,
			WebhookURL:       nil,
			Name:             "webhook-missing",
		},
	}, dispatcher)

	alarm := &model.Alarm{AlarmIdentifier: "CPU_OVERLOAD"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.True(t, result.Handled, "missing URL still treated as handled to keep flow non-blocking")
	assert.Equal(t, FilterActionNotifyWebhook, result.Action)
	assert.Empty(t, dispatcher.Calls(), "dispatcher should not be called when URL is empty")
}

// W1.5 charter 步骤 4：与真实 HTTP 服务对接，httptest.Server 替代 webhook.site。
func TestProcessAlarm_NotifyWebhook_EndToEnd(t *testing.T) {
	received := make(chan map[string]interface{}, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var payload map[string]interface{}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		received <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	url := ts.URL
	metrics := NewWebhookMetrics(nil)
	dispatcher := NewHTTPWebhookDispatcher(zap.NewNop(), metrics)
	repo := &mockFilterRuleRepo{rules: []AlarmFilterRule{
		{
			FilterType:       FilterTypeAlarmIdentifier,
			AlarmIdentifiers: []string{"DEVICE_OFFLINE"},
			Action:           FilterActionNotifyWebhook,
			WebhookURL:       &url,
			Name:             "webhook-e2e",
		},
	}}
	engine := NewFilterEngine(repo, &mockStoreForEngine{}, dispatcher, metrics, zap.NewNop())

	alarm := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-E2E",
		Severity:        model.AlarmMajor,
		AlarmIdentifier: "DEVICE_OFFLINE",
		AlarmSource:     strPtr("Device"),
		Status:          model.AlarmActive,
	}

	result, err := engine.ProcessAlarm(context.Background(), alarm, alarm.DeviceID)
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionNotifyWebhook, result.Action)

	select {
	case payload := <-received:
		assert.Equal(t, "DEVICE_OFFLINE", payload["alarm_identifier"])
		assert.Equal(t, "Device", payload["alarm_source"])
	default:
		t.Fatal("webhook server did not receive POST")
	}
}

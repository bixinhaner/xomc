package alarm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
func (m *mockStoreForEngine) GetHistoryByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error) { return nil, nil }
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
	mu       sync.Mutex
	calls    []dispatchCall
	failNext bool
	failErr  error // 显式指定失败时返回的错误（用于测 ErrDeadLetter 路径）
}

type dispatchCall struct {
	URL     string
	Secret  string
	Payload []byte
}

type mockDeviceGroupResolver struct {
	groupID *uuid.UUID
	err     error
}

func (m *mockDeviceGroupResolver) GetGroupID(_ context.Context, _ uuid.UUID) (*uuid.UUID, error) {
	return m.groupID, m.err
}

func (m *mockDispatcher) Dispatch(ctx context.Context, url, secret string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, dispatchCall{URL: url, Secret: secret, Payload: append([]byte(nil), payload...)})
	if m.failNext {
		m.failNext = false
		if m.failErr != nil {
			return m.failErr
		}
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
	return NewFilterEngine(repo, &mockStoreForEngine{}, dispatcher, nil, nil, zap.NewNop())
}

func TestMatch_IgnoreAction(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmSource, AlarmSources: []string{"Device"}, Action: FilterActionIgnore}
	alarm := &model.Alarm{AlarmSource: strPtr("Device"), AlarmIdentifier: "CPU_OVERLOAD"}
	assert.True(t, engine.match(context.Background(), alarm, uuid.UUID{}, rule))
}

func TestMatch_AutoAcknowledge(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"CPU_OVERLOAD"}, Action: FilterActionAutoAcknowledge}
	alarm := &model.Alarm{AlarmIdentifier: "CPU_OVERLOAD"}
	assert.True(t, engine.match(context.Background(), alarm, uuid.UUID{}, rule))
}

func TestMatch_AutoClear(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"TEMP_HIGH"}, Action: FilterActionAutoClear}
	alarm := &model.Alarm{AlarmIdentifier: "TEMP_HIGH"}
	assert.True(t, engine.match(context.Background(), alarm, uuid.UUID{}, rule))
}

func TestMatch_NoMatch(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	rule := &AlarmFilterRule{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"CPU_OVERLOAD"}, Action: FilterActionIgnore}
	alarm := &model.Alarm{AlarmIdentifier: "GPS_LOSS"}
	assert.False(t, engine.match(context.Background(), alarm, uuid.UUID{}, rule))
}

func TestMatch_MultipleDimensionsRequireAll(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	matchedDeviceID := uuid.New()
	rule := &AlarmFilterRule{
		FilterType:       FilterTypeAlarmIdentifier,
		AlarmIdentifiers: []string{"10001"},
		DeviceIDs:        []uuid.UUID{matchedDeviceID},
		Action:           FilterActionAutoAcknowledge,
	}

	assert.True(t, engine.match(context.Background(), &model.Alarm{AlarmIdentifier: "10001"}, matchedDeviceID, rule))
	assert.False(t, engine.match(context.Background(), &model.Alarm{AlarmIdentifier: "10001"}, uuid.New(), rule))
	assert.False(t, engine.match(context.Background(), &model.Alarm{AlarmIdentifier: "20002"}, matchedDeviceID, rule))
}

func TestMatch_DeviceGroup(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	groupID := uuid.New()
	engine.SetDeviceGroupResolver(&mockDeviceGroupResolver{groupID: &groupID})
	rule := &AlarmFilterRule{FilterType: FilterTypeDeviceGroup, DeviceGroupIDs: []uuid.UUID{groupID}, Action: FilterActionIgnore}
	assert.True(t, engine.match(context.Background(), &model.Alarm{}, uuid.New(), rule))
}

func TestMatch_DeviceGroupWithoutResolverFailsClosed(t *testing.T) {
	engine := newTestFilterEngine(nil, nil)
	rule := &AlarmFilterRule{FilterType: FilterTypeDeviceGroup, DeviceGroupIDs: []uuid.UUID{uuid.New()}, Action: FilterActionIgnore}
	assert.False(t, engine.match(context.Background(), &model.Alarm{}, uuid.New(), rule))
}

func TestProcessAlarm_DeviceGroup_IgnoreAction(t *testing.T) {
	groupID := uuid.New()
	engine := newTestFilterEngine([]AlarmFilterRule{
		{FilterType: FilterTypeDeviceGroup, DeviceGroupIDs: []uuid.UUID{groupID}, Action: FilterActionIgnore, Name: "ignore-group"},
	}, nil)
	engine.SetDeviceGroupResolver(&mockDeviceGroupResolver{groupID: &groupID})

	alarm := &model.Alarm{AlarmIdentifier: "DEVICE_OFFLINE"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.New())
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionIgnore, result.Action)
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
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"CPU_OVERLOAD"}, Action: FilterActionAutoAcknowledge, Name: "auto-ack-cpu", AcknowledgeDesc: "acknowledged by alarm rule"},
	}, nil)
	alarm := &model.Alarm{AlarmIdentifier: "CPU_OVERLOAD"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionAutoAcknowledge, result.Action)
	assert.Equal(t, model.AlarmAcknowledged, alarm.Status)
	require.NotNil(t, alarm.AckNote)
	assert.Equal(t, "acknowledged by alarm rule", *alarm.AckNote)
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
	require.NotNil(t, alarm.ClearedBy)
	assert.Equal(t, "system", *alarm.ClearedBy)
	require.NotNil(t, alarm.ClearNote)
	assert.Equal(t, "auto-cleared by alarm rule: auto-clear-gps", *alarm.ClearNote)
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

// ------------ W2.A.1 / T-0007 整合：notify_email 测试 ------------

// mockEmailDispatcher 记录所有 Dispatch 调用，用于断言。
type mockEmailDispatcher struct {
	mu    sync.Mutex
	calls []emailCall
}

type emailCall struct {
	To      []string
	Subject string
	Body    string
}

func (m *mockEmailDispatcher) Dispatch(_ context.Context, to []string, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, emailCall{To: append([]string(nil), to...), Subject: subject, Body: body})
	return nil
}

func (m *mockEmailDispatcher) Calls() []emailCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]emailCall, len(m.calls))
	copy(out, m.calls)
	return out
}

// W2.A.1：notify_email 命中规则后应调用 emailDispatcher，收件人/Subject/Body 与配置一致。
func TestProcessAlarm_NotifyEmail_Dispatched(t *testing.T) {
	dispatcher := &mockEmailDispatcher{}
	engine := newTestFilterEngine([]AlarmFilterRule{
		{
			FilterType:       FilterTypeAlarmIdentifier,
			AlarmIdentifiers: []string{"DEVICE_OFFLINE"},
			Action:           FilterActionNotifyEmail,
			EmailRecipients:  []string{"oncall@omc.local", "ops@omc.local"},
			Name:             "email-offline",
		},
	}, nil)
	engine.SetEmailDispatcher(dispatcher)

	alarm := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-OFFLINE",
		Severity:        model.AlarmCritical,
		AlarmIdentifier: "DEVICE_OFFLINE",
		AlarmSource:     strPtr("Device"),
		Status:          model.AlarmActive,
		RaisedAt:        time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
	}

	result, err := engine.ProcessAlarm(context.Background(), alarm, alarm.DeviceID)
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionNotifyEmail, result.Action)

	calls := dispatcher.Calls()
	assert.Len(t, calls, 1, "email dispatcher should be invoked once")
	assert.Equal(t, []string{"oncall@omc.local", "ops@omc.local"}, calls[0].To)
	assert.Contains(t, calls[0].Subject, "DEVICE_OFFLINE")
	assert.Contains(t, calls[0].Subject, "SN-OFFLINE")
	assert.Contains(t, calls[0].Body, "DEVICE_OFFLINE")
	assert.Contains(t, calls[0].Body, "Device")
	assert.Contains(t, calls[0].Body, "SN-OFFLINE")
}

// W2.A.1 兜底：收件人列表为空时，dispatcher 不被调用，rule 仍记 Handled=true。
func TestProcessAlarm_NotifyEmail_MissingRecipients_Skipped(t *testing.T) {
	dispatcher := &mockEmailDispatcher{}
	engine := newTestFilterEngine([]AlarmFilterRule{
		{
			FilterType:       FilterTypeAlarmIdentifier,
			AlarmIdentifiers: []string{"CPU_OVERLOAD"},
			Action:           FilterActionNotifyEmail,
			EmailRecipients:  nil,
			Name:             "email-missing",
		},
	}, nil)
	engine.SetEmailDispatcher(dispatcher)

	alarm := &model.Alarm{AlarmIdentifier: "CPU_OVERLOAD"}
	result, err := engine.ProcessAlarm(context.Background(), alarm, uuid.UUID{})
	assert.NoError(t, err)
	assert.True(t, result.Handled, "missing recipients still treated as handled")
	assert.Equal(t, FilterActionNotifyEmail, result.Action)
	assert.Empty(t, dispatcher.Calls(), "dispatcher should not be called when recipients empty")
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
	engine := NewFilterEngine(repo, &mockStoreForEngine{}, dispatcher, nil, metrics, zap.NewNop())

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

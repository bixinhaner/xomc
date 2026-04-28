package alarm

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockDeadLetterRepo 用于断言 dead-letter 是否被正确插入。
type mockDeadLetterRepo struct {
	mu      sync.Mutex
	records []DeadLetterRecord
	failOn  bool
}

func (m *mockDeadLetterRepo) Insert(_ context.Context, rec *DeadLetterRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failOn {
		return assert.AnError
	}
	m.records = append(m.records, *rec)
	return nil
}

func (m *mockDeadLetterRepo) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.records)
}

func (m *mockDeadLetterRepo) Snapshot() []DeadLetterRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]DeadLetterRecord, len(m.records))
	copy(out, m.records)
	return out
}

// TestWebhookDispatcher_Retry_ExponentialBackoff
//
// httptest 前两次返回 503，第三次返回 200。dispatcher 应在重试中成功并返回 nil。
// 用于验证 5xx -> retry 路径与最终成功的路径，且 retry 计数 >= 1。
func TestWebhookDispatcher_Retry_ExponentialBackoff(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	reg := prometheus.NewRegistry()
	metrics := NewWebhookMetrics(reg)
	d := NewHTTPWebhookDispatcher(zap.NewNop(), metrics)
	d.client.Timeout = 1 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := d.Dispatch(ctx, ts.URL, "", []byte(`{}`))
	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls), "dispatcher should retry until 200")

	assert.Equal(t, 1.0, counterValue(t, metrics.DispatchTotal.WithLabelValues("success")))
	assert.GreaterOrEqual(t, counterValue(t, metrics.DispatchTotal.WithLabelValues("retry")), 2.0,
		"retry counter should reflect at least 2 retries")
	assert.Equal(t, 0.0, counterValue(t, metrics.DispatchTotal.WithLabelValues("dead_letter")))
}

// TestWebhookDispatcher_DeadLetter_AfterMaxRetries
//
// httptest 持续 503，dispatcher 重试耗尽返回 ErrDeadLetter；
// FilterEngine 通过 mock DeadLetterRepository.Insert 被调用一次。
func TestWebhookDispatcher_DeadLetter_AfterMaxRetries(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	reg := prometheus.NewRegistry()
	metrics := NewWebhookMetrics(reg)
	dispatcher := NewHTTPWebhookDispatcher(zap.NewNop(), metrics)
	dispatcher.client.Timeout = 500 * time.Millisecond

	dlRepo := &mockDeadLetterRepo{}
	url := ts.URL
	ruleID := uuid.New()
	repo := &mockFilterRuleRepo{rules: []AlarmFilterRule{
		{
			ID:               ruleID,
			FilterType:       FilterTypeAlarmIdentifier,
			AlarmIdentifiers: []string{"DEVICE_OFFLINE"},
			Action:           FilterActionNotifyWebhook,
			WebhookURL:       &url,
			Name:             "webhook-dl",
		},
	}}
	engine := NewFilterEngine(repo, &mockStoreForEngine{}, dispatcher, dlRepo, metrics, zap.NewNop())

	alarm := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-DL",
		Severity:        model.AlarmCritical,
		AlarmIdentifier: "DEVICE_OFFLINE",
		AlarmSource:     strPtr("Device"),
		Status:          model.AlarmActive,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.ProcessAlarm(ctx, alarm, alarm.DeviceID)
	assert.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, FilterActionNotifyWebhook, result.Action)

	// 应该尝试 4 次（1 首发 + 3 重试）
	assert.Equal(t, int32(webhookRetryMaxRetry+1), atomic.LoadInt32(&calls))
	assert.Equal(t, 1, dlRepo.Calls(), "dead-letter repo should be called exactly once")

	rec := dlRepo.Snapshot()[0]
	assert.Equal(t, ruleID, rec.FilterID)
	assert.Equal(t, alarm.ID, rec.AlarmID)
	assert.Contains(t, rec.LastError, "exhausted retries")
	assert.GreaterOrEqual(t, counterValue(t, metrics.DispatchTotal.WithLabelValues("dead_letter")), 1.0)
}

// TestWebhookDispatcher_HMACSignature
//
// 验证：当 secret 非空时 dispatcher 在 X-OMC-Signature 头中携带 HMAC-SHA256 签名（hex），
// 服务端可重新计算并匹配；secret 为空时不带签名头。
func TestWebhookDispatcher_HMACSignature(t *testing.T) {
	const secret = "super-secret-key-2026"
	payload := []byte(`{"alarm_identifier":"DEVICE_OFFLINE"}`)

	expected := hmac.New(sha256.New, []byte(secret))
	expected.Write(payload)
	wantSig := "sha256=" + hex.EncodeToString(expected.Sum(nil))

	var (
		gotSig    string
		gotMethod string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotSig = r.Header.Get(webhookSignatureHdr)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	d := NewHTTPWebhookDispatcher(zap.NewNop(), nil)

	// 1. secret 非空 → 携带正确的签名头
	err := d.Dispatch(context.Background(), ts.URL, secret, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, wantSig, gotSig)
	assert.True(t, strings.HasPrefix(gotSig, "sha256="))

	// 2. secret 为空 → 不带签名头
	gotSig = ""
	err = d.Dispatch(context.Background(), ts.URL, "", payload)
	assert.NoError(t, err)
	assert.Empty(t, gotSig, "no signature header expected when secret is empty")
}

// TestWebhookDispatcher_DoNotRetry_On4xx
//
// 4xx 业务错（404/401/400）应立刻返回 ErrDeadLetter，不重试。
func TestWebhookDispatcher_DoNotRetry_On4xx(t *testing.T) {
	cases := []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound}
	for _, code := range cases {
		t.Run(http.StatusText(code), func(t *testing.T) {
			var calls int32
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				atomic.AddInt32(&calls, 1)
				w.WriteHeader(code)
			}))
			defer ts.Close()

			reg := prometheus.NewRegistry()
			metrics := NewWebhookMetrics(reg)
			d := NewHTTPWebhookDispatcher(zap.NewNop(), metrics)

			err := d.Dispatch(context.Background(), ts.URL, "", []byte(`{}`))
			assert.ErrorIs(t, err, ErrDeadLetter)
			assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "4xx should not be retried")
			assert.Equal(t, 1.0, counterValue(t, metrics.DispatchTotal.WithLabelValues("dead_letter")))
			assert.Equal(t, 0.0, counterValue(t, metrics.DispatchTotal.WithLabelValues("retry")))
		})
	}
}

package acs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// recordingConnReqSender 记录 postSessionWake 发出的 Connection Request，供断言。
type recordingConnReqSender struct {
	mu    sync.Mutex
	calls []string // device serials woken
}

func (s *recordingConnReqSender) Send(_ context.Context, deviceSN, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, deviceSN)
	return nil
}

func (s *recordingConnReqSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// Inform XML for a fixed SN, used by both "instances" — the device serial is what
// drives cross-instance orphan detection.
const acsHInformXInstXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header><cwmp:ID soap:mustUnderstand="1">XI-1</cwmp:ID></soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>TestVendor</Manufacturer>
        <OUI>001122</OUI>
        <ProductClass>SmallCell-LTE</ProductClass>
        <SerialNumber>SN-XINST-HANDLER</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[1]">
        <EventStruct><EventCode>2 PERIODIC</EventCode><CommandKey></CommandKey></EventStruct>
      </Event>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>2026-03-05T10:00:00Z</CurrentTime>
      <RetryCount>0</RetryCount>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[0]"></ParameterList>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>`

func newXInstHandler(t *testing.T, rdb redis.UniversalClient, taskSvc TaskService, admission AdmissionController,
	devStore DeviceSessionStore, sender ConnectionRequester) *Handler {
	t.Helper()
	reg := prometheus.NewRegistry()
	return &Handler{
		sessionStore:       NewRedisSessionStore(rdb, 5*time.Minute),
		taskService:        taskSvc,
		eventBus:           &acsHEventBus{},
		authenticator:      &auth.NoopAuthenticator{},
		rpcDispatcher:      rpc.NewDispatcher(),
		rateLimiter:        NewDeviceRateLimiter(1000, 1000, 10000, zap.NewNop()),
		admission:          admission,
		deviceSessionStore: devStore,
		metrics:            NewACSMetrics(reg),
		logger:             zap.NewNop(),
		connReqSender:      sender,
		postSessionWakeCfg: appconfig.PostSessionWakeConfig{
			Enabled:       true,
			DelayAfter:    10 * time.Millisecond,
			MaxContinuous: 100,
			CooldownTTL:   time.Minute,
		},
		redisClient: rdb,
	}
}

// TestHandler_CrossInstance_OrphanCleanedOnNewInform 是 issue #65 的端到端场景：
// 实例 A 收到 Inform 起会话但不 complete（CPE 失联）；实例 B 收到同 SN 的新 Inform →
//   - A 的旧会话在共享 Redis SessionStore 中被清除；
//   - A 占用的全局准入槽位被释放（计数仅剩 B 的 1 个）；
//   - 因队列仍有待执行命令 → 触发 postSessionWake 发出 Connection Request。
func TestHandler_CrossInstance_OrphanCleanedOnNewInform(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	const deviceSN = "SN-XINST-HANDLER"

	// 共享的全局准入控制器 + 设备会话指针存储 + 任务服务 + 唤醒记录器。
	admission := NewRedisAdmissionController(rdb, 1000, zap.NewNop())
	devStore := NewRedisDeviceSessionStore(rdb, 10*time.Minute)
	sender := &recordingConnReqSender{}

	// 共享任务服务，预置一条 pending 任务 —— 使孤儿清理后的 postSessionWake 满足"队列非空"。
	taskSvc := newAcsHTaskService()
	taskSvc.addTask(&task.Task{
		ID:        "pending-after-orphan",
		DeviceSN:  deviceSN,
		Method:    "GetParameterValues",
		Status:    task.TaskStatusPending,
		CreatedAt: time.Now(),
	})

	instanceA := newXInstHandler(t, rdb, taskSvc, admission, devStore, sender)
	instanceB := newXInstHandler(t, rdb, taskSvc, admission, devStore, sender)

	// 1) 实例 A 收到 Inform → 起会话，占 1 个全局槽位，写设备指针。
	wA := httptest.NewRecorder()
	rA := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformXInstXML))
	rA.RemoteAddr = "10.0.0.1:1000"
	instanceA.handleInform(wA, rA, []byte(acsHInformXInstXML), zap.NewNop())
	require.Equal(t, http.StatusOK, wA.Code)

	ctx := context.Background()
	assert.Equal(t, int64(1), admission.Current(ctx), "A holds one global admission slot")
	aSession, _ := devStore.Get(ctx, deviceSN)
	require.NotEmpty(t, aSession, "A registered the device session pointer")

	// 2) 实例 B 收到同 SN 的新 Inform → 读到 A 的孤儿会话并跨实例清理。
	wB := httptest.NewRecorder()
	rB := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformXInstXML))
	rB.RemoteAddr = "10.0.0.2:2000"
	instanceB.handleInform(wB, rB, []byte(acsHInformXInstXML), zap.NewNop())
	require.Equal(t, http.StatusOK, wB.Code)

	// A 的旧会话已从共享 SessionStore 删除（completeSession 跑过）。
	oldSess, err := instanceB.sessionStore.GetByID(ctx, aSession)
	require.NoError(t, err)
	assert.Nil(t, oldSess, "instance A's orphaned session must be deleted cross-instance")

	// 全局准入槽位：A 的被释放，只剩 B 的 1 个（不是泄漏的 2 个）。
	assert.Equal(t, int64(1), admission.Current(ctx), "orphan slot released, only B's slot remains")

	// 设备指针现在指向 B 的会话。
	bSession, _ := devStore.Get(ctx, deviceSN)
	assert.NotEqual(t, aSession, bSession, "device pointer now points at B's session")

	// postSessionWake 在 goroutine 中异步触发（队列非空）→ 等待记录到 CR。
	require.Eventually(t, func() bool { return sender.count() >= 1 }, 2*time.Second, 20*time.Millisecond,
		"orphan cleanup must trigger postSessionWake when queue is non-empty")
	assert.Equal(t, deviceSN, sender.calls[0])
}

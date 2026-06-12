package acs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/rpclog"
	"github.com/omcgo/omcgo/internal/trace"
)

// stubTraceSink 记录所有 EnqueueCapture 调用，供断言使用。
type stubTraceSink struct {
	mu       sync.Mutex
	captured []*trace.Message
}

func (s *stubTraceSink) EnqueueCapture(m *trace.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.captured = append(s.captured, m)
}

func (s *stubTraceSink) ObserveCaptureLatency(_ time.Duration) {}

func (s *stubTraceSink) snapshot() []*trace.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*trace.Message, len(s.captured))
	copy(out, s.captured)
	return out
}

// 测试用 SOAP fixtures —— 直接覆盖 trace_capture 现场会遇到的几种典型 XML 形态。
const (
	fixtureInformXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header><cwmp:ID soap:mustUnderstand="1">cwmp-1</cwmp:ID></soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>OEM</Manufacturer>
        <OUI>001122</OUI>
        <ProductClass>SmallCell</ProductClass>
        <SerialNumber>SN-TEST</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[0]"/>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[0]"/>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>`

	fixtureInformResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header><cwmp:ID soap:mustUnderstand="1">cwmp-1</cwmp:ID></soap:Header>
  <soap:Body>
    <cwmp:InformResponse><MaxEnvelopes>1</MaxEnvelopes></cwmp:InformResponse>
  </soap:Body>
</soap:Envelope>`

	fixtureGPVXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header><cwmp:ID soap:mustUnderstand="1">cwmp-2</cwmp:ID></soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames soap:arrayType="xsd:string[1]">
        <string>Device.DeviceInfo.SoftwareVersion</string>
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>`

	fixtureGPVResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header><cwmp:ID soap:mustUnderstand="1">cwmp-2</cwmp:ID></soap:Header>
  <soap:Body>
    <cwmp:GetParameterValuesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[1]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value>FW-2.1.0</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:GetParameterValuesResponse>
  </soap:Body>
</soap:Envelope>`
)

// newTraceTestHandler 构造一个仅初始化 trace 相关字段的 Handler，足以覆盖 maybeCaptureTrace 的判断分支。
func newTraceTestHandler(t *testing.T, deviceSN string, taskID uuid.UUID, sink *stubTraceSink) *Handler {
	t.Helper()
	wl := trace.NewWhitelistCache(nil, trace.DefaultWhitelistConfig(), zap.NewNop())
	wl.Add(deviceSN, taskID)
	return &Handler{
		traceWhitelist: wl,
		traceService:   sink,
	}
}

// TestMaybeCaptureTrace_InformPair：CPE 发 Inform，ACS 回 InformResponse。
// in 必须为 Inform、out 必须为 InformResponse，不能共用同一个方法名。
func TestMaybeCaptureTrace_InformPair(t *testing.T) {
	sink := &stubTraceSink{}
	deviceSN := "SN-INFORM"
	taskID := uuid.New()
	h := newTraceTestHandler(t, deviceSN, taskID, sink)

	entry := &rpclog.LogEntry{
		DeviceSN:  deviceSN,
		Method:    "Inform", // 模拟 handler 流转：先写 Inform；OUT 方向解析应忽略此字段
		SessionID: "sess-1",
		CwmpID:    "cwmp-1",
	}
	h.maybeCaptureTrace(entry, 200, fixtureInformXML, fixtureInformResponseXML)

	msgs := sink.snapshot()
	require.Len(t, msgs, 2, "每次 HTTP 事务应产生 2 条 trace 消息")

	assert.Equal(t, trace.DirectionIn, msgs[0].Direction)
	assert.Equal(t, "Inform", msgs[0].RPCMethod)
	assert.Equal(t, len(fixtureInformXML), msgs[0].PayloadSizeBytes)
	assert.Equal(t, "cwmp-1", msgs[0].CwmpID)

	assert.Equal(t, trace.DirectionOut, msgs[1].Direction)
	assert.Equal(t, "InformResponse", msgs[1].RPCMethod)
	assert.Equal(t, len(fixtureInformResponseXML), msgs[1].PayloadSizeBytes)
	assert.Equal(t, "cwmp-1", msgs[1].CwmpID)
}

// TestMaybeCaptureTrace_GPVPair_MidSession：会话中段，CPE 发 GPVResponse，ACS 同事务下发新一轮 GPV。
// 这是原 bug 的核心场景 —— 共用 entry.Method 会把两条都标成 GetParameterValues。
func TestMaybeCaptureTrace_GPVPair_MidSession(t *testing.T) {
	sink := &stubTraceSink{}
	deviceSN := "SN-MID"
	taskID := uuid.New()
	h := newTraceTestHandler(t, deviceSN, taskID, sink)

	entry := &rpclog.LogEntry{
		DeviceSN: deviceSN,
		// 模拟 handler.go 实际行为：先在 handleRPCResponse 写 GPVResponse，再在派发新任务时被覆盖
		Method:    "GetParameterValues",
		SessionID: "sess-2",
		CwmpID:    "cwmp-2",
	}
	h.maybeCaptureTrace(entry, 200, fixtureGPVResponseXML, fixtureGPVXML)

	msgs := sink.snapshot()
	require.Len(t, msgs, 2)

	assert.Equal(t, trace.DirectionIn, msgs[0].Direction)
	assert.Equal(t, "GetParameterValuesResponse", msgs[0].RPCMethod,
		"in 必须从请求体 XML 解析，不能受 entry.Method 被覆盖影响")

	assert.Equal(t, trace.DirectionOut, msgs[1].Direction)
	assert.Equal(t, "GetParameterValues", msgs[1].RPCMethod,
		"out 必须从响应体 XML 解析")
}

// TestMaybeCaptureTrace_SessionTail：会话尾，CPE 发 GPVResponse，ACS 没有更多任务 → 204 NoContent。
// in 有内容，out 为空 body —— 空消息也要入库，方法名标为 "Empty"（区分协议空 body vs 解析失败）。
func TestMaybeCaptureTrace_SessionTail(t *testing.T) {
	sink := &stubTraceSink{}
	deviceSN := "SN-TAIL"
	taskID := uuid.New()
	h := newTraceTestHandler(t, deviceSN, taskID, sink)

	entry := &rpclog.LogEntry{
		DeviceSN:  deviceSN,
		Method:    "GetParameterValuesResponse",
		SessionID: "sess-3",
		CwmpID:    "cwmp-2",
	}
	h.maybeCaptureTrace(entry, 204, fixtureGPVResponseXML, "")

	msgs := sink.snapshot()
	require.Len(t, msgs, 2, "空 body 也要入库，便于在 trace 视图上看到完整事务边界")

	assert.Equal(t, trace.DirectionIn, msgs[0].Direction)
	assert.Equal(t, "GetParameterValuesResponse", msgs[0].RPCMethod)

	assert.Equal(t, trace.DirectionOut, msgs[1].Direction)
	assert.Equal(t, "Empty", msgs[1].RPCMethod, "协议空 body 标记为 Empty，与解析失败的空字符串区分")
	assert.Equal(t, 0, msgs[1].PayloadSizeBytes)
	assert.Equal(t, 204, msgs[1].HTTPStatus)
}

// TestMaybeCaptureTrace_FirstDispatch：会话开头，CPE 空 POST（续会话占位），ACS 下发首个 GPV。
// in 空、out 非空。
func TestMaybeCaptureTrace_FirstDispatch(t *testing.T) {
	sink := &stubTraceSink{}
	deviceSN := "SN-FIRST"
	taskID := uuid.New()
	h := newTraceTestHandler(t, deviceSN, taskID, sink)

	entry := &rpclog.LogEntry{
		DeviceSN:  deviceSN,
		Method:    "Empty", // handler.go:536 给空 POST 的标签；out 解析应忽略
		SessionID: "sess-4",
		CwmpID:    "cwmp-2",
	}
	h.maybeCaptureTrace(entry, 200, "", fixtureGPVXML)

	msgs := sink.snapshot()
	require.Len(t, msgs, 2)

	assert.Equal(t, trace.DirectionIn, msgs[0].Direction)
	assert.Equal(t, "Empty", msgs[0].RPCMethod, "CPE 续会话的空 POST 标记为 Empty")
	assert.Equal(t, 0, msgs[0].PayloadSizeBytes)

	assert.Equal(t, trace.DirectionOut, msgs[1].Direction)
	assert.Equal(t, "GetParameterValues", msgs[1].RPCMethod)
}

// fixtureSPVWithPasswordXML：CPE 上报含密码的 SetParameterValues / GPVResponse。
// 模拟 issue #7 的核心风险：协议报文里携带 CPE 明文密码，落 trace 库必须脱敏。
const fixtureSPVWithPasswordXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header><cwmp:ID soap:mustUnderstand="1">cwmp-9</cwmp:ID></soap:Header>
  <soap:Body>
    <cwmp:SetParameterValues>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[2]">
        <ParameterValueStruct>
          <Name>Device.ManagementServer.Password</Name>
          <Value>SuperSecretCpePassword</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.ManagementServer.Username</Name>
          <Value>acs-user</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:SetParameterValues>
  </soap:Body>
</soap:Envelope>`

// TestMaybeCaptureTrace_RedactsCredentials：落盘前 trace 报文里的 CPE 凭证必须脱敏，
// 同时 PayloadSizeBytes 仍记录原始报文长度（用于诊断报文体积，不受脱敏影响）。
func TestMaybeCaptureTrace_RedactsCredentials(t *testing.T) {
	sink := &stubTraceSink{}
	deviceSN := "SN-CRED"
	taskID := uuid.New()
	h := newTraceTestHandler(t, deviceSN, taskID, sink)

	entry := &rpclog.LogEntry{
		DeviceSN:  deviceSN,
		Method:    "SetParameterValues",
		SessionID: "sess-cred",
		CwmpID:    "cwmp-9",
	}
	h.maybeCaptureTrace(entry, 200, fixtureSPVWithPasswordXML, fixtureInformResponseXML)

	msgs := sink.snapshot()
	require.Len(t, msgs, 2)

	in := msgs[0]
	assert.Equal(t, trace.DirectionIn, in.Direction)
	assert.NotContains(t, in.PayloadInline, "SuperSecretCpePassword",
		"CPE 明文密码必须在落盘前脱敏（等保合规）")
	assert.Contains(t, in.PayloadInline, "acs-user",
		"非敏感参数（用户名）应原样保留，便于诊断")
	assert.Equal(t, len(fixtureSPVWithPasswordXML), in.PayloadSizeBytes,
		"PayloadSizeBytes 记录原始报文长度，脱敏不应改变它")
}

// TestMaybeCaptureTrace_NotInWhitelist：设备不在白名单，不应产生任何消息。
func TestMaybeCaptureTrace_NotInWhitelist(t *testing.T) {
	sink := &stubTraceSink{}
	wl := trace.NewWhitelistCache(nil, trace.DefaultWhitelistConfig(), zap.NewNop())
	// 不调用 Add → 任何设备查询都未命中
	h := &Handler{traceWhitelist: wl, traceService: sink}

	entry := &rpclog.LogEntry{DeviceSN: "SN-NOT-LISTED", Method: "Inform"}
	h.maybeCaptureTrace(entry, 200, fixtureInformXML, fixtureInformResponseXML)

	assert.Empty(t, sink.snapshot())
}

// ---------------------------------------------------------------------------
// issue #186 端到端被动抓回归：起跟踪后，设备自发周期 Inform（无主动呼叫、
// 甚至会话被准入拒绝）也必须落进跟踪结果。
// ---------------------------------------------------------------------------

// newPassiveCaptureHandler 构造一个能跑 ServeHTTP 全链路的 handler，并把指定 SN 加入
// 跟踪白名单、挂上 protocolLogger（capture 旁路只在 protocolLogger != nil 时触发）。
// admissionSlots=1 且预占 1 槽 → 周期 Inform 会在 line 402 准入拒绝提前返回，
// 用来证明被动抓不依赖会话是否建成。
func newPassiveCaptureHandler(t *testing.T, sink *stubTraceSink, whitelistSN string, admissionSlots int64) *Handler {
	t.Helper()
	h := newTestACSHandlerWithDeps(newAcsHSessionStore(), &acsHEventBus{})
	wl := trace.NewWhitelistCache(nil, trace.DefaultWhitelistConfig(), zap.NewNop())
	if whitelistSN != "" {
		wl.Add(whitelistSN, uuid.New())
	}
	h.traceWhitelist = wl
	h.traceService = sink
	h.protocolLogger = zap.NewNop() // 必须非 nil，否则 capture 旁路 defer 不注册
	h.admission = NewAdmissionController(admissionSlots)
	return h
}

// acsHInformPeriodicSN 是 acsHInformPeriodicXML 里的 SerialNumber，复用以保持单一事实源。
const acsHInformPeriodicSN = "TEST-SN-002"

// TestServeHTTP_PassiveCapture_PeriodicInform_CapturedEvenWhenAdmissionDenied：
// 死判 passive-capture 的回归 —— 设备在跟踪窗口内自发周期 Inform，即便准入已满、
// 会话根本没建成（403/503 提前返回），网管收到的这条 Inform 仍必须被捕获。
// 这把"是否抓到报文"与"主动呼叫/会话是否建成"解耦（issue #186 核心修复）。
func TestServeHTTP_PassiveCapture_PeriodicInform_CapturedEvenWhenAdmissionDenied(t *testing.T) {
	sink := &stubTraceSink{}
	h := newPassiveCaptureHandler(t, sink, acsHInformPeriodicSN, 1)
	// 预占满唯一准入槽 → 后续周期 Inform 必被准入拒绝，handleInform 在建会话前提前返回。
	require.True(t, h.admission.Acquire(context.Background(), "pre-occupied-session"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformPeriodicXML))
	req.RemoteAddr = "10.0.0.9:5000"
	h.ServeHTTP(w, req)

	// 准入被拒 → 503，确认确实走了"会话没建成"的提前返回路径。
	require.Equal(t, http.StatusServiceUnavailable, w.Code,
		"准入已满，周期 Inform 应被拒绝（证明走的是无会话的提前返回路径）")

	msgs := sink.snapshot()
	require.NotEmpty(t, msgs, "起跟踪后设备自发周期 Inform 即便会话没建成也必须被捕获（issue #186）")

	// 至少有一条 IN 方向且 RPC 方法为 Inform。
	var capturedInform bool
	for _, m := range msgs {
		if m.Direction == trace.DirectionIn && m.RPCMethod == "Inform" {
			capturedInform = true
			assert.Equal(t, acsHInformPeriodicSN, m.DeviceSN)
		}
	}
	assert.True(t, capturedInform, "捕获结果里应含 RPC 方法=Inform 的入向报文")
}

// TestServeHTTP_PassiveCapture_NotWhitelisted_ZeroCaptured：死判 window-filter 反向 ——
// 未起跟踪（SN 不在白名单）时发同样的周期 Inform，捕获条数必须为 0，
// 证明是按设备过滤而非无差别全抓。
func TestServeHTTP_PassiveCapture_NotWhitelisted_ZeroCaptured(t *testing.T) {
	sink := &stubTraceSink{}
	// 白名单为空（whitelistSN=""）→ TEST-SN-002 未起跟踪；准入充足，走正常成功路径。
	h := newPassiveCaptureHandler(t, sink, "", 1000)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformPeriodicXML))
	req.RemoteAddr = "10.0.0.10:5000"
	h.ServeHTTP(w, req)

	assert.Equal(t, 0, len(sink.snapshot()),
		"未起跟踪的设备周期 Inform 不应被捕获（按设备+窗口过滤，非无差别全抓）")
}

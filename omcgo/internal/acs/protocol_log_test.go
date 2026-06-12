package acs

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/omcgo/omcgo/internal/trace"
)

// issue #218（高 CPU 回归治本）回归测试：
//   - 协议日志默认按 warn 级构建 → 每报文全量 XML 的 zap JSON 编码被核心层短路，稳态零开销；
//   - 调到 info 级 → 恢复全量抓包；
//   - max_body_size 截断生效；
//   - 报文跟踪(trace)与协议日志级别解耦：日志被抑制时 trace 命中白名单仍照常捕获。

// syncBuffer 是并发安全的 zapcore.WriteSyncer，便于在测试里读取协议日志输出。
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) Sync() error { return nil }

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// newLeveledProtocolLogger 构造一个写入内存 buffer、级别为 lvl 的协议日志器，
// 语义与 cmd/acs newProtocolLogger 一致（核心级别即抑制点）。
func newLeveledProtocolLogger(lvl zapcore.Level) (*zap.Logger, *syncBuffer) {
	buf := &syncBuffer{}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		buf,
		lvl,
	)
	return zap.New(core), buf
}

// newProtocolLogHandler 构造可跑 ServeHTTP 全链路、挂了协议日志器的 handler。
func newProtocolLogHandler(t *testing.T, lvl zapcore.Level, maxBodySize int) (*Handler, *syncBuffer) {
	t.Helper()
	h := newTestACSHandlerWithDeps(newAcsHSessionStore(), &acsHEventBus{})
	pl, buf := newLeveledProtocolLogger(lvl)
	h.protocolLogger = pl
	h.maxBodySize = maxBodySize
	return h, buf
}

// 成功路径 A（默认零开销）：协议日志器为 warn 级时，正常 Inform 事务流过后
// 不应写出任何 "rpc" 全量 XML 记录 —— 这是 issue #218 削峰的核心。
func TestProtocolLog_WarnLevel_SuppressesFullXML(t *testing.T) {
	h, buf := newProtocolLogHandler(t, zapcore.WarnLevel, 4096)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	req.RemoteAddr = "10.0.0.1:5000"
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "Inform 应成功返回 InformResponse")
	assert.NotContains(t, buf.String(), `"rpc"`,
		"warn 级协议日志必须抑制每报文全量 XML 记录（issue #218 削峰）")
	assert.Empty(t, buf.String(),
		"warn 级下不应有任何协议日志输出（零开销）")
}

// 成功路径 B（排障全量抓包）：协议日志器为 info 级时，应写出含 request_xml 的 "rpc" 记录。
func TestProtocolLog_InfoLevel_EmitsFullXML(t *testing.T) {
	h, buf := newProtocolLogHandler(t, zapcore.InfoLevel, 0)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	req.RemoteAddr = "10.0.0.2:5000"
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	out := buf.String()
	require.Contains(t, out, `"rpc"`, "info 级应恢复全量抓包")
	assert.Contains(t, out, "request_xml", "全量记录应含请求 XML 字段")
	assert.Contains(t, out, "cwmp:Inform", "请求 XML 应为实际 Inform 报文内容")
}

// 截断路径：info 级 + 小 max_body_size 时，落盘的 request_xml 应被截断标记。
func TestProtocolLog_InfoLevel_TruncatesByMaxBodySize(t *testing.T) {
	h, buf := newProtocolLogHandler(t, zapcore.InfoLevel, 64)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	req.RemoteAddr = "10.0.0.3:5000"
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	out := buf.String()
	require.Contains(t, out, `"rpc"`)
	assert.Contains(t, out, "...(truncated)",
		"超过 max_body_size 的 XML 必须截断，不整段进 zap 编码")
}

// 解耦回归（守护"不破坏 trace"）：协议日志被 warn 抑制时，报文跟踪命中白名单仍必须捕获。
// 这把 trace 与协议日志级别彻底解耦 —— 默认零开销不能以牺牲抓包能力为代价。
func TestProtocolLog_WarnLevel_TraceStillCaptures(t *testing.T) {
	sink := &stubTraceSink{}
	h, buf := newProtocolLogHandler(t, zapcore.WarnLevel, 4096)

	// acsHInformBootstrapXML 的 SerialNumber 是 TEST-SN-001。
	wl := trace.NewWhitelistCache(nil, trace.DefaultWhitelistConfig(), zap.NewNop())
	wl.Add("TEST-SN-001", uuid.New())
	h.traceWhitelist = wl
	h.traceService = sink

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	req.RemoteAddr = "10.0.0.4:5000"
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	// 协议日志（warn）被抑制：
	assert.Empty(t, buf.String(), "warn 级协议日志仍应抑制全量 XML 编码")
	// 但 trace 必须照常抓到（与日志级别解耦）：
	msgs := sink.snapshot()
	require.NotEmpty(t, msgs,
		"协议日志被抑制时，命中白名单的报文跟踪仍必须捕获（trace 与协议日志解耦，issue #218）")
}

// 解耦回归：协议日志器为 nil（关闭）但报文跟踪启用时，capturer/defer 仍须注册并抓到报文。
// 旧实现 capturer 仅以 protocolLogger != nil 为门，会漏掉"只开 trace 不开协议日志"的场景。
func TestProtocolLog_NilLogger_TraceStillCaptures(t *testing.T) {
	sink := &stubTraceSink{}
	h := newTestACSHandlerWithDeps(newAcsHSessionStore(), &acsHEventBus{})
	h.protocolLogger = nil // 协议日志关闭

	wl := trace.NewWhitelistCache(nil, trace.DefaultWhitelistConfig(), zap.NewNop())
	wl.Add("TEST-SN-001", uuid.New())
	h.traceWhitelist = wl
	h.traceService = sink

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	req.RemoteAddr = "10.0.0.5:5000"
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotEmpty(t, sink.snapshot(),
		"协议日志关闭但 trace 启用时仍须抓到报文（capturer 不再以 protocolLogger != nil 为唯一门，issue #218）")
}

// 确保 trace 与协议日志都未启用时，ServeHTTP 不 panic 且正常返回（零包装路径）。
func TestProtocolLog_BothDisabled_NoCapture(t *testing.T) {
	h := newTestACSHandlerWithDeps(newAcsHSessionStore(), &acsHEventBus{})
	h.protocolLogger = nil
	h.traceWhitelist = nil

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	req.RemoteAddr = "10.0.0.6:5000"
	require.NotPanics(t, func() { h.ServeHTTP(w, req) })
	assert.Equal(t, http.StatusOK, w.Code)
}

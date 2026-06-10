package rpclog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- LogEntry context 往返 ---

func TestEntryFromContext_RoundTrip(t *testing.T) {
	entry := &LogEntry{
		SessionID: "sess-1",
		DeviceSN:  "SN-001",
		Method:    "Inform",
		StartTime: time.Now(),
	}
	ctx := WithEntry(context.Background(), entry)

	got := EntryFromContext(ctx)
	require.NotNil(t, got)
	assert.Same(t, entry, got, "应返回同一指针，便于 handler 原地填充")
	assert.Equal(t, "SN-001", got.DeviceSN)
}

func TestEntryFromContext_Mutation(t *testing.T) {
	entry := &LogEntry{SessionID: "sess-2"}
	ctx := WithEntry(context.Background(), entry)

	// handler 通过 context 拿到的指针就地修改，原对象可见。
	EntryFromContext(ctx).Method = "GetParameterValues"
	EntryFromContext(ctx).Sequence = 3

	assert.Equal(t, "GetParameterValues", entry.Method)
	assert.Equal(t, 3, entry.Sequence)
}

func TestEntryFromContext_Absent(t *testing.T) {
	// 未注入时返回 nil，不 panic。
	assert.Nil(t, EntryFromContext(context.Background()))
}

func TestEntryFromContext_WrongType(t *testing.T) {
	// context 里塞了同 key 但错误类型（理论上不会发生，因为 ctxKey 私有）——
	// 这里直接验证陌生 context 安全返回 nil。
	ctx := context.WithValue(context.Background(), struct{}{}, "not-an-entry")
	assert.Nil(t, EntryFromContext(ctx))
}

// --- ResponseCapturer 成功路径 ---

func TestResponseCapturer_CapturesBodyAndStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapturer(rec)

	// 默认状态码 200（未显式 WriteHeader）。
	assert.Equal(t, http.StatusOK, cap.StatusCode())

	cap.WriteHeader(http.StatusAccepted)
	n, err := cap.Write([]byte("hello "))
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	_, err = cap.Write([]byte("world"))
	require.NoError(t, err)

	// 同时落到底层 writer 和内部缓冲。
	assert.Equal(t, http.StatusAccepted, cap.StatusCode())
	assert.Equal(t, "hello world", string(cap.Body()))
	assert.Equal(t, "hello world", rec.Body.String())
	assert.Equal(t, http.StatusAccepted, rec.Code)
}

func TestResponseCapturer_EmptyBody(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapturer(rec)

	// 无任何写入时 Body 为空、状态码默认 200。
	assert.Empty(t, cap.Body())
	assert.Equal(t, http.StatusOK, cap.StatusCode())
}

// failingWriter 实现 http.ResponseWriter，Write 始终报错，模拟下游连接中断。
type failingWriter struct {
	header http.Header
}

func (f *failingWriter) Header() http.Header {
	if f.header == nil {
		f.header = http.Header{}
	}
	return f.header
}

func (f *failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }
func (f *failingWriter) WriteHeader(int)            {}

// --- ResponseCapturer 失败路径：底层 Write 出错时 error 透传，但 body 仍被捕获 ---

func TestResponseCapturer_UnderlyingWriteError(t *testing.T) {
	cap := NewResponseCapturer(&failingWriter{})

	n, err := cap.Write([]byte("payload"))
	require.Error(t, err)
	assert.Equal(t, 0, n, "底层 writer 返回的 n 应透传")
	// 即便底层写失败，内部缓冲也已记录，便于事后排障审计报文。
	assert.Equal(t, "payload", string(cap.Body()))
}

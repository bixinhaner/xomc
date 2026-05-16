package trace

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBulkPutter 内存实现 BulkPutter；记录所有 Put 调用，便于断言。
type mockBulkPutter struct {
	mu       sync.Mutex
	enabled  bool
	puts     []bulkPutCall
	failNext bool // 注入失败，验证 fallback 保留 inline
}

type bulkPutCall struct {
	TaskID  uuid.UUID
	MsgID   uuid.UUID
	Payload string
}

func (m *mockBulkPutter) Enabled() bool { return m.enabled }

func (m *mockBulkPutter) Put(_ context.Context, taskID, msgID uuid.UUID, payload string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failNext {
		m.failNext = false
		return "", assert.AnError
	}
	m.puts = append(m.puts, bulkPutCall{TaskID: taskID, MsgID: msgID, Payload: payload})
	return "trace-bulk/" + taskID.String() + "/" + msgID.String() + ".xml.gz", nil
}

// TestCaptureConsumer_Externalize_LargePayloadGoesToBulk L-11 验证：
// > MaxInlinePayloadBytes 的 payload 必须写入 BulkStore，PG 仅留元数据；
// ≤ 阈值的 payload 保留 inline，不调 bulk。
func TestCaptureConsumer_Externalize_LargePayloadGoesToBulk(t *testing.T) {
	bulk := &mockBulkPutter{enabled: true}
	c := NewCaptureConsumer(&captureMockRepo{}, DefaultCaptureConsumerConfig(), nil)
	c.SetBulkStore(bulk)

	smallPayload := strings.Repeat("a", 100) // 100 字节
	exactPayload := strings.Repeat("b", MaxInlinePayloadBytes)
	largePayload := strings.Repeat("c", MaxInlinePayloadBytes+1024) // 33KB+1KB

	taskID := uuid.New()
	batch := []*Message{
		{TaskID: taskID, PayloadInline: smallPayload, PayloadSizeBytes: len(smallPayload)},
		{TaskID: taskID, PayloadInline: exactPayload, PayloadSizeBytes: len(exactPayload)},
		{TaskID: taskID, PayloadInline: largePayload, PayloadSizeBytes: len(largePayload)},
	}

	c.externalize(context.Background(), batch)

	// 小报文：保持 inline
	assert.Equal(t, smallPayload, batch[0].PayloadInline, "small payload should stay inline")
	assert.Empty(t, batch[0].PayloadObjectKey, "small payload should not be externalized")

	// 等于阈值：保持 inline（边界条件，<=）
	assert.Equal(t, exactPayload, batch[1].PayloadInline, "payload at exact threshold should stay inline")
	assert.Empty(t, batch[1].PayloadObjectKey, "exact-threshold payload should not be externalized")

	// 大报文：外置，inline 清空，object_key 非空
	assert.Empty(t, batch[2].PayloadInline, "large payload inline should be cleared")
	assert.Contains(t, batch[2].PayloadObjectKey, "trace-bulk/"+taskID.String()+"/",
		"large payload object_key should match bulk store format")
	assert.NotEqual(t, uuid.Nil, batch[2].ID, "large payload ID must be assigned for object key")

	// Bulk 只被调用 1 次（仅 largePayload）
	bulk.mu.Lock()
	require.Len(t, bulk.puts, 1)
	assert.Equal(t, largePayload, bulk.puts[0].Payload, "bulk Put should receive original large payload")
	bulk.mu.Unlock()
}

// TestCaptureConsumer_Externalize_BulkFailureKeepsInline 大报文外置失败时不丢报文 —
// fallback 保留 inline，避免数据丢失（即使 PG 行可能超 TOAST 限制也总比丢强）。
func TestCaptureConsumer_Externalize_BulkFailureKeepsInline(t *testing.T) {
	bulk := &mockBulkPutter{enabled: true, failNext: true}
	c := NewCaptureConsumer(&captureMockRepo{}, DefaultCaptureConsumerConfig(), nil)
	c.SetBulkStore(bulk)

	largePayload := strings.Repeat("c", MaxInlinePayloadBytes+1024)
	msg := &Message{TaskID: uuid.New(), PayloadInline: largePayload, PayloadSizeBytes: len(largePayload)}

	c.externalize(context.Background(), []*Message{msg})

	assert.Equal(t, largePayload, msg.PayloadInline, "bulk failure must keep inline as fallback")
	assert.Empty(t, msg.PayloadObjectKey, "no object_key on bulk failure")
}

// TestCaptureConsumer_Externalize_NilBulkIsNoop bulk store 未注入时 externalize 无副作用。
func TestCaptureConsumer_Externalize_NilBulkIsNoop(t *testing.T) {
	c := NewCaptureConsumer(&captureMockRepo{}, DefaultCaptureConsumerConfig(), nil)

	largePayload := strings.Repeat("c", MaxInlinePayloadBytes+1024)
	msg := &Message{TaskID: uuid.New(), PayloadInline: largePayload, PayloadSizeBytes: len(largePayload)}

	c.externalize(context.Background(), []*Message{msg})

	assert.Equal(t, largePayload, msg.PayloadInline, "without bulk store, payload stays inline")
	assert.Empty(t, msg.PayloadObjectKey)
}

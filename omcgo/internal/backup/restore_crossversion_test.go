package backup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// memAuditSink 记录跨版本审计事件，便于断言。
type memAuditSink struct {
	events []RestoreAuditEvent
}

func (m *memAuditSink) RecordRestoreAudit(_ context.Context, evt RestoreAuditEvent) {
	m.events = append(m.events, evt)
}

func TestCrossVersion_Match_NoAudit(t *testing.T) {
	sink := &memAuditSink{}
	c := NewCrossVersionChecker(CrossVersionWarnAudit, sink, zap.NewNop())
	res := c.Check(context.Background(), "r1", "SN001", "1.2.0", "1.2.0")
	assert.False(t, res.Mismatch)
	assert.False(t, res.Blocked)
	assert.Empty(t, sink.events, "版本一致不写审计")
}

func TestCrossVersion_CaseInsensitiveMatch(t *testing.T) {
	sink := &memAuditSink{}
	c := NewCrossVersionChecker(CrossVersionWarnAudit, sink, zap.NewNop())
	res := c.Check(context.Background(), "r1", "SN001", "V1.2.0", "v1.2.0")
	assert.False(t, res.Mismatch)
	assert.Empty(t, sink.events)
}

func TestCrossVersion_Mismatch_WarnAudit(t *testing.T) {
	sink := &memAuditSink{}
	c := NewCrossVersionChecker(CrossVersionWarnAudit, sink, zap.NewNop())
	res := c.Check(context.Background(), "r1", "SN001", "1.2.0", "2.0.0")
	assert.True(t, res.Mismatch)
	assert.False(t, res.Blocked, "warn_audit 不阻断")
	require.Len(t, sink.events, 1)
	assert.Equal(t, "SN001", sink.events[0].DeviceSN)
	assert.Equal(t, "1.2.0", sink.events[0].SourceVersion)
	assert.Equal(t, "2.0.0", sink.events[0].TargetVersion)
	assert.False(t, sink.events[0].Blocked)
}

func TestCrossVersion_Mismatch_Block(t *testing.T) {
	sink := &memAuditSink{}
	c := NewCrossVersionChecker(CrossVersionBlock, sink, zap.NewNop())
	res := c.Check(context.Background(), "r1", "SN001", "1.2.0", "2.0.0")
	assert.True(t, res.Mismatch)
	assert.True(t, res.Blocked, "block 策略下不一致即阻断")
	require.Len(t, sink.events, 1)
	assert.True(t, sink.events[0].Blocked)
}

func TestCrossVersion_Mismatch_Allow_NoAudit(t *testing.T) {
	sink := &memAuditSink{}
	c := NewCrossVersionChecker(CrossVersionAllow, sink, zap.NewNop())
	res := c.Check(context.Background(), "r1", "SN001", "1.2.0", "2.0.0")
	assert.True(t, res.Mismatch)
	assert.False(t, res.Blocked)
	assert.Empty(t, sink.events, "allow 策略不写审计")
}

func TestCrossVersion_UnknownVersion_Skipped(t *testing.T) {
	sink := &memAuditSink{}
	c := NewCrossVersionChecker(CrossVersionBlock, sink, zap.NewNop())
	// 源版本未知（空）→ 不可比 → 不告警不阻断。
	res := c.Check(context.Background(), "r1", "SN001", "", "2.0.0")
	assert.False(t, res.Mismatch)
	assert.False(t, res.Blocked)
	assert.Empty(t, sink.events)
}

func TestCrossVersion_NilAuditSink_NoPanic(t *testing.T) {
	c := NewCrossVersionChecker(CrossVersionWarnAudit, nil, zap.NewNop())
	res := c.Check(context.Background(), "r1", "SN001", "1.0", "2.0")
	assert.True(t, res.Mismatch) // 仍正确判定，只是不写审计
}

func TestCrossVersion_EmptyStrategy_DefaultsWarnAudit(t *testing.T) {
	sink := &memAuditSink{}
	c := NewCrossVersionChecker("", sink, zap.NewNop())
	res := c.Check(context.Background(), "r1", "SN001", "1.0", "2.0")
	assert.True(t, res.Mismatch)
	assert.False(t, res.Blocked)
	require.Len(t, sink.events, 1)
	assert.Equal(t, CrossVersionWarnAudit, sink.events[0].Strategy)
}

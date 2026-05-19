package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Test_Service_UpsertByDedup_InsertThenUpgrade 覆盖 T-0157 C3 核心场景：
// 同一 (user_id, dedup_key) 首次写入插入；二次写入升级 status / title / content。
func Test_Service_UpsertByDedup_InsertThenUpgrade(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())

	dedupKey := "task-uuid-001"
	first := &Notification{
		UserID:   "alice",
		Type:     NotifTypeTaskComplete,
		Status:   StatusQueued,
		Priority: PriorityNormal,
		Title:    "小区参数 · TAC: 3 → 4",
		Content:  "TAC: 3 → 4",
		Link:     "/device/detail/SN001?tab=quickSettings",
		Sender:   "system",
		DedupKey: &dedupKey,
	}

	got1, err := svc.UpsertByDedup(context.Background(), first)
	require.NoError(t, err)
	require.NotNil(t, got1)
	originalID := got1.ID
	assert.Equal(t, StatusQueued, got1.Status)

	// 二次 upsert：状态升级到 completed
	second := &Notification{
		UserID:   "alice",
		Type:     NotifTypeTaskComplete,
		Status:   StatusCompleted,
		Priority: PriorityNormal,
		Title:    "小区参数 · TAC: 3 → 4 (已完成)",
		Content:  "TAC: 3 → 4\n基站应答成功",
		Link:     "/device/detail/SN001?tab=quickSettings",
		Sender:   "system",
		DedupKey: &dedupKey,
	}
	got2, err := svc.UpsertByDedup(context.Background(), second)
	require.NoError(t, err)
	assert.Equal(t, originalID, got2.ID, "升级时 id 应保持不变")
	assert.Equal(t, StatusCompleted, got2.Status)
	assert.Equal(t, "小区参数 · TAC: 3 → 4 (已完成)", got2.Title)
}

// Test_Service_UpsertByDedup_DifferentUsers_Isolated 验证 (user_id, dedup_key) 隔离：
// alice 和 bob 各自的同 dedup_key 不互相覆盖（按用户隔离）。
func Test_Service_UpsertByDedup_DifferentUsers_Isolated(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())

	dedupKey := "task-shared-key"
	aliceNotif := &Notification{
		UserID:   "alice",
		Type:     NotifTypeTaskComplete,
		Status:   StatusQueued,
		Title:    "alice 的消息",
		DedupKey: &dedupKey,
	}
	bobNotif := &Notification{
		UserID:   "bob",
		Type:     NotifTypeTaskComplete,
		Status:   StatusCompleted,
		Title:    "bob 的消息",
		DedupKey: &dedupKey,
	}

	_, err := svc.UpsertByDedup(context.Background(), aliceNotif)
	require.NoError(t, err)
	_, err = svc.UpsertByDedup(context.Background(), bobNotif)
	require.NoError(t, err)

	aliceList, err := svc.List(context.Background(), NotificationFilter{UserID: "alice"})
	require.NoError(t, err)
	require.Len(t, aliceList.Items, 1)
	assert.Equal(t, StatusQueued, aliceList.Items[0].Status)
	assert.Equal(t, "alice 的消息", aliceList.Items[0].Title)

	bobList, err := svc.List(context.Background(), NotificationFilter{UserID: "bob"})
	require.NoError(t, err)
	require.Len(t, bobList.Items, 1)
	assert.Equal(t, StatusCompleted, bobList.Items[0].Status)
	assert.Equal(t, "bob 的消息", bobList.Items[0].Title)
}

// Test_Service_UpsertByDedup_NilDedupKey_FallsBackToCreate 验证 DedupKey 为 nil 时
// 走 Create 路径（每次插入新行），保留非 task 类消息的正常插入语义。
func Test_Service_UpsertByDedup_NilDedupKey_FallsBackToCreate(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())

	first := &Notification{
		UserID:  "alice",
		Type:    NotifTypeSystem,
		Status:  StatusCompleted,
		Title:   "系统消息 1",
		Sender:  "system",
		// DedupKey: nil
	}
	second := &Notification{
		UserID: "alice",
		Type:   NotifTypeSystem,
		Status: StatusCompleted,
		Title:  "系统消息 2",
		Sender: "system",
	}

	_, err := svc.UpsertByDedup(context.Background(), first)
	require.NoError(t, err)
	_, err = svc.UpsertByDedup(context.Background(), second)
	require.NoError(t, err)

	list, err := svc.List(context.Background(), NotificationFilter{UserID: "alice"})
	require.NoError(t, err)
	assert.Len(t, list.Items, 2, "DedupKey 为 nil 时不去重，两条独立插入")
}

// Test_Service_UpsertByDedup_RepoErrorPropagated 验证 repo.UpsertByDedup 错误向上传播。
func Test_Service_UpsertByDedup_RepoErrorPropagated(t *testing.T) {
	repo := newMockRepository()
	repo.upsertErr = errBoom
	svc := NewService(repo, nil, zap.NewNop())

	dedupKey := "any-key"
	_, err := svc.UpsertByDedup(context.Background(), &Notification{
		UserID: "alice", DedupKey: &dedupKey,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upsert notification by dedup")
}

package device

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- stub implementations ---

type stubAutoRecycleDeleter struct {
	findIDs     [][]uuid.UUID // 每次 FindOfflineForRecycle 返回的批次（按调用顺序）
	findCallIdx int
	deleted     []uuid.UUID // 记录 BatchDelete 收到的 ids
	metadata    RecycleMetadata
	batchErr    error
	findErr     error
}

func (s *stubAutoRecycleDeleter) FindOfflineForRecycle(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if s.findCallIdx >= len(s.findIDs) {
		return nil, nil
	}
	ids := s.findIDs[s.findCallIdx]
	s.findCallIdx++
	return ids, nil
}

func (s *stubAutoRecycleDeleter) BatchDeleteWithMetadata(_ context.Context, ids []uuid.UUID, metadata RecycleMetadata) (int64, error) {
	if s.batchErr != nil {
		return 0, s.batchErr
	}
	s.deleted = append(s.deleted, ids...)
	s.metadata = metadata
	return int64(len(ids)), nil
}

// stubLookup 构造从 map[category+key]value 返回结果的查找函数。
func stubLookup(m map[string]string) SysConfigLookupFn {
	return func(_ context.Context, category, key string) (string, bool) {
		v, ok := m[category+":"+key]
		return v, ok
	}
}

// --- 测试用例 ---

func TestAutoRecycleJob_Disabled(t *testing.T) {
	// 开关未开启，应跳过执行
	ops := &stubAutoRecycleDeleter{}
	job := NewAutoRecycleJob(
		stubLookup(map[string]string{"device:deviceOfflineEnable": "false", "device:deviceOfflineSaveDay": "30"}),
		ops, 0, zap.NewNop(),
	)
	n, err := job.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
	assert.Empty(t, ops.deleted)
}

func TestAutoRecycleJob_EnabledNotFound(t *testing.T) {
	// 开关 key 在 sys_configs 中根本不存在，应跳过执行
	ops := &stubAutoRecycleDeleter{}
	job := NewAutoRecycleJob(
		stubLookup(map[string]string{}), // 空 map
		ops, 0, zap.NewNop(),
	)
	n, err := job.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestAutoRecycleJob_MissingDayConfig(t *testing.T) {
	// 开关开启但 deviceOfflineSaveDay 未配置，应跳过并 warn
	ops := &stubAutoRecycleDeleter{}
	job := NewAutoRecycleJob(
		stubLookup(map[string]string{"device:deviceOfflineEnable": "true"}),
		ops, 0, zap.NewNop(),
	)
	n, err := job.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
	assert.Empty(t, ops.deleted)
}

func TestAutoRecycleJob_NoDevicesToRecycle(t *testing.T) {
	// 开关开启、配置齐全、但没有符合条件的设备
	ops := &stubAutoRecycleDeleter{findIDs: [][]uuid.UUID{{}}} // 返回空列表
	job := NewAutoRecycleJob(
		stubLookup(map[string]string{
			"device:deviceOfflineEnable":  "true",
			"device:deviceOfflineSaveDay": "90",
		}),
		ops, 0, zap.NewNop(),
	)
	n, err := job.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
	assert.Empty(t, ops.deleted)
}

func TestAutoRecycleJob_DeletesDevicesAsSystem(t *testing.T) {
	// 正常路径：有 3 个符合条件的设备，全部软删除
	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	ops := &stubAutoRecycleDeleter{
		findIDs: [][]uuid.UUID{{id1, id2, id3}},
	}
	job := NewAutoRecycleJob(
		stubLookup(map[string]string{
			"device:deviceOfflineEnable":  "true",
			"device:deviceOfflineSaveDay": "30",
		}),
		ops, 0, zap.NewNop(),
	)
	n, err := job.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
	assert.ElementsMatch(t, []uuid.UUID{id1, id2, id3}, ops.deleted)
	assert.Equal(t, RecycleMetadata{
		DeletedBy: "system",
		Type:      RecycleTypeAuto,
		Executor:  "system:auto_recycle",
	}, ops.metadata)
}

func TestAutoRecycleJob_DeletesAcrossMultipleBatches(t *testing.T) {
	// 多批次：batchSize=2，共 3 个设备（两批才清完）
	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	ops := &stubAutoRecycleDeleter{
		// 第一次 Find 返回满批（2 个），第二次返回剩余 1 个，第三次返回空
		findIDs: [][]uuid.UUID{{id1, id2}, {id3}, {}},
	}
	job := NewAutoRecycleJob(
		stubLookup(map[string]string{
			"device:deviceOfflineEnable":  "true",
			"device:deviceOfflineSaveDay": "30",
		}),
		ops, 2, zap.NewNop(),
	)
	n, err := job.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
	assert.ElementsMatch(t, []uuid.UUID{id1, id2, id3}, ops.deleted)
}

func TestAutoRecycleJob_ZeroDays(t *testing.T) {
	// deviceOfflineSaveDay=0 应返回错误
	ops := &stubAutoRecycleDeleter{}
	job := NewAutoRecycleJob(
		stubLookup(map[string]string{
			"device:deviceOfflineEnable":  "true",
			"device:deviceOfflineSaveDay": "0",
		}),
		ops, 0, zap.NewNop(),
	)
	_, err := job.Run(context.Background())
	assert.Error(t, err)
	assert.Empty(t, ops.deleted)
}

func TestAutoRecycleJob_InvalidDays(t *testing.T) {
	// deviceOfflineSaveDay=abc（非整数）应返回错误
	ops := &stubAutoRecycleDeleter{}
	job := NewAutoRecycleJob(
		stubLookup(map[string]string{
			"device:deviceOfflineEnable":  "true",
			"device:deviceOfflineSaveDay": "abc",
		}),
		ops, 0, zap.NewNop(),
	)
	_, err := job.Run(context.Background())
	assert.Error(t, err)
}

func TestAutoRecycleJob_FindError(t *testing.T) {
	// FindOfflineForRecycle 返回错误，Run 应透传错误
	ops := &stubAutoRecycleDeleter{findErr: assert.AnError}
	job := NewAutoRecycleJob(
		stubLookup(map[string]string{
			"device:deviceOfflineEnable":  "true",
			"device:deviceOfflineSaveDay": "30",
		}),
		ops, 0, zap.NewNop(),
	)
	_, err := job.Run(context.Background())
	assert.ErrorIs(t, err, assert.AnError)
}

func TestAutoRecycleJob_BatchDeleteError(t *testing.T) {
	// BatchDelete 返回错误，Run 应透传错误
	id1 := uuid.New()
	ops := &stubAutoRecycleDeleter{
		findIDs:  [][]uuid.UUID{{id1}},
		batchErr: assert.AnError,
	}
	job := NewAutoRecycleJob(
		stubLookup(map[string]string{
			"device:deviceOfflineEnable":  "true",
			"device:deviceOfflineSaveDay": "30",
		}),
		ops, 0, zap.NewNop(),
	)
	_, err := job.Run(context.Background())
	assert.ErrorIs(t, err, assert.AnError)
}

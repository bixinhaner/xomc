package stationlog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetentionPolicy_Defaults(t *testing.T) {
	p := NewRetentionPolicy(nil, nil) // nil lookup → 恒默认
	assert.Equal(t, DefaultMaxRetentionDays, p.MaxRetentionDays(context.Background()))
	assert.Equal(t, DefaultMaxFileCount, p.MaxFileCount(context.Background()))
	assert.Equal(t, DefaultCleanupIntervalMinutes, p.CleanupIntervalMinutes(context.Background()))
}

func TestRetentionPolicy_ReadsAndValidates(t *testing.T) {
	values := map[string]string{
		KeyMaxRetentionDays:       "90",
		KeyMaxFileCount:           "0", // 0 = 禁用配额
		KeyCleanupIntervalMinutes: "120",
	}
	lookup := func(_ context.Context, category, key string) (string, bool) {
		if category != RetentionCategory {
			return "", false
		}
		v, ok := values[key]
		return v, ok
	}
	p := NewRetentionPolicy(lookup, nil)
	assert.Equal(t, 90, p.MaxRetentionDays(context.Background()))
	assert.Equal(t, 0, p.MaxFileCount(context.Background()))
	assert.Equal(t, 120, p.CleanupIntervalMinutes(context.Background()))

	// 非法值回落默认。
	bad := NewRetentionPolicy(func(_ context.Context, _, key string) (string, bool) {
		return "-5", true
	}, nil)
	assert.Equal(t, DefaultMaxRetentionDays, bad.MaxRetentionDays(context.Background()))
	assert.Equal(t, DefaultCleanupIntervalMinutes, bad.CleanupIntervalMinutes(context.Background()))
}

func TestRetentionPolicy_CleanupIntervalMinutes_ReadsRangeAndFallback(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want int
	}{
		{name: "min", raw: "10", want: 10},
		{name: "max", raw: "1440", want: 1440},
		{name: "trim", raw: " 30 ", want: 30},
		{name: "too small", raw: "9", want: DefaultCleanupIntervalMinutes},
		{name: "too large", raw: "1441", want: DefaultCleanupIntervalMinutes},
		{name: "not int", raw: "abc", want: DefaultCleanupIntervalMinutes},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewRetentionPolicy(func(_ context.Context, category, key string) (string, bool) {
				if category != RetentionCategory || key != KeyCleanupIntervalMinutes {
					return "", false
				}
				return tc.raw, true
			}, nil)
			assert.Equal(t, tc.want, p.CleanupIntervalMinutes(context.Background()))
		})
	}
}

func TestRetentionPolicy_MaxFileCountPerDevice_Default(t *testing.T) {
	p := NewRetentionPolicy(nil, nil) // nil lookup → 恒默认
	assert.Equal(t, DefaultMaxFileCountPerDevice, p.MaxFileCountPerDevice(context.Background()))
}

func TestRetentionPolicy_MaxFileCountPerDevice_ReadsAndValidates(t *testing.T) {
	lookup := func(_ context.Context, category, key string) (string, bool) {
		if category != RetentionCategory || key != KeyMaxFileCountPerDevice {
			return "", false
		}
		return "8", true
	}
	p := NewRetentionPolicy(lookup, nil)
	assert.Equal(t, 8, p.MaxFileCountPerDevice(context.Background()))

	// 非法值（负数）回落默认。
	bad := NewRetentionPolicy(func(_ context.Context, _, key string) (string, bool) {
		if key == KeyMaxFileCountPerDevice {
			return "-3", true
		}
		return "", false
	}, nil)
	assert.Equal(t, DefaultMaxFileCountPerDevice, bad.MaxFileCountPerDevice(context.Background()))
}

func TestRetentionPolicy_InvalidateCacheReloadsUpdatedQuota(t *testing.T) {
	values := map[string]string{
		KeyMaxRetentionDays:      "60",
		KeyMaxFileCount:          "1000",
		KeyMaxFileCountPerDevice: "5",
	}
	lookup := func(_ context.Context, category, key string) (string, bool) {
		if category != RetentionCategory {
			return "", false
		}
		v, ok := values[key]
		return v, ok
	}
	p := NewRetentionPolicy(lookup, nil)

	assert.Equal(t, 5, p.MaxFileCountPerDevice(context.Background()))
	values[KeyMaxFileCountPerDevice] = "2"
	assert.Equal(t, 5, p.MaxFileCountPerDevice(context.Background()), "TTL 命中时应仍读缓存")

	p.InvalidateCache()
	assert.Equal(t, 2, p.MaxFileCountPerDevice(context.Background()), "失效缓存后应立刻读到新配额")
}

// ---- CleanupRunner ----

type fakeCleanupStore struct {
	remaining []*LogFile
	deleted   []uuid.UUID
}

func (f *fakeCleanupStore) ListExpired(_ context.Context, _ time.Time, limit int) ([]*LogFile, error) {
	if len(f.remaining) == 0 {
		return nil, nil
	}
	n := limit
	if n > len(f.remaining) {
		n = len(f.remaining)
	}
	out := f.remaining[:n]
	f.remaining = f.remaining[n:]
	return out, nil
}

func (f *fakeCleanupStore) MarkDeleted(_ context.Context, id uuid.UUID) error {
	f.deleted = append(f.deleted, id)
	return nil
}

type fakeRemover struct{ removed []string }

func (r *fakeRemover) RemoveObject(_ context.Context, bucket, object string, _ minio.RemoveObjectOptions) error {
	r.removed = append(r.removed, bucket+"/"+object)
	return nil
}

type failingRemover struct{ removed []string }

func (r *failingRemover) RemoveObject(_ context.Context, bucket, object string, _ minio.RemoveObjectOptions) error {
	r.removed = append(r.removed, bucket+"/"+object)
	return errors.New("minio unavailable")
}

func logFile(bucket, path string) *LogFile {
	return &LogFile{ID: uuid.New(), Bucket: bucket, ObjectPath: path}
}

type fakeTaskLogStore struct {
	remaining []TaskLogFile
	deleted   []int64
}

func (f *fakeTaskLogStore) ListExpiredTaskLogs(_ context.Context, _ time.Time, limit int) ([]TaskLogFile, error) {
	if len(f.remaining) == 0 {
		return nil, nil
	}
	n := limit
	if n > len(f.remaining) {
		n = len(f.remaining)
	}
	out := f.remaining[:n]
	f.remaining = f.remaining[n:]
	return out, nil
}

func (f *fakeTaskLogStore) MarkTaskLogDeleted(_ context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func TestCleanupRunner_DeletesExpiredObjectsAndRows(t *testing.T) {
	fault := &fakeCleanupStore{remaining: []*LogFile{
		logFile("logs", "fault/a"),
		logFile("logs", "fault/b"),
		logFile("", ""), // detected 占位：无对象，仅软删
	}}
	running := &fakeCleanupStore{remaining: []*LogFile{
		logFile("logs", "run/x"),
	}}
	remover := &fakeRemover{}
	runner := NewCleanupRunner(fault, running, remover, NewRetentionPolicy(nil, nil), nil)

	out, err := runner.Run(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, out)

	// 三条 fault + 一条 running 全部软删。
	assert.Len(t, fault.deleted, 3)
	assert.Len(t, running.deleted, 1)
	// 只对有 object_path 的记录删 MinIO（fault 2 条 + running 1 条 = 3），跳过空占位。
	assert.ElementsMatch(t, []string{"logs/fault/a", "logs/fault/b", "logs/run/x"}, remover.removed)
}

func TestCleanupRunner_DoesNotMarkDeletedWhenObjectRemovalFails(t *testing.T) {
	old := logFile("logs", "fault/a")
	placeholder := logFile("", "")
	fault := &fakeCleanupStore{remaining: []*LogFile{old, placeholder}}
	remover := &failingRemover{}
	runner := NewCleanupRunner(fault, nil, remover, NewRetentionPolicy(nil, nil), nil)

	out, err := runner.Run(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, out)

	assert.Equal(t, []string{"logs/fault/a"}, remover.removed)
	assert.Equal(t, []uuid.UUID{placeholder.ID}, fault.deleted, "MinIO 删除失败的记录不能错误软删；无对象占位仍可软删")
}

func TestCleanupRunner_DeletesExpiredTaskLogMetadata(t *testing.T) {
	taskLogs := &fakeTaskLogStore{remaining: []TaskLogFile{
		{ID: 101, Bucket: "backup", ObjectPath: "logs/running/task-a.log"},
		{ID: 102, Bucket: "backup", ObjectPath: "logs/fault/task-b.log"},
		{ID: 103}, // bad legacy metadata: no object, still soft-delete metadata
	}}
	remover := &fakeRemover{}
	runner := NewCleanupRunner(nil, nil, remover, NewRetentionPolicy(nil, nil), nil)
	runner.SetTaskLogStore(taskLogs)

	out, err := runner.Run(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, out)

	assert.Equal(t, []int64{101, 102, 103}, taskLogs.deleted)
	assert.ElementsMatch(t, []string{
		"backup/logs/running/task-a.log",
		"backup/logs/fault/task-b.log",
	}, remover.removed)
	assert.JSONEq(t, `{"days":60,"fault_deleted":0,"running_deleted":0,"task_log_deleted":3}`, string(out))
}

func TestCleanupRunner_DoesNotMarkTaskLogDeletedWhenObjectRemovalFails(t *testing.T) {
	taskLogs := &fakeTaskLogStore{remaining: []TaskLogFile{
		{ID: 101, Bucket: "backup", ObjectPath: "logs/running/task-a.log"},
		{ID: 102},
	}}
	remover := &failingRemover{}
	runner := NewCleanupRunner(nil, nil, remover, NewRetentionPolicy(nil, nil), nil)
	runner.SetTaskLogStore(taskLogs)

	out, err := runner.Run(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, out)

	assert.Equal(t, []string{"backup/logs/running/task-a.log"}, remover.removed)
	assert.Equal(t, []int64{102}, taskLogs.deleted, "MinIO 删除失败的任务文件不能错误软删；无对象占位仍可软删")
}

func TestCleanupRunner_NilStoresSafe(t *testing.T) {
	runner := NewCleanupRunner(nil, nil, nil, NewRetentionPolicy(nil, nil), nil)
	_, err := runner.Run(context.Background(), nil)
	require.NoError(t, err)
}

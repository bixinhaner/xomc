package stationlog

import (
	"context"
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
}

func TestRetentionPolicy_ReadsAndValidates(t *testing.T) {
	values := map[string]string{
		KeyMaxRetentionDays: "90",
		KeyMaxFileCount:     "0", // 0 = 禁用配额
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

	// 非法值回落默认。
	bad := NewRetentionPolicy(func(_ context.Context, _, key string) (string, bool) {
		return "-5", true
	}, nil)
	assert.Equal(t, DefaultMaxRetentionDays, bad.MaxRetentionDays(context.Background()))
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

func logFile(bucket, path string) *LogFile {
	return &LogFile{ID: uuid.New(), Bucket: bucket, ObjectPath: path}
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

func TestCleanupRunner_NilStoresSafe(t *testing.T) {
	runner := NewCleanupRunner(nil, nil, nil, NewRetentionPolicy(nil, nil), nil)
	_, err := runner.Run(context.Background(), nil)
	require.NoError(t, err)
}

package license

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeMinIO 是 archiveObjectIO 的内存实现（仅 PutObject + StatObject 两方法）。
type fakeMinIO struct {
	mu      sync.Mutex
	objects map[string][]byte // bucket/key → bytes
	putErr  error             // 注入失败
}

func newFakeMinIO() *fakeMinIO {
	return &fakeMinIO{objects: map[string][]byte{}}
}

func (m *fakeMinIO) PutObject(_ context.Context, bucket, key string, reader io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	if m.putErr != nil {
		return minio.UploadInfo{}, m.putErr
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[bucket+"/"+key] = body
	return minio.UploadInfo{Bucket: bucket, Key: key, Size: int64(len(body))}, nil
}

func (m *fakeMinIO) StatObject(_ context.Context, bucket, key string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, ok := m.objects[bucket+"/"+key]
	if !ok {
		return minio.ObjectInfo{}, errors.New("not found")
	}
	return minio.ObjectInfo{Key: key, Size: int64(len(body))}, nil
}

func (m *fakeMinIO) get(t *testing.T, bucket, key string) []byte {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	body, ok := m.objects[bucket+"/"+key]
	require.True(t, ok, "expected object %s/%s to exist", bucket, key)
	return body
}

// seedLogsForArchive 往 memLogRepo 灌入指定 created_at 的日志。
func seedLogsForArchive(t *testing.T, repo *memLogRepo, ts time.Time, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		log := &LicenseLog{
			ID:        uuid.New(),
			LogType:   LogTypeImport,
			Result:    LogResultSuccess,
			Details:   json.RawMessage(`{"summary":"seed"}`),
			CreatedAt: ts,
		}
		require.NoError(t, repo.Create(context.Background(), log))
	}
}

func TestLogArchiver_NoMinIO_ShortCircuits(t *testing.T) {
	repo := newMemLogRepo()
	a := NewLogArchiver(repo, nil, "", 6, zap.NewNop())
	res, err := a.ArchiveOnce(context.Background())
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 0, res.LogsArchived)
}

func TestLogArchiver_EmptyDB_NoOp(t *testing.T) {
	repo := newMemLogRepo()
	mc := newFakeMinIO()
	a := NewLogArchiver(repo, mc, "license-archive", 6, zap.NewNop())

	res, err := a.ArchiveOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, res.LogsArchived)
	assert.Equal(t, int64(0), res.LogsDeleted)
	assert.Empty(t, mc.objects)
}

func TestLogArchiver_HappyPath_SingleMonth(t *testing.T) {
	// 改 nowFunc 让 cutoff 确定
	origNow := nowFunc
	defer func() { nowFunc = origNow }()
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	nowFunc = func() time.Time { return now }

	repo := newMemLogRepo()
	mc := newFakeMinIO()

	// 早于 cutoff (now - 6 months = 2025-11-09) 的日志：归档
	oldTime := time.Date(2025, 8, 15, 10, 0, 0, 0, time.UTC)
	seedLogsForArchive(t, repo, oldTime, 3)

	// 晚于 cutoff 的日志：保留
	newTime := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	seedLogsForArchive(t, repo, newTime, 2)

	a := NewLogArchiver(repo, mc, "license-archive", 6, zap.NewNop())
	res, err := a.ArchiveOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, res.MonthsBucket)
	assert.Equal(t, 3, res.LogsArchived)
	assert.Equal(t, int64(3), res.LogsDeleted)

	// MinIO 内 key 命中
	body := mc.get(t, "license-archive", "license-logs/2025-08.jsonl.gz")
	gr, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	decoded, err := io.ReadAll(gr)
	require.NoError(t, err)
	require.NoError(t, gr.Close())
	lines := strings.Split(strings.TrimRight(string(decoded), "\n"), "\n")
	assert.Len(t, lines, 3, "JSONL has 3 records")

	// DB 残留 2 条新日志
	remaining := repo.snapshot()
	assert.Len(t, remaining, 2)
	for _, l := range remaining {
		assert.Equal(t, newTime, l.CreatedAt)
	}
}

func TestLogArchiver_HappyPath_MultipleMonths(t *testing.T) {
	origNow := nowFunc
	defer func() { nowFunc = origNow }()
	nowFunc = func() time.Time { return time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC) }

	repo := newMemLogRepo()
	mc := newFakeMinIO()

	// 三个月分别埋日志
	seedLogsForArchive(t, repo, time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC), 2)
	seedLogsForArchive(t, repo, time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), 3)
	seedLogsForArchive(t, repo, time.Date(2025, 10, 20, 0, 0, 0, 0, time.UTC), 1)

	a := NewLogArchiver(repo, mc, "lic-arch", 6, zap.NewNop())
	res, err := a.ArchiveOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, res.MonthsBucket)
	assert.Equal(t, 6, res.LogsArchived)
	assert.Equal(t, int64(6), res.LogsDeleted)

	for _, m := range []string{"2025-08", "2025-09", "2025-10"} {
		key := "license-logs/" + m + ".jsonl.gz"
		body := mc.get(t, "lic-arch", key)
		assert.Greater(t, len(body), 10, "month %s archive should be non-trivial", m)
	}
	assert.Empty(t, repo.snapshot())
}

func TestLogArchiver_PutObjectFails_DBNotDeleted(t *testing.T) {
	origNow := nowFunc
	defer func() { nowFunc = origNow }()
	nowFunc = func() time.Time { return time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC) }

	repo := newMemLogRepo()
	mc := newFakeMinIO()
	mc.putErr = errors.New("minio down")

	seedLogsForArchive(t, repo, time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), 5)

	a := NewLogArchiver(repo, mc, "lic-arch", 6, zap.NewNop())
	res, err := a.ArchiveOnce(context.Background())
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "put archive object")

	// DB 完整保留（下次 tick 重试）
	assert.Len(t, repo.snapshot(), 5, "DB rows must be preserved on MinIO failure")
}

func TestLogArchiver_RetentionDefault(t *testing.T) {
	// 0 → 默认 6 月
	a := NewLogArchiver(newMemLogRepo(), newFakeMinIO(), "b", 0, zap.NewNop())
	assert.Equal(t, 6, a.retentionMonths)
}

func TestEncodeLogsJSONLGz_Decompresses(t *testing.T) {
	logs := []LicenseLog{
		{ID: uuid.New(), LogType: LogTypeImport, Result: LogResultSuccess, Details: json.RawMessage(`{}`), CreatedAt: time.Now()},
		{ID: uuid.New(), LogType: LogTypeRevoke, Result: LogResultFailed, Details: json.RawMessage(`{}`), CreatedAt: time.Now()},
	}
	body, err := encodeLogsJSONLGz(logs)
	require.NoError(t, err)
	gr, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	out, err := io.ReadAll(gr)
	require.NoError(t, err)
	require.NoError(t, gr.Close())
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	assert.Len(t, lines, 2)
}

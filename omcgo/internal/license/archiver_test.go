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

func (m *fakeMinIO) get(t *testing.T, bucket, key string) []byte {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	body, ok := m.objects[bucket+"/"+key]
	require.True(t, ok, "expected object %s/%s to exist", bucket, key)
	return body
}

// listKeysByPrefix 返回 bucket 下匹配前缀的 key（用于 P4-B1 校验同月归档片段）。
func (m *fakeMinIO) listKeysByPrefix(bucket, prefix string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	full := bucket + "/" + prefix
	out := make([]string, 0)
	for k := range m.objects {
		if strings.HasPrefix(k, full) {
			out = append(out, strings.TrimPrefix(k, bucket+"/"))
		}
	}
	return out
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

	// T-0100-P4-B1：键名 = `license-logs/{YYYY-MM}/{tickTS}-{count}.jsonl.gz`
	keys := mc.listKeysByPrefix("license-archive", "license-logs/2025-08/")
	require.Len(t, keys, 1, "expected exactly 1 archive shard for 2025-08")
	assert.Contains(t, keys[0], "20260509T120000Z-3.jsonl.gz",
		"key includes tickTS UTC + log count")

	body := mc.get(t, "license-archive", keys[0])
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

	// T-0100-P4-B1：每月各一个 shard，前缀 `license-logs/{YYYY-MM}/`
	for _, m := range []string{"2025-08", "2025-09", "2025-10"} {
		keys := mc.listKeysByPrefix("lic-arch", "license-logs/"+m+"/")
		require.Len(t, keys, 1, "month %s should have exactly 1 archive shard", m)
		body := mc.get(t, "lic-arch", keys[0])
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

// TestLogArchiver_CrossTickSameMonth_NoOverwrite — T-0100-P4-B1 回归测试。
//
// 场景：周级 cron + 6 月保留期，cutoff 落在月内。第 N 周 tick 归档某月 1-9 日
// 日志，第 N+1 周 tick 归档同月 10-16 日日志。修复前两次 tick 都写到
// `license-logs/{YYYY-MM}.jsonl.gz` 同一个 key 互相覆盖；修复后两次写到不同
// 唯一 key（含 tickTS），同月所有 shard 在 `license-logs/{YYYY-MM}/` 前缀下
// 共存，零数据丢失。
//
// 此测试是 review 报告 REVIEW_922d87a4_chenbo01_license.md CRITICAL #1 的回归
// 防护，必须随 archiver.go 同步维护；移除前请先评估键名方案是否回退。
func TestLogArchiver_CrossTickSameMonth_NoOverwrite(t *testing.T) {
	origNow := nowFunc
	defer func() { nowFunc = origNow }()

	repo := newMemLogRepo()
	mc := newFakeMinIO()
	a := NewLogArchiver(repo, mc, "lic-arch", 6, zap.NewNop())

	// 同月（2025-11）跨两个时段灌日志
	earlyNov := time.Date(2025, 11, 5, 8, 0, 0, 0, time.UTC)
	lateNov := time.Date(2025, 11, 13, 8, 0, 0, 0, time.UTC)
	seedLogsForArchive(t, repo, earlyNov, 9)  // tick 1 会归档这 9 条
	seedLogsForArchive(t, repo, lateNov, 7)   // tick 2 才归档这 7 条

	// Tick 1：cutoff = 2025-11-10，归档 11-05 那 9 条；11-13 7 条留下
	nowFunc = func() time.Time { return time.Date(2026, 5, 10, 3, 0, 0, 0, time.UTC) }
	res1, err := a.ArchiveOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 9, res1.LogsArchived)
	assert.Equal(t, int64(9), res1.LogsDeleted)
	assert.Len(t, repo.snapshot(), 7, "11-13 logs not yet eligible")

	// Tick 2：cutoff = 2025-11-17，归档剩余 11-13 那 7 条
	nowFunc = func() time.Time { return time.Date(2026, 5, 17, 3, 0, 0, 0, time.UTC) }
	res2, err := a.ArchiveOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 7, res2.LogsArchived)
	assert.Equal(t, int64(7), res2.LogsDeleted)
	assert.Empty(t, repo.snapshot(), "all archived")

	// 关键断言：同月前缀下应有 2 个不同 shard，加起来 16 条记录
	keys := mc.listKeysByPrefix("lic-arch", "license-logs/2025-11/")
	require.Len(t, keys, 2, "two ticks must produce two distinct shards (no overwrite)")

	totalRecords := 0
	for _, k := range keys {
		body := mc.get(t, "lic-arch", k)
		gr, err := gzip.NewReader(bytes.NewReader(body))
		require.NoError(t, err)
		decoded, err := io.ReadAll(gr)
		require.NoError(t, err)
		require.NoError(t, gr.Close())
		totalRecords += len(strings.Split(strings.TrimRight(string(decoded), "\n"), "\n"))
	}
	assert.Equal(t, 16, totalRecords,
		"all 16 logs preserved across 2 shards; pre-fix this would be 7 (tick 2 overwrote tick 1)")
}

// TestLogArchiver_BatchOverflow_OnlyArchivedDeleted — T-0100-P4-B2 回归测试。
//
// 场景：单 tick 积压超过 batchSize（用 SetBatchSize(3) 模拟）。ListBefore
// 只取最早的 3 条；剩余 4 条仍 < cutoff 但本 tick 不归档。修复前 DeleteBefore
// 会按 cutoff 一刀切把 7 条全删（4 条没归档就丢了）；修复后 DeleteByIDs(本批 3 条 id)
// 仅删归档过的，剩余 4 条留 DB 等下个 tick 处理 → 零数据丢失。
//
// 此测试是 review 报告 REVIEW_922d87a4_chenbo01_license.md WARNING #3 的回归
// 防护，必须随 archiver.go 同步维护；移除前请先评估"按 id 删"的语义是否回退到
// "按 cutoff 删"。
func TestLogArchiver_BatchOverflow_OnlyArchivedDeleted(t *testing.T) {
	origNow := nowFunc
	defer func() { nowFunc = origNow }()
	nowFunc = func() time.Time { return time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC) }

	repo := newMemLogRepo()
	mc := newFakeMinIO()

	// 7 条全部早于 cutoff，但 batchSize=3 只能归档前 3 条
	oldTime := time.Date(2025, 8, 15, 10, 0, 0, 0, time.UTC)
	seedLogsForArchive(t, repo, oldTime, 7)

	a := NewLogArchiver(repo, mc, "lic-arch", 6, zap.NewNop())
	a.SetBatchSize(3)

	res, err := a.ArchiveOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, res.LogsArchived, "batchSize=3 限制本 tick 只归档 3 条")
	assert.Equal(t, int64(3), res.LogsDeleted, "DeleteByIDs 仅删本批 3 条")

	// 关键断言：DB 仍有 4 条剩余日志，下个 tick 接力归档
	remaining := repo.snapshot()
	assert.Len(t, remaining, 4,
		"4 条未归档行必须保留；pre-fix DeleteBefore(cutoff) 会把这 4 条也误删丢失")

	// 第二轮：再跑一次 ArchiveOnce 把剩余 3 条归档（不动 nowFunc，cutoff 不变）
	res2, err := a.ArchiveOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, res2.LogsArchived)
	assert.Len(t, repo.snapshot(), 1, "再剩 1 条等下次 tick")

	// 第三轮：清光剩余 1 条
	res3, err := a.ArchiveOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, res3.LogsArchived)
	assert.Empty(t, repo.snapshot())
}

// TestLogArchiver_KeyContainsTickTimestampAndCount — T-0100-P4-B1 显式验证键名格式。
func TestLogArchiver_KeyContainsTickTimestampAndCount(t *testing.T) {
	origNow := nowFunc
	defer func() { nowFunc = origNow }()
	nowFunc = func() time.Time { return time.Date(2026, 5, 10, 3, 15, 30, 0, time.UTC) }

	repo := newMemLogRepo()
	mc := newFakeMinIO()
	seedLogsForArchive(t, repo, time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), 5)

	a := NewLogArchiver(repo, mc, "b", 6, zap.NewNop())
	_, err := a.ArchiveOnce(context.Background())
	require.NoError(t, err)

	keys := mc.listKeysByPrefix("b", "license-logs/2025-08/")
	require.Len(t, keys, 1)
	// tickTS = 20260510T031530Z（UTC，秒级）；count = 5
	assert.Equal(t, "license-logs/2025-08/20260510T031530Z-5.jsonl.gz", keys[0])
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

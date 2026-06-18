package aggregator

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// captureExec 是 WatermarkExecer 的轻量 stub：记录最后一次 Exec 的 SQL/args，可注错。
type captureExec struct {
	calls   int
	lastSQL string
	args    []any
	err     error
}

func (e *captureExec) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	e.calls++
	e.lastSQL = sql
	e.args = args
	return pgconn.NewCommandTag("INSERT 0 1"), e.err
}

// fakeWatermarkStore 是一个内存版水位库，模拟「UPSERT 取 max 不回退 + 读」语义，
// 用于幂等/并发场景断言（不依赖真实 PG）。同时满足 WatermarkExecer + WatermarkQuerier。
type fakeWatermarkStore struct {
	// key = granularity|level → 当前完成格起点
	vals map[string]time.Time
}

func newFakeWatermarkStore() *fakeWatermarkStore {
	return &fakeWatermarkStore{vals: map[string]time.Time{}}
}

func (s *fakeWatermarkStore) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	// args = [granularity, level, bucketStart]
	g := args[0].(string)
	lv := args[1].(string)
	bs := args[2].(time.Time)
	key := g + "|" + lv
	cur, ok := s.vals[key]
	if !ok || bs.After(cur) {
		s.vals[key] = bs // 取 max：仅更晚（更大）才推进
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (s *fakeWatermarkStore) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	g := args[0].(string)
	lv := args[1].(string)
	key := g + "|" + lv
	v, ok := s.vals[key]
	if !ok {
		return errRow{err: pgx.ErrNoRows}
	}
	return &wmRow{gran: g, level: lv, start: v, updated: time.Now()}
}

type wmRow struct {
	gran, level string
	start       time.Time
	updated     time.Time
}

func (r *wmRow) Scan(dest ...any) error {
	*(dest[0].(*string)) = r.gran
	*(dest[1].(*string)) = r.level
	*(dest[2].(*time.Time)) = r.start
	*(dest[3].(*time.Time)) = r.updated
	return nil
}

// ---------------------------------------------------------------------------
// 水位 repository 读写
// ---------------------------------------------------------------------------

func Test_UpsertWatermark_BuildsMaxSemanticsSQL(t *testing.T) {
	exec := &captureExec{}
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	err := UpsertWatermark(context.Background(), exec, metrics.GranularityHourly, WatermarkLevelDevice, start)
	require.NoError(t, err)

	assert.Equal(t, 1, exec.calls)
	// 取 max 不回退：必须用 GREATEST + ON CONFLICT DO UPDATE
	assert.Contains(t, exec.lastSQL, "ON CONFLICT (granularity, level) DO UPDATE")
	assert.Contains(t, exec.lastSQL, "GREATEST(pm_completion_watermarks.completed_bucket_start, EXCLUDED.completed_bucket_start)")
	assert.Equal(t, []any{"hourly", "device", start}, exec.args)
}

func Test_UpsertWatermark_PropagatesError(t *testing.T) {
	exec := &captureExec{err: errors.New("boom")}
	err := UpsertWatermark(context.Background(), exec, metrics.GranularityDaily, WatermarkLevelGroup, time.Now())
	assert.Error(t, err)
}

func Test_WatermarkRepository_Get_NotFound(t *testing.T) {
	repo := NewWatermarkRepository(newFakeWatermarkStore())
	_, err := repo.Get(context.Background(), metrics.GranularityHourly, WatermarkLevelDevice)
	assert.ErrorIs(t, err, ErrWatermarkNotFound)
}

func Test_WatermarkRepository_UpsertThenGet_RoundTrip(t *testing.T) {
	store := newFakeWatermarkStore()
	repo := NewWatermarkRepository(store)
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Upsert(context.Background(), metrics.GranularityHourly, WatermarkLevelDevice, start))

	got, err := repo.Get(context.Background(), metrics.GranularityHourly, WatermarkLevelDevice)
	require.NoError(t, err)
	assert.True(t, got.CompletedBucketStart.Equal(start))
	assert.Equal(t, metrics.GranularityHourly, got.Granularity)
	assert.Equal(t, WatermarkLevelDevice, got.Level)
}

// 幂等 / 重复触发：取 max，旧格不回退。
func Test_WatermarkRepository_Upsert_MaxNoRollback(t *testing.T) {
	store := newFakeWatermarkStore()
	repo := NewWatermarkRepository(store)
	older := time.Date(2026, 5, 22, 9, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)

	// 先推进到 newer
	require.NoError(t, repo.Upsert(context.Background(), metrics.GranularityHourly, WatermarkLevelDevice, newer))
	// 再用 older（乱序补跑旧格）UPSERT —— 不应回退
	require.NoError(t, repo.Upsert(context.Background(), metrics.GranularityHourly, WatermarkLevelDevice, older))
	// 重复用 newer（幂等）
	require.NoError(t, repo.Upsert(context.Background(), metrics.GranularityHourly, WatermarkLevelDevice, newer))

	got, err := repo.Get(context.Background(), metrics.GranularityHourly, WatermarkLevelDevice)
	require.NoError(t, err)
	assert.True(t, got.CompletedBucketStart.Equal(newer), "水位应停在 newer，不被旧格拽回")
}

// ---------------------------------------------------------------------------
// Runner / GroupRunner 成功路径推进水位（含空格也推进）
// ---------------------------------------------------------------------------

func Test_Runner_Run_AdvancesDeviceWatermark_OnSuccess(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	payload, err := BuildPayload(start, end)
	require.NoError(t, err)

	aggrDB := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 5")}
	aggr := New(aggrDB, nil, nil)
	r := NewHourlyRunner(aggr)

	wmStore := newFakeWatermarkStore()
	r.SetWatermarkExec(wmStore)

	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload}
	_, err = r.Run(context.Background(), job)
	require.NoError(t, err)

	repo := NewWatermarkRepository(wmStore)
	got, err := repo.Get(context.Background(), metrics.GranularityHourly, WatermarkLevelDevice)
	require.NoError(t, err)
	assert.True(t, got.CompletedBucketStart.Equal(start), "设备级水位应推进到本格起点")
}

// 空格路径：counter 聚合 0 行（无源数据），水位仍推进——语义是「已处理」非「有数据」。
func Test_Runner_Run_AdvancesDeviceWatermark_OnEmptyBucket(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	payload, _ := BuildPayload(start, end)

	// 空格：Exec 返回 0 行（INSERT 0 0）
	aggrDB := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 0")}
	aggr := New(aggrDB, nil, nil)
	r := NewHourlyRunner(aggr)

	wmStore := newFakeWatermarkStore()
	r.SetWatermarkExec(wmStore)

	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload}
	res, err := r.Run(context.Background(), job)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(res, &m))
	assert.Equal(t, float64(0), m["counter_rows"], "本测试前提是空格（0 行）")

	repo := NewWatermarkRepository(wmStore)
	got, err := repo.Get(context.Background(), metrics.GranularityHourly, WatermarkLevelDevice)
	require.NoError(t, err)
	assert.True(t, got.CompletedBucketStart.Equal(start), "空格也应推进水位，否则下游卡死在空格上")
}

// 水位写失败 → 整个 Run 失败（让 asyncjob 重试；UPSERT 幂等重试安全）。
func Test_Runner_Run_FailsWhenWatermarkUpsertFails(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	payload, _ := BuildPayload(start, end)

	aggrDB := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 5")}
	r := NewHourlyRunner(New(aggrDB, nil, nil))
	r.SetWatermarkExec(&captureExec{err: errors.New("wm boom")})

	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload}
	_, err := r.Run(context.Background(), job)
	assert.Error(t, err)
}

// 未注入水位执行体时不写水位、不报错（退化为既有行为，不回归）。
func Test_Runner_Run_NoWatermarkExec_IsNoop(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	payload, _ := BuildPayload(start, end)

	r := NewHourlyRunner(New(&stubDB{execTag: pgconn.NewCommandTag("INSERT 0 5")}, nil, nil))
	// 不调 SetWatermarkExec
	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload}
	_, err := r.Run(context.Background(), job)
	require.NoError(t, err)
}

func Test_GroupRunner_Run_AdvancesGroupWatermark_OnSuccess(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	payload, _ := BuildPayload(start, end)

	aggrDB := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 7")}
	gr := NewHourlyGroupRunner(New(aggrDB, nil, nil))

	wmStore := newFakeWatermarkStore()
	gr.SetWatermarkExec(wmStore)

	job := &asyncjob.Job{ID: uuid.New(), JobType: gr.JobType(), Payload: payload}
	_, err := gr.Run(context.Background(), job)
	require.NoError(t, err)

	repo := NewWatermarkRepository(wmStore)
	got, err := repo.Get(context.Background(), metrics.GranularityHourly, WatermarkLevelGroup)
	require.NoError(t, err)
	assert.True(t, got.CompletedBucketStart.Equal(start), "组级水位应推进到本格起点")
}

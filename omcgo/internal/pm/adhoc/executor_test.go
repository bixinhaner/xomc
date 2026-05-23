package adhoc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// ---------------------------------------------------------------------------
// Stub 依赖
// ---------------------------------------------------------------------------

type stubAggr struct {
	rowsByGran map[metrics.Granularity][]aggregator.Row
}

func (s *stubAggr) Query(_ context.Context, req aggregator.QueryRequest) ([]aggregator.Row, error) {
	return s.rowsByGran[req.Granularity], nil
}

type stubRepo struct {
	mu           sync.Mutex
	insertedRows []ResultRow
	statusUpdates []struct {
		id       uuid.UUID
		status   Status
		progress *int
	}
}

func (s *stubRepo) Create(context.Context, CreateRequest) (uuid.UUID, error) { return uuid.Nil, nil }
func (s *stubRepo) Get(context.Context, uuid.UUID) (*Task, error)            { return nil, nil }
func (s *stubRepo) List(context.Context, ListFilter) ([]Task, error)         { return nil, nil }
func (s *stubRepo) Cancel(context.Context, uuid.UUID) error                  { return nil }
func (s *stubRepo) LockNextPending(context.Context, string) (*Task, error)   { return nil, nil }

func (s *stubRepo) UpdateStatus(_ context.Context, id uuid.UUID, status Status, progress *int, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusUpdates = append(s.statusUpdates, struct {
		id       uuid.UUID
		status   Status
		progress *int
	}{id, status, progress})
	return nil
}

func (s *stubRepo) InsertResults(_ context.Context, rows []ResultRow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.insertedRows = append(s.insertedRows, rows...)
	return nil
}

type stubPublisher struct {
	mu     sync.Mutex
	events []struct {
		subject string
		payload any
	}
}

func (s *stubPublisher) Publish(_ context.Context, subject string, payload any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, struct {
		subject string
		payload any
	}{subject, payload})
	return nil
}

// ---------------------------------------------------------------------------
// 3 case：ExecuteOneshot 主路径
// ---------------------------------------------------------------------------

func Test_Executor_ExecuteOneshot_SingleGranularity(t *testing.T) {
	stype := metrics.StatisSum
	aggr := &stubAggr{
		rowsByGran: map[metrics.Granularity][]aggregator.Row{
			metrics.GranularityHourly: {
				{
					DeviceOUI: "A", DeviceSN: "S1",
					MetricPath: "L.Cell.Avail.Dur", MetricType: metrics.MetricTypeCounter,
					MetricValue: 700, StatisType: &stype, Granularity: metrics.GranularityHourly,
					Time: time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
				},
			},
		},
	}
	repo := &stubRepo{}
	pub := &stubPublisher{}
	e := NewExecutor(aggr, repo, pub, nil)

	task := &Task{
		ID:            uuid.New(),
		Granularities: []string{"hourly"},
		DeviceSNs:     []string{"S1"},
		MetricPaths:   []string{"L.Cell.Avail.Dur"},
		WindowStart:   time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}

	n, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Len(t, repo.insertedRows, 1)
	assert.Equal(t, task.ID, repo.insertedRows[0].TaskID)
	assert.Equal(t, "sum", *repo.insertedRows[0].StatisType)

	// 1 个粒度 → 100% progress 一次
	require.Len(t, repo.statusUpdates, 1)
	assert.Equal(t, 100, *repo.statusUpdates[0].progress)

	// 1 个进度事件
	require.Len(t, pub.events, 1)
	assert.Equal(t, SubjectProgress, pub.events[0].subject)
}

func Test_Executor_ExecuteOneshot_MultiGranularityProgress(t *testing.T) {
	aggr := &stubAggr{
		rowsByGran: map[metrics.Granularity][]aggregator.Row{
			metrics.GranularityHourly: {{DeviceOUI: "A", DeviceSN: "S1", MetricPath: "X", MetricType: metrics.MetricTypeCounter, MetricValue: 1, Granularity: metrics.GranularityHourly}},
			metrics.GranularityDaily:  {{DeviceOUI: "A", DeviceSN: "S1", MetricPath: "X", MetricType: metrics.MetricTypeCounter, MetricValue: 2, Granularity: metrics.GranularityDaily}},
			metrics.GranularityWeekly: {{DeviceOUI: "A", DeviceSN: "S1", MetricPath: "X", MetricType: metrics.MetricTypeCounter, MetricValue: 3, Granularity: metrics.GranularityWeekly}},
		},
	}
	repo := &stubRepo{}
	pub := &stubPublisher{}
	e := NewExecutor(aggr, repo, pub, nil)

	task := &Task{
		ID:            uuid.New(),
		Granularities: []string{"hourly", "daily", "weekly"},
		DeviceSNs:     []string{"S1"},
		WindowStart:   time.Now().Add(-time.Hour),
		WindowEnd:     time.Now(),
	}

	n, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Len(t, repo.insertedRows, 3)

	// 3 个粒度 → 3 次 progress 上报 (33/66/99? or 33/66/100? 100/3*1=33, 100*2/3=66, 100*3/3=100)
	require.Len(t, repo.statusUpdates, 3)
	assert.Equal(t, 33, *repo.statusUpdates[0].progress)
	assert.Equal(t, 66, *repo.statusUpdates[1].progress)
	assert.Equal(t, 100, *repo.statusUpdates[2].progress)

	require.Len(t, pub.events, 3)
}

func Test_Executor_ExecuteOneshot_NoGranularitiesError(t *testing.T) {
	e := NewExecutor(&stubAggr{}, &stubRepo{}, nil, nil)
	task := &Task{ID: uuid.New(), Granularities: nil}
	_, err := e.ExecuteOneshot(context.Background(), task)
	assert.Error(t, err)
}

package regularreport

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type schedulerRepoStub struct {
	templates []Template
	requests  []EnsureRunRequest
}

func (s *schedulerRepoStub) ListEnabledTemplates(context.Context) ([]Template, error) {
	return s.templates, nil
}

func (s *schedulerRepoStub) EnsureRun(_ context.Context, request EnsureRunRequest) (bool, error) {
	s.requests = append(s.requests, request)
	return true, nil
}

func TestSchedulerCreatesOneRunPerSelectedPeriod(t *testing.T) {
	t.Parallel()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	repo := &schedulerRepoStub{templates: []Template{{
		ID: uuid.New(), Name: "Radio KPI", CreatorID: uuid.New(),
		Payload: []byte(`{"device_sns":["SN-1"],"metric_paths":["K-1"]}`),
		Config:  Config{Enabled: true, SendTime: "08:30", Periods: []Period{PeriodHourly, PeriodDaily}, EmailEnabled: true, Recipients: []string{"ops@example.com"}},
	}}}
	scheduler := NewScheduler(repo, func() *time.Location { return location }, nil)
	now := time.Date(2026, 8, 17, 8, 31, 0, 0, location)

	require.NoError(t, scheduler.ScheduleOnce(context.Background(), now))
	require.Len(t, repo.requests, 2)
	require.Equal(t, PeriodHourly, repo.requests[0].Period)
	require.Equal(t, "2026-08-17 07:00", repo.requests[0].WindowStart.In(location).Format("2006-01-02 15:04"))
	require.Equal(t, PeriodDaily, repo.requests[1].Period)
	require.Equal(t, "2026-08-16 00:00", repo.requests[1].WindowStart.In(location).Format("2006-01-02 15:04"))
}

func TestSchedulerSkipsBeforeSendTime(t *testing.T) {
	t.Parallel()
	repo := &schedulerRepoStub{templates: []Template{{
		ID: uuid.New(), Payload: []byte(`{}`),
		Config: Config{Enabled: true, SendTime: "08:30", Periods: []Period{PeriodDaily}, EmailEnabled: true, Recipients: []string{"ops@example.com"}},
	}}}
	scheduler := NewScheduler(repo, func() *time.Location { return time.UTC }, nil)
	require.NoError(t, scheduler.ScheduleOnce(context.Background(), time.Date(2026, 8, 17, 8, 29, 0, 0, time.UTC)))
	require.Empty(t, repo.requests)
}

func TestSchedulerWaitsForPMCloseGrace(t *testing.T) {
	t.Parallel()
	repo := &schedulerRepoStub{templates: []Template{{
		ID: uuid.New(), Payload: []byte(`{}`),
		Config: Config{Enabled: true, SendTime: "08:00", Periods: []Period{Period15Min, PeriodHourly}, EmailEnabled: true, Recipients: []string{"ops@example.com"}},
	}}}
	scheduler := NewScheduler(repo, func() *time.Location { return time.UTC }, nil).
		SetReadinessGrace(12*time.Minute, 15*time.Minute)

	require.NoError(t, scheduler.ScheduleOnce(context.Background(), time.Date(2026, 8, 17, 8, 11, 0, 0, time.UTC)))
	require.Empty(t, repo.requests)

	require.NoError(t, scheduler.ScheduleOnce(context.Background(), time.Date(2026, 8, 17, 8, 12, 0, 0, time.UTC)))
	require.Len(t, repo.requests, 2)
}

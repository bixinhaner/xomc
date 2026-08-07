package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/stationlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeWorkerJobRepo struct {
	inserts []asyncjob.InsertRequest
}

func (f *fakeWorkerJobRepo) Insert(_ context.Context, req asyncjob.InsertRequest) (uuid.UUID, error) {
	f.inserts = append(f.inserts, req)
	return uuid.New(), nil
}

func (f *fakeWorkerJobRepo) GetByID(context.Context, uuid.UUID) (*asyncjob.Job, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeWorkerJobRepo) LockNextPending(context.Context, string, string) (*asyncjob.Job, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeWorkerJobRepo) UpdateHeartbeat(context.Context, uuid.UUID) error {
	return errors.New("not implemented")
}

func (f *fakeWorkerJobRepo) MarkSucceeded(context.Context, uuid.UUID, json.RawMessage) error {
	return errors.New("not implemented")
}

func (f *fakeWorkerJobRepo) MarkFailed(context.Context, uuid.UUID, string) error {
	return errors.New("not implemented")
}

func (f *fakeWorkerJobRepo) ListZombies(context.Context, time.Duration) ([]asyncjob.Job, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeWorkerJobRepo) ResetZombie(context.Context, uuid.UUID) error {
	return errors.New("not implemented")
}

type fakeWorkerCronStateRepo struct {
	state   *asyncjob.CronState
	upserts []struct {
		jobType string
		spec    string
		end     time.Time
	}
}

func (f *fakeWorkerCronStateRepo) Get(context.Context, string) (*asyncjob.CronState, error) {
	if f.state == nil {
		return nil, asyncjob.ErrNoCronState
	}
	return f.state, nil
}

func (f *fakeWorkerCronStateRepo) Upsert(_ context.Context, jobType, cronExpr string, lastTriggeredAt, lastBucketEnd time.Time) error {
	end := lastBucketEnd
	f.state = &asyncjob.CronState{
		JobType:         jobType,
		CronExpr:        cronExpr,
		LastTriggeredAt: lastTriggeredAt,
		LastBucketEnd:   &end,
	}
	f.upserts = append(f.upserts, struct {
		jobType string
		spec    string
		end     time.Time
	}{jobType: jobType, spec: cronExpr, end: end})
	return nil
}

func TestStationLogCleanupCronEntry_UsesConfiguredIntervalWindow(t *testing.T) {
	loc := time.UTC
	entry := stationLogCleanupCronEntry(30, func() *time.Location { return loc })
	now := time.Date(2026, 8, 7, 10, 37, 0, 0, time.UTC)

	start, end := entry.window(now)

	require.Equal(t, stationlog.JobTypeStationLogCleanup, entry.jobType)
	require.Equal(t, "@every 30m", entry.spec)
	require.True(t, start.Equal(time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)))
	require.True(t, end.Equal(time.Date(2026, 8, 7, 10, 30, 0, 0, time.UTC)))
	require.True(t, entry.advance(end).Equal(time.Date(2026, 8, 7, 11, 0, 0, 0, time.UTC)))
}

func TestStationLogCleanupCronEntry_InvalidIntervalFallsBackToDefault(t *testing.T) {
	loc := time.UTC
	entry := stationLogCleanupCronEntry(9, func() *time.Location { return loc })
	now := time.Date(2026, 8, 7, 10, 37, 0, 0, time.UTC)

	start, end := entry.window(now)

	require.Equal(t, "@every 60m", entry.spec)
	require.True(t, start.Equal(time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)))
	require.True(t, end.Equal(time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)))
}

func TestTriggerStationLogCleanupIfDue_UsesCronStateAndBucketDedupeWindow(t *testing.T) {
	ctx := context.Background()
	jobRepo := &fakeWorkerJobRepo{}
	stateRepo := &fakeWorkerCronStateRepo{}
	loc := time.UTC
	logger := zap.NewNop()
	now := time.Date(2026, 8, 7, 10, 37, 0, 0, time.UTC)

	triggered := triggerStationLogCleanupIfDue(ctx, jobRepo, stateRepo, 30, now, logger, func() *time.Location { return loc })
	require.True(t, triggered)
	require.Len(t, jobRepo.inserts, 1)
	require.Equal(t, stationlog.JobTypeStationLogCleanup, jobRepo.inserts[0].JobType)
	require.NotNil(t, jobRepo.inserts[0].BucketStart)
	require.NotNil(t, jobRepo.inserts[0].BucketEnd)
	require.True(t, jobRepo.inserts[0].BucketStart.Equal(time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)))
	require.True(t, jobRepo.inserts[0].BucketEnd.Equal(time.Date(2026, 8, 7, 10, 30, 0, 0, time.UTC)))
	require.Len(t, stateRepo.upserts, 1)
	require.Equal(t, "@every 30m", stateRepo.upserts[0].spec)

	triggered = triggerStationLogCleanupIfDue(ctx, jobRepo, stateRepo, 30, now.Add(10*time.Minute), logger, func() *time.Location { return loc })
	require.False(t, triggered)
	require.Len(t, jobRepo.inserts, 1)

	triggered = triggerStationLogCleanupIfDue(ctx, jobRepo, stateRepo, 30, time.Date(2026, 8, 7, 11, 1, 0, 0, time.UTC), logger, func() *time.Location { return loc })
	require.True(t, triggered)
	require.Len(t, jobRepo.inserts, 2)
	require.True(t, jobRepo.inserts[1].BucketStart.Equal(time.Date(2026, 8, 7, 10, 30, 0, 0, time.UTC)))
	require.True(t, jobRepo.inserts[1].BucketEnd.Equal(time.Date(2026, 8, 7, 11, 0, 0, 0, time.UTC)))
}

func TestTriggerStationLogCleanupIfDue_SeventyMinuteIntervalDoesNotResetAtMidnight(t *testing.T) {
	ctx := context.Background()
	jobRepo := &fakeWorkerJobRepo{}
	stateRepo := &fakeWorkerCronStateRepo{}
	loc := time.UTC
	logger := zap.NewNop()

	triggered := triggerStationLogCleanupIfDue(
		ctx,
		jobRepo,
		stateRepo,
		70,
		time.Date(1970, 1, 1, 23, 21, 0, 0, time.UTC),
		logger,
		func() *time.Location { return loc },
	)
	require.True(t, triggered)
	require.Len(t, jobRepo.inserts, 1)
	require.True(t, jobRepo.inserts[0].BucketStart.Equal(time.Date(1970, 1, 1, 22, 10, 0, 0, time.UTC)))
	require.True(t, jobRepo.inserts[0].BucketEnd.Equal(time.Date(1970, 1, 1, 23, 20, 0, 0, time.UTC)))

	triggered = triggerStationLogCleanupIfDue(
		ctx,
		jobRepo,
		stateRepo,
		70,
		time.Date(1970, 1, 2, 0, 1, 0, 0, time.UTC),
		logger,
		func() *time.Location { return loc },
	)
	require.False(t, triggered, "23:20 后到 00:01 只隔 41 分钟，不能因跨午夜短周期触发")
	require.Len(t, jobRepo.inserts, 1)

	triggered = triggerStationLogCleanupIfDue(
		ctx,
		jobRepo,
		stateRepo,
		70,
		time.Date(1970, 1, 2, 0, 31, 0, 0, time.UTC),
		logger,
		func() *time.Location { return loc },
	)
	require.True(t, triggered)
	require.Len(t, jobRepo.inserts, 2)
	require.True(t, jobRepo.inserts[1].BucketStart.Equal(time.Date(1970, 1, 1, 23, 20, 0, 0, time.UTC)))
	require.True(t, jobRepo.inserts[1].BucketEnd.Equal(time.Date(1970, 1, 2, 0, 30, 0, 0, time.UTC)))
}

package aggregator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

func TestParseLateDataWindow(t *testing.T) {
	assert.Equal(t, 7*24*time.Hour, ParseLateDataWindow(""))
	assert.Equal(t, 48*time.Hour, ParseLateDataWindow("48h"))
	assert.Equal(t, 7*24*time.Hour, ParseLateDataWindow("-1h"))
	assert.Equal(t, 7*24*time.Hour, ParseLateDataWindow("bad"))
}

func TestEligibleChunkSQLRequiresCleanActiveHourlyVersions(t *testing.T) {
	compressSQL := buildEligibleChunkSQL(true)
	assert.Contains(t, compressSQL, "v.status='active'")
	assert.Contains(t, compressSQL, "v.dirty=false")
	assert.Contains(t, compressSQL, "generate_series")
	assert.Contains(t, compressSQL, "NOT c.is_compressed")

	dropSQL := buildEligibleChunkSQL(false)
	assert.NotContains(t, dropSQL, "NOT c.is_compressed")
}

func TestDirtyBucketSQLRequeuesAllDirtyBuckets(t *testing.T) {
	sql := buildDirtyBucketSQL()
	assert.Contains(t, sql, "status='active'")
	assert.Contains(t, sql, "dirty=true")
	assert.NotContains(t, sql, "bucket_end >=")
}

func TestStaleBuildingCleanupOnlySelectsTimedOutVersions(t *testing.T) {
	sql := buildStaleBuildingVersionsSQL()
	assert.Contains(t, sql, "status='building'")
	assert.Contains(t, sql, "created_at < now() - $1::interval")
	assert.Contains(t, sql, "LIMIT")

	updateSQL := buildFailStaleBuildingVersionSQL()
	assert.Contains(t, updateSQL, "SET status='failed'")
	assert.Contains(t, updateSQL, "status='building'")
}

func TestRecoveryMetricsExposeBoundedRecoveryAndWatermarkFailures(t *testing.T) {
	m := NewMetrics(nil)
	m.IncHourlyRecovery()
	m.SetRecoveryExhaustedBuckets(2)
	m.SetFailedBuckets(3)
	m.SetFailedVersions(4)
	m.SetStaleBuildingVersions(5)
	m.SetWatermarkLag(3 * time.Hour)

	require.Equal(t, float64(1), testutil.ToFloat64(m.HourlyRecoveries))
	require.Equal(t, float64(2), testutil.ToFloat64(m.RecoveryExhaustedBuckets))
	require.Equal(t, float64(3), testutil.ToFloat64(m.FailedBuckets))
	require.Equal(t, float64(4), testutil.ToFloat64(m.FailedVersions))
	require.Equal(t, float64(5), testutil.ToFloat64(m.StaleBuildingVersions))
	require.Equal(t, (3 * time.Hour).Seconds(), testutil.ToFloat64(m.WatermarkLag))
}

func TestRecoverFailedHourlyBucketsRequeuesOnlyRetriableSourceBucketWithoutActiveVersion(t *testing.T) {
	start := time.Now().UTC().Truncate(time.Hour).Add(-4 * time.Hour)
	bucket := func(offset time.Duration, message string, recoveries int) asyncjob.Job {
		s := start.Add(offset)
		e := s.Add(time.Hour)
		return asyncjob.Job{
			ID: uuid.New(), JobType: JobTypeHourly, Status: asyncjob.StatusFailed,
			BucketStart: &s, BucketEnd: &e, ErrorMessage: message,
			RecoveryCount: recoveries,
		}
	}
	eligible := bucket(0, "deadlock detected (SQLSTATE 40P01)", 0)
	nonRetriable := bucket(time.Hour, "violates check constraint (SQLSTATE 23514)", 0)
	exhausted := bucket(2*time.Hour, "serialization failure (SQLSTATE 40001)", DefaultHourlyRecoveryMax)
	alreadyActive := bucket(3*time.Hour, "serialization failure (SQLSTATE 40001)", 0)
	repo := &maintenanceRecoveryRepo{
		failed: []asyncjob.Job{eligible, nonRetriable, exhausted, alreadyActive},
	}
	db := &maintenanceTestDB{
		queryRows: []pgx.Row{
			recordingFormulaRow{values: []any{true, false}},
			recordingFormulaRow{values: []any{true, true}},
		},
	}
	m := NewMetrics(nil)

	err := recoverFailedHourlyBuckets(
		context.Background(), db, repo, DefaultLateDataWindow, m, zap.NewNop(),
	)

	require.NoError(t, err)
	require.Len(t, repo.recoveryRequests, 1)
	assert.Equal(t, *eligible.BucketStart, repo.recoveryRequests[0].BucketStart)
	assert.Equal(t, DefaultHourlyRecoveryMax, repo.recoveryRequests[0].MaxRecoveries)
	assert.Equal(t, DefaultHourlyRecoveryCooldown, repo.recoveryRequests[0].Cooldown)
	assert.Equal(t, float64(1), testutil.ToFloat64(m.HourlyRecoveries))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.RecoveryExhaustedBuckets))
	assert.Equal(t, float64(4), testutil.ToFloat64(m.FailedBuckets))
}

func TestFailStaleBuildingVersionsLeavesPendingAndRunningJobsAlone(t *testing.T) {
	start := time.Now().UTC().Truncate(time.Hour).Add(-4 * time.Hour)
	versions := [][]any{
		{int64(11), start, start.Add(time.Hour)},
		{int64(12), start.Add(time.Hour), start.Add(2 * time.Hour)},
		{int64(13), start.Add(2 * time.Hour), start.Add(3 * time.Hour)},
	}
	repo := &maintenanceRecoveryRepo{
		find: map[time.Time]*asyncjob.Job{
			start.Add(time.Hour): {
				Status: asyncjob.StatusFailed,
			},
			start.Add(2 * time.Hour): {
				Status: asyncjob.StatusRunning,
			},
		},
	}
	db := &maintenanceTestDB{
		rows: []pgx.Rows{&recordingFormulaRows{rows: versions}},
	}

	err := failStaleBuildingVersions(
		context.Background(), db, repo, DefaultStaleBuildingTimeout, zap.NewNop(),
	)

	require.NoError(t, err)
	assert.Equal(t, []int64{11, 12}, db.failedVersions,
		"absent and terminal jobs should fail stale versions; running jobs should remain building")
}

type maintenanceRecoveryRepo struct {
	failed           []asyncjob.Job
	find             map[time.Time]*asyncjob.Job
	recoveryRequests []asyncjob.FailedBucketRecoveryRequest
}

func (r *maintenanceRecoveryRepo) Insert(
	context.Context,
	asyncjob.InsertRequest,
) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (r *maintenanceRecoveryRepo) ListFailedNaturalBuckets(
	context.Context,
	string,
	time.Time,
	int,
) ([]asyncjob.Job, error) {
	return r.failed, nil
}

func (r *maintenanceRecoveryRepo) FindNaturalBucketJob(
	_ context.Context,
	_ string,
	start, _ time.Time,
) (*asyncjob.Job, bool, error) {
	job, ok := r.find[start]
	return job, ok, nil
}

func (r *maintenanceRecoveryRepo) RequeueRetriableFailedBucket(
	_ context.Context,
	req asyncjob.FailedBucketRecoveryRequest,
) (uuid.UUID, bool, error) {
	r.recoveryRequests = append(r.recoveryRequests, req)
	return uuid.New(), true, nil
}

type maintenanceTestDB struct {
	rows           []pgx.Rows
	queryRows      []pgx.Row
	failedVersions []int64
}

func (db *maintenanceTestDB) Exec(
	_ context.Context,
	sql string,
	args ...any,
) (pgconn.CommandTag, error) {
	if sql == buildFailStaleBuildingVersionSQL() {
		version, ok := args[0].(int64)
		if !ok {
			return pgconn.CommandTag{}, fmt.Errorf("unexpected version type %T", args[0])
		}
		db.failedVersions = append(db.failedVersions, version)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	return pgconn.CommandTag{}, fmt.Errorf("unexpected exec: %s", sql)
}

func (db *maintenanceTestDB) Query(
	_ context.Context,
	_ string,
	_ ...any,
) (pgx.Rows, error) {
	if len(db.rows) == 0 {
		return nil, fmt.Errorf("unexpected query")
	}
	rows := db.rows[0]
	db.rows = db.rows[1:]
	return rows, nil
}

func (db *maintenanceTestDB) QueryRow(
	_ context.Context,
	_ string,
	_ ...any,
) pgx.Row {
	if len(db.queryRows) == 0 {
		return recordingFormulaRow{err: fmt.Errorf("unexpected query row")}
	}
	row := db.queryRows[0]
	db.queryRows = db.queryRows[1:]
	return row
}

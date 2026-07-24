package aggregator

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

func TestVersionedHourlyValidateWindow(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 30, 0, 0, time.UTC)
	valid := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 7, 24, 11, 0, 0, 0, time.UTC),
	}
	require.NoError(t, validateVersionedHourlyWindow(valid, now))

	open := valid
	open.End = now.Add(time.Minute)
	open.Start = open.End.Add(-time.Hour)
	assert.ErrorContains(t, validateVersionedHourlyWindow(open, now), "closed")

	wrongDuration := valid
	wrongDuration.End = wrongDuration.Start.Add(2 * time.Hour)
	assert.ErrorContains(t, validateVersionedHourlyWindow(wrongDuration, now), "exactly one hour")

	wrongGranularity := valid
	wrongGranularity.Granularity = metrics.GranularityDaily
	assert.ErrorContains(t, validateVersionedHourlyWindow(wrongGranularity, now), "hourly granularity")
}

func TestVersionedHourlyBatchSQLIsBoundedAndAppendOnly(t *testing.T) {
	sql := buildVersionedHourlyBatchSQL()
	assert.Contains(t, sql, "a.device_dim_id = ANY($4::uuid[])")
	assert.Contains(t, sql, "INSERT INTO pm_hourly_anchors")
	assert.Contains(t, sql, "INSERT INTO pm_hourly_values")
	assert.Contains(t, sql, "pm_hourly_rollup_batches")
	valueInsert := sql[strings.Index(sql, "INSERT INTO pm_hourly_values"):]
	assert.NotContains(t, valueInsert, "ON CONFLICT")
	assert.NotContains(t, sql, "work_mem")
}

func TestVersionedHourlyPublicationSQLIsAtomicStateSwitch(t *testing.T) {
	assert.Contains(t, buildSupersedeHourlyVersionSQL(), "status = 'superseded'")
	assert.Contains(t, buildPublishHourlyVersionSQL(), "status = 'active'")
	assert.NotContains(t, buildPublishHourlyVersionSQL(), "ON CONFLICT")
}

func TestVersionedHourlyAdvisoryLockKeyIsStablePerBucket(t *testing.T) {
	start := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, hourlyAdvisoryLockKey(start), hourlyAdvisoryLockKey(start))
	assert.NotEqual(t, hourlyAdvisoryLockKey(start), hourlyAdvisoryLockKey(start.Add(time.Hour)))
}

func TestVersionedHourlyDeviceBatchCount(t *testing.T) {
	devices := make([]uuid.UUID, 401)
	for i := range devices {
		devices[i] = uuid.New()
	}
	batches := splitDeviceBatches(devices, 200)
	require.Len(t, batches, 3)
	assert.Len(t, batches[0], 200)
	assert.Len(t, batches[1], 200)
	assert.Len(t, batches[2], 1)
}

func TestParseHourlyBatchDevices(t *testing.T) {
	assert.Equal(t, 200, ParseHourlyBatchDevices(""))
	assert.Equal(t, 350, ParseHourlyBatchDevices("350"))
	assert.Equal(t, 200, ParseHourlyBatchDevices("0"))
	assert.Equal(t, 200, ParseHourlyBatchDevices("invalid"))
}

package aggregator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

package export

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverMetricColumns_UsesRequestedMetricPaths(t *testing.T) {
	keys, err := discoverMetricColumns(context.Background(), nil, "pm_metrics", []string{"K002", "C001", "K002", ""}, zeroTime(), zeroTime())
	require.NoError(t, err)

	assert.Equal(t, []colKey{
		{code: "K002", mtype: "kpi"},
		{code: "C001", mtype: "counter"},
	}, keys)
}

func zeroTime() (t time.Time) {
	return t
}

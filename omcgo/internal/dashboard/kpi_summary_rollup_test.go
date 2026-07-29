package dashboard

import (
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestBuildKPIOverviewFromNetworkRollupsSkipsCrossTechnologyConflicts(t *testing.T) {
	now := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	got, conflicts := buildKPIOverviewFromNetworkRollups([]NetworkRollupPoint{
		{Technology: model.TechLTE, MetricPath: "K-LTE", WindowStart: now, Value: jsonx.Float(11)},
		{Technology: model.TechNR, MetricPath: "K-NR", WindowStart: now, Value: jsonx.Float(22)},
		{Technology: model.TechLTE, MetricPath: "K-CONFLICT", WindowStart: now, Value: jsonx.Float(1)},
		{Technology: model.TechGSM, MetricPath: "K-CONFLICT", WindowStart: now, Value: jsonx.Float(2)},
	})

	assert.Equal(t, map[string]float64{"K-LTE": 11, "K-NR": 22}, got)
	assert.Equal(t, []string{"K-CONFLICT"}, conflicts)
}

func TestBuildKPIOverviewFromNetworkRollupsKeepsLatestSameTechnologyPoint(t *testing.T) {
	start := time.Date(2026, 7, 28, 22, 0, 0, 0, time.UTC)
	got, conflicts := buildKPIOverviewFromNetworkRollups([]NetworkRollupPoint{
		{Technology: model.TechLTE, MetricPath: "K1", WindowStart: start, Value: jsonx.Float(10)},
		{Technology: model.TechLTE, MetricPath: "K1", WindowStart: start.Add(time.Hour), Value: jsonx.Float(20)},
	})

	assert.Equal(t, map[string]float64{"K1": 20}, got)
	assert.Empty(t, conflicts)
}

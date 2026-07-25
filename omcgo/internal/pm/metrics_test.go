package pm

import (
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestPMMetrics_DiscoveredAndDeprecatedAliasShareConcurrentSnapshot(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewPMMetrics(reg)
	counter := m.DiscoveredCountersTotal.WithLabelValues("cmcc", "lte", "whitelist_miss")
	counter.Inc()

	stop := make(chan struct{})
	var updater sync.WaitGroup
	updater.Add(1)
	go func() {
		defer updater.Done()
		for {
			select {
			case <-stop:
				return
			default:
				counter.Inc()
			}
		}
	}()
	defer func() {
		close(stop)
		updater.Wait()
	}()

	for i := 0; i < 100; i++ {
		families, err := reg.Gather()
		require.NoError(t, err)

		values := make(map[string]float64, 2)
		for _, family := range families {
			switch family.GetName() {
			case "omc_pm_discovered_counters_total", "omc_pm_dropped_counters_total":
				require.Len(t, family.Metric, 1)
				values[family.GetName()] = family.Metric[0].GetCounter().GetValue()
			}
		}
		require.Contains(t, values, "omc_pm_discovered_counters_total")
		require.Contains(t, values, "omc_pm_dropped_counters_total")
		require.Equal(t,
			values["omc_pm_discovered_counters_total"],
			values["omc_pm_dropped_counters_total"],
			"deprecated alias must be emitted from the same gathered snapshot",
		)
	}
}

// G4-Gap-1: 验证 ReportDelaySeconds histogram 注册成功且能按 carrier×technology 维度记录。
func TestPMMetrics_ReportDelaySeconds(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	m := NewPMMetrics(reg)

	require.NotNil(t, m.ReportDelaySeconds, "ReportDelaySeconds histogram must be initialized")

	m.ReportDelaySeconds.WithLabelValues("cmcc", "lte").Observe(120.5)
	m.ReportDelaySeconds.WithLabelValues("cmcc", "lte").Observe(450.0)
	m.ReportDelaySeconds.WithLabelValues("ctcc", "nr").Observe(60.0)

	expected := `
# HELP omc_pm_report_delay_seconds PM file report delay = ingest_time - end_time (seconds). Negative values indicate device clock ahead of OMC.
# TYPE omc_pm_report_delay_seconds histogram
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="30"} 0
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="60"} 0
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="120"} 0
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="300"} 1
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="600"} 2
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="900"} 2
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="1800"} 2
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="3600"} 2
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="7200"} 2
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="14400"} 2
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="28800"} 2
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="86400"} 2
omc_pm_report_delay_seconds_bucket{carrier="cmcc",technology="lte",le="+Inf"} 2
omc_pm_report_delay_seconds_sum{carrier="cmcc",technology="lte"} 570.5
omc_pm_report_delay_seconds_count{carrier="cmcc",technology="lte"} 2
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="30"} 0
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="60"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="120"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="300"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="600"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="900"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="1800"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="3600"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="7200"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="14400"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="28800"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="86400"} 1
omc_pm_report_delay_seconds_bucket{carrier="ctcc",technology="nr",le="+Inf"} 1
omc_pm_report_delay_seconds_sum{carrier="ctcc",technology="nr"} 60
omc_pm_report_delay_seconds_count{carrier="ctcc",technology="nr"} 1
`

	err := testutil.CollectAndCompare(m.ReportDelaySeconds, strings.NewReader(expected), "omc_pm_report_delay_seconds")
	require.NoError(t, err)
}

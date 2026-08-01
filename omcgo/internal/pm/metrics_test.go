package pm

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestPMMetrics_FilteringReasonsUseIndependentCounters(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewPMMetrics(reg)

	m.WhitelistMissValuesTotal.WithLabelValues("cmcc", "lte").Add(2)
	m.KnownDisabledValuesTotal.WithLabelValues("cmcc", "lte").Add(3)
	m.TechnologyMismatchFilesTotal.WithLabelValues("lte", "gsm").Inc()

	expected := `
# HELP omc_pm_known_disabled_values_total PM counter values removed because their registered indicator is disabled.
# TYPE omc_pm_known_disabled_values_total counter
omc_pm_known_disabled_values_total{carrier="cmcc",technology="lte"} 3
# HELP omc_pm_technology_mismatch_files_total PM files quarantined because XML evidence conflicts with the declared device technology.
# TYPE omc_pm_technology_mismatch_files_total counter
omc_pm_technology_mismatch_files_total{declared_technology="lte",detected_technology="gsm"} 1
# HELP omc_pm_whitelist_miss_values_total PM counter values removed because the vendor report key is not registered in the routed indicator library.
# TYPE omc_pm_whitelist_miss_values_total counter
omc_pm_whitelist_miss_values_total{carrier="cmcc",technology="lte"} 2
`
	require.NoError(t, testutil.GatherAndCompare(
		reg,
		strings.NewReader(expected),
		"omc_pm_whitelist_miss_values_total",
		"omc_pm_known_disabled_values_total",
		"omc_pm_technology_mismatch_files_total",
	))
}

func TestPMMetrics_DiscardedFilesUsesReasonOnly(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewPMMetrics(reg)

	m.FilesDiscardedTotal.WithLabelValues("device_not_registered").Inc()

	expected := `
# HELP omc_pm_files_discarded_total PM files intentionally discarded before parsing, classified by bounded reason.
# TYPE omc_pm_files_discarded_total counter
omc_pm_files_discarded_total{reason="device_not_registered"} 1
`
	require.NoError(t, testutil.GatherAndCompare(
		reg,
		strings.NewReader(expected),
		"omc_pm_files_discarded_total",
	))
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

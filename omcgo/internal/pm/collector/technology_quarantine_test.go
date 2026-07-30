package collector

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/pm"
)

func TestCheckTechnologyConsistencyQuarantinesMismatchBeforeRouting(t *testing.T) {
	reg := prometheus.NewRegistry()
	store := &recordingQuarantineStore{}
	c := &PMCollector{
		quarantineStore: store,
		metrics:         pm.NewPMMetrics(reg),
		logger:          zap.NewNop(),
	}
	payload := &FileReceivedPayload{
		MinIOPath:  "pm/20260729/SN-1.xml.gz",
		DeviceSN:   "SN-1",
		Carrier:    "cmcc",
		Technology: "lte",
	}
	content := &PMFileContent{Counters: countersNamed(
		"BTS.ActiveSeconds",
		"SDCCH.Attempts",
		"TCH.SeizureAttempts",
	)}

	quarantined, err := c.checkTechnologyConsistency(context.Background(), payload, content)

	require.NoError(t, err)
	require.True(t, quarantined)
	require.Len(t, store.records, 1)
	require.Equal(t, "lte", store.records[0].DeclaredTechnology)
	require.Equal(t, "gsm", store.records[0].DetectedTechnology)
	require.Equal(t, technologyMismatchReason, store.records[0].Reason)
	require.Equal(t, float64(1), testutil.ToFloat64(
		c.metrics.TechnologyMismatchFilesTotal.WithLabelValues("lte", "gsm"),
	))

	quarantined, err = c.checkTechnologyConsistency(context.Background(), payload, content)
	require.NoError(t, err)
	require.True(t, quarantined)
	require.Len(t, store.records, 1)
	require.Equal(t, float64(1), testutil.ToFloat64(
		c.metrics.TechnologyMismatchFilesTotal.WithLabelValues("lte", "gsm"),
	))
}

func TestCheckTechnologyConsistencyRetriesWhenQuarantinePersistenceFails(t *testing.T) {
	c := &PMCollector{
		quarantineStore: &recordingQuarantineStore{err: errors.New("tsdb unavailable")},
		logger:          zap.NewNop(),
	}
	payload := &FileReceivedPayload{
		MinIOPath:  "pm/SN-1.xml",
		DeviceSN:   "SN-1",
		Technology: "lte",
	}
	content := &PMFileContent{Counters: countersNamed(
		"BTS.ActiveSeconds",
		"SDCCH.Attempts",
	)}

	quarantined, err := c.checkTechnologyConsistency(context.Background(), payload, content)

	require.False(t, quarantined)
	require.ErrorContains(t, err, "save PM technology mismatch quarantine")
}

func TestCheckTechnologyConsistencyAllowsUnknownEvidence(t *testing.T) {
	store := &recordingQuarantineStore{}
	c := &PMCollector{quarantineStore: store, logger: zap.NewNop()}
	payload := &FileReceivedPayload{DeviceSN: "SN-1", Technology: "lte"}
	content := &PMFileContent{Counters: countersNamed("OTHER.CellServiceTime")}

	quarantined, err := c.checkTechnologyConsistency(context.Background(), payload, content)

	require.NoError(t, err)
	require.False(t, quarantined)
	require.Empty(t, store.records)
}

type recordingQuarantineStore struct {
	records []QuarantineRecord
	err     error
}

func (s *recordingQuarantineStore) Save(_ context.Context, record QuarantineRecord) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	for _, existing := range s.records {
		if existing.SourceFileID == record.SourceFileID && existing.Reason == record.Reason {
			return false, nil
		}
	}
	s.records = append(s.records, record)
	return true, nil
}

package collector

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestClassifyTechnologyUsesIndependentMetricFamilies(t *testing.T) {
	tests := []struct {
		name     string
		counters []model.PMCounter
		want     string
	}{
		{
			name: "gsm",
			counters: countersNamed(
				"BTS.ActiveSeconds",
				"SDCCH.Attempts",
				"TCH.SeizureAttempts",
			),
			want: "gsm",
		},
		{
			name: "lte",
			counters: countersNamed(
				"ERAB.EstabInitAttNbr.Sum",
				"S1SIG.ConnEstabAtt",
				"X2SIG.HoAttempt",
			),
			want: "lte",
		},
		{
			name: "nr",
			counters: countersNamed(
				"NR.RRC.ConnEstabAtt",
				"GNBCU.UEContextSetup",
				"GNBDU.Cell.Availability",
			),
			want: "nr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyTechnology(tt.counters)

			require.Equal(t, tt.want, got.Technology)
			require.GreaterOrEqual(t, got.Confidence, 0.5)
			require.Len(t, got.Signals, 3)
		})
	}
}

func TestClassifyTechnologyLeavesAmbiguousPayloadUnknown(t *testing.T) {
	tests := []struct {
		name     string
		counters []model.PMCounter
	}{
		{
			name:     "only common metric",
			counters: countersNamed("OTHER.CellServiceTime"),
		},
		{
			name: "one family is insufficient",
			counters: countersNamed(
				"BTS.ActiveSeconds",
				"OTHER.CellServiceTime",
			),
		},
		{
			name: "technology tie",
			counters: countersNamed(
				"BTS.ActiveSeconds",
				"SDCCH.Attempts",
				"ERAB.EstabInitAttNbr.Sum",
				"S1SIG.ConnEstabAtt",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyTechnology(tt.counters)

			require.Empty(t, got.Technology)
			require.Zero(t, got.Confidence)
		})
	}
}

func TestClassifyTechnologyWithIdentityUsesManagedElementForSparsePayload(t *testing.T) {
	got := ClassifyTechnologyWithIdentity(
		"SubNetwork=1,MeContext=gNB-FIXTURE",
		countersNamed("RRC.ConnEstabAtt", "PDCP.UpOctDl"),
	)

	require.Equal(t, "nr", got.Technology)
	require.Equal(t, []string{"managedElement:SubNetwork=1,MeContext=gNB-FIXTURE"}, got.Signals)
}

func TestClassifyTechnologyBoundsEvidenceCardinality(t *testing.T) {
	names := make([]string, 0, 40)
	for i := 0; i < 40; i++ {
		switch i % 4 {
		case 0:
			names = append(names, "BTS.ActiveSeconds")
		case 1:
			names = append(names, "SDCCH.Attempts")
		case 2:
			names = append(names, "TCH.SeizureAttempts")
		default:
			names = append(names, "Call.TotalAttempts")
		}
	}

	got := ClassifyTechnology(countersNamed(names...))

	require.Equal(t, "gsm", got.Technology)
	require.LessOrEqual(t, len(got.Signals), 20)
}

func countersNamed(names ...string) []model.PMCounter {
	out := make([]model.PMCounter, 0, len(names))
	for _, name := range names {
		out = append(out, model.PMCounter{CounterName: name})
	}
	return out
}

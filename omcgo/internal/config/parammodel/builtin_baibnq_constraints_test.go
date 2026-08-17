package parammodel

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltinBaiBNQ_NRBandRangeMatchesSouthboundModel(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "data", "param-mappings", "BaiBNQ.xml"))
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(raw, &doc))
	const bandPath = "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.{i}.FreqBandIndicatorNR"
	for _, param := range doc.Params {
		if param.StandardPath == bandPath && param.Name == bandPath {
			require.Equal(t, "1", param.Min)
			require.Equal(t, "1024", param.Max)
			return
		}
	}
	t.Fatalf("BaiBNQ mapping %s not found", bandPath)
}

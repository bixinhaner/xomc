package indicator

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBLQCriticalReportKeysMatchDevicePayload(t *testing.T) {
	path := filepath.Join("..", "..", "..", "data", "indicator-library", "enb", "BLQ.xml")
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	var doc xmlIndicatorModel
	require.NoError(t, xml.NewDecoder(file).Decode(&doc))

	byID := make(map[string]xmlIndicator, len(doc.Indicators))
	for _, item := range doc.Indicators {
		byID[item.ID] = item
	}
	require.Equal(t, "ERAB.EstabInitAttNbr.Sum", byID["C000010070"].ReportKey)
	require.Equal(t, "ERAB.EstabInitSuccNbr.Sum", byID["C000010080"].ReportKey)
	require.Equal(t, "RRC.SuccConnEstab.HIGHPRIORITYACCESS", byID["C000000014"].ReportKey)
}

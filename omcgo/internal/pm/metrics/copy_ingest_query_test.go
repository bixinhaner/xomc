package metrics

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertFileMarkerQueryKeepsUploadAndMeasurementTimesIndependent(t *testing.T) {
	id := uuid.New()
	collect := time.Date(2026, 8, 3, 15, 9, 0, 0, time.UTC)
	start := time.Date(2026, 8, 3, 14, 45, 0, 0, time.UTC)
	end := time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC)
	marker := FileMarker{
		DeviceID: uuid.New(), DeviceSN: "SN1", Carrier: "blq", Technology: "lte",
		FileName: "pm.xml", FileSize: 42, CollectTime: collect,
		MeasurementStart: start, MeasurementEnd: end, MinioPath: "pm/SN1/pm.xml",
		ContentSHA256: make([]byte, 32), CounterCount: 7, RawCompressed: true,
	}

	query, args, err := insertFileMarkerQuery(id, marker)

	require.NoError(t, err)
	assert.Contains(t, query, "collect_time,measurement_start,measurement_end")
	assert.Contains(t, query, "ON CONFLICT DO NOTHING")
	require.Len(t, args, 15)
	assert.Equal(t, collect, args[7])
	assert.Equal(t, start, args[8])
	assert.Equal(t, end, args[9])
}

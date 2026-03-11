package parser

import (
	"os"
	"testing"

	"github.com/omcgo/omcgo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMROParser_Parse(t *testing.T) {
	f, err := os.Open("../../../test/fixtures/mr/sample_mro.xml")
	require.NoError(t, err)
	defer f.Close()

	p := NewMROParser()
	data, err := p.Parse(f, model.CarrierCMCC)
	require.NoError(t, err)

	assert.Equal(t, "mro", data.MRType)
	assert.Equal(t, "ENB001", data.DeviceSN)
	assert.False(t, data.CollectTime.IsZero())
	assert.Len(t, data.Records, 5) // 3 from Cell-1 + 2 from Cell-2

	// Check first record has expected fields
	first := data.Records[0]
	assert.Equal(t, "Cell-1", first.CellID)
	assert.Contains(t, first.MeasurementData, "MR.LteScRSRP")
	assert.Contains(t, first.MeasurementData, "MR.LteScRSRQ")
	assert.Contains(t, first.MeasurementData, "MR.LteScSinrUL")
}

func TestMROParser_ParseEmpty(t *testing.T) {
	f, err := os.CreateTemp("", "empty_mro_*.xml")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	f.WriteString(`<?xml version="1.0"?><bulkPmMrDataFile></bulkPmMrDataFile>`)
	f.Seek(0, 0)

	p := NewMROParser()
	_, err = p.Parse(f, model.CarrierCMCC)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no MRO records")
}

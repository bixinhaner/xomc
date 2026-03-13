package parser

import (
	"os"
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMRSParser_Parse(t *testing.T) {
	f, err := os.Open("../../../test/fixtures/mr/sample_mrs.xml")
	require.NoError(t, err)
	defer f.Close()

	p := NewMRSParser()
	data, err := p.Parse(f, model.CarrierCMCC)
	require.NoError(t, err)

	assert.Equal(t, "mrs", data.MRType)
	assert.Equal(t, "ENB001", data.DeviceSN)
	assert.Len(t, data.Records, 2) // 2 cells

	first := data.Records[0]
	assert.Equal(t, "Cell-1", first.CellID)
	assert.Contains(t, first.MeasurementData, "MR.RSRP.00")
}

func TestMRSParser_ParseEmpty(t *testing.T) {
	f, err := os.CreateTemp("", "empty_mrs_*.xml")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	f.WriteString(`<?xml version="1.0"?><bulkPmMrDataFile></bulkPmMrDataFile>`)
	f.Seek(0, 0)

	p := NewMRSParser()
	_, err = p.Parse(f, model.CarrierCMCC)
	assert.Error(t, err)
}

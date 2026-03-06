package parser

import (
	"os"
	"strings"
	"testing"

	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMREParser_Parse(t *testing.T) {
	f, err := os.Open("../../../test/fixtures/mr/sample_mre.xml")
	require.NoError(t, err)
	defer f.Close()

	p := NewMREParser()
	data, err := p.Parse(f, model.CarrierCMCC)
	require.NoError(t, err)

	assert.Equal(t, "mre", data.MRType)
	assert.Equal(t, "ENB001", data.DeviceSN)
	assert.Len(t, data.Records, 2)

	first := data.Records[0]
	assert.Contains(t, first.MeasurementData, "MR.UeCategory")
}

func TestMREParser_CUCCNotSupported(t *testing.T) {
	r := strings.NewReader(`<?xml version="1.0"?><bulkPmMrDataFile></bulkPmMrDataFile>`)
	p := NewMREParser()
	_, err := p.Parse(r, model.CarrierCUCC)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestMREParser_CMCCSupported(t *testing.T) {
	f, err := os.Open("../../../test/fixtures/mr/sample_mre.xml")
	require.NoError(t, err)
	defer f.Close()

	p := NewMREParser()
	data, err := p.Parse(f, model.CarrierCMCC)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestMREParser_CTCCSupported(t *testing.T) {
	f, err := os.Open("../../../test/fixtures/mr/sample_mre.xml")
	require.NoError(t, err)
	defer f.Close()

	p := NewMREParser()
	data, err := p.Parse(f, model.CarrierCTCC)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

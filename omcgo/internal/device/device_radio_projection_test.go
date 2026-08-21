package device

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestProjectRadioFrequencyFields_LTEPhysicalCells(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells": "2",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL":          "39751",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.EARFCNDL":      "42599",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNUL":          "39751",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.EARFCNDL":          "39952",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.EARFCNUL":          "39751",
		"Device.Services.FAPService.3.CellConfig.LTE.RAN.RF.EARFCNDL":          "42599",
		"Device.Services.FAPService.3.CellConfig.LTE.RAN.RF.EARFCNUL":          "42599",
	}

	got := projectRadioFrequencyFields(params, model.TechLTE, "FAP/MLN/DC")

	assert.Equal(t, "39751,39952", got.dlValue)
	assert.Equal(t, "39751,39751", got.ulValue)
	assert.True(t, got.dlObserved)
	assert.True(t, got.ulObserved)
	assert.True(t, got.complete)
	assert.Empty(t, got.reason)
}

func TestProjectRadioFrequencyFields_IncompleteKnownCount(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells": "2",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL":          "39751",
	}

	got := projectRadioFrequencyFields(params, model.TechLTE, "FAP/MLN/DC")

	assert.True(t, got.dlObserved)
	assert.Empty(t, got.dlValue, "不完整的已知小区投影必须清空，不能压缩位置")
	assert.False(t, got.ulObserved)
	assert.False(t, got.complete)
	assert.Contains(t, got.reason, "DL index 2 is missing")
}

func TestProjectRadioFrequencyFields_LTEFallsBackToCommonDLPerCell(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells": "2",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL":          "",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.EARFCNDL":      "39751",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.EARFCNDL":      "39952",
	}

	got := projectRadioFrequencyFields(params, model.TechLTE, "FAP/MLN/DC")

	assert.Equal(t, "39751,39952", got.dlValue)
	assert.True(t, got.complete)
}

func TestProjectRadioFrequencyFields_LTESingleCellUsesOnlyObservedIndex(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells": "1",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.EARFCNDL":          "1498",
	}

	got := projectRadioFrequencyFields(params, model.TechLTE, "FAP/BAIBLQ/SC")

	assert.Equal(t, "1498", got.dlValue)
	assert.True(t, got.dlObserved)
	assert.True(t, got.complete)
	assert.Empty(t, got.reason)
}

func TestProjectRadioFrequencyFields_NRPhysicalCells(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.1.CellConfig.NR.RAN.CA.PARAMS.NumOfCells": "2",
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.NRARFCNDL":       "513000",
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.NRARFCNUL":       "513100",
		"Device.Services.FAPService.1.CellConfig.2.NR.RAN.RF.NRARFCNDL":       "630000",
		"Device.Services.FAPService.1.CellConfig.2.NR.RAN.RF.NRARFCNUL":       "513100",
		"Device.Services.FAPService.1.CellConfig.3.NR.RAN.RF.NRARFCNDL":       "700000",
		"Device.Services.FAPService.1.CellConfig.3.NR.RAN.RF.NRARFCNUL":       "700100",
	}

	got := projectRadioFrequencyFields(params, model.TechNR, "FAP/BQN/CA")

	assert.Equal(t, "513000,630000", got.dlValue)
	assert.Equal(t, "513100,513100", got.ulValue)
	assert.True(t, got.complete)
}

func TestProjectRadioFrequencyFields_UnknownCountCompatibility(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.4.CellConfig.LTE.RAN.RF.EARFCNDL": "44000",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.EARFCNDL": "22000",
	}

	got := projectRadioFrequencyFields(params, model.TechLTE, "")

	assert.Equal(t, "22000,44000", got.dlValue)
	assert.True(t, got.dlObserved)
	assert.True(t, got.complete)
}

func TestProjectRadioFrequencyFields_CARequiresReportedCount(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL": "39751",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.EARFCNDL": "39952",
	}

	got := projectRadioFrequencyFields(params, model.TechLTE, "FAP/MLN/CA")

	assert.True(t, got.dlObserved)
	assert.Empty(t, got.dlValue)
	assert.False(t, got.complete)
	assert.Contains(t, got.reason, "CA product requires reported NumOfCells")
}

func TestProjectRadioFrequencyFields_UnsupportedTechnology(t *testing.T) {
	got := projectRadioFrequencyFields(map[string]string{
		"Device.Services.GsmBTSCellDT.1.CurrentArfcn": "1010",
	}, model.TechGSM, "FAP/PGSM")

	assert.False(t, got.dlObserved)
	assert.False(t, got.ulObserved)
	assert.True(t, got.complete)
}

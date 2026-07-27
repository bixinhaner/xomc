package device

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDedupeDeviceParameterRowsKeepsLastValuePerDeviceAndPath(t *testing.T) {
	firstDevice := uuid.New()
	secondDevice := uuid.New()
	rows := []deviceParameterUpsertRow{
		{deviceID: firstDevice, parameter: model.DeviceParameter{
			ParameterPath: "Device.DeviceInfo.UpTime", ParameterValue: "1",
		}},
		{deviceID: secondDevice, parameter: model.DeviceParameter{
			ParameterPath: "Device.DeviceInfo.UpTime", ParameterValue: "2",
		}},
		{deviceID: firstDevice, parameter: model.DeviceParameter{
			ParameterPath: "Device.DeviceInfo.UpTime", ParameterValue: "3",
		}},
	}

	got := dedupeDeviceParameterRows(rows)

	require.Len(t, got, 2)
	assert.Equal(t, firstDevice, got[0].deviceID)
	assert.Equal(t, "3", got[0].parameter.ParameterValue)
	assert.Equal(t, secondDevice, got[1].deviceID)
}

func TestBuildDeviceParameterUpsertUsesOneConditionalMultiRowStatement(t *testing.T) {
	rows := []deviceParameterUpsertRow{
		{deviceID: uuid.New(), parameter: model.DeviceParameter{
			ParameterPath:  "Device.FAPService.2.CellConfig.LTE.RAN.RF.EARFCNDL",
			ParameterValue: "100", ParameterType: model.ParameterType("unsignedInt"),
		}},
		{deviceID: uuid.New(), parameter: model.DeviceParameter{
			ParameterPath:  "Device.DeviceInfo.UpTime",
			ParameterValue: "10", ParameterType: model.ParameterType("unsignedInt"),
		}},
	}

	query, args, err := buildDeviceParameterUpsert(rows, time.Unix(100, 0))

	require.NoError(t, err)
	assert.Equal(t, 16, len(args))
	assert.Equal(t, 1, strings.Count(query, "),("))
	assert.Contains(t, query, "ON CONFLICT (device_id, parameter_path) DO UPDATE")
	assert.Contains(t, query, "device_parameters.parameter_value IS DISTINCT FROM EXCLUDED.parameter_value")
	assert.Contains(t, query, "RETURNING device_id")
	assert.Equal(t, 2, args[6])
	assert.Equal(t, "radio", args[7])
}

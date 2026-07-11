package device

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetAntennaSectors_UsesAntennaParameterGroup(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := newFakeDeviceRepo()
	deviceRepo.devices[deviceID] = &model.Device{ID: deviceID}
	paramRepo := newFakeParamRepo()
	service := NewDeviceService(deviceRepo, paramRepo, nil, nil, zap.NewNop())

	paramRepo.groupParams[deviceID] = map[string][]model.DeviceParameter{"antenna": {
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.Azimuth", ParameterValue: "120"},
	}}

	sectors, err := service.GetAntennaSectors(context.Background(), deviceID)
	require.NoError(t, err)
	require.Len(t, sectors, 1)
	require.Equal(t, 120.0, sectors[0].Azimuth)
	require.True(t, sectors[0].DirectionAvailable)
	require.Equal(t, "antenna", paramRepo.lastGroup)
}

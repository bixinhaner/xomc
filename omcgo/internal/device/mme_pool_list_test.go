package device

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

type batchMMEParamRepo struct {
	*mockParamRepo
	params map[uuid.UUID][]model.DeviceParameter
}

func (r *batchMMEParamRepo) GetByDeviceIDsAndGroup(
	_ context.Context,
	deviceIDs []uuid.UUID,
	group string,
) (map[uuid.UUID][]model.DeviceParameter, error) {
	result := make(map[uuid.UUID][]model.DeviceParameter, len(deviceIDs))
	if group != "mme_pool" {
		return result, nil
	}
	for _, deviceID := range deviceIDs {
		result[deviceID] = r.params[deviceID]
	}
	return result, nil
}

func TestListDevicesWithInfoDecoratesMMEPoolAndUsesRealPoolStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	infoRepo := NewMockDeviceInfoRepository(ctrl)
	deviceID := uuid.New()
	staleStatus := "disconnected"
	list := model.NewListResponse([]DeviceWithInfo{{
		Device:    model.Device{ID: deviceID, Technology: model.TechLTE},
		MMEStatus: &staleStatus,
	}}, 1, 1, 20)

	infoRepo.EXPECT().ComputeListStats(gomock.Any(), gomock.Any()).Return(nil, nil)
	infoRepo.EXPECT().ListDevicesWithInfo(gomock.Any(), gomock.Any()).Return(list, nil)

	paramRepo := &batchMMEParamRepo{
		mockParamRepo: &mockParamRepo{},
		params: map[uuid.UUID][]model.DeviceParameter{
			deviceID: {
				{DeviceID: deviceID, ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MMEIp1", ParameterValue: "172.24.224.88"},
				{DeviceID: deviceID, ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status", ParameterValue: "1"},
				{DeviceID: deviceID, ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.2.MMEIp1", ParameterValue: "172.24.224.91"},
				{DeviceID: deviceID, ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.2.MME1Status", ParameterValue: "0"},
			},
		},
	}
	service := NewDeviceService(nil, paramRepo, nil, nil, zap.NewNop())
	service.deviceInfoRepo = infoRepo

	result, err := service.ListDevicesWithInfo(context.Background(), DeviceFilter{})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "connected", *result.Items[0].MMEStatus)
	require.Equal(t, []MMEEntry{
		{Index: 1, IP: "172.24.224.88", Status: "active"},
		{Index: 2, IP: "172.24.224.91", Status: "inactive"},
	}, result.Items[0].MMEPool)
}

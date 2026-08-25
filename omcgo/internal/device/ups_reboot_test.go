package device

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestRebootDevice_UPSProductClassQueuesReboot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deviceID := uuid.New()
	devRepo := NewMockDeviceRepository(ctrl)
	taskSvc := &stubSuccessTaskSvc{taskID: "ups-reboot-task-1"}
	svc := NewDeviceService(devRepo, nil, nil, nil, zap.NewNop())
	svc.taskSvc = taskSvc

	devRepo.EXPECT().GetByID(gomock.Any(), deviceID).Return(&model.Device{
		ID:             deviceID,
		SerialNumber:   "UPS-SN-001",
		ProductClass:   "UPS_M3_BMU",
		InformInterval: 300,
	}, nil)

	err := svc.RebootDevice(context.Background(), deviceID)

	require.NoError(t, err)
	require.NotNil(t, taskSvc.lastReq)
	assert.Equal(t, "UPS-SN-001", taskSvc.lastReq.DeviceSN)
	assert.Equal(t, "Reboot", taskSvc.lastReq.Method)
	assert.Equal(t, task.TaskSourceAPI, taskSvc.lastReq.Source)
	assert.Equal(t, 600, taskSvc.lastReq.ExpiresIn)
	assert.NotEmpty(t, taskSvc.lastReq.CommandKey)
}

func TestRebootDevice_NonUPSUsesDefaultTaskExpiry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deviceID := uuid.New()
	devRepo := NewMockDeviceRepository(ctrl)
	taskSvc := &stubSuccessTaskSvc{taskID: "radio-reboot-task-1"}
	svc := NewDeviceService(devRepo, nil, nil, nil, zap.NewNop())
	svc.taskSvc = taskSvc

	devRepo.EXPECT().GetByID(gomock.Any(), deviceID).Return(&model.Device{
		ID:           deviceID,
		SerialNumber: "RADIO-SN-001",
		ProductClass: "FAP/BAIBLQ",
	}, nil)

	err := svc.RebootDevice(context.Background(), deviceID)

	require.NoError(t, err)
	require.NotNil(t, taskSvc.lastReq)
	assert.Equal(t, "RADIO-SN-001", taskSvc.lastReq.DeviceSN)
	assert.Equal(t, "Reboot", taskSvc.lastReq.Method)
	assert.Equal(t, 0, taskSvc.lastReq.ExpiresIn)
}

func TestRebootDevice_WithTargetPersistsTargetInTaskParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deviceID := uuid.New()
	devRepo := NewMockDeviceRepository(ctrl)
	taskSvc := &stubSuccessTaskSvc{taskID: "ims-core-web-reboot-task-1"}
	svc := NewDeviceService(devRepo, nil, nil, nil, zap.NewNop())
	svc.taskSvc = taskSvc

	devRepo.EXPECT().GetByID(gomock.Any(), deviceID).Return(&model.Device{
		ID:           deviceID,
		SerialNumber: "IMSCORE-SN-001",
		ProductClass: "IMSCORE",
	}, nil)

	err := svc.RebootDevice(context.Background(), deviceID, 3)

	require.NoError(t, err)
	require.NotNil(t, taskSvc.lastReq)
	var params map[string]int
	require.NoError(t, json.Unmarshal(taskSvc.lastReq.Params, &params))
	assert.Equal(t, map[string]int{"reboot_target": 3}, params)
}

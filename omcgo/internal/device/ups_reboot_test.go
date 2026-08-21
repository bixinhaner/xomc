package device

import (
	"context"
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
		ID:           deviceID,
		SerialNumber: "UPS-SN-001",
		ProductClass: "UPS_M3_BMU",
	}, nil)

	err := svc.RebootDevice(context.Background(), deviceID)

	require.NoError(t, err)
	require.NotNil(t, taskSvc.lastReq)
	assert.Equal(t, "UPS-SN-001", taskSvc.lastReq.DeviceSN)
	assert.Equal(t, "Reboot", taskSvc.lastReq.Method)
	assert.Equal(t, task.TaskSourceAPI, taskSvc.lastReq.Source)
	assert.NotEmpty(t, taskSvc.lastReq.CommandKey)
}

package software

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpgradeExecutor_ExecuteOne_UsesDownloadFileTypeOverride(t *testing.T) {
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()

	deviceID := uuid.New()
	subTaskID := uuid.New()
	taskID := uuid.New()
	var queued devtask.CreateTaskRequest

	executor := NewUpgradeExecutor(
		&svcMockTaskRepo{},
		&svcMockSubTaskRepo{},
		&svcMockDeviceRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
				require.Equal(t, deviceID, id)
				return &model.Device{
					ID:              deviceID,
					SerialNumber:    "SN-FPGA-001",
					Status:          model.DeviceActive,
					FirmwareVersion: "V1.0.0",
				}, nil
			},
		},
		&svcMockFirmwareRepo{},
		&svcMockCmdQueue{
			createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
				queued = *req
				return devtask.NewTask(req), nil
			},
		},
		nil,
		redisClient,
		&svcMockEventBus{},
		zap.NewNop(),
	)

	executor.ExecuteOne(context.Background(), &UpgradeSubTask{
		ID:         subTaskID,
		TaskID:     taskID,
		DeviceID:   deviceID,
		FirmwareID: ptrUUID(uuid.New()),
	}, &FirmwareVersion{
		FileName:  "fpga.bin",
		FileSize:  4096,
		FileType:  FileTypeFPGA,
		MinIOPath: "cmcc/SC/V1.0.0/fpga.bin",
		MD5Val:    "abc123",
		Version:   "V2.0.0",
	}, false, "Firmware Upgrade Fpga")

	require.Equal(t, "Download", queued.Method)
	var params map[string]any
	require.NoError(t, json.Unmarshal(queued.Params, &params))
	assert.Equal(t, "Firmware Upgrade Fpga", params["file_type"])
}

func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}

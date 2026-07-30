package software

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/core/model"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRollbackOne_EnableCheckGPVPayloadBuildsNonEmptyACSRequest(t *testing.T) {
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		require.NoError(t, redisClient.Close())
	})

	subTaskID := uuid.New()
	subTask := &UpgradeSubTask{
		ID:     subTaskID,
		TaskID: uuid.New(),
	}
	device := &model.Device{
		ID:              uuid.New(),
		SerialNumber:    "SN-ROLLBACK",
		Status:          model.DeviceActive,
		FirmwareVersion: "BaiBLQ_5.1.12.1",
		ProductClass:    "FAP/BAIBLQ/SC",
	}

	var capturedReq *devtask.CreateTaskRequest
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			copied := *req
			copied.Params = append([]byte(nil), req.Params...)
			capturedReq = &copied
			return devtask.NewTask(req), nil
		},
	}

	exec := NewRollbackExecutor(
		&svcMockTaskRepo{},
		&svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		cmdQueue,
		nil,
		redisClient,
		&svcMockEventBus{},
		zap.NewNop(),
	)
	exec.SetParamPathTranslator(rollbackEnableTranslator{})

	exec.RollbackOne(context.Background(), subTask, device, model.TechLTE)

	require.NotNil(t, capturedReq)
	assert.Equal(t, "GetParameterValues", capturedReq.Method)
	assert.Equal(t, rollbackEnableCmdKeyPrefix+subTaskID.String(), capturedReq.CommandKey)

	var payload struct {
		Names          []string `json:"names"`
		ParameterNames []string `json:"parameter_names"`
	}
	require.NoError(t, json.Unmarshal(capturedReq.Params, &payload))
	assert.Equal(t, []string{"Device.DeviceInfo.X_COM_ROLLBACK_ENABLE"}, payload.Names)
	assert.Empty(t, payload.ParameterNames)

	body, err := rpc.NewDispatcher().BuildRequest(&rpc.Command{
		Method:     capturedReq.Method,
		Params:     capturedReq.Params,
		CommandKey: capturedReq.CommandKey,
	}, "ID:test.rollback.enable")
	require.NoError(t, err)

	soapBody := string(body)
	assert.Contains(t, soapBody, `SOAP-ENC:arrayType="xsd:string[1]"`)
	assert.Contains(t, soapBody, "<string>Device.DeviceInfo.X_COM_ROLLBACK_ENABLE</string>")
}

type rollbackEnableTranslator struct{}

func (rollbackEnableTranslator) TranslateForDevice(_ context.Context, _, _ string, standardPaths []string) ([]TranslatedParamPath, error) {
	out := make([]TranslatedParamPath, 0, len(standardPaths))
	for _, path := range standardPaths {
		private := path
		if path == "Device.DeviceInfo.ROLLBACK_ENABLE" {
			private = "Device.DeviceInfo.X_COM_ROLLBACK_ENABLE"
		}
		out = append(out, TranslatedParamPath{
			Standard: path,
			Private:  private,
			Source:   "test",
		})
	}
	return out, nil
}

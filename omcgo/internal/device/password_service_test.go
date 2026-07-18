package device

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetLMTPassword_QueuesMessageTask(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
			require.Equal(t, deviceID, id)
			return &model.Device{ID: deviceID, SerialNumber: "CELL1123"}, nil
		},
	}
	taskSvc := &stubSuccessTaskSvc{taskID: "password-reset-task-1"}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetTaskService(taskSvc)

	result, err := svc.ResetLMTPassword(context.Background(), deviceID, "operator-1")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "password-reset-task-1", result.TaskID)
	require.NotNil(t, taskSvc.lastReq)
	assert.Equal(t, "CELL1123", taskSvc.lastReq.DeviceSN)
	assert.Equal(t, resetLMTPasswordMethod, taskSvc.lastReq.Method)
	assert.Equal(t, "operator-1", taskSvc.lastReq.CreatorID)
	assert.NotContains(t, string(taskSvc.lastReq.Params), "X_COM_Localweb_password")

	var params map[string]string
	require.NoError(t, json.Unmarshal(taskSvc.lastReq.Params, &params))
	assert.Equal(t, resetLMTPasswordMethod, params["message_type"])
}

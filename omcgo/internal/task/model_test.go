package task

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_TaskStatus_Constants(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusSent, "sent"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusFailed, "failed"},
		{TaskStatusExpired, "expired"},
		{TaskStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, TaskStatus(tt.expected), tt.status)
		})
	}
}

func Test_TaskSource_Constants(t *testing.T) {
	assert.Equal(t, TaskSource("api"), TaskSourceAPI)
	assert.Equal(t, TaskSource("scheduler"), TaskSourceScheduler)
	assert.Equal(t, TaskSource("system"), TaskSourceSystem)
}

func Test_Task_JSONMarshal(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	task := &Task{
		ID:       "test-uuid-1234",
		DeviceSN: "SN001",
		Method:   "GetParameterValues",
		Params:   json.RawMessage(`{"names":["Device.DeviceInfo."]}`),
		Priority: 10,
		Status:   TaskStatusPending,
		Source:   TaskSourceAPI,
	}
	task.CreatedAt = now

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded Task
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, task.ID, decoded.ID)
	assert.Equal(t, task.DeviceSN, decoded.DeviceSN)
	assert.Equal(t, task.Method, decoded.Method)
	assert.Equal(t, task.Priority, decoded.Priority)
	assert.Equal(t, task.Status, decoded.Status)
	assert.Equal(t, task.Source, decoded.Source)
	assert.JSONEq(t, `{"names":["Device.DeviceInfo."]}`, string(decoded.Params))
}

func Test_Task_JSONMarshal_OmitEmpty(t *testing.T) {
	task := &Task{
		ID:       "test-1",
		DeviceSN: "SN001",
		Method:   "Reboot",
		Status:   TaskStatusPending,
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// Fields with omitempty and zero value should be absent
	_, hasCommandKey := raw["command_key"]
	assert.False(t, hasCommandKey, "empty command_key should be omitted")

	_, hasCWMPID := raw["cwmp_id"]
	assert.False(t, hasCWMPID, "empty cwmp_id should be omitted")

	_, hasSentAt := raw["sent_at"]
	assert.False(t, hasSentAt, "nil sent_at should be omitted")

	_, hasCompletedAt := raw["completed_at"]
	assert.False(t, hasCompletedAt, "nil completed_at should be omitted")
}

func Test_NewTask_DefaultValues(t *testing.T) {
	req := &CreateTaskRequest{
		DeviceSN: "SN001",
		Method:   "GetParameterValues",
		Params:   json.RawMessage(`{"names":["Device."]}`),
	}

	task := NewTask(req)

	assert.NotEmpty(t, task.ID, "should generate UUID")
	assert.Equal(t, "SN001", task.DeviceSN)
	assert.Equal(t, "GetParameterValues", task.Method)
	assert.Equal(t, 10, task.Priority, "default priority should be 10")
	assert.Equal(t, 3, task.MaxRetries, "default max retries should be 3")
	assert.Equal(t, TaskStatusPending, task.Status)
	assert.Equal(t, TaskSourceAPI, task.Source)
	assert.Nil(t, task.ExpiresAt, "no expiry by default")
	assert.Equal(t, 0, task.RetryCount)
}

func Test_NewTask_CustomValues(t *testing.T) {
	req := &CreateTaskRequest{
		DeviceSN:    "SN002",
		Method:      "SetParameterValues",
		Params:      json.RawMessage(`{}`),
		Priority:    5,
		ExpiresIn:   3600,
		MaxRetries:  5,
		Source:      TaskSourceScheduler,
		CreatorID:   "user-123",
		Description: "test task",
	}

	task := NewTask(req)

	assert.Equal(t, 5, task.Priority)
	assert.Equal(t, 5, task.MaxRetries)
	assert.Equal(t, TaskSourceScheduler, task.Source)
	assert.Equal(t, "user-123", task.CreatorID)
	assert.Equal(t, "test task", task.Description)
	assert.NotNil(t, task.ExpiresAt, "should have expiry when ExpiresIn > 0")
	// ExpiresAt should be roughly 1 hour from now
	assert.WithinDuration(t, time.Now().Add(3600*time.Second), *task.ExpiresAt, 2*time.Second)
}

func Test_NewTask_ZeroPriority(t *testing.T) {
	req := &CreateTaskRequest{
		DeviceSN: "SN001",
		Method:   "Reboot",
		Priority: 0, // zero priority => should default to 10
	}

	task := NewTask(req)
	assert.Equal(t, 10, task.Priority, "zero priority should default to 10")
}

func Test_NewTask_NegativePriority(t *testing.T) {
	req := &CreateTaskRequest{
		DeviceSN: "SN001",
		Method:   "Reboot",
		Priority: -1,
	}

	task := NewTask(req)
	assert.Equal(t, 10, task.Priority, "negative priority should default to 10")
}

func Test_Task_IsExpired(t *testing.T) {
	t.Run("no expiry set", func(t *testing.T) {
		task := &Task{ExpiresAt: nil}
		assert.False(t, task.IsExpired())
	})

	t.Run("not expired yet", func(t *testing.T) {
		future := time.Now().Add(1 * time.Hour)
		task := &Task{ExpiresAt: &future}
		assert.False(t, task.IsExpired())
	})

	t.Run("already expired", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		task := &Task{ExpiresAt: &past}
		assert.True(t, task.IsExpired())
	})
}

func Test_Task_CanRetry(t *testing.T) {
	tests := []struct {
		name       string
		retryCount int
		maxRetries int
		expected   bool
	}{
		{"zero retries, max 3", 0, 3, true},
		{"2 retries, max 3", 2, 3, true},
		{"3 retries, max 3", 3, 3, false},
		{"4 retries, max 3", 4, 3, false},
		{"0 retries, max 0", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{RetryCount: tt.retryCount, MaxRetries: tt.maxRetries}
			assert.Equal(t, tt.expected, task.CanRetry())
		})
	}
}

func Test_Task_MarkSent(t *testing.T) {
	task := &Task{
		ID:     "task-1",
		Status: TaskStatusPending,
	}

	task.MarkSent("cwmp-id-123")

	assert.Equal(t, TaskStatusSent, task.Status)
	assert.Equal(t, "cwmp-id-123", task.CWMPID)
	assert.NotNil(t, task.SentAt)
	assert.WithinDuration(t, time.Now(), *task.SentAt, 2*time.Second)
}

func Test_Task_MarkCompleted(t *testing.T) {
	task := &Task{
		ID:     "task-1",
		Status: TaskStatusSent,
	}

	result := json.RawMessage(`{"status":"ok"}`)
	task.MarkCompleted(result)

	assert.Equal(t, TaskStatusCompleted, task.Status)
	assert.JSONEq(t, `{"status":"ok"}`, string(task.Result))
	assert.NotNil(t, task.CompletedAt)
	assert.WithinDuration(t, time.Now(), *task.CompletedAt, 2*time.Second)
}

func Test_Task_MarkFailed(t *testing.T) {
	task := &Task{
		ID:     "task-1",
		Status: TaskStatusSent,
	}

	task.MarkFailed(9001, "CPE rejected request")

	assert.Equal(t, TaskStatusFailed, task.Status)
	assert.Equal(t, 9001, task.ErrorCode)
	assert.Equal(t, "CPE rejected request", task.ErrorMessage)
	assert.NotNil(t, task.CompletedAt)
}

func Test_Task_MarkExpired(t *testing.T) {
	task := &Task{
		ID:     "task-1",
		Status: TaskStatusSent,
	}

	task.MarkExpired()

	assert.Equal(t, TaskStatusExpired, task.Status)
	assert.NotNil(t, task.CompletedAt)
}

func Test_Task_ResetForRetry(t *testing.T) {
	sentAt := time.Now().Add(-5 * time.Minute)
	task := &Task{
		ID:         "task-1",
		Status:     TaskStatusSent,
		CWMPID:     "old-cwmp-id",
		SentAt:     &sentAt,
		RetryCount: 1,
	}

	task.ResetForRetry()

	assert.Equal(t, TaskStatusPending, task.Status)
	assert.Empty(t, task.CWMPID)
	assert.Nil(t, task.SentAt)
	assert.Equal(t, 2, task.RetryCount, "retry count should increment by 1")
}

func Test_TaskListResponse_JSON(t *testing.T) {
	resp := &TaskListResponse{
		Tasks: []*Task{
			{ID: "task-1", DeviceSN: "SN001", Method: "Reboot", Status: TaskStatusPending},
		},
		Total:    1,
		Page:     1,
		PageSize: 20,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded TaskListResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, int64(1), decoded.Total)
	assert.Equal(t, 1, decoded.Page)
	assert.Equal(t, 20, decoded.PageSize)
	require.Len(t, decoded.Tasks, 1)
	assert.Equal(t, "task-1", decoded.Tasks[0].ID)
}

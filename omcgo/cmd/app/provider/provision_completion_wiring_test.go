package provider

import (
	"context"
	"testing"

	"github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingProvisionCompletionHandler struct {
	tasks []*task.Task
}

func (h *recordingProvisionCompletionHandler) OnTaskCompleted(_ context.Context, completed *task.Task) {
	h.tasks = append(h.tasks, completed)
}

func TestRegisterProvisionCompletionHandlerUsesExistingRouter(t *testing.T) {
	router := task.NewCompletionRouter(zap.NewNop())
	handler := &recordingProvisionCompletionHandler{}

	registerProvisionCompletionHandler(router, nil, handler, zap.NewNop())
	router.Dispatch(context.Background(), &task.Task{
		ID:       "download-task",
		Source:   task.TaskSourceSystem,
		SourceID: "provisioning-task",
		Status:   task.TaskStatusFailed,
	})

	require.Len(t, handler.tasks, 1)
	require.Equal(t, "download-task", handler.tasks[0].ID)
}

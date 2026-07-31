package paramsync

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/task"
)

func TestBuildPlannedTasksCarriesStructuredRunIdentity(t *testing.T) {
	run := &SyncRun{ID: uuid.New(), DeviceSN: "SN-1"}
	tasks, err := buildPlannedTasks(run, &Plan{Batches: []TaskBatch{{Paths: []string{"Device.Info."}}}})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, task.TaskSourceParamSync, tasks[0].Source)
	assert.Equal(t, run.ID.String(), tasks[0].SourceID)
	assert.Equal(t, parameterSyncTaskRetryIntervalSeconds, tasks[0].RetryIntervalSeconds)
	assert.Equal(t, parameterSyncTaskExpiresIn/parameterSyncTaskRetryIntervalSeconds, tasks[0].MaxRetries)
	assert.NotContains(t, tasks[0].CommandKey, "full")
	var params map[string][]string
	require.NoError(t, json.Unmarshal(tasks[0].Params, &params))
	assert.Equal(t, []string{"Device.Info."}, params["names"])
}

func TestBuildPlannedTasksAssignsStableBatchIndexes(t *testing.T) {
	run := &SyncRun{ID: uuid.New(), RequestID: uuid.New(), DeviceSN: "SN-WINDOW"}
	tasks, err := buildPlannedTasks(run, &Plan{Batches: []TaskBatch{
		{Paths: []string{"Device.A"}},
		{Paths: []string{"Device.B"}},
		{Paths: []string{"Device.C"}},
	}})
	require.NoError(t, err)
	require.Len(t, tasks, 3)
	for index, planned := range tasks {
		assert.Equal(t, index, planned.CommandIndex)
	}
}

func TestBuildPlannedTasksPreservesRequestPriority(t *testing.T) {
	run := &SyncRun{ID: uuid.New(), RequestID: uuid.New(), DeviceSN: "SN-PRIORITY"}
	tasks, err := buildPlannedTasks(run, &Plan{Priority: 80, Batches: []TaskBatch{
		{Paths: []string{"Device.A"}}, {Paths: []string{"Device.B"}},
	}})
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	assert.Equal(t, 80, tasks[0].Priority)
	assert.Equal(t, 81, tasks[1].Priority)
}

func TestInitialReleasePlanContainsAllBatchesInOrder(t *testing.T) {
	tasks := []*task.Task{{ID: "one"}, {ID: "two"}, {ID: "three"}}

	released := initialReleasePlan(tasks)

	require.Len(t, released, 3)
	assert.Equal(t, []string{"one", "two", "three"}, []string{released[0].ID, released[1].ID, released[2].ID})
}

func TestOutboxBackoffIsBounded(t *testing.T) {
	assert.Equal(t, time.Second, outboxBackoff(1))
	assert.Equal(t, 512*time.Second, outboxBackoff(10))
	assert.Equal(t, 512*time.Second, outboxBackoff(100))
}

func TestAutomaticSyncBackoffEscalatesAndCaps(t *testing.T) {
	assert.Equal(t, time.Minute, automaticSyncBackoff(1))
	assert.Equal(t, 5*time.Minute, automaticSyncBackoff(2))
	assert.Equal(t, 15*time.Minute, automaticSyncBackoff(3))
	assert.Equal(t, time.Hour, automaticSyncBackoff(4))
	assert.Equal(t, 6*time.Hour, automaticSyncBackoff(100))
}

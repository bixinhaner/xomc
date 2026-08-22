package appconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParamSyncConfigValidate(t *testing.T) {
	assert.NoError(t, (ParamSyncConfig{
		ResultConsumerShardCount: 8, ResultConsumerQueueDepth: 64, ResultConsumerPullBatchSize: 64,
		ResultConsumerPullConcurrency: 64, ResultConsumerAckWait: 2 * time.Minute,
		ResultConsumerMaxAckPending: 512, RecoveryRunLimit: 20, RecoveryTaskLimitPerRun: 200, RecoveryTaskBudget: 200,
	}).validate())
	assert.ErrorContains(t, (ParamSyncConfig{ResultConsumerQueueDepth: -1}).validate(), "queue_depth")
	assert.ErrorContains(t, (ParamSyncConfig{ResultConsumerMaxAckPending: -1}).validate(), "max_ack_pending")
	assert.ErrorContains(t, (ParamSyncConfig{StartupSyncMaxSubmissions: -1}).validate(), "startup_sync_max_submissions")
	assert.ErrorContains(t, (ParamSyncConfig{StartupSyncSubmitInterval: -time.Second}).validate(), "startup_sync_submit_interval")
	assert.ErrorContains(t, (ParamSyncConfig{RecoveryTaskBudget: -1}).validate(), "recovery_task_budget")
	assert.NoError(t, (ParamSyncConfig{ManualOfflineMode: "queue"}).validate())
	assert.ErrorContains(t, (ParamSyncConfig{ManualOfflineMode: "drop"}).validate(), "manual_offline_mode")
}

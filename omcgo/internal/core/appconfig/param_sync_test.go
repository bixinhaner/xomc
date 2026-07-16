package appconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParamSyncConfigValidate(t *testing.T) {
	assert.NoError(t, (ParamSyncConfig{}).validate())
	assert.NoError(t, (ParamSyncConfig{
		RunEnabled: true, ResultConsumerEnabled: true, StagingEnabled: true, CanaryPercent: 100,
		ResultConsumerShardCount: 8, ResultConsumerQueueDepth: 64, ResultConsumerPullBatchSize: 64,
		ResultConsumerPullConcurrency: 64, ResultConsumerAckWait: 2 * time.Minute,
		ResultConsumerMaxAckPending: 512, RecoveryRunLimit: 20, RecoveryTaskLimitPerRun: 200, RecoveryTaskBudget: 200,
	}).validate())
	assert.ErrorContains(t, (ParamSyncConfig{CanaryPercent: 101}).validate(), "between 0 and 100")
	assert.ErrorContains(t, (ParamSyncConfig{RunEnabled: true, StagingEnabled: true, CanaryPercent: 10}).validate(), "result_consumer_enabled")
	assert.ErrorContains(t, (ParamSyncConfig{RunEnabled: true, ResultConsumerEnabled: true, CanaryPercent: 10}).validate(), "staging_enabled")
	assert.ErrorContains(t, (ParamSyncConfig{ResultConsumerEnabled: true}).validate(), "run_enabled")
	assert.ErrorContains(t, (ParamSyncConfig{ResultConsumerQueueDepth: -1}).validate(), "queue_depth")
	assert.ErrorContains(t, (ParamSyncConfig{ResultConsumerMaxAckPending: -1}).validate(), "max_ack_pending")
	assert.ErrorContains(t, (ParamSyncConfig{RecoveryTaskBudget: -1}).validate(), "recovery_task_budget")
}

package appconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParamSyncConfigValidate(t *testing.T) {
	assert.NoError(t, (ParamSyncConfig{}).validate())
	assert.NoError(t, (ParamSyncConfig{RunEnabled: true, ResultConsumerEnabled: true, StagingEnabled: true, CanaryPercent: 100}).validate())
	assert.ErrorContains(t, (ParamSyncConfig{CanaryPercent: 101}).validate(), "between 0 and 100")
	assert.ErrorContains(t, (ParamSyncConfig{RunEnabled: true, StagingEnabled: true, CanaryPercent: 10}).validate(), "result_consumer_enabled")
	assert.ErrorContains(t, (ParamSyncConfig{RunEnabled: true, ResultConsumerEnabled: true, CanaryPercent: 10}).validate(), "staging_enabled")
	assert.ErrorContains(t, (ParamSyncConfig{ResultConsumerEnabled: true}).validate(), "run_enabled")
}

package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationalStatusQueriesRealMMEPoolStatuses(t *testing.T) {
	var paths []string
	for _, group := range DefaultQueryGroups {
		if group.Tier == TierEveryInform && group.Label == "operational-status" {
			paths = group.Paths
			break
		}
	}

	require.NotEmpty(t, paths)
	assert.Contains(t, paths, "Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_MmePool.MmePool1Status")
	assert.Contains(t, paths, "Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_MmePool.MmePool2Status")
	assert.Contains(t, paths, "Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeStatus")
}

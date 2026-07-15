package paramsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBindingTargetForRunStatus(t *testing.T) {
	tests := []struct {
		name          string
		status        RunStatus
		bindingStatus string
		provisioning  string
		terminal      bool
	}{
		{name: "active", status: RunStatusExecuting, bindingStatus: "waiting", provisioning: "syncing"},
		{name: "succeeded", status: RunStatusSucceeded, bindingStatus: "completed", provisioning: "completed", terminal: true},
		{name: "failed", status: RunStatusFailed, bindingStatus: "failed", provisioning: "failed", terminal: true},
		{name: "cancelled", status: RunStatusCancelled, bindingStatus: "cancelled", provisioning: "failed", terminal: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bindingTargetForRunStatus(tt.status)
			assert.Equal(t, tt.bindingStatus, got.bindingStatus)
			assert.Equal(t, tt.provisioning, got.provisioningStatus)
			assert.Equal(t, tt.terminal, got.terminal)
		})
	}
}

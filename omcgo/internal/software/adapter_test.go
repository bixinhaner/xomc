package software

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestDefaultAdapter_RollbackParameterPath_LTE(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	path := adapter.RollbackParameterPath(model.TechLTE)
	assert.Equal(t, "Device.DeviceInfo.X_COM_ROLLBACK_CONTROL", path)
}

func TestDefaultAdapter_RollbackParameterPath_NR(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	path := adapter.RollbackParameterPath(model.TechNR)
	assert.Equal(t, "Device.SoftwareCtrl.ActivateEnable", path)
}

func TestDefaultAdapter_RollbackParameterValue(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	assert.Equal(t, "1", adapter.RollbackParameterValue(model.TechLTE))
	assert.Equal(t, "1", adapter.RollbackParameterValue(model.TechNR))
}

func TestDefaultAdapter_RollbackNeedsEnableCheck_LTE(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	assert.True(t, adapter.RollbackNeedsEnableCheck(model.TechLTE))
}

func TestDefaultAdapter_RollbackNeedsEnableCheck_NR(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	assert.False(t, adapter.RollbackNeedsEnableCheck(model.TechNR))
}

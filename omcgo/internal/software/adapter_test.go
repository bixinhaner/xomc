package software

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestDefaultAdapter_RollbackEnableCheckPath(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	// 4G 走两阶段，需要 GET ROLLBACK_ENABLE（standardPath）
	assert.Equal(t, "Device.DeviceInfo.ROLLBACK_ENABLE", adapter.RollbackEnableCheckPath(model.TechLTE))
	// 5G 不需要 enable check
	assert.Equal(t, "", adapter.RollbackEnableCheckPath(model.TechNR))
}

func TestDefaultAdapter_RollbackParameterPath_LTE(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	path := adapter.RollbackParameterPath(model.TechLTE)
	// standardPath 形态（无 X_COM 前缀），Translator 翻译到各 param_model 私有路径
	assert.Equal(t, "Device.DeviceInfo.ROLLBACK_CONTROL", path)
}

func TestDefaultAdapter_RollbackParameterPath_NR(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	path := adapter.RollbackParameterPath(model.TechNR)
	assert.Equal(t, "Device.SoftwareCtrl.ActivateEnable", path)
}

func TestDefaultAdapter_RollbackParameterValue_NR_AlwaysOne(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	// 5G 不分私有路径，统一 string/1
	v, t1 := adapter.RollbackParameterValue(model.TechNR, "Device.SoftwareCtrl.ActivateEnable")
	assert.Equal(t, "1", v)
	assert.Equal(t, "xsd:string", t1)

	v, t1 = adapter.RollbackParameterValue(model.TechNR, "InternetGatewayDevice.SoftwareCtrl.ActivateEnable")
	assert.Equal(t, "1", v)
	assert.Equal(t, "xsd:string", t1)
}

func TestDefaultAdapter_RollbackParameterValue_LTE_TR098Boolean(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	// TR-098 系：InternetGatewayDevice.* + RollBackEnable（驼峰） → 走 boolean
	v, t1 := adapter.RollbackParameterValue(model.TechLTE, "InternetGatewayDevice.DeviceInfo.RollBackEnable")
	assert.Equal(t, "true", v)
	assert.Equal(t, "xsd:boolean", t1)
}

func TestDefaultAdapter_RollbackParameterValue_LTE_DefaultStringOne(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	// 其它（TR-181 风格 X_COM_ROLLBACK_CONTROL 等）→ string/1
	v, t1 := adapter.RollbackParameterValue(model.TechLTE, "Device.DeviceInfo.X_COM_ROLLBACK_CONTROL")
	assert.Equal(t, "1", v)
	assert.Equal(t, "xsd:string", t1)
}

func TestDefaultAdapter_RollbackNeedsEnableCheck_LTE(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	assert.True(t, adapter.RollbackNeedsEnableCheck(model.TechLTE))
}

func TestDefaultAdapter_RollbackNeedsEnableCheck_NR(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	assert.False(t, adapter.RollbackNeedsEnableCheck(model.TechNR))
}

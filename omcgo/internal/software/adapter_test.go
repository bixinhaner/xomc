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

// qa-614 c6 / #373：2G/GSM 回退分支显式列出，与 4G 同属 Baicells param_model
// 家族（两阶段 GPV ROLLBACK_ENABLE → SPV ROLLBACK_CONTROL），不再静默落 LTE default。
func TestDefaultAdapter_GSM_RollbackBranch(t *testing.T) {
	adapter := NewDefaultUpgradeAdapter()
	assert.Equal(t, "Device.DeviceInfo.ROLLBACK_ENABLE", adapter.RollbackEnableCheckPath(model.TechGSM))
	assert.Equal(t, "Device.DeviceInfo.ROLLBACK_CONTROL", adapter.RollbackParameterPath(model.TechGSM))
	assert.True(t, adapter.RollbackNeedsEnableCheck(model.TechGSM))
	v, xsd := adapter.RollbackParameterValue(model.TechGSM, "Device.DeviceInfo.X_COM_ROLLBACK_CONTROL")
	assert.Equal(t, "1", v)
	assert.Equal(t, "xsd:string", xsd)
}

// qa-614 c6 / #373：ResolveDeviceTech 三态识别 GSM / NR / LTE（兜底）。
func TestResolveDeviceTech_ThreeState(t *testing.T) {
	cases := []struct {
		name string
		dev  *model.Device
		want model.Technology
	}{
		{"gsm by technology field", &model.Device{Technology: model.TechGSM}, model.TechGSM},
		{"gsm by PGSM productClass", &model.Device{ProductClass: "FAP/PGSM"}, model.TechGSM},
		{"gsm by BTS productClass", &model.Device{ProductClass: "FAP/BTS"}, model.TechGSM},
		{"nr by technology field", &model.Device{Technology: model.TechNR}, model.TechNR},
		{"nr by BNQ productClass", &model.Device{ProductClass: "BNQ"}, model.TechNR},
		{"lte fallback unclassified", &model.Device{ProductClass: "4G eNB"}, model.TechLTE},
		{"lte fallback empty", &model.Device{}, model.TechLTE},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolveDeviceTech(tc.dev))
		})
	}
}

func TestIsGSM(t *testing.T) {
	assert.True(t, IsGSM(&model.Device{Technology: model.TechGSM}))
	assert.True(t, IsGSM(&model.Device{ProductClass: "FAP/PGSM"}))
	assert.True(t, IsGSM(&model.Device{ProductClass: "FAP/BTS"}))
	assert.False(t, IsGSM(&model.Device{ProductClass: "4G eNB"}))
	assert.False(t, IsGSM(&model.Device{ProductClass: "BNQ"}))
}

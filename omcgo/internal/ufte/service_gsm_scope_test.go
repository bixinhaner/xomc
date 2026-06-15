package ufte

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
)

// qa-614 c6 / #365 #373：GSM_IMG_UPGRADE 内置任务类型存在且分类/制式正确。
func TestBuiltInTaskTypes_IncludesGSMUpgrade(t *testing.T) {
	var gsm *TaskType
	for i := range builtInTaskTypesSlice() {
		it := builtInTaskTypesSlice()[i]
		if it.TypeCode == "GSM_IMG_UPGRADE" {
			gsm = &it
			break
		}
	}
	require.NotNil(t, gsm, "GSM_IMG_UPGRADE 应存在于 builtInTaskTypes()")
	assert.Equal(t, "gsm_upgrade", gsm.Category)
	assert.Equal(t, "2G升级", gsm.CategoryLabel)
	assert.True(t, gsm.BuiltIn)
	assert.Equal(t, software.TaskTypeUpgrade, gsm.softwareTaskType)
	require.NotNil(t, gsm.techHint)
	assert.Equal(t, coremodel.TechGSM, *gsm.techHint)
	assert.Equal(t, "CODE_GSM_UPGRADE_IMAGE", gsm.PermissionCode)
	require.NotNil(t, gsm.FirmwareFileType)
	assert.Equal(t, software.FileTypeIMG, *gsm.FirmwareFileType)
}

func builtInTaskTypesSlice() []TaskType { return builtInTaskTypes() }

// qa-614 c6 / #365 #373：matchesTaskTypeScope 对 *techHint==TechGSM 的 BSC/BTS/PGSM
// productClass 兜底返回 true（productTechLookup 不可用场景）。
func TestMatchesTaskTypeScope_GSM(t *testing.T) {
	gsm := coremodel.TechGSM
	// PlatformScope 故意留空，强制走 switch 兜底关键字白名单分支。
	item := TaskType{TypeCode: "GSM_IMG_UPGRADE", techHint: &gsm, PlatformScope: nil}

	cases := []struct {
		name         string
		productClass string
		want         bool
	}{
		{"BSC productClass", "FAP/PGSM", true},
		{"BTS productClass", "FAP/BTS", true},
		{"raw BSC", "BSC", true},
		{"raw GSM keyword", "GSM-CELL", true},
		{"raw 2G keyword", "2G-NODE", true},
		{"5G product not matched by gsm scope", "BBU-QSS", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, matchesTaskTypeScope(item, tc.productClass))
		})
	}
}

// 平台标签命中时不依赖兜底关键字（PlatformScope 直接子串命中）。
func TestMatchesTaskTypeScope_GSM_PlatformScopeHit(t *testing.T) {
	gsm := coremodel.TechGSM
	item := TaskType{TypeCode: "GSM_IMG_UPGRADE", techHint: &gsm, PlatformScope: []string{"2G BSC", "2G BTS"}}
	assert.True(t, matchesTaskTypeScope(item, "2G BSC"))
}

// qa-614 c6 / #373：deviceMatchesTaskType 经 productTechLookup 对 gsm 设备放行
// （PlatformScope 完全不命中、关键字也不命中时，精确路径仍能放行）。
func TestDeviceMatchesTaskType_GSM_ViaProductTechLookup(t *testing.T) {
	gsm := coremodel.TechGSM
	// 用一个既不命中 PlatformScope 也不命中关键字白名单的 productClass，
	// 验证精确路径（productTechLookup → tech=="gsm"）能放行。
	item := TaskType{TypeCode: "GSM_IMG_UPGRADE", techHint: &gsm, PlatformScope: []string{"2G BSC"}}

	svc := NewService(nil, nil, nil, nil, nil, zap.NewNop())
	svc.productTechLookup = func(_ context.Context, pc string) (string, bool) {
		if pc == "FAP/CUSTOM-2G" {
			return "gsm", true
		}
		return "", false
	}

	assert.True(t, svc.deviceMatchesTaskType(context.Background(), item, "FAP/CUSTOM-2G"),
		"gsm 设备应经 productTechLookup 精确放行")
	// 一个 5G 设备（lookup 返回 nr）不应被 gsm 升级类放行
	svc.productTechLookup = func(_ context.Context, pc string) (string, bool) {
		return "nr", true
	}
	assert.False(t, svc.deviceMatchesTaskType(context.Background(), item, "FAP/NR-CUSTOM"),
		"5G 设备不应被 GSM 升级类放行")
}

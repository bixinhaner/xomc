package ufte

import (
	"context"
	"testing"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type staticUFTEFeatureChecker struct {
	ok  bool
	err error
}

func (c staticUFTEFeatureChecker) CheckFeature(_ context.Context, _ string) (bool, error) {
	return c.ok, c.err
}

type emptyUFTEUpgradeTaskRepo struct {
	software.TaskRepository
}

func (r emptyUFTEUpgradeTaskRepo) List(_ context.Context, filter software.UpgradeTaskFilter) (*coremodel.ListResponse[software.UpgradeTask], error) {
	return &coremodel.ListResponse[software.UpgradeTask]{
		Items:      []software.UpgradeTask{},
		Total:      0,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: 1,
	}, nil
}

func TestBuiltInTaskTypes_IncludesUPSAPUpgrade(t *testing.T) {
	item, ok := findTaskTypeByCode(builtInTaskTypes(), "UPS_AP_UPGRADE")
	require.True(t, ok, "UPS_AP_UPGRADE 应存在于 builtInTaskTypes()")

	assert.Equal(t, "ups_upgrade", item.Category)
	assert.Equal(t, "UPS升级", item.CategoryLabel)
	assert.True(t, item.BuiltIn)
	assert.Equal(t, software.TaskTypeUpgrade, item.softwareTaskType)
	assert.Equal(t, "CODE_UPS_UPGRADE_IMAGE", item.PermissionCode)
	assert.Equal(t, []string{"UPS"}, item.Products)
	assert.Equal(t, "1 Firmware Upgrade Image", item.FileType)
	assert.Contains(t, item.Description, "1 BOOT")
	assert.Contains(t, item.StepChain, "WAIT_REBOOT_COMPLETE")
	assert.Nil(t, item.techHint)
	require.NotNil(t, item.FirmwareFileType)
	assert.Equal(t, software.FileTypeIMG, *item.FirmwareFileType)
}

func TestDeviceUpgradeVirtualCategoryIncludesUPSUpgrade(t *testing.T) {
	assert.True(t, categoryMatchesFilter("device_upgrade", "ups_upgrade"))
	assert.True(t, categoryMatchesFilter("device_upgrade", "enb_upgrade"))
	assert.True(t, categoryMatchesFilter("device_upgrade", "gnb_upgrade"))
	assert.True(t, categoryMatchesFilter("device_upgrade", "gsm_upgrade"))
	assert.False(t, categoryMatchesFilter("device_upgrade", "station_log"))
	assert.False(t, categoryMatchesFilter("device_upgrade", "version_rollback"))

	typeSet, err := filterTaskTypeSet(builtInTaskTypes(), "device_upgrade", "")
	require.NoError(t, err)
	assert.Contains(t, typeSet, software.TaskTypeUpgrade)
	assert.Contains(t, typeSet, software.TaskTypePatch)
	assert.Contains(t, typeSet, software.TaskTypeFPGA)
	assert.NotContains(t, typeSet, software.TaskTypeRollback)
	assert.NotContains(t, typeSet, software.TaskTypeLogCollect)
}

func TestResolveTaskTypeForTask_ProductScopePrefersUPSAPUpgrade(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil, zap.NewNop())
	svc.productNameLookup = func(_ context.Context, productClass string) (string, bool) {
		if productClass == "UPS_M3_BMU" || productClass == "UPS" {
			return "UPS", true
		}
		return "", false
	}

	item, ok := svc.resolveTaskTypeForTask(
		context.Background(),
		builtInTaskTypes(),
		software.TaskTypeUpgrade,
		"UPS_M3_BMU",
		"1 Firmware Upgrade Image",
	)

	require.True(t, ok)
	assert.Equal(t, "UPS_AP_UPGRADE", item.TypeCode)
}

func TestResolveTaskTypeForTask_UPSPrefixWorksWithoutProductLookup(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil, zap.NewNop())

	item, ok := svc.resolveTaskTypeForTask(
		context.Background(),
		builtInTaskTypes(),
		software.TaskTypeUpgrade,
		"UPS",
		"1 Firmware Upgrade Image",
	)

	require.True(t, ok)
	assert.Equal(t, "UPS_AP_UPGRADE", item.TypeCode)
}

func TestDeviceMatchesTaskType_UPSPrefixFallback(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil, zap.NewNop())
	item := TaskType{TypeCode: "UPS_AP_UPGRADE", Products: []string{"UPS"}}

	assert.True(t, svc.deviceMatchesTaskType(context.Background(), item, "UPS_M3_BMU"))
	assert.False(t, svc.deviceMatchesTaskType(context.Background(), item, "ups_m3_bmu"))
	assert.False(t, svc.deviceMatchesTaskType(context.Background(), item, "FAP/BNQ"))
}

func TestGetTaskTypesFiltersUPSByLicense(t *testing.T) {
	repo := &ensureBuiltInTaskTypeRepo{items: builtInTaskTypes()}
	svc := NewService(nil, repo, emptyUFTEUpgradeTaskRepo{}, nil, nil, zap.NewNop())
	svc.SetLicenseFeatureChecker(staticUFTEFeatureChecker{ok: false})

	items, err := svc.GetTaskTypes(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, items)
	assert.NotContains(t, typeCodesOf(items), "UPS_AP_UPGRADE")

	svc.SetLicenseFeatureChecker(staticUFTEFeatureChecker{ok: true})
	items, err = svc.GetTaskTypes(context.Background())
	require.NoError(t, err)
	assert.Contains(t, typeCodesOf(items), "UPS_AP_UPGRADE")
}

func TestCreateTaskRejectsUPSWhenLicenseDisabled(t *testing.T) {
	repo := &ensureBuiltInTaskTypeRepo{items: builtInTaskTypes()}
	svc := NewService(nil, repo, nil, nil, nil, zap.NewNop())
	svc.SetLicenseFeatureChecker(staticUFTEFeatureChecker{ok: false})

	_, err := svc.CreateTask(context.Background(), CreateTaskRequest{TypeCode: "UPS_AP_UPGRADE"}, "tester", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrLicenseFeatureNotAuthorized)
}

func typeCodesOf(items []TaskType) []string {
	codes := make([]string, 0, len(items))
	for _, item := range items {
		codes = append(codes, item.TypeCode)
	}
	return codes
}

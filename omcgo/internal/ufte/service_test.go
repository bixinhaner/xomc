package ufte

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/software"
)

type ensureBuiltInTaskTypeRepo struct {
	items    []TaskType
	upserted []TaskType
	deleted  []string
}

type stubDeviceRepo struct {
	device.DeviceRepository
	listResponse *coremodel.ListResponse[coremodel.Device]
	listError    error
}

func (s *stubDeviceRepo) List(_ context.Context, _ device.DeviceFilter) (*coremodel.ListResponse[coremodel.Device], error) {
	return s.listResponse, s.listError
}

func (r *ensureBuiltInTaskTypeRepo) List(_ context.Context) ([]TaskType, error) {
	result := make([]TaskType, len(r.items))
	copy(result, r.items)
	return result, nil
}

func (r *ensureBuiltInTaskTypeRepo) GetByCode(_ context.Context, typeCode string) (*TaskType, error) {
	for i := range r.items {
		if r.items[i].TypeCode == typeCode {
			item := r.items[i]
			return &item, nil
		}
	}
	return nil, nil
}

func (r *ensureBuiltInTaskTypeRepo) Upsert(_ context.Context, item *TaskType) error {
	r.upserted = append(r.upserted, *item)
	r.items = append(r.items, *item)
	return nil
}

func (r *ensureBuiltInTaskTypeRepo) Delete(_ context.Context, typeCode string) error {
	for index := range r.items {
		if r.items[index].TypeCode != typeCode {
			continue
		}
		r.deleted = append(r.deleted, typeCode)
		r.items = append(r.items[:index], r.items[index+1:]...)
		return nil
	}
	return nil
}

func TestService_EnsureBuiltInTaskTypes_InsertsOnlyMissingDefaults(t *testing.T) {
	defaults := builtInTaskTypes()
	repo := &ensureBuiltInTaskTypeRepo{
		items: []TaskType{
			{
				TypeCode:      defaults[0].TypeCode,
				Category:      defaults[0].Category,
				CategoryLabel: defaults[0].CategoryLabel,
				DisplayName:   "自定义后的 4G 升级名称",
				BuiltIn:       true,
			},
		},
	}
	svc := NewService(nil, repo, nil, nil, nil, zap.NewNop())

	inserted, err := svc.EnsureBuiltInTaskTypes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, len(defaults)-1, inserted)
	require.Len(t, repo.upserted, len(defaults)-1)
	for _, item := range repo.upserted {
		assert.NotEqual(t, defaults[0].TypeCode, item.TypeCode)
		assert.True(t, item.BuiltIn)
	}
	assert.Equal(t, "自定义后的 4G 升级名称", repo.items[0].DisplayName)
	assert.Equal(t, len(defaults), len(repo.items))
}

func TestBuiltInTaskTypes_CoversRequiredTemplates(t *testing.T) {
	defaults := builtInTaskTypes()
	required := map[string]struct{}{
		"ENB_IMG_UPGRADE":     {},
		"ENB_FPGA_UPGRADE":    {},
		"GNB_IMG_UPGRADE":     {},
		"VERSION_ROLLBACK":    {},
		"RUNTIME_LOG_COLLECT": {},
		"FAULT_LOG_COLLECT":   {},
		"CONFIG_BACKUP":       {},
		"CONFIG_RESTORE":      {},
	}

	seen := make(map[string]TaskType, len(defaults))
	for _, item := range defaults {
		seen[item.TypeCode] = item
	}

	for code := range required {
		item, ok := seen[code]
		require.Truef(t, ok, "required built-in task type %s should exist", code)
		assert.Truef(t, item.BuiltIn, "required built-in task type %s should be marked built-in", code)
		assert.NotEmptyf(t, item.Category, "required built-in task type %s should have category", code)
		assert.NotEmptyf(t, item.DisplayName, "required built-in task type %s should have display name", code)
	}

	assert.Equal(t, "station_log", seen["RUNTIME_LOG_COLLECT"].Category)
	assert.Equal(t, "config_backup", seen["CONFIG_BACKUP"].Category)
	assert.Equal(t, "config_restore", seen["CONFIG_RESTORE"].Category)
	assert.Equal(t, "1 Firmware Upgrade Image", seen["ENB_IMG_UPGRADE"].FileType)
	assert.Equal(t, "X {OUI} Software Upgrade Patch", seen["ENB_PATCH_UPGRADE"].FileType)
	assert.Equal(t, "Firmware Upgrade Fpga", seen["ENB_FPGA_UPGRADE"].FileType)
	_, has5GFpga := seen["GNB_FPGA_UPGRADE"]
	assert.False(t, has5GFpga)
}

func TestService_LoadTaskTypeCatalog_UsesStoredRowsAsSourceOfTruth(t *testing.T) {
	repo := &ensureBuiltInTaskTypeRepo{
		items: []TaskType{
			{
				TypeCode:       "ENB_IMG_UPGRADE",
				Category:       "enb_upgrade",
				CategoryLabel:  "4G升级",
				DisplayName:    "库里的 4G 升级模板",
				Description:    "from db",
				RPCType:        "DOWNLOAD",
				BuiltIn:        true,
				Enabled:        true,
				StepChain:      []string{"CHECK_PERMISSION", "SEND_RPC"},
				PermissionCode: "CODE_ENB_UPGRADE_IMAGE",
				PlatformScope:  []string{"4G eNB"},
				FileType:       "1 Firmware Upgrade Image",
				FileTypeLabel:  "1 Firmware Upgrade Image",
				LastEditor:     "system",
			},
			{
				TypeCode:       "CUSTOM_UPLOAD_SAMPLE",
				Category:       "custom_upload",
				CategoryLabel:  "自定义上传",
				DisplayName:    "自定义上传模板",
				Description:    "custom",
				RPCType:        "UPLOAD",
				BuiltIn:        false,
				Enabled:        true,
				StepChain:      []string{"SEND_RPC", "WAIT_RPC_RESPONSE"},
				PermissionCode: "CODE_CUSTOM_UPLOAD_SAMPLE",
				PlatformScope:  []string{"5G gNB"},
				FileType:       "9",
				FileTypeLabel:  "Custom Upload",
				LastEditor:     "tester",
			},
		},
	}
	svc := NewService(nil, repo, nil, nil, nil, zap.NewNop())

	catalog, err := svc.loadTaskTypeCatalog(context.Background())
	require.NoError(t, err)
	require.Len(t, catalog, 2)

	first, ok := findTaskTypeByCode(catalog, "ENB_IMG_UPGRADE")
	require.True(t, ok)
	assert.Equal(t, "库里的 4G 升级模板", first.DisplayName)
	assert.Equal(t, software.TaskTypeUpgrade, first.softwareTaskType)
	require.NotNil(t, first.techHint)

	custom, ok := findTaskTypeByCode(catalog, "CUSTOM_UPLOAD_SAMPLE")
	require.True(t, ok)
	assert.Equal(t, "自定义上传模板", custom.DisplayName)
	assert.Zero(t, custom.softwareTaskType)
	assert.Nil(t, custom.techHint)
}

func TestNormalizeTaskTypeFileType_DownloadTemplatesUseFinalCWMPString(t *testing.T) {
	assert.Equal(t, "1 Firmware Upgrade Image", normalizeTaskTypeFileType("DOWNLOAD", "1"))
	assert.Equal(t, "3 Vendor Configuration File", normalizeTaskTypeFileType("DOWNLOAD", "config"))
	assert.Equal(t, "Firmware Upgrade Fpga", normalizeTaskTypeFileType("DOWNLOAD", "Firmware Upgrade Fpga"))
	assert.Equal(t, "6", normalizeTaskTypeFileType("UPLOAD", "6"))
}

func TestService_ListDeviceCandidates_FiltersByTypeScope(t *testing.T) {
	taskTypeRepo := &ensureBuiltInTaskTypeRepo{
		items: builtInTaskTypes(),
	}
	deviceRepo := &stubDeviceRepo{listResponse: &coremodel.ListResponse[coremodel.Device]{
		Items: []coremodel.Device{
			{
				ID:              uuid.New(),
				SerialNumber:    "ENB00001",
				ProductClass:    "QAFA",
				Technology:      coremodel.TechLTE,
				FirmwareVersion: "V1.0.0",
				SiteName:        "北京 4G 站点",
			},
			{
				ID:              uuid.New(),
				SerialNumber:    "ENB00002",
				ProductClass:    "FAP/BU1810",
				Technology:      coremodel.TechLTE,
				FirmwareVersion: "V1.0.1",
				SiteName:        "广州 4G 站点",
			},
			{
				ID:              uuid.New(),
				SerialNumber:    "GNB00001",
				ProductClass:    "BBU-XSS",
				Technology:      coremodel.TechNR,
				FirmwareVersion: "V9.0.0",
				SiteName:        "上海 5G 站点",
			},
		},
		Total:    2,
		Page:     1,
		PageSize: 200,
	}}

	svc := NewService(nil, taskTypeRepo, nil, nil, deviceRepo, zap.NewNop())

	result, err := svc.ListDeviceCandidates(context.Background(), DeviceCandidateFilter{
		Category: "enb_upgrade",
		TypeCode: "ENB_PATCH_UPGRADE",
		Page:     1,
		PageSize: 200,
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	assert.Equal(t, "ENB00001", result.Items[0].DeviceSN)
	assert.Equal(t, "QAFA", result.Items[0].ProductType)
	assert.Equal(t, "ENB00002", result.Items[1].DeviceSN)
	assert.Equal(t, "FAP/BU1810", result.Items[1].ProductType)
}

func TestService_DeleteTaskType_DeletesCustomTypeOnly(t *testing.T) {
	repo := &ensureBuiltInTaskTypeRepo{
		items: []TaskType{
			{
				TypeCode:      "ENB_IMG_UPGRADE",
				Category:      "enb_upgrade",
				CategoryLabel: "4G升级",
				DisplayName:   "4G 基站软件升级",
				BuiltIn:       true,
			},
			{
				TypeCode:      "CUSTOM_UPLOAD_SAMPLE",
				Category:      "custom_upload",
				CategoryLabel: "自定义上传",
				DisplayName:   "自定义上传模板",
				BuiltIn:       false,
			},
		},
	}
	svc := NewService(nil, repo, nil, nil, nil, zap.NewNop())

	err := svc.DeleteTaskType(context.Background(), "CUSTOM_UPLOAD_SAMPLE")
	require.NoError(t, err)
	assert.Equal(t, []string{"CUSTOM_UPLOAD_SAMPLE"}, repo.deleted)

	err = svc.DeleteTaskType(context.Background(), "ENB_IMG_UPGRADE")
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrForbidden)
}

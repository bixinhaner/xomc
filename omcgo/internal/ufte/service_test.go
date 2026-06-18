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
		"CONFIG_BACKUP_XML":   {},
			"CONFIG_BACKUP_NV":    {},
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
	assert.Equal(t, "config_backup", seen["CONFIG_BACKUP_XML"].Category)
	assert.Equal(t, "config_restore", seen["CONFIG_RESTORE"].Category)
	assert.Equal(t, "1 Firmware Upgrade Image", seen["ENB_IMG_UPGRADE"].FileType)
	assert.Equal(t, "X {OUI} Software Upgrade Patch", seen["ENB_PATCH_UPGRADE"].FileType)
	assert.Equal(t, "Firmware Upgrade Fpga", seen["ENB_FPGA_UPGRADE"].FileType)
	require.NotNil(t, seen["ENB_IMG_UPGRADE"].FirmwareFileType)
	assert.Equal(t, software.FileTypeIMG, *seen["ENB_IMG_UPGRADE"].FirmwareFileType)
	require.NotNil(t, seen["ENB_PATCH_UPGRADE"].FirmwareFileType)
	assert.Equal(t, software.FileTypePATCH, *seen["ENB_PATCH_UPGRADE"].FirmwareFileType)
	require.NotNil(t, seen["ENB_FPGA_UPGRADE"].FirmwareFileType)
	assert.Equal(t, software.FileTypeFPGA, *seen["ENB_FPGA_UPGRADE"].FirmwareFileType)
	_, has5GFpga := seen["GNB_FPGA_UPGRADE"]
	assert.False(t, has5GFpga)
}

func TestService_LoadTaskTypeCatalog_UsesStoredRowsAsSourceOfTruth(t *testing.T) {
	repo := &ensureBuiltInTaskTypeRepo{
		items: []TaskType{
			{
				TypeCode:         "ENB_IMG_UPGRADE",
				Category:         "enb_upgrade",
				CategoryLabel:    "4G升级",
				DisplayName:      "库里的 4G 升级模板",
				Description:      "from db",
				RPCType:          "DOWNLOAD",
				BuiltIn:          true,
				Enabled:          true,
				StepChain:        []string{"CHECK_PERMISSION", "SEND_RPC"},
				PermissionCode:   "CODE_ENB_UPGRADE_IMAGE",
				PlatformScope:    []string{"4G eNB"},
				FileType:         "1 Firmware Upgrade Image",
				FileTypeLabel:    "1 Firmware Upgrade Image",
				FirmwareFileType: firmwareFileTypePtr(software.FileTypeIMG),
				LastEditor:       "system",
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
	require.NotNil(t, first.FirmwareFileType)
	assert.Equal(t, software.FileTypeIMG, *first.FirmwareFileType)

	custom, ok := findTaskTypeByCode(catalog, "CUSTOM_UPLOAD_SAMPLE")
	require.True(t, ok)
	assert.Equal(t, "自定义上传模板", custom.DisplayName)
	assert.Zero(t, custom.softwareTaskType)
	assert.Nil(t, custom.techHint)
}

func TestMaterializeTaskTypes_FillsMissingFirmwareFileType(t *testing.T) {
	catalog := materializeTaskTypes([]TaskType{
		{
			TypeCode:      "ENB_PATCH_UPGRADE",
			Category:      "enb_upgrade",
			CategoryLabel: "4G升级",
			DisplayName:   "库里的补丁模板",
			RPCType:       "DOWNLOAD",
			BuiltIn:       true,
			Enabled:       true,
			StepChain:     []string{"SEND_RPC"},
			PlatformScope: []string{"QAFA"},
			FileType:      "X {OUI} Software Upgrade Patch",
		},
		{
			TypeCode:      "CUSTOM_IMG_TEMPLATE",
			Category:      "enb_upgrade",
			CategoryLabel: "4G升级",
			DisplayName:   "自定义镜像模板",
			RPCType:       "DOWNLOAD",
			Enabled:       true,
			StepChain:     []string{"SEND_RPC"},
			PlatformScope: []string{"QAFA"},
			FileType:      "1 Firmware Upgrade Image",
		},
	})

	patch, ok := findTaskTypeByCode(catalog, "ENB_PATCH_UPGRADE")
	require.True(t, ok)
	require.NotNil(t, patch.FirmwareFileType)
	assert.Equal(t, software.FileTypePATCH, *patch.FirmwareFileType)

	custom, ok := findTaskTypeByCode(catalog, "CUSTOM_IMG_TEMPLATE")
	require.True(t, ok)
	require.NotNil(t, custom.FirmwareFileType)
	assert.Equal(t, software.FileTypeIMG, *custom.FirmwareFileType)
}

func TestNormalizeTaskTypeFileType_DownloadTemplatesUseFinalCWMPString(t *testing.T) {
	assert.Equal(t, "1 Firmware Upgrade Image", normalizeTaskTypeFileType("DOWNLOAD", "1"))
	assert.Equal(t, "3 Vendor Configuration File", normalizeTaskTypeFileType("DOWNLOAD", "config"))
	assert.Equal(t, "Firmware Upgrade Fpga", normalizeTaskTypeFileType("DOWNLOAD", "Firmware Upgrade Fpga"))
	assert.Equal(t, "6", normalizeTaskTypeFileType("UPLOAD", "6"))
}

// #492：候选过滤走 product_scope（产品英文名）。模板 Products 非空 → 设备 productClass
// 经 productNameLookup 解析出产品名，命中列表才入候选；候选回填 ProductName。
func TestService_ListDeviceCandidates_FiltersByProductName(t *testing.T) {
	// 自定义一条 ENB_IMG_UPGRADE，限定两个产品；materializeTaskTypes 会按 TypeCode
	// 叠加内置的 softwareTaskType/techHint，但保留这里设置的 Products。
	taskTypeRepo := &ensureBuiltInTaskTypeRepo{
		items: []TaskType{{
			TypeCode:      "ENB_IMG_UPGRADE",
			Category:      "enb_upgrade",
			CategoryLabel: "4G升级",
			Products:      []string{"甲产品", "乙产品"},
		}},
	}
	deviceRepo := &stubDeviceRepo{listResponse: &coremodel.ListResponse[coremodel.Device]{
		Items: []coremodel.Device{
			{ID: uuid.New(), SerialNumber: "ENB00001", ProductClass: "QAFA", Technology: coremodel.TechLTE, FirmwareVersion: "V1.0.0", DeviceName: "北京 4G 站点"},
			{ID: uuid.New(), SerialNumber: "ENB00002", ProductClass: "FAP/BU1810", Technology: coremodel.TechLTE, FirmwareVersion: "V1.0.1", DeviceName: "广州 4G 站点"},
			{ID: uuid.New(), SerialNumber: "ENB00003", ProductClass: "FAP/OTHER", Technology: coremodel.TechLTE, FirmwareVersion: "V1.0.2", DeviceName: "深圳 4G 站点"},
		},
		Total:    3,
		Page:     1,
		PageSize: 200,
	}}

	svc := NewService(nil, taskTypeRepo, nil, nil, deviceRepo, zap.NewNop())
	svc.productNameLookup = func(_ context.Context, pc string) (string, bool) {
		switch pc {
		case "QAFA":
			return "甲产品", true
		case "FAP/BU1810":
			return "乙产品", true
		case "FAP/OTHER":
			return "丙产品", true
		default:
			return "", false
		}
	}

	// 不带 productName → 命中模板 Products（甲/乙）的两台入候选，丙产品被排除。
	result, err := svc.ListDeviceCandidates(context.Background(), DeviceCandidateFilter{
		Category: "enb_upgrade",
		TypeCode: "ENB_IMG_UPGRADE",
		Page:     1,
		PageSize: 200,
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	assert.Equal(t, "ENB00001", result.Items[0].DeviceSN)
	assert.Equal(t, "甲产品", result.Items[0].ProductName)
	assert.Equal(t, "ENB00002", result.Items[1].DeviceSN)
	assert.Equal(t, "乙产品", result.Items[1].ProductName)

	// 带 productName=甲产品 → 进一步收窄到 1 台。
	narrowed, err := svc.ListDeviceCandidates(context.Background(), DeviceCandidateFilter{
		Category:    "enb_upgrade",
		TypeCode:    "ENB_IMG_UPGRADE",
		ProductName: "甲产品",
		Page:        1,
		PageSize:    200,
	})
	require.NoError(t, err)
	require.Len(t, narrowed.Items, 1)
	assert.Equal(t, "ENB00001", narrowed.Items[0].DeviceSN)
}

// #524：设备列表筛选改按产品名（DeviceItem.ProductName）。matchesDeviceFilter 命中
// ProductName 时大小写不敏感精确匹配，与「产品名称」列展示口径一致；ProductType 仍兼容。
func TestMatchesDeviceFilter_ByProductName(t *testing.T) {
	item := DeviceItem{
		DeviceSN:    "ENB00001",
		ProductType: "QAFA",   // productClass
		ProductName: "甲产品", // 由 mapDeviceItem 回填
	}

	tests := []struct {
		name   string
		filter DeviceListFilter
		want   bool
	}{
		{"产品名命中", DeviceListFilter{ProductName: "甲产品"}, true},
		{"传 productClass 当产品名→不命中", DeviceListFilter{ProductName: "QAFA"}, false}, // 按名过滤，不是按 class
		{"产品名不命中", DeviceListFilter{ProductName: "乙产品"}, false},
		{"空过滤器全放行", DeviceListFilter{}, true},
		{"productClass 口径仍兼容", DeviceListFilter{ProductType: "qafa"}, true},
		{"产品名 + class 同时命中", DeviceListFilter{ProductName: "甲产品", ProductType: "QAFA"}, true},
		{"产品名命中但 class 不命中→排除", DeviceListFilter{ProductName: "甲产品", ProductType: "OTHER"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, matchesDeviceFilter(item, tc.filter))
		})
	}
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

// TestExecutionModeForTask_SuspendedEcho 守护 #138：挂起任务必须回显 "suspended"。
// 统一挂起表示后，applyScheduleMode 与 CreatePlaceholderTrackingTask 两条链路的挂起
// 主任务都落 software.TaskSuspended + create_status=active，回显必须是 "suspended"
// 而非历史 bug 的 "immediate"。
func TestExecutionModeForTask_SuspendedEcho(t *testing.T) {
	tests := []struct {
		name         string
		status       software.TaskStatus
		createStatus string
		want         string
	}{
		{
			name:         "suspended task echoes suspended (#138)",
			status:       software.TaskSuspended,
			createStatus: software.CreateStatusActive,
			want:         "suspended",
		},
		{
			name:         "timing wins regardless of status",
			status:       software.TaskPending,
			createStatus: software.CreateStatusTiming,
			want:         "scheduled",
		},
		{
			name:         "timing wins even after scheduler flips status to in_progress",
			status:       software.TaskInProgress,
			createStatus: software.CreateStatusTiming,
			want:         "scheduled",
		},
		{
			name:         "in_progress active is immediate",
			status:       software.TaskInProgress,
			createStatus: software.CreateStatusActive,
			want:         "immediate",
		},
		{
			name:         "pending active is immediate (transient pre-dispatch)",
			status:       software.TaskPending,
			createStatus: software.CreateStatusActive,
			want:         "immediate",
		},
		{
			name:         "ended active is immediate",
			status:       software.TaskEnded,
			createStatus: software.CreateStatusActive,
			want:         "immediate",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, executionModeForTask(tc.status, tc.createStatus))
		})
	}
}

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
		"UPS_AP_UPGRADE":      {},
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
	assert.Equal(t, "config-backup/{object_path}", seen["CONFIG_RESTORE"].URLTemplate)
	assert.Equal(t, "/smallcell/FileDownloadService/config-backup/{object_path}", seen["CONFIG_RESTORE"].TransportPath)
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

func TestBuiltInTaskTypes_UpgradeProductScopesMatchTechnology(t *testing.T) {
	defaults := builtInTaskTypes()
	seen := make(map[string]TaskType, len(defaults))
	for _, item := range defaults {
		seen[item.TypeCode] = item
	}

	enbProducts := []string{
		"BLQ",
		"BLX",
		"QRTB",
		"MLQ",
		"MLN",
		"BM",
		"CICT SC3400(L1821)",
		"Datang fBS3251 Series",
		"Third-party FDD-LTE-Enterprise",
		"Huawei TCELL Series",
		"Comba LTE-FDD_N Series",
		"Comba femto_au",
	}
	for _, code := range []string{"ENB_IMG_UPGRADE", "ENB_PATCH_UPGRADE", "ENB_FPGA_UPGRADE"} {
		require.Contains(t, seen, code)
		assert.Equal(t, enbProducts, seen[code].Products)
		assert.NotContains(t, seen[code].Products, "BNQ")
		assert.NotContains(t, seen[code].Products, "BSC")
		assert.NotContains(t, seen[code].Products, "BTS")
	}

	require.Contains(t, seen, "GNB_IMG_UPGRADE")
	assert.Equal(t, []string{"BNQ"}, seen["GNB_IMG_UPGRADE"].Products)

	require.Contains(t, seen, "GSM_IMG_UPGRADE")
	assert.Equal(t, []string{"BSC", "BTS"}, seen["GSM_IMG_UPGRADE"].Products)
}

func TestMaterializeTaskTypes_NormalizesLegacyConfigBackupReferences(t *testing.T) {
	catalog := materializeTaskTypes([]TaskType{{
		TypeCode:      "CONFIG_RESTORE",
		Category:      "config_restore",
		RPCType:       "DOWNLOAD",
		BuiltIn:       true,
		Enabled:       true,
		URLTemplate:   "config_backup/{object_path}",
		TransportPath: "/smallcell/FileDownloadService/config_backup/{object_path}",
	}})

	require.Len(t, catalog, 1)
	assert.Equal(t, "config-backup/{object_path}", catalog[0].URLTemplate)
	assert.Equal(t, "/smallcell/FileDownloadService/config-backup/{object_path}", catalog[0].TransportPath)
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
	}, nil)
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
	}, nil)
	require.NoError(t, err)
	require.Len(t, narrowed.Items, 1)
	assert.Equal(t, "ENB00001", narrowed.Items[0].DeviceSN)
}

// stubGroupReader 满足 authz.GroupReader：按 deviceID 返回其所属设备组。
type stubGroupReader struct {
	groups map[uuid.UUID][]uuid.UUID
	calls  int
}

func (s *stubGroupReader) GetDeviceGroupIDs(_ context.Context, deviceID uuid.UUID) ([]uuid.UUID, error) {
	s.calls++
	return s.groups[deviceID], nil
}

// TestDeviceVisibility_ThreeStateContract 锁定 #63 读路径设备组过滤的三态语义：
// 超管(nil)放行全部、空集 fail-closed、限定组按交集放行；并验证 nil 设备 fail-closed
// 与按 deviceID 缓存（避免大列表重复查库）。
func TestDeviceVisibility_ThreeStateContract(t *testing.T) {
	g1 := uuid.New()
	g2 := uuid.New()
	devInG1 := uuid.New()
	devInG2 := uuid.New()
	reader := &stubGroupReader{groups: map[uuid.UUID][]uuid.UUID{
		devInG1: {g1},
		devInG2: {g2},
	}}
	svc := &Service{groupReader: reader}
	ctx := context.Background()

	// 超管 nil → 恒放行，且不查库（calls 不增）。
	vfNil := svc.newDeviceVisibility(nil)
	assert.True(t, vfNil.allow(ctx, devInG1))
	assert.True(t, vfNil.allow(ctx, devInG2))
	assert.Equal(t, 0, reader.calls, "超管不应触发 groupReader 查询")

	// 空集 → fail-closed，全部不可见。
	vfEmpty := svc.newDeviceVisibility([]uuid.UUID{})
	assert.False(t, vfEmpty.allow(ctx, devInG1))

	// 限定 g1 → 仅 g1 下设备可见。
	vfG1 := svc.newDeviceVisibility([]uuid.UUID{g1})
	assert.True(t, vfG1.allow(ctx, devInG1))
	assert.False(t, vfG1.allow(ctx, devInG2))

	// nil 设备 → fail-closed（不泄漏未归组孤儿行）。
	assert.False(t, vfG1.allow(ctx, uuid.Nil))

	// 缓存命中：同一 deviceID 再判定不应再查库。
	before := reader.calls
	assert.True(t, vfG1.allow(ctx, devInG1))
	assert.Equal(t, before, reader.calls, "重复 deviceID 应命中缓存，不再查库")
}

// pagingSubTaskRepo 按 ListByTaskID 分页契约切片返回，单页上限 100（与 pg 实现一致），
// 用于验证 listAllSubTasksByTaskID 会翻页取全量。
type pagingSubTaskRepo struct {
	software.SubTaskRepository
	all []software.UpgradeSubTaskWithTaskName
}

func (r *pagingSubTaskRepo) ListByTaskID(_ context.Context, _ uuid.UUID, filter software.SubTaskFilter) (*coremodel.ListResponse[software.UpgradeSubTaskWithTaskName], error) {
	ps := filter.PageSize
	if ps < 1 {
		ps = 20
	}
	if ps > 100 {
		ps = 100
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	total := len(r.all)
	start := (page - 1) * ps
	if start > total {
		start = total
	}
	end := start + ps
	if end > total {
		end = total
	}
	totalPages := (total + ps - 1) / ps
	if totalPages == 0 {
		totalPages = 1
	}
	return &coremodel.ListResponse[software.UpgradeSubTaskWithTaskName]{
		Items: r.all[start:end], Total: int64(total),
		Page: page, PageSize: ps, TotalPages: totalPages,
	}, nil
}

// TestListAllSubTasksByTaskID_PagesBeyondCap 锁定 H3：>100 台的任务必须翻页取全量，
// 不能被默认首页(20)/单页上限(100)截断——否则归属校验/直接派发漏覆盖尾部设备。
func TestListAllSubTasksByTaskID_PagesBeyondCap(t *testing.T) {
	const n = 250
	all := make([]software.UpgradeSubTaskWithTaskName, n)
	for i := range all {
		all[i].DeviceID = uuid.New()
	}
	svc := &Service{subTaskRepo: &pagingSubTaskRepo{all: all}}

	got, err := svc.listAllSubTasksByTaskID(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, got, n, "应翻页取回全部 250 条，而非被截断到 20/100")
}

// TestStartDirectDispatch_RejectsNonResumableStatus 锁定 H2：对非 suspended/pending
// 的 CONFIG_RESTORE/LICENSE 任务点「开始」必须被拒，避免二次派发（现网重复刷配置）。
func TestStartDirectDispatch_RejectsNonResumableStatus(t *testing.T) {
	svc := &Service{}
	for _, st := range []software.TaskStatus{software.TaskEnded, software.TaskInProgress} {
		task := &software.UpgradeTask{ID: uuid.New(), Status: st}
		err := svc.startDirectDispatchTask(context.Background(), task, &TaskType{TypeCode: "CONFIG_RESTORE"})
		require.Error(t, err, "status=%s 应被拒绝", st)
		assert.Contains(t, err.Error(), "not suspended or pending")
	}
}

// #524：设备列表筛选改按产品名（DeviceItem.ProductName）。matchesDeviceFilter 命中
// ProductName 时大小写不敏感精确匹配，与「产品名称」列展示口径一致；ProductType 仍兼容。
func TestMatchesDeviceFilter_ByProductName(t *testing.T) {
	item := DeviceItem{
		DeviceSN:    "ENB00001",
		ProductType: "QAFA", // productClass
		ProductName: "甲产品",  // 由 mapDeviceItem 回填
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

// #615：任务详情抽屉「已选设备列表」走 GET /ufte/devices?taskId=...，
// matchesDeviceFilter 按 DeviceItem.TaskID（来自 upgrade_sub_tasks.task_id）
// 精确匹配；空 TaskID ≡ 全量「执行明细」入口语义不变，保证向后兼容。
func TestMatchesDeviceFilter_ByTaskID(t *testing.T) {
	item := DeviceItem{
		DeviceSN:    "ENB00001",
		TaskID:      "task-aaa",
		Status:      "downloading",
		Category:    "enb_upgrade",
		TypeCode:    "ENB_IMG_UPGRADE",
		ProductType: "QAFA",
	}

	tests := []struct {
		name   string
		filter DeviceListFilter
		want   bool
	}{
		{"任务命中", DeviceListFilter{TaskID: "task-aaa"}, true},
		{"任务不命中", DeviceListFilter{TaskID: "task-bbb"}, false},
		{"空 taskId 全放行（保留旧行为）", DeviceListFilter{}, true},
		{"任务+状态都命中", DeviceListFilter{TaskID: "task-aaa", Status: "downloading"}, true},
		{"任务命中但状态不命中→排除", DeviceListFilter{TaskID: "task-aaa", Status: "ended"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, matchesDeviceFilter(item, tc.filter))
		})
	}
}

// #615 后续：parseTaskIDFilter 是 DeviceListFilter.TaskID(string) → AllSubTaskFilter.TaskID(*uuid.UUID)
// 的 SQL 下推参数适配器。空串=不下推（保留旧行为），合法 UUID=下推，非法格式→ErrInvalidInput(400)。
func TestParseTaskIDFilter(t *testing.T) {
	valid := uuid.New()

	t.Run("空串 → 不下推", func(t *testing.T) {
		got, err := parseTaskIDFilter("")
		assert.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("合法 UUID → 透传", func(t *testing.T) {
		got, err := parseTaskIDFilter(valid.String())
		assert.NoError(t, err)
		if assert.NotNil(t, got) {
			assert.Equal(t, valid, *got)
		}
	})

	t.Run("非法格式 → ErrInvalidInput", func(t *testing.T) {
		got, err := parseTaskIDFilter("not-a-uuid")
		assert.Nil(t, got)
		assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	})
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

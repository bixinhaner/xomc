package ufte

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/imsparam"
	"github.com/omcgo/omcgo/internal/software"
)

// ProductTechLookup 是 ufte 给设备候选列表 productClass 过滤用的最小依赖：
// 输入 device.product_class 字符串，输出该产品在 products 字典里的 tech 标识
// （"lte" / "nr" / "gsm"）。命中返回 (tech, true)；未注册 / 字典缺失 / 字典未装配
// 都返回 ("", false)，调用方退化到关键字模糊匹配（matchesTaskTypeScope 内）。
//
// 抽小接口而不是直接持有 *product.Registry 是为单测和解耦——provider/modules.go
// 用闭包把真实 Registry.MatchProductClass 包成这个签名注入。
type ProductTechLookup func(ctx context.Context, productClass string) (tech string, ok bool)

// ProductNameLookup 按设备 productClass 经 ProductRegistry 反查产品英文名（products.product_name）。
// #492：模板 product_scope（适用产品名列表）非空时，设备候选匹配用它做"产品目录精确匹配"。
type ProductNameLookup func(ctx context.Context, productClass string) (name string, ok bool)

type LicenseFeatureChecker interface {
	CheckFeature(ctx context.Context, path string) (bool, error)
}

type Service struct {
	softwareService *software.SoftwareService
	taskTypeRepo    TaskTypeRepository
	taskRepo        software.TaskRepository
	subTaskRepo     software.SubTaskRepository
	deviceRepo      device.DeviceRepository
	logger          *zap.Logger
	// productTechLookup 为 nil 时设备过滤退化为旧关键字匹配；推荐生产部署一定注入,
	// 否则像 FAP/BSC7041C243 这种 BNQ 产品因为字符串没 "5G/GNB/BBU" 关键字,
	// 5G 升级任务的设备候选会整批漏选（用户实测反馈过）。
	productTechLookup ProductTechLookup
	// productNameLookup 注入式：productClass → 产品英文名。nil 时 product_scope 匹配整批漏选
	// （退回为：product_scope 非空但查不到产品名 → 不放行），推荐生产部署一定注入。
	productNameLookup ProductNameLookup
	// downloadURLLookup 注入式回调：(sn, fileName) → presigned GET URL（1h 有效）。
	// nil 表示部署未装配 backup.FileRepository / MinIO，DeviceItem.DownloadURL 留空。
	// 返回 ("", nil) 表示元数据缺失（CPE 还没传完）；返回 ("", err) 仅在 DB/MinIO
	// 设施故障时出现，不阻断 mapping（外层降级为空 URL + 记日志）。
	// 与 backup.FileRepository 解耦，避免 ufte → backup 反向依赖(backup → ufte 已存在)。
	downloadURLLookup func(ctx context.Context, sn, fileName string) (string, error)

	// fileLandedLookup 注入式回调：按 (sn, mainTaskID) 精确反查 backup_restore_file。
	//
	// 设备厂商（如 baicells/MMMM）实际 PUT 用自己内部 NV 文件名（"mib-home-fap.nv"），
	// 跟 OMC 端预渲染模板无关；不同任务可能上传同一个 file_name，因此匹配键不能用
	// fileName。从 migrations/000133 起 backup_restore_file 加了 task_id 字段，
	// 唯一键 (sn, task_id, file_name) 保证不同任务隔离。这里用 (sn, mainTaskID)
	// 精确反查当前任务的落地记录，返回设备真实文件名。
	// nil → DeviceItem.TargetFile / DownloadURL 永远留空。
	fileLandedLookup func(ctx context.Context, sn, mainTaskID string) (fileName string, landed bool, err error)

	// fileDeletedLookup 注入式回调：按 (sn, mainTaskID, fileName) 判断 UFTE 任务文件
	// 元数据是否已标记删除。true 时前端保留文件名但禁用下载入口。
	fileDeletedLookup func(ctx context.Context, sn, mainTaskID, fileName string) (bool, error)

	// snapshotConfigRestoreDispatcher (T-0164)：CONFIG_RESTORE 任务的实际派发器。
	// 接收设备 SN 列表，从 config_snapshots 表取每设备最新一份配置 → 缺失整批拒绝
	// → 全部存在则创建 restore_tasks 主行 + 逐设备 Download device_tasks。
	// nil → CONFIG_RESTORE 任务创建直接返回 not configured。
	// 由 backup.RestoreService 满足该接口；wiring 在 cmd/app/provider/modules.go。
	snapshotConfigRestoreDispatcher SnapshotConfigRestoreDispatcher

	// licenseUpgradeDispatcher (T-0165)：LICENSE_UPGRADE 任务派发器（与上方
	// CONFIG_RESTORE 对称）。接 SN 列表 → device_licenses 表取每设备 license →
	// 缺失整批拒绝 → 逐设备 Download device_tasks（FileType="License File"）。
	// nil → LICENSE_UPGRADE 任务创建返回 not configured。
	// 由 backup.LicenseService 满足；wiring 在 cmd/app/provider/modules.go。
	licenseUpgradeDispatcher LicenseUpgradeDispatcher

	// imsParamDispatcher：IMS_FILE_DISTRIBUTE 任务派发器（docs/design/imscore-file-transfer.md）。
	// 接 SN 列表 + 参数文件 ID → ims_param_files 取文件 → 逐设备 Download
	// device_tasks（FileType="ImsCore Parameters File"，URL 带 paramType query）。
	// nil → IMS_FILE_DISTRIBUTE 任务创建返回 not configured。
	// 由 imsparam.Service 满足；wiring 在 cmd/app/provider/modules.go。
	imsParamDispatcher ImsParamDispatcher

	// groupReader 是 #63 设备组可见性强制层的按设备归属读取器（device_group_members）。
	// 由 device.NewPgDeviceGroupReader 满足；nil 表示 dev/test 退化（authz 包 nil-safe，
	// 不降低生产安全：生产路由始终注入 reader）。CreateTask 对 req.DeviceIDs 整批校验、
	// 各操作端点对「任务关联设备」反查后校验，统一委派 authz 包，避免越权语义在此复刻。
	groupReader authz.GroupReader

	licenseFeatureChecker LicenseFeatureChecker
}

// SetGroupReader 注入设备组归属读取器（#63 租户隔离强制层）。生产装配必注入；
// 不注入则 authz 退化为不强制（与 device/alarm nil-safe 语义一致）。
func (s *Service) SetGroupReader(reader authz.GroupReader) {
	s.groupReader = reader
}

func (s *Service) SetLicenseFeatureChecker(checker LicenseFeatureChecker) {
	s.licenseFeatureChecker = checker
}

func (s *Service) upsFeatureLicensed(ctx context.Context) bool {
	if s.licenseFeatureChecker == nil {
		return true
	}
	ok, err := s.licenseFeatureChecker.CheckFeature(ctx, "UPS.Monitor")
	if err != nil {
		return false
	}
	return ok
}

func isUPSTaskType(item TaskType) bool {
	return item.Category == "ups_upgrade" ||
		item.TypeCode == "UPS_AP_UPGRADE" ||
		taskTypeProductsContain(item.Products, "UPS")
}

func isUPSTask(item Task) bool {
	return item.Category == "ups_upgrade" ||
		item.TypeCode == "UPS_AP_UPGRADE" ||
		strings.EqualFold(strings.TrimSpace(item.ProductName), "UPS") ||
		strings.HasPrefix(strings.ToUpper(strings.TrimSpace(item.ProductType)), "UPS")
}

func isUPSDeviceItem(item DeviceItem) bool {
	return item.Category == "ups_upgrade" ||
		item.TypeCode == "UPS_AP_UPGRADE" ||
		strings.EqualFold(strings.TrimSpace(item.ProductName), "UPS") ||
		strings.HasPrefix(strings.ToUpper(strings.TrimSpace(item.ProductType)), "UPS")
}

func filterUPSTaskTypesByLicense(catalog []TaskType, upsLicensed bool) []TaskType {
	if upsLicensed {
		return catalog
	}
	filtered := make([]TaskType, 0, len(catalog))
	for _, item := range catalog {
		if isUPSTaskType(item) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func (s *Service) rejectUPSFeatureIfUnlicensed(ctx context.Context, item TaskType) error {
	if !isUPSTaskType(item) || s.upsFeatureLicensed(ctx) {
		return nil
	}
	return fmt.Errorf("UPS feature is not authorized: %w", commonerrors.ErrLicenseFeatureNotAuthorized)
}

// authorizeDevices 对一批设备 ID 做归属校验（CreateTask 用）。任一越权整批拒绝。
func (s *Service) authorizeDevices(ctx context.Context, visibleGroups []uuid.UUID, deviceIDs ...uuid.UUID) error {
	return authz.AuthorizeDevicesAccess(ctx, s.groupReader, visibleGroups, deviceIDs...)
}

// listAllSubTasksByTaskID 翻页取「任务的全部子任务」。ListByTaskID 单页上限 100，
// 不翻页就只拿默认首页（20 条）——大批量任务（>20/>100 台）会导致 #63 归属校验
// 漏校验 21+ 台、CONFIG_RESTORE/LICENSE 直接派发只发前 20 台还标全成功。这里循环
// 取尽所有页，保证按任务的校验/派发覆盖每一台设备。
func (s *Service) listAllSubTasksByTaskID(ctx context.Context, taskID uuid.UUID) ([]software.UpgradeSubTaskWithTaskName, error) {
	const pageSize = 100
	items := make([]software.UpgradeSubTaskWithTaskName, 0)
	for page := 1; ; page++ {
		res, err := s.subTaskRepo.ListByTaskID(ctx, taskID, software.SubTaskFilter{
			TaskID:      taskID,
			ListRequest: coremodel.ListRequest{Page: page, PageSize: pageSize},
		})
		if err != nil {
			return nil, err
		}
		items = append(items, res.Items...)
		if page >= res.TotalPages || len(res.Items) == 0 {
			break
		}
	}
	return items, nil
}

// authorizeTaskDevices 反查任务关联设备（从 upgrade_sub_tasks 的 device_id）后逐个
// 校验归属，用于 Start/Suspend/Terminate/Delete/Retry 等按任务操作的端点。
//
//	visibleGroups == nil      → 超管：直接放行，不查 sub_tasks。
//	groupReader == nil        → dev/test 退化：放行（authz nil-safe）。
//	任务无任何子任务设备       → 无可越权对象，放行（删一个空壳任务不构成跨租户读写）。
//	任一子任务设备越权         → 整体拒绝（ErrForbidden）。
func (s *Service) authorizeTaskDevices(ctx context.Context, taskID uuid.UUID, visibleGroups []uuid.UUID) error {
	if visibleGroups == nil || s.groupReader == nil {
		return nil
	}
	if s.subTaskRepo == nil {
		return nil
	}
	// #63 + H3：必须取全量子任务（翻页），否则 >20 台的任务只校验前 20 台归属，
	// 第 21+ 台可越权。
	subTasks, err := s.listAllSubTasksByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("list sub_tasks for task authz: %w", err)
	}
	seen := make(map[uuid.UUID]struct{}, len(subTasks))
	ids := make([]uuid.UUID, 0, len(subTasks))
	for i := range subTasks {
		did := subTasks[i].DeviceID
		if did == uuid.Nil {
			continue
		}
		if _, ok := seen[did]; ok {
			continue
		}
		seen[did] = struct{}{}
		ids = append(ids, did)
	}
	if len(ids) == 0 {
		return nil
	}
	return authz.AuthorizeDevicesAccess(ctx, s.groupReader, visibleGroups, ids...)
}

// deviceVisibility 是 #63 读路径的设备组可见性过滤器，复用与写路径同款的 groupReader，
// 把列表/导出/候选收窄到调用者可见设备组下的设备。三态契约同 authz 包：
//
//	visibleGroups == nil  → 超管：恒放行（不查库、不改变现网行为）。
//	groupReader == nil    → dev/test 退化：恒放行（authz nil-safe）。
//	[]uuid.UUID{}         → 已认证但无任何分组：全部不可见（fail-closed）。
//	[g1, g2, ...]         → 设备分组与之有交集才可见。
//
// 结果按 deviceID 缓存，避免大列表逐行重复查 device_group_members。
type deviceVisibility struct {
	groups []uuid.UUID
	reader authz.GroupReader
	cache  map[uuid.UUID]bool
}

func (s *Service) newDeviceVisibility(visibleGroups []uuid.UUID) *deviceVisibility {
	return &deviceVisibility{groups: visibleGroups, reader: s.groupReader, cache: make(map[uuid.UUID]bool)}
}

// allow 返回该设备是否对调用者可见。超管 / 未装配强制 → 恒 true。
func (f *deviceVisibility) allow(ctx context.Context, deviceID uuid.UUID) bool {
	if f == nil || f.groups == nil || f.reader == nil {
		return true
	}
	// 子任务异常未关联设备：fail-closed，不把未归组的孤儿行泄漏给限权用户。
	if deviceID == uuid.Nil {
		return false
	}
	if v, ok := f.cache[deviceID]; ok {
		return v
	}
	allowed := authz.AuthorizeDeviceAccess(ctx, f.reader, deviceID, f.groups) == nil
	f.cache[deviceID] = allowed
	return allowed
}

// SnapshotConfigRestoreDispatcher 是 UFTE 对 backup.RestoreService.CreateBySnapshot
// 的最小依赖。命名带 Snapshot 表明文件源恒来自 config_snapshots 表（每设备最新一份）。
//
// 返回：
//   - dispatchedID：成功时返回 restore_tasks 主行 UUID（外层包装为 ufte.Task.ID）
//   - missing：缺失快照的 SN 列表；非空时调用方应返 ErrNotFound 给上层
//   - err：基础设施错误（DB/MinIO 不可用等）
type SnapshotConfigRestoreDispatcher interface {
	// upgradeTaskID 是 ufte 提前创建的占位 upgrade_tasks ID，dispatcher 用它
	// 派生统一 CommandKey（与 sub_task 一致），让 TC 回流可自动推进。
	// 传 uuid.Nil 时退化用 restore_tasks.id-derived CommandKey（兼容旧调用，
	// 没有 TC 自动跟踪）。
	//
	// dispatchedFiles[sn] 是每台设备实际下发的快照文件名（basename）。UFTE 收到后
	// 写回 sub_task.dest_version，作为"目标文件"列展示数据来源。
	DispatchConfigRestoreBySnapshot(
		ctx context.Context, targetDeviceSNs []string, createUser string, upgradeTaskID uuid.UUID,
	) (dispatchedID uuid.UUID, dispatchedFiles map[string]string, missing []string, err error)

	// PreviewConfigRestoreFiles 只查不派发：返回每台设备如果派发会下发的快照文件名。
	// 给"挂起 / 定时"模式占位任务用：创建时设备已选定、snapshot 已确定，
	// 即使派发推迟也应立即把目标文件名写到 sub_task.dest_version 让前端"目标文件"列可见。
	PreviewConfigRestoreFiles(
		ctx context.Context, targetDeviceSNs []string,
	) (map[string]string, error)
}

// SetSnapshotConfigRestoreDispatcher 注入 CONFIG_RESTORE 派发器（T-0164）。
// 不调用则 CONFIG_RESTORE 任务创建始终返回 ErrInvalidInput。
func (s *Service) SetSnapshotConfigRestoreDispatcher(d SnapshotConfigRestoreDispatcher) {
	s.snapshotConfigRestoreDispatcher = d
}

// LicenseUpgradeDispatcher 是 UFTE 对 backup.LicenseService.DispatchLicenseUpgradeBySN
// 的最小依赖。命名体现"从 device_licenses 取每设备最新一份 license 派发 Download"。
//
// 返回：
//   - dispatchedID：成功时返回派发主 ID（UFTE 用作 placeholder Task.ID）
//   - missing：缺失 license 的 SN 列表；非空时调用方应返 ErrNotFound
//   - err：基础设施错误
type LicenseUpgradeDispatcher interface {
	// upgradeTaskID 是 ufte 提前创建的占位 upgrade_tasks 主行 ID。dispatcher
	// 用它派生统一 CommandKey（与 sub_task.CommandKey 一致），TC 到达可自动推进。
	//
	// dispatchedFiles[sn] 是每台设备实际下发的 license 文件名（basename）。UFTE 收到后
	// 写回 sub_task.dest_version，作为"目标文件"列展示数据来源。
	DispatchLicenseUpgradeBySN(
		ctx context.Context, targetDeviceSNs []string, createUser string, upgradeTaskID uuid.UUID,
	) (dispatchedID uuid.UUID, dispatchedFiles map[string]string, missing []string, err error)

	// PreviewLicenseFiles 只查不派发：见 SnapshotConfigRestoreDispatcher.PreviewConfigRestoreFiles 注释。
	PreviewLicenseFiles(
		ctx context.Context, targetDeviceSNs []string,
	) (map[string]string, error)
}

// SetLicenseUpgradeDispatcher 注入 LICENSE_UPGRADE 派发器（T-0165）。
func (s *Service) SetLicenseUpgradeDispatcher(d LicenseUpgradeDispatcher) {
	s.licenseUpgradeDispatcher = d
}

// ImsParamDispatcher 是 UFTE 对 imsparam.Service 的最小依赖（核心网参数下发）。
//
// 返回：
//   - dispatchedID：成功时返回派发主 ID（UFTE 用作 placeholder Task.ID）
//   - dispatchedFiles[sn]：每台设备实际下发的参数文件名（写回 sub_task.dest_version）
//   - err：基础设施错误（文件缺失等以 ErrNotFound 包装返回）
type ImsParamDispatcher interface {
	DispatchImsParamByFileID(
		ctx context.Context, targetDeviceSNs []string, fileID uuid.UUID, createUser string, upgradeTaskID uuid.UUID,
	) (dispatchedID uuid.UUID, dispatchedFiles map[string]string, err error)
	// PreviewImsParamFile 只查不派发：返回待下发文件（校验存在性 + 目标文件名展示）。
	PreviewImsParamFile(ctx context.Context, fileID uuid.UUID) (fileName string, paramType string, err error)
}

// SetImsParamDispatcher 注入 IMS 文件下发派发器。
func (s *Service) SetImsParamDispatcher(d ImsParamDispatcher) {
	s.imsParamDispatcher = d
}

func NewService(
	softwareService *software.SoftwareService,
	taskTypeRepo TaskTypeRepository,
	taskRepo software.TaskRepository,
	subTaskRepo software.SubTaskRepository,
	deviceRepo device.DeviceRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		softwareService: softwareService,
		taskTypeRepo:    taskTypeRepo,
		taskRepo:        taskRepo,
		subTaskRepo:     subTaskRepo,
		deviceRepo:      deviceRepo,
		logger:          logger.Named("ufte-service"),
	}
}

// SetProductTechLookup 注入"按 productClass 查 product.tech"的回调。生产环境
// 必装—— ufte 任务设备过滤的 5G/4G 识别会走该 lookup 调 ProductRegistry，避免
// 老的 productClass 字符串模糊匹配漏识别（如 FAP/BSC7041C243 是 5G 但旧逻辑
// 没 BSC 关键字会被漏掉）。
func (s *Service) SetProductTechLookup(fn ProductTechLookup) {
	s.productTechLookup = fn
}

// SetProductNameLookup 注入"按 productClass 查产品英文名 product.Name"的回调（#492）。
// 模板 product_scope 非空时，设备候选按产品名精确匹配走该 lookup 调 ProductRegistry。
func (s *Service) SetProductNameLookup(fn ProductNameLookup) {
	s.productNameLookup = fn
}

// SetDownloadURLLookup 注入"按 (sn, fileName) 拿 presigned URL"的回调。
// 由 cmd/app/provider/modules.go 用 backup.FileRepository + *minio.Client 闭包装配；
// 不调用则 LogCollect 类设备列表的 DownloadURL 始终留空（UI 灰显文件名，不可点击）。
func (s *Service) SetDownloadURLLookup(fn func(ctx context.Context, sn, fileName string) (string, error)) {
	s.downloadURLLookup = fn
}

// SetFileLandedLookup 注入"按 (sn, mainTaskID) 精确反查 backup_restore_file"的回调。
// 不注入则 DeviceItem.TargetFile / DownloadURL 永远留空。
func (s *Service) SetFileLandedLookup(fn func(ctx context.Context, sn, mainTaskID string) (fileName string, landed bool, err error)) {
	s.fileLandedLookup = fn
}

// SetFileDeletedLookup 注入"按 (sn, mainTaskID, fileName) 判断文件元数据是否已标记删除"的回调。
// 不注入则 DeviceItem.FileDeleted 始终为 false。
func (s *Service) SetFileDeletedLookup(fn func(ctx context.Context, sn, mainTaskID, fileName string) (bool, error)) {
	s.fileDeletedLookup = fn
}

func (s *Service) GetOverview(ctx context.Context) (*Overview, error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	tasks, err := s.loadAllTasks(ctx, catalog, TaskListFilter{})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	cutoff := now.AddDate(0, 0, -30)
	running := 0
	totalDevices30d := 0
	successDevices30d := 0
	upsLicensed := s.upsFeatureLicensed(ctx)
	for _, task := range tasks {
		mapped, mapErr := s.mapTask(ctx, catalog, &task)
		if mapErr == nil && !upsLicensed && isUPSTask(*mapped) {
			continue
		}
		if task.Status == software.TaskPending || task.Status == software.TaskInProgress || task.Status == software.TaskSuspended {
			running++
		}
		createdAt := time.Time(task.CreatedAt)
		if createdAt.Before(cutoff) {
			continue
		}
		totalDevices30d += task.TotalCount
		successDevices30d += task.SuccessCount
	}
	successRate := 0.0
	if totalDevices30d > 0 {
		successRate = float64(successDevices30d) / float64(totalDevices30d) * 100
	}
	enabledCount := 0
	customCount := 0
	visibleCatalog := filterUPSTaskTypesByLicense(catalog, upsLicensed)
	for _, item := range visibleCatalog {
		if item.Enabled {
			enabledCount++
		}
		if !item.BuiltIn {
			customCount++
		}
	}
	return &Overview{
		EnabledTypeCount: enabledCount,
		RunningTaskCount: running,
		CustomTypeCount:  customCount,
		SuccessRate30d:   successRate,
	}, nil
}

func (s *Service) GetTaskTypes(ctx context.Context) ([]TaskType, error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := s.buildTaskTypeStats(ctx, catalog)
	if err != nil {
		return nil, err
	}
	for i := range catalog {
		if stat, ok := stats[catalog[i].TypeCode]; ok {
			catalog[i].TaskCount30d = stat.taskCount
			catalog[i].SuccessRate30d = stat.successRate
		}
	}
	return filterUPSTaskTypesByLicense(catalog, s.upsFeatureLicensed(ctx)), nil
}

func (s *Service) EnsureBuiltInTaskTypes(ctx context.Context) (int, error) {
	stored, err := s.taskTypeRepo.List(ctx)
	if err != nil {
		return 0, err
	}

	existingCodes := make(map[string]struct{}, len(stored))
	for _, item := range stored {
		existingCodes[item.TypeCode] = struct{}{}
	}

	inserted := 0
	for _, item := range builtInTaskTypes() {
		if _, exists := existingCodes[item.TypeCode]; exists {
			continue
		}
		taskType := item
		if err := s.taskTypeRepo.Upsert(ctx, &taskType); err != nil {
			return inserted, fmt.Errorf("ensure built-in UFTE task type %s: %w", item.TypeCode, err)
		}
		inserted++
	}

	return inserted, nil
}

func (s *Service) CreateTaskType(ctx context.Context, req TaskTypeWriteRequest, editor string) (*TaskType, error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	base := generateTaskTypeCodeBase(req.Category, req.DisplayName)
	typeCode := base
	for suffix := 2; ; suffix++ {
		if _, exists := findTaskTypeByCode(catalog, typeCode); !exists {
			break
		}
		typeCode = fmt.Sprintf("%s_%d", base, suffix)
	}
	item := taskTypeFromWriteRequest(typeCode, "CODE_"+typeCode, false, req, editor)
	if err := s.rejectUPSFeatureIfUnlicensed(ctx, item); err != nil {
		return nil, err
	}
	if err := s.taskTypeRepo.Upsert(ctx, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) UpdateTaskType(ctx context.Context, typeCode string, req TaskTypeWriteRequest, editor string) (*TaskType, error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	existing, ok := findTaskTypeByCode(catalog, typeCode)
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	permissionCode := existing.PermissionCode
	if permissionCode == "" {
		permissionCode = "CODE_" + typeCode
	}
	updated := taskTypeFromWriteRequest(typeCode, permissionCode, existing.BuiltIn, req, editor)
	updated.softwareTaskType = existing.softwareTaskType
	updated.techHint = existing.techHint
	updated.TaskCount30d = existing.TaskCount30d
	updated.SuccessRate30d = existing.SuccessRate30d
	if err := s.rejectUPSFeatureIfUnlicensed(ctx, updated); err != nil {
		return nil, err
	}
	if err := s.taskTypeRepo.Upsert(ctx, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *Service) StartTask(ctx context.Context, taskID uuid.UUID, visibleGroups []uuid.UUID) error {
	// #63 租户隔离：操作任务前先反查任务关联设备并校验归属，拒绝跨租户操作。
	if err := s.authorizeTaskDevices(ctx, taskID, visibleGroups); err != nil {
		return err
	}
	// For log-collect / config-backup tasks (Upload RPC), ResumeUpgrade can't
	// resolve the transport path because it isn't stored on the task row.
	// Look it up from the UFTE catalog and call ResumeCollect instead.
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task for start: %w", err)
	}
	if task.TaskType == software.TaskTypeLogCollect {
		catalog, err := s.loadTaskTypeCatalog(ctx)
		if err != nil {
			return err
		}
		// 用读路径统一分类入口（resolveTaskType 带 IMS ParamType 尾段剥离重试）；
		// 旧的 tt.FileType == task.DownloadFileType 精确比对在 IMS 任务上必然 miss。
		matchedDef, ok := s.resolveTaskTypeForTask(ctx, catalog, task.TaskType, task.ProductClass, task.DownloadFileType)
		var matched *TaskType
		if ok {
			matched = &matchedDef
		}
		// CONFIG_RESTORE / LICENSE_UPGRADE / IMS 下发类（参数/鉴权/备份）是"直接派发
		// device_tasks"链路，不能走 ResumeCollect（那是 Upload RPC 调度）。改走对应
		// dispatcher 再派发一遍，然后把占位 upgrade_tasks 推到 ended。
		if matched != nil && (matched.TypeCode == "CONFIG_RESTORE" ||
			matched.TypeCode == "LICENSE_UPGRADE" ||
			matched.TypeCode == "IMS_FILE_DISTRIBUTE") {
			return s.startDirectDispatchTask(ctx, task, matched)
		}
		transportPath, rpcType, paramPath := "", "", ""
		if matched != nil {
			transportPath = matched.TransportPath
			rpcType = matched.RPCType
			// FAULT_LOG_COLLECT 用 URLTemplate 存 SPV 参数路径（如
			// Device.DeviceInfo.FaultLogURL）。详见 model.go 注释。
			paramPath = matched.URLTemplate
			// IMS 采集（参数/日志）：子类型落库在 download_file_type 尾段，重派发时
			// 重新渲染进 TransportPath（executor 只认 {sn}/{taskId} 等占位符，
			// 不认 {paramType}；License/恢复无尾段不需要渲染）。
			if pt := imsParamTypeFromStoredFileType(task.DownloadFileType); pt != "" {
				transportPath = renderImsParamPlaceholders(transportPath, pt)
			}
		}
		return s.softwareService.ResumeCollect(ctx, taskID, transportPath, rpcType, paramPath)
	}
	return s.softwareService.ResumeUpgrade(ctx, taskID)
}

// startDirectDispatchTask 处理 CONFIG_RESTORE / LICENSE_UPGRADE 类型挂起任务的"开始"：
// 从 sub_tasks 读出设备 SN 列表 → 调对应 dispatcher 真实派发 → 把占位 upgrade_tasks
// 推到 ended/success + sub_tasks 推到 completed。
func (s *Service) startDirectDispatchTask(
	ctx context.Context, task *software.UpgradeTask, typeDef *TaskType,
) error {
	// H2：与 ResumeUpgrade/ResumeCollect 对齐，只有 suspended/pending 任务可「开始」。
	// 否则对已 ended/进行中的 CONFIG_RESTORE/LICENSE 任务再点开始会二次派发 device_tasks
	// （现网重复刷配置 / 重发 license），破坏幂等。复用同一业务码 8004。
	if task.Status != software.TaskSuspended && task.Status != software.TaskPending {
		return commonerrors.NewBusinessError(8004, "task is not suspended or pending", commonerrors.ErrInvalidInput)
	}
	// H3：翻页取全量子任务 SN。不翻页只拿首页（20 条）→ >20 台的任务只派发前 20 台，
	// 却用 len(sns) 把整任务标 100% 成功（静默部分派发）。
	subTasks, err := s.listAllSubTasksByTaskID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("list sub_tasks for direct dispatch start: %w", err)
	}
	sns := make([]string, 0, len(subTasks))
	for _, sub := range subTasks {
		if sub.DeviceSN == "" {
			continue
		}
		sns = append(sns, sub.DeviceSN)
	}
	if len(sns) == 0 {
		return fmt.Errorf("%w: suspended task has no device sub-tasks to dispatch", commonerrors.ErrInvalidInput)
	}

	switch typeDef.TypeCode {
	case "CONFIG_RESTORE":
		if s.snapshotConfigRestoreDispatcher == nil {
			return fmt.Errorf("%w: CONFIG_RESTORE dispatcher not wired", commonerrors.ErrInvalidInput)
		}
		_, dispatchedFiles, missing, dErr := s.snapshotConfigRestoreDispatcher.
			DispatchConfigRestoreBySnapshot(ctx, sns, task.CreateUser, task.ID)
		if dErr != nil {
			if len(missing) > 0 {
				return fmt.Errorf("%w: %d device(s) missing config snapshot: %v",
					commonerrors.ErrNotFound, len(missing), missing)
			}
			return fmt.Errorf("dispatch CONFIG_RESTORE on resume: %w", dErr)
		}
		s.persistDispatchedFiles(ctx, "CONFIG_RESTORE", task.ID, dispatchedFiles)
	case "LICENSE_UPGRADE":
		if s.licenseUpgradeDispatcher == nil {
			return fmt.Errorf("%w: LICENSE_UPGRADE dispatcher not wired", commonerrors.ErrInvalidInput)
		}
		_, dispatchedFiles, missing, dErr := s.licenseUpgradeDispatcher.
			DispatchLicenseUpgradeBySN(ctx, sns, task.CreateUser, task.ID)
		if dErr != nil {
			if len(missing) > 0 {
				return fmt.Errorf("%w: %d device(s) missing license file: %v",
					commonerrors.ErrNotFound, len(missing), missing)
			}
			return fmt.Errorf("dispatch LICENSE_UPGRADE on resume: %w", dErr)
		}
		s.persistDispatchedFiles(ctx, "LICENSE_UPGRADE", task.ID, dispatchedFiles)
	case "IMS_FILE_DISTRIBUTE":
		if s.imsParamDispatcher == nil {
			return fmt.Errorf("%w: %s dispatcher not wired", commonerrors.ErrInvalidInput, typeDef.TypeCode)
		}
		// 文件 ID 在占位任务行 file_name 列（UUID）；文件类型在
		// download_file_type 尾段（"IMS_FILE_DISTRIBUTE:FT_ImsCore_*"）。
		fileID, fidErr := uuid.Parse(strings.TrimSpace(task.FileName))
		if fidErr != nil || fileID == uuid.Nil {
			return fmt.Errorf("%w: suspended IMS task missing file id (file_name=%q)",
				commonerrors.ErrInvalidInput, task.FileName)
		}
		paramType := imsParamTypeFromStoredFileType(task.DownloadFileType)
		if _, ok := imsparam.Lookup(paramType); !ok {
			return fmt.Errorf("%w: suspended IMS task missing paramType (download_file_type=%q)",
				commonerrors.ErrInvalidInput, task.DownloadFileType)
		}
		_, dispatchedFiles, dErr := s.imsParamDispatcher.
			DispatchImsParamByFileID(ctx, sns, fileID, task.CreateUser, task.ID)
		if dErr != nil {
			return fmt.Errorf("dispatch %s on resume: %w", typeDef.TypeCode, dErr)
		}
		s.persistDispatchedFiles(ctx, typeDef.TypeCode, task.ID, dispatchedFiles)
	default:
		return fmt.Errorf("%w: unsupported direct-dispatch type %s",
			commonerrors.ErrInvalidInput, typeDef.TypeCode)
	}

	// 派发成功后把占位行推到 ended/success（progress 100%）
	if err := s.softwareService.FinalizePlaceholderTrackingTask(ctx, task.ID, len(sns)); err != nil {
		s.logger.Warn("finalize placeholder after direct-dispatch start failed",
			zap.String("task_id", task.ID.String()), zap.Error(err))
	}
	return nil
}

// ResumeLogCollectSubTask 实现 software.LogCollectResumer：当被挂起的 LogCollect 类
// （备份 / 日志采集）子任务因 device.online 事件被唤醒时，从 UFTE catalog 解析
// transport_path 后调用 SoftwareService.ExecuteOneUploadDirect 重启 Upload RPC。
// 详见 docs/project/backup-display-fix-20260520.md F8。
func (s *Service) ResumeLogCollectSubTask(ctx context.Context, subTask *software.UpgradeSubTask, parent *software.UpgradeTask) error {
	if parent == nil || subTask == nil {
		return fmt.Errorf("nil sub-task or parent in resume log collect")
	}
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return fmt.Errorf("load UFTE catalog: %w", err)
	}
	transportPath, rpcType, paramPath := "", "", ""
	// resolveTaskTypeForTask 带 IMS ParamType 尾段剥离重试；旧精确比对在
	// "ImsCore Parameters File:FT_ImsCore_*" 上 miss 会导致挂起采集任务唤醒后 URL 退化。
	matchedDef, matchedOK := s.resolveTaskTypeForTask(ctx, catalog, parent.TaskType, parent.ProductClass, parent.DownloadFileType)
	if matchedOK {
		transportPath = matchedDef.TransportPath
		rpcType = matchedDef.RPCType
		paramPath = matchedDef.URLTemplate
		// IMS 采集（参数/日志）：子类型落库在 download_file_type 尾段，恢复时重新
		// 渲染进模板（License/恢复无尾段不需要渲染）。
		if pt := imsParamTypeFromStoredFileType(parent.DownloadFileType); pt != "" {
			transportPath = renderImsParamPlaceholders(transportPath, pt)
		}
	}
	if transportPath == "" {
		s.logger.Warn("no transport_path matched for log collect resume; sub-task will fall back to default URL",
			zap.String("sub_task_id", subTask.ID.String()),
			zap.String("file_type", parent.DownloadFileType))
	}
	if strings.EqualFold(strings.TrimSpace(rpcType), "SET_PARAM_VALUES") {
		s.softwareService.ExecuteOneSetParamCollectDirect(ctx, subTask, paramPath, transportPath)
		return nil
	}
	// 传任务行落库值（IMS 含 ":FT_ImsCore_*" 尾段），executor.ExecuteOneUpload 内部 split。
	s.softwareService.ExecuteOneUploadDirect(ctx, subTask, parent.DownloadFileType, parent.FileName, transportPath)
	return nil
}

func (s *Service) SuspendTask(ctx context.Context, taskID uuid.UUID, visibleGroups []uuid.UUID) error {
	if err := s.authorizeTaskDevices(ctx, taskID, visibleGroups); err != nil {
		return err
	}
	return s.softwareService.SuspendUpgrade(ctx, taskID)
}

func (s *Service) TerminateTask(ctx context.Context, taskID uuid.UUID, visibleGroups []uuid.UUID) error {
	if err := s.authorizeTaskDevices(ctx, taskID, visibleGroups); err != nil {
		return err
	}
	return s.softwareService.TerminateUpgrade(ctx, taskID)
}

func (s *Service) DeleteTask(ctx context.Context, taskID uuid.UUID, visibleGroups []uuid.UUID) error {
	if err := s.authorizeTaskDevices(ctx, taskID, visibleGroups); err != nil {
		return err
	}
	return s.softwareService.DeleteUpgrade(ctx, taskID)
}

// BatchDeleteTaskResult 单条任务删除结果。
type BatchDeleteTaskResult struct {
	TaskID  uuid.UUID `json:"task_id"`
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
}

// BatchDeleteTasks 逐条调 DeleteTask；单条失败不影响其它（如运行中任务、已不存在等）。
// 返回每条详细结果，前端据此提示"成功 X 个，失败 Y 个：xxx"。
// #63 租户隔离：逐条先校验任务关联设备归属，越权的那条标记失败（不阻断其它条），
// 与"单条失败不影响整体"的既有语义一致。
func (s *Service) BatchDeleteTasks(ctx context.Context, taskIDs []uuid.UUID, visibleGroups []uuid.UUID) []BatchDeleteTaskResult {
	results := make([]BatchDeleteTaskResult, 0, len(taskIDs))
	for _, id := range taskIDs {
		r := BatchDeleteTaskResult{TaskID: id, Success: true}
		if err := s.authorizeTaskDevices(ctx, id, visibleGroups); err != nil {
			r.Success = false
			r.Error = err.Error()
			results = append(results, r)
			continue
		}
		if err := s.softwareService.DeleteUpgrade(ctx, id); err != nil {
			r.Success = false
			r.Error = err.Error()
		}
		results = append(results, r)
	}
	return results
}

func (s *Service) RetryTask(ctx context.Context, taskID uuid.UUID, visibleGroups []uuid.UUID) error {
	if err := s.authorizeTaskDevices(ctx, taskID, visibleGroups); err != nil {
		return err
	}
	return s.softwareService.RetryUpgrade(ctx, taskID)
}

func (s *Service) DeleteTaskType(ctx context.Context, typeCode string) error {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return err
	}
	existing, ok := findTaskTypeByCode(catalog, typeCode)
	if !ok {
		return commonerrors.ErrNotFound
	}
	if existing.BuiltIn {
		return fmt.Errorf("%w: built-in UFTE task type cannot be deleted", commonerrors.ErrForbidden)
	}
	if err := s.taskTypeRepo.Delete(ctx, typeCode); err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateTask(ctx context.Context, req CreateTaskRequest, createUser string, visibleGroups []uuid.UUID) (*Task, error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	typeDef, ok := findTaskTypeByCode(catalog, req.TypeCode)
	if !ok {
		return nil, fmt.Errorf("%w: unsupported UFTE type %s", commonerrors.ErrInvalidInput, req.TypeCode)
	}
	if err := s.rejectUPSFeatureIfUnlicensed(ctx, typeDef); err != nil {
		return nil, err
	}
	if len(req.DeviceIDs) == 0 {
		return nil, fmt.Errorf("%w: device_ids is required", commonerrors.ErrInvalidInput)
	}
	// #63 租户隔离：派发任何文件传输任务前，整批校验目标设备归属当前调用者可见
	// 设备组；只要混入域外设备即整批拒绝（ErrForbidden → handler 映射 403）。
	if err := s.authorizeDevices(ctx, visibleGroups, req.DeviceIDs...); err != nil {
		return nil, err
	}

	// 解析 scheduled 模式的目标时间：仅在 ExecutionMode="scheduled" 且字符串非空时解析；
	// 解析失败 → 视作 immediate（前端发请求前会校验时间，后端这里是兜底）。
	var scheduledAt *time.Time
	if req.ExecutionMode == "scheduled" && req.ScheduledAt != "" {
		if t, parseErr := time.Parse(time.RFC3339, req.ScheduledAt); parseErr == nil {
			scheduledAt = &t
		} else {
			s.logger.Warn("invalid scheduledAt, falling back to immediate",
				zap.String("scheduled_at_raw", req.ScheduledAt), zap.Error(parseErr))
		}
	}

	var createdTask *software.UpgradeTask
	createSuspended := req.ExecutionMode == "suspended"

	// T-0164：CONFIG_RESTORE 走独立链路（backup.RestoreService.CreateBySnapshot），
	// 不进 software.upgrade_tasks 表，因此 createdTask 留 nil，由 dispatcher 返回
	// dispatchedID 后构造 placeholder。提前 return，避免落到下面的 firmware case。
	if typeDef.TypeCode == "CONFIG_RESTORE" {
		return s.createConfigRestoreTask(ctx, &typeDef, req, createUser, createSuspended, scheduledAt)
	}

	// T-0165：LICENSE_UPGRADE 与 CONFIG_RESTORE 同款，走 backup.LicenseService 派发。
	if typeDef.TypeCode == "LICENSE_UPGRADE" {
		return s.createLicenseUpgradeTask(ctx, &typeDef, req, createUser, createSuspended, scheduledAt)
	}

	// IMS 文件下发（OMC → 设备 Download）：占位 + 直接派发 device_tasks。
	if typeDef.TypeCode == "IMS_FILE_DISTRIBUTE" {
		return s.createImsParamDistributeTask(ctx, &typeDef, req, createUser, createSuspended, scheduledAt)
	}

	// IMS 文件采集（设备 → OMC Upload）。文件类型（FT_ImsCore_*）预渲染进
	// TransportPath（{paramType} 占位符），StoredFileType 让任务行落
	// "Ims File:FT_ImsCore_*"（反查键 + 文件类型）；CWMP 下发的 FileType 仍是
	// catalog 统一字面值 "Ims File"（executor 把子类型放进 <ParameterType> 标签）。
	if typeDef.TypeCode == "IMS_FILE_COLLECT" {
		paramDef, ok := imsparam.Lookup(req.ParamType)
		if !ok {
			return nil, fmt.Errorf("%w: missing/invalid paramType %q for IMS_FILE_COLLECT (expect FT_ImsCore_*)",
				commonerrors.ErrInvalidInput, req.ParamType)
		}
		if !paramDef.UploadSupported {
			return nil, fmt.Errorf("%w: file type %s does not support upload", commonerrors.ErrInvalidInput, paramDef.Code)
		}
		createdTask, err = s.softwareService.BatchCollect(ctx, software.BatchCollectRequest{
			DeviceIDs:              req.DeviceIDs,
			TaskName:               req.TaskName,
			FileType:               typeDef.FileType,
			StoredFileType:         typeDef.FileType + ":" + paramDef.Code,
			TargetFileNameTemplate: renderImsParamPlaceholders(typeDef.TargetFileNameTemplate, paramDef.Code),
			TransportPath:          renderImsParamPlaceholders(typeDef.TransportPath, paramDef.Code),
			RPCType:                typeDef.RPCType,
			ParamPath:              typeDef.URLTemplate,
			CreateUser:             createUser,
			CreateSuspended:        createSuspended,
			ScheduledAt:            scheduledAt,
		})
		if err != nil {
			return nil, err
		}
		return s.mapTask(ctx, catalog, createdTask)
	}

	switch {
	case typeDef.softwareTaskType == software.TaskTypeLogCollect:
		// 日志采集（Upload 或 SetParameterValues RPC）：不需要固件，直接通过 BatchCollect 下发。
		// FAULT_LOG_COLLECT 用 SPV 模式 — URLTemplate 字段复用为 SPV 参数路径
		// （Device.DeviceInfo.X_COM_Log.FaultLogURL）；其余走传统 Upload RPC。
		createdTask, err = s.softwareService.BatchCollect(ctx, software.BatchCollectRequest{
			DeviceIDs:              req.DeviceIDs,
			TaskName:               req.TaskName,
			FileType:               typeDef.FileType,
			TargetFileNameTemplate: typeDef.TargetFileNameTemplate,
			TransportPath:          typeDef.TransportPath,
			RPCType:                typeDef.RPCType,
			ParamPath:              typeDef.URLTemplate,
			CreateUser:             createUser,
			CreateSuspended:        createSuspended,
			ScheduledAt:            scheduledAt,
		})
	case typeDef.TypeCode == "VERSION_ROLLBACK":
		createdTask, err = s.softwareService.RollbackDevices(ctx, software.RollbackRequest{
			DeviceIDs:       req.DeviceIDs,
			TaskName:        req.TaskName,
			CreateUser:      createUser,
			CreateSuspended: createSuspended,
			ScheduledAt:     scheduledAt,
			Source:          software.RollbackSourceManual,
			Reason:          req.Note,
		})
	case typeDef.softwareTaskType != 0:
		// 固件下载类任务（升级、PATCH、FPGA 等）
		if req.FirmwareID == nil || *req.FirmwareID == uuid.Nil {
			return nil, fmt.Errorf("%w: firmware_id is required for upgrade tasks", commonerrors.ErrInvalidInput)
		}
		createdTask, err = s.softwareService.BatchUpgrade(ctx, software.BatchUpgradeRequest{
			DeviceIDs:        req.DeviceIDs,
			FirmwareID:       *req.FirmwareID,
			TaskName:         req.TaskName,
			TaskType:         typeDef.softwareTaskType,
			DownloadFileType: typeDef.FileType,
			ProductClassHint: req.ProductType,
			IsKeepConfig:     req.IsKeepConfig,
			Concurrency:      req.Concurrency,
			CreateUser:       createUser,
			CreateSuspended:  createSuspended,
			ScheduledAt:      scheduledAt,
		})
	default:
		return nil, fmt.Errorf("%w: UFTE type %s is not executable by the current backend adapter", commonerrors.ErrInvalidInput, req.TypeCode)
	}
	if err != nil {
		return nil, err
	}
	return s.mapTask(ctx, catalog, createdTask)
}

// createConfigRestoreTask (T-0164) 处理 CONFIG_RESTORE 任务创建。
//
// 与升级/采集/回滚不同，本类型**不**写入 software.upgrade_tasks 表，而是走
// backup.RestoreService.CreateBySnapshot：每台设备从 config_snapshots 取自己
// 最新一份配置 → 缺失整批拒绝 → 全部存在则创建 restore_tasks 主行 +
// 逐设备 Download device_tasks。
//
// 返回的 ufte.Task 是 placeholder（ID = restore_task UUID，Status/Progress 暂置
// 初值），UFTE "任务列表"不会展示它（列表查 upgrade_tasks）；CONFIG_RESTORE 历史
// 任务的展示由备份恢复模块的 RestoreTask 列表负责。
func (s *Service) createConfigRestoreTask(
	ctx context.Context,
	typeDef *TaskType,
	req CreateTaskRequest,
	createUser string,
	createSuspended bool,
	scheduledAt *time.Time,
) (*Task, error) {
	if s.snapshotConfigRestoreDispatcher == nil {
		return nil, fmt.Errorf("%w: CONFIG_RESTORE dispatcher not wired", commonerrors.ErrInvalidInput)
	}

	sns := make([]string, 0, len(req.DeviceIDs))
	for _, did := range req.DeviceIDs {
		dev, err := s.deviceRepo.GetByID(ctx, did)
		if err != nil || dev == nil {
			return nil, fmt.Errorf("%w: device %s not found", commonerrors.ErrInvalidInput, did)
		}
		sns = append(sns, dev.SerialNumber)
	}

	// 1) 先建占位 upgrade_tasks + sub_tasks（拿 task.ID 用于派生 CommandKey）。
	//    scheduledAt 非空 → 占位 status=pending + create_status=timing，等 scheduler 触发。
	placeholder, phErr := s.softwareService.CreatePlaceholderTrackingTask(ctx,
		software.PlaceholderTrackingRequest{
			DeviceIDs:   req.DeviceIDs,
			TaskName:    req.TaskName,
			TypeCode:    typeDef.TypeCode, // CommandKey 前缀 = "CONFIG_RESTORE"
			FileType:    typeDef.FileType, // download_file_type 列（catalog 字面，含 <OUI>）
			CreateUser:  createUser,
			Suspended:   createSuspended,
			ScheduledAt: scheduledAt,
		})
	if phErr != nil {
		return nil, fmt.Errorf("create CONFIG_RESTORE placeholder: %w", phErr)
	}

	// 1.5) 不管哪种执行模式都先 Preview 一遍每台设备的快照文件名，写到 sub_task.dest_version，
	// 让前端"目标文件"列在派发前（suspended / scheduled）也立即可见。
	// Preview 失败不阻断主流程（dest_version 留空，列表显示 "-"，用户至少能看到任务本身）。
	if previewFiles, prevErr := s.snapshotConfigRestoreDispatcher.PreviewConfigRestoreFiles(ctx, sns); prevErr == nil {
		s.persistDispatchedFiles(ctx, "CONFIG_RESTORE", placeholder.ID, previewFiles)
	} else {
		s.logger.Warn("preview CONFIG_RESTORE files failed; dest_version stays empty",
			zap.String("task_id", placeholder.ID.String()), zap.Error(prevErr))
	}

	// 2) Suspended / scheduled：到此结束；等用户点"开始"或 scheduler 到点触发 dispatcher。
	if createSuspended || (scheduledAt != nil && scheduledAt.After(time.Now())) {
		catalog, _ := s.loadTaskTypeCatalog(ctx)
		return s.mapTask(ctx, catalog, placeholder)
	}

	// 3) 派发 device_tasks（CommandKey 与 sub_task 一致 → TC 自动推进）。
	_, dispatchedFiles, missing, err := s.snapshotConfigRestoreDispatcher.
		DispatchConfigRestoreBySnapshot(ctx, sns, createUser, placeholder.ID)
	if err != nil {
		if len(missing) > 0 {
			return nil, fmt.Errorf("%w: %d device(s) missing config snapshot: %v",
				commonerrors.ErrNotFound, len(missing), missing)
		}
		return nil, fmt.Errorf("dispatch CONFIG_RESTORE by snapshot: %w", err)
	}
	s.persistDispatchedFiles(ctx, "CONFIG_RESTORE", placeholder.ID, dispatchedFiles)
	placeholder = s.finalizeDirectDispatchPlaceholder(ctx, placeholder, len(sns), "CONFIG_RESTORE")

	catalog, _ := s.loadTaskTypeCatalog(ctx)
	return s.mapTask(ctx, catalog, placeholder)
}

// createLicenseUpgradeTask (T-0165) 处理 LICENSE_UPGRADE 任务创建。
// 与 createConfigRestoreTask 同款：不写 software.upgrade_tasks，直接派发
// device_tasks。返回 placeholder Task 给前端展示。
func (s *Service) createLicenseUpgradeTask(
	ctx context.Context,
	typeDef *TaskType,
	req CreateTaskRequest,
	createUser string,
	createSuspended bool,
	scheduledAt *time.Time,
) (*Task, error) {
	if s.licenseUpgradeDispatcher == nil {
		return nil, fmt.Errorf("%w: LICENSE_UPGRADE dispatcher not wired", commonerrors.ErrInvalidInput)
	}

	sns := make([]string, 0, len(req.DeviceIDs))
	for _, did := range req.DeviceIDs {
		dev, err := s.deviceRepo.GetByID(ctx, did)
		if err != nil || dev == nil {
			return nil, fmt.Errorf("%w: device %s not found", commonerrors.ErrInvalidInput, did)
		}
		sns = append(sns, dev.SerialNumber)
	}

	// 1) 先建占位 upgrade_tasks + sub_tasks（拿到 task.ID 用于派生 CommandKey）。
	//    Suspended / scheduled 模式 sub_tasks=pending；immediate 模式 sub_tasks=downloading。
	placeholder, phErr := s.softwareService.CreatePlaceholderTrackingTask(ctx,
		software.PlaceholderTrackingRequest{
			DeviceIDs:   req.DeviceIDs,
			TaskName:    req.TaskName,
			TypeCode:    typeDef.TypeCode, // CommandKey 前缀（如 "LICENSE_UPGRADE"），与 dispatcher 一致
			FileType:    typeDef.FileType, // download_file_type 列（catalog 字面，含 <OUI> 占位）
			CreateUser:  createUser,
			Suspended:   createSuspended,
			ScheduledAt: scheduledAt,
		})
	if phErr != nil {
		return nil, fmt.Errorf("create LICENSE_UPGRADE placeholder: %w", phErr)
	}

	// 1.5) 同 CONFIG_RESTORE：先 Preview 每台设备的 license 文件名写到 sub_task.dest_version。
	if previewFiles, prevErr := s.licenseUpgradeDispatcher.PreviewLicenseFiles(ctx, sns); prevErr == nil {
		s.persistDispatchedFiles(ctx, "LICENSE_UPGRADE", placeholder.ID, previewFiles)
	} else {
		s.logger.Warn("preview LICENSE_UPGRADE files failed; dest_version stays empty",
			zap.String("task_id", placeholder.ID.String()), zap.Error(prevErr))
	}

	// 2) Suspended / scheduled：到此结束；等用户点"开始"或 scheduler 到点触发 dispatcher。
	if createSuspended || (scheduledAt != nil && scheduledAt.After(time.Now())) {
		catalog, _ := s.loadTaskTypeCatalog(ctx)
		return s.mapTask(ctx, catalog, placeholder)
	}

	// 3) 派发 device_tasks（CommandKey 与 sub_task 一致 → TC 自动推进）。
	_, dispatchedFiles, missing, err := s.licenseUpgradeDispatcher.
		DispatchLicenseUpgradeBySN(ctx, sns, createUser, placeholder.ID)
	if err != nil {
		if len(missing) > 0 {
			return nil, fmt.Errorf("%w: %d device(s) missing license file: %v",
				commonerrors.ErrNotFound, len(missing), missing)
		}
		return nil, fmt.Errorf("dispatch LICENSE_UPGRADE: %w", err)
	}
	s.persistDispatchedFiles(ctx, "LICENSE_UPGRADE", placeholder.ID, dispatchedFiles)
	placeholder = s.finalizeDirectDispatchPlaceholder(ctx, placeholder, len(sns), "LICENSE_UPGRADE")

	catalog, _ := s.loadTaskTypeCatalog(ctx)
	return s.mapTask(ctx, catalog, placeholder)
}

// createImsParamDistributeTask 处理 IMS_FILE_DISTRIBUTE（核心网文件下发）任务创建。
// 与 createLicenseUpgradeTask 同款：占位 upgrade_tasks + 直接派发 device_tasks，
// TC 经 CommandKey 回推占位 sub_tasks。差异点：
//   - 文件源是 ims_param_files（运营者在参数文件库上传），任务级单选一份文件；
//   - download_file_type 落 "IMS_FILE_DISTRIBUTE:FT_ImsCore_*"（文件类型尾段），
//     file_name 落文件 UUID —— 挂起/定时任务 Start 时按两者重派发同一份文件。
//
// createImsParamDistributeTask 处理 IMS_FILE_DISTRIBUTE（核心网文件下发）任务创建。
// 与 createLicenseUpgradeTask 同款：占位 upgrade_tasks + 直接派发 device_tasks，
// TC 经 CommandKey 回推占位 sub_tasks。差异点：
//   - 文件源是 ims_param_files（运营者在文件库上传），任务级单选一份文件；
//   - 需选文件类型（FT_ImsCore_*，Download 段），落库值带 ":FT_ImsCore_*" 尾段；
//   - file_name 落文件 UUID —— 挂起/定时任务 Start 时按其重派发同一份文件。
func (s *Service) createImsParamDistributeTask(
	ctx context.Context,
	typeDef *TaskType,
	req CreateTaskRequest,
	createUser string,
	createSuspended bool,
	scheduledAt *time.Time,
) (*Task, error) {
	if s.imsParamDispatcher == nil {
		return nil, fmt.Errorf("%w: %s dispatcher not wired", commonerrors.ErrInvalidInput, typeDef.TypeCode)
	}
	// 落库 FileType：参数类带子类型尾段（重派发时还原 <ParameterType>），
	// 鉴权/备份单一类型直接用 catalog 反查键。
	paramDef, ok := imsparam.Lookup(req.ParamType)
	if !ok || !paramDef.DownloadSupported {
		return nil, fmt.Errorf("%w: missing/invalid paramType %q for IMS_FILE_DISTRIBUTE (expect download FT_ImsCore_* types)",
			commonerrors.ErrInvalidInput, req.ParamType)
	}
	storedFileType := typeDef.FileType + ":" + paramDef.Code
	if req.FileID == nil || *req.FileID == uuid.Nil {
		return nil, fmt.Errorf("%w: fileId is required for %s", commonerrors.ErrInvalidInput, typeDef.TypeCode)
	}
	previewName, previewParamType, err := s.imsParamDispatcher.PreviewImsParamFile(ctx, *req.FileID)
	if err != nil {
		return nil, fmt.Errorf("preview %s file: %w", typeDef.TypeCode, err)
	}
	paramDef, ok = imsparam.Lookup(previewParamType)
	if !ok || paramDef.Code != imsparam.NormalizeParamType(req.ParamType) {
		return nil, fmt.Errorf("%w: file %s (type %s) does not match selected param type %s",
			commonerrors.ErrInvalidInput, previewName, previewParamType, req.ParamType)
	}

	sns := make([]string, 0, len(req.DeviceIDs))
	for _, did := range req.DeviceIDs {
		dev, err := s.deviceRepo.GetByID(ctx, did)
		if err != nil || dev == nil {
			return nil, fmt.Errorf("%w: device %s not found", commonerrors.ErrInvalidInput, did)
		}
		sns = append(sns, dev.SerialNumber)
	}

	// 1) 先建占位 upgrade_tasks + sub_tasks（拿 task.ID 用于派生 CommandKey）。
	placeholder, phErr := s.softwareService.CreatePlaceholderTrackingTask(ctx,
		software.PlaceholderTrackingRequest{
			DeviceIDs:   req.DeviceIDs,
			TaskName:    req.TaskName,
			TypeCode:    typeDef.TypeCode,    // CommandKey 前缀 = 模板 TypeCode
			FileType:    storedFileType,      // 反查键（参数类含子类型尾段）
			FileName:    req.FileID.String(), // 重派发用文件 UUID
			CreateUser:  createUser,
			Suspended:   createSuspended,
			ScheduledAt: scheduledAt,
		})
	if phErr != nil {
		return nil, fmt.Errorf("create %s placeholder: %w", typeDef.TypeCode, phErr)
	}

	// 1.5) 预览目标文件名（文件存在性和类型已在创建占位任务前校验），写到
	// sub_task.dest_version 让挂起 / 定时任务在派发前也能看到"目标文件"。
	previewFiles := make(map[string]string, len(sns))
	for _, sn := range sns {
		previewFiles[sn] = previewName
	}
	s.persistDispatchedFiles(ctx, typeDef.TypeCode, placeholder.ID, previewFiles)

	// 2) Suspended / scheduled：到此结束；等用户点"开始"或 scheduler 到点触发。
	if createSuspended || (scheduledAt != nil && scheduledAt.After(time.Now())) {
		catalog, _ := s.loadTaskTypeCatalog(ctx)
		return s.mapTask(ctx, catalog, placeholder)
	}

	// 3) 派发 device_tasks（CommandKey 与 sub_task 一致 → TC 自动推进）。
	_, dispatchedFiles, err := s.imsParamDispatcher.
		DispatchImsParamByFileID(ctx, sns, *req.FileID, createUser, placeholder.ID)
	if err != nil {
		return nil, fmt.Errorf("dispatch %s: %w", typeDef.TypeCode, err)
	}
	s.persistDispatchedFiles(ctx, typeDef.TypeCode, placeholder.ID, dispatchedFiles)
	placeholder = s.finalizeDirectDispatchPlaceholder(ctx, placeholder, len(sns), typeDef.TypeCode)

	catalog, _ := s.loadTaskTypeCatalog(ctx)
	return s.mapTask(ctx, catalog, placeholder)
}

func (s *Service) finalizeDirectDispatchPlaceholder(
	ctx context.Context,
	placeholder *software.UpgradeTask,
	deviceCount int,
	typeCode string,
) *software.UpgradeTask {
	if placeholder == nil || s.softwareService == nil {
		return placeholder
	}
	if err := s.softwareService.FinalizePlaceholderTrackingTask(ctx, placeholder.ID, deviceCount); err != nil {
		s.logger.Warn("finalize direct-dispatch placeholder failed",
			zap.String("type_code", typeCode),
			zap.String("task_id", placeholder.ID.String()),
			zap.Error(err))
		return placeholder
	}

	now := coremodel.Time(time.Now())
	placeholder.Status = software.TaskInProgress
	placeholder.StartedAt = &now
	if s.taskRepo == nil {
		return placeholder
	}
	refreshed, err := s.taskRepo.GetByID(ctx, placeholder.ID)
	if err != nil {
		s.logger.Warn("reload direct-dispatch placeholder failed",
			zap.String("type_code", typeCode),
			zap.String("task_id", placeholder.ID.String()),
			zap.Error(err))
		return placeholder
	}
	if refreshed == nil {
		return placeholder
	}
	return refreshed
}

// persistDispatchedFiles 把 dispatcher 返回的 sn → 文件名映射写回 upgrade_sub_tasks.dest_version。
// CommandKey 用 software.BuildDirectDispatchCommandKey 重建，与 placeholder 创建时
// sub_task.CommandKey、dispatcher enqueue device_task.CommandKey 一致。
//
// best-effort：任一 sn 写失败仅 warn，不阻断整体派发——派发已成功提交，dest_version
// 仅影响"目标文件"列展示。下次 list 时这一行 dest_version 留空 = "-"，可手动重派后修复。
func (s *Service) persistDispatchedFiles(ctx context.Context, typeCode string, upgradeTaskID uuid.UUID, files map[string]string) {
	if len(files) == 0 || s.subTaskRepo == nil {
		return
	}
	for sn, fileName := range files {
		if sn == "" || fileName == "" {
			continue
		}
		commandKey := software.BuildDirectDispatchCommandKey(typeCode, upgradeTaskID, sn)
		if err := s.subTaskRepo.UpdateDestVersionByCommandKey(ctx, commandKey, fileName); err != nil {
			s.logger.Warn("persist dispatched file name failed (sub_task.dest_version)",
				zap.String("type_code", typeCode),
				zap.String("upgrade_task_id", upgradeTaskID.String()),
				zap.String("device_sn", sn),
				zap.String("file_name", fileName),
				zap.Error(err))
		}
	}
}

func (s *Service) ListTasks(ctx context.Context, filter TaskListFilter, visibleGroups []uuid.UUID) (*coremodel.ListResponse[Task], error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	tasks, err := s.loadAllTasks(ctx, catalog, filter)
	if err != nil {
		return nil, err
	}
	// #63 租户隔离：非超管时把任务收窄到「至少含一台可见设备」的任务。超管
	// (visibleGroups==nil) 跳过——visibleTaskIDs 为 nil，下面的成员判定直接放行，
	// 不额外查 sub_tasks，保持现网行为与开销不变。
	var visibleTaskIDs map[uuid.UUID]struct{}
	if visibleGroups != nil && s.groupReader != nil {
		visibleTaskIDs, err = s.visibleTaskIDSet(ctx, catalog, filter, visibleGroups)
		if err != nil {
			return nil, err
		}
	}
	upsLicensed := s.upsFeatureLicensed(ctx)
	items := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if visibleTaskIDs != nil {
			if _, ok := visibleTaskIDs[task.ID]; !ok {
				continue
			}
		}
		taskCopy := task
		mapped, err := s.mapTask(ctx, catalog, &taskCopy)
		if err != nil {
			s.logger.Debug("skip UFTE task", zap.String("task_id", task.ID.String()), zap.Error(err))
			continue
		}
		if !upsLicensed && isUPSTask(*mapped) {
			continue
		}
		if !matchesTaskFilter(*mapped, filter) {
			continue
		}
		items = append(items, *mapped)
	}
	sort.Slice(items, func(i, j int) bool {
		return timePtrAfter(items[i].CreatedAt, items[j].CreatedAt)
	})
	return paginate(items, filter.Page, filter.PageSize), nil
}

// GetTask returns a single UFTE/file-transfer task by ID. It is intentionally a
// service-level helper so other modules can reuse the same catalog mapping and
// visibility semantics as ListTasks without going through HTTP.
func (s *Service) GetTask(ctx context.Context, rawID string, visibleGroups []uuid.UUID) (*Task, error) {
	id, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid task id", commonerrors.ErrInvalidInput)
	}
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	if visibleGroups != nil && s.groupReader != nil {
		visibleTaskIDs, err := s.visibleTaskIDSet(ctx, catalog, TaskListFilter{}, visibleGroups)
		if err != nil {
			return nil, err
		}
		if _, ok := visibleTaskIDs[id]; !ok {
			return nil, commonerrors.ErrForbidden
		}
	}
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, commonerrors.ErrNotFound
	}
	mapped, err := s.mapTask(ctx, catalog, task)
	if err != nil {
		return nil, err
	}
	if !s.upsFeatureLicensed(ctx) && isUPSTask(*mapped) {
		return nil, commonerrors.ErrForbidden
	}
	return mapped, nil
}

// visibleTaskIDSet 用一次批量 sub_tasks 加载，算出「至少含一台可见设备」的任务 ID 集合。
// 仅在非超管（visibleGroups != nil）时调用。语义为「任务触及调用者任一可见设备即可见」，
// 与写路径「整批含域外设备即拒绝」是读/写两侧各自合理的取舍：读侧不因混入域外设备而
// 把本租户也有份的任务整体藏掉。注意设备级明细仍由 ListDevices/Export 单独按设备过滤。
func (s *Service) visibleTaskIDSet(
	ctx context.Context, catalog []TaskType, filter TaskListFilter, visibleGroups []uuid.UUID,
) (map[uuid.UUID]struct{}, error) {
	subTasks, err := s.loadAllSubTasks(ctx, catalog, filter.Category, filter.TypeCode, "")
	if err != nil {
		return nil, err
	}
	vf := s.newDeviceVisibility(visibleGroups)
	set := make(map[uuid.UUID]struct{})
	for _, st := range subTasks {
		if _, ok := set[st.TaskID]; ok {
			continue
		}
		if vf.allow(ctx, st.DeviceID) {
			set[st.TaskID] = struct{}{}
		}
	}
	return set, nil
}

func (s *Service) ListDevices(ctx context.Context, filter DeviceListFilter, visibleGroups []uuid.UUID) (*coremodel.ListResponse[DeviceItem], error) {
	items, err := s.collectFilteredDeviceItems(ctx, filter, visibleGroups)
	if err != nil {
		return nil, err
	}
	return paginate(items, filter.Page, filter.PageSize), nil
}

// StreamDeviceItems 流式过滤 + 逐条输出设备子任务，每条调一次 write 回调。
// 给导出 CSV 用 —— 全程不把全量结果累积在内存：
//  1. 按 taskType / page 分批拉 sub_tasks（每批 200 条）
//  2. 当批 mapDeviceItem + matchesDeviceFilter → 命中即立刻 write 回调出去
//  3. 调用方在 write 里写 CSV + 周期性 Flush，bytes 直推 HTTP 流
//
// 内存峰值约 = 1 批 sub_tasks + deviceCache + parentCache + catalog（cache 体量
// 随设备 / 任务总数线性增长但单项 ~百字节，相对全量 DeviceItem 累积小得多）。
//
// 与 ListDevices 区别：**不做全局排序**——排序需要全量在内存。导出可在 Excel
// 里自行排序，不影响数据完整性。
func (s *Service) StreamDeviceItems(
	ctx context.Context, filter DeviceListFilter, visibleGroups []uuid.UUID, write func(DeviceItem) error,
) error {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return err
	}
	typeSet, err := deviceTypeFilterKeys(catalog, filter.Category, filter.TypeCode)
	if err != nil {
		return fmt.Errorf("%w: %v", commonerrors.ErrInvalidInput, err)
	}
	if len(typeSet) == 0 {
		return nil
	}
	taskIDPtr, err := parseTaskIDFilter(filter.TaskID)
	if err != nil {
		return err
	}
	vis := s.newDeviceVisibility(visibleGroups) // #63 租户隔离
	upsLicensed := s.upsFeatureLicensed(ctx)
	deviceCache := make(map[uuid.UUID]*coremodel.Device)
	parentCache := make(map[uuid.UUID]*software.UpgradeTask)
	const batchSize = 200

	for taskType := range typeSet {
		page := 1
		for {
			pageResult, err := s.subTaskRepo.ListAll(ctx, software.AllSubTaskFilter{
				TaskType: taskTypePtr(taskType),
				TaskID:   taskIDPtr, // #615 SQL 下推：空 = 不过滤；非空 = WHERE ust.task_id = ?
				ListRequest: coremodel.ListRequest{
					Page: page, PageSize: batchSize,
				},
			})
			if err != nil {
				return err
			}
			for _, subTask := range pageResult.Items {
				if !vis.allow(ctx, subTask.DeviceID) { // #63 跳过越权设备
					continue
				}
				parent, ok := parentCache[subTask.TaskID]
				if !ok {
					parent, err = s.taskRepo.GetByID(ctx, subTask.TaskID)
					if err != nil {
						s.logger.Debug("skip UFTE sub-task parent lookup (stream)",
							zap.String("task_id", subTask.TaskID.String()), zap.Error(err))
						continue
					}
					parentCache[subTask.TaskID] = parent
				}
				mapped, mErr := s.mapDeviceItem(ctx, catalog, subTask, parent, deviceCache)
				if mErr != nil {
					s.logger.Debug("skip UFTE sub-task mapping (stream)",
						zap.String("sub_task_id", subTask.ID.String()), zap.Error(mErr))
					continue
				}
				if !upsLicensed && isUPSDeviceItem(*mapped) {
					continue
				}
				if !matchesDeviceFilter(*mapped, filter) {
					continue
				}
				if wErr := write(*mapped); wErr != nil {
					return wErr
				}
			}
			if page >= pageResult.TotalPages || len(pageResult.Items) == 0 {
				break
			}
			page++
		}
	}
	return nil
}

// collectFilteredDeviceItems 把 ListDevices 的核心抓取/映射/过滤/排序逻辑提出来复用。
func (s *Service) collectFilteredDeviceItems(ctx context.Context, filter DeviceListFilter, visibleGroups []uuid.UUID) ([]DeviceItem, error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	subTasks, err := s.loadAllSubTasks(ctx, catalog, filter.Category, filter.TypeCode, filter.TaskID)
	if err != nil {
		return nil, err
	}
	vis := s.newDeviceVisibility(visibleGroups) // #63 租户隔离
	upsLicensed := s.upsFeatureLicensed(ctx)
	deviceCache := make(map[uuid.UUID]*coremodel.Device)
	parentCache := make(map[uuid.UUID]*software.UpgradeTask)
	items := make([]DeviceItem, 0, len(subTasks))
	for _, subTask := range subTasks {
		if !vis.allow(ctx, subTask.DeviceID) { // #63 跳过越权设备
			continue
		}
		parent, ok := parentCache[subTask.TaskID]
		if !ok {
			parent, err = s.taskRepo.GetByID(ctx, subTask.TaskID)
			if err != nil {
				s.logger.Debug("skip UFTE sub-task parent lookup", zap.String("task_id", subTask.TaskID.String()), zap.Error(err))
				continue
			}
			parentCache[subTask.TaskID] = parent
		}
		mapped, err := s.mapDeviceItem(ctx, catalog, subTask, parent, deviceCache)
		if err != nil {
			s.logger.Debug("skip UFTE sub-task mapping", zap.String("sub_task_id", subTask.ID.String()), zap.Error(err))
			continue
		}
		if !upsLicensed && isUPSDeviceItem(*mapped) {
			continue
		}
		if !matchesDeviceFilter(*mapped, filter) {
			continue
		}
		items = append(items, *mapped)
	}
	// 按子任务创建时间倒序——新建任务的设备排在最前面。
	// 不能按 LastReportAt 排：它只在文件上报成功的终态才有值（见 mapDeviceItem / issue #195），
	// 新建、未上报的设备为空串，倒序会把它们挤到列表最后（本次修复的 bug）。
	// CreatedAt（子任务 created_at）建任务即有值且 NOT NULL，倒序即"最新建的在最上"。
	sort.Slice(items, func(i, j int) bool {
		return timePtrAfter(items[i].CreatedAt, items[j].CreatedAt)
	})
	return items, nil
}

func (s *Service) ListDeviceCandidates(ctx context.Context, filter DeviceCandidateFilter, visibleGroups []uuid.UUID) (*coremodel.ListResponse[DeviceItem], error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	var typeDef *TaskType
	if filter.TypeCode != "" {
		item, ok := findTaskTypeByCode(catalog, filter.TypeCode)
		if !ok {
			return nil, fmt.Errorf("%w: unsupported type_code: %s", commonerrors.ErrInvalidInput, filter.TypeCode)
		}
		if err := s.rejectUPSFeatureIfUnlicensed(ctx, item); err != nil {
			return nil, err
		}
		typeDef = &item
		if filter.Category == "" {
			filter.Category = item.Category
		}
	}
	upsLicensed := s.upsFeatureLicensed(ctx)
	if !upsLicensed && filter.Category == "ups_upgrade" {
		return &coremodel.ListResponse[DeviceItem]{
			Items:      []DeviceItem{},
			Total:      0,
			Page:       filter.Page,
			PageSize:   filter.PageSize,
			TotalPages: 1,
		}, nil
	}
	deviceFilter := device.DeviceFilter{
		ListRequest: coremodel.ListRequest{
			Page:     filter.Page,
			PageSize: filter.PageSize,
		},
		// #63 租户隔离：候选设备直接在 device 仓储层按可见分组三态过滤
		// （nil 超管不过滤 / [] fail-closed / [g..] 限定）。
		VisibleGroups: visibleGroups,
	}
	if filter.Keyword != "" {
		keyword := filter.Keyword
		deviceFilter.Search = &keyword
	}
	if filter.ProductType != "" {
		productType := filter.ProductType
		deviceFilter.ProductClass = &productType
	}
	if tech := techForCandidateFilter(catalog, filter.Category); tech != nil {
		deviceFilter.Technology = tech
	}

	devices, err := s.deviceRepo.List(ctx, deviceFilter)
	if err != nil {
		return nil, err
	}
	items := make([]DeviceItem, 0, len(devices.Items))
	for _, item := range devices.Items {
		if !upsLicensed && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(item.ProductClass)), "UPS") {
			continue
		}
		if typeDef != nil && !s.deviceMatchesTaskType(ctx, *typeDef, item.ProductClass) {
			continue
		}
		// #492：设备 productClass → 产品英文名（候选列表展示 + 按产品名收窄）。
		productName := ""
		if s.productNameLookup != nil && item.ProductClass != "" {
			if n, ok := s.productNameLookup(ctx, item.ProductClass); ok {
				productName = n
			}
		}
		if filter.ProductName != "" && productName != filter.ProductName {
			continue
		}
		currentVersion := item.FirmwareVersion
		if currentVersion == "" {
			currentVersion = "-"
		}
		items = append(items, DeviceItem{
			ID:              item.ID.String(),
			TaskID:          "",
			TaskName:        "",
			Category:        filter.Category,
			CategoryLabel:   categoryLabelForCandidate(catalog, filter.Category),
			TypeCode:        "",
			TypeDisplayName: "",
			DeviceName:      defaultDeviceName(item.DeviceName, item.SerialNumber, item.ProductClass),
			DeviceSN:        item.SerialNumber,
			ProductType:     item.ProductClass,
			ProductName:     productName,
			CurrentVersion:  currentVersion,
			TargetVersion:   "",
			Status:          "pending",
			Progress:        0,
			LastReportAt:    optionalTimePtr(item.UpdatedAt),
			OperatorScope:   "",
		})
	}
	return &coremodel.ListResponse[DeviceItem]{
		Items:      items,
		Total:      devices.Total,
		Page:       devices.Page,
		PageSize:   devices.PageSize,
		TotalPages: devices.TotalPages,
	}, nil
}

func (s *Service) loadTaskTypeCatalog(ctx context.Context) ([]TaskType, error) {
	stored, err := s.taskTypeRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	return materializeTaskTypes(stored), nil
}

func (s *Service) loadAllTasks(ctx context.Context, catalog []TaskType, filter TaskListFilter) ([]software.UpgradeTask, error) {
	typeSet, err := taskTypeFilterKeys(catalog, filter)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", commonerrors.ErrInvalidInput, err)
	}
	if len(typeSet) == 0 {
		return []software.UpgradeTask{}, nil
	}
	items := make([]software.UpgradeTask, 0)
	for taskType := range typeSet {
		page := 1
		for {
			pageResult, err := s.taskRepo.List(ctx, software.UpgradeTaskFilter{
				TaskType: taskTypePtr(taskType),
				Status:   softwareTaskStatusPtr(filter.Status),
				ListRequest: coremodel.ListRequest{
					Page:     page,
					PageSize: 200,
				},
			})
			if err != nil {
				return nil, err
			}
			items = append(items, pageResult.Items...)
			if page >= pageResult.TotalPages || len(pageResult.Items) == 0 {
				break
			}
			page++
		}
	}
	return items, nil
}

func (s *Service) loadAllSubTasks(ctx context.Context, catalog []TaskType, category, typeCode, taskIDStr string) ([]software.UpgradeSubTaskWithTaskName, error) {
	typeSet, err := deviceTypeFilterKeys(catalog, category, typeCode)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", commonerrors.ErrInvalidInput, err)
	}
	if len(typeSet) == 0 {
		return []software.UpgradeSubTaskWithTaskName{}, nil
	}
	taskIDPtr, err := parseTaskIDFilter(taskIDStr)
	if err != nil {
		return nil, err
	}
	items := make([]software.UpgradeSubTaskWithTaskName, 0)
	for taskType := range typeSet {
		page := 1
		for {
			pageResult, err := s.subTaskRepo.ListAll(ctx, software.AllSubTaskFilter{
				TaskType: taskTypePtr(taskType),
				TaskID:   taskIDPtr, // #615 SQL 下推：空 = 不过滤；非空 = WHERE ust.task_id = ?
				ListRequest: coremodel.ListRequest{
					Page:     page,
					PageSize: 200,
				},
			})
			if err != nil {
				return nil, err
			}
			items = append(items, pageResult.Items...)
			if page >= pageResult.TotalPages || len(pageResult.Items) == 0 {
				break
			}
			page++
		}
	}
	return items, nil
}

// parseTaskIDFilter 把 DeviceListFilter.TaskID（前端 query 形态：UUID 字符串）转成 *uuid.UUID。
// 空串 ≡ 不下推（保留原全 type 扫描语义）；非空但格式错 → ErrInvalidInput（→ 400）。
func parseTaskIDFilter(s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid taskId: %v", commonerrors.ErrInvalidInput, err)
	}
	return &id, nil
}

// deviceMatchesTaskType 判断 device 的 productClass 是否落在 taskType 的可选范围。
//
// 优先级：
//  1. PlatformScope 字符串子串匹配（item.PlatformScope 配的"5G gNB" / "4G eNB"
//     这种平台标签，与 productClass 大写后双向 Contains）—— matchesTaskTypeScope
//     已实现这一段。
//  2. techHint != nil 时，用 ProductRegistry 查 device.productClass → product.tech
//     ("lte" / "nr" / "gsm")，与 techHint 严格相等才放行。这一步对了，BNQ 产品
//     (pattern FAP/\w*BSC\w+, tech="nr") 在 5G 升级任务里就能命中，不再依赖
//     "BSC" 是不是出现在关键字白名单。
//  3. ProductRegistry 不可用 / productClass 字典里没注册 → 退回 matchesTaskTypeScope
//     的旧关键字模糊匹配（保留向后兼容，避免新装环境 / 单测环境无 registry 时一刀切）。
func (s *Service) deviceMatchesTaskType(ctx context.Context, item TaskType, productClass string) bool {
	// #492：product_scope（适用产品=产品英文名列表）= 设备匹配的唯一口径（弃用旧
	// PlatformScope 子串 + tech 关键字白名单，历史不兼容）。
	//   · 空    → 不限产品 = 适用「全部产品」（用户口径：空=所有；非升级模板如日志/恢复等默认如此）。
	//   · 非空  → 设备 productClass → ProductRegistry → product.Name，命中列表才放行。
	if len(item.Products) == 0 {
		return true
	}
	productClass = strings.TrimSpace(productClass)
	if productClass == "" {
		return false
	}
	if taskTypeProductsContain(item.Products, "UPS") && strings.HasPrefix(productClass, "UPS") {
		return true
	}
	if s.productNameLookup == nil {
		return false
	}
	name, ok := s.productNameLookup(ctx, productClass)
	if !ok || name == "" {
		return false
	}
	for _, p := range item.Products {
		if p == name {
			return true
		}
	}
	return false
}

// resolveTaskTypeForTask 是 mapTask / mapDeviceItem 用的"读路径"分类入口。
//
// 修 BUG: FAP/BSC7041C243 这种 5G 产品在 matchesTaskTypeScope 关键字白名单里
// "FAP" 子串先命中 LTE → resolveTaskType 把任务误归 ENB_IMG_UPGRADE → 5G 升级
// 任务跑进了 4G tab。修法是 read 路径上跟 create 路径（deviceMatchesTaskType）
// 对齐：先查 ProductRegistry 拿 tech，按 tech 直接选 catalog 里 techHint 匹配的
// catalog 项（GNB_IMG_UPGRADE / ENB_IMG_UPGRADE）；查不到才回退 resolveTaskType
// 的关键字模糊匹配。
//
// 只针对 TaskTypeUpgrade 走这条路径（升级类才有 4G/5G 之分；备份/日志/恢复等
// LogCollect 类不区分 tech）。
func (s *Service) resolveTaskTypeForTask(
	ctx context.Context,
	catalog []TaskType,
	taskType software.TaskType,
	productClass string,
	fileType string,
) (TaskType, bool) {
	if taskType == software.TaskTypeUpgrade && strings.HasPrefix(strings.TrimSpace(productClass), "UPS") {
		if item, ok := findTaskTypeByCode(catalog, "UPS_AP_UPGRADE"); ok {
			return item, true
		}
	}
	if taskType == software.TaskTypeUpgrade && s.productNameLookup != nil && productClass != "" {
		if productName, ok := s.productNameLookup(ctx, productClass); ok && strings.TrimSpace(productName) != "" {
			for _, item := range catalog {
				if item.softwareTaskType == taskType && taskTypeProductsContain(item.Products, productName) {
					return item, true
				}
			}
		}
	}
	if taskType == software.TaskTypeUpgrade && s.productTechLookup != nil && productClass != "" {
		if tech, ok := s.productTechLookup(ctx, productClass); ok && tech != "" {
			techCode := coremodel.Technology(tech)
			for _, item := range catalog {
				if item.softwareTaskType != taskType {
					continue
				}
				if item.techHint != nil && *item.techHint == techCode {
					return item, true
				}
			}
		}
	}
	return resolveTaskType(catalog, taskType, productClass, fileType)
}

func taskTypeProductsContain(products []string, productName string) bool {
	needle := strings.TrimSpace(productName)
	if needle == "" {
		return false
	}
	for _, product := range products {
		if strings.EqualFold(strings.TrimSpace(product), needle) {
			return true
		}
	}
	return false
}

func (s *Service) mapTask(ctx context.Context, catalog []TaskType, task *software.UpgradeTask) (*Task, error) {
	typeDef, ok := s.resolveTaskTypeForTask(ctx, catalog, task.TaskType, task.ProductClass, task.DownloadFileType)
	if !ok {
		return nil, fmt.Errorf("unsupported software task type %d", task.TaskType)
	}
	firmwareID := ""
	if task.FirmwareID != nil {
		firmwareID = task.FirmwareID.String()
	}
	// 备份 / 日志采集 / 配置恢复 等"OUTPUT 文件"类（softwareTaskType=LogCollect）
	// 的 task.FileName 存的是 *未渲染的模板字符串*（如 "backup-{task_id8}-{sn}.nv"，
	// 详见 software/service.go:BatchCollect 注释），对跨多设备的主任务来说没有意义。
	// 真实的"目标文件"在设备子任务级（DeviceItem.TargetFile）。
	taskTargetVersion := ""
	if typeDef.softwareTaskType != software.TaskTypeLogCollect {
		taskTargetVersion = strings.TrimSpace(task.FileName)
	}
	// IMS 任务把 ParamType 附加到类型展示名（"核心网参数采集（P3 IMS用户设置）"），
	// 让任务列表无需新列即可区分同模板不同参数类型的具体任务。
	typeDisplayName := typeDef.DisplayName
	fileType := typeDef.FileTypeLabel
	if typeDef.Category == "ims_core" {
		if pt := imsParamTypeFromStoredFileType(task.DownloadFileType); pt != "" {
			fileType = imsparam.DisplayName(pt)
			typeDisplayName = fmt.Sprintf("%s（%s）", typeDef.DisplayName, fileType)
		}
	}
	return &Task{
		ID:              task.ID.String(),
		TaskName:        task.TaskName,
		Category:        typeDef.Category,
		CategoryLabel:   typeDef.CategoryLabel,
		TypeCode:        typeDef.TypeCode,
		TypeDisplayName: typeDisplayName,
		FileType:        fileType,
		FirmwareID:      firmwareID,
		TargetVersion:   taskTargetVersion,
		ProductType:     task.ProductClass,
		ProductName:     s.resolveTaskProductName(ctx, task),
		IsKeepConfig:    task.IsKeepConfig,
		Status:          string(task.Status),
		Result:          normalizeTaskResult(task.Result),
		Progress:        progressFromCounts(task.TotalCount, task.SuccessCount, task.FailCount, task.Status),
		TotalCount:      task.TotalCount,
		SuccessCount:    task.SuccessCount,
		FailCount:       task.FailCount,
		CurrentStep:     stepForTask(typeDef, task.Status),
		ExecutionMode:   executionModeForTask(task.Status, task.CreateStatus),
		CreateUser:      task.CreateUser,
		CreatedAt:       optionalTimePtr(time.Time(task.CreatedAt)),
		StartedAt:       modelTimePtrToTimePtr(task.StartedAt),
		EndedAt:         modelTimePtrToTimePtr(task.EndedAt),
		ScheduledAt:     modelTimePtrToTimePtr(task.ScheduledAt),
		OperatorScope:   task.CreateUser,
	}, nil
}

// resolveTaskProductName 解析任务的产品英文名（与 DeviceItem.ProductName 同口径）。
// 升级类任务的 task.product_class 历史上写入的是 firmware.product_class（如「4G eNB」
// 这种宽泛类别），ProductRegistry 索引的是产品名（如「QAFA」→「甲产品」），直接 lookup 必 miss；
// 只有回滚类把 device.product_class 正确写入 task.product_class，能直接命中。
// 因此先按 task.product_class 直接 lookup（命中即返回 → 回滚/已正确写入场景），
// miss 时 fallback 查任意一个子任务的 device.product_class 再 lookup（升级/备份/license 场景）。
// productNameLookup 未注入或两条路径全 miss → 返回空串，前端 fallback 到 productType。
//
// 性能取舍：list 路径每个 task fallback 触发 1 次 sub_task 查询 + 1 次 device 查询，
// 单页 20-100 task 量级可接受；如压测确认是瓶颈再做 batch 预解析（独立 perf issue）。
func (s *Service) resolveTaskProductName(ctx context.Context, task *software.UpgradeTask) string {
	if s.productNameLookup == nil {
		return ""
	}
	if task.ProductClass != "" {
		if name, ok := s.productNameLookup(ctx, task.ProductClass); ok {
			return name
		}
	}
	if s.subTaskRepo == nil || s.deviceRepo == nil {
		return ""
	}
	res, err := s.subTaskRepo.ListByTaskID(ctx, task.ID, software.SubTaskFilter{
		TaskID:      task.ID,
		ListRequest: coremodel.ListRequest{Page: 1, PageSize: 1},
	})
	if err != nil || res == nil || len(res.Items) == 0 {
		return ""
	}
	deviceID := res.Items[0].DeviceID
	if deviceID == uuid.Nil {
		return ""
	}
	dev, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil || dev == nil || dev.ProductClass == "" {
		return ""
	}
	if name, ok := s.productNameLookup(ctx, dev.ProductClass); ok {
		return name
	}
	return ""
}

// normalizedTime 把 DB/model 读出的时间规范为 UTC instant。
// 后续统一响应层会按系统时区转换展示；这里不能提前按容器本地时区格式化成字符串。
func normalizedTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return t.UTC()
}

// modelTimePtrToTimePtr 保留可空语义，同时把时间规范为 UTC instant。
// 用于 Task.StartedAt / EndedAt / ScheduledAt 这类可空时间字段；nil 时 JSON omitempty 不输出。
func modelTimePtrToTimePtr(t *coremodel.Time) *time.Time {
	if t == nil {
		return nil
	}
	value := normalizedTime(time.Time(*t))
	if value.IsZero() {
		return nil
	}
	return &value
}

func optionalTimePtr(t time.Time) *time.Time {
	value := normalizedTime(t)
	if value.IsZero() {
		return nil
	}
	return &value
}

func isTaskLogCollectType(typeCode string) bool {
	return typeCode == "RUNTIME_LOG_COLLECT" || typeCode == "FAULT_LOG_COLLECT"
}

func timePtrAfter(left, right *time.Time) bool {
	if left == nil {
		return false
	}
	if right == nil {
		return true
	}
	return left.After(*right)
}

func (s *Service) mapDeviceItem(
	ctx context.Context,
	catalog []TaskType,
	subTask software.UpgradeSubTaskWithTaskName,
	parent *software.UpgradeTask,
	deviceCache map[uuid.UUID]*coremodel.Device,
) (*DeviceItem, error) {
	typeDef, ok := s.resolveTaskTypeForTask(ctx, catalog, parent.TaskType, parent.ProductClass, parent.DownloadFileType)
	if !ok {
		return nil, fmt.Errorf("unsupported software task type %d", parent.TaskType)
	}
	var dev *coremodel.Device
	if cached, ok := deviceCache[subTask.DeviceID]; ok {
		dev = cached
	} else {
		lookup, err := s.deviceRepo.GetByID(ctx, subTask.DeviceID)
		if err == nil {
			dev = lookup
			deviceCache[subTask.DeviceID] = lookup
		}
	}
	deviceName := defaultDeviceName("", subTask.DeviceSN, parent.ProductClass)
	productType := parent.ProductClass
	currentVersion := subTask.OriVersion
	if dev != nil {
		deviceName = defaultDeviceName(dev.DeviceName, dev.SerialNumber, dev.ProductClass)
		if dev.ProductClass != "" {
			productType = dev.ProductClass
		}
		if currentVersion == "" {
			currentVersion = dev.FirmwareVersion
		}
	}
	if currentVersion == "" {
		currentVersion = "-"
	}
	// #492：解析设备所属产品英文名（与候选列表 ListDeviceCandidates 同口径），让设备列表
	// 「产品名称」列展示产品名而非裸 productClass；解析不到（孤儿/未注册）留空，前端回退 productType。
	productName := ""
	if s.productNameLookup != nil && productType != "" {
		if n, ok := s.productNameLookup(ctx, productType); ok {
			productName = n
		}
	}
	// 备份 / 日志采集 / 配置恢复（softwareTaskType=LogCollect）的 TargetFile 完全由
	// 设备实际 PUT 上来的文件名决定（厂商如 baicells/MMMM 用自己的 NV 文件名，
	// 如 "mib-home-fap.nv"，跟 OMC 端模板渲染的 "backup-{taskId8}-{sn}.nv" 无关）。
	// 因此：按 (sn, sub_task.StartedAt) 反查 backup_restore_file 最新一行——同 sn
	// 同时只能有一个 active sub_task（DB unique 索引保证），started_at 之后的第一条
	// 必然属于当前任务。设备未上报前 TargetFile 留空。
	//
	// 升级 / 回滚类保持原口径：subTask.DestVersion → 退化 parent.FileName。
	targetVersion := subTask.DestVersion
	targetFile := "" // 设备实际上传的文件名（反查得到），未落地时空
	if typeDef.softwareTaskType == software.TaskTypeLogCollect {
		targetVersion = ""
	} else if targetVersion == "" {
		targetVersion = parent.FileName
	}
	// CONFIG_RESTORE / LICENSE_UPGRADE / IMS_FILE_DISTRIBUTE 是 ACS → 设备 的下发方向：
	// UFTE 派发后把实际下发文件名（snapshot / license / 参数文件 basename）写到
	// sub_task.dest_version，这里直接拎出来填到 TargetFile 让前端"目标文件"列正常展示。
	// 这几类不走 fileLandedLookup（没有上行文件落地），下面 if 分支也跳过。
	isDirectDispatchFile := typeDef.TypeCode == "CONFIG_RESTORE" ||
		typeDef.TypeCode == "LICENSE_UPGRADE" ||
		typeDef.TypeCode == "IMS_FILE_DISTRIBUTE"
	if isDirectDispatchFile && subTask.DestVersion != "" {
		targetFile = subTask.DestVersion
	}
	fileLanded := false
	mainTaskID := subTask.TaskID.String()
	if typeDef.softwareTaskType == software.TaskTypeLogCollect && !isDirectDispatchFile &&
		s.fileLandedLookup != nil && subTask.DeviceSN != "" {
		landedFile, landed, lookupErr := s.fileLandedLookup(ctx, subTask.DeviceSN, mainTaskID)
		if lookupErr != nil {
			s.logger.Debug("file landed lookup failed; treating as not-landed",
				zap.String("device_sn", subTask.DeviceSN),
				zap.String("main_task_id", mainTaskID),
				zap.Error(lookupErr))
		} else if landed {
			fileLanded = true
			targetFile = landedFile
		}
	}
	status := normalizeDeviceStatusForTask(typeDef.softwareTaskType, subTask.Status, fileLanded)
	result := ""
	if status == "ended" {
		result = "success"
	}
	if status == "failed" {
		if subTask.Status == software.UpgradeTerminated {
			result = "terminated"
		} else {
			result = "failure"
		}
	}
	// 上报时间只在文件真正上报成功（终态 ended）时才填——issue #195。
	// 子任务一生成 / 中间状态流转都会刷新 sub_task.updated_at，但那不是
	// "文件上报成功"的时刻，直接拿 updated_at 会在任务刚创建时就显示一个
	// 误导性时间。未到成功终态时留零值，formatTime 渲染为空。
	var lastReport time.Time
	if status == "ended" {
		lastReport = time.Time(subTask.UpdatedAt)
	}
	fileDeleted := false
	if isTaskLogCollectType(typeDef.TypeCode) && targetFile != "" && status == "ended" &&
		s.fileDeletedLookup != nil && subTask.DeviceSN != "" {
		deleted, lookupErr := s.fileDeletedLookup(ctx, subTask.DeviceSN, mainTaskID, targetFile)
		if lookupErr != nil {
			s.logger.Debug("file deleted lookup failed; treating as active",
				zap.String("device_sn", subTask.DeviceSN),
				zap.String("main_task_id", mainTaskID),
				zap.String("target_file", targetFile),
				zap.Error(lookupErr))
		} else {
			fileDeleted = deleted
		}
	}
	// 完成态 LogCollect 类（备份等）且注入了下载回调时，拉取 1h presigned GET URL。
	// 用 targetFile（设备实际上传文件名，从 fileLanded 反查得到）做 lookup key——
	// 设备厂商命名不可预测，预渲染模板名匹配不上 MinIO 对象路径。
	//
	// CONFIG_RESTORE / LICENSE_UPGRADE 是 ACS → 设备 的下行方向，downloadURLLookup
	// 查的是设备上行的"已落地文件"，对它们语义不对——跳过，文件名以纯文本展示。
	downloadURL := ""
	if !fileDeleted && !isDirectDispatchFile && targetFile != "" && status == "ended" && s.downloadURLLookup != nil && subTask.DeviceSN != "" {
		url, lookupErr := s.downloadURLLookup(ctx, subTask.DeviceSN, targetFile)
		if lookupErr != nil {
			s.logger.Debug("download URL lookup failed; leaving blank",
				zap.String("device_sn", subTask.DeviceSN),
				zap.String("target_file", targetFile),
				zap.Error(lookupErr))
		} else {
			downloadURL = url
		}
	}
	// 配置备份类（CONFIG_BACKUP_XML / NV）展示用规范化名 <SN>_CFG.<ext>，与
	// 配置快照库口径一致。downloadURL 已用真实 CPE 文件名换好 presigned URL，
	// 这里改名只影响 UI 展示，不影响下载。日志类（RUNTIME_LOG / FAULT_LOG）
	// 保留设备原始上传名。
	if targetFile != "" && (typeDef.TypeCode == "CONFIG_BACKUP_XML" || typeDef.TypeCode == "CONFIG_BACKUP_NV") {
		if dot := strings.LastIndex(targetFile, "."); dot >= 0 {
			ext := strings.ToLower(targetFile[dot+1:])
			if ext == "xml" || ext == "nv" {
				targetFile = fmt.Sprintf("%s_CFG.%s", subTask.DeviceSN, ext)
			}
		}
	}

	startedAt := modelTimePtrToTimePtr(subTask.StartedAt)
	endedAt := modelTimePtrToTimePtr(subTask.CompletedAt)
	failureReason := normalizeFailureReasonForTask(typeDef.softwareTaskType, subTask.CommandKey, subTask.FailureReason, subTask.ErrorMessage)
	failureDetail := normalizeFailureDetailForTask(typeDef.softwareTaskType, failureReason, subTask.ErrorMessage)
	// issue #655 追加：被操作者主动终止的子任务，多半在「未进入执行态」前就被叫停，
	// 因此 sub_task.started_at 一般为空。前端列「开始时间 / 结束时间」只有结束、没开始
	// 显示别扭——补一个 startedAt = endedAt 让两端对齐（语义上"在结束的同一刻被终止"）。
	// failure_reason 同理：TerminateUpgrade 只写 error_message="task terminated by operator"，
	// failure_reason 留空，前端「失败原因」列空着不够直观——补稳定 code，由前端 i18n
	// 和 CSV 导出各自翻译，避免英文环境露中文。
	if result == "terminated" {
		if startedAt == nil && endedAt != nil {
			startedAt = endedAt
		}
		if failureReason == "" {
			failureReason = failureCodeOperatorTerminated
		}
	}
	if isDirectDispatchFile && status == "ended" && startedAt == nil && endedAt != nil {
		startedAt = endedAt
	}

	// IMS 任务设备列表同步附加 ParamType 后缀（与 mapTask 口径一致）。
	deviceTypeDisplayName := typeDef.DisplayName
	if typeDef.Category == "ims_core" {
		if pt := imsParamTypeFromStoredFileType(parent.DownloadFileType); pt != "" {
			deviceTypeDisplayName = fmt.Sprintf("%s（%s）", typeDef.DisplayName, imsparam.DisplayName(pt))
		}
	}
	return &DeviceItem{
		ID:              subTask.ID.String(),
		TaskID:          subTask.TaskID.String(),
		TaskName:        parent.TaskName,
		Category:        typeDef.Category,
		CategoryLabel:   typeDef.CategoryLabel,
		TypeCode:        typeDef.TypeCode,
		TypeDisplayName: deviceTypeDisplayName,
		DeviceName:      deviceName,
		DeviceSN:        subTask.DeviceSN,
		ProductType:     productType,
		ProductName:     productName,
		CurrentVersion:  currentVersion,
		TargetVersion:   targetVersion,
		TargetFile:      targetFile,
		DownloadURL:     downloadURL,
		FileDeleted:     fileDeleted,
		Status:          status,
		Result:          result,
		Progress:        progressForDeviceStatus(status),
		// issue #655：StartedAt / EndedAt 直接透传 PG repo 写入的 sub_task.started_at /
		// completed_at（见 pg_upgrade_repository.go UpdateStatusWithCode，COALESCE 守卫
		// 首次执行态写一次后不再覆盖）。LastReportAt 保留兼容 CSV / 北向 API。
		// 特例：terminated 以及旧 CONFIG_RESTORE / LICENSE_UPGRADE 成功终态缺 started_at 时，
		// startedAt 兜底 = endedAt，避免页面显示 "-".
		StartedAt:     startedAt,
		EndedAt:       endedAt,
		LastReportAt:  optionalTimePtr(lastReport),
		CreatedAt:     optionalTimePtr(time.Time(subTask.CreatedAt)),
		OperatorScope: parent.CreateUser,
		FailureReason: failureReason,
		FailureDetail: failureDetail,
	}, nil
}

func paginate[T any](items []T, page, pageSize int) *coremodel.ListResponse[T] {
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if page < 1 {
		page = 1
	}
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}
	return &coremodel.ListResponse[T]{
		Items:      items[start:end],
		Total:      int64(total),
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

func matchesTaskFilter(item Task, filter TaskListFilter) bool {
	if filter.Status != "" && item.Status != filter.Status {
		return false
	}
	if !categoryMatchesFilter(filter.Category, item.Category) {
		return false
	}
	if filter.TypeCode != "" && item.TypeCode != filter.TypeCode {
		return false
	}
	if filter.Keyword != "" {
		needle := strings.ToLower(strings.TrimSpace(filter.Keyword))
		if !strings.Contains(strings.ToLower(item.TaskName), needle) &&
			!strings.Contains(strings.ToLower(item.TypeDisplayName), needle) &&
			!strings.Contains(strings.ToLower(item.ProductType), needle) &&
			!strings.Contains(strings.ToLower(item.FileType), needle) {
			return false
		}
	}
	return true
}

func matchesDeviceFilter(item DeviceItem, filter DeviceListFilter) bool {
	if filter.Status != "" && item.Status != filter.Status {
		return false
	}
	if !categoryMatchesFilter(filter.Category, item.Category) {
		return false
	}
	if filter.TypeCode != "" && item.TypeCode != filter.TypeCode {
		return false
	}
	// #615：按主任务 ID 精确收窄。空值 ≡ 不过滤（原有「执行明细」全量入口语义）。
	if filter.TaskID != "" && item.TaskID != filter.TaskID {
		return false
	}
	if filter.ProductType != "" && !strings.EqualFold(item.ProductType, filter.ProductType) {
		return false
	}
	// #524：按产品名过滤。item.ProductName 由 mapDeviceItem 经 productNameLookup 回填
	// （productClass→产品英文名），口径与设备列表「产品名称」列一致。
	if filter.ProductName != "" && !strings.EqualFold(item.ProductName, filter.ProductName) {
		return false
	}
	if filter.Keyword != "" {
		needle := strings.ToLower(strings.TrimSpace(filter.Keyword))
		if !strings.Contains(strings.ToLower(item.TaskName), needle) &&
			!strings.Contains(strings.ToLower(item.DeviceName), needle) &&
			!strings.Contains(strings.ToLower(item.DeviceSN), needle) &&
			!strings.Contains(strings.ToLower(item.ProductType), needle) {
			return false
		}
	}
	return true
}

// executionModeForTask 把后端 task.Status × task.CreateStatus 映射到前端 executionMode 枚举。
//
//	create_status='timing' 且 status in (pending, in_progress) → "scheduled"
//	status='suspended'                                          → "suspended"
//	其它（status=in_progress / pending(active) / ended）         → "immediate"
//
// 定时任务被 scheduler 触发后 status 翻成 in_progress 但 create_status 仍是 timing；
// 前端列"执行方式"应继续显示"定时执行"，所以 timing 的优先级最高。
func executionModeForTask(status software.TaskStatus, createStatus string) string {
	if createStatus == software.CreateStatusTiming {
		return "scheduled"
	}
	if status == software.TaskSuspended {
		return "suspended"
	}
	return "immediate"
}

func softwareTaskStatusPtr(status string) *software.TaskStatus {
	if status == "" {
		return nil
	}
	v := software.TaskStatus(status)
	return &v
}

func taskTypePtr(value software.TaskType) *software.TaskType {
	v := value
	return &v
}

type taskTypeStat struct {
	taskCount   int
	successRate float64
	totalCount  int
	successes   int
}

func (s *Service) buildTaskTypeStats(ctx context.Context, catalog []TaskType) (map[string]taskTypeStat, error) {
	tasks, err := s.loadAllTasks(ctx, catalog, TaskListFilter{})
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().AddDate(0, 0, -30)
	stats := make(map[string]taskTypeStat)
	for _, task := range tasks {
		createdAt := time.Time(task.CreatedAt)
		if createdAt.Before(cutoff) {
			continue
		}
		typeDef, ok := resolveTaskType(catalog, task.TaskType, task.ProductClass, task.DownloadFileType)
		if !ok {
			continue
		}
		stat := stats[typeDef.TypeCode]
		stat.taskCount++
		stat.totalCount += task.TotalCount
		stat.successes += task.SuccessCount
		stats[typeDef.TypeCode] = stat
	}
	for typeCode, stat := range stats {
		if stat.totalCount > 0 {
			stat.successRate = float64(stat.successes) / float64(stat.totalCount) * 100
		}
		stats[typeCode] = stat
	}
	return stats, nil
}

func taskTypeFromWriteRequest(typeCode, permissionCode string, builtIn bool, req TaskTypeWriteRequest, editor string) TaskType {
	updatedAt := formatTime(time.Now())
	rpcType := strings.TrimSpace(req.RPCType)
	fileType := normalizeTaskTypeFileType(rpcType, strings.TrimSpace(req.FileType))
	firmwareFileType := normalizeTaskTypeFirmwareFileType(rpcType, req.FirmwareFileType, fileType)
	fileTypeLabel := strings.TrimSpace(req.FileTypeLabel)
	if fileTypeLabel == "" || fileTypeLabel == strings.TrimSpace(req.FileType) {
		fileTypeLabel = fileType
	}
	return TaskType{
		TypeCode:               typeCode,
		Category:               strings.TrimSpace(req.Category),
		CategoryLabel:          strings.TrimSpace(req.CategoryLabel),
		DisplayName:            strings.TrimSpace(req.DisplayName),
		Description:            strings.TrimSpace(req.Description),
		RPCType:                rpcType,
		BuiltIn:                builtIn,
		Enabled:                req.Enabled,
		StepChain:              normalizeStringSlice(req.StepChain),
		PostTCEventCode:        strings.TrimSpace(req.PostTCEventCode),
		PermissionCode:         permissionCode,
		PlatformScope:          normalizeStringSlice(req.PlatformScope),
		Products:               normalizeStringSlice(req.Products),
		FileType:               fileType,
		FileTypeLabel:          fileTypeLabel,
		FileTypeEditable:       req.FileTypeEditable,
		FirmwareFileType:       firmwareFileType,
		URLTemplate:            strings.TrimSpace(req.URLTemplate),
		TargetFileNameTemplate: strings.TrimSpace(req.TargetFileNameTemplate),
		FileNameTemplate:       strings.TrimSpace(req.FileNameTemplate),
		FileSizeField:          strings.TrimSpace(req.FileSizeField),
		ChecksumField:          strings.TrimSpace(req.ChecksumField),
		RawMode:                strings.TrimSpace(req.RawMode),
		DelaySeconds:           req.DelaySeconds,
		TransportPath:          strings.TrimSpace(req.TransportPath),
		LastEditor:             strings.TrimSpace(editor),
		UpdatedAt:              updatedAt,
	}
}

func normalizeTaskTypeFileType(rpcType, fileType string) string {
	if strings.ToUpper(strings.TrimSpace(rpcType)) != "DOWNLOAD" {
		return fileType
	}

	switch strings.TrimSpace(fileType) {
	case "1", "firmware", "Firmware Upgrade Image":
		return "1 Firmware Upgrade Image"
	case "2", "web", "Web Content":
		return "2 Web Content"
	case "3", "config", "Vendor Configuration File":
		return "3 Vendor Configuration File"
	case "101", "script", "Script File":
		return "101 Script File"
	case "103", "startup", "Base Station Startup File":
		return "103 Base Station Startup File"
	default:
		return fileType
	}
}

func normalizeTaskTypeFirmwareFileType(rpcType string, firmwareFileType *software.FileType, fileType string) *software.FileType {
	if strings.ToUpper(strings.TrimSpace(rpcType)) != "DOWNLOAD" {
		return nil
	}
	if firmwareFileType != nil {
		if *firmwareFileType < 0 {
			return nil
		}
		return firmwareFileTypePtr(*firmwareFileType)
	}
	return inferTaskTypeFirmwareFileType(fileType)
}

func inferTaskTypeFirmwareFileType(fileType string) *software.FileType {
	normalized := strings.ToUpper(strings.TrimSpace(fileType))
	switch {
	case normalized == "":
		return nil
	case strings.Contains(normalized, "FPGA"):
		return firmwareFileTypePtr(software.FileTypeFPGA)
	case strings.Contains(normalized, "PATCH"):
		return firmwareFileTypePtr(software.FileTypePATCH)
	case strings.HasPrefix(normalized, "1 "), normalized == "1", strings.Contains(normalized, "FIRMWARE UPGRADE IMAGE"):
		return firmwareFileTypePtr(software.FileTypeIMG)
	default:
		return nil
	}
}

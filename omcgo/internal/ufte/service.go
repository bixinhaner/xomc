package ufte

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/software"
)

type Service struct {
	softwareService *software.SoftwareService
	taskTypeRepo    TaskTypeRepository
	taskRepo        software.TaskRepository
	subTaskRepo     software.SubTaskRepository
	deviceRepo      device.DeviceRepository
	logger          *zap.Logger
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
	for _, task := range tasks {
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
	for _, item := range catalog {
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
	return catalog, nil
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
	if err := s.taskTypeRepo.Upsert(ctx, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *Service) StartTask(ctx context.Context, taskID uuid.UUID) error {
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
		transportPath, rpcType, paramPath := "", "", ""
		for _, tt := range catalog {
			if tt.softwareTaskType == software.TaskTypeLogCollect && tt.FileType == task.DownloadFileType {
				transportPath = tt.TransportPath
				rpcType = tt.RPCType
				// FAULT_LOG_COLLECT 用 URLTemplate 存 SPV 参数路径（如
				// Device.DeviceInfo.X_COM_Log.FaultLogURL）。详见 model.go 注释。
				paramPath = tt.URLTemplate
				break
			}
		}
		return s.softwareService.ResumeCollect(ctx, taskID, transportPath, rpcType, paramPath)
	}
	return s.softwareService.ResumeUpgrade(ctx, taskID)
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
	for _, tt := range catalog {
		if tt.softwareTaskType == software.TaskTypeLogCollect && tt.FileType == parent.DownloadFileType {
			transportPath = tt.TransportPath
			rpcType = tt.RPCType
			paramPath = tt.URLTemplate
			break
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
	s.softwareService.ExecuteOneUploadDirect(ctx, subTask, parent.DownloadFileType, parent.FileName, transportPath)
	return nil
}

func (s *Service) SuspendTask(ctx context.Context, taskID uuid.UUID) error {
	return s.softwareService.SuspendUpgrade(ctx, taskID)
}

func (s *Service) TerminateTask(ctx context.Context, taskID uuid.UUID) error {
	return s.softwareService.TerminateUpgrade(ctx, taskID)
}

func (s *Service) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	return s.softwareService.DeleteUpgrade(ctx, taskID)
}

func (s *Service) RetryTask(ctx context.Context, taskID uuid.UUID) error {
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

func (s *Service) CreateTask(ctx context.Context, req CreateTaskRequest, createUser string) (*Task, error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	typeDef, ok := findTaskTypeByCode(catalog, req.TypeCode)
	if !ok {
		return nil, fmt.Errorf("%w: unsupported UFTE type %s", commonerrors.ErrInvalidInput, req.TypeCode)
	}
	if req.ExecutionMode == "scheduled" {
		return nil, fmt.Errorf("%w: scheduled execution is not supported by the current UFTE upgrade/rollback adapter", commonerrors.ErrInvalidInput)
	}
	if len(req.DeviceIDs) == 0 {
		return nil, fmt.Errorf("%w: device_ids is required", commonerrors.ErrInvalidInput)
	}

	var createdTask *software.UpgradeTask
	createSuspended := req.ExecutionMode == "suspended"

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
		})
	case typeDef.TypeCode == "VERSION_ROLLBACK":
		createdTask, err = s.softwareService.RollbackDevices(ctx, software.RollbackRequest{
			DeviceIDs:       req.DeviceIDs,
			TaskName:        req.TaskName,
			CreateUser:      createUser,
			CreateSuspended: createSuspended,
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
			IsKeepConfig:     req.IsKeepConfig,
			CreateSuspended:  createSuspended,
		})
	default:
		return nil, fmt.Errorf("%w: UFTE type %s is not executable by the current backend adapter", commonerrors.ErrInvalidInput, req.TypeCode)
	}
	if err != nil {
		return nil, err
	}
	return s.mapTask(catalog, createdTask)
}

func (s *Service) ListTasks(ctx context.Context, filter TaskListFilter) (*coremodel.ListResponse[Task], error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	tasks, err := s.loadAllTasks(ctx, catalog, filter)
	if err != nil {
		return nil, err
	}
	items := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		taskCopy := task
		mapped, err := s.mapTask(catalog, &taskCopy)
		if err != nil {
			s.logger.Debug("skip UFTE task", zap.String("task_id", task.ID.String()), zap.Error(err))
			continue
		}
		if !matchesTaskFilter(*mapped, filter) {
			continue
		}
		items = append(items, *mapped)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return paginate(items, filter.Page, filter.PageSize), nil
}

func (s *Service) ListDevices(ctx context.Context, filter DeviceListFilter) (*coremodel.ListResponse[DeviceItem], error) {
	catalog, err := s.loadTaskTypeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	subTasks, err := s.loadAllSubTasks(ctx, catalog, filter.Category, filter.TypeCode)
	if err != nil {
		return nil, err
	}
	deviceCache := make(map[uuid.UUID]*coremodel.Device)
	parentCache := make(map[uuid.UUID]*software.UpgradeTask)
	items := make([]DeviceItem, 0, len(subTasks))
	for _, subTask := range subTasks {
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
		if !matchesDeviceFilter(*mapped, filter) {
			continue
		}
		items = append(items, *mapped)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastReportAt > items[j].LastReportAt
	})
	return paginate(items, filter.Page, filter.PageSize), nil
}

func (s *Service) ListDeviceCandidates(ctx context.Context, filter DeviceCandidateFilter) (*coremodel.ListResponse[DeviceItem], error) {
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
		typeDef = &item
		if filter.Category == "" {
			filter.Category = item.Category
		}
	}
	deviceFilter := device.DeviceFilter{
		ListRequest: coremodel.ListRequest{
			Page:     filter.Page,
			PageSize: filter.PageSize,
		},
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
		if typeDef != nil && !matchesTaskTypeScope(*typeDef, item.ProductClass) {
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
			CurrentVersion:  currentVersion,
			TargetVersion:   "",
			Status:          "pending",
			Progress:        0,
			LastReportAt:    formatTime(item.UpdatedAt),
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

func (s *Service) loadAllSubTasks(ctx context.Context, catalog []TaskType, category, typeCode string) ([]software.UpgradeSubTaskWithTaskName, error) {
	typeSet, err := deviceTypeFilterKeys(catalog, category, typeCode)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", commonerrors.ErrInvalidInput, err)
	}
	if len(typeSet) == 0 {
		return []software.UpgradeSubTaskWithTaskName{}, nil
	}
	items := make([]software.UpgradeSubTaskWithTaskName, 0)
	for taskType := range typeSet {
		page := 1
		for {
			pageResult, err := s.subTaskRepo.ListAll(ctx, software.AllSubTaskFilter{
				TaskType: taskTypePtr(taskType),
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

func (s *Service) mapTask(catalog []TaskType, task *software.UpgradeTask) (*Task, error) {
	typeDef, ok := resolveTaskType(catalog, task.TaskType, task.ProductClass, task.DownloadFileType)
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
	return &Task{
		ID:              task.ID.String(),
		TaskName:        task.TaskName,
		Category:        typeDef.Category,
		CategoryLabel:   typeDef.CategoryLabel,
		TypeCode:        typeDef.TypeCode,
		TypeDisplayName: typeDef.DisplayName,
		FirmwareID:      firmwareID,
		TargetVersion:   taskTargetVersion,
		ProductType:     task.ProductClass,
		IsKeepConfig:    task.IsKeepConfig,
		Status:          string(task.Status),
		Result:          normalizeTaskResult(task.Result),
		Progress:        progressFromCounts(task.TotalCount, task.SuccessCount, task.FailCount, task.Status),
		TotalCount:      task.TotalCount,
		SuccessCount:    task.SuccessCount,
		FailCount:       task.FailCount,
		CurrentStep:     stepForTask(typeDef, task.Status),
		ExecutionMode:   executionModeForTask(task.Status),
		CreateUser:      task.CreateUser,
		CreatedAt:       formatTime(time.Time(task.CreatedAt)),
		OperatorScope:   task.CreateUser,
	}, nil
}

func (s *Service) mapDeviceItem(
	ctx context.Context,
	catalog []TaskType,
	subTask software.UpgradeSubTaskWithTaskName,
	parent *software.UpgradeTask,
	deviceCache map[uuid.UUID]*coremodel.Device,
) (*DeviceItem, error) {
	typeDef, ok := resolveTaskType(catalog, parent.TaskType, parent.ProductClass, parent.DownloadFileType)
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
	fileLanded := false
	if typeDef.softwareTaskType == software.TaskTypeLogCollect &&
		s.fileLandedLookup != nil && subTask.DeviceSN != "" {
		mainTaskID := subTask.TaskID.String()
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
	status := normalizeDeviceStatus(subTask.Status, fileLanded)
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
	lastReport := time.Time(subTask.UpdatedAt)
	// 完成态 LogCollect 类（备份等）且注入了下载回调时，拉取 1h presigned GET URL。
	// 用 targetFile（设备实际上传文件名，从 fileLanded 反查得到）做 lookup key——
	// 设备厂商命名不可预测，预渲染模板名匹配不上 MinIO 对象路径。
	downloadURL := ""
	if targetFile != "" && status == "ended" && s.downloadURLLookup != nil && subTask.DeviceSN != "" {
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

	return &DeviceItem{
		ID:              subTask.ID.String(),
		TaskID:          subTask.TaskID.String(),
		TaskName:        parent.TaskName,
		Category:        typeDef.Category,
		CategoryLabel:   typeDef.CategoryLabel,
		TypeCode:        typeDef.TypeCode,
		TypeDisplayName: typeDef.DisplayName,
		DeviceName:      deviceName,
		DeviceSN:        subTask.DeviceSN,
		ProductType:     productType,
		CurrentVersion:  currentVersion,
		TargetVersion:   targetVersion,
		TargetFile:      targetFile,
		DownloadURL:     downloadURL,
		Status:          status,
		Result:          result,
		Progress:        progressForDeviceStatus(status),
		LastReportAt:    formatTime(lastReport),
		OperatorScope:   parent.CreateUser,
		FailureReason:   subTask.FailureReason,
		FailureDetail:   subTask.ErrorMessage,
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
	if filter.Category != "" && item.Category != filter.Category {
		return false
	}
	if filter.TypeCode != "" && item.TypeCode != filter.TypeCode {
		return false
	}
	if filter.Keyword != "" {
		needle := strings.ToLower(strings.TrimSpace(filter.Keyword))
		if !strings.Contains(strings.ToLower(item.TaskName), needle) &&
			!strings.Contains(strings.ToLower(item.TypeDisplayName), needle) &&
			!strings.Contains(strings.ToLower(item.ProductType), needle) {
			return false
		}
	}
	return true
}

func matchesDeviceFilter(item DeviceItem, filter DeviceListFilter) bool {
	if filter.Status != "" && item.Status != filter.Status {
		return false
	}
	if filter.Category != "" && item.Category != filter.Category {
		return false
	}
	if filter.TypeCode != "" && item.TypeCode != filter.TypeCode {
		return false
	}
	if filter.ProductType != "" && !strings.EqualFold(item.ProductType, filter.ProductType) {
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

func executionModeForTask(status software.TaskStatus) string {
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

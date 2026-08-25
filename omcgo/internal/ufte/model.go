package ufte

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/imsparam"
	"github.com/omcgo/omcgo/internal/software"
)

const deviceUpgradeVirtualCategory = "device_upgrade"

type Overview struct {
	EnabledTypeCount int     `json:"enabledTypeCount"`
	RunningTaskCount int     `json:"runningTaskCount"`
	CustomTypeCount  int     `json:"customTypeCount"`
	SuccessRate30d   float64 `json:"successRate30d"`
}

type TaskType struct {
	TypeCode        string   `json:"typeCode"`
	Category        string   `json:"category"`
	CategoryLabel   string   `json:"categoryLabel"`
	DisplayName     string   `json:"displayName"`
	Description     string   `json:"description"`
	RPCType         string   `json:"rpcType"`
	BuiltIn         bool     `json:"builtIn"`
	Enabled         bool     `json:"enabled"`
	StepChain       []string `json:"stepChain"`
	PostTCEventCode string   `json:"postTcEventCode,omitempty"`
	PermissionCode  string   `json:"permissionCode"`
	PlatformScope   []string `json:"platformScope"`
	// Products 是「适用产品」= 产品英文名列表（引用 products.product_name，#492）。
	// 非空时设备候选匹配走产品目录精确匹配（deviceMatchesTaskType），制式由所选产品 tech 派生；
	// 空则回退旧 PlatformScope 子串 + techHint 关键字匹配（灰度兼容）。
	Products               []string           `json:"products"`
	FileType               string             `json:"fileType"`
	FileTypeLabel          string             `json:"fileTypeLabel"`
	FileTypeEditable       bool               `json:"fileTypeEditable"`
	FirmwareFileType       *software.FileType `json:"firmwareFileType,omitempty"`
	URLTemplate            string             `json:"urlTemplate,omitempty"`
	TargetFileNameTemplate string             `json:"targetFileNameTemplate,omitempty"`
	FileNameTemplate       string             `json:"fileNameTemplate,omitempty"`
	FileSizeField          string             `json:"fileSizeField,omitempty"`
	ChecksumField          string             `json:"checksumField,omitempty"`
	RawMode                string             `json:"rawMode,omitempty"`
	DelaySeconds           int                `json:"delaySeconds,omitempty"`
	TransportPath          string             `json:"transportPath,omitempty"`
	// SortOrder 控制「任务创建」/「模板配置」子 tab 显示顺序；数字小靠前。
	// 内置模板初始值由 migrations/000146 赋（10/20/30/...），自定义默认 100。
	// 后续在「模板配置」UI 拖拽可改。
	SortOrder      int     `json:"sortOrder"`
	LastEditor     string  `json:"lastEditor"`
	TaskCount30d   int     `json:"taskCount30d"`
	SuccessRate30d float64 `json:"successRate30d"`
	UpdatedAt      string  `json:"updatedAt"`

	softwareTaskType software.TaskType
	techHint         *coremodel.Technology
}

type Task struct {
	ID              string `json:"id"`
	TaskName        string `json:"taskName"`
	Category        string `json:"category"`
	CategoryLabel   string `json:"categoryLabel"`
	TypeCode        string `json:"typeCode"`
	TypeDisplayName string `json:"typeDisplayName"`
	FileType        string `json:"fileType"`
	FirmwareID      string `json:"firmwareId,omitempty"`
	TargetVersion   string `json:"targetVersion,omitempty"`
	ProductType     string `json:"productType,omitempty"`
	// ProductName 是 task.product_class 经 ProductRegistry 解析出的产品英文名（如「甲产品」），
	// 与 DeviceItem.ProductName 同口径；解析不到（task.product_class 是「4G eNB」这种宽泛
	// 类别 / 未注册）时由 mapTask 回退查任一子任务设备的 productClass 再解析，仍无果则空串
	// → 前端 fallback 到 ProductType。修复 #682：任务列表「产品名称」误显示 firmware.product_class。
	ProductName   string     `json:"productName,omitempty"`
	IsKeepConfig  bool       `json:"isKeepConfig,omitempty"`
	Status        string     `json:"status"`
	Result        string     `json:"result,omitempty"`
	Progress      int        `json:"progress"`
	TotalCount    int        `json:"totalCount"`
	SuccessCount  int        `json:"successCount"`
	FailCount     int        `json:"failCount"`
	CurrentStep   string     `json:"currentStep"`
	ExecutionMode string     `json:"executionMode"`
	CreateUser    string     `json:"createUser"`
	CreatedAt     *time.Time `json:"createdAt,omitempty"`
	// StartedAt 任务真正开始下发（状态首次进入 in_progress 时由 PG repo 自动写入 upgrade_tasks.started_at）；未开始时为空 → JSON omitempty 不输出。
	StartedAt *time.Time `json:"startedAt,omitempty"`
	// EndedAt 任务到达终态（ended / 含成功/失败/终止）时由 PG repo 自动写入 upgrade_tasks.ended_at；未结束时为空。
	EndedAt       *time.Time `json:"endedAt,omitempty"`
	ScheduledAt   *time.Time `json:"scheduledAt,omitempty"`
	OperatorScope string     `json:"operatorScope"`
}

type DeviceItem struct {
	ID              string `json:"id"`
	TaskID          string `json:"taskId"`
	TaskName        string `json:"taskName"`
	Category        string `json:"category"`
	CategoryLabel   string `json:"categoryLabel"`
	TypeCode        string `json:"typeCode"`
	TypeDisplayName string `json:"typeDisplayName"`
	DeviceName      string `json:"deviceName"`
	DeviceSN        string `json:"deviceSN"`
	ProductType     string `json:"productType"`
	// ProductName 是设备 productClass 经 ProductRegistry 解析出的产品英文名（#492）。
	// 前端候选/设备列表展示产品名（取代裸 productClass）；解析不到（孤儿/未注册）时为空。
	ProductName    string `json:"productName,omitempty"`
	CurrentVersion string `json:"currentVersion"`
	TargetVersion  string `json:"targetVersion"`
	// TargetFile 是"OUTPUT 文件类"（备份 / 日志采集 / 配置恢复，softwareTaskType=LogCollect）
	// 的目标文件名（如 "backup-a1b2c3d4-SN001.nv"），由 {task_id8}/{sn} 模板按运行时渲染。
	// 升级 / 回滚类不写本字段。
	TargetFile string `json:"targetFile,omitempty"`
	// DownloadURL 是 CPE 上传完成、ACS 落 MinIO 后的 1h presigned GET 链接；
	// 仅当子任务已 ended 且 backup_restore_file 元数据存在时填充，否则留空。
	DownloadURL string `json:"downloadUrl,omitempty"`
	// FileDeleted 表示 TargetFile 对应的日志文件已被站点日志配额软删。
	// 前端仍展示文件名，但不可点击下载，并提示清理原因。
	FileDeleted bool   `json:"fileDeleted,omitempty"`
	Status      string `json:"status"`
	Result      string `json:"result,omitempty"`
	Progress    int    `json:"progress"`
	// StartedAt 子任务首次进入执行态（downloading/uploading/rebooting/verifying）时由 PG repo
	// 写入 upgrade_sub_tasks.started_at；未开始为空 → JSON omitempty 不输出。
	// 三皮肤设备列表「开始时间」列展示。
	StartedAt *time.Time `json:"startedAt,omitempty"`
	// EndedAt 子任务到达终态（completed/failed/terminated）时由 PG repo 写入
	// upgrade_sub_tasks.completed_at；未结束为空。三皮肤设备列表「结束时间」列展示。
	EndedAt *time.Time `json:"endedAt,omitempty"`
	// LastReportAt 仅在子任务到达上报成功终态（ended）时填 sub_task.updated_at；
	// 历史字段，CSV 导出 / 北向 API 仍依赖，前端三皮肤设备列表已改用 StartedAt + EndedAt
	// 展示，不再直接用本字段（见 issue #655）。
	LastReportAt *time.Time `json:"lastReportAt,omitempty"`
	// CreatedAt 是设备子任务的创建时刻（设备加入任务的时间），仅用于列表稳定排序。
	// 不能用 LastReportAt 排序：它只在上报成功终态才有值（见 mapDeviceItem / issue #195），
	// 未上报设备为空串，倒序会把新建任务的设备挤到列表最后。
	CreatedAt     *time.Time `json:"createdAt,omitempty"`
	OperatorScope string     `json:"operatorScope"`
	FailureReason string     `json:"failureReason,omitempty"`
	// FailureDetail 是设备失败时的详细错误描述（含 FaultCode + FaultString 原文），
	// 比如 TC 失败时填："FaultCode: 0, FaultString: httpUpload OM Http Put Upload stat file error"。
	// 前端「设备失败信息」列展示原文，比单看 i18n 化的 FailureReason code（如 "TC_FAULT"）
	// 更便于排查厂商侧故障。
	FailureDetail string `json:"failureDetail,omitempty"`
}

type CreateTaskRequest struct {
	TaskName     string      `json:"taskName" binding:"required"`
	TypeCode     string      `json:"typeCode" binding:"required"`
	ProductType  string      `json:"productType"`
	FirmwareID   *uuid.UUID  `json:"firmwareId,omitempty"`
	IsKeepConfig bool        `json:"isKeepConfig"`
	DeviceIDs    []uuid.UUID `json:"deviceIds" binding:"required,min=1"`
	DeviceCount  int         `json:"deviceCount"`
	// Concurrency 任务内设备并发执行数，仅固件下载类（升级/PATCH/FPGA）生效，
	// 透传 software.BatchUpgradeRequest.Concurrency；≤0 由 software 层兜底默认值。
	Concurrency   int    `json:"concurrency"`
	ExecutionMode string `json:"executionMode" binding:"required"`
	// ScheduledAt 仅在 ExecutionMode="scheduled" 时使用，ISO 8601 字符串（前端 dayjs.toISOString()）。
	// 解析失败 / 时间已过 → 退化为 immediate（不阻断主流程，由 service 层兜底）。
	ScheduledAt string `json:"scheduledAt,omitempty"`
	Note        string `json:"note"`
	// ParamType 核心网（ims_core）任务必填：参数类型标签 FT_ImsCore_*，
	// 区分 ImsCore Parameters File 下的具体参数配置（FT1~FT12）。
	ParamType string `json:"paramType,omitempty"`
	// FileID IMS_PARAM_DISTRIBUTE 必填：参数文件库（ims_param_files）目标文件 ID。
	FileID *uuid.UUID `json:"fileId,omitempty"`
}

type TaskTypeWriteRequest struct {
	Category        string   `json:"category" binding:"required"`
	CategoryLabel   string   `json:"categoryLabel" binding:"required"`
	DisplayName     string   `json:"displayName" binding:"required"`
	Description     string   `json:"description"`
	RPCType         string   `json:"rpcType" binding:"required"`
	StepChain       []string `json:"stepChain" binding:"required,min=1"`
	PostTCEventCode string   `json:"postTcEventCode"`
	Enabled         bool     `json:"enabled"`
	PlatformScope   []string `json:"platformScope"`
	// Products「适用产品」= 产品英文名列表（#492）。前端模板编辑改为产品名多选后提交此字段；
	// 留空则沿用 PlatformScope 旧口径。
	Products               []string           `json:"products"`
	FileType               string             `json:"fileType" binding:"required"`
	FileTypeLabel          string             `json:"fileTypeLabel" binding:"required"`
	FileTypeEditable       bool               `json:"fileTypeEditable"`
	FirmwareFileType       *software.FileType `json:"firmwareFileType,omitempty"`
	URLTemplate            string             `json:"urlTemplate"`
	TargetFileNameTemplate string             `json:"targetFileNameTemplate"`
	FileNameTemplate       string             `json:"fileNameTemplate"`
	FileSizeField          string             `json:"fileSizeField"`
	ChecksumField          string             `json:"checksumField"`
	RawMode                string             `json:"rawMode"`
	DelaySeconds           int                `json:"delaySeconds"`
	TransportPath          string             `json:"transportPath"`
}

type TaskListFilter struct {
	Status   string `form:"status"`   // 任务状态：pending、in_progress、suspended、ended。
	TypeCode string `form:"typeCode"` // 任务类型，如 RUNTIME_LOG_COLLECT。
	Keyword  string `form:"keyword"`  // 匹配任务名称、类型显示名或产品类型。
	Category string `form:"category"` // 业务分类，如日志收集 station_log。
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type DeviceListFilter struct {
	Status      string `form:"status"`   // 设备执行状态；failed 用于定位失败记录。
	TypeCode    string `form:"typeCode"` // 任务类型，如 RUNTIME_LOG_COLLECT。
	Keyword     string `form:"keyword"`  // 匹配任务名、设备名、设备 SN 或产品类型。
	Category    string `form:"category"` // 业务分类，如日志收集 station_log。
	ProductType string `form:"productType"`
	// ProductName #524：设备列表筛选改按产品名（DeviceItem.ProductName 由 mapDeviceItem
	// 经 productNameLookup 回填）。前端「产品名称」下拉传此参数；与 ProductType(productClass)
	// 二选一，优先 ProductName。列表与 CSV 导出共用本过滤器，故两条路径同时生效。
	ProductName string `form:"productName"`
	// TaskID #615：按主任务 ID 精确收窄（任务详情抽屉「已选设备列表」用）。
	// 复用 /ufte/devices 端点 + matchesDeviceFilter，不新增路由；空值 ≡ 原有"全量"行为，
	// 完全向后兼容。底层来自 upgrade_sub_tasks.task_id（mapDeviceItem 已回填到 DeviceItem.TaskID）。
	TaskID   string `form:"taskId"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type DeviceCandidateFilter struct {
	Category    string `form:"category"`
	TypeCode    string `form:"typeCode"`
	ProductType string `form:"productType"`
	// ProductName #492：按产品英文名收窄候选（设备 productClass→ProductRegistry→name == 该值）。
	// 前端「产品类型」下拉改为产品名后传此参数；与 ProductType(productClass) 二选一，优先 ProductName。
	ProductName string `form:"productName"`
	Keyword     string `form:"keyword"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

func builtInTaskTypes() []TaskType {
	now := time.Now().Format(time.RFC3339)
	lte := coremodel.TechLTE
	nr := coremodel.TechNR
	gsm := coremodel.TechGSM
	enbProducts := builtInENBProductScope()
	return []TaskType{
		{
			TypeCode:               "ENB_IMG_UPGRADE",
			SortOrder:              10,
			Category:               "enb_upgrade",
			CategoryLabel:          "4G升级",
			DisplayName:            "4G 基站软件升级",
			Description:            "复用现网软件升级链路，统一承载 4G 基站镜像升级任务。",
			RPCType:                "DOWNLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_FILE_TRANSFER", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:         "CODE_ENB_UPGRADE_IMAGE",
			PlatformScope:          []string{"4G eNB", "QAFA", "QAFB"},
			Products:               enbProducts,
			FileType:               "1 Firmware Upgrade Image",
			FileTypeLabel:          "1 Firmware Upgrade Image",
			FileTypeEditable:       true,
			FirmwareFileType:       firmwareFileTypePtr(software.FileTypeIMG),
			URLTemplate:            "firmware/{minio_path}",
			TargetFileNameTemplate: "{firmware_name}",
			FileNameTemplate:       "{firmware_name}",
			FileSizeField:          "firmware.fileSize",
			ChecksumField:          "firmware.md5",
			RawMode:                "false",
			TransportPath:          "/smallcell/FileDownloadService/firmware/img/{path}",
			LastEditor:             "system",
			UpdatedAt:              now,
			softwareTaskType:       software.TaskTypeUpgrade,
			techHint:               &lte,
		},
		{
			TypeCode:               "ENB_PATCH_UPGRADE",
			SortOrder:              15,
			Category:               "enb_upgrade",
			CategoryLabel:          "4G升级",
			DisplayName:            "4G Patch 增量升级",
			Description:            "复用软件管理补丁升级任务链路，统一收口到 UFTE 任务入口。",
			RPCType:                "DOWNLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_FILE_TRANSFER", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:         "CODE_ENB_UPGRADE_PATCH",
			PlatformScope:          []string{"4G eNB", "QAFA", "QAFB", "PATCH"},
			Products:               enbProducts,
			FileType:               "X {OUI} Software Upgrade Patch",
			FileTypeLabel:          "X {OUI} Software Upgrade Patch",
			FileTypeEditable:       true,
			FirmwareFileType:       firmwareFileTypePtr(software.FileTypePATCH),
			URLTemplate:            "firmware/{patch_path}",
			TargetFileNameTemplate: "{patch_name}",
			FileNameTemplate:       "{patch_name}",
			FileSizeField:          "firmware.fileSize",
			ChecksumField:          "firmware.md5",
			RawMode:                "true",
			TransportPath:          "/smallcell/FileDownloadService/firmware/patch/{path}",
			LastEditor:             "system",
			UpdatedAt:              now,
			softwareTaskType:       software.TaskTypePatch,
			techHint:               &lte,
		},
		{
			TypeCode:               "ENB_FPGA_UPGRADE",
			SortOrder:              18,
			Category:               "enb_upgrade",
			CategoryLabel:          "4G升级",
			DisplayName:            "4G FPGA 升级",
			Description:            "复用 4G 侧 FPGA 升级任务链路，统一到 UFTE 任务中心。",
			RPCType:                "DOWNLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_FILE_TRANSFER", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:         "CODE_ENB_UPGRADE_FPGA",
			PlatformScope:          []string{"4G eNB", "QAFA", "QAFB", "FPGA"},
			Products:               enbProducts,
			FileType:               "Firmware Upgrade Fpga",
			FileTypeLabel:          "Firmware Upgrade Fpga",
			FileTypeEditable:       true,
			FirmwareFileType:       firmwareFileTypePtr(software.FileTypeFPGA),
			URLTemplate:            "firmware/{fpga_path}",
			TargetFileNameTemplate: "{fpga_name}",
			FileNameTemplate:       "{fpga_name}",
			FileSizeField:          "firmware.fileSize",
			ChecksumField:          "firmware.md5",
			RawMode:                "false",
			TransportPath:          "/smallcell/FileDownloadService/firmware/fpga/{path}",
			LastEditor:             "system",
			UpdatedAt:              now,
			softwareTaskType:       software.TaskTypeFPGA,
			techHint:               &lte,
		},
		{
			TypeCode:               "GNB_IMG_UPGRADE",
			SortOrder:              10,
			Category:               "gnb_upgrade",
			CategoryLabel:          "5G升级",
			DisplayName:            "5G 基站软件升级",
			Description:            "复用现网 5G 升级调测通过的下载与升级完成事件链路。",
			RPCType:                "DOWNLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_FILE_TRANSFER", "WAIT_TRANSFER_COMPLETE", "WAIT_INFORM_EVENT"},
			PostTCEventCode:        "102 UPGRADE FINISH",
			PermissionCode:         "CODE_GNB_UPGRADE_IMAGE",
			PlatformScope:          []string{"5G gNB", "BBU-XSS", "BBU-QSS"},
			Products:               []string{"BNQ"},
			FileType:               "1 Firmware Upgrade Image",
			FileTypeLabel:          "1 Firmware Upgrade Image",
			FileTypeEditable:       true,
			FirmwareFileType:       firmwareFileTypePtr(software.FileTypeIMG),
			URLTemplate:            "firmware/{minio_path}",
			TargetFileNameTemplate: "{firmware_name}",
			FileNameTemplate:       "{firmware_name}",
			FileSizeField:          "firmware.fileSize",
			ChecksumField:          "firmware.md5",
			RawMode:                "false",
			TransportPath:          "/smallcell/FileDownloadService/firmware/img/{path}",
			LastEditor:             "system",
			UpdatedAt:              now,
			softwareTaskType:       software.TaskTypeUpgrade,
			techHint:               &nr,
		},
		{
			TypeCode:      "GSM_IMG_UPGRADE",
			SortOrder:     19,
			Category:      "gsm_upgrade",
			CategoryLabel: "2G升级",
			DisplayName:   "2G 基站软件升级",
			// 镜像下载链路本身与制式无关，复用与 4G/5G 同一条 Download 链路。
			Description:            "复用现网软件升级链路，统一承载 2G(GSM) 基站镜像升级任务。",
			RPCType:                "DOWNLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_FILE_TRANSFER", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:         "CODE_GSM_UPGRADE_IMAGE",
			PlatformScope:          []string{"2G BSC", "2G BTS", "BSC", "BTS", "PGSM"},
			Products:               []string{"BSC", "BTS"},
			FileType:               "1 Firmware Upgrade Image",
			FileTypeLabel:          "1 Firmware Upgrade Image",
			FileTypeEditable:       true,
			FirmwareFileType:       firmwareFileTypePtr(software.FileTypeIMG),
			URLTemplate:            "firmware/{minio_path}",
			TargetFileNameTemplate: "{firmware_name}",
			FileNameTemplate:       "{firmware_name}",
			FileSizeField:          "firmware.fileSize",
			ChecksumField:          "firmware.md5",
			RawMode:                "false",
			TransportPath:          "/smallcell/FileDownloadService/firmware/img/{path}",
			LastEditor:             "system",
			UpdatedAt:              now,
			softwareTaskType:       software.TaskTypeUpgrade,
			techHint:               &gsm,
		},
		{
			TypeCode:               "UPS_AP_UPGRADE",
			SortOrder:              25,
			Category:               "ups_upgrade",
			CategoryLabel:          "UPS升级",
			DisplayName:            "UPS 软件升级",
			Description:            "复用 TR-069 Download + TransferComplete + 1 BOOT 版本确认链路，统一承载 UPS 固件升级任务。",
			RPCType:                "DOWNLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_FILE_TRANSFER", "WAIT_TRANSFER_COMPLETE", "WAIT_REBOOT_COMPLETE"},
			PermissionCode:         "CODE_UPS_UPGRADE_IMAGE",
			PlatformScope:          []string{"UPS"},
			Products:               []string{"UPS"},
			FileType:               "1 Firmware Upgrade Image",
			FileTypeLabel:          "1 Firmware Upgrade Image",
			FileTypeEditable:       true,
			FirmwareFileType:       firmwareFileTypePtr(software.FileTypeIMG),
			URLTemplate:            "firmware/{minio_path}",
			TargetFileNameTemplate: "{firmware_name}",
			FileNameTemplate:       "{firmware_name}",
			FileSizeField:          "firmware.fileSize",
			ChecksumField:          "firmware.md5",
			RawMode:                "false",
			TransportPath:          "/smallcell/FileDownloadService/firmware/img/{path}",
			LastEditor:             "system",
			UpdatedAt:              now,
			softwareTaskType:       software.TaskTypeUpgrade,
		},
		{
			TypeCode:         "VERSION_ROLLBACK",
			SortOrder:        20,
			Category:         "version_rollback",
			CategoryLabel:    "基站版本回退",
			DisplayName:      "基站版本回退",
			Description:      "通过 TR069 SetParameterValues 触发设备回退到上一版本，不需要下载文件。",
			RPCType:          "SET_PARAM_VALUES",
			BuiltIn:          true,
			Enabled:          true,
			StepChain:        []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_REBOOT_COMPLETE"},
			PermissionCode:   "CODE_VERSION_ROLLBACK",
			PlatformScope:    []string{"4G eNB", "5G gNB", "QAFA", "QAFB", "BBU-XSS", "BBU-QSS"},
			FileType:         "",
			FileTypeLabel:    "版本回退",
			FileTypeEditable: false,
			LastEditor:       "system",
			UpdatedAt:        now,
			softwareTaskType: software.TaskTypeRollback,
		},
		{
			TypeCode:               "RUNTIME_LOG_COLLECT",
			SortOrder:              30,
			Category:               "station_log",
			CategoryLabel:          "日志收集",
			DisplayName:            "运行日志采集",
			Description:            "复用现网日志采集 Upload 链路，提供 UFTE 内置的运行日志收集模板。",
			RPCType:                "UPLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:         "CODE_RUNTIME_LOG_COLLECT",
			PlatformScope:          []string{"4G eNB", "5G gNB", "DXDF", "BBU-QSS"},
			FileType:               "4 Vendor Log File 1,2,3,4",
			FileTypeLabel:          "4 Vendor Log File 1,2,3,4",
			FileTypeEditable:       true,
			TargetFileNameTemplate: "runtime-{task_id8}-{sn}.tar.gz",
			FileNameTemplate:       "runtime-{task_id8}-{sn}.tar.gz",
			// 厂商 baicells/MMMM 真实样本对齐（migrations/000141）：
			// · fileType=LOG（字面值，非项目自定义编号）
			// · 无 sn 参数（厂商样本只有 fileType+taskId+filename）
			// · taskId 用 32 字符纯 hex 无连字符 → 模板用 {taskId32} 占位符
			// · filename 留空让设备自己决定上传名（同 NV/XML）
			TransportPath:    "/smallcell/FileUploadService?fileType=LOG&sn={sn}&taskId={taskId32}&filename=",
			LastEditor:       "system",
			UpdatedAt:        now,
			softwareTaskType: software.TaskTypeLogCollect,
		},
		{
			TypeCode:      "FAULT_LOG_COLLECT",
			SortOrder:     35,
			Category:      "station_log",
			CategoryLabel: "日志收集",
			DisplayName:   "故障日志采集",
			Description:   "通过 SetParameterValues 写设备私有参数 Device.DeviceInfo.FaultLogURL 触发 CPE 主动上传故障日志，落 MinIO 后由 backup.file.received 推进任务完成。",
			// SPV 触发的"反向上传"：厂商私有 ACS 协议不实现 Upload RPC，而是把 ACS 端
			// FileUploadService 的完整 URL 写到设备私有参数 FaultLogURL，CPE 拿到 URL
			// 后异步 HTTP PUT 日志到该 URL。
			RPCType:        "SET_PARAM_VALUES",
			BuiltIn:        true,
			Enabled:        true,
			StepChain:      []string{"CHECK_PERMISSION", "CHECK_ONLINE", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_FILE_UPLOAD"},
			PermissionCode: "CODE_FAULT_LOG_COLLECT",
			PlatformScope:  []string{"4G eNB", "5G gNB", "DXDF", "BBU-QSS"},
			// FileType 仍保留 "8"，用于 transfer_router.ForUploadFileType 路由到
			// fault_log_collect_tasks / fault_log_collect_sub_tasks 物理表。
			FileType:         "8",
			FileTypeLabel:    "故障日志",
			FileTypeEditable: true,
			// URLTemplate 复用为 SPV 参数路径。BatchCollect 把它透传给 executor.ExecuteOneSetParamCollect
			// 作为 SetParameterValues 的 paramName。路径来源：migrations/seed/000126
			// standard_params 字典（STANDARD 类型 READ_WRITE string）。最初按用户口述写成
			// "X_COM_Log.FaultLogURL"，CPE 回 "Empty parameter list" 因为该路径在 CPE 数据
			// 模型里不存在，已对齐字典。
			URLTemplate:            "Device.DeviceInfo.FaultLogURL",
			TargetFileNameTemplate: "fault-{sn}-{timestamp}.tar.gz",
			FileNameTemplate:       "fault-{sn}-{timestamp}.tar.gz",
			// TransportPath 是 SPV 下发给 CPE 的 URL 路径模板。{id} = 主任务 UUID（与
			// ACS upload handler 发布事件时写入的 task_id 同源）；{sn} = 设备 SN；
			// {fileName} 留空让 CPE 自行决定上传名。ACS 落 MinIO 前会二次命名为
			// fault-{sn}-{yyyyMMddHHmmssSSS}.tar.gz，避免额外 taskId8 目录。
			TransportPath:    "/smallcell/FileUploadService?fileType=RL&id={id}&sn={sn}&fileName=",
			LastEditor:       "system",
			UpdatedAt:        now,
			softwareTaskType: software.TaskTypeLogCollect,
		},
		{
			TypeCode:         "CONFIG_BACKUP_XML",
			SortOrder:        40,
			Category:         "config_backup",
			CategoryLabel:    "配置文件备份",
			DisplayName:      "配置文件备份（XML）",
			Description:      "标准平台（BLQ/QLS 等）配置文件备份，TR-069 Upload FileType=10 {OUI} Configuration File。",
			RPCType:          "UPLOAD",
			BuiltIn:          true,
			Enabled:          true,
			StepChain:        []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:   "CODE_CONFIG_BACKUP",
			PlatformScope:    []string{"4G eNB", "5G gNB", "QAFA", "QAFB", "BBU-XSS", "BBU-QSS"},
			FileType:         "10 {OUI} Configuration File",
			FileTypeLabel:    "10 {OUI} Configuration File",
			FileTypeEditable: false,
			// issue #585: 配置文件备份对象名规范化为 {sn}_CFG.xml。
			// 真实文件名由 ACS upload handler 在收报文时强制覆写（设备
			// 在 TR-069 Upload RPC 中无法选定上传文件名），这里 Template
			// 仅作为 UI 展示与 executor 兜底拼接，保持与 ACS 一致。
			TargetFileNameTemplate: "{sn}_CFG.xml",
			FileNameTemplate:       "{sn}_CFG.xml",
			// URL `filename=` 留空：详见 RUNTIME_LOG_COLLECT 同名说明。
			TransportPath:    "/smallcell/FileUploadService?fileType=CONFIGBACKUP_XML&sn={sn}&taskId={taskId}&filename=",
			LastEditor:       "system",
			UpdatedAt:        now,
			softwareTaskType: software.TaskTypeLogCollect,
		},
		{
			TypeCode:         "CONFIG_BACKUP_NV",
			SortOrder:        45,
			Category:         "config_backup",
			CategoryLabel:    "配置文件备份",
			DisplayName:      "配置文件备份（NV）",
			Description:      "NV 平台（MLQ/MLN/BM 等）配置文件备份，TR-069 Upload FileType=12 {OUI} Configuration File。",
			RPCType:          "UPLOAD",
			BuiltIn:          true,
			Enabled:          true,
			StepChain:        []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:   "CODE_CONFIG_BACKUP",
			PlatformScope:    []string{"MLQ", "MLN", "BM"},
			FileType:         "12 {OUI} Configuration File",
			FileTypeLabel:    "12 {OUI} Configuration File",
			FileTypeEditable: false,
			// issue #585: 同 CONFIG_BACKUP_XML，对象名规范化为 {sn}_CFG.nv。
			TargetFileNameTemplate: "{sn}_CFG.nv",
			FileNameTemplate:       "{sn}_CFG.nv",
			// URL `filename=` 留空：详见 RUNTIME_LOG_COLLECT 同名说明。
			TransportPath:    "/smallcell/FileUploadService?fileType=CONFIGBACKUP_NV&sn={sn}&taskId={taskId}&filename=",
			LastEditor:       "system",
			UpdatedAt:        now,
			softwareTaskType: software.TaskTypeLogCollect,
		},
		{
			TypeCode:               "CONFIG_RESTORE",
			SortOrder:              50,
			Category:               "config_restore",
			CategoryLabel:          "配置文件恢复",
			DisplayName:            "配置文件恢复",
			Description:            "复用现网配置恢复 Download 链路，提供 UFTE 内置的配置文件恢复模板。",
			RPCType:                "DOWNLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:         "CODE_CONFIG_RESTORE",
			PlatformScope:          []string{"4G eNB", "5G gNB", "QAFA", "QAFB", "BBU-XSS", "BBU-QSS"},
			FileType:               "10 <OUI> Configuration File",
			FileTypeLabel:          "10 <OUI> Configuration File",
			FileTypeEditable:       false,
			URLTemplate:            "config-backup/{object_path}",
			TargetFileNameTemplate: "{file_name}",
			FileNameTemplate:       "{file_name}",
			TransportPath:          "/smallcell/FileDownloadService/config-backup/{object_path}",
			LastEditor:             "system",
			UpdatedAt:              now,
			// 走 LogCollect 桶以便复用 ufte.loadAllTasks 的 typeSet 过滤，
			// 让 CONFIG_RESTORE 占位 upgrade_tasks 行在任务列表里可见。
			// 早 return 路径（createConfigRestoreTask）保证不会被 BatchCollect 误派发。
			softwareTaskType: software.TaskTypeLogCollect,
		},
		{
			TypeCode:               "LICENSE_UPGRADE",
			SortOrder:              55,
			Category:               "license_upgrade",
			CategoryLabel:          "设备License升级",
			DisplayName:            "设备License升级",
			Description:            "从 license 库取目标设备最新 license，通过 TR-069 Download RPC 下发。",
			RPCType:                "DOWNLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:         "CODE_LICENSE_UPGRADE",
			PlatformScope:          []string{"4G eNB", "5G gNB", "QAFA", "QAFB", "BBU-XSS", "BBU-QSS"},
			FileType:               "License File",
			FileTypeLabel:          "License File",
			FileTypeEditable:       false,
			URLTemplate:            "device-licenses/{object_path}",
			TargetFileNameTemplate: "{file_name}",
			FileNameTemplate:       "{file_name}",
			TransportPath:          "/smallcell/FileDownloadService/device-licenses/{object_path}",
			LastEditor:             "system",
			UpdatedAt:              now,
			softwareTaskType:       software.TaskTypeLogCollect, // 同上
		},
		{
			// 核心网文件采集（imscore_filetask.txt Upload 段，设备 → OMC）。
			// CWMP FileType 统一 "ImsCore Parameters File"（核心网侧要求所有文件传输
			// 共用此字面值；方向由 Upload RPC 表达），具体类型由报文 <ParameterType>
			// 标签承载（FT_ImsCore_*），在 CreateTask 时预渲染进 TransportPath 的
			// {paramType} 占位符；任务行 download_file_type 落
			// "ImsCore Parameters File:FT_ImsCore_*"（反查键 + 文件类型，见 resolveTaskType）。
			TypeCode:         "IMS_FILE_COLLECT",
			SortOrder:        60,
			Category:         "ims_core",
			CategoryLabel:    "核心网",
			DisplayName:      "核心网文件采集",
			Description:      "核心网（IMS Core）文件采集（Upload 段 17 种文件类型），FileType 统一为 ImsCore Parameters File，文件类型（ParameterType）区分。",
			RPCType:          "UPLOAD",
			BuiltIn:          true,
			Enabled:          true,
			StepChain:        []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:   "CODE_IMS_FILE_COLLECT",
			PlatformScope:    []string{},
			Products:         []string{},
			FileType:         imsparam.CWMPFileTypeImsParam,
			FileTypeLabel:    "ImsCore Parameters File",
			FileTypeEditable: false,
			// 目标文件名由 executor 固定生成（类型去 FT_ImsCore_ 前缀 + 年月日时分秒，
			// 日志类 .log / 其余 .dat），模板值仅作 UI 展示。
			TargetFileNameTemplate: "{paramTypeShort}_{yyyyMMddHHmmss}.dat",
			FileNameTemplate:       "{paramTypeShort}_{yyyyMMddHHmmss}.dat",
			TransportPath:          "/smallcell/FileUploadService?fileType=" + imsparam.UploadQueryFileTypeAlias + "&paramType={paramType}&sn={sn}&taskId={taskId}&filename=",
			LastEditor:             "system",
			UpdatedAt:              now,
			softwareTaskType:       software.TaskTypeLogCollect,
		},
		{
			// 核心网文件下发（imscore_filetask.txt Download 段，OMC → 设备）。
			// 与 LICENSE_UPGRADE / CONFIG_RESTORE 同款"占位 + 直接派发 device_tasks"链路；
			// catalog FileType "IMS_FILE_DISTRIBUTE" 仅作任务行反查键（真实 CWMP
			// FileType 由 imsparam.Service 派发时统一写死为 ImsCore Parameters File），任务行
			// download_file_type 落 "IMS_FILE_DISTRIBUTE:FT_ImsCore_*"，
			// file_name 落所选文件 UUID（挂起/定时重派发用）。
			TypeCode:               "IMS_FILE_DISTRIBUTE",
			SortOrder:              65,
			Category:               "ims_core",
			CategoryLabel:          "核心网",
			DisplayName:            "核心网文件下发",
			Description:            "从文件库选取指定文件类型（Download 段 9 种）的文件，通过 TR-069 Download RPC 下发到核心网设备。",
			RPCType:                "DOWNLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:         "CODE_IMS_FILE_DISTRIBUTE",
			PlatformScope:          []string{},
			Products:               []string{},
			FileType:               "IMS_FILE_DISTRIBUTE",
			FileTypeLabel:          "ImsCore Parameters File",
			FileTypeEditable:       false,
			TargetFileNameTemplate: "{file_name}",
			FileNameTemplate:       "{file_name}",
			TransportPath:          "/smallcell/FileDownloadService/ims-params/{object_path}",
			LastEditor:             "system",
			UpdatedAt:              now,
			softwareTaskType:       software.TaskTypeLogCollect,
		},
	}
}

func builtInENBProductScope() []string {
	return []string{
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
}

func findTaskTypeByCode(catalog []TaskType, typeCode string) (TaskType, bool) {
	for _, item := range catalog {
		if item.TypeCode == typeCode {
			return item, true
		}
	}
	return TaskType{}, false
}

// BuiltInTaskType returns the built-in task type definition for the given type code.
// Callers (e.g. backup.BackupExecutor) use this to read fields such as FileType
// from the canonical template definition, avoiding duplication.
// Returns nil, false if the type code is not found in the built-in catalog.
func BuiltInTaskType(typeCode string) (*TaskType, bool) {
	tt, ok := findTaskTypeByCode(builtInTaskTypes(), typeCode)
	if !ok {
		return nil, false
	}
	return &tt, true
}

func resolveTaskType(catalog []TaskType, taskType software.TaskType, productClass string, fileType string) (TaskType, bool) {
	// When multiple catalog entries share the same softwareTaskType (e.g.
	// RUNTIME_LOG_COLLECT, CONFIG_BACKUP_XML, CONFIG_BACKUP_NV, CONFIG_RESTORE
	// all use TaskTypeLogCollect=10), disambiguate by matching the FileType
	// stored on the upgrade_task row against each catalog entry's FileType.
	//
	// IMS 任务（ims_core）的落库值带 ParamType 尾段（"ImsCore Parameters File:FT_ImsCore_*" /
	// "IMS_PARAM_DISTRIBUTE:FT_ImsCore_*"）：先精确匹配，未命中且串含 ':' 时去掉最后一段
	// 再匹配一次（catalog 字面值 "ImsCore Parameters File" / "IMS_PARAM_DISTRIBUTE"）。
	if item, ok := matchTaskTypeByFileType(catalog, taskType, fileType); ok {
		return item, true
	}
	if idx := strings.LastIndex(fileType, ":"); idx > 0 {
		if item, ok := matchTaskTypeByFileType(catalog, taskType, fileType[:idx]); ok {
			return item, true
		}
	}
	var fallback TaskType
	foundFallback := false
	for _, item := range catalog {
		if item.softwareTaskType != taskType {
			continue
		}
		if !foundFallback {
			if item.techHint == nil {
				fallback = item
				foundFallback = true
			} else if matchesTaskTypeScope(item, productClass) {
				fallback = item
				foundFallback = true
			}
		}
	}
	if foundFallback {
		return fallback, true
	}
	if taskType == software.TaskTypeUpgrade {
		return findTaskTypeByCode(catalog, "ENB_IMG_UPGRADE")
	}
	return TaskType{}, false
}

// matchTaskTypeByFileType 精确匹配 fileType == item.FileType 的 catalog 条目。
func matchTaskTypeByFileType(catalog []TaskType, taskType software.TaskType, fileType string) (TaskType, bool) {
	if fileType == "" {
		return TaskType{}, false
	}
	for _, item := range catalog {
		if item.softwareTaskType != taskType {
			continue
		}
		if item.FileType != "" && fileType == item.FileType {
			return item, true
		}
	}
	return TaskType{}, false
}

// imsParamTypeFromStoredFileType 从任务行 download_file_type 落库值解析 ParamType
// 尾段（"ImsCore Parameters File:FT_ImsCore_Policy_Setting_UD" → "FT_ImsCore_Policy_Setting_UD"）。
// 尾段不是合法 FT_ImsCore_* 时返回空串。
func imsParamTypeFromStoredFileType(fileType string) string {
	idx := strings.LastIndex(fileType, ":")
	if idx < 0 || idx == len(fileType)-1 {
		return ""
	}
	tail := fileType[idx+1:]
	if _, ok := imsparam.Lookup(tail); !ok {
		return ""
	}
	return imsparam.NormalizeParamType(tail)
}

// renderImsParamPlaceholders 把模板里的 {paramType} 占位符替换为运行时 ParamType。
// 在 CreateTask / Resume 时预渲染，executor 零改动。
func renderImsParamPlaceholders(tmpl, paramType string) string {
	if tmpl == "" || paramType == "" {
		return tmpl
	}
	return strings.ReplaceAll(tmpl, "{paramType}", paramType)
}

func matchesTaskTypeScope(item TaskType, productClass string) bool {
	if productClass == "" {
		return item.techHint == nil
	}
	upper := strings.ToUpper(productClass)
	for _, scope := range item.PlatformScope {
		scopeUpper := strings.ToUpper(scope)
		if upper == scopeUpper || strings.Contains(upper, scopeUpper) || strings.Contains(scopeUpper, upper) {
			return true
		}
	}
	if item.techHint == nil {
		return true
	}
	switch *item.techHint {
	case coremodel.TechNR:
		// 关键字模糊匹配只是兜底；真正的精确识别由 service.resolveTaskTypeForTask
		// 用 ProductRegistry 查 product.tech 完成。FAP/BSC7041C243 这种 5G 产品
		// "FAP" 子串也命中 LTE 白名单（line 571），所以 read 路径过去一直把它错归
		// 4G —— 修复 commit 把 BSC 也加进 NR 兜底，并让 mapTask 走 ProductRegistry
		// 优先识别（model.go 关键字只在 registry 不可用时兜底）。
		return strings.Contains(upper, "5G") || strings.Contains(upper, "GNB") || strings.Contains(upper, "QSS") || strings.Contains(upper, "XSS") || strings.Contains(upper, "BBU") || strings.Contains(upper, "BSC")
	case coremodel.TechLTE:
		return strings.Contains(upper, "4G") || strings.Contains(upper, "ENB") || strings.Contains(upper, "QAFA") || strings.Contains(upper, "QAFB") || strings.Contains(upper, "FAP") || strings.Contains(upper, "BM") || strings.Contains(upper, "BNQ") || strings.Contains(upper, "MLQ") || strings.Contains(upper, "MLN") || strings.Contains(upper, "BLQ")
	case coremodel.TechGSM:
		// 2G/GSM 兜底关键字白名单（productTechLookup 不可用 / 未注册时生效）。
		// products.xml：BSC=^FAP/PGSM$、BTS=^FAP/BTS$，均 tech="2G" deviceType="gsm"。
		// 注意："FAP" 子串也命中 LTE 白名单，故 2G 设备的精确识别仍以 service.
		// deviceMatchesTaskType 经 productTechLookup 查 product.tech=="gsm" 为准，
		// 本兜底只在 registry 不可用时放行 PGSM/BTS/BSC/GSM/2G 关键字。
		return strings.Contains(upper, "2G") || strings.Contains(upper, "GSM") || strings.Contains(upper, "PGSM") || strings.Contains(upper, "BTS") || strings.Contains(upper, "BSC")
	default:
		return false
	}
}

func supportedSoftwareTaskTypes() []software.TaskType {
	return []software.TaskType{
		software.TaskTypeUpgrade,
		software.TaskTypePatch,
		software.TaskTypeFPGA,
		software.TaskTypeRollback,
	}
}

func normalizeTaskResult(result software.TaskResult) string {
	switch result {
	case software.TaskResultFailed:
		return "failure"
	default:
		return string(result)
	}
}

// normalizeDeviceStatus 把 software.UpgradeSubTask.Status 翻译成设备列表展示状态。
//
// 状态本身就是任务类型语义：
//   - UpgradeDownloading（普通升级 Download RPC）→ "downloading"
//   - UpgradeUploading（Upload RPC，备份 / 日志采集）→ "uploading"
//     · 子分支：fileLanded=true（backup_restore_file 已落地）→ "awaiting_tc"
//     —— TR-069 上"等 UploadResponse"和"CPE PUT 文件中"两步紧贴且无独立 ACS 信号，
//     合并到"上传中"；CPE 完成 HTTP PUT → ACS 写 backup_restore_file 是唯一可观测分界点，
//     之后等 CPE 主动发 TransferComplete 是独立的一段，单独展示。
//   - 其它状态按状态机直译。
func normalizeDeviceStatus(status software.UpgradeState, fileLanded bool) string {
	switch status {
	case software.UpgradeDownloading:
		return "downloading"
	case software.UpgradeUploading:
		if fileLanded {
			return "awaiting_tc"
		}
		return "uploading"
	case software.UpgradeVerifying, software.UpgradeRebooting:
		return "verifying"
	case software.UpgradeSuspended:
		return "suspended"
	case software.UpgradeCompleted:
		return "ended"
	case software.UpgradeFailed, software.UpgradeTerminated:
		return "failed"
	default:
		return "pending"
	}
}

func normalizeDeviceStatusForTask(taskType software.TaskType, status software.UpgradeState, fileLanded bool) string {
	if taskType == software.TaskTypeRollback {
		switch status {
		case software.UpgradeDownloading:
			return "rollback_checking"
		case software.UpgradeRebooting, software.UpgradeVerifying:
			return "rolling_back"
		}
	}
	return normalizeDeviceStatus(status, fileLanded)
}

const rollbackEnableCheckCommandKeyPrefix = "rollback-enable-check-"

func normalizeFailureReasonForTask(taskType software.TaskType, commandKey, failureReason, failureDetail string) string {
	if taskType != software.TaskTypeRollback || failureReason == "" {
		return failureReason
	}

	detailLower := strings.ToLower(failureDetail)
	switch failureReason {
	case string(software.FailureDownloadTimeout):
		if strings.HasPrefix(commandKey, rollbackEnableCheckCommandKeyPrefix) {
			return string(software.FailureRollbackEnableCheckTimeout)
		}
	case string(software.FailureTaskTimeout):
		return string(software.FailureRollbackApplyTimeout)
	case string(software.FailureUploadFault):
		if strings.Contains(detailLower, "rollback") && strings.Contains(detailLower, "setparametervalues") {
			return string(software.FailureRollbackSetFault)
		}
	case string(software.FailureInternalError):
		if strings.Contains(detailLower, "enable check rejected") {
			return string(software.FailureRollbackEnableCheckFault)
		}
		if strings.Contains(detailLower, "does not support rollback") {
			return string(software.FailureRollbackNotSupported)
		}
	}
	return failureReason
}

func normalizeFailureDetailForTask(taskType software.TaskType, failureReason, failureDetail string) string {
	if taskType != software.TaskTypeRollback || failureDetail == "" {
		return failureDetail
	}

	detailLower := strings.ToLower(failureDetail)
	switch failureReason {
	case string(software.FailureRollbackEnableCheckTimeout):
		if strings.Contains(detailLower, "downloadresponse") {
			return "Rollback enable check timed out: no GetParameterValuesResponse from device."
		}
	case string(software.FailureRollbackApplyTimeout):
		if strings.Contains(detailLower, "transfercomplete") {
			return "Rollback timed out: no reboot completion from device after SetParameterValues."
		}
	}
	return failureDetail
}

func progressFromCounts(total, success, failed int, status software.TaskStatus) int {
	if total <= 0 {
		if status == software.TaskEnded {
			return 100
		}
		return 0
	}
	done := success + failed
	if done < 0 {
		done = 0
	}
	if done > total {
		done = total
	}
	return int(float64(done) / float64(total) * 100)
}

func stepForTask(item TaskType, status software.TaskStatus) string {
	// 备份 / 日志采集 / 配置恢复 等"OUTPUT 文件"类型走 LogCollect 编排，跨多设备
	// 没有"主任务当前步骤"的概念——设备级 RPC 步骤应该出现在设备子任务列表，
	// 不该塞到主任务上（详见 docs/project/backup-display-fix-20260520.md）。
	if item.softwareTaskType == software.TaskTypeLogCollect {
		return ""
	}
	if status == software.TaskPending || status == software.TaskSuspended {
		return "SEND_RPC"
	}
	if item.softwareTaskType == software.TaskTypeRollback {
		return "WAIT_REBOOT_COMPLETE"
	}
	if item.softwareTaskType == software.TaskTypeUpgrade && taskTypeProductsContain(item.Products, "UPS") {
		return "WAIT_REBOOT_COMPLETE"
	}
	if item.PostTCEventCode != "" {
		return "WAIT_INFORM_EVENT"
	}
	return "WAIT_TRANSFER_COMPLETE"
}

func progressForDeviceStatus(status string) int {
	switch status {
	case "downloading", "uploading", "rollback_checking":
		return 45
	case "awaiting_tc":
		return 70
	case "verifying", "rolling_back":
		return 75
	case "ended", "failed":
		return 100
	default:
		return 0
	}
}

// renderUFTEFileNameTemplate replaces {task_id8} 与 {sn} 占位符为运行时实际值。
// 运行日志、配置备份等同步模板可用它预览文件名；FAULT_LOG_COLLECT 完成后改从
// backup_restore_file 按 (sn, task_id) 反查 ACS 二次命名的真实落地文件名。
func renderUFTEFileNameTemplate(tmpl string, taskID uuid.UUID, sn string) string {
	if tmpl == "" {
		return ""
	}
	hex := strings.ReplaceAll(taskID.String(), "-", "")
	if len(hex) > 8 {
		hex = hex[:8]
	}
	out := strings.ReplaceAll(tmpl, "{task_id8}", hex)
	out = strings.ReplaceAll(out, "{sn}", sn)
	return out
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func defaultDeviceName(siteName, serialNumber, productClass string) string {
	if strings.TrimSpace(siteName) != "" {
		return siteName
	}
	return "-"
}

func materializeTaskTypes(stored []TaskType) []TaskType {
	defaults := builtInTaskTypes()
	defaultByCode := make(map[string]TaskType, len(defaults))
	for _, item := range defaults {
		defaultByCode[item.TypeCode] = item
	}

	result := make([]TaskType, 0, len(stored))
	for _, item := range stored {
		item.StepChain = normalizeStringSlice(item.StepChain)
		item.PlatformScope = normalizeStringSlice(item.PlatformScope)
		item.Products = normalizeStringSlice(item.Products)
		item.URLTemplate = appconfig.NormalizeConfigBackupReference(item.URLTemplate)
		item.TransportPath = appconfig.NormalizeConfigBackupReference(item.TransportPath)
		item.FirmwareFileType = normalizeTaskTypeFirmwareFileType(item.RPCType, item.FirmwareFileType, item.FileType)
		if base, ok := defaultByCode[item.TypeCode]; ok {
			item.BuiltIn = item.BuiltIn || base.BuiltIn
			item.softwareTaskType = base.softwareTaskType
			item.techHint = base.techHint
			item.RPCType = base.RPCType
			item.StepChain = base.StepChain
			if item.FirmwareFileType == nil && base.FirmwareFileType != nil {
				item.FirmwareFileType = firmwareFileTypePtr(*base.FirmwareFileType)
			}
			// 内置模板且 FileType 不可编辑（固定反查键：IMS_FILE_* / CONFIG_* /
			// LICENSE_UPGRADE / VERSION_ROLLBACK 等）→ 强制跟随代码定义，避免代码
			// 升级后 DB 里残留的旧字面值（如 "ImsCore_Parameters_Type"）导致
			// resolveTaskType 反查 miss、任务错归模板。升级类（fileTypeEditable=true）
			// 保留用户编辑值。
			if !base.FileTypeEditable {
				item.FileType = base.FileType
				item.FileTypeLabel = base.FileTypeLabel
			}
			// 核心网（ims_core）内置模板的 TransportPath 同样强制跟随代码定义：
			// 内置模板的传输路径由代码统一定义（模板 UI 不暴露 transport_path 编辑），
			// 且含 fileType URL 别名等与报文契约耦合的细节——EnsureBuiltInTaskTypes
			// 只插入不更新，不同步会让代码升级后新建任务仍用旧别名（如 IMS_FILE）。
			if base.Category == "ims_core" {
				item.TransportPath = base.TransportPath
				item.URLTemplate = base.URLTemplate
			}
		}
		result = append(result, item)
	}
	return result
}

func firmwareFileTypePtr(value software.FileType) *software.FileType {
	copyValue := value
	return &copyValue
}

func taskTypeFilterKeys(catalog []TaskType, filter TaskListFilter) (map[software.TaskType]struct{}, error) {
	return filterTaskTypeSet(catalog, filter.Category, filter.TypeCode)
}

func deviceTypeFilterKeys(catalog []TaskType, category, typeCode string) (map[software.TaskType]struct{}, error) {
	return filterTaskTypeSet(catalog, category, typeCode)
}

func filterTaskTypeSet(catalog []TaskType, category, typeCode string) (map[software.TaskType]struct{}, error) {
	if typeCode != "" {
		item, ok := findTaskTypeByCode(catalog, typeCode)
		if !ok {
			return nil, fmt.Errorf("unsupported type_code: %s", typeCode)
		}
		if item.softwareTaskType == 0 {
			return map[software.TaskType]struct{}{}, nil
		}
		return map[software.TaskType]struct{}{item.softwareTaskType: {}}, nil
	}
	set := make(map[software.TaskType]struct{})
	for _, item := range catalog {
		if !categoryMatchesFilter(category, item.Category) {
			continue
		}
		if item.softwareTaskType == 0 {
			continue
		}
		set[item.softwareTaskType] = struct{}{}
	}
	return set, nil
}

func categoryMatchesFilter(filterCategory, itemCategory string) bool {
	if filterCategory == "" {
		return true
	}
	if filterCategory == deviceUpgradeVirtualCategory {
		return isDeviceUpgradeCategoryMember(itemCategory)
	}
	return itemCategory == filterCategory
}

func isDeviceUpgradeCategoryMember(category string) bool {
	switch category {
	case "enb_upgrade", "gnb_upgrade", "gsm_upgrade", "ups_upgrade", deviceUpgradeVirtualCategory:
		return true
	default:
		return false
	}
}

func techForCandidateFilter(catalog []TaskType, category string) *coremodel.Technology {
	for _, item := range catalog {
		if item.Category != category || item.techHint == nil {
			continue
		}
		return item.techHint
	}
	return nil
}

func categoryLabelForCandidate(catalog []TaskType, category string) string {
	if category == deviceUpgradeVirtualCategory {
		return "设备升级"
	}
	for _, item := range catalog {
		if item.Category == category {
			return item.CategoryLabel
		}
	}
	return category
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return []string{}
	}
	return result
}

func generateTaskTypeCodeBase(category, displayName string) string {
	parts := []string{normalizeIdentifier(category), normalizeIdentifier(displayName)}
	base := strings.Join(parts, "_")
	base = strings.Trim(base, "_")
	if base == "" {
		return "CUSTOM_TASK_TYPE"
	}
	if !strings.HasPrefix(base, "CUSTOM_") {
		base = "CUSTOM_" + base
	}
	return base
}

func normalizeIdentifier(raw string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range raw {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToUpper(r))
			lastUnderscore = false
		case !lastUnderscore:
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

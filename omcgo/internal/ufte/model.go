package ufte

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
)

type Overview struct {
	EnabledTypeCount int     `json:"enabledTypeCount"`
	RunningTaskCount int     `json:"runningTaskCount"`
	CustomTypeCount  int     `json:"customTypeCount"`
	SuccessRate30d   float64 `json:"successRate30d"`
}

type TaskType struct {
	TypeCode               string   `json:"typeCode"`
	Category               string   `json:"category"`
	CategoryLabel          string   `json:"categoryLabel"`
	DisplayName            string   `json:"displayName"`
	Description            string   `json:"description"`
	RPCType                string   `json:"rpcType"`
	BuiltIn                bool     `json:"builtIn"`
	Enabled                bool     `json:"enabled"`
	StepChain              []string `json:"stepChain"`
	PostTCEventCode        string   `json:"postTcEventCode,omitempty"`
	PermissionCode         string   `json:"permissionCode"`
	PlatformScope          []string `json:"platformScope"`
	FileType               string   `json:"fileType"`
	FileTypeLabel          string   `json:"fileTypeLabel"`
	FileTypeEditable       bool     `json:"fileTypeEditable"`
	URLTemplate            string   `json:"urlTemplate,omitempty"`
	TargetFileNameTemplate string   `json:"targetFileNameTemplate,omitempty"`
	FileNameTemplate       string   `json:"fileNameTemplate,omitempty"`
	FileSizeField          string   `json:"fileSizeField,omitempty"`
	ChecksumField          string   `json:"checksumField,omitempty"`
	RawMode                string   `json:"rawMode,omitempty"`
	DelaySeconds           int      `json:"delaySeconds,omitempty"`
	TransportPath          string   `json:"transportPath,omitempty"`
	LastEditor             string   `json:"lastEditor"`
	TaskCount30d           int      `json:"taskCount30d"`
	SuccessRate30d         float64  `json:"successRate30d"`
	UpdatedAt              string   `json:"updatedAt"`

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
	FirmwareID      string `json:"firmwareId,omitempty"`
	TargetVersion   string `json:"targetVersion,omitempty"`
	ProductType     string `json:"productType,omitempty"`
	IsKeepConfig    bool   `json:"isKeepConfig,omitempty"`
	Status          string `json:"status"`
	Result          string `json:"result,omitempty"`
	Progress        int    `json:"progress"`
	TotalCount      int    `json:"totalCount"`
	SuccessCount    int    `json:"successCount"`
	FailCount       int    `json:"failCount"`
	CurrentStep     string `json:"currentStep"`
	ExecutionMode   string `json:"executionMode"`
	CreateUser      string `json:"createUser"`
	CreatedAt       string `json:"createdAt"`
	ScheduledAt     string `json:"scheduledAt,omitempty"`
	OperatorScope   string `json:"operatorScope"`
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
	CurrentVersion  string `json:"currentVersion"`
	TargetVersion   string `json:"targetVersion"`
	Status          string `json:"status"`
	Result          string `json:"result,omitempty"`
	Progress        int    `json:"progress"`
	LastReportAt    string `json:"lastReportAt"`
	OperatorScope   string `json:"operatorScope"`
	FailureReason   string `json:"failureReason,omitempty"`
}

type CreateTaskRequest struct {
	TaskName      string      `json:"taskName" binding:"required"`
	TypeCode      string      `json:"typeCode" binding:"required"`
	ProductType   string      `json:"productType"`
	FirmwareID    *uuid.UUID  `json:"firmwareId,omitempty"`
	IsKeepConfig  bool        `json:"isKeepConfig"`
	DeviceIDs     []uuid.UUID `json:"deviceIds" binding:"required,min=1"`
	DeviceCount   int         `json:"deviceCount"`
	ExecutionMode string      `json:"executionMode" binding:"required"`
	Note          string      `json:"note"`
}

type TaskTypeWriteRequest struct {
	Category               string   `json:"category" binding:"required"`
	CategoryLabel          string   `json:"categoryLabel" binding:"required"`
	DisplayName            string   `json:"displayName" binding:"required"`
	Description            string   `json:"description"`
	RPCType                string   `json:"rpcType" binding:"required"`
	StepChain              []string `json:"stepChain" binding:"required,min=1"`
	PostTCEventCode        string   `json:"postTcEventCode"`
	Enabled                bool     `json:"enabled"`
	PlatformScope          []string `json:"platformScope"`
	FileType               string   `json:"fileType" binding:"required"`
	FileTypeLabel          string   `json:"fileTypeLabel" binding:"required"`
	FileTypeEditable       bool     `json:"fileTypeEditable"`
	URLTemplate            string   `json:"urlTemplate"`
	TargetFileNameTemplate string   `json:"targetFileNameTemplate"`
	FileNameTemplate       string   `json:"fileNameTemplate"`
	FileSizeField          string   `json:"fileSizeField"`
	ChecksumField          string   `json:"checksumField"`
	RawMode                string   `json:"rawMode"`
	DelaySeconds           int      `json:"delaySeconds"`
	TransportPath          string   `json:"transportPath"`
}

type TaskListFilter struct {
	Status   string `form:"status"`
	TypeCode string `form:"typeCode"`
	Keyword  string `form:"keyword"`
	Category string `form:"category"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type DeviceListFilter struct {
	Status      string `form:"status"`
	TypeCode    string `form:"typeCode"`
	Keyword     string `form:"keyword"`
	Category    string `form:"category"`
	ProductType string `form:"productType"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

type DeviceCandidateFilter struct {
	Category    string `form:"category"`
	TypeCode    string `form:"typeCode"`
	ProductType string `form:"productType"`
	Keyword     string `form:"keyword"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

func builtInTaskTypes() []TaskType {
	now := time.Now().Format(time.RFC3339)
	lte := coremodel.TechLTE
	nr := coremodel.TechNR
	return []TaskType{
		{
			TypeCode:               "ENB_IMG_UPGRADE",
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
			FileType:               "1 Firmware Upgrade Image",
			FileTypeLabel:          "1 Firmware Upgrade Image",
			FileTypeEditable:       true,
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
			FileType:               "X {OUI} Software Upgrade Patch",
			FileTypeLabel:          "X {OUI} Software Upgrade Patch",
			FileTypeEditable:       true,
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
			FileType:               "Firmware Upgrade Fpga",
			FileTypeLabel:          "Firmware Upgrade Fpga",
			FileTypeEditable:       true,
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
			FileType:               "1 Firmware Upgrade Image",
			FileTypeLabel:          "1 Firmware Upgrade Image",
			FileTypeEditable:       true,
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
			TypeCode:         "VERSION_ROLLBACK",
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
			FileType:               "6",
			FileTypeLabel:          "运行日志",
			FileTypeEditable:       true,
			TargetFileNameTemplate: "runtime-{task_id8}-{sn}.tar.gz",
			FileNameTemplate:       "runtime-{task_id8}-{sn}.tar.gz",
			TransportPath:          "/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}",
			LastEditor:             "system",
			UpdatedAt:              now,
		},
		{
			TypeCode:               "FAULT_LOG_COLLECT",
			Category:               "station_log",
			CategoryLabel:          "日志收集",
			DisplayName:            "故障日志采集",
			Description:            "复用现网日志采集 Upload 链路，提供 UFTE 内置的故障日志收集模板。",
			RPCType:                "UPLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:         "CODE_FAULT_LOG_COLLECT",
			PlatformScope:          []string{"4G eNB", "5G gNB", "DXDF", "BBU-QSS"},
			FileType:               "8",
			FileTypeLabel:          "故障日志",
			FileTypeEditable:       true,
			TargetFileNameTemplate: "fault-{task_id8}-{sn}.tar.gz",
			FileNameTemplate:       "fault-{task_id8}-{sn}.tar.gz",
			TransportPath:          "/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}",
			LastEditor:             "system",
			UpdatedAt:              now,
		},
		{
			TypeCode:               "CONFIG_BACKUP",
			Category:               "config_backup",
			CategoryLabel:          "配置文件备份",
			DisplayName:            "配置文件备份",
			Description:            "复用现网配置备份 Upload 链路，提供 UFTE 内置的配置文件备份模板。",
			RPCType:                "UPLOAD",
			BuiltIn:                true,
			Enabled:                true,
			StepChain:              []string{"CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_TRANSFER_COMPLETE"},
			PermissionCode:         "CODE_CONFIG_BACKUP",
			PlatformScope:          []string{"4G eNB", "5G gNB", "QAFA", "QAFB", "BBU-XSS", "BBU-QSS"},
			FileType:               "3",
			FileTypeLabel:          "Vendor Configuration File",
			FileTypeEditable:       false,
			TargetFileNameTemplate: "backup-{task_id8}-{sn}.xml",
			FileNameTemplate:       "backup-{task_id8}-{sn}.xml",
			TransportPath:          "/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}",
			LastEditor:             "system",
			UpdatedAt:              now,
		},
		{
			TypeCode:               "CONFIG_RESTORE",
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
			FileType:               "3 Vendor Configuration File",
			FileTypeLabel:          "3 Vendor Configuration File",
			FileTypeEditable:       false,
			URLTemplate:            "config_backup/{object_path}",
			TargetFileNameTemplate: "{file_name}",
			FileNameTemplate:       "{file_name}",
			TransportPath:          "/smallcell/FileDownloadService/config_backup/{object_path}",
			LastEditor:             "system",
			UpdatedAt:              now,
		},
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

func resolveTaskType(catalog []TaskType, taskType software.TaskType, productClass string) (TaskType, bool) {
	for _, item := range catalog {
		if item.softwareTaskType != taskType {
			continue
		}
		if item.techHint == nil {
			return item, true
		}
		if matchesTaskTypeScope(item, productClass) {
			return item, true
		}
	}
	if taskType == software.TaskTypeUpgrade {
		return findTaskTypeByCode(catalog, "ENB_IMG_UPGRADE")
	}
	return TaskType{}, false
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
		return strings.Contains(upper, "5G") || strings.Contains(upper, "GNB") || strings.Contains(upper, "QSS") || strings.Contains(upper, "XSS") || strings.Contains(upper, "BBU")
	case coremodel.TechLTE:
		return strings.Contains(upper, "4G") || strings.Contains(upper, "ENB") || strings.Contains(upper, "QAFA") || strings.Contains(upper, "QAFB") || strings.Contains(upper, "FAP") || strings.Contains(upper, "BM") || strings.Contains(upper, "BNQ") || strings.Contains(upper, "MLQ") || strings.Contains(upper, "MLN") || strings.Contains(upper, "BLQ")
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

func normalizeDeviceStatus(status software.UpgradeState) string {
	switch status {
	case software.UpgradeDownloading:
		return "downloading"
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
	if status == software.TaskPending || status == software.TaskSuspended {
		return "SEND_RPC"
	}
	if item.softwareTaskType == software.TaskTypeRollback {
		return "WAIT_REBOOT_COMPLETE"
	}
	if item.PostTCEventCode != "" {
		return "WAIT_INFORM_EVENT"
	}
	return "WAIT_TRANSFER_COMPLETE"
}

func progressForDeviceStatus(status string) int {
	switch status {
	case "downloading":
		return 45
	case "verifying":
		return 75
	case "ended", "failed":
		return 100
	default:
		return 0
	}
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
		if base, ok := defaultByCode[item.TypeCode]; ok {
			item.BuiltIn = item.BuiltIn || base.BuiltIn
			item.softwareTaskType = base.softwareTaskType
			item.techHint = base.techHint
			item.RPCType = base.RPCType
			item.StepChain = base.StepChain
		}
		result = append(result, item)
	}
	return result
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
		if category != "" && item.Category != category {
			continue
		}
		if item.softwareTaskType == 0 {
			continue
		}
		set[item.softwareTaskType] = struct{}{}
	}
	return set, nil
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

package imsparam

import (
	"time"

	"github.com/google/uuid"
)

// ParamFile 是运营者上传到参数文件库的一份核心网参数文件（ims_param_files 行）。
// 下发任务（IMS_PARAM_DISTRIBUTE）从这里按 ID 取文件派发 Download RPC。
type ParamFile struct {
	ID           uuid.UUID
	ParamType    string // FT_ImsCore_*
	FileName     string // 原始文件名（下发 TargetFileName 同名）
	ObjectBucket string
	ObjectPath   string // ims-params/{paramType}/{fileName}
	MD5          *string
	FileSize     int64
	Description  *string
	UploadedBy   *string
	DeviceSN     string // 设备采集文件的来源 SN；手动上传为空
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ParamFileFilter 列表过滤。
type ParamFileFilter struct {
	ParamType  string
	FileName   string
	UploadedBy string
	DeviceSN   string
	Page       int
	PageSize   int
	SortBy     string // created_at | updated_at
	SortDir    string // asc | desc
}

func (f ParamFileFilter) Limit() int {
	if f.PageSize <= 0 || f.PageSize > 200 {
		return 20
	}
	return f.PageSize
}

func (f ParamFileFilter) Offset() int {
	page := f.Page
	if page <= 0 {
		page = 1
	}
	return (page - 1) * f.Limit()
}

// ImportFailure / ImportResult 与 backup.LicenseImportResult 同形。
type ImportFailure struct {
	FileName  string `json:"file_name"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

type ImportResult struct {
	Succeeded []string        `json:"succeeded"`
	Failed    []ImportFailure `json:"failed"`
}

// 导入失败错误码（对齐 backup.ImportErr* 命名习惯）。
const (
	ImportErrInvalidName = "invalid_name"
	ImportErrEmptyBody   = "empty_body"
	ImportErrPutObject   = "put_object"
	ImportErrUpsert      = "upsert"
)

package mr

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/mr/parser"
)

// MRFileInfo holds metadata about an MR file.
type MRFileInfo struct {
	ID          uuid.UUID `json:"id" db:"id"`
	DeviceID    uuid.UUID `json:"device_id" db:"device_id"`
	DeviceSN    string    `json:"device_sn" db:"device_sn"`
	Carrier     string    `json:"carrier" db:"carrier"`
	MRType      string    `json:"mr_type" db:"mr_type"`
	FileName    string    `json:"file_name" db:"file_name"`
	FileSize    int64     `json:"file_size" db:"file_size"`
	CollectTime time.Time `json:"collect_time" db:"collect_time"`
	MinioPath   string    `json:"minio_path" db:"minio_path"`
	Parsed      bool      `json:"parsed" db:"parsed"`
	ParsedAt    *time.Time `json:"parsed_at,omitempty" db:"parsed_at"`
	RecordCount int       `json:"record_count" db:"record_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// MRRecordEntry is a stored MR record.
type MRRecordEntry struct {
	Time            time.Time              `json:"time"`
	FileID          uuid.UUID              `json:"file_id"`
	DeviceID        uuid.UUID              `json:"device_id"`
	CellID          string                 `json:"cell_id"`
	MRType          string                 `json:"mr_type"`
	MeasurementData map[string]interface{} `json:"measurement_data"`
}

// MRFileFilter defines query parameters for MR file retrieval.
type MRFileFilter struct {
	DeviceID  *uuid.UUID
	DeviceSN  *string // 给 File Management → MR Tab 的 per-device 抽屉用
	MRType    *string
	StartTime *time.Time
	EndTime   *time.Time
	model.ListRequest
}

// MRRecordFilter defines query parameters for MR record retrieval.
type MRRecordFilter struct {
	DeviceID *uuid.UUID
	FileID   *uuid.UUID
	MRType   *string
	CellID   *string
	StartTime *time.Time
	EndTime   *time.Time
	model.ListRequest
}

// MRFileDeviceAggregate 是某设备的 MR 文件聚合视图（mr_files GROUP BY device_sn）。
// 给 File Management 的"按设备列出"列表用：每行 1 个设备，含起止时间 + 文件数 + 上报状态
// + 站名 + 产品类（LEFT JOIN devices）。
type MRFileDeviceAggregate struct {
	DeviceSN         string    `db:"device_sn"          json:"device_sn"`
	SiteName         string    `db:"site_name"          json:"site_name"`
	ProductClass     string    `db:"product_class"      json:"product_class"`
	FirstCollectTime time.Time `db:"first_collect_time" json:"first_collect_time"`
	LastCollectTime  time.Time `db:"last_collect_time"  json:"last_collect_time"`
	FileCount        int64     `db:"file_count"         json:"file_count"`
	// Reporting 表示该设备当前是否在某个 task_status='on' 的 MR 任务里。前端用来
	// 在列表里 Badge 显示"上报中 / 已停止"。判定：mr_customize_task task_status='on'
	// AND device_sn = ANY(target_device_sns) 存在即 true。
	Reporting bool `db:"reporting" json:"reporting"`
}

// MRFileDeviceFilter 给 ListFileDeviceAggregates 用的过滤参数。
type MRFileDeviceFilter struct {
	Keyword      *string // 按 device_sn ILIKE
	SiteName     *string // 按 devices.site_name ILIKE
	ProductClass *string // 按 devices.product_class ILIKE
	model.ListRequest
}

// MRStore defines the interface for MR data persistence.
type MRStore interface {
	SaveFile(ctx context.Context, file *MRFileInfo) error
	UpdateFileParsed(ctx context.Context, fileID uuid.UUID, recordCount int) error
	BatchInsertRecords(ctx context.Context, fileID, deviceID uuid.UUID, mrType string, records []parser.MRRecord) error
	ListFiles(ctx context.Context, filter MRFileFilter) (*model.ListResponse[MRFileInfo], error)
	GetFileByID(ctx context.Context, fileID uuid.UUID) (*MRFileInfo, error)
	QueryRecords(ctx context.Context, filter MRRecordFilter) (*model.ListResponse[MRRecordEntry], error)

	// DeleteFilesBefore 删除 collect_time < cutoff 的 mr_files 行。
	// 由 internal/mr/task/cleaner.go 在 MinIO 清理后调用，保持两端一致。
	// 返回删除行数。
	DeleteFilesBefore(ctx context.Context, cutoff time.Time) (int64, error)

	// ListFileDeviceAggregates 按 device_sn 聚合 mr_files，返回每台设备的
	// 起止 collect_time + 文件数。给 File Management → MR Tab 主列表用。
	ListFileDeviceAggregates(ctx context.Context, filter MRFileDeviceFilter) (*model.ListResponse[MRFileDeviceAggregate], error)

	// ListFilesBySN 取某 SN 下全部 mr_files 元数据（无分页），给 handler 批量
	// 删除时收集 MinIO 路径用。
	ListFilesBySN(ctx context.Context, sn string) ([]MRFileInfo, error)

	// DeleteFilesBySN 删除该 SN 下所有 mr_files 元数据行。返回删除行数。
	// MinIO 对象由 handler 在调用前删除（参考 backup.LicenseService.BatchDelete 模式）。
	DeleteFilesBySN(ctx context.Context, sn string) (int64, error)

	// ListUncompressed / MarkCompressed 供 rawarchive.Sweeper 补偿扫描"原始 XML 压缩回写"
	// （issue #321 加固）：列出尚未确认压缩的对象键、标记其已压缩并把 minio_path 改键为
	// 压缩后的 .xml.gz（renames: old→new）。MarkCompressed 也被内联压缩成功后单条调用。
	ListUncompressed(ctx context.Context, olderThan time.Time, limit int) ([]string, error)
	MarkCompressed(ctx context.Context, renames map[string]string) error
}

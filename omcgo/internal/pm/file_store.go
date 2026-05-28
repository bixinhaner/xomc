package pm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// PMFileInfo represents metadata for a PM file stored in MinIO.
type PMFileInfo struct {
	ID           uuid.UUID  `json:"id"`
	DeviceID     uuid.UUID  `json:"device_id"`
	DeviceSN     string     `json:"device_sn"`
	Carrier      string     `json:"carrier"`
	Technology   string     `json:"technology"`
	FileName     string     `json:"file_name"`
	FileSize     int64      `json:"file_size"`
	CollectTime  time.Time  `json:"collect_time"`
	MinioPath    string     `json:"minio_path"`
	Parsed       bool       `json:"parsed"`
	ParsedAt     *time.Time `json:"parsed_at,omitempty"`
	CounterCount int        `json:"counter_count"`
	CreatedAt    time.Time  `json:"created_at"`
}

// PMFileFilter specifies criteria for listing PM files.
type PMFileFilter struct {
	DeviceID  *uuid.UUID
	DeviceSN  *string // 给 File Management → PM Tab 的 per-device 抽屉用
	StartTime *time.Time
	EndTime   *time.Time
	model.ListRequest
}

// PMFileDeviceAggregate 是某设备的 PM 文件聚合视图（pm_files GROUP BY device_sn）。
// 给 File Management 的 "按设备列出" 列表用：每行 1 个设备，含起止时间 + 文件数 + 上报状态。
type PMFileDeviceAggregate struct {
	DeviceSN         string    `db:"device_sn"          json:"device_sn"`
	FirstCollectTime time.Time `db:"first_collect_time" json:"first_collect_time"`
	LastCollectTime  time.Time `db:"last_collect_time"  json:"last_collect_time"`
	FileCount        int64     `db:"file_count"         json:"file_count"`
	// Reporting 用 last_collect_time 与 now() 的间隔判定：2 小时内有新文件视为
	// "上报中"。PM 没有像 MR 那样的订阅任务表，用最近活跃度做代理信号。
	Reporting bool `db:"reporting" json:"reporting"`
}

// PMFileDeviceFilter 给 ListFileDeviceAggregates 用。
type PMFileDeviceFilter struct {
	Keyword *string // 按 device_sn ILIKE
	model.ListRequest
}

// PMFileStore provides persistence for PM file metadata.
type PMFileStore interface {
	SaveFile(ctx context.Context, info *PMFileInfo) error
	GetFileByID(ctx context.Context, id uuid.UUID) (*PMFileInfo, error)
	ListFiles(ctx context.Context, filter PMFileFilter) (*model.ListResponse[PMFileInfo], error)
	ListFileDeviceAggregates(ctx context.Context, filter PMFileDeviceFilter) (*model.ListResponse[PMFileDeviceAggregate], error)
	UpdateFileParsed(ctx context.Context, id uuid.UUID, counterCount int) error
}

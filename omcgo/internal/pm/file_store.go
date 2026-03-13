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
	StartTime *time.Time
	EndTime   *time.Time
	model.ListRequest
}

// PMFileStore provides persistence for PM file metadata.
type PMFileStore interface {
	SaveFile(ctx context.Context, info *PMFileInfo) error
	GetFileByID(ctx context.Context, id uuid.UUID) (*PMFileInfo, error)
	ListFiles(ctx context.Context, filter PMFileFilter) (*model.ListResponse[PMFileInfo], error)
	UpdateFileParsed(ctx context.Context, id uuid.UUID, counterCount int) error
}

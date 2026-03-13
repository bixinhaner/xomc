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

// MRStore defines the interface for MR data persistence.
type MRStore interface {
	SaveFile(ctx context.Context, file *MRFileInfo) error
	UpdateFileParsed(ctx context.Context, fileID uuid.UUID, recordCount int) error
	BatchInsertRecords(ctx context.Context, fileID, deviceID uuid.UUID, mrType string, records []parser.MRRecord) error
	ListFiles(ctx context.Context, filter MRFileFilter) (*model.ListResponse[MRFileInfo], error)
	GetFileByID(ctx context.Context, fileID uuid.UUID) (*MRFileInfo, error)
	QueryRecords(ctx context.Context, filter MRRecordFilter) (*model.ListResponse[MRRecordEntry], error)
}

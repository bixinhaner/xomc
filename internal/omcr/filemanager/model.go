package filemanager

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// FileType represents the category of a managed file.
type FileType string

const (
	FileConfig   FileType = "config"
	FileLog      FileType = "log"
	FileFirmware FileType = "firmware"
	FileScript   FileType = "script"
	FileOther    FileType = "other"
)

// FileStatus represents the processing state of a managed file.
type FileStatus string

const (
	FileUploaded   FileStatus = "uploaded"
	FileProcessing FileStatus = "processing"
	FileReady      FileStatus = "ready"
	FileFailed     FileStatus = "failed"
)

// ManagedFile represents a file stored in MinIO with metadata persisted in PostgreSQL.
type ManagedFile struct {
	ID          uuid.UUID  `json:"id"`
	FileName    string     `json:"file_name"`
	FileType    FileType   `json:"file_type"`
	FileSize    int64      `json:"file_size"`
	MinIOPath   string     `json:"minio_path"`
	ContentType string     `json:"content_type"`
	Uploader    *string    `json:"uploader,omitempty"`
	DeviceSN    *string    `json:"device_sn,omitempty"`
	Status      FileStatus `json:"status"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// FileFilter specifies criteria for listing managed files.
type FileFilter struct {
	FileType *FileType
	DeviceSN *string
	Status   *FileStatus
	Search   *string
	model.ListRequest
}

package report

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ReportType represents the type of report.
type ReportType string

const (
	ReportPerformance ReportType = "performance"
	ReportAlarm       ReportType = "alarm"
	ReportDevice      ReportType = "device"
	ReportCapacity    ReportType = "capacity"
	ReportSecurity    ReportType = "security"
)

// ReportStatus represents the status of a report definition.
type ReportStatus string

const (
	ReportDraft     ReportStatus = "draft"
	ReportPublished ReportStatus = "published"
	ReportArchived  ReportStatus = "archived"
)

// ReportPeriod represents the period/frequency of a report.
type ReportPeriod string

const (
	PeriodDaily     ReportPeriod = "daily"
	PeriodWeekly    ReportPeriod = "weekly"
	PeriodMonthly   ReportPeriod = "monthly"
	PeriodQuarterly ReportPeriod = "quarterly"
	PeriodCustom    ReportPeriod = "custom"
)

// RecordStatus represents the status of a generated report record.
type RecordStatus string

const (
	RecordGenerating RecordStatus = "generating"
	RecordReady      RecordStatus = "ready"
	RecordFailed     RecordStatus = "failed"
)

// ReportDefinition represents a report template/definition.
type ReportDefinition struct {
	ID             uuid.UUID    `json:"id"`
	ReportName     string       `json:"report_name"`
	ReportType     ReportType   `json:"report_type"`
	Description    string       `json:"description,omitempty"`
	Format         []string     `json:"format"`
	Period         ReportPeriod `json:"period"`
	KPICodes       []string     `json:"kpi_codes"`
	DeviceGroups   []string     `json:"device_groups"`
	AutoGenerate   bool         `json:"auto_generate"`
	CronExpression string       `json:"cron_expression,omitempty"`
	Status         ReportStatus `json:"status"`
	Creator        string       `json:"creator,omitempty"`
	LastGenTime    *time.Time   `json:"last_gen_time,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// FormatJSON returns the Format field as JSON bytes.
func (d *ReportDefinition) FormatJSON() []byte {
	b, _ := json.Marshal(d.Format)
	return b
}

// KPICodesJSON returns the KPICodes field as JSON bytes.
func (d *ReportDefinition) KPICodesJSON() []byte {
	b, _ := json.Marshal(d.KPICodes)
	return b
}

// DeviceGroupsJSON returns the DeviceGroups field as JSON bytes.
func (d *ReportDefinition) DeviceGroupsJSON() []byte {
	b, _ := json.Marshal(d.DeviceGroups)
	return b
}

// ReportRecord represents a generated instance of a report.
type ReportRecord struct {
	ID                 uuid.UUID    `json:"id"`
	ReportDefinitionID uuid.UUID    `json:"report_definition_id"`
	ReportName         string       `json:"report_name"`
	Period             string       `json:"period,omitempty"`
	GenerateTime       time.Time    `json:"generate_time"`
	FileSize           int64        `json:"file_size"`
	DownloadURL        string       `json:"download_url,omitempty"`
	Format             string       `json:"format"`
	Status             RecordStatus `json:"status"`
	MinioPath          string       `json:"minio_path,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
}

// DefinitionFilter specifies criteria for listing report definitions.
type DefinitionFilter struct {
	ReportType *ReportType
	Status     *ReportStatus
	model.ListRequest
}

// RecordFilter specifies criteria for listing report records.
type RecordFilter struct {
	DefinitionID *uuid.UUID
	Format       *string
	model.ListRequest
}

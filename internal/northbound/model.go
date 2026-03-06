package northbound

import "time"

// PushTarget is the runtime representation of a northbound push target loaded from config.
type PushTarget struct {
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	AuthType   string   `json:"auth_type"`   // none, bearer, basic
	AuthToken  string   `json:"auth_token"`
	DataTypes  []string `json:"data_types"`  // alarm, pm, config, device
	Format     string   `json:"format"`      // json, xml
	BatchSize  int      `json:"batch_size"`
	RetryCount int      `json:"retry_count"`
	Enabled    bool     `json:"enabled"`
}

// SyncRequest describes a northbound data synchronization request.
type SyncRequest struct {
	DataType string    `json:"data_type" form:"data_type"` // alarm, pm, config, device
	Since    time.Time `json:"since" form:"since"`
	Format   string    `json:"format" form:"format"` // json, xml
}

// SyncResult holds the result of a full or incremental sync operation.
type SyncResult struct {
	DataType  string      `json:"data_type"`
	Items     interface{} `json:"items"`
	Total     int         `json:"total"`
	SyncedAt  time.Time   `json:"synced_at"`
	Truncated bool        `json:"truncated,omitempty"`
}

// ExportPMRequest defines parameters for a PM data export.
type ExportPMRequest struct {
	DeviceID     string `json:"device_id" form:"device_id"`
	CellID       string `json:"cell_id" form:"cell_id"`
	CounterGroup string `json:"counter_group" form:"counter_group"`
	StartTime    string `json:"start_time" form:"start_time" binding:"required"`
	EndTime      string `json:"end_time" form:"end_time" binding:"required"`
	Format       string `json:"format" form:"format"`
}

// ExportAlarmRequest defines parameters for an alarm data export.
type ExportAlarmRequest struct {
	DeviceSN  string `json:"device_sn" form:"device_sn"`
	Severity  string `json:"severity" form:"severity"`
	Status    string `json:"status" form:"status"`
	StartTime string `json:"start_time" form:"start_time"`
	EndTime   string `json:"end_time" form:"end_time"`
	Format    string `json:"format" form:"format"`
}

// AddPushTargetRequest is the API request for adding a push target.
type AddPushTargetRequest struct {
	ID         string   `json:"id" binding:"required"`
	URL        string   `json:"url" binding:"required"`
	AuthType   string   `json:"auth_type"`
	AuthToken  string   `json:"auth_token"`
	DataTypes  []string `json:"data_types" binding:"required"`
	Format     string   `json:"format"`
	BatchSize  int      `json:"batch_size"`
	RetryCount int      `json:"retry_count"`
	Enabled    bool     `json:"enabled"`
}

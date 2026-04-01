package model

import (
	"time"

	"github.com/google/uuid"
)

// DeviceParameter 表示特定设备上的一个 TR-069 参数当前实际值。
// 对应数据库 device_parameters 表，由 ACS Inform 处理时批量写入。
// 用于开站参数对比、GPV 查询和参数同步。
type DeviceParameter struct {
	DeviceID       uuid.UUID     `json:"device_id" db:"device_id"`
	ParameterPath  string        `json:"parameter_path" db:"parameter_path"`
	ParameterValue string        `json:"parameter_value" db:"parameter_value"`
	ParameterType  ParameterType `json:"parameter_type" db:"parameter_type"`
	Writable       bool          `json:"writable" db:"writable"`
	LastUpdatedAt  time.Time     `json:"last_updated_at" db:"last_updated_at"`
	FAPInstance    int           `json:"fap_instance" db:"fap_instance"`
	ParamGroup     string        `json:"param_group" db:"param_group"`
}

// ParameterDefinition 描述数据模型定义中的一个参数元数据。
// 由 ModelUploadService 将 CPE 上传的 XML 解析后嵌入到 data_model_definitions 表。
// 用于参数模板匹配、开站容窗参数标准化。
type ParameterDefinition struct {
	Path         string        `json:"path"`
	Type         ParameterType `json:"type"`
	Writable     bool          `json:"writable"`
	DefaultValue string        `json:"default_value,omitempty"`
	Description  string        `json:"description,omitempty"`
	MinValue     *int64        `json:"min_value,omitempty"`
	MaxValue     *int64        `json:"max_value,omitempty"`
	MaxLength    *int          `json:"max_length,omitempty"`
	EnumValues   []string      `json:"enum_values,omitempty"`
}

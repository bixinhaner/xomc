package indicator

import (
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ── Device Type ─────────────────────────────────────────────────────────────

type DeviceType string

const (
	DeviceTypeENB DeviceType = "ENB"
	DeviceTypeGSM DeviceType = "GSM"
	DeviceTypeGNB DeviceType = "GNB"
)

func (dt DeviceType) Suffix() string {
	switch dt {
	case DeviceTypeENB:
		return "enb"
	case DeviceTypeGSM:
		return "gsm"
	case DeviceTypeGNB:
		return "gnb"
	default:
		return ""
	}
}

func (dt DeviceType) GroupTable() string {
	return fmt.Sprintf("indicator_group_%s", dt.Suffix())
}

func (dt DeviceType) IndicatorTable() string {
	return fmt.Sprintf("perf_indicators_%s", dt.Suffix())
}

func (dt DeviceType) FormulaTable() string {
	return fmt.Sprintf("rela_platform_indicator_formula_%s", dt.Suffix())
}

func (dt DeviceType) EnabledTable() string {
	return fmt.Sprintf("enabled_pm_indicators_%s", dt.Suffix())
}

func ParseDeviceType(s string) (DeviceType, error) {
	switch s {
	case "ENB", "enb":
		return DeviceTypeENB, nil
	case "GSM", "gsm":
		return DeviceTypeGSM, nil
	case "GNB", "gnb":
		return DeviceTypeGNB, nil
	default:
		return "", fmt.Errorf("unknown device type: %s", s)
	}
}

// HasProductTypes returns true for ENB/GSM (which have product_types and indicator_level columns).
func (dt DeviceType) HasProductTypes() bool {
	return dt == DeviceTypeENB || dt == DeviceTypeGSM
}

// ── Domain Models ───────────────────────────────────────────────────────────

type IndicatorGroup struct {
	ID             string            `json:"id"`
	EnName         string            `json:"en_name,omitempty"`
	OperatorCode   *string           `json:"operator_code,omitempty"`
	IsBuildIn      string            `json:"is_build_in"`
	Description    *string           `json:"description,omitempty"`
	ParentID       string            `json:"parent_id"`
	CnName         *string           `json:"cn_name,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Children       []*IndicatorGroup `json:"children,omitempty"`
	IndicatorCount int               `json:"indicator_count,omitempty"`
}

type PerfIndicator struct {
	ID                string    `json:"id"`
	EnName            string    `json:"en_name"`
	CnName            *string   `json:"cn_name,omitempty"`
	EnDescription     *string   `json:"en_description,omitempty"`
	CnDescription     *string   `json:"cn_description,omitempty"`
	GroupID           string    `json:"group_id"`
	OperatorCode      *string   `json:"operator_code,omitempty"`
	DataType          *string   `json:"data_type,omitempty"`
	UnitID            *string   `json:"unit_id,omitempty"`
	Updator           *string   `json:"updator,omitempty"`
	IsBuildIn         string    `json:"is_build_in"`
	IsCounter         string    `json:"is_counter"`
	Arithmetic        *string   `json:"arithmetic,omitempty"`
	StatisType        *string   `json:"statis_type,omitempty"`
	CalculatingStatus *string   `json:"calculating_status,omitempty"`
	ProductTypes      *string   `json:"product_types,omitempty"`   // ENB/GSM only
	IndicatorLevel    *string   `json:"indicator_level,omitempty"` // ENB/GSM only
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type PlatformFormula struct {
	ID           string    `json:"id"`
	PlatformName string    `json:"platform_name"`
	IndicatorID  string    `json:"indicator_id"`
	Formula      string    `json:"formula"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type EnabledIndicator struct {
	OperatorCode string    `json:"operator_code"`
	IndicatorID  string    `json:"indicator_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CustName struct {
	OperatorCode string    `json:"operator_code"`
	PerfID       string    `json:"perf_id"`
	CustName     string    `json:"cust_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type IndicatorThreshold struct {
	ID              string    `json:"id"`
	IndicatorID     string    `json:"indicator_id,omitempty"`
	ThresholdPeriod string    `json:"threshold_period,omitempty"`
	ThresholdColor  string    `json:"threshold_color,omitempty"`
	ThresholdLow    string    `json:"threshold_low,omitempty"`
	ThresholdHigh   string    `json:"threshold_high,omitempty"`
	ThresholdLevel  string    `json:"threshold_level,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type IndicatorUnit struct {
	ID        string    `json:"id"`
	EnName    string    `json:"en_name,omitempty"`
	CnName    string    `json:"cn_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PerfAlarmThreshold struct {
	ID              string    `json:"id"`
	TempID          string    `json:"temp_id,omitempty"`
	IndicatorID     string    `json:"indicator_id,omitempty"`
	Comparison      string    `json:"comparison,omitempty"`
	ThresholdValue  string    `json:"threshold_value,omitempty"`
	Comparison2     string    `json:"comparison2,omitempty"`
	ThresholdValue2 string    `json:"threshold_value2,omitempty"`
	Operation       string    `json:"operation,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type TemplateRelArithmetic struct {
	ID          string    `json:"id"`
	TempID      string    `json:"temp_id"`
	IndicatorID string    `json:"indicator_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ── Request / Filter DTOs ───────────────────────────────────────────────────

type IndicatorGroupTreeRequest struct {
	DeviceType   string `form:"device_type" binding:"required,oneof=ENB GSM GNB"`
	OperatorCode string `form:"operator_code"`
}

type CreateGroupRequest struct {
	DeviceType   string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	EnName       string `json:"en_name"`
	CnName       string `json:"cn_name"`
	ParentID     string `json:"parent_id" binding:"required"`
	OperatorCode string `json:"operator_code"`
}

type UpdateGroupRequest struct {
	EnName      *string `json:"en_name"`
	CnName      *string `json:"cn_name"`
	Description *string `json:"description"`
}

type IndicatorListFilter struct {
	DeviceType     string  `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	GroupID        *string `json:"group_id"`
	IsBuildIn      *string `json:"is_build_in"`
	IsCounter      *string `json:"is_counter"`
	Keyword        *string `json:"keyword"`
	OperatorCode   *string `json:"operator_code"`
	ProductType    *string `json:"product_type"`
	IndicatorLevel *string `json:"indicator_level"`
	IsEnabled      *string `json:"is_enabled"`
	model.ListRequest
}

// IndicatorListItem enriches PerfIndicator with enabled status and custom name.
type IndicatorListItem struct {
	PerfIndicator
	IsEnabled bool   `json:"is_enabled"`
	CustName  string `json:"cust_name,omitempty"`
	GroupName string `json:"group_name,omitempty"` // 来自 indicator_group_* JOIN，前端列表显示分组用
}

type CreateIndicatorRequest struct {
	DeviceType     string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	EnName         string `json:"en_name" binding:"required"`
	CnName         string `json:"cn_name" binding:"required"`
	EnDescription  string `json:"en_description"`
	CnDescription  string `json:"cn_description"`
	GroupID        string `json:"group_id" binding:"required"`
	DataType       string `json:"data_type"`
	UnitID         string `json:"unit_id"`
	Updator        string `json:"updator"`
	IsCounter      string `json:"is_counter"`
	Arithmetic     string `json:"arithmetic"`
	StatisType     string `json:"statis_type"`
	ProductTypes   string `json:"product_types"`
	IndicatorLevel string `json:"indicator_level"`
	OperatorCode   string `json:"operator_code"`
}

type UpdateIndicatorRequest struct {
	EnName            *string `json:"en_name"`
	CnName            *string `json:"cn_name"`
	EnDescription     *string `json:"en_description"`
	CnDescription     *string `json:"cn_description"`
	GroupID           *string `json:"group_id"`
	DataType          *string `json:"data_type"`
	UnitID            *string `json:"unit_id"`
	Updator           *string `json:"updator"`
	Arithmetic        *string `json:"arithmetic"`
	StatisType        *string `json:"statis_type"`
	ProductTypes      *string `json:"product_types"`
	IndicatorLevel    *string `json:"indicator_level"`
	CalculatingStatus *string `json:"calculating_status"`
}

type EnableIndicatorsRequest struct {
	DeviceType   string   `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	OperatorCode string   `json:"operator_code" binding:"required"`
	IndicatorIDs []string `json:"indicator_ids" binding:"required,min=1"`
	Enable       bool     `json:"enable"`
}

type ExportRequest struct {
	DeviceType   string  `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	GroupID      *string `json:"group_id"`
	OperatorCode *string `json:"operator_code"`
}

type CustNameUpdateRequest struct {
	DeviceType   string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	OperatorCode string `json:"operator_code" binding:"required"`
	PerfID       string `json:"perf_id" binding:"required"`
	CustName     string `json:"cust_name"`
}

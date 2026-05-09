package license

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// LicenseStatus represents the current state of a license.
type LicenseStatus string

const (
	StatusActive  LicenseStatus = "active"
	StatusExpired LicenseStatus = "expired"
	StatusPending LicenseStatus = "pending"
	StatusTrial   LicenseStatus = "trial"
	StatusRevoked LicenseStatus = "revoked"
)

// LicenseType represents the type of a license.
type LicenseType string

const (
	TypePerpetual    LicenseType = "perpetual"
	TypeSubscription LicenseType = "subscription"
	TypeTrial        LicenseType = "trial"
	TypeEvaluation   LicenseType = "evaluation"
)

// License represents a single license record.
type License struct {
	ID          uuid.UUID       `json:"id"`
	LicenseName string          `json:"license_name"`
	LicenseCode string          `json:"license_code"`
	ProductName string          `json:"product_name"`
	LicenseType LicenseType     `json:"license_type"`
	Status      LicenseStatus   `json:"status"`
	MaxDevices  int             `json:"max_devices"`
	UsedDevices int             `json:"used_devices"`
	Features    json.RawMessage `json:"features"`
	IssueDate   time.Time       `json:"issue_date"`
	ExpiryDate  *time.Time      `json:"expiry_date"`
	Licensor    *string         `json:"licensor"`
	DeviceType  *string         `json:"device_type"`
	Region      *string         `json:"region"`
	Notes       *string         `json:"notes"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`

	// Enforcement extension (T-0015 / R-103, migration 000043).

	// GracePeriodDays is the number of days after ExpiryDate that the license
	// is still considered active (0 = no grace). Range [0, 365].
	GracePeriodDays int `json:"grace_period_days"`

	// CapacityAlertThresholds is a JSON array of percentage breakpoints (e.g.
	// [80, 90, 95]) that trigger capacity alerts when used_devices/max_devices
	// crosses each threshold.
	CapacityAlertThresholds json.RawMessage `json:"capacity_alert_thresholds"`

	// LastCapacityAlertAt tracks the most recent capacity alert dispatch for
	// dedup (6h window per threshold). Nil if never alerted.
	LastCapacityAlertAt *time.Time `json:"last_capacity_alert_at"`

	// LastCapacityAlertThreshold records the threshold percentage of the most
	// recent alert so the monitor can detect threshold crossings.
	LastCapacityAlertThreshold *int `json:"last_capacity_alert_threshold"`
}

// Quota describes the current license enforcement state, returned by
// GET /api/v1/licenses/quota.
type Quota struct {
	HasActiveLicense bool    `json:"has_active_license"`
	MaxDevices       int     `json:"max_devices"`
	UsedDevices      int     `json:"used_devices"`
	UsageRatio       float64 `json:"usage_ratio"`
	DaysRemaining    int     `json:"days_remaining"` // -1 = perpetual / no expiry
	LicenseType      string  `json:"license_type"`
	GracePeriodDays  int     `json:"grace_period_days"`
}

// LicenseSummary contains aggregated license statistics.
//
// EnforcementHits7d 是 T-0100-P2 加的字段：近 7 天 enforcement 拒绝次数
// （license_logs WHERE result='denied' AND created_at >= now()-7d）。
// 用于 LicenseList 顶部"近 7 天 enforcement 命中"卡片，让运维一眼看到
// 拦截次数趋势。值不可用时（如 license_logs 表不存在）退化为 0。
type LicenseSummary struct {
	Total             int64 `json:"total"`
	Active            int64 `json:"active"`
	Expired           int64 `json:"expired"`
	Pending           int64 `json:"pending"`
	ExpiringSoon      int64 `json:"expiring_soon"`
	EnforcementHits7d int64 `json:"enforcement_hits_7d"`
}

// LicenseFilter specifies criteria for listing licenses.
type LicenseFilter struct {
	Status      *LicenseStatus
	LicenseType *LicenseType
	DeviceType  *string
	model.ListRequest
}

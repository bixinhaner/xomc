// system_license_model.go — F06 System License 重构 P1 Step 1 新增。
//
// PRD: docs/project/prd/F06-system-license-redesign.md
//
// 与 model.go 老 License 类型**并存**——本文件是新模型；老 License/LicenseStatus/
// LicenseType 等留作 P1 过渡期，等 Step 5 才删除。两套类型共享同一包 license，
// 但在领域语义上正交：老的是 multi-license + Activate/Revoke 模型，本文件是
// singleton system_license + Update 模型。
package license

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// SystemLicenseType 是 system license 类型枚举（DB CHECK 约束保证）。
type SystemLicenseType string

const (
	SystemLicenseTypeCommercial SystemLicenseType = "Commercial"
	SystemLicenseTypeTrial      SystemLicenseType = "Trial"
	SystemLicenseTypeEvaluation SystemLicenseType = "Evaluation"
	SystemLicenseTypeInternal   SystemLicenseType = "Internal"
)

// SystemLicenseSignatureStatus 复用现有 SignatureStatus 字串值（verified/unverified/invalid）。
// 这里仅声明类型 alias，避免跨字段重复定义。
type SystemLicenseSignatureStatus = SignatureStatus

// DevicesSupport 是设备类型 → 容量配额的 map。
// JSON 形态：{"eNB":10000,"gNB":10000,"CPE":10000,"WCG":1000,"UPS":1000}
//
// 设计 map 而非固定字段，原因：未来新增设备类型（如 RRU / Massive MIMO）不需
// 改 schema，只需 license 文件加 key。enforcer 按 device.type 查 map 算容量。
type DevicesSupport map[string]int

// FeatureList 是三级嵌套功能矩阵（PRD §4）。
//
// 顶层 key = 一级模块名（Dashboard / MAP / eNB / gNB / CPE / Performance / System / ...）
// 值可以是：
//   - string "All"             — 整个模块全部解锁
//   - []string ["a","b","c"]   — 该模块部分功能
//   - map[string][]string      — 嵌套：子模块 → 功能项列表
//
// 用 json.RawMessage 保灵活（避免 Go 端强类型化限制三层结构）；service 层
// 用 FeatureChecker（P2 实现）解析后做权限判断。
type FeatureList json.RawMessage

// MarshalJSON 透传 raw bytes，不重复包装。
func (f FeatureList) MarshalJSON() ([]byte, error) {
	if len(f) == 0 {
		return []byte("{}"), nil
	}
	return f, nil
}

// UnmarshalJSON 直接吞 raw bytes，留给上层 FeatureChecker 解析。
func (f *FeatureList) UnmarshalJSON(data []byte) error {
	*f = FeatureList(append((*f)[:0], data...))
	return nil
}

// SystemLicense 对应 system_license 表行。
//
// 业务不变量（DB 约束保证）：
//   - 同时最多 1 行 is_current=true（partial unique index）
//   - license_id 全局唯一（UNIQUE column）
//   - signature_status ∈ {verified, unverified, invalid}（CHECK）
//   - license_type ∈ {Commercial, Trial, Evaluation, Internal}（CHECK）
type SystemLicense struct {
	ID                uuid.UUID                    `json:"id"                          db:"id"`
	LicenseID         string                       `json:"license_id"                  db:"license_id"`
	LicenseType       SystemLicenseType            `json:"license_type"                db:"license_type"`
	Issuer            *string                      `json:"issuer,omitempty"            db:"issuer"`
	Licensee          *string                      `json:"licensee,omitempty"          db:"licensee"`
	IssuedAt          time.Time                    `json:"issued_at"                   db:"issued_at"`
	ExpiryDate        *time.Time                   `json:"expiry_date,omitempty"       db:"expiry_date"`
	DevicesSupport    DevicesSupport               `json:"devices_support"             db:"devices_support"`
	FeatureList       FeatureList                  `json:"feature_list"                db:"feature_list"`
	RawContent        string                       `json:"raw_content,omitempty"       db:"raw_content"`
	Signature         *string                      `json:"signature,omitempty"         db:"signature"`
	SignatureKeyID    *string                      `json:"signature_key_id,omitempty"  db:"signature_key_id"`
	SignatureStatus   SystemLicenseSignatureStatus `json:"signature_status"            db:"signature_status"`
	UploadedAt        time.Time                    `json:"uploaded_at"                 db:"uploaded_at"`
	UploadedByUserID  *uuid.UUID                   `json:"uploaded_by_user_id,omitempty" db:"uploaded_by_user_id"`
	IsCurrent         bool                         `json:"is_current"                  db:"is_current"`
	CreatedAt         time.Time                    `json:"created_at"                  db:"created_at"`
	UpdatedAt         time.Time                    `json:"updated_at"                  db:"updated_at"`

	// 以下为 compute-on-read 的运行时状态（非 DB 列），由 SystemLicenseService
	// GetCurrent 填充。CumulativeUsedHours 含未推进的增量（now - last_visited）
	// 以保证页面展示与 enforcement 裁决一致。
	CumulativeUsedHours  *float64 `json:"cumulative_used_hours,omitempty" db:"-"`
	CumulativeLimitHours int      `json:"cumulative_limit_hours"         db:"-"`
	IsExpired            bool     `json:"is_expired"                     db:"-"`
}

// SystemLicenseHistory 对应 system_license_history 表行——每次 Update 把
// 当前 system_license 的副本拷贝进来，留作合规审计。
//
// 与 SystemLicense 的区别：
//   - 无 is_current（永远是历史副本）
//   - 多 replaced_at / replaced_by_id（标记被哪条新 license 替换 + 替换时刻）
type SystemLicenseHistory struct {
	ID                uuid.UUID                    `json:"id"                          db:"id"`
	LicenseID         string                       `json:"license_id"                  db:"license_id"`
	LicenseType       SystemLicenseType            `json:"license_type"                db:"license_type"`
	Issuer            *string                      `json:"issuer,omitempty"            db:"issuer"`
	Licensee          *string                      `json:"licensee,omitempty"          db:"licensee"`
	IssuedAt          time.Time                    `json:"issued_at"                   db:"issued_at"`
	ExpiryDate        *time.Time                   `json:"expiry_date,omitempty"       db:"expiry_date"`
	DevicesSupport    DevicesSupport               `json:"devices_support"             db:"devices_support"`
	FeatureList       FeatureList                  `json:"feature_list"                db:"feature_list"`
	RawContent        string                       `json:"raw_content,omitempty"       db:"raw_content"`
	Signature         *string                      `json:"signature,omitempty"         db:"signature"`
	SignatureKeyID    *string                      `json:"signature_key_id,omitempty"  db:"signature_key_id"`
	SignatureStatus   SystemLicenseSignatureStatus `json:"signature_status"            db:"signature_status"`
	UploadedAt        time.Time                    `json:"uploaded_at"                 db:"uploaded_at"`
	UploadedByUserID  *uuid.UUID                   `json:"uploaded_by_user_id,omitempty" db:"uploaded_by_user_id"`
	ReplacedAt        time.Time                    `json:"replaced_at"                 db:"replaced_at"`
	ReplacedByID      *uuid.UUID                   `json:"replaced_by_id,omitempty"    db:"replaced_by_id"`
	CreatedAt         time.Time                    `json:"created_at"                  db:"created_at"`
}

// SystemLicenseHistoryFilter — ListHistory 查询参数。
type SystemLicenseHistoryFilter struct {
	model.ListRequest
	LicenseID *string // 按 license_id 过滤（同 license_id 多次替换历史）
}

package license

import (
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// LogType 是 license_logs.log_type 枚举（与 migration 000073 chk_license_logs_log_type 保持一致）。
type LogType string

const (
	// 用户写 / 敏感读
	LogTypeImport      LogType = "import"
	LogTypeActivate    LogType = "activate"
	LogTypeRevoke      LogType = "revoke"
	LogTypeQueryDetail LogType = "query_detail"

	// enforcer 拒绝（system）
	LogTypeEnforcementCapacity LogType = "enforcement_capacity"
	LogTypeEnforcementExpiry   LogType = "enforcement_expiry"

	// monitor cron（system）
	LogTypeCapacityAlert LogType = "capacity_alert"
	LogTypeExpiryAlert   LogType = "expiry_alert"
	LogTypeAutoExpire    LogType = "auto_expire"
)

// LogResult 是 license_logs.result 枚举（与 migration 000073 chk_license_logs_result 保持一致）。
type LogResult string

const (
	LogResultSuccess LogResult = "success"
	LogResultFailed  LogResult = "failed"
	LogResultDenied  LogResult = "denied"
	LogResultWarning LogResult = "warning"
)

// LicenseLog 对应 license_logs 表的一行记录。
type LicenseLog struct {
	ID          uuid.UUID       `json:"id"`
	LicenseID   *uuid.UUID      `json:"license_id"` // 可空：license 被删除时 ON DELETE SET NULL
	LogType     LogType         `json:"log_type"`
	ActorUserID *uuid.UUID      `json:"actor_user_id"` // 可空：system 操作（cron / enforcer）
	Result      LogResult       `json:"result"`
	Details     json.RawMessage `json:"details"`
	ClientIP    *string         `json:"client_ip"`
	UserAgent   *string         `json:"user_agent"`
	CreatedAt   time.Time       `json:"created_at"`
}

// LicenseLogEntry 是 LogWriter.Write 的入参。
//
// 用 map[string]any 而非 json.RawMessage 是为了让调用方写日志时不用手动 json.Marshal。
// LogWriter 内部统一 marshal 后入库；marshal 失败时 details 落空对象 + warn 日志，
// 不阻断业务主流程（见 LogWriter.Write 实现）。
type LicenseLogEntry struct {
	LicenseID   *uuid.UUID
	LogType     LogType
	ActorUserID *uuid.UUID // nil = system
	Result      LogResult
	Details     map[string]any
	ClientIP    string
	UserAgent   string
}

// LicenseLogFilter 用于 LicenseLogRepository.List 的过滤条件。
//
// 全部字段可空：
//   - LicenseID 锁定单 license（List 详情抽屉 / 跳转过滤）
//   - LogTypes / Results 多值过滤（前端 multi-select）
//   - ActorUserID 单用户过滤（"我的操作历史"）
//   - StartTime / EndTime 时间窗
//   - Search 在 details 文本中模糊匹配（暂用 ::text ILIKE）
type LicenseLogFilter struct {
	LicenseID   *uuid.UUID  `form:"license_id"`
	LogTypes    []LogType   `form:"log_type"`
	Results     []LogResult `form:"result"`
	ActorUserID *uuid.UUID  `form:"actor_user_id"`
	StartTime   *time.Time  `form:"start_time"`
	EndTime     *time.Time  `form:"end_time"`
	Search      *string     `form:"search"`
	model.ListRequest
}

// parseClientIP 把字符串 IP 转 *net.IP；空 / 非法 IP 返 nil（PG INET 列允许 NULL）。
// repo 写入时调用，避免 PG 报 invalid input syntax for type inet。
func parseClientIP(s string) *net.IP {
	if s == "" {
		return nil
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil
	}
	return &ip
}

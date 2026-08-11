// Package audit provides cross-module business audit logging for the OMC.
//
// Unlike core/components/logger (process-level zap structured log) and
// core/middleware/redact (PII field redaction), this package records BUSINESS
// AUDIT events (5 categories per W3.G.2 charter) into the PostgreSQL audit_logs
// table for compliance and operational forensics.
//
// The 5 charter-mandated action categories are:
//
//   - login    - authentication events (login_success, login_failed, logout)
//   - config   - configuration changes (template apply, baseline change, parameter set)
//   - upgrade  - software/firmware upgrade triggers
//   - reboot   - device reboot commands (single + batch)
//   - delete   - resource deletion (device, template, license, alarm filter, etc.)
//
// Usage from any module:
//
//	audit.Log(ctx, audit.Entry{
//	    Action:       audit.ActionDelete,
//	    ResourceType: audit.ResourceDevice,
//	    ResourceID:   id.String(),
//	    Success:      err == nil,
//	    ErrorMessage: errMessageOrEmpty(err),
//	})
//
// The package uses a lazily-initialized singleton sink configured at app
// boot via SetDefault. If no sink is configured, Log is a no-op (write to
// the optional fallback zap logger if set), so unit tests that don't wire
// the audit package still work.
package audit

import (
	"context"
	"sync/atomic"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Action constants for the 5 W3.G.2 charter-mandated audit categories.
//
// These are the canonical action prefixes. Sub-actions (e.g.
// "login_success", "delete_batch") are encoded as-is; the prefix lets
// dashboards filter by category via SQL LIKE 'login%'.
const (
	// ActionLogin covers login_success / login_failed / logout.
	ActionLogin = "login"
	// ActionConfig covers config push/pull, template apply, baseline change.
	ActionConfig = "config"
	// ActionUpgrade covers firmware/software upgrade trigger and rollback.
	ActionUpgrade = "upgrade"
	// ActionReboot covers single + batch device reboot.
	ActionReboot = "reboot"
	// ActionDelete covers any resource deletion (device, template, license, etc).
	ActionDelete = "delete"
	// ActionUserCreate covers admin-initiated user account creation (issue #649).
	// 非 W3.G.2 charter 原列类目，为「默认密码接通」合规追溯新增：details.used_default_password
	// 区分「手填 vs 默认密码」两条路径。
	ActionUserCreate = "user_create"
	// ActionPasswordReset covers admin-initiated password reset (issue #649).
	// 区别于用户自助改密；含硬规则强制改密 + 旧 token 立即失效（OWASP A07）。
	ActionPasswordReset = "password_reset"
	// ActionPasswordChange covers authenticated users changing their own password.
	ActionPasswordChange = "password_change"
)

// ResourceType constants identify the entity an action targets.
const (
	ResourceAuth             = "auth"
	ResourceDevice           = "device"
	ResourceTemplate         = "template"
	ResourceBaseline         = "baseline"
	ResourceFirmware         = "firmware"
	ResourceUpgradeTask      = "upgrade_task"
	ResourceLicense          = "license"
	ResourceAlarmFilter      = "alarm_filter"
	ResourceUser             = "user"
	ResourceRole             = "role"
	ResourceConfig           = "config"
	ResourceGeofence         = "geofence"
	ResourceGeofenceBinding  = "geofence_binding"
	ResourceGeofenceSettings = "geofence_settings"
)

// Entry is a single business audit record.
//
// Action MUST be one of the Action* constants (or a sub-action prefixed
// with one of them). ResourceType SHOULD be one of the Resource* constants
// where applicable. Username, IP, UserAgent are recommended but optional.
type Entry struct {
	UserID       *uuid.UUID
	Username     string
	Action       string
	ResourceType string
	ResourceID   string
	Details      map[string]interface{}
	IPAddress    string
	UserAgent    string
	Success      bool
	ErrorMessage string
}

// Sink is the persistence interface for audit entries.
//
// The default implementation in this package wraps admin.AuditRepository,
// but tests can swap any Sink (in-memory, no-op, channel-based) via
// SetDefault.
type Sink interface {
	Write(ctx context.Context, e Entry) error
}

// noopSink discards entries; used when no sink is configured.
type noopSink struct{}

func (noopSink) Write(_ context.Context, _ Entry) error { return nil }

// sinkHolder wraps Sink so atomic.Value sees a stable concrete type
// regardless of which Sink implementation is stored (atomic.Value rejects
// stores of differing dynamic types).
type sinkHolder struct{ s Sink }

// defaultSink is the package-level singleton. We use atomic.Value so
// SetDefault is race-safe for the rare case of late wiring.
var defaultSink atomic.Value // holds sinkHolder

// loggerHolder wraps *zap.Logger for the same atomic.Value reason.
type loggerHolder struct{ l *zap.Logger }

// fallbackLogger is an optional zap logger used only to surface errors
// from the configured sink. Never set in tests.
var fallbackLogger atomic.Value // holds loggerHolder

func init() {
	defaultSink.Store(sinkHolder{s: noopSink{}})
	fallbackLogger.Store(loggerHolder{l: nil})
}

// SetDefault wires the package-level audit sink. Call once at app boot.
// Passing nil resets to no-op (useful in tests).
func SetDefault(s Sink) {
	if s == nil {
		s = noopSink{}
	}
	defaultSink.Store(sinkHolder{s: s})
}

// SetFallbackLogger registers a zap logger used to report sink write errors.
// Optional; tests typically leave this nil.
func SetFallbackLogger(l *zap.Logger) {
	if l == nil {
		return
	}
	fallbackLogger.Store(loggerHolder{l: l})
}

// Default returns the configured sink (never nil).
func Default() Sink {
	v := defaultSink.Load()
	if v == nil {
		return noopSink{}
	}
	h, ok := v.(sinkHolder)
	if !ok || h.s == nil {
		return noopSink{}
	}
	return h.s
}

// Log writes a single audit entry through the configured sink.
//
// Errors from the sink are swallowed but reported to the fallback zap
// logger if registered — audit logging MUST NOT break business flows.
// The call is synchronous; callers wanting fire-and-forget should wrap
// in a goroutine themselves.
func Log(ctx context.Context, e Entry) {
	if e.Action == "" {
		return
	}
	sink := Default()
	if err := sink.Write(ctx, e); err != nil {
		if v := fallbackLogger.Load(); v != nil {
			if h, ok := v.(loggerHolder); ok && h.l != nil {
				h.l.Error("audit sink write failed",
					zap.String("action", e.Action),
					zap.String("resource_type", e.ResourceType),
					zap.String("resource_id", e.ResourceID),
					zap.Error(err),
				)
			}
		}
	}
}

// LogAsync is a fire-and-forget variant. Useful for hot paths (login,
// reboot) where the audit write must not add latency. Uses a fresh
// background context so request cancellation does not abort the write.
func LogAsync(e Entry) {
	go Log(context.Background(), e)
}

package license

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LogWriter 是审计日志的写入入口。
//
// 设计原则：
//   - **写日志失败不阻断主业务流程**：Write 内部把 repo 错误转为 zap.Warn，
//     永不返回 error。这样 enforcer / monitor / handler 调用 Write 时不需要关心
//     失败处理路径。
//   - **details 由 LogWriter 统一序列化**：调用方传 map[string]any，本服务
//     marshal；marshal 失败时回退为 `{}` + warn 日志。
//   - **接口而非具体类型**：方便测试用 in-memory 实现替换；enforcer 通过 setter
//     注入避免循环依赖。
type LogWriter interface {
	Write(ctx context.Context, entry LicenseLogEntry)
}

// NoopLogWriter 在没注入真实 LogWriter 时兜底（早期 bootstrap 或测试场景）。
// 这样 enforcer / monitor / handler 可以无条件调 Write，不需要 nil 判空。
type NoopLogWriter struct{}

// Write 实现 LogWriter，丢弃所有日志。
func (NoopLogWriter) Write(_ context.Context, _ LicenseLogEntry) {}

// pgLogWriter 是基于 LicenseLogRepository 的真实实现。
type pgLogWriter struct {
	repo   LicenseLogRepository
	logger *zap.Logger
}

// NewLogWriter 构造一个把日志写入数据库的 LogWriter。
func NewLogWriter(repo LicenseLogRepository, logger *zap.Logger) LogWriter {
	return &pgLogWriter{
		repo:   repo,
		logger: logger.Named("license-log-writer"),
	}
}

// Write 实现 LogWriter.Write。
func (w *pgLogWriter) Write(ctx context.Context, entry LicenseLogEntry) {
	var details []byte
	if len(entry.Details) == 0 {
		// nil / empty map → 规范化为 `{}`（不是 `null`），与表 default 对齐
		details = []byte(`{}`)
	} else {
		var err error
		details, err = json.Marshal(entry.Details)
		if err != nil {
			// marshal 失败极罕见（map[string]any 包含不可 JSON 化的值，如 chan / func），
			// 降级为 `{}` 但仍写一条日志保留事件本身。
			w.logger.Warn("license log details marshal failed, fallback to empty object",
				zap.String("log_type", string(entry.LogType)),
				zap.Error(err),
			)
			details = []byte(`{}`)
		}
	}

	var clientIP *string
	if entry.ClientIP != "" {
		ip := entry.ClientIP
		clientIP = &ip
	}
	var userAgent *string
	if entry.UserAgent != "" {
		ua := entry.UserAgent
		userAgent = &ua
	}

	log := &LicenseLog{
		ID:          uuid.New(),
		LicenseID:   entry.LicenseID,
		LogType:     entry.LogType,
		ActorUserID: entry.ActorUserID,
		Result:      entry.Result,
		Details:     details,
		ClientIP:    clientIP,
		UserAgent:   userAgent,
	}

	if err := w.repo.Create(ctx, log); err != nil {
		// 写日志失败不阻断主业务：记 warn，让人能从 app.log 看到落库失败但前台不知情。
		w.logger.Warn("license_log persist failed (non-blocking)",
			zap.String("log_type", string(entry.LogType)),
			zap.String("result", string(entry.Result)),
			zap.Any("license_id", entry.LicenseID),
			zap.Any("actor_user_id", entry.ActorUserID),
			zap.Error(err),
		)
	}
}

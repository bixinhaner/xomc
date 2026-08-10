package admin

import (
	"context"

	"github.com/omcgo/omcgo/internal/admin/audit"
)

// AuditSink adapts an admin.AuditRepository to the audit.Sink interface,
// translating cross-module audit.Entry records into the admin.AuditLog
// schema persisted by PgAuditRepository.
//
// This adapter lives in the admin package (not the audit subpackage) to
// avoid an import cycle: admin already depends on audit transitively
// for the constants, and admin owns AuditRepository.
type AuditSink struct {
	repo AuditRepository
}

var _ audit.Sink = (*AuditSink)(nil)

// NewAuditSink creates a Sink backed by the given AuditRepository.
func NewAuditSink(repo AuditRepository) *AuditSink {
	return &AuditSink{repo: repo}
}

// Write persists an audit entry by mapping it to admin.AuditLog.
//
// Action+sub-action encoding: when Success is false we suffix the action
// with "_failed" (e.g. "delete" -> "delete_failed") so DB queries can
// distinguish the two cases without parsing Details.
func (s *AuditSink) Write(ctx context.Context, e audit.Entry) error {
	if s == nil || s.repo == nil {
		return nil
	}

	return s.repo.Create(ctx, auditLogFromEntry(e))
}

func auditLogFromEntry(e audit.Entry) *AuditLog {
	action := e.Action
	if !e.Success && e.Action != "" {
		action = e.Action + "_failed"
	}

	details := e.Details
	if e.ErrorMessage != "" {
		if details == nil {
			details = map[string]interface{}{}
		}
		details["error"] = e.ErrorMessage
	}

	log := &AuditLog{
		UserID:    e.UserID,
		Username:  e.Username,
		Action:    action,
		Resource:  e.ResourceType,
		Details:   details,
		IPAddress: e.IPAddress,
		UserAgent: e.UserAgent,
	}
	if e.ResourceID != "" {
		log.ResourceID = e.ResourceID
	}
	return log
}

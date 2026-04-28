package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin/audit"
)

// AuditContextFromGin extracts the authenticated user identity and request
// metadata from a Gin context to populate an audit.Entry. All fields default
// to zero values when the request is unauthenticated (e.g. system-triggered).
//
// This helper is exposed so other modules (device, software, config, ...)
// can construct audit.Entry without re-reading admin's middleware context
// keys directly.
func AuditContextFromGin(c *gin.Context) audit.Entry {
	if c == nil {
		return audit.Entry{}
	}

	e := audit.Entry{
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}

	if v, ok := c.Get(CtxKeyUserID); ok {
		if id, ok2 := v.(uuid.UUID); ok2 && id != uuid.Nil {
			id := id // copy to take address
			e.UserID = &id
		}
	}
	if v, ok := c.Get(CtxKeyUsername); ok {
		if name, ok2 := v.(string); ok2 {
			e.Username = name
		}
	}
	return e
}

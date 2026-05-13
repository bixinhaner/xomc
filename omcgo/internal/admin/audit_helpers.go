package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin/audit"
)

// UserIDFromCtx 从 gin context 提取 user_id (uuid.UUID)。
//
// 中间件（admin.RequireAuth / RequireAPIKey）当前向 ctx 存的是 uuid.UUID，
// 但历史代码 / 测试 / 跨进程透传可能存 string，本 helper 同时兼容两种存储类型。
//
// 返回 (uuid.Nil, false) 表示 ctx 不存在 user_id、类型无法识别、或值为 uuid.Nil。
// 历史教训：task/handler.go 旧版用 `userID.(string)` 强转，遇到 middleware 存
// 的 uuid.UUID 触发 panic（commit 3d931671）。这是该模式的统一收敛点。
func UserIDFromCtx(c *gin.Context) (uuid.UUID, bool) {
	if c == nil {
		return uuid.Nil, false
	}
	v, exists := c.Get(CtxKeyUserID)
	if !exists {
		return uuid.Nil, false
	}
	switch u := v.(type) {
	case uuid.UUID:
		if u == uuid.Nil {
			return uuid.Nil, false
		}
		return u, true
	case string:
		if id, err := uuid.Parse(u); err == nil && id != uuid.Nil {
			return id, true
		}
	}
	return uuid.Nil, false
}

// UserIDStringFromCtx 返回 user_id 的字符串形式（uuid.String() 或空串）。
// 适用于审计字段、creator_id 这类持久化为 string 的场景。
func UserIDStringFromCtx(c *gin.Context) string {
	if id, ok := UserIDFromCtx(c); ok {
		return id.String()
	}
	return ""
}

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

	if id, ok := UserIDFromCtx(c); ok {
		id := id // copy to take address
		e.UserID = &id
	}
	if v, ok := c.Get(CtxKeyUsername); ok {
		if name, ok2 := v.(string); ok2 {
			e.Username = name
		}
	}
	return e
}

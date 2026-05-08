package northbound

import (
	"time"

	"github.com/google/uuid"
)

// ServerRole 北向 OSS 服务器角色：主用 / 备用。
type ServerRole string

const (
	ServerRolePrimary ServerRole = "primary"
	ServerRoleStandby ServerRole = "standby"
)

// IsValid reports whether the role is one of the well-known values.
func (r ServerRole) IsValid() bool {
	return r == ServerRolePrimary || r == ServerRoleStandby
}

// Server 表示一行 northbound_servers 表记录。
//
// 关联：
//   - DDL：migrations/000065_northbound_servers.sql
//   - seed：migrations/seed/000066_seed_northbound_servers.sql
//   - 业务说明：F08 北向/OSS 接口主备配置；is_active 全表唯一为 true 由 partial
//     unique index 保证。
type Server struct {
	ID          uuid.UUID  `json:"id"`
	Role        ServerRole `json:"role"`
	Host        string     `json:"host"`
	Port        int        `json:"port"`
	Description string     `json:"description"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// SwitchActiveRequest 是 PUT /admin/northbound/servers/active 的请求体。
type SwitchActiveRequest struct {
	Role ServerRole `json:"role" binding:"required"`
}

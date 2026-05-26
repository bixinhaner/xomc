// Package querytemplate provides persistence and REST handlers for the
// pm_query_templates table (T-0174 阶段 1).
//
// 模板用于保存"指标查询"页面的可复用查询表单状态：
//   - 设备 SN 列表 + 指标路径列表
//   - 粒度 (15min / hourly / daily / weekly / monthly)
//   - 时间窗预设 + 自定义起止
//   - 其它前端可序列化的查询表单状态
//
// 可见性：
//   - visibility=public  公共模板，仅 super_admin 可写
//   - visibility=private 私有模板，仅创建者可写
package querytemplate

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Visibility 是模板的可见性，只允许 public / private 两个值。
type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

// Template 是 pm_query_templates 表的一行业务模型。
type Template struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Visibility  Visibility `json:"visibility"`
	CreatorID   uuid.UUID  `json:"creator_id"`
	Description string     `json:"description,omitempty"`
	Payload     []byte     `json:"-"`       // raw JSONB；序列化由 handler 处理
	PayloadJSON any        `json:"payload"` // handler 层临时填充用于 JSON 输出
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateRequest 是 POST 入参。
type CreateRequest struct {
	Name        string
	Visibility  Visibility
	CreatorID   uuid.UUID
	Description string
	Payload     []byte // 已 marshal 好的 JSONB bytes
}

// UpdateRequest 是 PATCH 入参；nil 字段不动。
type UpdateRequest struct {
	Name        *string
	Description *string
	Payload     []byte // nil 表示不动；非 nil 表示覆盖
	Visibility  *Visibility
}

// ListFilter 是 List 端点的过滤条件。
type ListFilter struct {
	// CallerID 是当前登录用户 ID。私有模板可见性按此过滤（creator_id = CallerID）。
	CallerID uuid.UUID
	// CallerIsSuperAdmin 决定能否看到全部 public + 全部 private（true）
	// 还是只能看到 public + 自己的 private（false）。
	CallerIsSuperAdmin bool

	// Visibility nil = 不过滤；指定则只取该可见性。
	Visibility *Visibility
	// Search 模糊匹配 name（ILIKE）。
	Search string

	Page     int // 1-based；< 1 视为 1
	PageSize int // < 1 视为 50；> 200 视为 200
}

// Errors
var (
	ErrNotFound  = errors.New("querytemplate: not found")
	ErrForbidden = errors.New("querytemplate: forbidden")
	ErrDuplicate = errors.New("querytemplate: duplicate name for creator")
)

// ValidVisibility 校验 visibility 字符串是否合法。
func ValidVisibility(v string) bool {
	switch Visibility(v) {
	case VisibilityPublic, VisibilityPrivate:
		return true
	}
	return false
}

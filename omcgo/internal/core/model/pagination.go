package model

// ListRequest 封装分页和排序参数，由 Gin 自动绑定自 URL Query String。
// 使用场景：所有列表类接口的请求结构体嵌入此字段，如 DeviceListRequest{ ListRequest ... }。
type ListRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=1000"`
	SortBy   string `form:"sort_by"`
	SortDir  string `form:"sort_dir" binding:"omitempty,oneof=asc desc"`
}

// DefaultListRequest returns a ListRequest with sensible defaults.
func DefaultListRequest() ListRequest {
	return ListRequest{Page: 1, PageSize: 20, SortDir: "desc"}
}

// Offset returns the SQL OFFSET value.
func (r *ListRequest) Offset() int {
	if r.Page < 1 {
		r.Page = 1
	}
	return (r.Page - 1) * r.PageSize
}

// Limit returns the SQL LIMIT value.
func (r *ListRequest) Limit() int {
	if r.PageSize < 1 {
		r.PageSize = 20
	}
	if r.PageSize > 1000 {
		r.PageSize = 1000
	}
	return r.PageSize
}

// ListResponse 是分页列表接口的通用返回体。
// 使用范例：
type ListResponse[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NewListResponse 创建分页列表响应并自动计算总页数。
// 使用场景：所有列表 Handler 的最后一行，封装查询结果并返回给前端。
// total 为 DB 查询返回的总记录数，items 为当前页数据，不要传入 nil。
func NewListResponse[T any](items []T, total int64, page, pageSize int) *ListResponse[T] {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return &ListResponse[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

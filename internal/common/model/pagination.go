package model

// ListRequest contains pagination and sorting parameters.
type ListRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
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
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	return r.PageSize
}

// ListResponse is a generic paginated response.
type ListResponse[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NewListResponse creates a ListResponse computing TotalPages.
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

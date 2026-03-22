package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DefaultListRequest(t *testing.T) {
	r := DefaultListRequest()

	assert.Equal(t, 1, r.Page)
	assert.Equal(t, 20, r.PageSize)
	assert.Equal(t, "desc", r.SortDir)
	assert.Equal(t, "", r.SortBy)
}

func Test_ListRequest_Offset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		want     int
	}{
		{"page 1 offset is 0", 1, 20, 0},
		{"page 2 offset is 20", 2, 20, 20},
		{"page 3 with size 10", 3, 10, 20},
		{"page 5 with size 50", 5, 50, 200},
		{"page 0 normalizes to page 1", 0, 20, 0},
		{"negative page normalizes to page 1", -1, 20, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ListRequest{Page: tt.page, PageSize: tt.pageSize}
			assert.Equal(t, tt.want, r.Offset())
		})
	}
}

func Test_ListRequest_Limit(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int
		want     int
	}{
		{"normal size", 20, 20},
		{"size 1", 1, 1},
		{"size 100", 100, 100},
		{"zero defaults to 20", 0, 20},
		{"negative defaults to 20", -5, 20},
		{"over 100 capped to 100", 150, 100},
		{"size 101 capped to 100", 101, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ListRequest{PageSize: tt.pageSize}
			assert.Equal(t, tt.want, r.Limit())
		})
	}
}

func Test_ListRequest_Limit_MutatesPageSize(t *testing.T) {
	// Calling Limit() on an out-of-range PageSize should normalize the field
	r := ListRequest{PageSize: 0}
	_ = r.Limit()
	assert.Equal(t, 20, r.PageSize, "PageSize should be normalized after Limit()")

	r2 := ListRequest{PageSize: 200}
	_ = r2.Limit()
	assert.Equal(t, 100, r2.PageSize, "PageSize should be capped after Limit()")
}

func Test_NewListResponse(t *testing.T) {
	tests := []struct {
		name       string
		items      []string
		total      int64
		page       int
		pageSize   int
		wantPages  int
		wantLen    int
	}{
		{
			name:      "exact division",
			items:     []string{"a", "b"},
			total:     10,
			page:      1,
			pageSize:  5,
			wantPages: 2,
			wantLen:   2,
		},
		{
			name:      "remainder adds a page",
			items:     []string{"a"},
			total:     11,
			page:      1,
			pageSize:  5,
			wantPages: 3,
			wantLen:   1,
		},
		{
			name:      "single item single page",
			items:     []string{"a"},
			total:     1,
			page:      1,
			pageSize:  20,
			wantPages: 1,
			wantLen:   1,
		},
		{
			name:      "empty result",
			items:     []string{},
			total:     0,
			page:      1,
			pageSize:  20,
			wantPages: 0,
			wantLen:   0,
		},
		{
			name:      "nil items",
			items:     nil,
			total:     0,
			page:      1,
			pageSize:  20,
			wantPages: 0,
			wantLen:   0,
		},
		{
			name:      "large total",
			items:     []string{"a"},
			total:     10000,
			page:      500,
			pageSize:  20,
			wantPages: 500,
			wantLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewListResponse(tt.items, tt.total, tt.page, tt.pageSize)

			assert.Equal(t, tt.total, resp.Total)
			assert.Equal(t, tt.page, resp.Page)
			assert.Equal(t, tt.pageSize, resp.PageSize)
			assert.Equal(t, tt.wantPages, resp.TotalPages)
			assert.Len(t, resp.Items, tt.wantLen)
		})
	}
}

func Test_NewListResponse_WithStructType(t *testing.T) {
	type item struct {
		ID   int
		Name string
	}

	items := []item{{1, "first"}, {2, "second"}}
	resp := NewListResponse(items, 50, 1, 20)

	assert.Equal(t, int64(50), resp.Total)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "first", resp.Items[0].Name)
	assert.Equal(t, 3, resp.TotalPages)
}

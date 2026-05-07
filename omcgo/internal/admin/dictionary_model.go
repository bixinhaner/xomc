package admin

import (
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

// Dictionary represents a dictionary entry for managing enumeration values.
type Dictionary struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Status      bool      `json:"status"`
	Description string    `json:"desc"`
	Details     []DictionaryDetail `json:"sysDictionaryDetails,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DictionaryDetail represents a single option within a dictionary.
//
// PRD docs/prd/system/data-dictionary.md §10（v0.2 添加子项）：
//   - ParentID: NULL = 顶层；非 NULL = 子项（自引用 FK，删父级联子）
//   - Level:    0=顶层 / 1=一级子 / 2=二级子；最大深度 3 层（应用层 enforce）
type DictionaryDetail struct {
	ID              int64     `json:"id"`
	Label           string    `json:"label"`
	Value           string    `json:"value"`
	Extend          string    `json:"extend"`
	Status          bool      `json:"status"`
	Sort            int       `json:"sort"`
	SysDictionaryID int64     `json:"sysDictionaryId"`
	ParentID        *int64    `json:"parent_id,omitempty"`
	Level           int       `json:"level"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// MaxDictionaryDetailDepth 字典明细最大深度（PRD §10.6 Q1）。
// level 取值 0/1/2 → 共 3 层。
const MaxDictionaryDetailDepth = 3

// CreateDictionaryRequest is the input for creating a new dictionary.
type CreateDictionaryRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Status      *bool  `json:"status"`
	Description string `json:"desc"`
}

// UpdateDictionaryRequest is the input for updating a dictionary.
type UpdateDictionaryRequest struct {
	ID          int64   `json:"id" binding:"required"`
	Name        *string `json:"name"`
	Type        *string `json:"type"`
	Status      *bool   `json:"status"`
	Description *string `json:"desc"`
}

// DeleteDictionaryRequest is the input for deleting a dictionary.
type DeleteDictionaryRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// FindDictionaryRequest is the input for finding a dictionary by type.
type FindDictionaryRequest struct {
	Type string `form:"type" binding:"required"`
}

// DictionaryListRequest is the input for listing dictionaries.
type DictionaryListRequest struct {
	model.ListRequest
}

// CreateDictionaryDetailRequest is the input for creating a dictionary detail.
type CreateDictionaryDetailRequest struct {
	Label           string `json:"label" binding:"required"`
	Value           string `json:"value" binding:"required"`
	Extend          string `json:"extend"`
	Status          *bool  `json:"status"`
	Sort            *int   `json:"sort"`
	SysDictionaryID int64  `json:"sysDictionaryId" binding:"required"`
	// PRD §10：可选父明细 ID。为 nil 时插入顶层项（level=0）。
	ParentID *int64 `json:"parent_id"`
}

// UpdateDictionaryDetailRequest is the input for updating a dictionary detail.
type UpdateDictionaryDetailRequest struct {
	ID              int64   `json:"id" binding:"required"`
	Label           *string `json:"label"`
	Value           *string `json:"value"`
	Extend          *string `json:"extend"`
	Status          *bool   `json:"status"`
	Sort            *int    `json:"sort"`
	SysDictionaryID *int64  `json:"sysDictionaryId"`
	// PRD §10：可改父明细。Set 为指针类型表示是否变更：
	//   nil           → 不改 parent_id
	//   非 nil + 内层 nil → 显式改成顶层（不支持，请求体里只能用 number 或不传）
	//   非 nil + 内层非 nil → 改成该 id 的子项
	// 实际语义：使用 *int64（仅当请求里包含 parent_id 字段时才动）。
	ParentID *int64 `json:"parent_id"`
}

// DeleteDictionaryDetailRequest is the input for deleting a dictionary detail.
type DeleteDictionaryDetailRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// FindDictionaryDetailRequest is the input for finding a dictionary detail by ID.
type FindDictionaryDetailRequest struct {
	ID int64 `form:"id" binding:"required"`
}

// DictionaryDetailListRequest is the input for listing dictionary details with filtering.
type DictionaryDetailListRequest struct {
	Label           *string `form:"label"`
	Value           *string `form:"value"`
	Status          *bool   `form:"status"`
	SysDictionaryID *int64  `form:"sysDictionaryId"`
	model.ListRequest
}

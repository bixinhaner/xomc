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
type DictionaryDetail struct {
	ID               int64     `json:"id"`
	Label            string    `json:"label"`
	Value            string    `json:"value"`
	Extend           string    `json:"extend"`
	Status           bool      `json:"status"`
	Sort             int       `json:"sort"`
	SysDictionaryID  int64     `json:"sysDictionaryId"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

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

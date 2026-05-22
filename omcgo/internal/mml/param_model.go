package mml

import (
	"github.com/google/uuid"
)

// CommandGroup represents a hierarchical group of parameters.
type CommandGroup struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	GroupCode            string     `json:"group_code" db:"group_code"`
	GroupNameZh          string     `json:"group_name_zh" db:"group_name_zh"`
	GroupNameEn          string     `json:"group_name_en" db:"group_name_en"`
	ParentID             *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	Level                int        `json:"level" db:"level"`
	IsListable           bool       `json:"is_listable" db:"is_listable"`
	IsModifiable         bool       `json:"is_modifiable" db:"is_modifiable"`
	IsAddable            bool       `json:"is_addable" db:"is_addable"`
	IsRemovable          bool       `json:"is_removable" db:"is_removable"`
	AddObjectPath        *string    `json:"add_object_path,omitempty" db:"add_object_path"`
	DeleteObjectPath     *string    `json:"delete_object_path,omitempty" db:"delete_object_path"`
	ParamVersion         string     `json:"param_version" db:"param_version"`
	PlatformSupport      []string   `json:"platform_support" db:"platform_support"`
	MobileSupport        bool       `json:"mobile_support" db:"mobile_support"`
	BroadbandSupport     bool       `json:"broadband_support" db:"broadband_support"`
	CellNumber           int        `json:"cell_number" db:"cell_number"`
	CellIndexLocation    int        `json:"cell_index_location" db:"cell_index_location"`
	RequireSecondConfirm bool       `json:"require_second_confirm" db:"require_second_confirm"`
	ConfirmMessageZh     *string    `json:"confirm_message_zh,omitempty" db:"confirm_message_zh"`
	ConfirmMessageEn     *string    `json:"confirm_message_en,omitempty" db:"confirm_message_en"`
	DisplayOrder         int        `json:"display_order" db:"display_order"`

	// T-0123-P0 catalog 管理元数据（migration 000095）
	Source           string `json:"source" db:"source"`                       // standard / admin
	CatalogProtected bool   `json:"catalog_protected" db:"catalog_protected"` // standard 行不可删除

	Children []CommandGroup `json:"children,omitempty"`
}

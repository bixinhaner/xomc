package mml

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ParamVersion represents a parameter library version (e.g. QB1.0, MLN1.0).
type ParamVersion struct {
	VersionCode     string    `json:"version_code" db:"version_code"`
	VersionName     string    `json:"version_name" db:"version_name"`
	Description     string    `json:"description" db:"description"`
	ReleaseDate     *time.Time `json:"release_date,omitempty" db:"release_date"`
	ProductModels   []string  `json:"product_models" db:"product_models"`
	SoftwareVersions []string  `json:"software_versions" db:"software_versions"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	IsDeprecated    bool      `json:"is_deprecated" db:"is_deprecated"`
	GroupCount      int       `json:"group_count" db:"group_count"`
	ParamCount      int       `json:"param_count" db:"param_count"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// ParamGroup represents a hierarchical group of parameters.
type ParamGroup struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	GroupCode             string     `json:"group_code" db:"group_code"`
	GroupNameZh           string     `json:"group_name_zh" db:"group_name_zh"`
	GroupNameEn           string     `json:"group_name_en" db:"group_name_en"`
	ParentID              *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	Level                 int        `json:"level" db:"level"`
	IsListable            bool       `json:"is_listable" db:"is_listable"`
	IsModifiable          bool       `json:"is_modifiable" db:"is_modifiable"`
	IsAddable             bool       `json:"is_addable" db:"is_addable"`
	IsRemovable           bool       `json:"is_removable" db:"is_removable"`
	AddObjectPath         *string    `json:"add_object_path,omitempty" db:"add_object_path"`
	DeleteObjectPath      *string    `json:"delete_object_path,omitempty" db:"delete_object_path"`
	ParamVersion          string     `json:"param_version" db:"param_version"`
	PlatformSupport       []string   `json:"platform_support" db:"platform_support"`
	MobileSupport         bool       `json:"mobile_support" db:"mobile_support"`
	BroadbandSupport      bool       `json:"broadband_support" db:"broadband_support"`
	CellNumber            int        `json:"cell_number" db:"cell_number"`
	CellIndexLocation     int        `json:"cell_index_location" db:"cell_index_location"`
	RequireSecondConfirm  bool       `json:"require_second_confirm" db:"require_second_confirm"`
	ConfirmMessageZh      *string    `json:"confirm_message_zh,omitempty" db:"confirm_message_zh"`
	ConfirmMessageEn      *string    `json:"confirm_message_en,omitempty" db:"confirm_message_en"`
	DisplayOrder          int        `json:"display_order" db:"display_order"`
	Children              []ParamGroup `json:"children,omitempty"`
}

// Param represents a single TR-069 parameter definition.
type Param struct {
	ID                   uuid.UUID       `json:"id" db:"id"`
	ParamCode            string          `json:"param_code" db:"param_code"`
	ParamNameZh          string          `json:"param_name_zh" db:"param_name_zh"`
	ParamNameEn          string          `json:"param_name_en" db:"param_name_en"`
	Tr069Path            string          `json:"tr069_path" db:"tr069_path"`
	ValueType            string          `json:"value_type" db:"value_type"`
	ValueConstraint      json.RawMessage `json:"value_constraint" db:"value_constraint"`
	DefaultValue         *string         `json:"default_value,omitempty" db:"default_value"`
	JsRegex              *string         `json:"js_regex,omitempty" db:"js_regex"`
	IsWritable           bool            `json:"is_writable" db:"is_writable"`
	IsListable           bool            `json:"is_listable" db:"is_listable"`
	IsModifiable         bool            `json:"is_modifiable" db:"is_modifiable"`
	IsAddable            bool            `json:"is_addable" db:"is_addable"`
	IsRemovable          bool            `json:"is_removable" db:"is_removable"`
	IsLeaf               bool            `json:"is_leaf" db:"is_leaf"`
	IsDynamic            bool            `json:"is_dynamic" db:"is_dynamic"`
	DisplayOrder         int             `json:"display_order" db:"display_order"`
	ParamVersion         string          `json:"param_version" db:"param_version"`
	SoftwareVersion      *string         `json:"software_version,omitempty" db:"software_version"`
	PlatformSupport      []string        `json:"platform_support" db:"platform_support"`
	MobileSupport        bool            `json:"mobile_support" db:"mobile_support"`
	BroadbandSupport     bool            `json:"broadband_support" db:"broadband_support"`
	Memo                 *string         `json:"memo,omitempty" db:"memo"`
	ExplanationZh        *string         `json:"explanation_zh,omitempty" db:"explanation_zh"`
	ExplanationEn        *string         `json:"explanation_en,omitempty" db:"explanation_en"`
	TitleZh              *string         `json:"title_zh,omitempty" db:"title_zh"`
	TitleEn              *string         `json:"title_en,omitempty" db:"title_en"`
	RequireSecondConfirm bool            `json:"require_second_confirm" db:"require_second_confirm"`
	ConfirmMessageZh     *string         `json:"confirm_message_zh,omitempty" db:"confirm_message_zh"`
	ConfirmMessageEn     *string         `json:"confirm_message_en,omitempty" db:"confirm_message_en"`
}

// ParamFilter specifies criteria for searching parameters.
type ParamFilter struct {
	VersionCode string
	GroupID     *uuid.UUID
	Search      *string
	Tr069Path   *string
}

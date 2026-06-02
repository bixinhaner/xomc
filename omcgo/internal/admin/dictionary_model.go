package admin

import (
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

// Dictionary represents a dictionary entry for managing enumeration values.
//
// T-0182 数据源绑定字段(下半段) — PRD docs/project/prd/F06-data-dictionary-source.md §3.2.1:
//   - SourceTable / SourceLabelField / SourceValueField 三个一组,nil = 手工字典;
//   - 非 nil 必须三个同时填写且通过 dictsource.Registry 白名单校验;
//   - LastRefresh* 由 SyncEngine 写入,UI 显示"上次同步:5 分钟前"等。
type Dictionary struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Status      bool      `json:"status"`
	Description string    `json:"desc"`
	// i18n JSONB 列 (migration 000003)。形态 {"zh-CN": "...", "en-US": "..."},
	// 前端按 appLocale 取;不存在时 fallback 到 Name/Description。
	NameI18n        map[string]string `json:"name_i18n,omitempty"`
	DescriptionI18n map[string]string `json:"description_i18n,omitempty"`
	Details     []DictionaryDetail `json:"sysDictionaryDetails,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// 数据源绑定 — T-0182
	SourceTable        *string    `json:"source_table,omitempty"`
	SourceLabelField   *string    `json:"source_label_field,omitempty"`
	SourceValueField   *string    `json:"source_value_field,omitempty"`
	LastRefreshAt      *time.Time `json:"last_refresh_at,omitempty"`
	LastRefreshStatus  *string    `json:"last_refresh_status,omitempty"`
	LastRefreshError   *string    `json:"last_refresh_error,omitempty"`
	LastRefreshCount   *int       `json:"last_refresh_count,omitempty"`
}

// DictionaryDetail represents a single option within a dictionary.
//
// PRD docs/prd/system/data-dictionary.md §10（v0.2 添加子项）：
//   - ParentID: NULL = 顶层；非 NULL = 子项（自引用 FK，删父级联子）
//   - Level:    0=顶层 / 1=一级子 / 2=二级子；最大深度 3 层（应用层 enforce）
type DictionaryDetail struct {
	ID              int64     `json:"id"`
	Label           string    `json:"label"`
	// LabelI18n: i18n JSONB 列 (migration 000003)。前端按 appLocale 取;不存在时 fallback 到 Label。
	LabelI18n       map[string]string `json:"label_i18n,omitempty"`
	Value           string    `json:"value"`
	Extend          string    `json:"extend"`
	Status          bool      `json:"status"`
	Sort            int       `json:"sort"`
	SysDictionaryID int64     `json:"sysDictionaryId"`
	ParentID        *int64    `json:"parent_id,omitempty"`
	Level           int       `json:"level"`
	// Origin manual=手工 / auto=数据源同步 (T-0182, PRD §3.2.1)。
	// 同步任务只动 auto 行,manual 行保留。
	Origin          string    `json:"origin"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// 字典项来源常量(对应 sys_dictionary_details.origin 列)。
const (
	OriginManual = "manual"
	OriginAuto   = "auto"
)

// MaxDictionaryDetailDepth 字典明细最大深度（PRD §10.6 Q1）。
// level 取值 0/1/2 → 共 3 层。
const MaxDictionaryDetailDepth = 3

// CreateDictionaryRequest is the input for creating a new dictionary.
//
// T-0182:支持绑定数据源。Source* 三个字段同时为空(手工字典)或同时填写(托管字典)。
type CreateDictionaryRequest struct {
	Name             string  `json:"name" binding:"required"`
	Type             string  `json:"type" binding:"required"`
	Status           *bool   `json:"status"`
	Description      string  `json:"desc"`
	SourceTable      *string `json:"source_table,omitempty"`
	SourceLabelField *string `json:"source_label_field,omitempty"`
	SourceValueField *string `json:"source_value_field,omitempty"`
}

// UpdateDictionaryRequest is the input for updating a dictionary.
//
// T-0182:Source* 三字段使用 **JSON 是否包含字段** 判定语义(Go 反序列化层面用 *string):
//   - 三个都为 nil (JSON 缺字段)              → 不动数据源绑定
//   - 三个 *string 都非 nil 且内层非空        → 绑定/切换数据源(切换会清空旧 auto 项)
//   - 三个 *string 都非 nil 且内层为空字符串  → 解绑(清空 auto 项,保留 manual 项)
//
// 任何"部分填写"组合(只填 1 个或 2 个)都被 service 层拒绝。
type UpdateDictionaryRequest struct {
	ID               int64   `json:"id" binding:"required"`
	Name             *string `json:"name"`
	Type             *string `json:"type"`
	Status           *bool   `json:"status"`
	Description      *string `json:"desc"`
	SourceTable      *string `json:"source_table,omitempty"`
	SourceLabelField *string `json:"source_label_field,omitempty"`
	SourceValueField *string `json:"source_value_field,omitempty"`
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
	// LabelI18n: 展示值多语言({"zh-CN":...,"en-US":...}),写入 label_i18n JSONB。
	LabelI18n       map[string]string `json:"label_i18n"`
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
	// LabelI18n: 展示值多语言;非 nil 时整体覆盖 label_i18n。
	LabelI18n       map[string]string `json:"label_i18n"`
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

// ====================================================================
// T-0182 数据字典数据源 — 新增请求/响应类型
// PRD docs/project/prd/F06-data-dictionary-source.md §3.5
// ====================================================================

// PreviewSourceRequest 是 GET /admin/sysDictionary/sources/preview 的 query 参数。
type PreviewSourceRequest struct {
	Table      string `form:"table" binding:"required"`
	LabelField string `form:"label" binding:"required"`
	ValueField string `form:"value" binding:"required"`
	Limit      int    `form:"limit"`
}

// PreviewSourceResponse 是 dry-run 端点的返回结构。
type PreviewSourceResponse struct {
	Rows  []PreviewSourceRow `json:"rows"`
	Total int                `json:"total"`
}

// PreviewSourceRow 是预览样本里的一对 (label, value)。
type PreviewSourceRow struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// RefreshSourceResponse 是手动刷新端点的返回结构。
type RefreshSourceResponse struct {
	Inserted  int `json:"inserted"`
	Updated   int `json:"updated"`
	Deleted   int `json:"deleted"`
	Total     int `json:"total"`
	DurationMS int64 `json:"duration_ms"`
}

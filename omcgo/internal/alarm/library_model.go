package alarm

import (
	"time"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// AlarmLibrary 告警库条目
type AlarmLibrary struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	AlarmCode     string          `json:"alarm_code" db:"alarm_code"`
	AlarmSource   string          `json:"alarm_source" db:"alarm_source"`
	EventType     string          `json:"event_type" db:"event_type"`
	Severity      int             `json:"severity" db:"severity"`
	Enabled       bool            `json:"enabled" db:"enabled"`
	ProbableCause string          `json:"probable_cause" db:"probable_cause"`
	Explanation   string          `json:"explanation,omitempty" db:"explanation"`
	AdditionalInfo map[string]any `json:"additional_info,omitempty" db:"additional_info"`
	Carrier       *string         `json:"carrier,omitempty" db:"carrier"`
	Technology    *string         `json:"technology,omitempty" db:"technology"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// AlarmLibraryI18n 告警库国际化
type AlarmLibraryI18n struct {
	ID            uuid.UUID `json:"id" db:"id"`
	LibraryID     uuid.UUID `json:"library_id" db:"library_id"`
	Locale        string    `json:"locale" db:"locale"`
	ProbableCause string    `json:"probable_cause" db:"probable_cause"`
	Explanation   string    `json:"explanation,omitempty" db:"explanation"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// AlarmLibraryFilter 告警库查询过滤
type AlarmLibraryFilter struct {
	AlarmCode   *string
	AlarmSource *string
	Severity    *int
	Enabled     *bool
	Carrier     *string
	EventType   *string
	Keyword     *string
	model.ListRequest
}

// CreateAlarmLibraryRequest 创建告警库请求
type CreateAlarmLibraryRequest struct {
	AlarmCode     string          `json:"alarm_code" binding:"required"`
	AlarmSource   string          `json:"alarm_source" binding:"required"`
	EventType     string          `json:"event_type" binding:"required"`
	Severity      int             `json:"severity" binding:"required,min=1,max=4"`
	ProbableCause string          `json:"probable_cause" binding:"required"`
	Explanation   string          `json:"explanation"`
	AdditionalInfo map[string]any `json:"additional_info"`
	Carrier       string          `json:"carrier"`
	Technology    string          `json:"technology"`
	Enabled       *bool           `json:"enabled"`
}

// UpdateAlarmLibraryRequest 更新告警库请求
type UpdateAlarmLibraryRequest struct {
	Severity      *int            `json:"severity" binding:"omitempty,min=1,max=4"`
	Enabled       *bool           `json:"enabled"`
	ProbableCause *string         `json:"probable_cause"`
	Explanation   *string         `json:"explanation"`
	AdditionalInfo map[string]any `json:"additional_info"`
	Carrier       *string         `json:"carrier"`
	Technology    *string         `json:"technology"`
}

// CreateAlarmLibraryI18nRequest 创建国际化请求
type CreateAlarmLibraryI18nRequest struct {
	Locale        string `json:"locale" binding:"required"`
	ProbableCause string `json:"probable_cause" binding:"required"`
	Explanation   string `json:"explanation"`
}
package alarm

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// AlarmRule represents a configurable alarm rule.
type AlarmRule struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	AlarmCode       string          `json:"alarm_code,omitempty"`
	Severity        int             `json:"severity"`
	ConditionType   string          `json:"condition_type"`
	ConditionConfig json.RawMessage `json:"condition_config"`
	ActionType      string          `json:"action_type"`
	ActionConfig    json.RawMessage `json:"action_config,omitempty"`
	Carrier         string          `json:"carrier,omitempty"`
	Technology      string          `json:"technology,omitempty"`
	Enabled         bool            `json:"enabled"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// AlarmRuleFilter specifies criteria for listing alarm rules.
type AlarmRuleFilter struct {
	Carrier       *string
	Technology    *string
	Enabled       *bool
	AlarmCode     *string
	ConditionType *string
	model.ListRequest
}

// CreateAlarmRuleRequest is the JSON body for creating an alarm rule.
type CreateAlarmRuleRequest struct {
	Name            string          `json:"name" binding:"required"`
	Description     string          `json:"description"`
	AlarmCode       string          `json:"alarm_code"`
	Severity        int             `json:"severity"`
	ConditionType   string          `json:"condition_type" binding:"required"`
	ConditionConfig json.RawMessage `json:"condition_config" binding:"required"`
	ActionType      string          `json:"action_type" binding:"required"`
	ActionConfig    json.RawMessage `json:"action_config"`
	Carrier         string          `json:"carrier"`
	Technology      string          `json:"technology"`
	Enabled         *bool           `json:"enabled"`
}

// UpdateAlarmRuleRequest is the JSON body for updating an alarm rule.
type UpdateAlarmRuleRequest struct {
	Name            *string          `json:"name"`
	Description     *string          `json:"description"`
	AlarmCode       *string          `json:"alarm_code"`
	Severity        *int             `json:"severity"`
	ConditionType   *string          `json:"condition_type"`
	ConditionConfig *json.RawMessage `json:"condition_config"`
	ActionType      *string          `json:"action_type"`
	ActionConfig    *json.RawMessage `json:"action_config"`
	Carrier         *string          `json:"carrier"`
	Technology      *string          `json:"technology"`
	Enabled         *bool            `json:"enabled"`
}

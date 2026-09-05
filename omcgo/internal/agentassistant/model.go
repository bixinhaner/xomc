// Package agentassistant owns the business-side assistant lifecycle. The remote
// runtime receives immutable execution snapshots, never ownership or ACL state.
package agentassistant

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Capability struct {
	OperationID  string `json:"operationId"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Path         string `json:"path"`
	DeviceScoped bool   `json:"deviceScoped"`
}
type EventCapability struct {
	Type   string   `json:"type"`
	Title  string   `json:"title"`
	Fields []string `json:"fields"`
}

var Events = []EventCapability{
	{Type: "omc.alarm.severe-raised.v1", Title: "重要告警出现 / Important alarm raised", Fields: []string{"severity", "alarmType"}},
	{Type: "omc.task.failed.v1", Title: "设备任务失败 / Device task failed", Fields: []string{"taskType", "errorCode", "retryCount"}},
}

type Condition struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}
type Trigger struct {
	Kind            string      `json:"kind"`
	IntervalMinutes int         `json:"intervalMinutes,omitempty"`
	Time            string      `json:"time,omitempty"`
	Timezone        string      `json:"timezone,omitempty"`
	Weekdays        []int       `json:"weekdays,omitempty"`
	EventType       string      `json:"eventType,omitempty"`
	Conditions      []Condition `json:"conditions"`
}
type Scope struct {
	Kind     string `json:"kind"`
	DeviceID string `json:"deviceId,omitempty"`
	Label    string `json:"label,omitempty"`
}
type Definition struct {
	Name            string   `json:"name"`
	Goal            string   `json:"goal"`
	Scope           Scope    `json:"scope"`
	Trigger         Trigger  `json:"trigger"`
	Operations      []string `json:"operations"`
	Notify          string   `json:"notify"`
	CooldownMinutes int      `json:"cooldownMinutes"`
}
type Message struct {
	Role string `json:"role"`
	Text string `json:"text"`
}
type PlanRequest struct {
	Message        string            `json:"message"`
	Definition     *Definition       `json:"definition"`
	Messages       []Message         `json:"messages"`
	Capabilities   []Capability      `json:"capabilities"`
	Events         []EventCapability `json:"events"`
	Locale         string            `json:"locale"`
	Timezone       string            `json:"timezone"`
	ExternalUserID string            `json:"externalUserId"`
}
type PlanResponse struct {
	Reply               string      `json:"reply"`
	Readiness           string      `json:"readiness"`
	Questions           []string    `json:"questions"`
	MissingCapabilities []string    `json:"missingCapabilities"`
	Definition          *Definition `json:"definition"`
}
type Assistant struct {
	ID                  string      `json:"id"`
	Locale              string      `json:"locale"`
	Timezone            string      `json:"timezone"`
	OwnerID             string      `json:"-"`
	RoleID              string      `json:"-"`
	Revision            int         `json:"revision"`
	Definition          *Definition `json:"definition"`
	Messages            []Message   `json:"messages"`
	Readiness           string      `json:"readiness"`
	Questions           []string    `json:"questions"`
	MissingCapabilities []string    `json:"missingCapabilities"`
	State               string      `json:"state"`
	PublishedRevision   *int        `json:"publishedRevision"`
	PublishedDefinition *Definition `json:"publishedDefinition,omitempty"`
	PublishedAt         *time.Time  `json:"publishedAt,omitempty"`
	NextRunAt           *time.Time  `json:"nextRunAt"`
	LastRunAt           *time.Time  `json:"lastRunAt"`
	LastError           string      `json:"lastError,omitempty"`
	CreatedAt           time.Time   `json:"createdAt"`
	UpdatedAt           time.Time   `json:"updatedAt"`
}
type Principal struct {
	UserID      string `json:"userId"`
	RoleID      string `json:"roleId"`
	ScopeDigest string `json:"scopeDigest,omitempty"`
}
type Limits struct {
	TimeoutSeconds int `json:"timeoutSeconds"`
	MaxToolCalls   int `json:"maxToolCalls"`
	MaxOutputBytes int `json:"maxOutputBytes"`
}
type ExecutionRequest struct {
	ContractVersion  string         `json:"contractVersion"`
	RunID            string         `json:"runId"`
	AssistantID      string         `json:"assistantId"`
	Revision         int            `json:"revision"`
	Definition       Definition     `json:"definition"`
	DefinitionDigest string         `json:"definitionDigest"`
	HandbookDigest   string         `json:"handbookDigest"`
	APIHandbook      map[string]any `json:"apiHandbook"`
	Locale           string         `json:"locale"`
	Timezone         string         `json:"timezone"`
	ExternalUserID   string         `json:"externalUserId"`
	TriggerContext   map[string]any `json:"triggerContext"`
	Limits           Limits         `json:"limits"`
}
type Fact struct {
	Text         string   `json:"text"`
	EvidenceRefs []string `json:"evidenceRefs"`
}
type Result struct {
	Outcome    string   `json:"outcome"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Facts      []Fact   `json:"facts"`
	Hypotheses []string `json:"hypotheses"`
	NextSteps  []string `json:"nextSteps"`
}
type RemoteError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}
type ToolProgress struct {
	ID          string    `json:"id"`
	OperationID string    `json:"operationId"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}
type RemoteRun struct {
	ID          string         `json:"id"`
	Status      string         `json:"status"`
	Output      *Result        `json:"output"`
	Error       *RemoteError   `json:"error"`
	StartedAt   *time.Time     `json:"startedAt"`
	CompletedAt *time.Time     `json:"completedAt"`
	Tools       []ToolProgress `json:"tools"`
}
type Run struct {
	ID            string           `json:"id"`
	AssistantID   string           `json:"assistantId"`
	OwnerID       string           `json:"-"`
	Revision      int              `json:"revision"`
	Kind          string           `json:"kind"`
	Status        string           `json:"status"`
	ConnectorID   string           `json:"-"`
	Request       ExecutionRequest `json:"-"`
	Principal     Principal        `json:"-"`
	Output        *Result          `json:"output"`
	ErrorCode     string           `json:"errorCode,omitempty"`
	ErrorMessage  string           `json:"errorMessage,omitempty"`
	Tools         []ToolProgress   `json:"tools"`
	Attempts      int              `json:"attempts"`
	LeaseToken    string           `json:"-"`
	CreatedAt     time.Time        `json:"createdAt"`
	StartedAt     *time.Time       `json:"startedAt"`
	CompletedAt   *time.Time       `json:"completedAt"`
	ReadAt        *time.Time       `json:"readAt"`
	NotifyVisible bool             `json:"notifyVisible"`
}
type Event struct {
	OccurredAt time.Time
	ID         string
	Type       string
	Data       map[string]any
	DeviceID   string
	Context    map[string]any
}

func ValidateDefinition(d *Definition, capabilities []Capability) error {
	if d == nil || strings.TrimSpace(d.Name) == "" || len([]rune(d.Name)) > 100 || strings.TrimSpace(d.Goal) == "" || len(d.Goal) > 32000 {
		return fmt.Errorf("ASSISTANT_INVALID_DEFINITION")
	}
	// Keep the JSON array shape consistent with Studio and the browser.
	if d.Trigger.Conditions == nil {
		d.Trigger.Conditions = []Condition{}
	}
	if d.Notify != "always" && d.Notify != "findings" {
		return fmt.Errorf("ASSISTANT_INVALID_DELIVERY")
	}
	if d.CooldownMinutes < 0 || d.CooldownMinutes > 10080 {
		return fmt.Errorf("ASSISTANT_INVALID_COOLDOWN")
	}
	if d.Scope.Kind != "visible" && d.Scope.Kind != "device" {
		return fmt.Errorf("ASSISTANT_INVALID_SCOPE")
	}
	if d.Scope.Kind == "device" {
		if _, err := uuid.Parse(d.Scope.DeviceID); err != nil {
			return fmt.Errorf("ASSISTANT_SCOPE_NOT_RESOLVED")
		}
	}
	if len(d.Operations) == 0 || len(d.Operations) > 24 {
		return fmt.Errorf("ASSISTANT_CAPABILITY_REQUIRED")
	}
	catalog := map[string]Capability{}
	for _, c := range capabilities {
		catalog[c.OperationID] = c
	}
	seen := map[string]bool{}
	for _, id := range d.Operations {
		c, ok := catalog[id]
		if !ok || seen[id] {
			return fmt.Errorf("ASSISTANT_UNKNOWN_CAPABILITY: %s", id)
		}
		seen[id] = true
		if d.Scope.Kind == "device" && !c.DeviceScoped {
			return fmt.Errorf("ASSISTANT_CAPABILITY_SCOPE_MISMATCH: %s", id)
		}
	}
	t := d.Trigger
	switch t.Kind {
	case "manual":
	case "interval":
		if t.IntervalMinutes < 5 || t.IntervalMinutes > 10080 {
			return fmt.Errorf("ASSISTANT_INVALID_SCHEDULE")
		}
	case "schedule":
		if _, err := time.Parse("15:04", t.Time); err != nil {
			return fmt.Errorf("ASSISTANT_INVALID_SCHEDULE")
		}
		if _, err := time.LoadLocation(t.Timezone); err != nil {
			return fmt.Errorf("ASSISTANT_INVALID_TIMEZONE")
		}
		if len(t.Weekdays) == 0 || len(t.Weekdays) > 7 {
			return fmt.Errorf("ASSISTANT_INVALID_SCHEDULE")
		}
		for _, day := range t.Weekdays {
			if day < 0 || day > 6 {
				return fmt.Errorf("ASSISTANT_INVALID_SCHEDULE")
			}
		}
	case "event":
		var e *EventCapability
		for i := range Events {
			if Events[i].Type == t.EventType {
				e = &Events[i]
			}
		}
		if e == nil || len(t.Conditions) > 10 {
			return fmt.Errorf("ASSISTANT_UNKNOWN_EVENT")
		}
		for _, c := range t.Conditions {
			if !slices.Contains(e.Fields, c.Field) || !slices.Contains([]string{"eq", "ne", "in", "gte", "lte"}, c.Op) {
				return fmt.Errorf("ASSISTANT_INVALID_CONDITION")
			}
			if c.Op == "gte" || c.Op == "lte" {
				if _, ok := number(c.Value); !ok {
					return fmt.Errorf("ASSISTANT_INVALID_CONDITION")
				}
			}
			if c.Op == "in" {
				if values, ok := c.Value.([]any); !ok || len(values) == 0 {
					return fmt.Errorf("ASSISTANT_INVALID_CONDITION")
				}
			}
			switch v := c.Value.(type) {
			case string, bool, float64, int, json.Number:
			case []any:
				if c.Op != "in" || len(v) > 20 {
					return fmt.Errorf("ASSISTANT_INVALID_CONDITION")
				}
				for _, x := range v {
					if _, ok := x.(string); !ok {
						return fmt.Errorf("ASSISTANT_INVALID_CONDITION")
					}
				}
			default:
				return fmt.Errorf("ASSISTANT_INVALID_CONDITION")
			}
		}
	default:
		return fmt.Errorf("ASSISTANT_INVALID_TRIGGER")
	}
	return nil
}

// NextRun coalesces missed ticks, uses the assistant's timezone, and returns a
// strictly future instant. Scanning real UTC minutes handles DST gaps and folds.
func NextRun(t Trigger, now time.Time) *time.Time {
	if t.Kind == "interval" && t.IntervalMinutes >= 5 {
		v := now.Add(time.Duration(t.IntervalMinutes) * time.Minute)
		return &v
	}
	if t.Kind != "schedule" {
		return nil
	}
	loc, err := time.LoadLocation(t.Timezone)
	if err != nil {
		return nil
	}
	start := now.Truncate(time.Minute).Add(time.Minute)
	for n := 0; n < 8*24*60; n++ {
		v := start.Add(time.Duration(n) * time.Minute)
		local := v.In(loc)
		if local.Format("2006-01-02") == now.In(loc).Format("2006-01-02") && t.Time <= now.In(loc).Format("15:04") {
			continue
		}
		if local.Format("15:04") == t.Time && slices.Contains(t.Weekdays, int(local.Weekday())) {
			return &v
		}
	}
	return nil
}
func DefinitionDigest(d Definition) string {
	b, _ := json.Marshal(d)
	return fmt.Sprintf("sha256:%x", sha256.Sum256(b))
}
func Matches(conditions []Condition, data map[string]any) bool {
	for _, c := range conditions {
		value, exists := data[c.Field]
		if !exists {
			return false
		}
		switch c.Op {
		case "eq":
			if !equalValue(value, c.Value) {
				return false
			}
		case "ne":
			if equalValue(value, c.Value) {
				return false
			}
		case "in":
			found := false
			switch values := c.Value.(type) {
			case []any:
				for _, v := range values {
					found = found || equalValue(value, v)
				}
			case []string:
				found = slices.Contains(values, fmt.Sprint(value))
			}
			if !found {
				return false
			}
		case "gte", "lte":
			a, aok := number(value)
			b, bok := number(c.Value)
			if !aok || !bok || (c.Op == "gte" && a < b) || (c.Op == "lte" && a > b) {
				return false
			}
		default:
			return false
		}
	}
	return true
}
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, e := n.Float64()
		return f, e == nil
	}
	return 0, false
}

func equalValue(a, b any) bool {
	if an, ok := number(a); ok {
		bn, bok := number(b)
		return bok && an == bn
	}
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	}
	return false
}

// ShouldNotify merges notifications for an assistant, not its execution records.
func ShouldNotify(kind, status string, outcome string, definition Definition, last *time.Time, now time.Time) bool {
	if kind == "trial" || status == "CANCELLED" {
		return false
	}
	if status != "FAILED" && (status != "COMPLETED" || (outcome == "no_change" && definition.Notify != "always")) {
		return false
	}
	return last == nil || !now.Before(last.Add(time.Duration(definition.CooldownMinutes)*time.Minute))
}

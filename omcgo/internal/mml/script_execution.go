package mml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// ErrScriptExecutionWarnings tells the HTTP layer that the script has only
// warnings and the caller must explicitly confirm them before execution.
var ErrScriptExecutionWarnings = errors.New("mml script execution warnings require confirmation")

// ScriptExecutionValidationError carries the line-level authoritative result
// back to the API boundary while preserving errors.Is(ErrInvalidInput) for
// callers that use the common domain error mapping.
type ScriptExecutionValidationError struct {
	Result   *ScriptValidationResult
	Warnings bool
	err      error
}

func (e *ScriptExecutionValidationError) Error() string {
	if e == nil || e.err == nil {
		return "mml script execution validation failed"
	}
	return e.err.Error()
}

func (e *ScriptExecutionValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// ScriptExecutionRequest is intentionally limited to scheduling and retry
// policy. Commands, device_sns and plan_items never cross this API boundary;
// they are copied from the stored imported-script snapshot.
type ScriptExecutionRequest struct {
	TaskName            string      `json:"task_name"`
	RequestID           string      `json:"request_id,omitempty"`
	ExecuteType         ExecuteType `json:"execute_type"`
	ScheduledAt         *string     `json:"scheduled_at"`
	PeriodStart         *string     `json:"period_start"`
	PeriodEnd           *string     `json:"period_end"`
	PeriodTime          string      `json:"period_time"`
	OfflineRetry        bool        `json:"offline_retry"`
	OfflineRetryWait    int         `json:"offline_retry_wait"`
	FailedRetry         bool        `json:"failed_retry"`
	FailedRetryCount    int         `json:"failed_retry_count"`
	FailedRetryInterval int         `json:"failed_retry_interval"`
	ConfirmWarnings     bool        `json:"confirm_warnings"`
}

// CreateScriptExecution creates an immutable task snapshot from an imported
// script. The stored plan is the only source of command/device data.
func (s *Service) CreateScriptExecution(ctx context.Context, scriptID uuid.UUID, actor string, req ScriptExecutionRequest) (*MMLTask, *ScriptValidationResult, error) {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return nil, nil, commonerrors.ErrUnauthorized
	}
	switch req.ExecuteType {
	case "", ExecuteImmediate, ExecuteScheduled, ExecutePeriodic, ExecuteSuspended:
		// empty means immediate, matching ExecuteCommand's internal default.
	default:
		return nil, nil, fmt.Errorf("unsupported execute_type %q: %w", req.ExecuteType, commonerrors.ErrInvalidInput)
	}
	if scriptID == uuid.Nil {
		return nil, nil, fmt.Errorf("script id is required: %w", commonerrors.ErrInvalidInput)
	}
	req.RequestID = strings.TrimSpace(req.RequestID)
	if s == nil || s.scriptRepo == nil || s.taskRepo == nil {
		return nil, nil, fmt.Errorf("script execution dependencies are unavailable: %w", commonerrors.ErrUnavailable)
	}
	if req.RequestID != "" {
		if existing, lookupErr := s.taskRepo.GetByRequestID(ctx, actor, req.RequestID); lookupErr == nil {
			return existing, nil, nil
		} else if !errors.Is(lookupErr, commonerrors.ErrNotFound) {
			return nil, nil, fmt.Errorf("check script execution request id: %w", lookupErr)
		}
	}
	script, err := s.scriptRepo.GetByID(ctx, scriptID)
	if err != nil {
		return nil, nil, fmt.Errorf("get script for execution: %w", err)
	}
	if script == nil {
		return nil, nil, commonerrors.ErrNotFound
	}
	if script.Creator != "" && script.Creator != actor {
		return nil, nil, commonerrors.ErrForbidden
	}
	plan := cloneScriptPlanItems(script.PlanItems)
	if len(plan) == 0 {
		return nil, nil, fmt.Errorf("script has no executable plan: %w", commonerrors.ErrInvalidInput)
	}

	validation := &ScriptValidationResult{PlanItems: cloneScriptPlanItems(plan)}
	if s.scriptExecutionValidator != nil {
		parsed := parsedScriptFromPlan(script, plan)
		validated, validateErr := s.scriptExecutionValidator.Validate(ctx, parsed, ValidationActor{Username: actor})
		if validateErr != nil {
			return nil, nil, fmt.Errorf("preflight script execution: %w", validateErr)
		}
		if validated != nil {
			validation = validated
			// The validator may return a fresh normalized plan, but execution
			// must still use the persisted snapshot, never a re-imported/client
			// plan. Keep only its dynamic line issues and summary.
			validation.PlanItems = cloneScriptPlanItems(plan)
		}
	} else {
		validation.Summary, validation.Issues = scriptValidationSnapshot(script)
	}
	if validation.Summary.ErrorCount == 0 {
		validation.Summary.ErrorCount = countIssueSeverity(validation.Issues, IssueError)
	}
	if validation.Summary.WarningCount == 0 {
		validation.Summary.WarningCount = countIssueSeverity(validation.Issues, IssueWarning)
	}
	if hasScriptErrors(validation.Issues) || validation.Summary.ErrorCount > 0 {
		return nil, validation, &ScriptExecutionValidationError{Result: validation, err: commonerrors.ErrInvalidInput}
	}
	if validation.Summary.WarningCount > 0 && !req.ConfirmWarnings {
		return nil, validation, &ScriptExecutionValidationError{Result: validation, Warnings: true, err: ErrScriptExecutionWarnings}
	}

	if strings.TrimSpace(req.TaskName) == "" {
		req.TaskName = script.ScriptName
	}
	scriptIDString := scriptID.String()
	execReq := ExecuteRequest{
		TaskName:                req.TaskName,
		Creator:                 actor,
		Executor:                actor,
		ScriptID:                &scriptIDString,
		ExecuteMode:             string(TaskExecuteModeDeviceBound),
		PlanItems:               cloneScriptPlanItems(plan),
		ExecuteType:             req.ExecuteType,
		ScheduledAt:             req.ScheduledAt,
		PeriodStart:             req.PeriodStart,
		PeriodEnd:               req.PeriodEnd,
		PeriodTime:              req.PeriodTime,
		OfflineRetry:            req.OfflineRetry,
		OfflineRetryWait:        req.OfflineRetryWait,
		FailedRetry:             req.FailedRetry,
		FailedRetryCount:        req.FailedRetryCount,
		FailedRetryInterval:     req.FailedRetryInterval,
		ScriptContentSHA256:     script.ContentSHA256,
		ScriptValidationVersion: script.ValidationVersion,
		PreservePlanSnapshot:    true,
		RequestID:               req.RequestID,
	}
	task, err := s.ExecuteCommand(ctx, execReq)
	if err != nil {
		if req.RequestID != "" && isUniqueViolation(err) {
			if existing, lookupErr := s.taskRepo.GetByRequestID(ctx, actor, req.RequestID); lookupErr == nil {
				return existing, validation, nil
			}
		}
		return nil, validation, err
	}
	return task, validation, nil
}

// preflightTask runs dynamic checks for a task just before scheduled/periodic
// fanout. A nil result means no validator is wired (legacy/common task).
func (s *Service) preflightTask(ctx context.Context, task *MMLTask) (*ScriptValidationResult, error) {
	if task == nil || task.ScriptID == nil || s.scriptExecutionValidator == nil {
		return nil, nil
	}
	plan := cloneScriptPlanItems(task.PlanItems)
	result, err := s.scriptExecutionValidator.Validate(ctx, parsedScriptFromTask(task), ValidationActor{Username: task.Creator})
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &ScriptValidationResult{}
	}
	result.PlanItems = plan
	result.Summary.ErrorCount = maxInt(result.Summary.ErrorCount, countIssueSeverity(result.Issues, IssueError))
	result.Summary.WarningCount = maxInt(result.Summary.WarningCount, countIssueSeverity(result.Issues, IssueWarning))
	return result, nil
}

func (s *Service) failPreflightTask(ctx context.Context, task *MMLTask, result *ScriptValidationResult, now time.Time) error {
	if task == nil {
		return nil
	}
	task.Status = TaskFailed
	task.Result = resultFailedPtr()
	task.FinishedAt = &now
	task.NextTriggerAt = nil
	task.FailedCount++
	task.Results = make([]map[string]interface{}, 0, len(result.Issues))
	for _, issue := range result.Issues {
		task.Results = append(task.Results, map[string]interface{}{
			"line_no": issue.LineNo, "code": issue.Code, "severity": issue.Severity,
			"raw_line": issue.RawLine, "field": issue.Field, "message": issue.Message,
		})
	}
	if s.taskRepo != nil {
		return s.taskRepo.Update(ctx, task)
	}
	return nil
}

func resultFailedPtr() *TaskResult {
	v := ResultFailed
	return &v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func cloneScriptPlanItems(items []MMLPlanItem) []MMLPlanItem {
	if items == nil {
		return []MMLPlanItem{}
	}
	out := make([]MMLPlanItem, len(items))
	for i, item := range items {
		out[i] = item
		out[i].Command = cloneAnyMap(item.Command)
		out[i].Parameters = cloneAnyMap(item.Parameters)
	}
	return out
}

func cloneAnyMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return cloneCommandMap(in)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return cloneCommandMap(in)
	}
	return out
}

func parsedScriptFromPlan(script *MMLScript, plan []MMLPlanItem) *ParsedScript {
	parsed := &ParsedScript{}
	if script != nil {
		parsed.NormalizedContent = script.Content
		parsed.SHA256 = script.ContentSHA256
	}
	parsed.Lines = make([]ParsedScriptLine, 0, len(plan))
	for _, item := range plan {
		code := item.CommandCode
		op := item.OperationType
		params := map[string]string{}
		if item.Command != nil {
			if v, ok := item.Command["command_code"].(string); ok && code == "" {
				code = v
			}
			if v, ok := item.Command["operation_type"].(string); ok && op == "" {
				op = v
			}
			if raw, ok := item.Command["parameters"].(map[string]interface{}); ok {
				for key, value := range raw {
					params[key] = fmt.Sprint(value)
				}
			}
		}
		parsed.Lines = append(parsed.Lines, ParsedScriptLine{LineNo: item.LineNo, RawLine: item.RawLine, DeviceSN: item.DeviceSN, Order: item.Order, CommandCode: code, OperationType: op, Parameters: params})
	}
	return parsed
}

func parsedScriptFromTask(task *MMLTask) *ParsedScript {
	return parsedScriptFromPlan(nil, task.PlanItems)
}

func scriptValidationSnapshot(script *MMLScript) (ScriptValidationSummary, []ScriptIssue) {
	if script == nil || script.ValidationSummary == nil {
		return ScriptValidationSummary{}, nil
	}
	var payload struct {
		Summary ScriptValidationSummary `json:"summary"`
		Issues  []ScriptIssue           `json:"issues"`
	}
	raw, err := json.Marshal(script.ValidationSummary)
	if err != nil || json.Unmarshal(raw, &payload) != nil {
		return ScriptValidationSummary{}, nil
	}
	return payload.Summary, payload.Issues
}

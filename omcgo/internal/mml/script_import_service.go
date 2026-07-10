package mml

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"go.uber.org/zap"
)

// ScriptImportValidationRunner keeps the import service independent from the
// concrete validator and makes the authority boundary explicit in tests.
type ScriptImportValidationRunner interface {
	Validate(ctx context.Context, parsed *ParsedScript, actor ValidationActor) (*ScriptValidationResult, error)
}

type ImportValidationResponse struct {
	ValidationToken   string                  `json:"validation_token,omitempty"`
	OriginalFilename  string                  `json:"original_filename"`
	NormalizedContent string                  `json:"normalized_content"`
	ContentSHA256     string                  `json:"content_sha256"`
	ValidationVersion string                  `json:"validation_version"`
	ValidatedAt       time.Time               `json:"validated_at"`
	PlanItems         []MMLPlanItem           `json:"plan_items"`
	Summary           ScriptValidationSummary `json:"summary"`
	Issues            []ScriptIssue           `json:"issues"`
}

// SaveImportedScriptRequest deliberately contains metadata and a bearer token
// only. Content, commands and plan_items are obtained from Redis's snapshot.
type SaveImportedScriptRequest struct {
	ValidationToken string   `json:"validation_token"`
	ScriptName      string   `json:"script_name"`
	Description     string   `json:"description"`
	Tags            []string `json:"tags"`
	RequestID       string   `json:"request_id,omitempty"`
}

type ReplaceImportedScriptRequest struct {
	ValidationToken   string    `json:"validation_token"`
	ScriptName        string    `json:"script_name"`
	Description       string    `json:"description"`
	Tags              []string  `json:"tags"`
	ExpectedUpdatedAt time.Time `json:"expected_updated_at"`
	RequestID         string    `json:"request_id,omitempty"`
}

type ScriptImportService struct {
	repo      ImportedScriptRepository
	validator ScriptImportValidationRunner
	sessions  ImportSessionStore
	logger    *zap.Logger
}

func NewScriptImportService(repo ImportedScriptRepository, validator ScriptImportValidationRunner, sessions ImportSessionStore, logger *zap.Logger) *ScriptImportService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ScriptImportService{repo: repo, validator: validator, sessions: sessions, logger: logger.Named("mml-script-import")}
}

func (s *ScriptImportService) ValidateScriptImport(ctx context.Context, username, filename string, raw []byte) (*ImportValidationResponse, error) {
	if strings.TrimSpace(username) == "" {
		return nil, fmt.Errorf("validate script import: %w", commonerrors.ErrUnauthorized)
	}
	parsed, parseIssues := ParseScriptTXT(raw)
	response := &ImportValidationResponse{OriginalFilename: strings.TrimSpace(filename), ValidationVersion: ValidationVersion, Issues: append([]ScriptIssue(nil), parseIssues...), PlanItems: []MMLPlanItem{}}
	if parsed == nil {
		response.Summary.ErrorCount = countIssueSeverity(response.Issues, IssueError)
		return response, nil
	}
	response.NormalizedContent, response.ContentSHA256 = parsed.NormalizedContent, parsed.SHA256
	result := &ScriptValidationResult{PlanItems: []MMLPlanItem{}, Issues: []ScriptIssue{}}
	if hasScriptErrors(parseIssues) {
		response.Summary.ErrorCount = countIssueSeverity(parseIssues, IssueError)
		response.Summary.WarningCount = countIssueSeverity(parseIssues, IssueWarning)
		return response, nil
	}
	if s.validator == nil {
		return nil, fmt.Errorf("validate script import: validator is nil")
	}
	validated, err := s.validator.Validate(ctx, parsed, ValidationActor{Username: username})
	if err != nil {
		return nil, fmt.Errorf("validate script import: %w", err)
	}
	if validated != nil {
		result = validated
		response.PlanItems = append(response.PlanItems, validated.PlanItems...)
		response.Summary = validated.Summary
		response.Issues = append(response.Issues, validated.Issues...)
	}
	response.Summary.ErrorCount += countIssueSeverity(parseIssues, IssueError)
	response.Summary.WarningCount += countIssueSeverity(parseIssues, IssueWarning)
	response.ValidatedAt = time.Now().UTC()
	if hasScriptErrors(response.Issues) {
		return response, nil
	}
	if s.sessions == nil {
		return nil, fmt.Errorf("validate script import: session store is nil")
	}
	token, err := s.sessions.Put(ctx, username, &ImportSession{OriginalFilename: response.OriginalFilename, NormalizedContent: response.NormalizedContent, ContentSHA256: response.ContentSHA256, ValidationVersion: response.ValidationVersion, ValidatedAt: response.ValidatedAt, Validation: *result})
	if err != nil {
		return nil, fmt.Errorf("store script import validation: %w", err)
	}
	response.ValidationToken = token
	return response, nil
}

func (s *ScriptImportService) CreateScriptFromImport(ctx context.Context, username string, req SaveImportedScriptRequest) (*MMLScript, error) {
	if err := validateImportMetadata(username, req.ValidationToken, req.ScriptName); err != nil {
		return nil, err
	}
	requestID := req.RequestID
	if requestID == "" {
		requestID = uuid.NewString()
	}
	session, err := s.sessions.Claim(ctx, req.ValidationToken, username, requestID)
	if err != nil {
		if errors.Is(err, ErrImportTokenConsumed) {
			if reader, ok := s.sessions.(ConsumedImportSessionReader); ok {
				if consumed, readErr := reader.GetConsumed(ctx, req.ValidationToken, username); readErr == nil {
					if existing, lookupErr := s.repo.GetByImportSessionID(ctx, consumed.ID); lookupErr == nil {
						return existing, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("claim script import: %w", err)
	}
	if hasScriptErrors(session.Validation.Issues) {
		_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
		return nil, fmt.Errorf("save script import: %w", commonerrors.ErrInvalidInput)
	}
	if existing, lookupErr := s.repo.GetByImportSessionID(ctx, session.ID); lookupErr == nil {
		if existing.Creator != username {
			_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
			return nil, commonerrors.ErrForbidden
		}
		if err := s.sessions.Finalize(ctx, req.ValidationToken, username, requestID); err != nil {
			return nil, fmt.Errorf("finalize replayed script import: %w", err)
		}
		return existing, nil
	} else if !errors.Is(lookupErr, commonerrors.ErrNotFound) {
		_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
		return nil, fmt.Errorf("check imported script: %w", lookupErr)
	}
	script := scriptFromImportSession(session, username, req.ScriptName, req.Description, req.Tags)
	if err := s.repo.CreateImported(ctx, script); err != nil {
		if existing, lookupErr := s.repo.GetByImportSessionID(ctx, session.ID); lookupErr == nil {
			if existing.Creator != username {
				_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
				return nil, commonerrors.ErrForbidden
			}
			if finalizeErr := s.sessions.Finalize(ctx, req.ValidationToken, username, requestID); finalizeErr != nil {
				return nil, fmt.Errorf("finalize replayed script import: %w", finalizeErr)
			}
			return existing, nil
		}
		_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
		return nil, fmt.Errorf("create imported script: %w", err)
	}
	if err := s.sessions.Finalize(ctx, req.ValidationToken, username, requestID); err != nil {
		return nil, fmt.Errorf("finalize script import: %w", err)
	}
	return script, nil
}

func (s *ScriptImportService) ReplaceScriptFromImport(ctx context.Context, id uuid.UUID, username string, req ReplaceImportedScriptRequest) (*MMLScript, error) {
	if err := validateImportMetadata(username, req.ValidationToken, req.ScriptName); err != nil {
		return nil, err
	}
	requestID := req.RequestID
	if requestID == "" {
		requestID = uuid.NewString()
	}
	session, err := s.sessions.Claim(ctx, req.ValidationToken, username, requestID)
	if err != nil {
		if errors.Is(err, ErrImportTokenConsumed) {
			if reader, ok := s.sessions.(ConsumedImportSessionReader); ok {
				if consumed, readErr := reader.GetConsumed(ctx, req.ValidationToken, username); readErr == nil {
					if existing, lookupErr := s.repo.GetByImportSessionID(ctx, consumed.ID); lookupErr == nil {
						if existing.ID != id || existing.Creator != username {
							return nil, commonerrors.ErrForbidden
						}
						return existing, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("claim script replacement: %w", err)
	}
	if hasScriptErrors(session.Validation.Issues) {
		_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
		return nil, fmt.Errorf("replace script import: %w", commonerrors.ErrInvalidInput)
	}
	if existing, lookupErr := s.repo.GetByImportSessionID(ctx, session.ID); lookupErr == nil {
		if existing.ID != id || existing.Creator != username {
			_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
			return nil, commonerrors.ErrForbidden
		}
		if finalizeErr := s.sessions.Finalize(ctx, req.ValidationToken, username, requestID); finalizeErr != nil {
			return nil, fmt.Errorf("finalize replayed script replacement: %w", finalizeErr)
		}
		return existing, nil
	} else if !errors.Is(lookupErr, commonerrors.ErrNotFound) {
		_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
		return nil, fmt.Errorf("check replacement import: %w", lookupErr)
	}
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
		return nil, fmt.Errorf("get script for replacement: %w", err)
	}
	if current.Creator != username {
		_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
		return nil, commonerrors.ErrForbidden
	}
	if req.ExpectedUpdatedAt.IsZero() {
		_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
		return nil, fmt.Errorf("replace imported script: %w", commonerrors.ErrInvalidInput)
	}
	replacement := scriptFromImportSession(session, username, req.ScriptName, req.Description, req.Tags)
	replacement.ID = current.ID
	replacement.Creator = current.Creator
	if err := s.repo.ReplaceImported(ctx, replacement, req.ExpectedUpdatedAt); err != nil {
		_ = s.sessions.Release(ctx, req.ValidationToken, username, requestID)
		return nil, fmt.Errorf("replace imported script: %w", err)
	}
	if err := s.sessions.Finalize(ctx, req.ValidationToken, username, requestID); err != nil {
		return nil, fmt.Errorf("finalize script replacement: %w", err)
	}
	return replacement, nil
}

func scriptFromImportSession(session *ImportSession, username, name, description string, tags []string) *MMLScript {
	now := session.ValidatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &MMLScript{ImportSessionID: session.ID, ScriptName: strings.TrimSpace(name), Description: description, Content: session.NormalizedContent, OriginalFilename: session.OriginalFilename, ContentSHA256: session.ContentSHA256, ValidationVersion: session.ValidationVersion, ValidatedAt: &now, PlanItems: append([]MMLPlanItem(nil), session.Validation.PlanItems...), ValidationSummary: JSONMap{"summary": session.Validation.Summary, "issues": session.Validation.Issues}, Creator: username, Tags: append([]string(nil), tags...), Status: ScriptActive, Type: ScriptTypeBatch}
}

func validateImportMetadata(username, token, name string) error {
	if strings.TrimSpace(username) == "" {
		return commonerrors.ErrUnauthorized
	}
	if strings.TrimSpace(token) == "" || strings.TrimSpace(name) == "" {
		return commonerrors.ErrInvalidInput
	}
	return nil
}

func hasScriptErrors(issues []ScriptIssue) bool { return countIssueSeverity(issues, IssueError) > 0 }
func countIssueSeverity(issues []ScriptIssue, severity IssueSeverity) int {
	n := 0
	for _, issue := range issues {
		if issue.Severity == severity {
			n++
		}
	}
	return n
}

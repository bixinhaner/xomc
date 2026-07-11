package mml

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
	"go.uber.org/zap"
)

// ScriptImportServiceAPI is the HTTP boundary for the two-phase TXT import.
// Keeping it narrow lets handler tests use a deterministic fake without a
// database or Redis.
type ScriptImportServiceAPI interface {
	ValidateScriptImport(ctx context.Context, username, filename string, raw []byte) (*ImportValidationResponse, error)
	CreateScriptFromImport(ctx context.Context, username string, req SaveImportedScriptRequest) (*MMLScript, error)
	ReplaceScriptFromImport(ctx context.Context, id uuid.UUID, username string, req ReplaceImportedScriptRequest) (*MMLScript, error)
}

const (
	scriptImportCodeMissingFile   = "MML_FILE_REQUIRED"
	scriptImportCodeTypeInvalid   = "MML_FILE_TYPE_INVALID"
	scriptImportCodeTooLarge      = "MML_FILE_TOO_LARGE"
	scriptImportCodeValidation    = "MML_SCRIPT_VALIDATION_FAILED"
	scriptImportCodeTokenExpired  = "MML_IMPORT_TOKEN_EXPIRED"
	scriptImportCodeTokenConsumed = "MML_IMPORT_TOKEN_CONSUMED"
	scriptImportCodeVersion       = "MML_SCRIPT_VERSION_CONFLICT"
	scriptImportCodeInternal      = "MML_SCRIPT_IMPORT_FAILED"
)

type scriptImportErrorPayload struct {
	Code   string        `json:"code"`
	Msg    string        `json:"message"`
	Issues []ScriptIssue `json:"issues,omitempty"`
}

func (h *Handler) SetScriptImportService(service ScriptImportServiceAPI) {
	h.scriptImportService = service
}

func (h *Handler) GetScriptImportTemplate(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="MMLTemplate.txt"`)
	c.Data(http.StatusOK, "text/plain; charset=utf-8", scriptImportTemplate)
}

func (h *Handler) ValidateScriptImport(c *gin.Context) {
	username, ok := authenticatedUsername(c)
	if !ok {
		h.writeScriptImportError(c, http.StatusUnauthorized, "MML_UNAUTHORIZED", "authentication required", nil)
		return
	}
	filename, raw, status, code, msg := readScriptImportFile(c)
	if status != 0 {
		h.writeScriptImportError(c, status, code, msg, nil)
		return
	}
	h.logScriptImportFile(filename, raw)
	if h.scriptImportService == nil {
		h.writeScriptImportError(c, http.StatusServiceUnavailable, scriptImportCodeInternal, "script import service unavailable", nil)
		return
	}
	result, err := h.scriptImportService.ValidateScriptImport(c.Request.Context(), username, filename, raw)
	if err != nil {
		h.writeScriptImportServiceError(c, err)
		return
	}
	if result == nil {
		h.writeScriptImportError(c, http.StatusInternalServerError, scriptImportCodeInternal, "script import validation returned no result", nil)
		return
	}
	if result.Summary.ErrorCount > 0 || hasScriptErrors(result.Issues) {
		h.writeScriptValidationFailure(c, result)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ValidateScriptReplacement(c *gin.Context) {
	username, ok := authenticatedUsername(c)
	if !ok {
		h.writeScriptImportError(c, http.StatusUnauthorized, "MML_UNAUTHORIZED", "authentication required", nil)
		return
	}
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		h.writeScriptImportError(c, http.StatusBadRequest, "MML_SCRIPT_ID_INVALID", "invalid script id", nil)
		return
	}
	filename, raw, status, code, msg := readScriptImportFile(c)
	if status != 0 {
		h.writeScriptImportError(c, status, code, msg, nil)
		return
	}
	h.logScriptImportFile(filename, raw)
	if h.scriptImportService == nil {
		h.writeScriptImportError(c, http.StatusServiceUnavailable, scriptImportCodeInternal, "script import service unavailable", nil)
		return
	}
	// Replacement validation uses the same server-authoritative validation
	// snapshot as a new import. The script id is intentionally only checked by
	// the replacement save operation, after the caller has reviewed the TXT.
	result, err := h.scriptImportService.ValidateScriptImport(c.Request.Context(), username, filename, raw)
	if err != nil {
		h.writeScriptImportServiceError(c, err)
		return
	}
	if result.Summary.ErrorCount > 0 || hasScriptErrors(result.Issues) {
		h.writeScriptValidationFailure(c, result)
		return
	}
	response.OK(c, result)
}

func (h *Handler) CreateScriptFromImport(c *gin.Context) {
	username, ok := authenticatedUsername(c)
	if !ok {
		h.writeScriptImportError(c, http.StatusUnauthorized, "MML_UNAUTHORIZED", "authentication required", nil)
		return
	}
	var req SaveImportedScriptRequest
	if err := decodeScriptImportJSON(c, &req); err != nil {
		h.writeScriptImportError(c, http.StatusBadRequest, "MML_IMPORT_REQUEST_INVALID", "invalid import request", nil)
		return
	}
	if h.scriptImportService == nil {
		h.writeScriptImportError(c, http.StatusServiceUnavailable, scriptImportCodeInternal, "script import service unavailable", nil)
		return
	}
	script, err := h.scriptImportService.CreateScriptFromImport(c.Request.Context(), username, req)
	if err != nil {
		h.writeScriptImportServiceError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, script)
}

func (h *Handler) ReplaceScriptFromImport(c *gin.Context) {
	username, ok := authenticatedUsername(c)
	if !ok {
		h.writeScriptImportError(c, http.StatusUnauthorized, "MML_UNAUTHORIZED", "authentication required", nil)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.writeScriptImportError(c, http.StatusBadRequest, "MML_SCRIPT_ID_INVALID", "invalid script id", nil)
		return
	}
	var req ReplaceImportedScriptRequest
	if err := decodeScriptImportJSON(c, &req); err != nil {
		h.writeScriptImportError(c, http.StatusBadRequest, "MML_IMPORT_REQUEST_INVALID", "invalid import request", nil)
		return
	}
	if h.scriptImportService == nil {
		h.writeScriptImportError(c, http.StatusServiceUnavailable, scriptImportCodeInternal, "script import service unavailable", nil)
		return
	}
	script, err := h.scriptImportService.ReplaceScriptFromImport(c.Request.Context(), id, username, req)
	if err != nil {
		h.writeScriptImportServiceError(c, err)
		return
	}
	response.OK(c, script)
}

func authenticatedUsername(c *gin.Context) (string, bool) {
	value, exists := c.Get("username")
	username, ok := value.(string)
	return strings.TrimSpace(username), exists && ok && strings.TrimSpace(username) != ""
}

func decodeScriptImportJSON(c *gin.Context, dst any) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func readScriptImportFile(c *gin.Context) (filename string, raw []byte, status int, code, msg string) {
	// Leave a small allowance for multipart headers while keeping the actual
	// script bounded by MaxScriptBytes.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(MaxScriptBytes)+64*1024)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "request body too large") {
			return "", nil, http.StatusRequestEntityTooLarge, scriptImportCodeTooLarge, "script file is too large"
		}
		return "", nil, http.StatusBadRequest, scriptImportCodeMissingFile, "file is required"
	}
	defer file.Close()
	filename = safeImportFilename(header.Filename)
	if !strings.EqualFold(filepath.Ext(filename), ".txt") {
		return "", nil, http.StatusBadRequest, scriptImportCodeTypeInvalid, "only .txt files are supported"
	}
	if header.Size > MaxScriptBytes {
		return "", nil, http.StatusRequestEntityTooLarge, scriptImportCodeTooLarge, "script file is too large"
	}
	raw, err = io.ReadAll(io.LimitReader(file, int64(MaxScriptBytes)+1))
	if err != nil {
		return "", nil, http.StatusBadRequest, scriptImportCodeInternal, "failed to read script file"
	}
	if len(raw) > MaxScriptBytes {
		return "", nil, http.StatusRequestEntityTooLarge, scriptImportCodeTooLarge, "script file is too large"
	}
	return filename, raw, 0, "", ""
}

func safeImportFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func (h *Handler) writeScriptImportServiceError(c *gin.Context, err error) {
	status, code, msg := scriptImportErrorStatus(err)
	h.writeScriptImportError(c, status, code, msg, nil)
}

func scriptImportErrorStatus(err error) (int, string, string) {
	switch {
	case errors.Is(err, commonerrors.ErrForbidden):
		return http.StatusForbidden, "MML_IMPORT_FORBIDDEN", "script import is not owned by the current user"
	case errors.Is(err, commonerrors.ErrUnauthorized):
		return http.StatusUnauthorized, "MML_UNAUTHORIZED", "authentication required"
	case errors.Is(err, commonerrors.ErrInvalidInput):
		return http.StatusBadRequest, "MML_IMPORT_REQUEST_INVALID", "invalid script import request"
	case errors.Is(err, commonerrors.ErrAlreadyExists):
		return http.StatusConflict, "MML_SCRIPT_NAME_DUPLICATED", "script name already exists"
	case errors.Is(err, ErrImportTokenExpired):
		return http.StatusConflict, scriptImportCodeTokenExpired, "validation token expired"
	case errors.Is(err, ErrImportTokenConsumed):
		return http.StatusConflict, scriptImportCodeTokenConsumed, "validation token already consumed"
	case errors.Is(err, ErrImportTokenClaimed), errors.Is(err, ErrScriptVersionConflict):
		return http.StatusConflict, scriptImportCodeVersion, "script import conflicts with another request"
	default:
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "redis") || strings.Contains(lower, "validator is nil") || strings.Contains(lower, "repository is nil") {
			return http.StatusServiceUnavailable, scriptImportCodeInternal, "script import dependency unavailable"
		}
		return http.StatusInternalServerError, scriptImportCodeInternal, "script import failed"
	}
}

func (h *Handler) writeScriptImportError(c *gin.Context, status int, code, msg string, issues []ScriptIssue) {
	if h.logger != nil {
		fields := []zap.Field{zap.String("code", code)}
		if len(issues) > 0 {
			fields = append(fields, zap.Int("issue_count", len(issues)))
			codes := make([]string, 0, len(issues))
			for _, issue := range issues {
				if issue.Code != "" {
					codes = append(codes, issue.Code)
				}
			}
			fields = append(fields, zap.Strings("issue_codes", codes))
		}
		h.logger.Warn("mml script import request failed", fields...)
	}
	payload := scriptImportErrorPayload{Code: code, Msg: msg, Issues: issues}
	// Keep the normal API envelope while exposing stable string code/message/
	// issues fields for clients that consume the import contract directly.
	c.AbortWithStatusJSON(status, gin.H{
		"ret":     0,
		"msg":     msg,
		"data":    payload,
		"code":    code,
		"message": msg,
		"issues":  issues,
	})
}

func (h *Handler) writeScriptValidationFailure(c *gin.Context, result *ImportValidationResponse) {
	if h.logger != nil {
		codes := make([]string, 0, len(result.Issues))
		for _, issue := range result.Issues {
			if issue.Code != "" {
				codes = append(codes, issue.Code)
			}
		}
		h.logger.Warn("mml script import validation failed", zap.Strings("issue_codes", codes))
	}
	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
		"ret":     0,
		"msg":     "script validation failed",
		"data":    result,
		"code":    scriptImportCodeValidation,
		"message": "script validation failed",
		"issues":  result.Issues,
	})
}

func (h *Handler) logScriptImportFile(filename string, raw []byte) {
	if h.logger == nil {
		return
	}
	digest := sha256.Sum256(raw)
	h.logger.Info("mml script import file received",
		zap.String("filename", filename),
		zap.Int("size", len(raw)),
		zap.String("sha256", hex.EncodeToString(digest[:])),
	)
}

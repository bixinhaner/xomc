package mml

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/middleware"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeScriptImportHTTPService struct {
	validation                         *ImportValidationResponse
	created                            *MMLScript
	validateErr, createErr, replaceErr error
}

func (f *fakeScriptImportHTTPService) ValidateScriptImport(_ context.Context, _, _ string, _ []byte) (*ImportValidationResponse, error) {
	if f.validateErr != nil {
		return nil, f.validateErr
	}
	return f.validation, nil
}
func (f *fakeScriptImportHTTPService) CreateScriptFromImport(_ context.Context, _ string, _ SaveImportedScriptRequest) (*MMLScript, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.created, nil
}
func (f *fakeScriptImportHTTPService) ReplaceScriptFromImport(_ context.Context, _ uuid.UUID, _ string, _ ReplaceImportedScriptRequest) (*MMLScript, error) {
	if f.replaceErr != nil {
		return nil, f.replaceErr
	}
	return f.created, nil
}

func importHTTPRouter(svc ScriptImportServiceAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Locale())
	h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
	h.SetScriptImportService(svc)
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func importMultipart(t *testing.T, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	return &body, mw.FormDataContentType()
}

func importRequestWithUser(method, target string, body *bytes.Buffer, contentType, user string) *http.Request {
	req := httptest.NewRequest(method, target, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	// Gin's context is populated by this tiny route middleware below.
	req.Header.Set("X-Test-Username", user)
	return req
}

func TestHandler_ScriptImportTemplate(t *testing.T) {
	r := importHTTPRouter(&fakeScriptImportHTTPService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/scripts/import/template", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/plain")
	require.Contains(t, rec.Header().Get("Content-Disposition"), "MMLTemplate.txt")
	require.Equal(t, "zh-CN", rec.Header().Get("Content-Language"))
	require.Contains(t, rec.Body.String(), "使用标准 PATH")
	require.Contains(t, rec.Body.String(), "PRIVATE:")
	require.Contains(t, rec.Body.String(), "支持操作")
	require.NotContains(t, rec.Body.String(), "DEL")
	require.NotContains(t, rec.Body.String(), "PATH:")
	require.Contains(t, rec.Body.String(), "# RMV Device.IP.Interface.1.IPv4Address.3.;DEVICE_SN")
	require.Contains(t, rec.Body.String(), "# LST Device.DeviceInfo.SoftwareVersion;DEVICE_SN,SECOND_SN")
}

func TestHandler_ScriptImportTemplateEnglish(t *testing.T) {
	r := importHTTPRouter(&fakeScriptImportHTTPService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/scripts/import/template", nil)
	req.Header.Set("Accept-Language", "en-US")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "en-US", rec.Header().Get("Content-Language"))
	body := rec.Body.String()
	require.Contains(t, body, "MML TXT Script Template")
	require.Contains(t, body, "Supported operations")
	require.Contains(t, body, "Standard PATH")
	require.Contains(t, body, "PRIVATE:")
	require.NotContains(t, body, "DEL")
	require.NotContains(t, body, "支持操作")
	require.NotContains(t, body, "PATH:")
	require.Contains(t, body, "# RMV Device.IP.Interface.1.IPv4Address.3.;DEVICE_SN")
	require.Contains(t, body, "# LST Device.DeviceInfo.SoftwareVersion;DEVICE_SN,SECOND_SN")
}

func TestHandler_ValidateScriptImport_RejectsNonTXT(t *testing.T) {
	r := importHTTPRouter(&fakeScriptImportHTTPService{})
	body, contentType := importMultipart(t, "script.csv", []byte("LST DEVICE_INFO;SN1\n"))
	req := importRequestWithUser(http.MethodPost, "/api/v1/mml/scripts/import/validate", body, contentType, "admin")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	// No auth context means the handler must not inspect or trust form fields.
	require.Contains(t, rec.Body.String(), "MML_UNAUTHORIZED")
}

func TestHandler_ValidateScriptImport_RejectsNonTXTAuthenticated(t *testing.T) {
	svc := &fakeScriptImportHTTPService{}
	r := gin.New()
	h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
	h.SetScriptImportService(svc)
	r.Use(func(c *gin.Context) { c.Set("username", "admin"); c.Next() })
	h.RegisterRoutes(r.Group("/api/v1"))
	body, contentType := importMultipart(t, "script.csv", []byte("LST DEVICE_INFO;SN1\n"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/import/validate", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "MML_FILE_TYPE_INVALID")
}

func TestHandler_ValidateScriptImport_RejectsMissingFile(t *testing.T) {
	r := gin.New()
	h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
	h.SetScriptImportService(&fakeScriptImportHTTPService{})
	r.Use(func(c *gin.Context) { c.Set("username", "admin"); c.Next() })
	h.RegisterRoutes(r.Group("/api/v1"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/import/validate", bytes.NewBufferString("not multipart"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), scriptImportCodeMissingFile)
}

func TestHandler_ValidateScriptImport_Returns422Issues(t *testing.T) {
	svc := &fakeScriptImportHTTPService{validation: &ImportValidationResponse{Summary: ScriptValidationSummary{ErrorCount: 1}, Issues: []ScriptIssue{{Code: "MML_COMMAND_NOT_FOUND", Severity: IssueError, LineNo: 3}}}}
	r := gin.New()
	h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
	h.SetScriptImportService(svc)
	r.Use(func(c *gin.Context) { c.Set("username", "admin"); c.Next() })
	h.RegisterRoutes(r.Group("/api/v1"))
	body, contentType := importMultipart(t, "script.txt", []byte("LST DEVICE_INFO;SN1\n"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/import/validate", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Contains(t, rec.Body.String(), "MML_COMMAND_NOT_FOUND")
}

func TestHandler_ValidateScriptImport_LocalizesIssueDisplayMessage(t *testing.T) {
	tests := []struct {
		name           string
		acceptLanguage string
		want           string
	}{
		{name: "default Chinese", want: "命令末尾必须使用 ;设备SN"},
		{name: "English", acceptLanguage: "en-US", want: "Command must end with ;device SN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeScriptImportHTTPService{validation: &ImportValidationResponse{
				Summary: ScriptValidationSummary{ErrorCount: 1},
				Issues: []ScriptIssue{{
					Code:     "MML_DEVICE_SN_REQUIRED",
					Severity: IssueError,
					LineNo:   21,
					RawLine:  "LST Device.DeviceInfo.SoftwareVersion",
					Message:  "command must end with ;SN",
				}},
			}}
			r := gin.New()
			r.Use(middleware.Locale())
			h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
			h.SetScriptImportService(svc)
			r.Use(func(c *gin.Context) { c.Set("username", "admin"); c.Next() })
			h.RegisterRoutes(r.Group("/api/v1"))
			body, contentType := importMultipart(t, "script.txt", []byte("LST Device.DeviceInfo.SoftwareVersion\n"))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/import/validate", body)
			req.Header.Set("Content-Type", contentType)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			var envelope struct {
				Issues []ScriptIssue            `json:"issues"`
				Data   ImportValidationResponse `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
			require.Equal(t, tt.want, envelope.Issues[0].DisplayMessage)
			require.Equal(t, tt.want, envelope.Data.Issues[0].DisplayMessage)
		})
	}
}

func TestHandler_CreateScriptFromImport201(t *testing.T) {
	id := uuid.New()
	svc := &fakeScriptImportHTTPService{created: &MMLScript{ID: id, ScriptName: "巡检"}}
	r := gin.New()
	h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
	h.SetScriptImportService(svc)
	r.Use(func(c *gin.Context) { c.Set("username", "admin"); c.Next() })
	h.RegisterRoutes(r.Group("/api/v1"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/import", bytes.NewReader([]byte(`{"validation_token":"token","script_name":"巡检"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), id.String())
}

func TestHandler_CreateScriptFromImportMapsForbidden(t *testing.T) {
	svc := &fakeScriptImportHTTPService{createErr: commonerrors.ErrForbidden}
	r := gin.New()
	h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
	h.SetScriptImportService(svc)
	r.Use(func(c *gin.Context) { c.Set("username", "admin"); c.Next() })
	h.RegisterRoutes(r.Group("/api/v1"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/import", bytes.NewReader([]byte(`{"validation_token":"token","script_name":"巡检"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "MML_IMPORT_FORBIDDEN")
}

func TestHandler_CreateScriptFromImportMapsConsumedTo409(t *testing.T) {
	svc := &fakeScriptImportHTTPService{createErr: ErrImportTokenConsumed}
	r := gin.New()
	h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
	h.SetScriptImportService(svc)
	r.Use(func(c *gin.Context) { c.Set("username", "admin"); c.Next() })
	h.RegisterRoutes(r.Group("/api/v1"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/import", bytes.NewReader([]byte(`{"validation_token":"token","script_name":"巡检"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), "MML_IMPORT_TOKEN_CONSUMED")
}

func TestHandler_ScriptImportReplacementValidateRoute(t *testing.T) {
	svc := &fakeScriptImportHTTPService{validation: &ImportValidationResponse{ValidationToken: "token", Summary: ScriptValidationSummary{ValidLines: 1}}}
	r := gin.New()
	h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
	h.SetScriptImportService(svc)
	r.Use(func(c *gin.Context) { c.Set("username", "admin"); c.Next() })
	h.RegisterRoutes(r.Group("/api/v1"))
	body, contentType := importMultipart(t, "replacement.txt", []byte("LST DEVICE_INFO;SN1\n"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/"+uuid.NewString()+"/import/validate", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "validation_token")
}

func TestHandler_ScriptImportReplacementSaveRoute(t *testing.T) {
	id := uuid.New()
	svc := &fakeScriptImportHTTPService{created: &MMLScript{ID: id, ScriptName: "替换后"}}
	r := gin.New()
	h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
	h.SetScriptImportService(svc)
	r.Use(func(c *gin.Context) { c.Set("username", "admin"); c.Next() })
	h.RegisterRoutes(r.Group("/api/v1"))
	req := httptest.NewRequest(http.MethodPut, "/api/v1/mml/scripts/"+id.String()+"/import", bytes.NewBufferString(`{"validation_token":"token","script_name":"替换后","expected_updated_at":"2026-07-10T00:00:00Z"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), id.String())
}

func TestHandler_ValidateScriptImport_RejectsOversize(t *testing.T) {
	r := gin.New()
	h := NewHandler(NewService(&hCmdRepo{}, &hScriptRepo{}, &hTaskRepo{}, &hCustomCommandRepo{}, nil, zap.NewNop()), zap.NewNop())
	h.SetScriptImportService(&fakeScriptImportHTTPService{})
	r.Use(func(c *gin.Context) { c.Set("username", "admin"); c.Next() })
	h.RegisterRoutes(r.Group("/api/v1"))
	body, contentType := importMultipart(t, "script.txt", bytes.Repeat([]byte("x"), MaxScriptBytes+1))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/import/validate", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	require.Contains(t, rec.Body.String(), scriptImportCodeTooLarge)
}

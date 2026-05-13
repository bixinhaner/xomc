package template

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock: ConfigTemplateRepository
// ---------------------------------------------------------------------------

type mockConfigTemplateRepo struct {
	templates map[uuid.UUID]*ConfigTemplate
}

func newMockConfigTemplateRepo() *mockConfigTemplateRepo {
	return &mockConfigTemplateRepo{
		templates: make(map[uuid.UUID]*ConfigTemplate),
	}
}

func (m *mockConfigTemplateRepo) Create(_ context.Context, t *ConfigTemplate) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	m.templates[t.ID] = t
	return nil
}

func (m *mockConfigTemplateRepo) GetByID(_ context.Context, id uuid.UUID) (*ConfigTemplate, error) {
	t, ok := m.templates[id]
	if !ok {
		return nil, errTemplateNotFound
	}
	return t, nil
}

func (m *mockConfigTemplateRepo) Update(_ context.Context, t *ConfigTemplate) error {
	if _, ok := m.templates[t.ID]; !ok {
		return errTemplateNotFound
	}
	t.UpdatedAt = time.Now()
	m.templates[t.ID] = t
	return nil
}

func (m *mockConfigTemplateRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.templates[id]; !ok {
		return errTemplateNotFound
	}
	delete(m.templates, id)
	return nil
}

func (m *mockConfigTemplateRepo) List(_ context.Context, filter ConfigTemplateFilter) (*model.ListResponse[ConfigTemplate], error) {
	items := make([]ConfigTemplate, 0, len(m.templates))
	for _, t := range m.templates {
		items = append(items, *t)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

func (m *mockConfigTemplateRepo) FindByCarrierTech(_ context.Context, carrier model.CarrierCode, tech model.Technology, templateType TemplateType) ([]ConfigTemplate, error) {
	var results []ConfigTemplate
	for _, t := range m.templates {
		if t.Carrier == carrier && t.Technology == tech && t.TemplateType == templateType {
			results = append(results, *t)
		}
	}
	return results, nil
}

func (m *mockConfigTemplateRepo) FindBestMatch(_ context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, templateType TemplateType) (*ConfigTemplate, error) {
	for _, t := range m.templates {
		if t.Carrier == carrier && t.Technology == tech && t.TemplateType == templateType && t.Active {
			if productClass != "" && t.ProductClass == productClass {
				return t, nil
			}
		}
	}
	// Fallback: match without product class.
	for _, t := range m.templates {
		if t.Carrier == carrier && t.Technology == tech && t.TemplateType == templateType && t.Active {
			return t, nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var errTemplateNotFound = fmt.Errorf("template not found")

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func newTestHandler() (*Handler, *mockConfigTemplateRepo) {
	repo := newMockConfigTemplateRepo()
	h := NewHandler(repo)
	return h, repo
}

func seedTemplate(repo *mockConfigTemplateRepo, id uuid.UUID, name string, carrier model.CarrierCode, tech model.Technology, tt TemplateType) *ConfigTemplate {
	now := time.Now()
	t := &ConfigTemplate{
		ID:           id,
		Name:         name,
		Carrier:      carrier,
		Technology:   tech,
		TemplateType: tt,
		Parameters:   json.RawMessage(`{"key":"value"}`),
		Priority:     1,
		Version:      1,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	repo.templates[id] = t
	return t
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_ListTemplates(t *testing.T) {
	h, repo := newTestHandler()
	router := setupRouter(h)

	seedTemplate(repo, uuid.New(), "Template-A", model.CarrierCMCC, model.TechLTE, TemplateProvisioning)
	seedTemplate(repo, uuid.New(), "Template-B", model.CarrierCTCC, model.TechNR, TemplateBatchConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[ConfigTemplate]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

func TestHandler_CreateTemplate(t *testing.T) {
	h, _ := newTestHandler()
	router := setupRouter(h)

	body := createTemplateRequest{
		Name:         "Provisioning Template",
		Carrier:      "cmcc",
		Technology:   "lte",
		TemplateType: "provisioning",
		Parameters:   json.RawMessage(`{"param1":"val1"}`),
		Priority:     10,
		Description:  "Test template",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp ConfigTemplate
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "Provisioning Template", resp.Name)
	assert.Equal(t, model.CarrierCode("cmcc"), resp.Carrier)
	assert.Equal(t, model.Technology("lte"), resp.Technology)
	assert.Equal(t, TemplateProvisioning, resp.TemplateType)
	assert.Equal(t, 10, resp.Priority)
	assert.Equal(t, 1, resp.Version)
	assert.True(t, resp.Active)
	assert.NotEqual(t, uuid.Nil, resp.ID)
}

func TestHandler_GetTemplate(t *testing.T) {
	h, repo := newTestHandler()
	router := setupRouter(h)

	tmplID := uuid.New()
	seedTemplate(repo, tmplID, "My Template", model.CarrierCUCC, model.TechLTE, TemplateProvisioning)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+tmplID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ConfigTemplate
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, tmplID, resp.ID)
	assert.Equal(t, "My Template", resp.Name)
}

func TestHandler_GetTemplate_NotFound(t *testing.T) {
	h, _ := newTestHandler()
	router := setupRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UpdateTemplate(t *testing.T) {
	h, repo := newTestHandler()
	router := setupRouter(h)

	tmplID := uuid.New()
	seedTemplate(repo, tmplID, "Old Name", model.CarrierCMCC, model.TechLTE, TemplateProvisioning)

	body := updateTemplateRequest{
		Name:         "Updated Name",
		Carrier:      "ctcc",
		Technology:   "nr",
		TemplateType: "batch_config",
		Parameters:   json.RawMessage(`{"updated":"true"}`),
		Priority:     5,
		Active:       true,
		Description:  "Updated description",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/"+tmplID.String(), bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ConfigTemplate
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "Updated Name", resp.Name)
	assert.Equal(t, model.CarrierCode("ctcc"), resp.Carrier)
	assert.Equal(t, model.Technology("nr"), resp.Technology)
	assert.Equal(t, TemplateBatchConfig, resp.TemplateType)
}

func TestHandler_DeleteTemplate(t *testing.T) {
	h, repo := newTestHandler()
	router := setupRouter(h)

	tmplID := uuid.New()
	seedTemplate(repo, tmplID, "ToDelete", model.CarrierCMCC, model.TechLTE, TemplateProvisioning)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/"+tmplID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response.DecodeData(t, w.Body, nil)

	// Verify template removed from repo.
	_, exists := repo.templates[tmplID]
	assert.False(t, exists)
}

func TestHandler_DeleteTemplate_NotFound(t *testing.T) {
	h, _ := newTestHandler()
	router := setupRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Mock: TemplateDispatcher (T-0120)
// ---------------------------------------------------------------------------

type mockTemplateDispatcher struct {
	calls    []dispatcherCall
	resultFn func(tmplID, deviceID uuid.UUID) (uuid.UUID, error)
}

type dispatcherCall struct {
	TemplateID uuid.UUID
	DeviceID   uuid.UUID
}

func (m *mockTemplateDispatcher) DispatchTemplate(_ context.Context, tmpl *ConfigTemplate, deviceID uuid.UUID) (uuid.UUID, error) {
	m.calls = append(m.calls, dispatcherCall{TemplateID: tmpl.ID, DeviceID: deviceID})
	if m.resultFn != nil {
		return m.resultFn(tmpl.ID, deviceID)
	}
	return uuid.New(), nil
}

func newTestHandlerWithDispatcher() (*Handler, *mockConfigTemplateRepo, *mockTemplateDispatcher) {
	h, repo := newTestHandler()
	disp := &mockTemplateDispatcher{}
	h.SetDispatcher(disp)
	return h, repo, disp
}

// ---------------------------------------------------------------------------
// T-0120 Dispatch tests
// ---------------------------------------------------------------------------

func TestHandler_Dispatch_NotConfigured(t *testing.T) {
	h, repo := newTestHandler()
	router := setupRouter(h)
	tmplID := uuid.New()
	seedTemplate(repo, tmplID, "Tmpl", model.CarrierCMCC, model.TechLTE, TemplateProvisioning)

	body := dispatchTemplateRequest{DeviceIDs: []string{uuid.New().String()}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/dispatch",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandler_Dispatch_HappyPath_SingleDevice(t *testing.T) {
	h, repo, disp := newTestHandlerWithDispatcher()
	router := setupRouter(h)
	tmplID := uuid.New()
	seedTemplate(repo, tmplID, "Tmpl", model.CarrierCMCC, model.TechLTE, TemplateProvisioning)
	deviceID := uuid.New()
	expectedTaskID := uuid.New()
	disp.resultFn = func(_, _ uuid.UUID) (uuid.UUID, error) { return expectedTaskID, nil }

	body := dispatchTemplateRequest{DeviceIDs: []string{deviceID.String()}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/dispatch",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp dispatchTemplateResponse
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, tmplID.String(), resp.TemplateID)
	assert.Equal(t, 1, resp.TotalDevices)
	require.Len(t, resp.Dispatched, 1)
	assert.Equal(t, deviceID.String(), resp.Dispatched[0].DeviceID)
	assert.Equal(t, expectedTaskID.String(), resp.Dispatched[0].TaskID)
	assert.Empty(t, resp.Failed)
	require.Len(t, disp.calls, 1)
	assert.Equal(t, tmplID, disp.calls[0].TemplateID)
	assert.Equal(t, deviceID, disp.calls[0].DeviceID)
}

func TestHandler_Dispatch_PartialFailure(t *testing.T) {
	h, repo, disp := newTestHandlerWithDispatcher()
	router := setupRouter(h)
	tmplID := uuid.New()
	seedTemplate(repo, tmplID, "Tmpl", model.CarrierCMCC, model.TechLTE, TemplateProvisioning)
	okDevice := uuid.New()
	failDevice := uuid.New()
	disp.resultFn = func(_, deviceID uuid.UUID) (uuid.UUID, error) {
		if deviceID == failDevice {
			return uuid.Nil, fmt.Errorf("simulated dispatch failure")
		}
		return uuid.New(), nil
	}

	body := dispatchTemplateRequest{DeviceIDs: []string{okDevice.String(), failDevice.String()}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/dispatch",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusMultiStatus, w.Code)
	var resp dispatchTemplateResponse
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, 2, resp.TotalDevices)
	require.Len(t, resp.Dispatched, 1)
	require.Len(t, resp.Failed, 1)
	assert.Equal(t, okDevice.String(), resp.Dispatched[0].DeviceID)
	assert.Equal(t, failDevice.String(), resp.Failed[0].DeviceID)
	assert.Contains(t, resp.Failed[0].Error, "simulated dispatch failure")
}

func TestHandler_Dispatch_TemplateNotFound(t *testing.T) {
	h, _, _ := newTestHandlerWithDispatcher()
	router := setupRouter(h)
	body := dispatchTemplateRequest{DeviceIDs: []string{uuid.New().String()}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+uuid.New().String()+"/dispatch",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Dispatch_EmptyDeviceIDs(t *testing.T) {
	h, repo, _ := newTestHandlerWithDispatcher()
	router := setupRouter(h)
	tmplID := uuid.New()
	seedTemplate(repo, tmplID, "Tmpl", model.CarrierCMCC, model.TechLTE, TemplateProvisioning)
	body := dispatchTemplateRequest{DeviceIDs: []string{}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/dispatch",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Dispatch_AllFailed_UnprocessableEntity(t *testing.T) {
	h, repo, disp := newTestHandlerWithDispatcher()
	router := setupRouter(h)
	tmplID := uuid.New()
	seedTemplate(repo, tmplID, "Tmpl", model.CarrierCMCC, model.TechLTE, TemplateProvisioning)
	disp.resultFn = func(_, _ uuid.UUID) (uuid.UUID, error) {
		return uuid.Nil, fmt.Errorf("device offline")
	}
	body := dispatchTemplateRequest{DeviceIDs: []string{uuid.New().String(), uuid.New().String()}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/dispatch",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var resp dispatchTemplateResponse
	response.DecodeData(t, w.Body, &resp)
	assert.Empty(t, resp.Dispatched)
	assert.Len(t, resp.Failed, 2)
}

func TestHandler_Dispatch_InvalidUUID_MixedWithValid(t *testing.T) {
	h, repo, disp := newTestHandlerWithDispatcher()
	router := setupRouter(h)
	tmplID := uuid.New()
	seedTemplate(repo, tmplID, "Tmpl", model.CarrierCMCC, model.TechLTE, TemplateProvisioning)
	okDevice := uuid.New()
	body := dispatchTemplateRequest{DeviceIDs: []string{okDevice.String(), "not-a-uuid"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/dispatch",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusMultiStatus, w.Code)
	var resp dispatchTemplateResponse
	response.DecodeData(t, w.Body, &resp)
	assert.Len(t, resp.Dispatched, 1)
	assert.Len(t, resp.Failed, 1)
	assert.Equal(t, "not-a-uuid", resp.Failed[0].DeviceID)
	assert.Contains(t, resp.Failed[0].Error, "invalid device_id format")
	require.Len(t, disp.calls, 1) // 仅 ok 设备进 dispatcher
	assert.Equal(t, okDevice, disp.calls[0].DeviceID)
}

func TestHandler_CreateTemplate_BadRequest(t *testing.T) {
	h, _ := newTestHandler()
	router := setupRouter(h)

	// Missing required fields (name, carrier, technology, template_type, parameters).
	body := map[string]string{
		"description": "incomplete",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

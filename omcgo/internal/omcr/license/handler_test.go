package license

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/common/model"
)

// ---------------------------------------------------------------------------
// Mock: LicenseRepository
// ---------------------------------------------------------------------------

type fakeLicenseRepo struct {
	licenses map[uuid.UUID]*License
	byCode   map[string]*License
}

func newFakeLicenseRepo() *fakeLicenseRepo {
	return &fakeLicenseRepo{
		licenses: make(map[uuid.UUID]*License),
		byCode:   make(map[string]*License),
	}
}

func (m *fakeLicenseRepo) Create(_ context.Context, lic *License) error {
	lic.ID = uuid.New()
	now := time.Now()
	lic.CreatedAt = now
	lic.UpdatedAt = now
	m.licenses[lic.ID] = lic
	m.byCode[lic.LicenseCode] = lic
	return nil
}

func (m *fakeLicenseRepo) GetByID(_ context.Context, id uuid.UUID) (*License, error) {
	lic, ok := m.licenses[id]
	if !ok {
		return nil, nil
	}
	return lic, nil
}

func (m *fakeLicenseRepo) GetByCode(_ context.Context, code string) (*License, error) {
	lic, ok := m.byCode[code]
	if !ok {
		return nil, nil
	}
	return lic, nil
}

func (m *fakeLicenseRepo) Update(_ context.Context, lic *License) error {
	lic.UpdatedAt = time.Now()
	m.licenses[lic.ID] = lic
	m.byCode[lic.LicenseCode] = lic
	return nil
}

func (m *fakeLicenseRepo) List(_ context.Context, filter LicenseFilter) (*model.ListResponse[License], error) {
	items := make([]License, 0, len(m.licenses))
	for _, lic := range m.licenses {
		if filter.Status != nil && lic.Status != *filter.Status {
			continue
		}
		if filter.LicenseType != nil && lic.LicenseType != *filter.LicenseType {
			continue
		}
		if filter.DeviceType != nil && (lic.DeviceType == nil || *lic.DeviceType != *filter.DeviceType) {
			continue
		}
		items = append(items, *lic)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

func (m *fakeLicenseRepo) Summary(_ context.Context) (*LicenseSummary, error) {
	summary := &LicenseSummary{}
	for _, lic := range m.licenses {
		summary.Total++
		switch lic.Status {
		case StatusActive:
			summary.Active++
		case StatusExpired:
			summary.Expired++
		case StatusPending:
			summary.Pending++
		}
	}
	return summary, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupLicenseRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func newTestLicenseHandler() (*Handler, *fakeLicenseRepo) {
	repo := newFakeLicenseRepo()
	logger := zap.NewNop()
	svc := NewService(repo, logger)
	h := NewHandler(svc, logger)
	return h, repo
}

func seedLicense(repo *fakeLicenseRepo, name, code, product string, licType LicenseType, status LicenseStatus) *License {
	now := time.Now()
	lic := &License{
		ID:          uuid.New(),
		LicenseName: name,
		LicenseCode: code,
		ProductName: product,
		LicenseType: licType,
		Status:      status,
		MaxDevices:  100,
		UsedDevices: 10,
		Features:    json.RawMessage(`["feature_a","feature_b"]`),
		IssueDate:   now.AddDate(0, -1, 0),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repo.licenses[lic.ID] = lic
	repo.byCode[lic.LicenseCode] = lic
	return lic
}

func mustMarshalLicense(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_List(t *testing.T) {
	h, repo := newTestLicenseHandler()
	router := setupLicenseRouter(h)

	seedLicense(repo, "License A", "LIC-001", "Product X", TypeSubscription, StatusActive)
	seedLicense(repo, "License B", "LIC-002", "Product Y", TypePerpetual, StatusPending)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[License]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
}

func TestHandler_GetByID(t *testing.T) {
	h, repo := newTestLicenseHandler()
	router := setupLicenseRouter(h)

	lic := seedLicense(repo, "License A", "LIC-GET-001", "Product X", TypeSubscription, StatusActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/"+lic.ID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp License
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, lic.ID, resp.ID)
	assert.Equal(t, "License A", resp.LicenseName)
	assert.Equal(t, "LIC-GET-001", resp.LicenseCode)
	assert.Equal(t, TypeSubscription, resp.LicenseType)
	assert.Equal(t, StatusActive, resp.Status)
}

func TestHandler_GetSummary(t *testing.T) {
	h, repo := newTestLicenseHandler()
	router := setupLicenseRouter(h)

	seedLicense(repo, "Lic A", "LIC-S-001", "P1", TypeSubscription, StatusActive)
	seedLicense(repo, "Lic B", "LIC-S-002", "P2", TypeSubscription, StatusActive)
	seedLicense(repo, "Lic C", "LIC-S-003", "P3", TypePerpetual, StatusExpired)
	seedLicense(repo, "Lic D", "LIC-S-004", "P4", TypeTrial, StatusPending)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/summary", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp LicenseSummary
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(4), resp.Total)
	assert.Equal(t, int64(2), resp.Active)
	assert.Equal(t, int64(1), resp.Expired)
	assert.Equal(t, int64(1), resp.Pending)
}

func TestHandler_Activate(t *testing.T) {
	h, repo := newTestLicenseHandler()
	router := setupLicenseRouter(h)

	lic := seedLicense(repo, "License Pending", "LIC-ACT-001", "Product X", TypeSubscription, StatusPending)

	body := ActivateRequest{
		LicenseCode: "LIC-ACT-001",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/activate", bytes.NewReader(mustMarshalLicense(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp License
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, lic.ID, resp.ID)
	assert.Equal(t, StatusActive, resp.Status)

	// Verify the repo was updated.
	assert.Equal(t, StatusActive, repo.licenses[lic.ID].Status)
}

func TestHandler_Revoke(t *testing.T) {
	h, repo := newTestLicenseHandler()
	router := setupLicenseRouter(h)

	lic := seedLicense(repo, "License Active", "LIC-REV-001", "Product X", TypeSubscription, StatusActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/"+lic.ID.String()+"/revoke", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "revoked", resp["status"])

	// Verify the repo was updated.
	assert.Equal(t, StatusRevoked, repo.licenses[lic.ID].Status)
}

func TestHandler_Import(t *testing.T) {
	h, _ := newTestLicenseHandler()
	router := setupLicenseRouter(h)

	issueDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	expiryDate := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	licensor := "Vendor Corp"
	deviceType := "pico"
	region := "east"
	notes := "Imported for testing"

	body := ImportRequest{
		LicenseName: "Imported License",
		LicenseCode: "LIC-IMP-001",
		ProductName: "Product Z",
		LicenseType: TypePerpetual,
		Status:      StatusPending,
		MaxDevices:  500,
		UsedDevices: 0,
		Features:    json.RawMessage(`["advanced_pm","auto_provision"]`),
		IssueDate:   issueDate,
		ExpiryDate:  &expiryDate,
		Licensor:    &licensor,
		DeviceType:  &deviceType,
		Region:      &region,
		Notes:       &notes,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/import", bytes.NewReader(mustMarshalLicense(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp License
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, "Imported License", resp.LicenseName)
	assert.Equal(t, "LIC-IMP-001", resp.LicenseCode)
	assert.Equal(t, "Product Z", resp.ProductName)
	assert.Equal(t, TypePerpetual, resp.LicenseType)
	assert.Equal(t, StatusPending, resp.Status)
	assert.Equal(t, 500, resp.MaxDevices)
	assert.Equal(t, 0, resp.UsedDevices)
	require.NotNil(t, resp.Licensor)
	assert.Equal(t, "Vendor Corp", *resp.Licensor)
	require.NotNil(t, resp.DeviceType)
	assert.Equal(t, "pico", *resp.DeviceType)
	require.NotNil(t, resp.Region)
	assert.Equal(t, "east", *resp.Region)
	require.NotNil(t, resp.Notes)
	assert.Equal(t, "Imported for testing", *resp.Notes)
}

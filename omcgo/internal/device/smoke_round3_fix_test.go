package device

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

// fakeRegistrationRepo is a minimal in-memory RegistrationRepository stub used
// to exercise the registration handler validation path without a real DB.
type fakeRegistrationRepo struct {
	created *DeviceRegistration
}

func (f *fakeRegistrationRepo) Create(ctx context.Context, reg *DeviceRegistration) error {
	reg.ID = uuid.New()
	f.created = reg
	return nil
}
func (f *fakeRegistrationRepo) BatchCreate(ctx context.Context, regs []*DeviceRegistration) (int, error) {
	return len(regs), nil
}
func (f *fakeRegistrationRepo) GetBySerialNumber(ctx context.Context, sn string) (*DeviceRegistration, error) {
	return nil, nil // never pre-registered
}
func (f *fakeRegistrationRepo) List(ctx context.Context, filter RegistrationFilter) (*model.ListResponse[DeviceRegistration], error) {
	return model.NewListResponse([]DeviceRegistration{}, 0, 1, 20), nil
}
func (f *fakeRegistrationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (f *fakeRegistrationRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

// ---------------------------------------------------------------------------
// issue #126 item 2: POST /device-registrations without group_id must 400,
// not fall through to a DB NOT NULL constraint 500.
// ---------------------------------------------------------------------------

func TestCreateRegistration_GroupIDValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
	}{
		{
			name: "missing group_id returns 400",
			body: map[string]any{
				"serial_number": "SNTEST0001",
				"carrier":       "cmcc",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "empty group_id returns 400",
			body: map[string]any{
				"serial_number": "SNTEST0002",
				"carrier":       "cmcc",
				"group_id":      "",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "valid group_id returns 201",
			body: map[string]any{
				"serial_number": "SNTEST0003",
				"carrier":       "cmcc",
				"group_id":      uuid.New().String(),
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewRegistrationService(&fakeRegistrationRepo{}, zap.NewNop())
			h := NewRegistrationHandler(svc)

			router := gin.New()
			api := router.Group("/api/v1")
			h.RegisterRoutes(api)

			payload, err := json.Marshal(tt.body)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/device-registrations", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code, "body: %s", rec.Body.String())
		})
	}
}

// ---------------------------------------------------------------------------
// issue #145 item A: PUT /devices/:id/info on a missing device / absent
// device_info row must map to 404 (repo returns ErrNotFound sentinel, handler
// maps via HTTPStatusFromError), not 500.
// ---------------------------------------------------------------------------

// stubInfoUpdateRepo lets us drive UpdateManualFields' return value while
// satisfying the broader DeviceInfoRepository surface via embedding the gomock.
type stubInfoUpdateRepo struct {
	DeviceInfoRepository
	updateErr error
}

func (s stubInfoUpdateRepo) UpdateManualFields(ctx context.Context, deviceID uuid.UUID, req UpdateDeviceInfoRequest, updater string) error {
	return s.updateErr
}

func TestUpdateDeviceInfo_NotFoundMapsTo404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		updateErr  error
		wantStatus int
	}{
		{
			name:       "missing device_info row maps to 404",
			updateErr:  commonerrors.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "wrapped not-found still maps to 404",
			updateErr:  errWrap("device_info not found for device", commonerrors.ErrNotFound),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "successful update returns 200",
			updateErr:  nil,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// UpdateDeviceInfo only touches deviceInfoRepo; the other deps are
			// unused on this path so nil is acceptable for a focused unit test.
			svc := NewDeviceService(nil, nil, nil, nil, zap.NewNop())
			svc.SetDeviceInfoRepo(stubInfoUpdateRepo{updateErr: tt.updateErr})
			h := NewDeviceInfoHandler(svc)

			router := gin.New()
			api := router.Group("/api/v1")
			h.RegisterRoutes(api)

			body := []byte(`{"device_name":"test"}`)
			url := "/api/v1/devices/" + uuid.New().String() + "/info"
			req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code, "body: %s", rec.Body.String())
		})
	}
}

// errWrap wraps sentinel preserving errors.Is matching (mirrors the repo path).
func errWrap(msg string, sentinel error) error {
	return wrappedErr{msg: msg, err: sentinel}
}

type wrappedErr struct {
	msg string
	err error
}

func (w wrappedErr) Error() string { return w.msg + ": " + w.err.Error() }
func (w wrappedErr) Unwrap() error { return w.err }

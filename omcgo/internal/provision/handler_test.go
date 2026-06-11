package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// stubDeviceChecker satisfies deviceExistenceChecker for handler tests.
// GetFn lets each case decide whether the device exists (non-nil),
// is absent (nil, nil — the PgDeviceRepository semantics for a missing row),
// or the lookup errored.
type stubDeviceChecker struct {
	GetFn func(ctx context.Context, id uuid.UUID) (*model.Device, error)
}

func (s *stubDeviceChecker) GetDevice(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	if s.GetFn != nil {
		return s.GetFn(ctx, id)
	}
	return nil, nil
}

// newCreateRequest builds a POST /provisioning/tasks gin context for the handler.
func newCreateRequest(t *testing.T, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provisioning/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, rec
}

func TestHandler_Create_DeviceExistencePrecheck(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		body           string
		checker        deviceExistenceChecker
		wantStatus     int
		wantTaskCreate bool // whether repo.Create should have been invoked
	}{
		{
			name: "device exists -> 201 and task created",
			body: fmt.Sprintf(`{"device_id":%q}`, existingID),
			checker: &stubDeviceChecker{GetFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
				return &model.Device{ID: id, SerialNumber: "SN-EXIST"}, nil
			}},
			wantStatus:     http.StatusCreated,
			wantTaskCreate: true,
		},
		{
			name: "device absent -> 404 and no orphan task",
			body: fmt.Sprintf(`{"device_id":%q}`, uuid.New()),
			checker: &stubDeviceChecker{GetFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
				return nil, nil // PgDeviceRepository returns (nil, nil) for missing row
			}},
			wantStatus:     http.StatusNotFound,
			wantTaskCreate: false,
		},
		{
			name: "device lookup ErrNotFound -> 404 and no orphan task",
			body: fmt.Sprintf(`{"device_id":%q}`, uuid.New()),
			checker: &stubDeviceChecker{GetFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
				return nil, commonerrors.ErrNotFound
			}},
			wantStatus:     http.StatusNotFound,
			wantTaskCreate: false,
		},
		{
			name: "device lookup internal error -> 500 and no orphan task",
			body: fmt.Sprintf(`{"device_id":%q}`, uuid.New()),
			checker: &stubDeviceChecker{GetFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
				return nil, errors.New("db down")
			}},
			wantStatus:     http.StatusInternalServerError,
			wantTaskCreate: false,
		},
		{
			name:           "missing device_id -> 400 (binding required)",
			body:           `{}`,
			checker:        &stubDeviceChecker{},
			wantStatus:     http.StatusBadRequest,
			wantTaskCreate: false,
		},
		{
			name:           "malformed device_id -> 400",
			body:           `{"device_id":"not-a-uuid"}`,
			checker:        &stubDeviceChecker{},
			wantStatus:     http.StatusBadRequest,
			wantTaskCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var created bool
			repo := &mockTaskRepo{
				CreateFn: func(_ context.Context, _ *ProvisioningTask) error {
					created = true
					return nil
				},
			}
			h := NewHandler(repo, nil)
			h.SetDeviceChecker(tt.checker)

			c, rec := newCreateRequest(t, tt.body)
			h.Create(c)

			assert.Equal(t, tt.wantStatus, rec.Code, "HTTP status")
			assert.Equal(t, tt.wantTaskCreate, created, "repo.Create invocation")

			// On the non-existent-device path, ensure the response is the
			// standard failure envelope (ret=0) and not a 201 success.
			if tt.wantStatus == http.StatusNotFound {
				var env struct {
					Ret int `json:"ret"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
				assert.Equal(t, 0, env.Ret, "404 body must be failure envelope")
			}
		})
	}
}

// TestHandler_Create_NilChecker confirms the precheck is skipped when no checker
// is wired (defensive: keeps the handler usable even if wiring is absent),
// preserving prior behavior in that degenerate case.
func TestHandler_Create_NilChecker(t *testing.T) {
	var created bool
	repo := &mockTaskRepo{
		CreateFn: func(_ context.Context, _ *ProvisioningTask) error {
			created = true
			return nil
		},
	}
	h := NewHandler(repo, nil) // no SetDeviceChecker

	c, rec := newCreateRequest(t, fmt.Sprintf(`{"device_id":%q}`, uuid.New()))
	h.Create(c)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, created, "without a checker the task is created (precheck skipped)")
}

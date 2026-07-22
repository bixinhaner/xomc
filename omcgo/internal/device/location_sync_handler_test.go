package device

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLocationSyncAuditDetailsIncludesCoordinateTransition(t *testing.T) {
	before := &Location{Latitude: 1.25, Longitude: 2.5}
	after := &Location{Latitude: 3.75, Longitude: 4.5}
	reported := &ReportedLocation{Latitude: 3.75, Longitude: 4.5, Version: 8}

	details := locationSyncAuditDetails(&LocationSync{
		AcceptedBefore: before,
		Accepted:       after,
		Reported:       reported,
	}, 8)

	require.Equal(t, int64(8), details["reported_version"])
	require.Equal(t, map[string]float64{"latitude": 1.25, "longitude": 2.5}, details["accepted_before"])
	require.Equal(t, map[string]float64{"latitude": 3.75, "longitude": 4.5}, details["accepted_after"])
}

func TestAcceptLocationSyncHandlerReturnsUpdatedState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deviceID := uuid.New()
	repo := &fakeLocationSyncRepository{result: &LocationSync{Status: LocationSyncInSync}}
	h := NewHandler(NewDeviceService(nil, nil, nil, nil, zap.NewNop()))
	h.SetLocationSyncService(NewLocationSyncService(repo))
	r := gin.New()
	r.POST("/devices/:id/location-sync/accept", h.AcceptLocationSync)

	body, _ := json.Marshal(AcceptLocationRequest{ReportedVersion: 4})
	req := httptest.NewRequest(http.MethodPost, "/devices/"+deviceID.String()+"/location-sync/accept", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, deviceID, repo.gotID)
	require.Equal(t, int64(4), repo.gotVersion)
}

func TestAcceptLocationSyncHandlerMapsVersionConflictTo409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeLocationSyncRepository{err: errors.Join(commonerrors.ErrAlreadyExists, errors.New("reported location version conflict"))}
	h := NewHandler(NewDeviceService(nil, nil, nil, nil, zap.NewNop()))
	h.SetLocationSyncService(NewLocationSyncService(repo))
	r := gin.New()
	r.POST("/devices/:id/location-sync/accept", h.AcceptLocationSync)

	body, _ := json.Marshal(AcceptLocationRequest{ReportedVersion: 4})
	req := httptest.NewRequest(http.MethodPost, "/devices/"+uuid.New().String()+"/location-sync/accept", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusConflict, resp.Code)
}

var _ LocationSyncRepository = (*fakeLocationSyncRepository)(nil)
var _ LocationObservationRepository = (*fakeLocationSyncRepository)(nil)

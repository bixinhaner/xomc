package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type thirdPartyLocationStoreStub struct {
	mu      sync.Mutex
	items   []ThirdPartyLocationDevice
	errBySN map[string]error
}

type thirdPartyLocationBatchRepositoryStub struct {
	key    string
	hash   string
	result ThirdPartyLocationResult
}

func (s *thirdPartyLocationBatchRepositoryStub) Claim(
	_ context.Context,
	key string,
	hash string,
) (*ThirdPartyLocationResult, error) {
	if s.key == "" {
		s.key, s.hash = key, hash
		return nil, nil
	}
	if s.key != key {
		return nil, fmt.Errorf("unexpected idempotency key")
	}
	if s.hash != hash {
		return nil, ErrThirdPartyLocationBatchConflict
	}
	return &s.result, nil
}

func (s *thirdPartyLocationBatchRepositoryStub) Complete(
	_ context.Context,
	key string,
	result ThirdPartyLocationResult,
) error {
	s.key = key
	s.result = result
	return nil
}

func (s *thirdPartyLocationStoreStub) SaveThirdPartyLocation(
	_ context.Context,
	item ThirdPartyLocationDevice,
	_ time.Time,
	_ float64,
	_ float64,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errBySN[item.SerialNumber]; err != nil {
		return err
	}
	s.items = append(s.items, item)
	return nil
}

func validThirdPartyLocationRequest() ThirdPartyLocationRequest {
	return ThirdPartyLocationRequest{Devices: []ThirdPartyLocationDevice{{
		VesselName:   "浙渔001",
		SerialNumber: "SN202501130001",
		Longitude:    "121.2347",
		Latitude:     "31.2345",
		UpdateTime:   "2025-11-25 19:55:00",
	}}}
}

func TestUpdateThirdPartyLocationsReturnsPerDeviceResults(t *testing.T) {
	store := &thirdPartyLocationStoreStub{
		errBySN: map[string]error{"missing": context.Canceled},
	}
	service := NewService(nil, nil)
	service.SetThirdPartyLocationStore(store)
	request := validThirdPartyLocationRequest()
	request.Devices = append(request.Devices, ThirdPartyLocationDevice{
		VesselName:   "浙渔002",
		SerialNumber: "missing",
		Longitude:    "121.3478",
		Latitude:     "31.3458",
		UpdateTime:   "2025-11-25 19:55:00",
	})

	result, err := service.UpdateThirdPartyLocations(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, 2, result.TotalCount)
	require.Equal(t, 1, result.SuccessCount)
	require.Equal(t, 1, result.FailCount)
	require.Equal(t, "missing", result.FailedDevices[0].SerialNumber)
	require.Len(t, store.items, 1)
}

func TestUpdateThirdPartyLocationsIdempotencyReplaysWithoutSavingAgain(t *testing.T) {
	store := &thirdPartyLocationStoreStub{}
	batches := &thirdPartyLocationBatchRepositoryStub{}
	service := NewService(nil, nil)
	service.SetThirdPartyLocationStore(store)
	service.SetThirdPartyLocationBatchRepository(batches)
	request := validThirdPartyLocationRequest()

	first, replayed, err := service.UpdateThirdPartyLocationsIdempotent(
		context.Background(), request, "batch-1",
	)
	require.NoError(t, err)
	require.False(t, replayed)
	require.Equal(t, 1, first.SuccessCount)
	require.Len(t, store.items, 1)

	second, replayed, err := service.UpdateThirdPartyLocationsIdempotent(
		context.Background(), request, "batch-1",
	)
	require.NoError(t, err)
	require.True(t, replayed)
	require.Equal(t, first, second)
	require.Len(t, store.items, 1)
}

func TestUpdateThirdPartyLocationsIdempotencyRejectsChangedRequest(t *testing.T) {
	batches := &thirdPartyLocationBatchRepositoryStub{}
	service := NewService(nil, nil)
	service.SetThirdPartyLocationStore(&thirdPartyLocationStoreStub{})
	service.SetThirdPartyLocationBatchRepository(batches)
	request := validThirdPartyLocationRequest()

	_, _, err := service.UpdateThirdPartyLocationsIdempotent(context.Background(), request, "batch-1")
	require.NoError(t, err)
	request.Devices[0].Latitude = "32.0000"
	_, _, err = service.UpdateThirdPartyLocationsIdempotent(context.Background(), request, "batch-1")
	require.ErrorIs(t, err, ErrThirdPartyLocationBatchConflict)
}

func TestUpdateThirdPartyLocationsRejectsMoreThan1000Devices(t *testing.T) {
	request := validThirdPartyLocationRequest()
	request.Devices = make([]ThirdPartyLocationDevice, thirdPartyLocationMaxDevices+1)
	service := NewService(nil, nil)
	service.SetThirdPartyLocationStore(&thirdPartyLocationStoreStub{})

	_, err := service.UpdateThirdPartyLocations(context.Background(), request)

	require.Error(t, err)
	require.Contains(t, err.Error(), "1000")
}

func TestUpdateThirdPartyLocationsRejectsDuplicateSerialNumbers(t *testing.T) {
	store := &thirdPartyLocationStoreStub{}
	service := NewService(nil, nil)
	service.SetThirdPartyLocationStore(store)
	request := validThirdPartyLocationRequest()
	request.Devices = append(request.Devices, request.Devices[0])

	result, err := service.UpdateThirdPartyLocations(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	require.Equal(t, 1, result.FailCount)
	require.Equal(t, request.Devices[1].SerialNumber, result.FailedDevices[0].SerialNumber)
	require.Contains(t, result.FailedDevices[0].Reason, "duplicate")
	require.Len(t, store.items, 1)
}

func TestUpdateThirdPartyLocationsCountsInvalidItemWithoutSerialNumber(t *testing.T) {
	service := NewService(nil, nil)
	service.SetThirdPartyLocationStore(&thirdPartyLocationStoreStub{})
	request := validThirdPartyLocationRequest()
	request.Devices = append(request.Devices, ThirdPartyLocationDevice{
		VesselName: "浙渔002",
		Longitude:  "121.3478",
		Latitude:   "31.3458",
		UpdateTime: "2025-11-25 19:55:00",
	})

	result, err := service.UpdateThirdPartyLocations(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, result.TotalCount, result.SuccessCount+result.FailCount)
	require.Equal(t, 1, result.FailCount)
	require.Len(t, result.FailedDevices, 1)
	require.Empty(t, result.FailedDevices[0].SerialNumber)
}

func TestUpdateThirdPartyLocationsNormalizesDeviceIdentifiers(t *testing.T) {
	store := &thirdPartyLocationStoreStub{}
	service := NewService(nil, nil)
	service.SetThirdPartyLocationStore(store)
	request := validThirdPartyLocationRequest()
	request.Devices[0].VesselName = "  浙渔001  "
	request.Devices[0].SerialNumber = "  SN202501130001  "

	result, err := service.UpdateThirdPartyLocations(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	require.Equal(t, "浙渔001", store.items[0].VesselName)
	require.Equal(t, "SN202501130001", store.items[0].SerialNumber)
}

func TestParseThirdPartyLocationRejectsInvalidInput(t *testing.T) {
	cases := []ThirdPartyLocationDevice{
		{SerialNumber: "SN", VesselName: "v", Longitude: "181", Latitude: "1", UpdateTime: "2025-11-25 19:55:00"},
		{SerialNumber: "SN", VesselName: "v", Longitude: "1", Latitude: "91", UpdateTime: "2025-11-25 19:55:00"},
		{SerialNumber: "SN", VesselName: "v", Longitude: "1", Latitude: "1", UpdateTime: "bad"},
	}
	for _, item := range cases {
		_, _, _, err := parseThirdPartyLocation(item)
		require.Error(t, err)
	}
}

func TestBatchUpdateDeviceLocationRouteUsesAPIEnvelope(t *testing.T) {
	service := NewService(nil, nil)
	service.SetThirdPartyLocationStore(&thirdPartyLocationStoreStub{})
	router := gin.New()
	NewHandler(service, nil).RegisterThirdPartyRoutes(router.Group(""))
	body, err := json.Marshal(validThirdPartyLocationRequest())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/fence/batchUpdateDeviceLocation", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var envelope struct {
		Success bool                     `json:"success"`
		Data    ThirdPartyLocationResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &envelope))
	require.True(t, envelope.Success)
	require.Equal(t, 1, envelope.Data.SuccessCount)
}

func TestBatchUpdateDeviceLocationRouteNormalizesLegacyAdapterAliases(t *testing.T) {
	store := &thirdPartyLocationStoreStub{}
	service := NewService(nil, nil)
	service.SetThirdPartyLocationStore(store)
	router := gin.New()
	NewHandler(service, nil).RegisterThirdPartyRoutes(router.Group(""))
	body := `{"devices":[{"serialName":"legacy-vessel","serialNumber":"SN-LEGACY",` +
		`"longitude":"121.2345","latitude":"31.2345",` +
		`"updatetime":"2025-11-25 19:55:00"}]}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/fence/batchUpdateDeviceLocation",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, store.items, 1)
	require.Equal(t, "legacy-vessel", store.items[0].VesselName)
	require.Equal(t, "2025-11-25 19:55:00", store.items[0].UpdateTime)
}

func TestThirdPartyLocationAdapterPrefersCanonicalFieldsOverAliases(t *testing.T) {
	payload := thirdPartyLocationRequestPayload{
		Devices: []thirdPartyLocationDevicePayload{{
			VesselName:       "canonical-vessel",
			LegacySerialName: "legacy-vessel",
			UpdateTime:       "2025-11-25 19:55:00",
			LegacyUpdateTime: "2025-11-25 18:55:00",
		}},
	}

	request := payload.canonicalRequest()

	require.Equal(t, "canonical-vessel", request.Devices[0].VesselName)
	require.Equal(t, "2025-11-25 19:55:00", request.Devices[0].UpdateTime)
}

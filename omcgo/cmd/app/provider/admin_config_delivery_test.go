package provider

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/admin"
	coreevent "github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type configDeliveryBus struct {
	publishErr error
	subject    string
	event      coreevent.Event
}

func (b *configDeliveryBus) Publish(_ context.Context, subject string, evt coreevent.Event) error {
	b.subject = subject
	b.event = evt
	return b.publishErr
}

func (b *configDeliveryBus) Subscribe(string, coreevent.EventHandler) (coreevent.Subscription, error) {
	return nil, nil
}

func (b *configDeliveryBus) QueueSubscribe(string, string, coreevent.EventHandler) (coreevent.Subscription, error) {
	return nil, nil
}

func (b *configDeliveryBus) PullSubscribe(string, string, coreevent.EventHandler) (coreevent.Subscription, error) {
	return nil, nil
}

func (b *configDeliveryBus) Close() error { return nil }

type acsTransferValidationRepo struct {
	batchCalled bool
}

func (r *acsTransferValidationRepo) Create(context.Context, *admin.SysConfig) error { return nil }
func (r *acsTransferValidationRepo) GetByID(context.Context, uuid.UUID) (*admin.SysConfig, error) {
	return nil, nil
}
func (r *acsTransferValidationRepo) GetByKey(context.Context, string, string) (*admin.SysConfig, error) {
	return nil, nil
}
func (r *acsTransferValidationRepo) List(_ context.Context, category string, _ bool) ([]admin.SysConfig, error) {
	return []admin.SysConfig{
		{Category: category, Key: transfercfg.KeyUploadBaseURL, Value: "http://acs.example.com:8080"},
		{Category: category, Key: transfercfg.KeyDownloadBaseURL, Value: "http://acs.example.com:8080"},
	}, nil
}
func (r *acsTransferValidationRepo) Update(context.Context, *admin.SysConfig) error { return nil }
func (r *acsTransferValidationRepo) Delete(context.Context, uuid.UUID) error        { return nil }
func (r *acsTransferValidationRepo) BatchUpsert(context.Context, string, []admin.BatchItem) (int, error) {
	r.batchCalled = true
	return 1, nil
}

func TestACSTransferProductionValidatorsRejectInvalidBatchBeforeSave(t *testing.T) {
	repo := &acsTransferValidationRepo{}
	service := admin.NewSysConfigService(repo)
	registerACSTransferValidators(service)
	savedHookCalled := false
	service.RegisterSavedHook(func(context.Context, string) { savedHookCalled = true })
	handler := admin.NewSysConfigHandler(service)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/admin"))

	req := httptest.NewRequest(http.MethodPost, "/admin/sysConfig/batch", bytes.NewBufferString(
		`{"category":"acs_transfer","items":[{"key":"protocolPolicy","value":"prefer_https"}]}`,
	))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.False(t, repo.batchCalled)
	assert.False(t, savedHookCalled)
}

func TestACSTransferProductionValidatorsRejectMalformedHTTPAndHTTPSURLs(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			name: "HTTP port zero",
			body: `{"category":"acs_transfer","items":[{"key":"uploadBaseURL","value":"http://acs.example.com:0"}]}`,
		},
		{
			name: "HTTPS port overflow",
			body: `{"category":"acs_transfer","items":[{"key":"httpsUploadBaseURL","value":"https://acs.example.com:65536"}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &acsTransferValidationRepo{}
			service := admin.NewSysConfigService(repo)
			registerACSTransferValidators(service)
			savedHookCalled := false
			service.RegisterSavedHook(func(context.Context, string) { savedHookCalled = true })
			handler := admin.NewSysConfigHandler(service)
			router := gin.New()
			handler.RegisterRoutes(router.Group("/admin"))

			req := httptest.NewRequest(http.MethodPost, "/admin/sysConfig/batch", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.False(t, repo.batchCalled)
			assert.False(t, savedHookCalled)
		})
	}
}

func TestACSConfigDeliveryHandlerPublishesDurableBatchIdentity(t *testing.T) {
	bus := &configDeliveryBus{}
	handler := newACSConfigDeliveryHandler(bus)
	work := admin.ConfigApplyWork{Batch: admin.ConfigApplyBatch{
		ID:            uuid.New(),
		Category:      "acs_transfer",
		ConfigVersion: 17,
	}}

	actual, err := handler(context.Background(), work)
	require.NoError(t, err)
	require.Equal(t, coreevent.SubjectSysConfigSaved, bus.subject)
	require.Equal(t, "published", actual["delivery"])
	require.Equal(t, bus.event.ID, actual["event_id"])

	var payload coreevent.SysConfigSavedPayload
	require.NoError(t, bus.event.DecodePayload(&payload))
	require.Equal(t, work.Batch.Category, payload.Category)
	require.Equal(t, work.Batch.ID, payload.BatchID)
	require.Equal(t, work.Batch.ConfigVersion, payload.ConfigVersion)
}

func TestACSConfigDeliveryHandlerReturnsPublishFailure(t *testing.T) {
	bus := &configDeliveryBus{publishErr: errors.New("nats unavailable")}
	handler := newACSConfigDeliveryHandler(bus)

	_, err := handler(context.Background(), admin.ConfigApplyWork{Batch: admin.ConfigApplyBatch{
		ID:            uuid.New(),
		Category:      "acs_transfer",
		ConfigVersion: 18,
	}})

	require.ErrorContains(t, err, "publish ACS transfer config event")
	require.ErrorContains(t, err, "nats unavailable")
}

func TestACSConfigDeliveryHandlerFailsWhenEventBusUnavailable(t *testing.T) {
	handler := newACSConfigDeliveryHandler(nil)

	_, err := handler(context.Background(), admin.ConfigApplyWork{Batch: admin.ConfigApplyBatch{
		ID:            uuid.New(),
		Category:      "acs_transfer",
		ConfigVersion: 19,
	}})

	require.ErrorContains(t, err, "event bus is unavailable")
}

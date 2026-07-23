package integration

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/pkg/soap"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestInformBootstrapRegister tests the full chain:
// Parse Bootstrap Inform → register device in DB → verify InformResponse.
//
// This test requires a running PostgreSQL instance.
// Set OMCGO_TEST_DB_DSN to run (e.g. "postgres://omcgo:omcgo123@localhost:5432/omcgo_test?sslmode=disable").
func TestInformBootstrapRegister(t *testing.T) {
	dsn := os.Getenv("OMCGO_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TEST_DB_DSN not set, skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to DB
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	// Clean up test data
	pool.Exec(ctx, "DELETE FROM device_parameters WHERE device_id IN (SELECT id FROM devices WHERE serial_number = 'TEST-SN-001')")
	pool.Exec(ctx, "DELETE FROM devices WHERE serial_number = 'TEST-SN-001'")

	logger, _ := zap.NewDevelopment()

	// 1. Parse the bootstrap Inform fixture
	fixtureData, err := os.ReadFile("../../test/fixtures/soap/inform_bootstrap.xml")
	require.NoError(t, err)

	inform, cwmpID, err := soap.DecodeInform(bytes.NewReader(fixtureData))
	require.NoError(t, err)
	assert.NotEmpty(t, cwmpID)
	assert.Equal(t, "TEST-SN-001", inform.DeviceId.SerialNumber)
	assert.Equal(t, "001122", inform.DeviceId.OUI)
	assert.True(t, tr069.IsBootstrap(inform.Event))

	// 2. Register device via DeviceService
	deviceRepo := device.NewPgDeviceRepository(pool)
	paramRepo := device.NewPgDeviceParameterRepository(pool)
	svc := device.NewDeviceService(deviceRepo, paramRepo, nil, nil, logger)

	registration, err := svc.RegisterFromInform(ctx, inform, model.CarrierCMCC)
	require.NoError(t, err)
	require.True(t, registration.Created)
	registered := registration.Device
	assert.Equal(t, "TEST-SN-001", registered.SerialNumber)
	assert.Equal(t, "001122", registered.OUI)
	assert.Equal(t, model.CarrierCMCC, registered.Carrier)
	assert.Equal(t, model.DeviceDiscovered, registered.Status)
	assert.NotNil(t, registered.LastInformAt)

	// 3. Verify device is persisted in DB
	fetched, err := deviceRepo.GetBySerialNumber(ctx, "TEST-SN-001")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, registered.ID, fetched.ID)
	assert.Equal(t, model.DeviceDiscovered, fetched.Status)
	assert.Equal(t, model.CarrierCMCC, fetched.Carrier)

	// 4. Verify parameters were stored
	params, err := paramRepo.GetByDevice(ctx, registered.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, params)

	// 5. Verify InformResponse can be rendered
	resp, err := soap.RenderResponse(soap.InformResponseTmpl, soap.InformResponseData{ID: cwmpID})
	require.NoError(t, err)
	assert.Contains(t, string(resp), "InformResponse")
	assert.Contains(t, string(resp), "MaxEnvelopes")

	// 6. Verify ChannelEventBus publish/subscribe works with Inform events
	bus := event.NewChannelEventBus(100, logger)
	defer bus.Close()

	received := make(chan event.Event, 1)
	_, err = bus.Subscribe(event.SubjectDeviceBootstrap, func(ctx context.Context, evt event.Event) error {
		received <- evt
		return nil
	})
	require.NoError(t, err)

	evt, err := event.NewEvent(event.SubjectDeviceBootstrap, map[string]interface{}{
		"device_id": inform.DeviceId,
		"events":    tr069.EventCodes(inform.Event),
	})
	require.NoError(t, err)

	err = bus.Publish(ctx, event.SubjectDeviceBootstrap, evt)
	require.NoError(t, err)

	select {
	case got := <-received:
		assert.Equal(t, event.SubjectDeviceBootstrap, got.Subject)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	// 7. Clean up
	pool.Exec(ctx, "DELETE FROM device_parameters WHERE device_id = $1", registered.ID)
	pool.Exec(ctx, "DELETE FROM devices WHERE id = $1", registered.ID)
}

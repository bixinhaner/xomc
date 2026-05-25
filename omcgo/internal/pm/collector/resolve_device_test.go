package collector

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// fakeDeviceLookup lets tests script GetBySerialNumber outcomes.
type fakeDeviceLookup struct {
	dev *model.Device
	err error
}

func (f *fakeDeviceLookup) GetBySerialNumber(_ context.Context, _ string) (*model.Device, error) {
	return f.dev, f.err
}

func TestResolveDevice_fatPayloadPassthrough(t *testing.T) {
	// transfer.Bridge path: device_id already populated → resolveDevice is a no-op
	// (no DeviceLookup call). We assert by passing a lookup that would explode
	// if called.
	exploder := &fakeDeviceLookup{err: errors.New("must not be called")}
	c := &PMCollector{logger: zap.NewNop(), deviceLookup: exploder}

	payload := &FileReceivedPayload{
		DeviceID:   uuid.New().String(),
		DeviceSN:   "SN-ignored",
		DeviceOUI:  "ABCDEF",
		Carrier:    "cmcc",
		Technology: "lte",
	}
	originalID := payload.DeviceID

	require.NoError(t, c.resolveDevice(context.Background(), payload))
	assert.Equal(t, originalID, payload.DeviceID, "device_id must not be overwritten when already set")
}

func TestResolveDevice_thinPayloadFillsFields(t *testing.T) {
	// acs.upload.Handler path: device_id empty, only device_sn known.
	// resolveDevice must call DeviceLookup and write back ID / OUI / Carrier / Technology.
	devID := uuid.New()
	lookup := &fakeDeviceLookup{dev: &model.Device{
		ID:           devID,
		SerialNumber: "1202000240194DP0015",
		OUI:          "48BF74",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
	}}
	c := &PMCollector{logger: zap.NewNop(), deviceLookup: lookup}

	payload := &FileReceivedPayload{DeviceSN: "1202000240194DP0015"}
	require.NoError(t, c.resolveDevice(context.Background(), payload))

	assert.Equal(t, devID.String(), payload.DeviceID)
	assert.Equal(t, "48BF74", payload.DeviceOUI)
	assert.Equal(t, string(model.CarrierCMCC), payload.Carrier)
	assert.Equal(t, string(model.TechLTE), payload.Technology)
}

func TestResolveDevice_thinPayloadNoLookupErrors(t *testing.T) {
	// Defensive: if upload handler ever publishes a thin payload but worker
	// hasn't wired DeviceLookup, fail loudly so the misconfiguration shows up
	// in DLQ / logs instead of silently dropping PM data.
	c := &PMCollector{logger: zap.NewNop()}
	payload := &FileReceivedPayload{DeviceSN: "SN1"}

	err := c.resolveDevice(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no DeviceLookup wired")
}

func TestResolveDevice_emptySNErrors(t *testing.T) {
	c := &PMCollector{logger: zap.NewNop(), deviceLookup: &fakeDeviceLookup{}}
	payload := &FileReceivedPayload{}

	err := c.resolveDevice(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both device_id and device_sn empty")
}

func TestResolveDevice_deviceNotFoundErrors(t *testing.T) {
	// Transient: device row not yet inserted. Returning error makes the
	// runner retry; by the time the retry fires the inform/registration
	// usually landed.
	c := &PMCollector{logger: zap.NewNop(), deviceLookup: &fakeDeviceLookup{dev: nil}}
	payload := &FileReceivedPayload{DeviceSN: "UnknownSN"}

	err := c.resolveDevice(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device not found")
}

func TestResolveDevice_lookupErrorPropagates(t *testing.T) {
	c := &PMCollector{logger: zap.NewNop(), deviceLookup: &fakeDeviceLookup{err: errors.New("PG down")}}
	payload := &FileReceivedPayload{DeviceSN: "SN1"}

	err := c.resolveDevice(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PG down")
}

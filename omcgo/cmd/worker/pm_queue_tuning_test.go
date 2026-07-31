package main

import (
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/pm/collector"
	"github.com/stretchr/testify/assert"
)

func TestPMQueueTuningCoversDeviceRegistrationWindow(t *testing.T) {
	tuning := pmQueueTuning(32)

	assert.Equal(t, 2*time.Minute, tuning.AckWait)
	assert.Equal(
		t,
		event.MaxDeliveriesForRetryHorizon(collector.DeviceRegistrationGrace),
		tuning.MaxDeliver,
	)
	assert.Equal(t, 128, tuning.MaxAckPending)
}

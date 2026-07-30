package appconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGPVResponseConsumerConfigDefaultsAreBoundedAndLossless(t *testing.T) {
	got := (GPVResponseConsumerConfig{}).Defaults()

	require.Equal(t, "provision-gpv", got.ProvisionQueue)
	require.Equal(t, 2, got.ProvisionConcurrency)
	require.Equal(t, 256, got.ProvisionQueueDepth)
	require.Equal(t, "device-rpc-gpv", got.RPCDurable)
	require.Zero(t, got.RPCStartSequence)
	require.Equal(t, 2, got.RPCConcurrency)
	require.Equal(t, 1000, got.RPCQueueDepth)
	require.Equal(t, 30*time.Second, got.AckWait)
	require.Equal(t, 5, got.MaxDeliver)
	require.Equal(t, 2000, got.MaxAckPending)
}

func TestGPVResponseConsumerConfigPreservesExplicitHandoff(t *testing.T) {
	got := (GPVResponseConsumerConfig{
		ProvisionQueue:       "existing-provision",
		ProvisionConcurrency: 3,
		ProvisionQueueDepth:  99,
		RPCDurable:           "Lo0yTlW5-handoff",
		RPCStartSequence:     416825,
		RPCConcurrency:       4,
		RPCQueueDepth:        88,
		AckWait:              time.Minute,
		MaxDeliver:           7,
		MaxAckPending:        3000,
	}).Defaults()

	require.Equal(t, "existing-provision", got.ProvisionQueue)
	require.Equal(t, 3, got.ProvisionConcurrency)
	require.Equal(t, 99, got.ProvisionQueueDepth)
	require.Equal(t, "Lo0yTlW5-handoff", got.RPCDurable)
	require.Equal(t, uint64(416825), got.RPCStartSequence)
	require.Equal(t, 4, got.RPCConcurrency)
	require.Equal(t, 88, got.RPCQueueDepth)
	require.Equal(t, time.Minute, got.AckWait)
	require.Equal(t, 7, got.MaxDeliver)
	require.Equal(t, 3000, got.MaxAckPending)
}

func TestProductionConfigLoadsGPVDurableHandoffFromEnvironment(t *testing.T) {
	t.Setenv("OMCGO_JWT_SECRET", "test-only-jwt-secret-with-at-least-32-bytes")
	t.Setenv("GPV_RPC_DURABLE", "device-rpc-gpv-cutover")
	t.Setenv("GPV_RPC_START_SEQUENCE", "416825")
	t.Setenv("GPV_RPC_CONCURRENCY", "3")
	t.Setenv("GPV_PROVISION_CONCURRENCY", "4")

	var config AppConfig
	require.NoError(t, Load("../../../cmd/app/etc/config.prod.yaml", &config))

	require.Equal(t, "device-rpc-gpv-cutover", config.Provision.GPVResponse.RPCDurable)
	require.Equal(t, uint64(416825), config.Provision.GPVResponse.RPCStartSequence)
	require.Equal(t, 3, config.Provision.GPVResponse.RPCConcurrency)
	require.Equal(t, 4, config.Provision.GPVResponse.ProvisionConcurrency)
}

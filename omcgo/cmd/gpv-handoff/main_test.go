package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadHandoffConfigIgnoresUnrelatedAppSecrets(t *testing.T) {
	t.Setenv("OMCGO_JWT_SECRET", "")

	path := filepath.Join(t.TempDir(), "app.prod.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
nats:
  url: nats://nats:4222
jwt:
  secret: "${OMCGO_JWT_SECRET}"
provision:
  gpv_response:
    provision_queue: provision-custom
    rpc_durable: rpc-custom
    ack_wait: 45s
    max_deliver: 7
    max_ack_pending: 3000
`), 0o600))

	config, err := loadHandoffConfig(path)
	require.NoError(t, err)
	require.Equal(t, "nats://nats:4222", config.NATS.URL)

	gpv := config.Provision.GPVResponse.Defaults()
	require.Equal(t, "provision-custom", gpv.ProvisionQueue)
	require.Equal(t, "rpc-custom", gpv.RPCDurable)
	require.Equal(t, 7, gpv.MaxDeliver)
	require.Equal(t, 3000, gpv.MaxAckPending)
}

func TestLoadHandoffConfigRequiresNATSURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.prod.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
provision:
  gpv_response:
    rpc_durable: rpc-custom
`), 0o600))

	_, err := loadHandoffConfig(path)
	require.ErrorContains(t, err, "nats.url must not be empty")
}

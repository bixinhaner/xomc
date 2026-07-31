package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
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

func TestRunFreshInstallBootstrapsEmptyJetStream(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}

	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)
	resetJetStream(t, js)
	_, err = js.StreamNameBySubject("command.get_parameters.response")
	require.ErrorIs(t, err, nats.ErrNoMatchingStream)

	path := filepath.Join(t.TempDir(), "app.prod.yaml")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`
nats:
  url: %s
provision:
  gpv_response:
    rpc_durable: rpc-fresh-install
`, url)), 0o600))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, run(ctx, path, "", true))

	stream, err := js.StreamNameBySubject("command.get_parameters.response")
	require.NoError(t, err)
	require.Equal(t, "COMMAND", stream)
	consumer, err := js.ConsumerInfo(stream, "rpc-fresh-install")
	require.NoError(t, err)
	require.Equal(t, nats.DeliverNewPolicy, consumer.Config.DeliverPolicy)
}

func TestRunBootstrapIfMissingBootstrapsEmptyJetStream(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}

	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)
	resetJetStream(t, js)

	path := filepath.Join(t.TempDir(), "app.prod.yaml")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`
nats:
  url: %s
provision:
  gpv_response:
    rpc_durable: rpc-bootstrap-if-missing
`, url)), 0o600))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, runWithOptions(ctx, path, "", false, true))

	stream, err := js.StreamNameBySubject("command.get_parameters.response")
	require.NoError(t, err)
	consumer, err := js.ConsumerInfo(stream, "rpc-bootstrap-if-missing")
	require.NoError(t, err)
	require.Equal(t, nats.DeliverNewPolicy, consumer.Config.DeliverPolicy)
}

func TestRunBootstrapIfMissingDoesNotSkipExistingMessages(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}

	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)
	resetJetStream(t, js)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      "COMMAND",
		Subjects:  []string{"command.>"},
		Retention: nats.LimitsPolicy,
	})
	require.NoError(t, err)
	_, err = js.Publish("command.get_parameters.response", []byte("retained"))
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "app.prod.yaml")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`
nats:
  url: %s
provision:
  gpv_response:
    rpc_durable: rpc-must-not-skip
`, url)), 0o600))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = runWithOptions(ctx, path, "", false, true)
	require.ErrorContains(t, err, "refusing DeliverNew")
	_, consumerErr := js.ConsumerInfo("COMMAND", "rpc-must-not-skip")
	require.ErrorIs(t, consumerErr, nats.ErrConsumerNotFound)
}

func resetJetStream(t *testing.T, js nats.JetStreamContext) {
	t.Helper()
	for name := range js.StreamNames() {
		require.NoError(t, js.DeleteStream(name))
	}
}

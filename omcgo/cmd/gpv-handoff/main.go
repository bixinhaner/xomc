package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	natscomponent "github.com/omcgo/omcgo/internal/core/components/nats"
	"github.com/omcgo/omcgo/internal/core/event"
)

func main() {
	var configPath string
	var sourceConsumer string
	var freshInstall bool
	var bootstrapIfMissing bool
	flag.StringVar(&configPath, "config", "/etc/omcgo/app.prod.yaml", "app configuration path")
	flag.StringVar(
		&sourceConsumer,
		"source-consumer",
		os.Getenv("GPV_RPC_SOURCE_CONSUMER"),
		"legacy GPV RPC consumer name; empty enables safe auto-discovery",
	)
	flag.BoolVar(
		&freshInstall,
		"fresh-install",
		false,
		"explicitly allow DeliverNew when initializing a fresh environment",
	)
	flag.BoolVar(
		&bootstrapIfMissing,
		"bootstrap-if-missing",
		false,
		"bootstrap streams only when the GPV subject has no stream; preserve strict handoff when it exists",
	)
	flag.Parse()

	if err := runWithOptions(
		context.Background(),
		configPath,
		sourceConsumer,
		freshInstall,
		bootstrapIfMissing,
	); err != nil {
		fmt.Fprintf(os.Stderr, "GPV handoff failed: %v\n", err)
		os.Exit(1)
	}
}

// handoffConfig deliberately contains only the configuration required to
// create the GPV durable. The handoff command runs before the new app starts
// and must not depend on unrelated application secrets or backing services.
type handoffConfig struct {
	NATS      appconfig.NATSConfig `mapstructure:"nats"`
	Provision struct {
		GPVResponse appconfig.GPVResponseConsumerConfig `mapstructure:"gpv_response"`
	} `mapstructure:"provision"`
}

func loadHandoffConfig(path string) (handoffConfig, error) {
	var config handoffConfig
	if err := appconfig.Load(path, &config); err != nil {
		return handoffConfig{}, err
	}
	if strings.TrimSpace(config.NATS.URL) == "" {
		return handoffConfig{}, fmt.Errorf("nats.url must not be empty")
	}
	return config, nil
}

func run(ctx context.Context, configPath, sourceConsumer string, freshInstall bool) error {
	return runWithOptions(ctx, configPath, sourceConsumer, freshInstall, false)
}

func runWithOptions(
	ctx context.Context,
	configPath string,
	sourceConsumer string,
	freshInstall bool,
	bootstrapIfMissing bool,
) error {
	config, err := loadHandoffConfig(configPath)
	if err != nil {
		return fmt.Errorf("load app config: %w", err)
	}
	gpv := config.Provision.GPVResponse.Defaults()
	if freshInstall || bootstrapIfMissing {
		client, bootstrapErr := natscomponent.NewNATSClient(config.NATS, zap.NewNop())
		if bootstrapErr != nil {
			return fmt.Errorf("connect NATS for stream bootstrap check: %w", bootstrapErr)
		}
		defer client.Conn.Close()
		if bootstrapIfMissing && !freshInstall {
			_, lookupErr := client.JS.StreamNameBySubject(
				event.SubjectCommandGetParamsResponse,
				nats.Context(ctx),
			)
			switch {
			case lookupErr == nil:
			case errors.Is(lookupErr, nats.ErrNoMatchingStream):
				freshInstall = true
			default:
				return fmt.Errorf("check GPV stream before bootstrap: %w", lookupErr)
			}
		}
		if freshInstall {
			if bootstrapErr := client.EnsureStreams(ctx, false); bootstrapErr != nil {
				return fmt.Errorf("bootstrap fresh install streams: %w", bootstrapErr)
			}
		}
	}
	nc, err := nats.Connect(
		config.NATS.URL,
		nats.Name("omcgo-gpv-handoff"),
		nats.Timeout(5*time.Second),
		nats.MaxReconnects(0),
	)
	if err != nil {
		return fmt.Errorf("connect NATS: %w", err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("open JetStream management context: %w", err)
	}

	result, err := event.PrepareGPVHandoff(ctx, js, event.GPVHandoffConfig{
		Subject:              event.SubjectCommandGetParamsResponse,
		TargetDurable:        gpv.RPCDurable,
		SourceConsumer:       sourceConsumer,
		ProvisionPullDurable: pullDurableName(gpv.ProvisionQueue),
		FreshInstall:         freshInstall,
		AckWait:              gpv.AckWait,
		MaxDeliver:           gpv.MaxDeliver,
		MaxAckPending:        gpv.MaxAckPending,
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func pullDurableName(queue string) string {
	if strings.HasSuffix(queue, "-pull") {
		return queue
	}
	return queue + "-pull"
}

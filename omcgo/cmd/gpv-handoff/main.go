package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
)

func main() {
	var configPath string
	var sourceConsumer string
	var freshInstall bool
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
	flag.Parse()

	if err := run(context.Background(), configPath, sourceConsumer, freshInstall); err != nil {
		fmt.Fprintf(os.Stderr, "GPV handoff failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configPath, sourceConsumer string, freshInstall bool) error {
	var config appconfig.AppConfig
	if err := appconfig.Load(configPath, &config); err != nil {
		return fmt.Errorf("load app config: %w", err)
	}
	gpv := config.Provision.GPVResponse.Defaults()
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

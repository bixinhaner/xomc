package main

import (
	"context"
	"fmt"
	"os"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/appconfig"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/bootstrap"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/mr"
	mrcollector "github.com/omcgo/omcgo/internal/mr/collector"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/collector"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/report"
	"github.com/omcgo/omcgo/internal/transfer"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-worker",
		Short: "OMC Background Worker",
		Long:  "Background worker process for PM/MR file processing and KPI calculation",
		RunE:  runWorker,
	}

	rootCmd.Flags().String("config", "cmd/worker/etc/config.dev.yaml", "configuration file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runWorker(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Flags().GetString("config")
	var cfg appconfig.WorkerConfig
	if err := appconfig.Load(cfgPath, &cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	app, err := bootstrap.InitForWorker(context.Background(), &cfg)
	if err != nil {
		return err
	}
	defer app.Logger.Sync()
	app.Logger.Info("omcgo-worker starting", zap.String("config", cfgPath))

	// Register all event subscribers
	registerSubscribers(app, &cfg)

	app.Logger.Info("omcgo-worker ready, waiting for events...")
	return app.WaitAndShutdown(nil)
}

func registerSubscribers(app *bootstrap.App, cfg *appconfig.WorkerConfig) {
	logger := app.Logger

	// PM Collector
	counterRepo := counter.NewPgCounterRepository(app.TsPool)
	kpiRepo := kpi.NewPgKPIRepository(app.TsPool)
	kpiEngine := kpi.NewKPIEngine(counterRepo, kpiRepo, app.Carriers, logger)
	pmParser := collector.NewPMXMLParser()
	pmFileStore := pm.NewPgPMFileStore(app.PgPool)
	pmCollector := collector.NewPMCollector(app.MinIO, cfg.MinIO.Buckets.PMFiles, pmParser, counterRepo, kpiEngine, pmFileStore, app.EventBus, logger)
	if err := pmCollector.Subscribe(app.EventBus); err != nil {
		logger.Warn("subscribe PM collector", zap.Error(err))
	}
	logger.Info("PM collector started")

	// Alarm Receiver
	alarmPgStore := alarm.NewPgAlarmStore(app.PgPool, app.TsPool)
	alarmRedisStore := alarm.NewRedisAlarmStore(app.Redis)
	alarmEngine := alarm.NewAlarmEngine(alarmPgStore, alarmRedisStore, app.Carriers, app.EventBus, logger)
	alarmReceiver := alarm.NewAlarmReceiver(alarmEngine, logger)
	if err := alarmReceiver.Subscribe(app.EventBus); err != nil {
		logger.Warn("subscribe alarm receiver", zap.Error(err))
	}
	logger.Info("alarm receiver started")

	// MR Collector
	mrStore := mr.NewPgMRStore(app.PgPool, app.TsPool)
	mrCollector := mrcollector.NewMRCollector(app.MinIO, cfg.MinIO.Buckets.MRFiles, mrStore, app.EventBus, logger)
	if err := mrCollector.Subscribe(app.EventBus); err != nil {
		logger.Warn("subscribe MR collector", zap.Error(err))
	}
	logger.Info("MR collector started")

	// Transfer Bridge
	deviceRepo := device.NewPgDeviceRepository(app.PgPool)
	transferBridge := transfer.NewTransferBridge(
		deviceRepo, app.MinIO,
		cfg.MinIO.Buckets.PMFiles, cfg.MinIO.Buckets.MRFiles, cfg.MinIO.Buckets.Logs,
		app.EventBus, logger,
	)
	if err := transferBridge.Subscribe(app.EventBus); err != nil {
		logger.Warn("subscribe transfer bridge", zap.Error(err))
	}
	logger.Info("transfer bridge started")

	// Backup Executor
	backupTaskRepo := backup.NewPgTaskRepository(app.PgPool)
	cmdQueue := cmdqueue.NewRedisCommandQueue(app.Redis)
	connReqClient := connreq.NewClient(app.Redis, logger)
	backupExecutor := backup.NewBackupExecutor(
		backupTaskRepo, deviceRepo, cmdQueue, connReqClient,
		app.EventBus, logger,
	)
	if err := backupExecutor.Subscribe(app.EventBus); err != nil {
		logger.Warn("subscribe backup executor", zap.Error(err))
	}
	logger.Info("backup executor started")

	// Report Generator
	reportDefRepo := report.NewPgDefinitionRepository(app.PgPool)
	reportRecordRepo := report.NewPgRecordRepository(app.PgPool)
	reportBucket := cfg.MinIO.Buckets.Reports
	if reportBucket == "" {
		reportBucket = "reports"
	}
	reportGenerator := report.NewReportGenerator(
		reportRecordRepo, reportDefRepo, kpiRepo, alarmPgStore,
		app.MinIO, reportBucket,
		app.EventBus, logger,
	)
	if err := reportGenerator.Subscribe(app.EventBus); err != nil {
		logger.Warn("subscribe report generator", zap.Error(err))
	}
	logger.Info("report generator started")
}

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/core/reliability/dlq"
	"github.com/omcgo/omcgo/internal/core/reliability/runner"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/mr"
	mrcollector "github.com/omcgo/omcgo/internal/mr/collector"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/collector"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/report"
	"github.com/omcgo/omcgo/internal/task"
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

	// Handle LOG_OUTPUT_PATHS environment variable (comma-separated)
	if outputPaths := os.Getenv("OMCGO_LOG_OUTPUT_PATHS"); outputPaths != "" {
		cfg.Log.OutputPaths = parseStringSlice(outputPaths)
	}

	ctx := context.Background()
	w, err := initWorker(ctx, &cfg)
	if err != nil {
		return err
	}
	defer w.Logger.Sync()
	w.Logger.Info("omcgo-worker starting", zap.String("config", cfgPath))

	// 启动时把 device_tasks 里仍为 pending 的任务重灌进 Redis 设备队列。
	// ZScore 去重保证多 Worker/多次重启都不会重复入队；sent 任务不在此路径，
	// 由 CPE 重连时的 RecoverPendingTasks 走陈旧阈值恢复。
	if _, err := w.TaskService.RestorePendingQueues(ctx, 0); err != nil {
		w.Logger.Warn("restore pending task queues failed", zap.Error(err))
	}

	// Register all event subscribers
	registerSubscribers(w, &cfg)

	w.Logger.Info("omcgo-worker ready, waiting for events...")
	return w.WaitAndShutdown(nil)
}

func registerSubscribers(w *workerInfra, cfg *appconfig.WorkerConfig) {
	logger := w.Logger

	// PM Collector — wraps handler with retry+DLQ runner (T-0012 / R-106).
	counterRepo := counter.NewPgCounterRepository(w.TsPool)
	kpiRepo := kpi.NewPgKPIRepository(w.TsPool)
	kpiEngine := kpi.NewKPIEngine(counterRepo, kpiRepo, w.Carriers, logger)
	pmParser := collector.NewPMXMLParser()
	pmFileStore := pm.NewPgPMFileStore(w.PgPool)
	pmCollector := collector.NewPMCollector(w.MinIO, cfg.MinIO.Buckets.PMFiles, pmParser, counterRepo, kpiEngine, pmFileStore, w.EventBus, logger)
	pmMetrics := pm.NewPMMetrics(w.MetricsReg)
	pmCollector.SetMetrics(pmMetrics)

	// Runner wires retry + DLQ instrumentation around the PM handler.
	// dlqRepo + runnerMetrics are scoped to the worker process; admin handler
	// in app process reads the same dead_letters table directly.
	dlqRepo := dlq.NewPgRepository(w.PgPool)
	runnerMetrics := runner.NewMetrics(w.MetricsReg)
	pmRunner := runner.NewRunner("pm", reliability.DefaultRetryConfig(), dlqRepo, w.EventBus, runnerMetrics, logger)
	pmCollector.SetRunner(pmRunner)

	if err := pmCollector.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe PM collector", zap.Error(err))
	}
	logger.Info("PM collector started with retry+DLQ runner")

	// Alarm Receiver + Sync
	alarmPgStore := alarm.NewPgAlarmStore(w.PgPool, w.TsPool)
	alarmRedisStore := alarm.NewRedisAlarmStore(w.Redis)
	alarmEngine := alarm.NewAlarmEngine(alarmPgStore, alarmRedisStore, w.Carriers, w.EventBus, logger)
	alarmMetrics := alarm.NewAlarmMetrics(w.MetricsReg)
	alarmEngine.SetMetrics(alarmMetrics)

	// Alarm Sync Service (creates GPV tasks to query device alarms)
	alarmSyncService := alarm.NewAlarmSyncService(w.TaskService, w.Redis, w.EventBus, logger)
	if err := alarmSyncService.Subscribe(); err != nil {
		logger.Warn("subscribe alarm sync service", zap.Error(err))
	}

	// Alarm Sync Processor (handles GPV responses, applies diff)
	alarmSyncProcessor := alarm.NewAlarmSyncProcessor(alarmEngine, alarmPgStore, alarmSyncService, w.EventBus, logger)
	if err := alarmSyncProcessor.Start(context.Background()); err != nil {
		logger.Warn("start alarm sync processor", zap.Error(err))
	}

	// T-0098 P2-10：构造 AlarmDefinition Registry + ProductRegistry adapter，启用 fallback 决策。
	// 任何一步失败都仅记 WARN 后退化到旧路径（不阻塞 worker 启动）。
	alarmReceiver := alarm.NewAlarmReceiver(alarmEngine, w.EventBus, logger)
	alarmDefRepo := definition.NewPgRepository(w.PgPool)
	alarmDefMetrics := definition.NewRegistryMetrics(w.MetricsReg)
	alarmDefRegistry := definition.NewRegistry(alarmDefRepo, alarmDefMetrics, logger)
	if err := alarmDefRegistry.Refresh(context.Background()); err != nil {
		logger.Warn("alarm-definition registry refresh failed; fallback disabled", zap.Error(err))
	} else {
		productRepo := product.NewPgRepository(w.PgPool)
		productMetrics := product.NewRegistryMetrics(w.MetricsReg)
		// worker 进程告警频率低，无需 Redis L2，直接 NopCache。
		productRegistry := product.NewRegistry(productRepo, product.NopCache{}, productMetrics, logger)
		if err := productRegistry.Refresh(context.Background()); err != nil {
			logger.Warn("product registry refresh failed in worker; alarm fallback disabled", zap.Error(err))
		} else {
			productResolver := &definition.ProductRegistryAdapter{Registry: productRegistry}
			alarmReceiver = alarmReceiver.WithAlarmDefRegistry(alarmDefRegistry, productResolver)
			logger.Info("alarm-definition fallback enabled",
				zap.Int("definitions_loaded", alarmDefRegistry.Count()))
		}
	}
	if err := alarmReceiver.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe alarm receiver", zap.Error(err))
	}
	logger.Info("alarm receiver + sync started")

	// Expedited Alarm Receiver (real-time alarm notifications via VALUE CHANGE ExpeditedEvent)
	// Uses deviceRepo from below (created early for transfer bridge); create a separate instance
	// here since deviceRepo is declared later in this function.
	expeditedDeviceRepo := device.NewPgDeviceRepository(w.PgPool)
	expeditedReceiver := alarm.NewExpeditedEventReceiver(alarmEngine, expeditedDeviceRepo, w.EventBus, logger)
	if err := expeditedReceiver.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe expedited alarm receiver", zap.Error(err))
	}
	logger.Info("expedited alarm receiver started")

	// Frequent abnormal reboot monitor (F04)：滑动窗口内异常重启 >=阈值抬升告警。
	rebootMonitor := alarm.NewRebootMonitor(alarmEngine, w.Redis, logger)
	if err := rebootMonitor.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe reboot monitor", zap.Error(err))
	}
	logger.Info("reboot monitor started")

	// Reboot Task Closer (F01/F06)：M Reboot Inform 兜底收敛未 ACK 的 Reboot/FactoryReset 任务。
	rebootCloser := task.NewRebootCloser(w.TaskRepo, w.TaskService, logger)
	if err := rebootCloser.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe reboot task closer", zap.Error(err))
	}
	logger.Info("reboot task closer started")

	// MR Collector
	mrStore := mr.NewPgMRStore(w.PgPool, w.TsPool)
	mrCollector := mrcollector.NewMRCollector(w.MinIO, cfg.MinIO.Buckets.MRFiles, mrStore, w.EventBus, logger)
	if err := mrCollector.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe MR collector", zap.Error(err))
	}
	logger.Info("MR collector started")

	// Transfer Bridge
	deviceRepo := device.NewPgDeviceRepository(w.PgPool)
	transferDeduper := event.NewDeduper(w.Redis, 24*time.Hour, logger)
	transferBridge := transfer.NewTransferBridge(
		deviceRepo, w.MinIO,
		cfg.MinIO.Buckets,
		w.EventBus, transferDeduper, logger,
	)
	if err := transferBridge.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe transfer bridge", zap.Error(err))
	}
	logger.Info("transfer bridge started")

	// Backup Executor
	backupTaskRepo := backup.NewPgTaskRepository(w.PgPool)
	connReqClient := connreq.NewClient(w.Redis, logger)
	backupExecutor := backup.NewBackupExecutor(
		backupTaskRepo, deviceRepo, w.TaskService, connReqClient,
		w.EventBus, logger,
	)
	// T-0073 Phase 1: opt-in backup-failure alarm publish via PolicyService.
	// Worker shares the same backup_policies table as app; reads policy on each
	// failure to honour latest AlertOnFailure flag.
	backupPolicyRepo := backup.NewPgPolicyRepository(w.PgPool)
	backupPolicyService := backup.NewPolicyService(backupPolicyRepo, logger)
	backupPolicyMetrics := backup.NewPolicyMetrics(w.MetricsReg)
	backupExecutor.SetPolicyEnforcement(backupPolicyService, backupPolicyMetrics)
	if err := backupExecutor.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe backup executor", zap.Error(err))
	}
	logger.Info("backup executor started (with T-0073 failure-alarm enforcement)")

	// Report Generator
	reportDefRepo := report.NewPgDefinitionRepository(w.PgPool)
	reportRecordRepo := report.NewPgRecordRepository(w.PgPool)
	reportBucket := cfg.MinIO.Buckets.Reports
	if reportBucket == "" {
		reportBucket = "reports"
	}
	reportGenerator := report.NewReportGenerator(
		reportRecordRepo, reportDefRepo, kpiRepo, alarmPgStore,
		w.MinIO, reportBucket,
		w.EventBus, logger,
	)
	if err := reportGenerator.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe report generator", zap.Error(err))
	}
	logger.Info("report generator started")
}

// parseStringSlice parses a comma-separated string into a slice.
func parseStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

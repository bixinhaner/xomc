package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/config"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/paramsync"
	"github.com/omcgo/omcgo/internal/provision"
	"github.com/omcgo/omcgo/internal/task"
)

// paramSyncCompletionHandled is registered with CompletionRouter because the
// dedicated TaskTerminalBridge is the business consumer for PARAM_SYNC. The
// no-op registration documents that ownership and avoids false "no handler"
// warnings without double-processing terminal results.
type paramSyncCompletionHandled struct{}

func (paramSyncCompletionHandled) OnTaskCompleted(context.Context, *task.Task) {}

type paramSyncMaintainer interface {
	SweepExpiredRequests(context.Context, int) (int64, error)
	ReconcileStalledRequests(context.Context, time.Time, int) (int64, error)
	ReconcileStalledRuns(context.Context, time.Time, int) (int64, error)
	ReconcileCancellingRuns(context.Context, int) (int64, error)
	ReconcileRunCounts(context.Context) (int64, error)
	ReconcileTerminalBindings(context.Context) (int64, error)
	RecoverMissingResults(context.Context, int, int, int) (int, error)
	CleanStaging(context.Context, time.Time, int) (int64, error)
	CollectMetrics(context.Context) error
}

type paramSyncProjector interface {
	ReconcilePending(context.Context, int) (int, error)
}

type pullTuningSetter interface {
	SetPullTuning(subject string, tuning event.PullTuning)
}

// runParamSyncMaintenance deliberately executes every independent repair. A
// corrupt historical row must not suppress result republishing, staging cleanup,
// or durable metrics for the whole service.
type paramSyncMaintenanceConfig struct {
	recoveryRunLimit   int
	recoveryTaskLimit  int
	recoveryTaskBudget int
}

func defaultParamSyncMaintenanceConfig() paramSyncMaintenanceConfig {
	return paramSyncMaintenanceConfig{recoveryRunLimit: 20, recoveryTaskLimit: 200, recoveryTaskBudget: 200}
}

func paramSyncMaintenanceConfigFromApp(cfg appconfig.ParamSyncConfig) paramSyncMaintenanceConfig {
	out := defaultParamSyncMaintenanceConfig()
	if cfg.RecoveryRunLimit > 0 {
		out.recoveryRunLimit = cfg.RecoveryRunLimit
	}
	if cfg.RecoveryTaskLimitPerRun > 0 {
		out.recoveryTaskLimit = cfg.RecoveryTaskLimitPerRun
	}
	if cfg.RecoveryTaskBudget > 0 {
		out.recoveryTaskBudget = cfg.RecoveryTaskBudget
	}
	return out
}

func paramSyncPullTuningFromApp(cfg appconfig.ParamSyncConfig) event.PullTuning {
	return event.PullTuning{
		BatchSize:     cfg.ResultConsumerPullBatchSize,
		Concurrency:   cfg.ResultConsumerPullConcurrency,
		AckWait:       cfg.ResultConsumerAckWait,
		MaxAckPending: cfg.ResultConsumerMaxAckPending,
	}
}

func runParamSyncMaintenance(ctx context.Context, maintainer paramSyncMaintainer, now time.Time, cfg paramSyncMaintenanceConfig) error {
	var errs []error
	if _, err := maintainer.SweepExpiredRequests(ctx, 100); err != nil {
		errs = append(errs, fmt.Errorf("sweep expired requests: %w", err))
	}
	if _, err := maintainer.ReconcileStalledRequests(ctx, now.Add(-5*time.Minute), 100); err != nil {
		errs = append(errs, fmt.Errorf("reconcile stalled requests: %w", err))
	}
	if _, err := maintainer.ReconcileStalledRuns(ctx, now.Add(-5*time.Minute), 100); err != nil {
		errs = append(errs, fmt.Errorf("reconcile stalled runs: %w", err))
	}
	if _, err := maintainer.ReconcileCancellingRuns(ctx, 100); err != nil {
		errs = append(errs, fmt.Errorf("reconcile cancelling runs: %w", err))
	}
	if _, err := maintainer.ReconcileRunCounts(ctx); err != nil {
		errs = append(errs, fmt.Errorf("reconcile run counts: %w", err))
	}
	if _, err := maintainer.RecoverMissingResults(ctx, cfg.recoveryRunLimit, cfg.recoveryTaskLimit, cfg.recoveryTaskBudget); err != nil {
		errs = append(errs, fmt.Errorf("recover missing results: %w", err))
	}
	if _, err := maintainer.ReconcileTerminalBindings(ctx); err != nil {
		errs = append(errs, fmt.Errorf("reconcile terminal bindings: %w", err))
	}
	if _, err := maintainer.CleanStaging(ctx, now.Add(-24*time.Hour), 10000); err != nil {
		errs = append(errs, fmt.Errorf("clean staging: %w", err))
	}
	if err := maintainer.CollectMetrics(ctx); err != nil {
		errs = append(errs, fmt.Errorf("collect metrics: %w", err))
	}
	return errors.Join(errs...)
}

func runParamSyncReconciliation(ctx context.Context, maintainer paramSyncMaintainer, projector paramSyncProjector, now time.Time, cfg paramSyncMaintenanceConfig) error {
	var errs []error
	maintenanceCtx, cancelMaintenance := context.WithTimeout(ctx, 20*time.Second)
	if err := runParamSyncMaintenance(maintenanceCtx, maintainer, now, cfg); err != nil {
		errs = append(errs, err)
	}
	cancelMaintenance()
	projectionCtx, cancelProjection := context.WithTimeout(ctx, 20*time.Second)
	defer cancelProjection()
	if _, err := projector.ReconcilePending(projectionCtx, 100); err != nil {
		errs = append(errs, fmt.Errorf("reconcile pending parameter sync projections: %w", err))
	}
	return errors.Join(errs...)
}

type paramSyncMappingProvider struct{ c *Container }

func (p paramSyncMappingProvider) GetMappingSet(ctx context.Context, partial *model.Device) (*parammodel.MappingSet, error) {
	dev, err := p.c.DeviceRepo.GetByID(ctx, partial.ID)
	if err != nil || dev == nil {
		return nil, fmt.Errorf("load device: %w", err)
	}
	matched, err := p.c.ProductRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || matched == nil || matched.Product == nil {
		return nil, fmt.Errorf("match product: %w", err)
	}
	set, err := p.c.ParamRegistry.GetByProduct(ctx, matched.Product.ID, dev.FirmwareVersion)
	if err != nil || set == nil {
		return set, err
	}
	// Registry results may be cached. Copy before attaching the exact device
	// capability key used by the durable planner.
	resolved := *set
	resolved.ProductID = matched.Product.ID
	resolved.SoftwareVersion = dev.FirmwareVersion
	return &resolved, nil
}

type paramSyncFullRunProjection struct {
	devices  device.DeviceRepository
	info     *device.InfoSyncer
	nameSync *provision.DeviceNameSyncHook
}

func (p *paramSyncFullRunProjection) Refresh(ctx context.Context, deviceID uuid.UUID) error {
	dev, err := p.devices.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("load device for parameter sync projection: %w", err)
	}
	if dev == nil {
		return fmt.Errorf("device %s not found for parameter sync projection", deviceID)
	}
	if _, err := p.info.SyncFromParameters(ctx, deviceID, dev.Carrier, dev.Technology); err != nil {
		return fmt.Errorf("refresh device_info from parameters: %w", err)
	}
	if p.nameSync != nil {
		if err := p.nameSync.Execute(ctx, dev); err != nil {
			return fmt.Errorf("synchronize device name: %w", err)
		}
	}
	return nil
}

type paramSyncStarter struct {
	service *paramsync.Service
	flags   paramsync.FeatureFlags
	legacy  device.ParamSyncStarter
	binding *paramsync.BindingCoordinator
	devices device.DeviceRepository
}

type licenseParamSyncSubmitter interface {
	Submit(context.Context, paramsync.SubmitCommand) (*paramsync.SubmitResult, error)
}

func submitLicenseParamSync(ctx context.Context, submitter licenseParamSyncSubmitter, dev *model.Device, sourceID string, paths []string) (int, error) {
	result, err := submitter.Submit(ctx, paramsync.SubmitCommand{
		DeviceID: dev.ID, DeviceSN: dev.SerialNumber, CallerType: "license",
		TriggerReason: paramsync.TriggerLicense, Scope: paramsync.SyncScopePartial,
		RequestedPaths: paths, IdempotencyKey: sourceID,
	})
	if err != nil {
		return 0, err
	}
	if result.ResultCode == paramsync.ResultCodePathBUnavailable {
		return 0, fmt.Errorf("durable license parameter sync unavailable: %s", result.ResultCode)
	}
	if result.Status == paramsync.RequestStatusRejected {
		return 0, fmt.Errorf("durable license parameter sync rejected: %s", result.ResultCode)
	}
	return result.TaskCount, nil
}

func (s *paramSyncStarter) StartLicenseSync(ctx context.Context, dev *model.Device, paths []string, sourceID string) (int, error) {
	if !s.flags.EnabledForDevice(dev.ID.String()) {
		return 0, fmt.Errorf("durable license parameter sync is disabled for this device")
	}
	return submitLicenseParamSync(ctx, s.service, dev, sourceID, paths)
}

func (s *paramSyncStarter) SubmitConfigPull(ctx context.Context, deviceSN string, paths []string, key string) (config.DurablePullResult, error) {
	dev, err := s.devices.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return config.DurablePullResult{}, fmt.Errorf("load config-pull device: %w", err)
	}
	if dev == nil {
		return config.DurablePullResult{}, fmt.Errorf("config-pull device %s not found", deviceSN)
	}
	if !s.flags.EnabledForDevice(dev.ID.String()) {
		return config.DurablePullResult{Handled: false}, nil
	}
	if key == "" {
		key = uuid.NewString()
	}
	result, err := s.service.Submit(ctx, paramsync.SubmitCommand{
		DeviceID: dev.ID, DeviceSN: dev.SerialNumber, CallerType: "config",
		TriggerReason: paramsync.TriggerConfigPull, Scope: paramsync.SyncScopePartial,
		RequestedPaths: paths, IdempotencyKey: key,
	})
	if err != nil {
		return config.DurablePullResult{}, err
	}
	if result.ResultCode == paramsync.ResultCodePathBUnavailable && s.flags.LegacyFallbackEnabled {
		return config.DurablePullResult{Handled: false}, nil
	}
	runID := ""
	if result.RunID != nil {
		runID = result.RunID.String()
	} else if result.ActiveRunID != nil {
		runID = result.ActiveRunID.String()
	}
	request, _ := s.service.GetRequest(ctx, result.RequestID)
	errorMessage := ""
	if request != nil {
		errorMessage = request.ErrorMessage
	}
	return config.DurablePullResult{
		RequestID: result.RequestID.String(), RunID: runID, TaskCount: result.TaskCount,
		Handled: true, Status: string(result.Status), ResultCode: string(result.ResultCode), ErrorMessage: errorMessage,
	}, nil
}

func (s *paramSyncStarter) StartDurableSync(ctx context.Context, dev *model.Device, sourceID, reason string, paths []string) (bool, int, error) {
	if !s.flags.EnabledForDevice(dev.ID.String()) {
		return false, 0, nil
	}
	trigger := paramsync.TriggerReason(reason)
	if trigger == "" {
		return true, 0, fmt.Errorf("durable parameter sync trigger reason is required")
	}
	scope := paramsync.SyncScopeFull
	if len(paths) > 0 {
		scope = paramsync.SyncScopePartial
	}
	result, err := s.service.Submit(ctx, paramsync.SubmitCommand{
		DeviceID: dev.ID, DeviceSN: dev.SerialNumber, CallerType: "provision", TriggerReason: trigger,
		Scope: scope, RequestedPaths: paths, IdempotencyKey: sourceID,
	})
	if err != nil {
		return true, 0, err
	}
	if result.ResultCode == paramsync.ResultCodePathBUnavailable {
		if s.flags.LegacyFallbackEnabled {
			return false, 0, nil
		}
		return true, 0, fmt.Errorf("durable parameter sync unavailable: %s", result.ResultCode)
	}
	if result.Status == paramsync.RequestStatusRejected && result.ResultCode == paramsync.ResultCodeActiveSyncExists {
		return true, 0, fmt.Errorf("durable parameter sync is busy for this device")
	}
	if provisioningTaskID, parseErr := uuid.Parse(sourceID); parseErr == nil && s.binding != nil {
		runID := result.RunID
		if runID == nil {
			runID = result.ActiveRunID
		}
		if runID != nil {
			if err := s.binding.Bind(ctx, result.RequestID, *runID, provisioningTaskID); err != nil {
				return true, 0, err
			}
		} else if result.ResultCode == paramsync.ResultCodeNoStorablePath || result.ResultCode == paramsync.ResultCodeAutomaticBackoff {
			if err := s.binding.CompleteWithoutRun(ctx, provisioningTaskID); err != nil {
				return true, 0, err
			}
		}
	}
	return true, result.TaskCount, nil
}

func (s *paramSyncStarter) SetLegacy(legacy device.ParamSyncStarter) { s.legacy = legacy }

func (s *paramSyncStarter) StartManualSync(ctx context.Context, dev *model.Device, sourceID string, paths []string) (bool, int, error) {
	result, err := s.StartManualSyncDetailed(ctx, dev, sourceID, paths)
	if result == nil {
		return false, 0, err
	}
	return result.Used, result.TaskCount, err
}

func (s *paramSyncStarter) StartManualSyncDetailed(ctx context.Context, dev *model.Device, sourceID string, paths []string) (*device.ManualParamSyncStart, error) {
	if !s.flags.EnabledForDevice(dev.ID.String()) {
		if s.legacy == nil {
			return &device.ManualParamSyncStart{}, nil
		}
		used, count, err := s.legacy.StartManualSync(ctx, dev, sourceID, paths)
		return &device.ManualParamSyncStart{Used: used, TaskCount: count, Status: "queued"}, err
	}
	scope := paramsync.SyncScopeFull
	if len(paths) > 0 {
		scope = paramsync.SyncScopePartial
	}
	result, err := s.service.Submit(ctx, paramsync.SubmitCommand{
		DeviceID: dev.ID, DeviceSN: dev.SerialNumber, CallerType: "manual", TriggerReason: paramsync.TriggerManual,
		Scope: scope, RequestedPaths: paths, IdempotencyKey: sourceID,
	})
	if err != nil {
		return &device.ManualParamSyncStart{Used: true}, err
	}
	if result.ResultCode == paramsync.ResultCodePathBUnavailable {
		return &device.ManualParamSyncStart{RequestID: result.RequestID, Status: string(result.Status), ResultCode: string(result.ResultCode)}, nil
	}
	if result.Status == paramsync.RequestStatusRejected && result.ResultCode == paramsync.ResultCodeActiveSyncExists {
		return &device.ManualParamSyncStart{Used: true, RequestID: result.RequestID, RunID: result.RunID, Status: string(result.Status), ResultCode: string(result.ResultCode)}, commonerrors.NewBusinessError(
			global.ErrCodeRuleTaskRunning,
			"parameter sync already running for this device, try again in a few seconds",
			commonerrors.ErrAlreadyExists,
		)
	}
	return &device.ManualParamSyncStart{Used: true, TaskCount: result.TaskCount, RequestID: result.RequestID, RunID: result.RunID, Status: string(result.Status), ResultCode: string(result.ResultCode)}, nil
}

func initParamSyncModule(c *Container) error {
	logger := c.Logger.Named("param-sync")
	flags := paramsync.FeatureFlags{
		RunEnabled: c.Cfg.ParamSync.RunEnabled, ResultConsumerEnabled: c.Cfg.ParamSync.ResultConsumerEnabled,
		StagingEnabled: c.Cfg.ParamSync.StagingEnabled, CanaryPercent: c.Cfg.ParamSync.CanaryPercent,
		LegacyFallbackEnabled: c.Cfg.ParamSync.LegacyFallbackEnabled,
	}
	if err := flags.Validate(); err != nil {
		return err
	}
	if !flags.RunEnabled {
		logger.Info("reliable parameter sync module disabled")
		return nil
	}

	repo := paramsync.NewPGRepository(c.PgPool)
	unsupportedPaths := paramsync.NewPGReadUnsupportedPathRepository(c.PgPool)
	planner := paramsync.NewPlanner(paramSyncMappingProvider{c: c}, c.Cfg.Provision.AutoSync.GPVBatchSize).
		WithUnsupportedPaths(unsupportedPaths)
	metrics := paramsync.NewMetrics(c.MetricsReg)
	maintenanceCfg := paramSyncMaintenanceConfigFromApp(c.Cfg.ParamSync)
	if setter, ok := c.EventBus.(pullTuningSetter); ok {
		setter.SetPullTuning(event.SubjectParamSyncTaskResult, paramSyncPullTuningFromApp(c.Cfg.ParamSync))
	}
	service := paramsync.NewService(repo, planner).WithMetrics(metrics).WithDispatcher(paramsync.NewPGTaskDispatcher(c.PgPool))
	outbox := paramsync.NewOutboxDispatcher(c.PgPool, c.TaskSvc, 20).WithEventBus(c.EventBus)
	resultProcessor := paramsync.NewPGResultProcessor(c.PgPool).WithMetrics(metrics)
	reconciler := paramsync.NewReconciler(c.PgPool, c.EventBus, metrics).WithResultProcessor(resultProcessor)
	bridge := paramsync.NewTaskTerminalBridge(c.EventBus)
	binding := paramsync.NewBindingCoordinator(c.PgPool, c.EventBus)
	infoSyncer := device.NewInfoSyncer(c.DeviceInfoRepo, c.ParamRepo, device.NewPgDeviceRepository(c.PgPool), c.Carriers, logger)
	nameSyncCfgRepo := admin.NewPgSysConfigRepository(c.PgPool)
	nameSyncHook := provision.NewDeviceNameSyncHook(
		func(ctx context.Context, category, key string) (string, bool) {
			cfg, err := nameSyncCfgRepo.GetByKey(ctx, category, key)
			if err != nil || cfg == nil {
				return "", false
			}
			return cfg.Value, true
		},
		c.ParamRepo, c.DeviceInfoRepo, c.DeviceInfoRepo, logger,
	).SetSiteUpdater(device.NewPgDeviceRepository(c.PgPool)).
		SetSPVSender(provision.NewDeviceNameTaskSender(c.TaskSvc))
	projector := paramsync.NewCompletionProjector(c.PgPool, c.EventBus, &paramSyncFullRunProjection{
		devices: c.DeviceRepo, info: infoSyncer, nameSync: nameSyncHook,
	})
	requestConsumer := paramsync.NewRequestConsumer(c.EventBus, service, c.DeviceRepo, flags)
	var resultConsumer *paramsync.ResultConsumer
	initialized := false
	defer func() {
		if initialized {
			return
		}
		_ = requestConsumer.Stop()
		_ = projector.Stop()
		_ = binding.Stop()
		_ = bridge.Stop()
		if resultConsumer != nil {
			_ = resultConsumer.Stop()
		}
	}()
	starter := &paramSyncStarter{service: service, flags: flags, binding: binding, devices: c.DeviceRepo}
	c.miscDeps.paramSyncHandler = paramsync.NewHandler(service).
		WithOperations(paramsync.NewOperations(c.PgPool, service)).
		WithAuthorization(c.PermService, c.DeviceService)
	c.miscDeps.paramSyncStarter = starter
	if err := bridge.Start(); err != nil {
		return err
	}
	if err := binding.Start(); err != nil {
		return err
	}
	if err := projector.Start(); err != nil {
		return err
	}
	if err := requestConsumer.Start(); err != nil {
		return err
	}
	if c.GS != nil {
		c.GS.Register("param-sync-terminal-bridge", 2, func(context.Context) error { return bridge.Stop() })
		c.GS.Register("param-sync-binding", 2, func(context.Context) error { return binding.Stop() })
		c.GS.Register("param-sync-completion-projector", 2, func(context.Context) error { return projector.Stop() })
		c.GS.Register("param-sync-request-consumer", 2, func(context.Context) error { return requestConsumer.Stop() })
	}
	if flags.ResultConsumerEnabled {
		resultConsumer = paramsync.NewResultConsumer(c.EventBus, resultProcessor).
			WithMetrics(metrics).
			WithWorkerConfig(c.Cfg.ParamSync.ResultConsumerShardCount, c.Cfg.ParamSync.ResultConsumerQueueDepth)
		if err := resultConsumer.Start(); err != nil {
			return err
		}
		c.miscDeps.paramSyncConsumer = resultConsumer
		if c.GS != nil {
			c.GS.Register("param-sync-result-consumer", 2, func(context.Context) error { return resultConsumer.Stop() })
		}
	}

	maintenanceCtx, maintenanceCancel := context.WithCancel(context.Background())
	if c.GS != nil {
		c.GS.Register("param-sync-maintenance", 1, func(context.Context) error { maintenanceCancel(); return nil })
	} else {
		// A maintenance goroutine without an owner cannot be stopped in tests or
		// reduced startup modes. Keep the module usable, but do not leak it.
		maintenanceCancel()
		logger.Warn("parameter sync maintenance disabled: graceful shutdown manager is unavailable")
	}
	if c.GS != nil {
		go func() {
			ticker := time.NewTicker(time.Second)
			reconcileTicker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			defer reconcileTicker.Stop()
			for {
				select {
				case <-maintenanceCtx.Done():
					return
				case <-ticker.C:
					ctx, cancel := context.WithTimeout(maintenanceCtx, 10*time.Second)
					_, err := outbox.RequeueStaleDeliveries(ctx, time.Now().Add(-time.Minute))
					if err == nil {
						_, err = outbox.DispatchPending(ctx, 100)
					}
					if err == nil {
						_, err = service.DispatchQueued(ctx, 100)
					}
					cancel()
					if err != nil && !errors.Is(err, context.Canceled) {
						logger.Warn("parameter sync outbox dispatch failed", zap.Error(err))
					}
				case <-reconcileTicker.C:
					err := runParamSyncReconciliation(maintenanceCtx, reconciler, projector, time.Now(), maintenanceCfg)
					if err != nil && !errors.Is(err, context.Canceled) {
						logger.Warn("parameter sync reconciliation failed", zap.Error(err))
					}
				}
			}
		}()
	}
	logger.Info("reliable parameter sync module initialized",
		zap.Bool("run_enabled", flags.RunEnabled), zap.Int("canary_percent", flags.CanaryPercent),
		zap.Bool("result_consumer_enabled", flags.ResultConsumerEnabled), zap.Bool("staging_enabled", flags.StagingEnabled))
	initialized = true
	return nil
}

var _ device.ParamSyncStarter = (*paramSyncStarter)(nil)
var _ device.LicenseSyncStarter = (*paramSyncStarter)(nil)

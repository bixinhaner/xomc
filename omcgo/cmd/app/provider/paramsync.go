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

const (
	paramSyncOutboxWorkers = 8
	paramSyncQueuedWorkers = 8
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

type registeredDeviceSyncCandidateRepository interface {
	ListRegisteredSyncCandidates(context.Context, int) ([]*model.Device, error)
}

type registeredDeviceSyncReconciler struct {
	repo      registeredDeviceSyncCandidateRepository
	submitter licenseParamSyncSubmitter
}

func (r registeredDeviceSyncReconciler) Reconcile(ctx context.Context, limit int) (int, error) {
	devices, err := r.repo.ListRegisteredSyncCandidates(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("list registered-device parameter sync candidates: %w", err)
	}
	var errs []error
	for _, dev := range devices {
		sourceEventID := "device_registered:" + dev.ID.String()
		attemptKey := sourceEventID + ":" + uuid.NewString()
		if err := submitRegisteredDeviceSync(ctx, r.submitter, dev, sourceEventID, attemptKey); err != nil {
			errs = append(errs, fmt.Errorf("resubmit registered-device parameter sync for %s: %w", dev.ID, err))
		}
	}
	return len(devices), errors.Join(errs...)
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
	stepTimeout        time.Duration
	recoveryTimeout    time.Duration
}

func defaultParamSyncMaintenanceConfig() paramSyncMaintenanceConfig {
	return paramSyncMaintenanceConfig{
		recoveryRunLimit: 200, recoveryTaskLimit: 200, recoveryTaskBudget: 200,
		stepTimeout: 10 * time.Second, recoveryTimeout: 20 * time.Second,
	}
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
	step := func(timeout time.Duration, run func(context.Context) error) error {
		stepCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return run(stepCtx)
	}
	if err := step(cfg.stepTimeout, func(stepCtx context.Context) error {
		_, err := maintainer.ReconcileRunCounts(stepCtx)
		return err
	}); err != nil {
		errs = append(errs, fmt.Errorf("reconcile run counts: %w", err))
	}
	if err := step(cfg.stepTimeout, func(stepCtx context.Context) error {
		_, err := maintainer.SweepExpiredRequests(stepCtx, 100)
		return err
	}); err != nil {
		errs = append(errs, fmt.Errorf("sweep expired requests: %w", err))
	}
	if err := step(cfg.stepTimeout, func(stepCtx context.Context) error {
		_, err := maintainer.ReconcileStalledRequests(stepCtx, now.Add(-5*time.Minute), 100)
		return err
	}); err != nil {
		errs = append(errs, fmt.Errorf("reconcile stalled requests: %w", err))
	}
	if err := step(cfg.stepTimeout, func(stepCtx context.Context) error {
		_, err := maintainer.ReconcileStalledRuns(stepCtx, now.Add(-5*time.Minute), 100)
		return err
	}); err != nil {
		errs = append(errs, fmt.Errorf("reconcile stalled runs: %w", err))
	}
	if err := step(cfg.stepTimeout, func(stepCtx context.Context) error {
		_, err := maintainer.ReconcileCancellingRuns(stepCtx, 100)
		return err
	}); err != nil {
		errs = append(errs, fmt.Errorf("reconcile cancelling runs: %w", err))
	}
	if err := step(cfg.recoveryTimeout, func(stepCtx context.Context) error {
		_, err := maintainer.RecoverMissingResults(stepCtx, cfg.recoveryRunLimit, cfg.recoveryTaskLimit, cfg.recoveryTaskBudget)
		return err
	}); err != nil {
		errs = append(errs, fmt.Errorf("recover missing results: %w", err))
	}
	if err := step(cfg.stepTimeout, func(stepCtx context.Context) error {
		_, err := maintainer.ReconcileTerminalBindings(stepCtx)
		return err
	}); err != nil {
		errs = append(errs, fmt.Errorf("reconcile terminal bindings: %w", err))
	}
	if err := step(cfg.stepTimeout, func(stepCtx context.Context) error {
		_, err := maintainer.CleanStaging(stepCtx, now.Add(-24*time.Hour), 10000)
		return err
	}); err != nil {
		errs = append(errs, fmt.Errorf("clean staging: %w", err))
	}
	if err := step(cfg.stepTimeout, maintainer.CollectMetrics); err != nil {
		errs = append(errs, fmt.Errorf("collect metrics: %w", err))
	}
	return errors.Join(errs...)
}

func runParamSyncReconciliation(ctx context.Context, maintainer paramSyncMaintainer, projector paramSyncProjector, now time.Time, cfg paramSyncMaintenanceConfig) error {
	var errs []error
	if err := runParamSyncMaintenance(ctx, maintainer, now, cfg); err != nil {
		errs = append(errs, err)
	}
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
	if _, err := p.info.SyncFromParameters(ctx, deviceID, dev.Carrier, dev.Technology, dev.ProductClass); err != nil {
		return fmt.Errorf("refresh device_info from parameters: %w", err)
	}
	if p.nameSync != nil {
		if err := p.nameSync.Execute(ctx, dev); err != nil {
			return fmt.Errorf("synchronize device name: %w", err)
		}
	}
	return nil
}

type licenseParamSyncSubmitter interface {
	Submit(context.Context, paramsync.SubmitCommand) (*paramsync.SubmitResult, error)
}

type paramSyncService interface {
	licenseParamSyncSubmitter
	GetRequest(context.Context, uuid.UUID) (*paramsync.SyncRequest, error)
}

type paramSyncStarter struct {
	service paramSyncService
	flags   paramsync.FeatureFlags
	legacy  device.ParamSyncStarter
	binding *paramsync.BindingCoordinator
	devices device.DeviceRepository
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

func submitRegisteredDeviceSync(
	ctx context.Context,
	submitter licenseParamSyncSubmitter,
	dev *model.Device,
	sourceID string,
	idempotencyKeys ...string,
) error {
	idempotencyKey := sourceID
	if len(idempotencyKeys) > 0 {
		idempotencyKey = idempotencyKeys[0]
	}
	result, err := submitter.Submit(ctx, paramsync.SubmitCommand{
		DeviceID:        dev.ID,
		DeviceSN:        dev.SerialNumber,
		CallerType:      "provision",
		TriggerReason:   paramsync.TriggerDeviceRegistered,
		Scope:           paramsync.SyncScopeFull,
		IdempotencyKey:  idempotencyKey,
		SourceEventID:   sourceID,
		OriginEventType: "device_registered",
	})
	if err != nil {
		return err
	}
	switch result.ResultCode {
	case paramsync.ResultCodePathBUnavailable,
		paramsync.ResultCodeAutomaticBackoff,
		paramsync.ResultCodeActiveSyncExists:
		return nil
	}
	if result.Status == paramsync.RequestStatusRejected {
		return fmt.Errorf(
			"durable registered-device parameter sync rejected: %s",
			result.ResultCode,
		)
	}
	return nil
}

func submitReleaseSync(
	ctx context.Context,
	submitter licenseParamSyncSubmitter,
	dev *model.Device,
	campaignID uuid.UUID,
	attemptID uuid.UUID,
) (bool, error) {
	attemptKey := fmt.Sprintf(
		"omc_upgrade:%s:%s:%s",
		campaignID,
		dev.ID,
		attemptID,
	)
	sourceEventID := fmt.Sprintf(
		"omc_upgrade:%s:%s",
		campaignID,
		dev.ID,
	)
	result, err := submitter.Submit(ctx, paramsync.SubmitCommand{
		DeviceID:        dev.ID,
		DeviceSN:        dev.SerialNumber,
		CallerType:      "provision",
		TriggerReason:   paramsync.TriggerOMCUpgrade,
		Scope:           paramsync.SyncScopeFull,
		IdempotencyKey:  attemptKey,
		CampaignID:      &campaignID,
		SourceEventID:   sourceEventID,
		OriginEventType: "omc_upgrade",
	})
	if err != nil {
		return false, err
	}
	switch result.ResultCode {
	case paramsync.ResultCodeAutomaticBackoff,
		paramsync.ResultCodeActiveSyncExists:
		return false, nil
	case paramsync.ResultCodePathBUnavailable:
		return false, fmt.Errorf(
			"durable OMC release parameter sync unavailable: %s",
			result.ResultCode,
		)
	}
	if result.Status == paramsync.RequestStatusRejected {
		return false, fmt.Errorf(
			"durable OMC release parameter sync rejected: %s",
			result.ResultCode,
		)
	}
	return true, nil
}

func (s *paramSyncStarter) StartLicenseSync(ctx context.Context, dev *model.Device, paths []string, sourceID string) (int, error) {
	if !s.flags.EnabledForDevice(dev.ID.String()) {
		return 0, fmt.Errorf("durable license parameter sync is disabled for this device")
	}
	return submitLicenseParamSync(ctx, s.service, dev, sourceID, paths)
}

func (s *paramSyncStarter) StartRegisteredDeviceSync(
	ctx context.Context,
	dev *model.Device,
	sourceID string,
) error {
	if !s.flags.RunEnabled {
		return nil
	}
	return submitRegisteredDeviceSync(ctx, s.service, dev, sourceID)
}

func (s *paramSyncStarter) SubmitDeviceOnlineFullSync(
	ctx context.Context,
	dev *model.Device,
	idempotencyKey string,
	sourceEventID string,
	originEventType string,
) (*provision.DeviceOnlineFullSyncResult, error) {
	if dev == nil {
		return nil, fmt.Errorf("durable device-online parameter sync requires a device")
	}
	if !s.flags.EnabledForDevice(dev.ID.String()) {
		return nil, fmt.Errorf("durable device-online parameter sync is disabled for device %s", dev.ID)
	}
	result, err := s.service.Submit(ctx, paramsync.SubmitCommand{
		DeviceID:        dev.ID,
		DeviceSN:        dev.SerialNumber,
		CallerType:      "provision",
		TriggerReason:   paramsync.TriggerDeviceOnline,
		Scope:           paramsync.SyncScopeFull,
		IdempotencyKey:  idempotencyKey,
		SourceEventID:   sourceEventID,
		OriginEventType: originEventType,
	})
	if err != nil {
		return nil, fmt.Errorf("submit durable device-online parameter sync: %w", err)
	}
	runID := result.RunID
	if runID == nil {
		runID = result.ActiveRunID
	}
	return &provision.DeviceOnlineFullSyncResult{
		RequestID:  result.RequestID,
		RunID:      runID,
		Status:     string(result.Status),
		ResultCode: string(result.ResultCode),
		TaskCount:  result.TaskCount,
	}, nil
}

func (s *paramSyncStarter) StartReleaseSync(
	ctx context.Context,
	dev *model.Device,
	campaignID uuid.UUID,
	attemptID uuid.UUID,
) (bool, error) {
	if !s.flags.RunEnabled {
		return false, nil
	}
	return submitReleaseSync(
		ctx,
		s.service,
		dev,
		campaignID,
		attemptID,
	)
}

func (s *paramSyncStarter) SubmitModelUploadParamSync(ctx context.Context, dev *model.Device, sourceID string, modelUploadID uuid.UUID, status string) (bool, int, error) {
	if !s.flags.EnabledForDevice(dev.ID.String()) {
		return false, 0, fmt.Errorf("durable model-upload parameter sync is disabled for this device")
	}
	result, err := s.service.Submit(ctx, paramsync.SubmitCommand{
		DeviceID: dev.ID, DeviceSN: dev.SerialNumber, CallerType: "provision",
		TriggerReason: paramsync.TriggerModelUpload, Scope: paramsync.SyncScopeFull,
		IdempotencyKey: sourceID, SourceEventID: sourceID, OriginEventType: "model_upload",
		ModelUploadIntentID: &modelUploadID, ModelUploadStatus: status,
	})
	if err != nil {
		return true, 0, err
	}
	if result.ResultCode == paramsync.ResultCodePathBUnavailable {
		return true, 0, fmt.Errorf("durable model-upload parameter sync unavailable: %s", result.ResultCode)
	}
	if result.Status == paramsync.RequestStatusRejected && result.ResultCode == paramsync.ResultCodeActiveSyncExists {
		return true, 0, fmt.Errorf("durable model-upload parameter sync is busy for this device")
	}
	return true, result.TaskCount, nil
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
	trigger := paramsync.TriggerReason(reason)
	if trigger == "" {
		return true, 0, fmt.Errorf("durable parameter sync trigger reason is required")
	}
	if !s.flags.EnabledForDevice(dev.ID.String()) {
		// All parameter sync triggers enter the durable parameter_sync_* path
		// first. The legacy sync-gpv Path B path is retained only as a
		// temporary fallback until param_sync_running is stable enough to remove
		// the old pipeline.
		return false, 0, nil
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
			// Temporary fallback: all triggers prefer parameter_sync_*; legacy
			// sync-gpv Path B remains only while param_sync_running rollout is
			// being stabilized.
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
		// Manual sync follows the same transition rule as periodic,
		// device_online, firmware_changed, bootstrap, and model_upload:
		// parameter_sync_* first; legacy sync-gpv Path B only as a temporary
		// fallback until param_sync_running is stable and the old path can be
		// deleted.
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
		if s.flags.LegacyFallbackEnabled && s.legacy != nil {
			used, count, err := s.legacy.StartManualSync(ctx, dev, sourceID, paths)
			return &device.ManualParamSyncStart{Used: used, TaskCount: count, Status: "queued"}, err
		}
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
	var registeredSyncReconciler *registeredDeviceSyncReconciler
	if c.Cfg.ParamSync.RoutingMode == "durable" {
		registeredSyncReconciler = &registeredDeviceSyncReconciler{
			repo:      repo,
			submitter: service,
		}
	}
	bridge := paramsync.NewTaskTerminalBridge(c.EventBus)
	binding := paramsync.NewBindingCoordinator(c.PgPool, c.EventBus)
	infoSyncer := device.NewInfoSyncer(c.DeviceInfoRepo, c.ParamRepo, device.NewPgDeviceRepository(c.PgPool), c.Carriers, logger, device.NewPgLocationObservationRepository(c.PgPool))
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
	c.miscDeps.paramSyncService = service
	if c.miscDeps.nbRouter != nil {
		c.miscDeps.nbRouter.SetParamSyncService(service)
		logger.Info("northbound parameter sync status facade wired")
	}
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
			indexCtx, cancel := context.WithTimeout(maintenanceCtx, 30*time.Minute)
			defer cancel()
			if err := outbox.EnsurePerformanceIndexes(indexCtx); err != nil && !errors.Is(err, context.Canceled) {
				logger.Warn("ensure parameter sync outbox indexes failed", zap.Error(err))
			}
		}()
		go runPeriodicMaintenance(maintenanceCtx, time.Second, func(context.Context) {
			ctx, cancel := context.WithTimeout(maintenanceCtx, 10*time.Second)
			_, err := outbox.RequeueStaleDeliveries(ctx, time.Now().Add(-time.Minute))
			cancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Warn("parameter sync stale outbox requeue failed", zap.Error(err))
			}

			ctx, cancel = context.WithTimeout(maintenanceCtx, 15*time.Second)
			_, err = outbox.DispatchPendingConcurrent(ctx, paramSyncOutboxWorkers, 25)
			cancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Warn("parameter sync outbox delivery failed", zap.Error(err))
			}
		})
		go runPeriodicMaintenance(maintenanceCtx, time.Second, func(context.Context) {
			ctx, cancel := context.WithTimeout(maintenanceCtx, 10*time.Second)
			_, err := repo.ReconcileAutomaticAdmission(ctx, time.Now().UTC(), 500)
			cancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Warn("automatic parameter sync admission reconciliation failed", zap.Error(err))
				return
			}

			ctx, cancel = context.WithTimeout(maintenanceCtx, 30*time.Second)
			_, err = service.DispatchQueuedConcurrent(ctx, 100, paramSyncQueuedWorkers)
			cancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Warn("queued parameter sync request dispatch failed", zap.Error(err))
			}
		})
		go runPeriodicMaintenance(maintenanceCtx, 15*time.Second, func(context.Context) {
			ctx, cancel := context.WithTimeout(maintenanceCtx, 10*time.Second)
			_, err := repo.RepairAutomaticAdmissionCounters(ctx, time.Now().UTC())
			if err != nil && !errors.Is(err, context.Canceled) {
				cancel()
				logger.Warn("automatic parameter sync admission counter repair failed", zap.Error(err))
				return
			}
			stats, err := repo.GetAutomaticAdmissionStats(ctx, time.Now().UTC())
			cancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Warn("automatic parameter sync admission metrics failed", zap.Error(err))
				return
			}
			metrics.AutomaticReservedRuns.Set(float64(stats.ReservedRuns))
			metrics.AutomaticQueuedRequests.Set(float64(stats.QueuedRequests))
			metrics.AutomaticOldestQueueAge.Set(stats.OldestQueueAgeSecond)
		})
		go runPeriodicMaintenance(maintenanceCtx, 30*time.Second, func(context.Context) {
			err := runParamSyncReconciliation(maintenanceCtx, reconciler, projector, time.Now(), maintenanceCfg)
			if registeredSyncReconciler != nil {
				_, registeredErr := registeredSyncReconciler.Reconcile(maintenanceCtx, 100)
				err = errors.Join(err, registeredErr)
			}
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Warn("parameter sync reconciliation failed", zap.Error(err))
			}
		})
	}
	logger.Info("reliable parameter sync module initialized",
		zap.Bool("run_enabled", flags.RunEnabled), zap.Int("canary_percent", flags.CanaryPercent),
		zap.Bool("result_consumer_enabled", flags.ResultConsumerEnabled), zap.Bool("staging_enabled", flags.StagingEnabled))
	initialized = true
	return nil
}

func runPeriodicMaintenance(ctx context.Context, interval time.Duration, task func(context.Context)) {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			task(ctx)
			timer.Reset(interval)
		}
	}
}

var _ device.ParamSyncStarter = (*paramSyncStarter)(nil)
var _ device.LicenseSyncStarter = (*paramSyncStarter)(nil)

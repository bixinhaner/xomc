package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// SyncService handles batch parameter value synchronization from devices.
type SyncService struct {
	paramRepo     device.DeviceParameterRepository
	discoveryRepo ParameterDiscoveryLogRepository
	cmdQueue      cmdqueue.CommandQueue
	config        appconfig.AutoSyncConfig
	batchSize     int
	logger        *zap.Logger
}

// NewSyncService creates a new SyncService.
func NewSyncService(
	paramRepo device.DeviceParameterRepository,
	discoveryRepo ParameterDiscoveryLogRepository,
	cmdQueue cmdqueue.CommandQueue,
	config appconfig.AutoSyncConfig,
	batchSize int,
	logger *zap.Logger,
) *SyncService {
	if batchSize <= 0 {
		batchSize = 50
	}
	return &SyncService{
		paramRepo:     paramRepo,
		discoveryRepo: discoveryRepo,
		cmdQueue:      cmdQueue,
		config:        config,
		batchSize:     batchSize,
		logger:        logger,
	}
}

// StartSync initiates a full parameter value sync by enqueuing batched
// GetParameterValues commands for the device's root object.
func (s *SyncService) StartSync(ctx context.Context, dev *model.Device, paramPaths []string) error {
	// Update discovery log status to syncing if exists.
	log, _ := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if log != nil {
		_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoverySyncing, "")
	}

	// Batch parameter paths into GPV commands.
	batches := batchPaths(paramPaths, s.batchSize)
	for i, batch := range batches {
		gpvParams, err := json.Marshal(map[string]interface{}{
			"names": batch,
		})
		if err != nil {
			return fmt.Errorf("marshal GPV batch %d: %w", i, err)
		}

		cmd := &cmdqueue.Command{
			ID:         uuid.New().String(),
			Method:     MethodGetParameterValues,
			Params:     gpvParams,
			Priority:   10 + i, // Lower priority than discovery commands.
			CommandKey: fmt.Sprintf("sync-gpv-%s-%d", dev.SerialNumber, i),
		}

		if err := s.cmdQueue.Push(ctx, dev.SerialNumber, cmd); err != nil {
			return fmt.Errorf("enqueue GPV batch %d: %w", i, err)
		}
	}

	s.logger.Info("parameter sync started",
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("total_params", len(paramPaths)),
		zap.Int("batches", len(batches)),
	)

	return nil
}

// HandleSyncResult processes a GPV response and upserts parameter values into the database.
func (s *SyncService) HandleSyncResult(ctx context.Context, dev *model.Device,
	paramValues []tr069.ParameterValueStruct) error {

	if len(paramValues) == 0 {
		return nil
	}

	params := make([]model.DeviceParameter, 0, len(paramValues))
	now := time.Now()
	for _, pv := range paramValues {
		params = append(params, model.DeviceParameter{
			DeviceID:       dev.ID,
			ParameterPath:  pv.Name,
			ParameterValue: pv.Value,
			ParameterType:  inferParameterType(pv.Value, pv.Type),
			Writable:       false, // Will be updated from GPN data if available.
			LastUpdatedAt:  now,
		})
	}

	if err := s.paramRepo.BatchUpsert(ctx, dev.ID, params); err != nil {
		return fmt.Errorf("batch upsert parameters: %w", err)
	}

	s.logger.Debug("parameters synced",
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("count", len(params)),
	)

	return nil
}

// CompleteSyncLog marks the discovery log as completed after all sync batches finish.
func (s *SyncService) CompleteSyncLog(ctx context.Context, deviceID uuid.UUID) error {
	log, err := s.discoveryRepo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get discovery log: %w", err)
	}
	if log != nil && log.Status == DiscoverySyncing {
		return s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryCompleted, "")
	}
	return nil
}

// batchPaths splits parameter paths into batches of the given size.
func batchPaths(paths []string, batchSize int) [][]string {
	var batches [][]string
	for i := 0; i < len(paths); i += batchSize {
		end := i + batchSize
		if end > len(paths) {
			end = len(paths)
		}
		batches = append(batches, paths[i:end])
	}
	return batches
}

// inferParameterType tries to determine the TR069 parameter type from value and SOAP type hint.
func inferParameterType(value, soapType string) model.ParameterType {
	switch soapType {
	case "xsd:string", "string":
		return model.ParameterType("string")
	case "xsd:unsignedInt", "unsignedInt":
		return model.ParameterType("unsignedInt")
	case "xsd:int", "int":
		return model.ParameterType("int")
	case "xsd:boolean", "boolean":
		return model.ParameterType("boolean")
	case "xsd:dateTime", "dateTime":
		return model.ParameterType("dateTime")
	default:
		return model.ParameterType("string")
	}
}

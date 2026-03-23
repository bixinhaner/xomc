package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

const (
	// MethodGetParameterNames is the TR069 RPC method for discovering parameters.
	MethodGetParameterNames = "GetParameterNames"

	// discoveryStateTTL is the TTL for Redis discovery tracking keys.
	// If discovery stalls (device disconnects), keys auto-expire.
	discoveryStateTTL = 1 * time.Hour
)

// Redis key helpers for level-by-level discovery state.
func discoveryPendingKey(deviceSN string) string {
	return fmt.Sprintf("provision:discovery:pending:%s", deviceSN)
}

func discoveryParamsKey(deviceSN string) string {
	return fmt.Sprintf("provision:discovery:params:%s", deviceSN)
}

// DiscoveryService handles automatic parameter tree discovery and data model creation.
type DiscoveryService struct {
	discoveryRepo ParameterDiscoveryLogRepository
	dmRepo        datamodel.DataModelRepository
	dmRegistry    *datamodel.DataModelRegistry
	cmdQueue      cmdqueue.CommandQueue
	redis         redis.UniversalClient
	config        appconfig.AutoDiscoveryConfig
	logger        *zap.Logger
}

// NewDiscoveryService creates a new DiscoveryService.
func NewDiscoveryService(
	discoveryRepo ParameterDiscoveryLogRepository,
	dmRepo datamodel.DataModelRepository,
	dmRegistry *datamodel.DataModelRegistry,
	cmdQueue cmdqueue.CommandQueue,
	redisClient redis.UniversalClient,
	config appconfig.AutoDiscoveryConfig,
	logger *zap.Logger,
) *DiscoveryService {
	return &DiscoveryService{
		discoveryRepo: discoveryRepo,
		dmRepo:        dmRepo,
		dmRegistry:    dmRegistry,
		cmdQueue:      cmdQueue,
		redis:         redisClient,
		config:        config,
		logger:        logger,
	}
}

// StartDiscovery initiates parameter tree discovery for a device by enqueuing
// a GetParameterNames RPC command with NextLevel=true (one level at a time).
// Subsequent levels are discovered recursively via HandleLevelGPNResponse.
func (s *DiscoveryService) StartDiscovery(ctx context.Context, dev *model.Device) (*ParameterDiscoveryLog, error) {
	log := NewParameterDiscoveryLog(dev.ID, dev.SerialNumber, dev.OUI, dev.ProductClass, dev.FirmwareVersion)
	log.Status = DiscoveryDiscovering

	if err := s.discoveryRepo.Create(ctx, log); err != nil {
		return nil, fmt.Errorf("create discovery log: %w", err)
	}

	// Initialize Redis tracking state: pending=1 (for the root GPN).
	pendingKey := discoveryPendingKey(dev.SerialNumber)
	paramsKey := discoveryParamsKey(dev.SerialNumber)
	s.redis.Del(ctx, pendingKey, paramsKey)
	s.redis.Set(ctx, pendingKey, 1, discoveryStateTTL)

	// Enqueue GetParameterNames for the root object with NextLevel=true.
	// This returns only the immediate children, avoiding overwhelming the device.
	gpnParams, err := json.Marshal(map[string]interface{}{
		"path":       "Device.",
		"next_level": true,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal GPN params: %w", err)
	}

	cmd := &cmdqueue.Command{
		ID:         uuid.New().String(),
		Method:     MethodGetParameterNames,
		Params:     gpnParams,
		Priority:   1,
		CommandKey: fmt.Sprintf("discovery-gpn-%s-Device_", dev.SerialNumber),
	}

	if err := s.cmdQueue.Push(ctx, dev.SerialNumber, cmd); err != nil {
		_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryFailed, err.Error())
		s.redis.Del(ctx, pendingKey, paramsKey)
		return nil, fmt.Errorf("enqueue GPN for discovery: %w", err)
	}

	s.logger.Info("parameter discovery started (level-by-level)",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("discovery_id", log.ID.String()),
	)

	return log, nil
}

// HandleLevelGPNResponse processes one level's GPN response in the incremental
// discovery process. It separates leaf parameters from sub-objects, queues new
// GPN requests for non-excluded sub-objects, and finalizes the DataModel when
// all levels have been explored.
//
// Returns a non-nil DataModel only when discovery is fully complete.
func (s *DiscoveryService) HandleLevelGPNResponse(ctx context.Context, dev *model.Device,
	paramInfos []tr069.ParameterInfoStruct) (*datamodel.DataModel, error) {

	var leafParams []tr069.ParameterInfoStruct
	var subObjects []string

	for _, info := range paramInfos {
		if strings.HasSuffix(info.Name, ".") {
			// Sub-object node — check exclusion list before queuing deeper discovery.
			if !s.isExcludedPath(info.Name) {
				subObjects = append(subObjects, info.Name)
			} else {
				s.logger.Info("excluding path from discovery",
					zap.String("device_sn", dev.SerialNumber),
					zap.String("path", info.Name),
				)
			}
		} else {
			// Leaf parameter — accumulate for DataModel creation.
			leafParams = append(leafParams, info)
		}
	}

	paramsKey := discoveryParamsKey(dev.SerialNumber)
	pendingKey := discoveryPendingKey(dev.SerialNumber)

	// Store leaf params in Redis list.
	if len(leafParams) > 0 {
		values := make([]interface{}, len(leafParams))
		for i, p := range leafParams {
			data, _ := json.Marshal(p)
			values[i] = data
		}
		if err := s.redis.RPush(ctx, paramsKey, values...).Err(); err != nil {
			s.logger.Error("store leaf params in Redis", zap.Error(err))
		}
		// Refresh TTL on params key.
		s.redis.Expire(ctx, paramsKey, discoveryStateTTL)
	}

	// Queue GPN for each non-excluded sub-object, incrementing pending count.
	for _, objPath := range subObjects {
		gpnParams, _ := json.Marshal(map[string]interface{}{
			"path":       objPath,
			"next_level": true,
		})

		// Use path as part of command key for deduplication.
		cmdKey := fmt.Sprintf("discovery-gpn-%s-%s",
			dev.SerialNumber, strings.ReplaceAll(objPath, ".", "_"))

		cmd := &cmdqueue.Command{
			ID:         uuid.New().String(),
			Method:     MethodGetParameterNames,
			Params:     gpnParams,
			Priority:   1,
			CommandKey: cmdKey,
		}

		if err := s.cmdQueue.Push(ctx, dev.SerialNumber, cmd); err != nil {
			s.logger.Warn("enqueue sub-level GPN",
				zap.Error(err),
				zap.String("path", objPath),
			)
			continue
		}

		// Increment pending count for the new GPN request.
		s.redis.Incr(ctx, pendingKey)
	}

	// Decrement pending for this completed response.
	remaining, err := s.redis.Decr(ctx, pendingKey).Result()
	if err != nil {
		s.logger.Error("decrement discovery pending counter", zap.Error(err))
		return nil, nil
	}

	s.logger.Info("discovery level processed",
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("leaf_params", len(leafParams)),
		zap.Int("sub_objects", len(subObjects)),
		zap.Int64("remaining_levels", remaining),
	)

	if remaining > 0 {
		return nil, nil // More levels to explore.
	}

	// All levels done — finalize discovery.
	return s.finalizeDiscovery(ctx, dev)
}

// finalizeDiscovery aggregates all accumulated parameters from Redis and creates the DataModel.
func (s *DiscoveryService) finalizeDiscovery(ctx context.Context, dev *model.Device) (*datamodel.DataModel, error) {
	paramsKey := discoveryParamsKey(dev.SerialNumber)
	pendingKey := discoveryPendingKey(dev.SerialNumber)

	// Read all accumulated leaf parameters from Redis.
	rawParams, err := s.redis.LRange(ctx, paramsKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("read accumulated params from Redis: %w", err)
	}

	allParams := make([]tr069.ParameterInfoStruct, 0, len(rawParams))
	for _, raw := range rawParams {
		var p tr069.ParameterInfoStruct
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			allParams = append(allParams, p)
		}
	}

	// Clean up Redis tracking keys.
	s.redis.Del(ctx, pendingKey, paramsKey)

	s.logger.Info("discovery finalized, creating data model",
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("total_leaf_params", len(allParams)),
	)

	// Delegate to HandleDiscoveryResult for DataModel creation.
	return s.HandleDiscoveryResult(ctx, dev, allParams)
}

// CleanupState removes Redis tracking keys for an in-progress discovery.
// Called when a device reconnects and we need to restart discovery from scratch.
func (s *DiscoveryService) CleanupState(ctx context.Context, deviceSN string) {
	pendingKey := discoveryPendingKey(deviceSN)
	paramsKey := discoveryParamsKey(deviceSN)
	deleted, _ := s.redis.Del(ctx, pendingKey, paramsKey).Result()
	if deleted > 0 {
		s.logger.Info("cleaned up stale discovery Redis state",
			zap.String("device_sn", deviceSN),
			zap.Int64("keys_deleted", deleted),
		)
	}
}

// isExcludedPath checks if a path should be skipped during discovery.
func (s *DiscoveryService) isExcludedPath(path string) bool {
	for _, excluded := range s.config.ExcludePaths {
		if path == excluded || strings.HasPrefix(path, excluded) {
			return true
		}
	}
	return false
}

// HandleDiscoveryResult processes the aggregated GPN results and creates a new DataModel.
// It is called by finalizeDiscovery after all levels have been explored,
// or directly for manual trigger scenarios.
func (s *DiscoveryService) HandleDiscoveryResult(ctx context.Context, dev *model.Device,
	paramInfos []tr069.ParameterInfoStruct) (*datamodel.DataModel, error) {

	// Get the latest discovery log for this device (may not exist for manual triggers).
	log, err := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if err != nil {
		// No discovery log found — this is a manual trigger. Create one on-the-fly.
		log = NewParameterDiscoveryLog(dev.ID, dev.SerialNumber, dev.OUI, dev.ProductClass, dev.FirmwareVersion)
		log.Status = DiscoveryDiscovering
		if createErr := s.discoveryRepo.Create(ctx, log); createErr != nil {
			s.logger.Warn("create discovery log for manual trigger", zap.Error(createErr))
			// Continue without log — don't fail the discovery.
			log = nil
		}
	}

	// Check if a matching auto-discovered DataModel already exists for this OUI+ProductClass.
	// Auto templates are stored without firmware_version, so we search with empty firmware.
	existingDM, err := s.dmRegistry.ResolveWithFirmware(ctx, dev.Carrier, dev.Technology,
		dev.OUI, dev.ProductClass, "")
	if err != nil {
		return nil, fmt.Errorf("check existing data model: %w", err)
	}
	if existingDM != nil && existingDM.SourceType == datamodel.SourceAutoDiscovered {
		// Update existing auto template's parameter tree instead of creating a new one.
		paramTree, buildErr := buildParameterTree(paramInfos)
		if buildErr != nil {
			if log != nil {
				log.Status = DiscoveryFailed
				log.ErrorMessage = buildErr.Error()
				_ = s.discoveryRepo.Update(ctx, log)
			}
			return nil, fmt.Errorf("build parameter tree for update: %w", buildErr)
		}
		existingDM.ParameterTree = paramTree
		if updateErr := s.dmRepo.Update(ctx, existingDM); updateErr != nil {
			s.logger.Warn("failed to update existing auto template, will create new",
				zap.Error(updateErr),
			)
		} else {
			if log != nil {
				log.DataModelID = &existingDM.ID
				log.ParameterCount = len(paramInfos)
				log.Status = DiscoveryCompleted
				_ = s.discoveryRepo.Update(ctx, log)
			}
			s.logger.Info("existing auto template updated",
				zap.String("device_sn", dev.SerialNumber),
				zap.String("model_id", existingDM.ID.String()),
			)
			if existingDM.IsActive {
				_ = s.dmRegistry.InvalidateCache(ctx, existingDM)
			}
			return existingDM, nil
		}
	} else if existingDM != nil {
		// A manual template already exists for this OUI+ProductClass — skip creation.
		if log != nil {
			log.DataModelID = &existingDM.ID
			log.ParameterCount = len(paramInfos)
			log.Status = DiscoveryCompleted
			_ = s.discoveryRepo.Update(ctx, log)
		}
		s.logger.Info("manual template already exists, skipping auto creation",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("model_id", existingDM.ID.String()),
		)
		return existingDM, nil
	}

	// Build parameter tree from GPN response.
	paramTree, err := buildParameterTree(paramInfos)
	if err != nil {
		if log != nil {
			log.Status = DiscoveryFailed
			log.ErrorMessage = err.Error()
			_ = s.discoveryRepo.Update(ctx, log)
		}
		return nil, fmt.Errorf("build parameter tree: %w", err)
	}

	// Create new auto-discovered DataModel.
	// Auto templates store OUI + ProductClass only (no FirmwareVersion).
	dm := &datamodel.DataModel{
		ID:              uuid.New(),
		Carrier:         dev.Carrier,
		Technology:      dev.Technology,
		Version:         "1.0",
		OUI:             dev.OUI,
		ProductClass:    dev.ProductClass,
		FirmwareVersion: "", // Auto templates do not store firmware version.
		Scope:           model.ScopeProduct,
		Status:          datamodel.StatusDraft,
		RootObject:      detectRootObject(paramInfos),
		ParameterTree:   paramTree,
		SourceType:      datamodel.SourceAutoDiscovered,
		Source:          "auto_discovered",
		Description:     fmt.Sprintf("自动发现: %s %s from %s", dev.OUI, dev.ProductClass, dev.SerialNumber),
	}

	if s.config.AutoActivateModel {
		dm.Status = datamodel.StatusActive
		dm.IsActive = true
	}

	if err := s.dmRepo.Create(ctx, dm); err != nil {
		if log != nil {
			log.Status = DiscoveryFailed
			log.ErrorMessage = err.Error()
			_ = s.discoveryRepo.Update(ctx, log)
		}
		return nil, fmt.Errorf("create data model: %w", err)
	}

	// Update discovery log.
	if log != nil {
		log.DataModelID = &dm.ID
		log.ParameterCount = len(paramInfos)
		log.Status = DiscoveryCompleted
		if err := s.discoveryRepo.Update(ctx, log); err != nil {
			s.logger.Warn("update discovery log after model creation",
				zap.Error(err),
			)
		}
	}

	// Invalidate cache so new model is discoverable.
	if dm.IsActive {
		if err := s.dmRegistry.InvalidateCache(ctx, dm); err != nil {
			s.logger.Warn("invalidate cache after auto-discovery",
				zap.Error(err),
			)
		}
	}

	s.logger.Info("data model created via auto-discovery",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("model_id", dm.ID.String()),
		zap.Int("parameter_count", len(paramInfos)),
		zap.String("status", string(dm.Status)),
	)

	return dm, nil
}

// buildParameterTree converts GPN response parameter info list to a JSON parameter tree.
func buildParameterTree(infos []tr069.ParameterInfoStruct) (json.RawMessage, error) {
	params := make([]map[string]interface{}, 0, len(infos))
	for _, info := range infos {
		// Skip object nodes (paths ending with "."), only keep leaf parameters.
		if strings.HasSuffix(info.Name, ".") {
			continue
		}
		params = append(params, map[string]interface{}{
			"path":     info.Name,
			"writable": info.Writable,
			"type":     "string", // Default type; actual type determined during GPV sync.
		})
	}

	data, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal parameter tree: %w", err)
	}
	return data, nil
}

// detectRootObject determines the root object from the discovered parameters.
func detectRootObject(infos []tr069.ParameterInfoStruct) string {
	for _, info := range infos {
		if info.Name == "Device." || strings.HasPrefix(info.Name, "Device.") {
			return "Device."
		}
		if info.Name == "InternetGatewayDevice." || strings.HasPrefix(info.Name, "InternetGatewayDevice.") {
			return "InternetGatewayDevice."
		}
	}
	return "Device."
}

package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

// discoveryEnqueuedKey tracks which GPN paths have already been enqueued,
// preventing duplicate commands when discovery spans multiple ACS sessions.
func discoveryEnqueuedKey(deviceSN string) string {
	return fmt.Sprintf("provision:discovery:enqueued:%s", deviceSN)
}

// discoveryStallCheckKey is a Redis mutex key to ensure only one stall-check
// goroutine runs per device at a time. TTL auto-expires if the goroutine crashes.
func discoveryStallCheckKey(deviceSN string) string {
	return fmt.Sprintf("provision:discovery:stall_check:%s", deviceSN)
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
	enqueuedKey := discoveryEnqueuedKey(dev.SerialNumber)
	stallKey := discoveryStallCheckKey(dev.SerialNumber)
	s.redis.Del(ctx, pendingKey, paramsKey, enqueuedKey, stallKey)
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

	// Guard: if the pending key no longer exists, discovery has already finalized.
	// Late-arriving GPN responses (dispatched before finalization) must be ignored
	// to prevent cascading re-finalizations on a non-existent counter.
	pendingExists, _ := s.redis.Exists(ctx, discoveryPendingKey(dev.SerialNumber)).Result()
	if pendingExists == 0 {
		s.logger.Debug("ignoring late GPN response, discovery already finalized",
			zap.String("device_sn", dev.SerialNumber),
		)
		return nil, nil
	}

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

	// Filter multi-instance objects: only explore the lowest-numbered instance per parent.
	var skippedCount int
	subObjects, skippedCount = filterMultiInstanceObjects(subObjects)
	if skippedCount > 0 {
		s.logger.Info("multi-instance objects filtered",
			zap.String("device_sn", dev.SerialNumber),
			zap.Int("skipped_instances", skippedCount),
			zap.Int("remaining_sub_objects", len(subObjects)),
		)
	}

	// Queue GPN for each non-excluded sub-object, incrementing pending count.
	// Use a Redis Set to deduplicate: if a path was already enqueued in this
	// discovery round, skip it. This prevents duplicate commands when discovery
	// spans multiple ACS sessions.
	enqueuedKey := discoveryEnqueuedKey(dev.SerialNumber)
	for _, objPath := range subObjects {
		// Check if this path was already enqueued.
		added, _ := s.redis.SAdd(ctx, enqueuedKey, objPath).Result()
		if added == 0 {
			continue // Already enqueued in this discovery round.
		}
		s.redis.Expire(ctx, enqueuedKey, discoveryStateTTL)

		gpnParams, _ := json.Marshal(map[string]interface{}{
			"path":       objPath,
			"next_level": true,
		})

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
			// Remove from enqueued set since push failed.
			s.redis.SRem(ctx, enqueuedKey, objPath)
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
		// Check if queue is empty — may indicate a stall, but could also be a
		// temporary race: ACS dispatches commands faster than the provision engine
		// processes NATS events and enqueues new sub-level GPNs.
		// Use delayed stall detection instead of immediate force-finalize.
		queueLen, qErr := s.cmdQueue.Len(ctx, dev.SerialNumber)
		if qErr == nil && queueLen == 0 {
			s.scheduleStallRecovery(dev, remaining)
		}
		return nil, nil // More levels to explore.
	}

	// All levels done — finalize discovery.
	return s.finalizeDiscovery(ctx, dev)
}

// finalizeDiscovery aggregates all accumulated parameters from Redis and creates the DataModel.
func (s *DiscoveryService) finalizeDiscovery(ctx context.Context, dev *model.Device) (*datamodel.DataModel, error) {
	paramsKey := discoveryParamsKey(dev.SerialNumber)
	pendingKey := discoveryPendingKey(dev.SerialNumber)
	enqueuedKey := discoveryEnqueuedKey(dev.SerialNumber)

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

	// Clean up Redis tracking keys (including stall check mutex).
	stallKey := discoveryStallCheckKey(dev.SerialNumber)
	s.redis.Del(ctx, pendingKey, paramsKey, enqueuedKey, stallKey)

	s.logger.Info("discovery finalized, creating data model",
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("total_leaf_params", len(allParams)),
	)

	// Delegate to HandleDiscoveryResult for DataModel creation.
	return s.HandleDiscoveryResult(ctx, dev, allParams)
}

// scheduleStallRecovery launches a background goroutine (at most one per device)
// that waits a short period and then re-checks whether discovery is truly stalled.
// This avoids premature force-finalize caused by the race between ACS command
// dispatch and provision engine NATS processing.
func (s *DiscoveryService) scheduleStallRecovery(dev *model.Device, currentPending int64) {
	stallKey := discoveryStallCheckKey(dev.SerialNumber)
	ctx := context.Background()

	// SETNX mutex: only one stall-check goroutine per device.
	// TTL = 30s as safety net if goroutine crashes.
	ok, err := s.redis.SetNX(ctx, stallKey, 1, 30*time.Second).Result()
	if err != nil || !ok {
		return // Another stall-check goroutine is already running for this device.
	}

	go func() {
		defer s.redis.Del(context.Background(), stallKey)

		// Wait for provision engine to catch up with NATS processing.
		time.Sleep(15 * time.Second)

		bgCtx := context.Background()

		// Re-check: is discovery still pending?
		pendingExists, _ := s.redis.Exists(bgCtx, discoveryPendingKey(dev.SerialNumber)).Result()
		if pendingExists == 0 {
			return // Discovery already finalized normally.
		}

		remaining, _ := s.redis.Get(bgCtx, discoveryPendingKey(dev.SerialNumber)).Int64()
		if remaining <= 0 {
			return // Pending reached zero, finalizeDiscovery will handle it.
		}

		queueLen, qErr := s.cmdQueue.Len(bgCtx, dev.SerialNumber)
		if qErr != nil || queueLen > 0 {
			return // Queue has items, discovery is progressing.
		}

		// Stall confirmed: pending > 0 AND queue empty after 15s delay.
		s.logger.Warn("discovery stall confirmed after delay, force-finalizing",
			zap.String("device_sn", dev.SerialNumber),
			zap.Int64("remaining_pending", remaining),
		)

		if _, err := s.finalizeDiscovery(bgCtx, dev); err != nil {
			s.logger.Error("force-finalize after stall detection failed",
				zap.String("device_sn", dev.SerialNumber),
				zap.Error(err),
			)
		}
	}()
}

// CleanupState removes Redis tracking keys for an in-progress discovery.
// Called when a device reconnects and we need to restart discovery from scratch.
func (s *DiscoveryService) CleanupState(ctx context.Context, deviceSN string) {
	pendingKey := discoveryPendingKey(deviceSN)
	paramsKey := discoveryParamsKey(deviceSN)
	enqueuedKey := discoveryEnqueuedKey(deviceSN)
	stallKey := discoveryStallCheckKey(deviceSN)
	deleted, _ := s.redis.Del(ctx, pendingKey, paramsKey, enqueuedKey, stallKey).Result()
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

	// Check if a matching auto-discovered DataModel already exists for this OUI+ProductClass+FirmwareVersion.
	existingDM, err := s.dmRegistry.ResolveWithFirmware(ctx, dev.Carrier, dev.Technology,
		dev.OUI, dev.ProductClass, dev.FirmwareVersion)
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
	dm := &datamodel.DataModel{
		ID:              uuid.New(),
		Carrier:         dev.Carrier,
		Technology:      dev.Technology,
		Version:         "1.0",
		OUI:             dev.OUI,
		ProductClass:    dev.ProductClass,
		FirmwareVersion: dev.FirmwareVersion,
		Scope:           model.ScopeProduct,
		Status:          datamodel.StatusDraft,
		RootObject:      detectRootObject(paramInfos),
		ParameterTree:   paramTree,
		SourceType:      datamodel.SourceAutoDiscovered,
		Source:          "auto_discovered",
		Description:     fmt.Sprintf("自动发现: %s %s (FW: %s) from %s", dev.OUI, dev.ProductClass, dev.FirmwareVersion, dev.SerialNumber),
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

// filterMultiInstanceObjects deduplicates multi-instance object paths.
// TR069 multi-instance objects have numeric instance IDs: "Parent.1.", "Parent.2.", etc.
// Since all instances share the same parameter structure, we only need to explore
// the lowest-numbered instance per parent path and skip the rest.
//
// Example:
//
//	Input:  ["Device.Services.FAPService.1.", "Device.Services.FAPService.2.", "Device.ManagementServer."]
//	Output: ["Device.Services.FAPService.1.", "Device.ManagementServer."], skipped=1
func filterMultiInstanceObjects(paths []string) (filtered []string, skippedCount int) {
	if len(paths) == 0 {
		return paths, 0
	}

	// Group by parent path. For each parent, track the lowest numeric instance
	// and collect non-numeric children.
	type instanceInfo struct {
		minNum  int    // lowest instance number seen
		minPath string // full path of the lowest instance
		count   int    // total numeric instances seen
	}
	instances := make(map[string]*instanceInfo) // parent → info
	var nonInstancePaths []string               // paths that are not numeric instances

	for _, p := range paths {
		parent, segment := splitLastSegment(p)
		if parent == "" {
			// No parent (top-level object) — always keep.
			nonInstancePaths = append(nonInstancePaths, p)
			continue
		}

		num, err := strconv.Atoi(segment)
		if err != nil {
			// Non-numeric segment — regular sub-object, always keep.
			nonInstancePaths = append(nonInstancePaths, p)
			continue
		}

		// Numeric segment — this is a multi-instance entry.
		info, exists := instances[parent]
		if !exists {
			instances[parent] = &instanceInfo{minNum: num, minPath: p, count: 1}
		} else {
			info.count++
			if num < info.minNum {
				info.minNum = num
				info.minPath = p
			}
		}
	}

	// Build the filtered list: non-instance paths + one representative per instance group.
	filtered = make([]string, 0, len(nonInstancePaths)+len(instances))
	filtered = append(filtered, nonInstancePaths...)
	for _, info := range instances {
		filtered = append(filtered, info.minPath)
		skippedCount += info.count - 1
	}

	return filtered, skippedCount
}

// splitLastSegment splits "Device.Services.FAPService.1." into
// parent="Device.Services.FAPService" and segment="1".
// The input path is expected to end with ".".
func splitLastSegment(path string) (parent, segment string) {
	// Remove trailing dot: "Device.Services.FAPService.1." → "Device.Services.FAPService.1"
	trimmed := strings.TrimSuffix(path, ".")
	idx := strings.LastIndex(trimmed, ".")
	if idx < 0 {
		return "", trimmed
	}
	return trimmed[:idx], trimmed[idx+1:]
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

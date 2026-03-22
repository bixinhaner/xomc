package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
)

// DiscoveryService handles automatic parameter tree discovery and data model creation.
type DiscoveryService struct {
	discoveryRepo ParameterDiscoveryLogRepository
	dmRepo        datamodel.DataModelRepository
	dmRegistry    *datamodel.DataModelRegistry
	cmdQueue      cmdqueue.CommandQueue
	config        appconfig.AutoDiscoveryConfig
	logger        *zap.Logger
}

// NewDiscoveryService creates a new DiscoveryService.
func NewDiscoveryService(
	discoveryRepo ParameterDiscoveryLogRepository,
	dmRepo datamodel.DataModelRepository,
	dmRegistry *datamodel.DataModelRegistry,
	cmdQueue cmdqueue.CommandQueue,
	config appconfig.AutoDiscoveryConfig,
	logger *zap.Logger,
) *DiscoveryService {
	return &DiscoveryService{
		discoveryRepo: discoveryRepo,
		dmRepo:        dmRepo,
		dmRegistry:    dmRegistry,
		cmdQueue:      cmdQueue,
		config:        config,
		logger:        logger,
	}
}

// StartDiscovery initiates parameter tree discovery for a device by enqueuing
// a GetParameterNames RPC command. The actual processing of the response happens
// in HandleDiscoveryResult when the ACS engine calls back.
func (s *DiscoveryService) StartDiscovery(ctx context.Context, dev *model.Device) (*ParameterDiscoveryLog, error) {
	log := NewParameterDiscoveryLog(dev.ID, dev.SerialNumber, dev.OUI, dev.ProductClass, dev.FirmwareVersion)
	log.Status = DiscoveryDiscovering

	if err := s.discoveryRepo.Create(ctx, log); err != nil {
		return nil, fmt.Errorf("create discovery log: %w", err)
	}

	// Enqueue GetParameterNames for the root object with NextLevel=false (full tree).
	gpnParams, err := json.Marshal(map[string]interface{}{
		"path":       "Device.",
		"next_level": false,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal GPN params: %w", err)
	}

	timeout := s.config.GPNTimeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	cmd := &cmdqueue.Command{
		ID:         uuid.New().String(),
		Method:     MethodGetParameterNames,
		Params:     gpnParams,
		Priority:   1,
		CommandKey: fmt.Sprintf("discovery-gpn-%s", log.ID),
	}

	if err := s.cmdQueue.Push(ctx, dev.SerialNumber, cmd); err != nil {
		_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryFailed, err.Error())
		return nil, fmt.Errorf("enqueue GPN for discovery: %w", err)
	}

	s.logger.Info("parameter discovery started",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("discovery_id", log.ID.String()),
	)

	return log, nil
}

// HandleDiscoveryResult processes the GPN response and creates a new DataModel.
// It is called by the provisioning engine when the ACS receives a GetParameterNamesResponse.
func (s *DiscoveryService) HandleDiscoveryResult(ctx context.Context, dev *model.Device,
	paramInfos []tr069.ParameterInfoStruct) (*datamodel.DataModel, error) {

	// Get the latest discovery log for this device.
	log, err := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
	if err != nil {
		return nil, fmt.Errorf("get discovery log: %w", err)
	}

	// Check if a matching DataModel was created by another concurrent device.
	existingDM, err := s.dmRegistry.ResolveWithFirmware(ctx, dev.Carrier, dev.Technology,
		dev.OUI, dev.ProductClass, dev.FirmwareVersion)
	if err != nil {
		return nil, fmt.Errorf("check existing data model: %w", err)
	}
	if existingDM != nil {
		log.DataModelID = &existingDM.ID
		log.ParameterCount = len(paramInfos)
		log.Status = DiscoveryCompleted
		_ = s.discoveryRepo.Update(ctx, log)
		s.logger.Info("data model already exists, skipping creation",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("model_id", existingDM.ID.String()),
		)
		return existingDM, nil
	}

	// Build parameter tree from GPN response.
	paramTree, err := buildParameterTree(paramInfos)
	if err != nil {
		log.Status = DiscoveryFailed
		log.ErrorMessage = err.Error()
		_ = s.discoveryRepo.Update(ctx, log)
		return nil, fmt.Errorf("build parameter tree: %w", err)
	}

	// Create new DataModel.
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
		Source:          fmt.Sprintf("auto-discovery from device %s", dev.SerialNumber),
		Description:     fmt.Sprintf("自动发现: %s %s (FW: %s)", dev.OUI, dev.ProductClass, dev.FirmwareVersion),
	}

	if s.config.AutoActivateModel {
		dm.Status = datamodel.StatusActive
		dm.IsActive = true
	}

	if err := s.dmRepo.Create(ctx, dm); err != nil {
		log.Status = DiscoveryFailed
		log.ErrorMessage = err.Error()
		_ = s.discoveryRepo.Update(ctx, log)
		return nil, fmt.Errorf("create data model: %w", err)
	}

	// Update discovery log.
	log.DataModelID = &dm.ID
	log.ParameterCount = len(paramInfos)
	log.Status = DiscoveryCompleted
	if err := s.discoveryRepo.Update(ctx, log); err != nil {
		s.logger.Warn("update discovery log after model creation",
			zap.Error(err),
		)
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

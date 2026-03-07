package device

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// DeviceService provides business logic for device management.
type DeviceService struct {
	deviceRepo DeviceRepository
	paramRepo  DeviceParameterRepository
	heartbeat  *HeartbeatMonitor
	logger     *zap.Logger
}

// NewDeviceService creates a new DeviceService.
func NewDeviceService(
	deviceRepo DeviceRepository,
	paramRepo DeviceParameterRepository,
	heartbeat *HeartbeatMonitor,
	logger *zap.Logger,
) *DeviceService {
	return &DeviceService{
		deviceRepo: deviceRepo,
		paramRepo:  paramRepo,
		heartbeat:  heartbeat,
		logger:     logger,
	}
}

// RegisterFromInform creates a new device from a bootstrap Inform message.
func (s *DeviceService) RegisterFromInform(ctx context.Context, inform *tr069.InformMessage, carrier model.CarrierCode) (*model.Device, error) {
	// Check if device already exists
	existing, err := s.deviceRepo.GetBySerialNumber(ctx, inform.DeviceId.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("lookup device: %w", err)
	}
	if existing != nil {
		// Device already registered, just update it
		return s.UpdateFromInform(ctx, inform)
	}

	// Detect technology from parameters
	tech := detectTechnology(inform.ParameterList)

	now := time.Now()
	device := &model.Device{
		ID:                   uuid.New(),
		SerialNumber:         inform.DeviceId.SerialNumber,
		OUI:                  inform.DeviceId.OUI,
		ProductClass:         inform.DeviceId.ProductClass,
		Manufacturer:         inform.DeviceId.Manufacturer,
		Carrier:              carrier,
		Technology:           tech,
		Status:               model.DeviceActive,
		FirmwareVersion:      findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion"),
		ConnectionRequestURL: findParamValue(inform.ParameterList, "Device.ManagementServer.ConnectionRequestURL"),
		LastInformAt:         &now,
		LastInformEvents:     tr069.EventCodes(inform.Event),
		InformInterval:       300,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}

	// Store parameters
	s.storeInformParameters(ctx, device.ID, inform.ParameterList)

	// Refresh heartbeat
	if s.heartbeat != nil {
		s.heartbeat.RefreshHeartbeat(ctx, device.SerialNumber, device.InformInterval)
	}

	s.logger.Info("device registered from bootstrap",
		zap.String("serial_number", device.SerialNumber),
		zap.String("carrier", string(carrier)),
		zap.String("oui", device.OUI),
	)

	return device, nil
}

// UpdateFromInform updates an existing device from a periodic Inform message.
func (s *DeviceService) UpdateFromInform(ctx context.Context, inform *tr069.InformMessage) (*model.Device, error) {
	device, err := s.deviceRepo.GetBySerialNumber(ctx, inform.DeviceId.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("lookup device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", inform.DeviceId.SerialNumber)
	}

	// Update fields from Inform
	now := time.Now()
	device.OUI = inform.DeviceId.OUI
	device.ProductClass = inform.DeviceId.ProductClass
	device.Manufacturer = inform.DeviceId.Manufacturer
	device.FirmwareVersion = findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion")
	device.ConnectionRequestURL = findParamValue(inform.ParameterList, "Device.ManagementServer.ConnectionRequestURL")
	device.LastInformAt = &now
	device.LastInformEvents = tr069.EventCodes(inform.Event)

	ipAddr := findParamValue(inform.ParameterList, "Device.ManagementServer.UDPConnectionRequestAddress")
	if ipAddr != "" {
		device.IPAddress = ipAddr
	}

	// Auto-transition to active when device informs (it's communicating, so it's online)
	if device.Status == model.DeviceDiscovered || device.Status == model.DeviceOffline || device.Status == model.DeviceRegistered {
		device.Status = model.DeviceActive
		s.logger.Info("device auto-transitioned to active on inform",
			zap.String("serial_number", device.SerialNumber),
			zap.String("previous_status", string(device.Status)),
		)
	}

	if err := s.deviceRepo.Update(ctx, device); err != nil {
		return nil, fmt.Errorf("update device: %w", err)
	}

	// Store parameters
	s.storeInformParameters(ctx, device.ID, inform.ParameterList)

	// Refresh heartbeat
	if s.heartbeat != nil {
		s.heartbeat.RefreshHeartbeat(ctx, device.SerialNumber, device.InformInterval)
	}

	return device, nil
}

// TransitionStatus validates and executes a device state transition.
func (s *DeviceService) TransitionStatus(ctx context.Context, deviceID uuid.UUID, newStatus model.DeviceStatus) error {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}
	if device == nil {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	if err := ValidateTransition(device.Status, newStatus); err != nil {
		return err
	}

	if err := s.deviceRepo.UpdateStatus(ctx, deviceID, newStatus); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	s.logger.Info("device status transitioned",
		zap.String("device_id", deviceID.String()),
		zap.String("from", string(device.Status)),
		zap.String("to", string(newStatus)),
	)

	return nil
}

// GetDevice retrieves a device by ID.
func (s *DeviceService) GetDevice(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	return s.deviceRepo.GetByID(ctx, id)
}

// GetBySerialNumber retrieves a device by its serial number.
func (s *DeviceService) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	return s.deviceRepo.GetBySerialNumber(ctx, sn)
}

// ListDevices returns a paginated list of devices.
func (s *DeviceService) ListDevices(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error) {
	return s.deviceRepo.List(ctx, filter)
}

// GetDeviceParameters retrieves all parameters for a device.
func (s *DeviceService) GetDeviceParameters(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	return s.paramRepo.GetByDevice(ctx, deviceID)
}

// CountByStatus returns device counts grouped by status.
func (s *DeviceService) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return s.deviceRepo.CountByStatus(ctx, carrier)
}

func (s *DeviceService) storeInformParameters(ctx context.Context, deviceID uuid.UUID, params []tr069.ParameterValueStruct) {
	if len(params) == 0 {
		return
	}

	modelParams := make([]model.DeviceParameter, 0, len(params))
	for _, p := range params {
		modelParams = append(modelParams, model.DeviceParameter{
			DeviceID:       deviceID,
			ParameterPath:  p.Name,
			ParameterValue: p.Value,
			ParameterType:  model.ParamString,
			Writable:       false,
			LastUpdatedAt:  time.Now(),
		})
	}

	if err := s.paramRepo.BatchUpsert(ctx, deviceID, modelParams); err != nil {
		s.logger.Error("store inform parameters",
			zap.Error(err),
			zap.String("device_id", deviceID.String()),
		)
	}
}

// detectTechnology tries to determine the radio technology from Inform parameters.
func detectTechnology(params []tr069.ParameterValueStruct) model.Technology {
	for _, p := range params {
		if p.Name == "Device.DeviceInfo.ModelName" || p.Name == "Device.DeviceInfo.Description" {
			// Simple heuristic: check for NR/5G keywords
			if containsAny(p.Value, "NR", "5G", "gNB") {
				return model.TechNR
			}
		}
	}
	return model.TechLTE
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func findParamValue(params []tr069.ParameterValueStruct, name string) string {
	for _, p := range params {
		if p.Name == name {
			return p.Value
		}
	}
	return ""
}

// CreateDevice creates a new device from an API request.
func (s *DeviceService) CreateDevice(ctx context.Context, req CreateDeviceRequest) (*model.Device, error) {
	existing, err := s.deviceRepo.GetBySerialNumber(ctx, req.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("check existing device: %w", err)
	}
	if existing != nil {
		return nil, commonerrors.ErrAlreadyExists
	}

	now := time.Now()
	device := &model.Device{
		ID:           uuid.New(),
		SerialNumber: req.SerialNumber,
		OUI:          req.OUI,
		ProductClass: req.ProductClass,
		Manufacturer: req.Manufacturer,
		ModelName:    req.ModelName,
		Carrier:      req.Carrier,
		Technology:   req.Technology,
		Status:       model.DeviceRegistered,
		SiteName:     req.SiteName,
		SiteID:       req.SiteID,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}

	s.logger.Info("device created via API",
		zap.String("serial_number", device.SerialNumber),
		zap.String("carrier", string(device.Carrier)),
	)
	return device, nil
}

// UpdateDevice updates an existing device from an API request.
func (s *DeviceService) UpdateDevice(ctx context.Context, id uuid.UUID, req UpdateDeviceRequest) (*model.Device, error) {
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	if device == nil {
		return nil, nil
	}

	if req.SiteName != nil {
		device.SiteName = *req.SiteName
	}
	if req.SiteID != nil {
		device.SiteID = *req.SiteID
	}
	if req.ModelName != nil {
		device.ModelName = *req.ModelName
	}
	if req.Latitude != nil {
		device.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		device.Longitude = *req.Longitude
	}
	if req.Status != nil {
		device.Status = *req.Status
	}

	if err := s.deviceRepo.Update(ctx, device); err != nil {
		return nil, fmt.Errorf("update device: %w", err)
	}
	return device, nil
}

// DeleteDevice deletes a device by ID.
func (s *DeviceService) DeleteDevice(ctx context.Context, id uuid.UUID) error {
	return s.deviceRepo.Delete(ctx, id)
}

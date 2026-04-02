package device

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// GroupAssigner assigns a device to a group.
type GroupAssigner interface {
	BatchAddDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error)
}

// DeviceService provides business logic for device management.
type DeviceService struct {
	deviceRepo     DeviceRepository
	paramRepo      DeviceParameterRepository
	deviceInfoRepo DeviceInfoRepository
	regRepo        RegistrationRepository
	groupAssigner  GroupAssigner
	infoSyncer     *InfoSyncer
	heartbeat      *HeartbeatMonitor
	eventBus       event.EventBus
	cmdQueue       CommandQueue
	connReq        ConnectionRequester
	stunUpdater    StunAddressUpdater
	cache          *DeviceCache
	metrics        *DeviceMetrics
	logger         *zap.Logger
}

// CommandQueue defines the interface for queuing RPC commands to devices.
type CommandQueue interface {
	Push(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error
}

// ConnectionRequester sends Connection Request to wake a CPE device.
type ConnectionRequester interface {
	Send(ctx context.Context, deviceSN string, url string) error
}

// StunAddressUpdater syncs device STUN addresses to the address cache.
type StunAddressUpdater interface {
	SetFromInform(ctx context.Context, deviceSN, udpAddr string) error
}

// NewDeviceService creates a new DeviceService.
func NewDeviceService(
	deviceRepo DeviceRepository,
	paramRepo DeviceParameterRepository,
	heartbeat *HeartbeatMonitor,
	eventBus event.EventBus,
	logger *zap.Logger,
) *DeviceService {
	return &DeviceService{
		deviceRepo: deviceRepo,
		paramRepo:  paramRepo,
		heartbeat:  heartbeat,
		eventBus:   eventBus,
		logger:     logger,
	}
}

// SetCommandQueue sets the ACS command queue for device operations.
func (s *DeviceService) SetCommandQueue(q CommandQueue) {
	s.cmdQueue = q
}

// SetConnectionRequester sets the connection request client.
func (s *DeviceService) SetConnectionRequester(cr ConnectionRequester) {
	s.connReq = cr
}

// SetStunAddressUpdater sets the STUN address updater for syncing
// UDPConnectionRequestAddress from Inform to the STUN address cache.
func (s *DeviceService) SetStunAddressUpdater(u StunAddressUpdater) {
	s.stunUpdater = u
}

// SetDeviceCache sets the Redis device cache for fast serial number lookups.
func (s *DeviceService) SetDeviceCache(c *DeviceCache) {
	s.cache = c
}

// SetDeviceInfoRepo sets the device info repository for extended info management.
func (s *DeviceService) SetDeviceInfoRepo(repo DeviceInfoRepository) {
	s.deviceInfoRepo = repo
}

// SetInfoSyncer sets the parameter-to-device_info syncer.
func (s *DeviceService) SetInfoSyncer(syncer *InfoSyncer) {
	s.infoSyncer = syncer
}

// SetRegistrationRepo sets the device registration repository for pre-registration lookup.
func (s *DeviceService) SetRegistrationRepo(repo RegistrationRepository) {
	s.regRepo = repo
}

// SetGroupAssigner sets the group assigner for assigning devices to groups.
func (s *DeviceService) SetGroupAssigner(ga GroupAssigner) {
	s.groupAssigner = ga
}

// RebootDevice queues a Reboot command for the given device via the ACS command queue.
func (s *DeviceService) RebootDevice(ctx context.Context, id uuid.UUID) error {
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get device for reboot: %w", err)
	}
	if device == nil {
		return commonerrors.ErrNotFound
	}

	if s.cmdQueue == nil {
		return fmt.Errorf("command queue not configured")
	}

	cmd := &cmdqueue.Command{
		ID:       uuid.New().String(),
		Method:   "Reboot",
		Priority: 0, // highest priority
	}
	cmd.CommandKey = cmd.ID

	if err := s.cmdQueue.Push(ctx, device.SerialNumber, cmd); err != nil {
		return fmt.Errorf("queue reboot command: %w", err)
	}

	s.logger.Info("reboot command queued",
		zap.String("device_id", id.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.String("command_id", cmd.ID),
	)

	// Optionally trigger Connection Request to wake the device immediately
	if s.connReq != nil && device.ConnectionRequestURL != "" {
		go func() {
			if err := s.connReq.Send(context.Background(), device.SerialNumber, device.ConnectionRequestURL); err != nil {
				s.logger.Warn("connection request for reboot failed",
					zap.String("serial_number", device.SerialNumber),
					zap.Error(err),
				)
			}
		}()
	}

	return nil
}

// TriggerParamSync queues a GetParameterValues command to sync all parameters from the device.
func (s *DeviceService) TriggerParamSync(ctx context.Context, deviceID uuid.UUID) error {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device for param sync: %w", err)
	}
	if device == nil {
		return commonerrors.ErrNotFound
	}
	if s.cmdQueue == nil {
		return fmt.Errorf("command queue not configured")
	}

	paramsJSON, _ := json.Marshal(map[string]interface{}{
		"ParameterNames": []string{"Device."},
	})
	cmd := &cmdqueue.Command{
		ID:     uuid.New().String(),
		Method: "GetParameterValues",
		Params: paramsJSON,
	}
	cmd.CommandKey = cmd.ID

	if err := s.cmdQueue.Push(ctx, device.SerialNumber, cmd); err != nil {
		return fmt.Errorf("queue param sync command: %w", err)
	}

	s.logger.Info("param sync command queued",
		zap.String("device_id", deviceID.String()),
		zap.String("serial_number", device.SerialNumber))

	// Wake the device.
	if s.connReq != nil && device.ConnectionRequestURL != "" {
		go func() {
			s.connReq.Send(context.Background(), device.SerialNumber, device.ConnectionRequestURL)
		}()
	}

	return nil
}

// SetRFSwitch queues a SetParameterValues command to enable/disable the device RF.
// TODO(carrier): RF control path is currently hardcoded for LTE. When Carrier adapter
// layer is completed, replace with carrier.GetRFControlPath(device.Technology) to
// support both LTE (FAPControl.LTE.AdminState) and NR (FAPControl.NR.AdminState).
func (s *DeviceService) SetRFSwitch(ctx context.Context, deviceID uuid.UUID, enabled bool) error {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device for RF switch: %w", err)
	}
	if device == nil {
		return commonerrors.ErrNotFound
	}
	if s.cmdQueue == nil {
		return fmt.Errorf("command queue not configured")
	}

	// RF switch value: "1" for enabled, "0" for disabled.
	value := "0"
	if enabled {
		value = "1"
	}

	rfParamsJSON, _ := json.Marshal(map[string]interface{}{
		"ParameterList": []map[string]string{
			{"Name": "Device.Services.FAPService.1.FAPControl.LTE.AdminState", "Value": value},
		},
	})
	cmd := &cmdqueue.Command{
		ID:     uuid.New().String(),
		Method: "SetParameterValues",
		Params: rfParamsJSON,
	}
	cmd.CommandKey = cmd.ID

	if err := s.cmdQueue.Push(ctx, device.SerialNumber, cmd); err != nil {
		return fmt.Errorf("queue RF switch command: %w", err)
	}

	s.logger.Info("RF switch command queued",
		zap.String("device_id", deviceID.String()),
		zap.Bool("enabled", enabled))

	// Wake the device.
	if s.connReq != nil && device.ConnectionRequestURL != "" {
		go func() {
			s.connReq.Send(context.Background(), device.SerialNumber, device.ConnectionRequestURL)
		}()
	}

	return nil
}

// SetParameters queues a SetParameterValues RPC command for the given device.
func (s *DeviceService) SetParameters(ctx context.Context, deviceID uuid.UUID, params []ParameterValueItem) error {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device for set params: %w", err)
	}
	if device == nil {
		return commonerrors.ErrNotFound
	}

	if s.cmdQueue == nil {
		return fmt.Errorf("command queue not configured")
	}

	// Build SPV parameter list.
	spvParams := make([]map[string]string, 0, len(params))
	for _, p := range params {
		paramType := p.Type
		if paramType == "" {
			paramType = "xsd:string"
		}
		spvParams = append(spvParams, map[string]string{
			"name":  p.Path,
			"value": p.Value,
			"type":  paramType,
		})
	}

	paramsJSON, err := json.Marshal(map[string]interface{}{
		"parameter_list": spvParams,
		"parameter_key":  fmt.Sprintf("ui-spv-%d", time.Now().Unix()),
	})
	if err != nil {
		return fmt.Errorf("marshal SPV params: %w", err)
	}

	cmd := &cmdqueue.Command{
		ID:         uuid.New().String(),
		Method:     "SetParameterValues",
		Params:     paramsJSON,
		Priority:   5,
		CommandKey: fmt.Sprintf("ui-spv-%s", uuid.New().String()[:8]),
	}

	if err := s.cmdQueue.Push(ctx, device.SerialNumber, cmd); err != nil {
		return fmt.Errorf("queue SPV command: %w", err)
	}

	s.logger.Info("set parameter values command queued",
		zap.String("device_id", deviceID.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.Int("param_count", len(params)),
	)

	return nil
}

// ParameterValueItem represents a single parameter to set.
type ParameterValueItem struct {
	Path  string `json:"path"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// GetCommandQueue returns the command queue, or nil if not configured.
func (s *DeviceService) GetCommandQueue() CommandQueue {
	return s.cmdQueue
}

// SetMetrics attaches Prometheus metrics to the service.
func (s *DeviceService) SetMetrics(m *DeviceMetrics) {
	s.metrics = m
}

// RegisterFromInform creates a new device from a bootstrap Inform message.
func (s *DeviceService) RegisterFromInform(ctx context.Context, inform *tr069.InformMessage, carrier model.CarrierCode) (*model.Device, error) {
	ctx, span := tracing.StartSpan(ctx, tracing.DeviceTracerName, "Device RegisterFromInform",
		attribute.String("device.serial_number", inform.DeviceId.SerialNumber),
		attribute.String("device.oui", inform.DeviceId.OUI),
		attribute.String("device.carrier", string(carrier)),
	)
	defer span.End()

	s.logger.Info("RegisterFromInform: start",
		zap.String("serial_number", inform.DeviceId.SerialNumber),
		zap.String("oui", inform.DeviceId.OUI),
		zap.String("product_class", inform.DeviceId.ProductClass),
		zap.String("carrier", string(carrier)),
		zap.Int("param_count", len(inform.ParameterList)))

	// Check if device already exists
	s.logger.Debug("RegisterFromInform: checking if device exists",
		zap.String("serial_number", inform.DeviceId.SerialNumber))
	existing, err := s.getDeviceBySerialNumber(ctx, inform.DeviceId.SerialNumber)
	if err != nil {
		s.logger.Error("RegisterFromInform: GetBySerialNumber failed",
			zap.Error(err),
			zap.String("serial_number", inform.DeviceId.SerialNumber))
		return nil, fmt.Errorf("lookup device: %w", err)
	}
	if existing != nil {
		// Device already registered, just update it
		s.logger.Info("RegisterFromInform: device already exists, updating instead",
			zap.String("serial_number", inform.DeviceId.SerialNumber),
			zap.String("existing_device_id", existing.ID.String()))
		return s.UpdateFromInform(ctx, inform)
	}

	// Detect technology from parameters
	tech := detectTechnology(inform.ParameterList)
	s.logger.Debug("RegisterFromInform: technology detected",
		zap.String("technology", string(tech)))

	now := time.Now()
	udpAddr := findParamValue(inform.ParameterList, "Device.ManagementServer.UDPConnectionRequestAddress")

	device := &model.Device{
		ID:                          uuid.New(),
		SerialNumber:                inform.DeviceId.SerialNumber,
		OUI:                         inform.DeviceId.OUI,
		ProductClass:                inform.DeviceId.ProductClass,
		Manufacturer:                inform.DeviceId.Manufacturer,
		Carrier:                     carrier,
		Technology:                  tech,
		Status:                      model.DeviceActive,
		FirmwareVersion:             findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion"),
		ConnectionRequestURL:        findParamValue(inform.ParameterList, "Device.ManagementServer.ConnectionRequestURL"),
		IPAddress:                   udpAddr,
		NatDetected:                 udpAddr != "",
		UDPConnectionRequestAddress: udpAddr,
		LastInformAt:                &now,
		LastInformEvents:            tr069.EventCodes(inform.Event),
		InformInterval:              300,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}

	s.logger.Info("RegisterFromInform: creating device in DB",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.String("carrier", string(carrier)),
		zap.String("technology", string(tech)),
		zap.String("firmware_version", device.FirmwareVersion))

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		s.logger.Error("RegisterFromInform: deviceRepo.Create failed",
			zap.Error(err),
			zap.String("serial_number", device.SerialNumber))
		return nil, fmt.Errorf("create device: %w", err)
	}

	s.logger.Info("RegisterFromInform: device created in DB successfully",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber))

	// Write-through: cache the new device
	s.cacheDevice(ctx, device)

	// Create device_info record for extended info
	if err := s.CreateDeviceInfo(ctx, device.ID); err != nil {
		s.logger.Warn("create device_info for new device",
			zap.String("device_id", device.ID.String()),
			zap.Error(err))
	}

	// Check pre-registration: assign to specified group if found.
	if s.regRepo != nil && s.groupAssigner != nil {
		preReg, _ := s.regRepo.GetBySerialNumber(ctx, device.SerialNumber)
		if preReg != nil && preReg.GroupID != nil {
			if _, err := s.groupAssigner.BatchAddDevices(ctx, *preReg.GroupID, []uuid.UUID{device.ID}); err != nil {
				s.logger.Warn("assign device to pre-registered group",
					zap.String("device_id", device.ID.String()),
					zap.String("group_id", preReg.GroupID.String()),
					zap.Error(err))
			} else {
				s.logger.Info("device assigned to pre-registered group",
					zap.String("device_id", device.ID.String()),
					zap.String("group_id", preReg.GroupID.String()))
			}
			// Mark registration as online.
			s.regRepo.UpdateStatus(ctx, preReg.ID, string(global.RegistrationOnline))
		} else if preReg == nil {
			// No pre-registration: assign to default L2 group.
			defaultGroupID, _ := uuid.Parse(global.DefaultLevel2GroupID)
			if _, err := s.groupAssigner.BatchAddDevices(ctx, defaultGroupID, []uuid.UUID{device.ID}); err != nil {
				s.logger.Warn("assign device to default group",
					zap.String("device_id", device.ID.String()),
					zap.Error(err))
			}
		}
	}

	// Sync UDP address to STUN cache
	if udpAddr != "" && s.stunUpdater != nil {
		if err := s.stunUpdater.SetFromInform(ctx, device.SerialNumber, udpAddr); err != nil {
			s.logger.Warn("sync udp address to stun store on register",
				zap.String("serial_number", device.SerialNumber),
				zap.Error(err))
		}
	}

	// Record registration metric
	if s.metrics != nil {
		s.metrics.RegistrationsTotal.WithLabelValues(string(carrier)).Inc()
		s.metrics.DevicesTotal.WithLabelValues(string(device.Status), string(carrier)).Inc()
	}

	// Store parameters
	s.logger.Debug("RegisterFromInform: storing inform parameters",
		zap.Int("param_count", len(inform.ParameterList)))
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
	s.logger.Debug("UpdateFromInform: looking up device",
		zap.String("serial_number", inform.DeviceId.SerialNumber))

	device, err := s.getDeviceBySerialNumber(ctx, inform.DeviceId.SerialNumber)
	if err != nil {
		s.logger.Error("UpdateFromInform: device lookup failed",
			zap.Error(err),
			zap.String("serial_number", inform.DeviceId.SerialNumber))
		return nil, fmt.Errorf("lookup device: %w", err)
	}
	if device == nil {
		s.logger.Warn("UpdateFromInform: device not found in DB, caller should register",
			zap.String("serial_number", inform.DeviceId.SerialNumber))
		return nil, nil
	}

	s.logger.Debug("UpdateFromInform: device found, updating",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber))

	// Update fields from Inform
	now := time.Now()
	device.OUI = inform.DeviceId.OUI
	device.ProductClass = inform.DeviceId.ProductClass
	device.Manufacturer = inform.DeviceId.Manufacturer
	device.FirmwareVersion = findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion")
	device.ConnectionRequestURL = findParamValue(inform.ParameterList, "Device.ManagementServer.ConnectionRequestURL")
	device.LastInformAt = &now
	device.LastInformEvents = tr069.EventCodes(inform.Event)

	udpAddr := findParamValue(inform.ParameterList, "Device.ManagementServer.UDPConnectionRequestAddress")
	if udpAddr != "" {
		device.IPAddress = udpAddr
		device.UDPConnectionRequestAddress = udpAddr
		device.NatDetected = true

		// Sync to STUN address cache for UDP Connection Request
		if s.stunUpdater != nil {
			if err := s.stunUpdater.SetFromInform(ctx, device.SerialNumber, udpAddr); err != nil {
				s.logger.Warn("sync udp address to stun store",
					zap.String("serial_number", device.SerialNumber),
					zap.Error(err))
			}
		}
	}

	// Auto-transition to active when device informs (it's communicating, so it's online)
	if device.Status == model.DeviceDiscovered || device.Status == model.DeviceOffline || device.Status == model.DeviceRegistered {
		oldStatus := device.Status
		device.Status = model.DeviceActive
		s.logger.Info("UpdateFromInform: device auto-transitioned to active",
			zap.String("serial_number", device.SerialNumber),
			zap.String("previous_status", string(oldStatus)),
		)
	}

	s.logger.Debug("UpdateFromInform: calling deviceRepo.Update",
		zap.String("device_id", device.ID.String()))

	if err := s.deviceRepo.Update(ctx, device); err != nil {
		s.logger.Error("UpdateFromInform: deviceRepo.Update failed",
			zap.Error(err),
			zap.String("device_id", device.ID.String()))
		return nil, fmt.Errorf("update device: %w", err)
	}

	s.logger.Info("UpdateFromInform: device updated in DB successfully",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber))

	// Write-through: refresh cache with updated device
	s.cacheDevice(ctx, device)

	// Store parameters
	s.storeInformParameters(ctx, device.ID, inform.ParameterList)

	// Sync key parameters to device_info for fast query access
	if s.infoSyncer != nil {
		if err := s.infoSyncer.SyncFromParameters(ctx, device.ID, device.Carrier, device.Technology); err != nil {
			s.logger.Warn("sync device info from parameters",
				zap.String("device_id", device.ID.String()),
				zap.Error(err))
		}
	}

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

	// Update gauge: decrement old status, increment new status
	if s.metrics != nil {
		s.metrics.DevicesTotal.WithLabelValues(string(device.Status), string(device.Carrier)).Dec()
		s.metrics.DevicesTotal.WithLabelValues(string(newStatus), string(device.Carrier)).Inc()
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

// GetBySerialNumber retrieves a device by its serial number (Redis cache → PostgreSQL).
func (s *DeviceService) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	return s.getDeviceBySerialNumber(ctx, sn)
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
	counts, err := s.deviceRepo.CountByStatus(ctx, carrier)
	if err != nil {
		return nil, err
	}

	// Sync gauge metrics with actual DB counts
	if s.metrics != nil && carrier != nil {
		for status, count := range counts {
			s.metrics.DevicesTotal.WithLabelValues(string(status), string(*carrier)).Set(float64(count))
		}
	}

	return counts, nil
}

// ListDevicesWithInfo returns a paginated list of devices joined with extended info.
func (s *DeviceService) ListDevicesWithInfo(ctx context.Context, filter DeviceFilter) (*model.ListResponse[DeviceWithInfo], error) {
	if s.deviceInfoRepo == nil {
		// Fallback: if device_info repo not configured, use standard list
		result, err := s.deviceRepo.List(ctx, filter)
		if err != nil {
			return nil, err
		}
		items := make([]DeviceWithInfo, len(result.Items))
		for i, d := range result.Items {
			items[i] = DeviceWithInfo{Device: d}
		}
		return model.NewListResponse(items, result.Total, result.Page, result.PageSize), nil
	}
	return s.deviceInfoRepo.ListDevicesWithInfo(ctx, filter)
}

// GetDeviceInfo retrieves extended info for a device.
func (s *DeviceService) GetDeviceInfo(ctx context.Context, deviceID uuid.UUID) (*DeviceInfo, error) {
	if s.deviceInfoRepo == nil {
		return nil, nil
	}
	return s.deviceInfoRepo.GetByDeviceID(ctx, deviceID)
}

// UpdateDeviceInfo updates the manually editable device info fields.
func (s *DeviceService) UpdateDeviceInfo(ctx context.Context, deviceID uuid.UUID, req UpdateDeviceInfoRequest, updater string) error {
	if s.deviceInfoRepo == nil {
		return fmt.Errorf("device info repository not configured")
	}
	return s.deviceInfoRepo.UpdateManualFields(ctx, deviceID, req, updater)
}

// CreateDeviceInfo creates the initial device_info record for a newly registered device.
func (s *DeviceService) CreateDeviceInfo(ctx context.Context, deviceID uuid.UUID) error {
	if s.deviceInfoRepo == nil {
		return nil
	}
	now := time.Now()
	info := &DeviceInfo{
		DeviceID:        deviceID,
		FirstOnlineTime: &now,
	}
	return s.deviceInfoRepo.Create(ctx, info)
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

// PublishDeviceRegistered publishes a device.registered event for the given device.
// Called by InformHandler on BOOTSTRAP/BOOT events to trigger provisioning engine.
func (s *DeviceService) PublishDeviceRegistered(ctx context.Context, device *model.Device) {
	if s.eventBus == nil {
		return
	}
	payload := map[string]interface{}{
		"device_id":     device.ID,
		"serial_number": device.SerialNumber,
		"oui":           device.OUI,
		"product_class": device.ProductClass,
		"carrier":       string(device.Carrier),
		"technology":    string(device.Technology),
	}
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, payload)
	if err != nil {
		s.logger.Error("create device.registered event", zap.Error(err))
		return
	}
	if err := s.eventBus.Publish(ctx, event.SubjectDeviceRegistered, evt); err != nil {
		s.logger.Warn("publish device.registered event", zap.Error(err))
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

// getDeviceBySerialNumber looks up a device with Redis cache → PostgreSQL fallback.
func (s *DeviceService) getDeviceBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if s.cache != nil {
		return s.cache.GetOrLoad(ctx, sn, func(ctx context.Context, sn string) (*model.Device, error) {
			return s.deviceRepo.GetBySerialNumber(ctx, sn)
		})
	}
	return s.deviceRepo.GetBySerialNumber(ctx, sn)
}

// cacheDevice writes the device to Redis cache after DB mutations.
func (s *DeviceService) cacheDevice(ctx context.Context, device *model.Device) {
	if s.cache != nil {
		s.cache.Set(ctx, device)
	}
}

// GetDeviceDetailComposite assembles a comprehensive device detail view by querying
// device, device_info, and device_parameters (via prefix queries for MME/License/Antenna/Cells).
func (s *DeviceService) GetDeviceDetailComposite(ctx context.Context, deviceID uuid.UUID) (*DeviceDetailComposite, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	if device == nil {
		return nil, nil
	}

	result := &DeviceDetailComposite{
		Device: device,
	}

	// Get device_info
	if s.deviceInfoRepo != nil {
		info, err := s.deviceInfoRepo.GetByDeviceID(ctx, deviceID)
		if err != nil {
			s.logger.Warn("get device info for detail composite",
				zap.String("device_id", deviceID.String()),
				zap.Error(err))
		}
		result.Info = info
	}

	// MME pool — 按分组查询取代 LIKE
	mmeParams, err := s.paramRepo.GetByGroup(ctx, deviceID, "mme_pool")
	if err != nil {
		s.logger.Warn("get mme_pool params",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
	}
	result.MMEPool = AssembleMMEPool(mmeParams)

	// License
	licenseParams, err := s.paramRepo.GetByGroup(ctx, deviceID, "license")
	if err != nil {
		s.logger.Warn("get license params",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
	}
	result.License = AssembleLicenseDetail(licenseParams)

	// Antenna
	antennaParams, err := s.paramRepo.GetByGroup(ctx, deviceID, "antenna")
	if err != nil {
		s.logger.Warn("get antenna params",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
	}
	result.Antenna = AssembleAntennaInfo(antennaParams)

	// Cells — 需要跨多个 group，仍用全量查询
	numOfCells := 1
	if result.Info != nil && result.Info.NumOfCells > 0 {
		numOfCells = result.Info.NumOfCells
	}
	allParams, err := s.paramRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		s.logger.Warn("get all params for cells",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
		return result, nil
	}
	result.Cells = AssembleCells(allParams, numOfCells)

	return result, nil
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

// BatchDeleteDevices deletes multiple devices by their IDs.
// Returns a BatchOperationResult summarising successes and failures.
func (s *DeviceService) BatchDeleteDevices(ctx context.Context, ids []uuid.UUID) BatchOperationResult {
	result := BatchOperationResult{Total: len(ids)}

	deleted, err := s.deviceRepo.BatchDelete(ctx, ids)
	if err != nil {
		s.logger.Error("batch delete devices failed",
			zap.Int("total", len(ids)),
			zap.Error(err),
		)
		// All items failed
		result.Failed = len(ids)
		for _, id := range ids {
			result.Errors = append(result.Errors, BatchItemError{
				ID:      id.String(),
				Message: err.Error(),
			})
		}
		return result
	}

	result.Succeeded = int(deleted)
	result.Failed = len(ids) - int(deleted)

	s.logger.Info("batch delete devices",
		zap.Int("total", len(ids)),
		zap.Int64("deleted", deleted),
	)
	return result
}

// BatchRebootDevices queues a Reboot command for each device in the list.
// Returns a BatchOperationResult summarising successes and failures.
func (s *DeviceService) BatchRebootDevices(ctx context.Context, ids []uuid.UUID) BatchOperationResult {
	result := BatchOperationResult{Total: len(ids)}

	for _, id := range ids {
		if err := s.RebootDevice(ctx, id); err != nil {
			s.logger.Warn("batch reboot: reboot device failed",
				zap.String("device_id", id.String()),
				zap.Error(err),
			)
			result.Failed++
			result.Errors = append(result.Errors, BatchItemError{
				ID:      id.String(),
				Message: err.Error(),
			})
		} else {
			result.Succeeded++
		}
	}

	s.logger.Info("batch reboot devices",
		zap.Int("total", len(ids)),
		zap.Int("succeeded", result.Succeeded),
		zap.Int("failed", result.Failed),
	)
	return result
}

// ListGeo returns devices with geographic coordinates for map display.
func (s *DeviceService) ListGeo(ctx context.Context, filter GeoDeviceFilter) ([]GeoDevice, int64, error) {
	return s.deviceRepo.ListGeo(ctx, filter)
}

// GetGeoStats returns device statistics for map display.
func (s *DeviceService) GetGeoStats(ctx context.Context, groupIDs []string) (*GeoStats, error) {
	return s.deviceRepo.GetGeoStats(ctx, groupIDs)
}

// SearchDevices searches devices by keyword for map display.
func (s *DeviceService) SearchDevices(ctx context.Context, keyword string, limit int) ([]GeoDevice, error) {
	return s.deviceRepo.SearchDevices(ctx, keyword, limit)
}

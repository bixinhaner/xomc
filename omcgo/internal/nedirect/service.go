package nedirect

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
)

// DefaultSessionTimeoutMinutes is the default session inactivity timeout.
const DefaultSessionTimeoutMinutes = 30

// Service provides business logic for the NE Direct module.
type Service struct {
	sessionRepo   SessionRepository
	commandRepo   CommandRepository
	deviceService *device.DeviceService
	alarmEngine   *alarm.AlarmEngine
	eventBus      event.EventBus
	logger        *zap.Logger
}

// NewService creates a new NE Direct Service.
func NewService(
	sessionRepo SessionRepository,
	commandRepo CommandRepository,
	deviceService *device.DeviceService,
	alarmEngine *alarm.AlarmEngine,
	eventBus event.EventBus,
	logger *zap.Logger,
) *Service {
	return &Service{
		sessionRepo:   sessionRepo,
		commandRepo:   commandRepo,
		deviceService: deviceService,
		alarmEngine:   alarmEngine,
		eventBus:      eventBus,
		logger:        logger.Named("nedirect"),
	}
}

// ---- Session management ----

// Connect establishes a new NE direct session between a user and a device.
// If the user already has an active session to this device, it returns the existing one.
//
// 安全：会话归属（userID/username）来自调用方传入的「已认证身份」，而非请求体 ——
// handler 负责用 ctx 内 Principal 覆盖请求体里的 user_id/username，杜绝伪造身份建会话。
func (s *Service) Connect(ctx context.Context, deviceSN, userID, username string) (*Session, error) {
	// Verify device exists
	dev, err := s.deviceService.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return nil, fmt.Errorf("lookup device %q: %w", deviceSN, err)
	}
	if dev == nil {
		return nil, commonerrors.ErrNotFound
	}

	// Check for existing active session
	existing, err := s.sessionRepo.GetActiveByDeviceAndUser(ctx, deviceSN, userID)
	if err != nil {
		return nil, fmt.Errorf("check existing session: %w", err)
	}
	if existing != nil {
		// Refresh last active time
		existing.LastActiveAt = time.Now()
		if err := s.sessionRepo.Update(ctx, existing); err != nil {
			s.logger.Warn("refresh existing session failed", zap.Error(err))
		}
		return existing, nil
	}

	now := time.Now()
	session := &Session{
		DeviceID:     dev.ID,
		DeviceSN:     deviceSN,
		UserID:       userID,
		Username:     username,
		Status:       SessionActive,
		IPAddress:    dev.IPAddress,
		ConnectedAt:  now,
		LastActiveAt: now,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	s.logger.Info("ne-direct session created",
		zap.String("session_id", session.ID.String()),
		zap.String("device_sn", deviceSN),
		zap.String("username", username),
	)

	s.publishEvent(ctx, event.SubjectNEDirectConnect, map[string]string{
		"session_id": session.ID.String(),
		"device_sn":  deviceSN,
		"username":   username,
	})

	return session, nil
}

// ErrSessionOwnership 表示调用方试图操作不属于自己的会话（IDOR 越权）。
// 映射为 HTTP 403/404，避免泄露会话是否存在。
var ErrSessionOwnership = fmt.Errorf("session ownership denied: %w", commonerrors.ErrForbidden)

// authorizeSessionOwnership 校验 principal 是否有权操作指定会话（IDOR 守卫）。
//
// 规则：
//   - principal 为 nil（未注入鉴权，dev/test 退化）→ 放行；
//   - 超管 → 放行（可操作任意会话）；
//   - 否则 session.UserID 必须等于 principal.UserID，否则视为越权（ErrSessionOwnership）。
//
// 商用级要求：网元直连会话承载对基站的 CLI/MML 下发能力，必须确保操作人只能
// 操作自己建立的会话，杜绝凭 session_id 横向越权。
func authorizeSessionOwnership(principal *Principal, session *Session) error {
	if principal == nil {
		return nil
	}
	if principal.IsSuperAdmin {
		return nil
	}
	if session.UserID != principal.UserID.String() {
		return ErrSessionOwnership
	}
	return nil
}

// Disconnect closes an active NE direct session.
//
// principal 非 nil 时执行 IDOR 归属校验：非超管只能断开自己建立的会话。
func (s *Service) Disconnect(ctx context.Context, principal *Principal, sessionID uuid.UUID) error {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if err := authorizeSessionOwnership(principal, session); err != nil {
		return err
	}

	if session.Status != SessionActive {
		return nil // Already disconnected
	}

	now := time.Now()
	session.Status = SessionDisconnected
	session.DisconnectAt = &now

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	s.logger.Info("ne-direct session disconnected",
		zap.String("session_id", sessionID.String()),
		zap.String("device_sn", session.DeviceSN),
	)

	s.publishEvent(ctx, event.SubjectNEDirectDisconnect, map[string]string{
		"session_id": sessionID.String(),
		"device_sn":  session.DeviceSN,
	})

	return nil
}

// GetSession retrieves a NE direct session by ID.
func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	return s.sessionRepo.GetByID(ctx, id)
}

// ListSessions returns a paginated list of NE direct sessions.
//
// principal 非 nil 且非超管时，强制把 UserID 过滤收紧为调用方自己 —— 防止越权
// 枚举他人会话（即便请求显式传了其它 user_id，也会被覆盖）。超管 / principal=nil
// （dev/test）不收紧。
func (s *Service) ListSessions(ctx context.Context, principal *Principal, filter SessionFilter) (*model.ListResponse[Session], error) {
	if principal != nil && !principal.IsSuperAdmin {
		uid := principal.UserID.String()
		filter.UserID = &uid
	}
	return s.sessionRepo.List(ctx, filter)
}

// CloseExpiredSessions closes sessions that have been inactive beyond the timeout.
func (s *Service) CloseExpiredSessions(ctx context.Context) (int64, error) {
	count, err := s.sessionRepo.CloseExpiredSessions(ctx, DefaultSessionTimeoutMinutes)
	if err != nil {
		return 0, fmt.Errorf("close expired sessions: %w", err)
	}
	if count > 0 {
		s.logger.Info("closed expired ne-direct sessions", zap.Int64("count", count))
	}
	return count, nil
}

// ---- Command execution ----

// SendCommand sends a CLI/MML command through a NE direct session.
//
// principal 非 nil 时执行 IDOR 归属校验：非超管只能向自己建立的会话下发命令。
func (s *Service) SendCommand(ctx context.Context, principal *Principal, sessionID uuid.UUID, commandStr string) (*Command, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if err := authorizeSessionOwnership(principal, session); err != nil {
		return nil, err
	}

	if session.Status != SessionActive {
		return nil, fmt.Errorf("session is not active: %w", commonerrors.ErrInvalidInput)
	}

	cmd := &Command{
		SessionID:  sessionID,
		DeviceSN:   session.DeviceSN,
		CommandStr: commandStr,
		Status:     CommandPending,
		SentAt:     time.Now(),
	}

	if err := s.commandRepo.Create(ctx, cmd); err != nil {
		return nil, fmt.Errorf("create command: %w", err)
	}

	// Update session last active time
	session.LastActiveAt = time.Now()
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		s.logger.Warn("update session last_active_at failed", zap.Error(err))
	}

	s.logger.Info("ne-direct command sent",
		zap.String("command_id", cmd.ID.String()),
		zap.String("session_id", sessionID.String()),
		zap.String("device_sn", session.DeviceSN),
		zap.String("command", commandStr),
	)

	s.publishEvent(ctx, event.SubjectNEDirectCommand, map[string]string{
		"command_id": cmd.ID.String(),
		"session_id": sessionID.String(),
		"device_sn":  session.DeviceSN,
		"command":    commandStr,
	})

	return cmd, nil
}

// UpdateCommandResult updates the response of a previously sent command.
func (s *Service) UpdateCommandResult(ctx context.Context, commandID uuid.UUID, response, errMsg string) (*Command, error) {
	cmd, err := s.commandRepo.GetByID(ctx, commandID)
	if err != nil {
		return nil, fmt.Errorf("get command: %w", err)
	}

	now := time.Now()
	cmd.RespondAt = &now
	cmd.Response = response
	cmd.ErrorMsg = errMsg

	if errMsg != "" {
		cmd.Status = CommandFailed
	} else {
		cmd.Status = CommandCompleted
	}

	if err := s.commandRepo.Update(ctx, cmd); err != nil {
		return nil, fmt.Errorf("update command: %w", err)
	}

	return cmd, nil
}

// GetCommand retrieves a NE direct command by ID.
func (s *Service) GetCommand(ctx context.Context, id uuid.UUID) (*Command, error) {
	return s.commandRepo.GetByID(ctx, id)
}

// ListCommands returns a paginated list of NE direct commands.
func (s *Service) ListCommands(ctx context.Context, filter CommandFilter) (*model.ListResponse[Command], error) {
	return s.commandRepo.List(ctx, filter)
}

// ---- Device operations (delegated to existing services) ----

// GetDeviceStatus retrieves current device status via device service.
func (s *Service) GetDeviceStatus(ctx context.Context, serialNumber string) (*model.Device, error) {
	dev, err := s.deviceService.GetBySerialNumber(ctx, serialNumber)
	if err != nil {
		return nil, fmt.Errorf("get device status: %w", err)
	}
	if dev == nil {
		return nil, commonerrors.ErrNotFound
	}
	return dev, nil
}

// GetDeviceConfig retrieves current device parameters via device service.
func (s *Service) GetDeviceConfig(ctx context.Context, serialNumber string) ([]model.DeviceParameter, error) {
	dev, err := s.deviceService.GetBySerialNumber(ctx, serialNumber)
	if err != nil {
		return nil, fmt.Errorf("lookup device: %w", err)
	}
	if dev == nil {
		return nil, commonerrors.ErrNotFound
	}

	params, err := s.deviceService.GetDeviceParameters(ctx, dev.ID)
	if err != nil {
		return nil, fmt.Errorf("get device parameters: %w", err)
	}
	return params, nil
}

// ReportFault processes a fault report from a NE direct connection.
func (s *Service) ReportFault(ctx context.Context, report FaultReport) error {
	alarmObj := &model.Alarm{
		DeviceSN:    report.SerialNumber,
		Carrier:     model.CarrierCMCC, // NE Direct is CMCC-specific
		Severity:    model.AlarmSeverity(report.Severity),
		AlarmType:   "ne_direct",
		AlarmIdentifier:   report.AlarmCode,
		Description: report.Description,
		Status:      model.AlarmActive,
		RaisedAt:    time.Now(),
	}

	// Lookup device to get device_id
	dev, err := s.deviceService.GetBySerialNumber(ctx, report.SerialNumber)
	if err == nil && dev != nil {
		alarmObj.DeviceID = dev.ID
	}

	if err := s.alarmEngine.Process(ctx, alarmObj); err != nil {
		return fmt.Errorf("process fault: %w", err)
	}

	s.logger.Info("ne-direct fault reported",
		zap.String("serial_number", report.SerialNumber),
		zap.String("alarm_identifier", report.AlarmCode),
	)

	s.publishEvent(ctx, event.SubjectNEDirectFault, report)

	return nil
}

// RegisterDevice processes a NE direct device registration.
func (s *Service) RegisterDevice(ctx context.Context, req RegisterRequest) (*model.Device, bool, error) {
	existing, err := s.deviceService.GetBySerialNumber(ctx, req.SerialNumber)
	if err != nil {
		return nil, false, fmt.Errorf("lookup device: %w", err)
	}

	if existing != nil {
		return existing, false, nil
	}

	// Publish registration event for the provisioning pipeline
	s.publishEvent(ctx, event.SubjectNEDirectRegister, req)
	return nil, true, nil
}

// ---- internal helpers ----

func (s *Service) publishEvent(ctx context.Context, subject string, payload interface{}) {
	evt, err := event.NewEvent(subject, payload)
	if err != nil {
		s.logger.Warn("create event failed", zap.String("subject", subject), zap.Error(err))
		return
	}
	if err := s.eventBus.Publish(ctx, subject, evt); err != nil {
		s.logger.Warn("publish event failed", zap.String("subject", subject), zap.Error(err))
	}
}

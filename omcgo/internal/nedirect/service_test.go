package nedirect

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
)

// --- Mock implementations ---

type mockSessionRepo struct {
	createFn                func(ctx context.Context, session *Session) error
	getByIDFn               func(ctx context.Context, id uuid.UUID) (*Session, error)
	getActiveByDeviceUserFn func(ctx context.Context, deviceSN, userID string) (*Session, error)
	updateFn                func(ctx context.Context, session *Session) error
	listFn                  func(ctx context.Context, filter SessionFilter) (*model.ListResponse[Session], error)
	listActiveByDeviceFn    func(ctx context.Context, deviceSN string) ([]Session, error)
	closeExpiredFn          func(ctx context.Context, timeout int) (int64, error)
}

func (m *mockSessionRepo) Create(ctx context.Context, session *Session) error {
	if m.createFn != nil {
		return m.createFn(ctx, session)
	}
	session.ID = uuid.New()
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	return nil
}

func (m *mockSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}

func (m *mockSessionRepo) GetActiveByDeviceAndUser(ctx context.Context, deviceSN, userID string) (*Session, error) {
	if m.getActiveByDeviceUserFn != nil {
		return m.getActiveByDeviceUserFn(ctx, deviceSN, userID)
	}
	return nil, nil
}

func (m *mockSessionRepo) Update(ctx context.Context, session *Session) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, session)
	}
	return nil
}

func (m *mockSessionRepo) List(ctx context.Context, filter SessionFilter) (*model.ListResponse[Session], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return model.NewListResponse([]Session{}, 0, 1, 20), nil
}

func (m *mockSessionRepo) ListActiveByDevice(ctx context.Context, deviceSN string) ([]Session, error) {
	if m.listActiveByDeviceFn != nil {
		return m.listActiveByDeviceFn(ctx, deviceSN)
	}
	return []Session{}, nil
}

func (m *mockSessionRepo) CloseExpiredSessions(ctx context.Context, timeout int) (int64, error) {
	if m.closeExpiredFn != nil {
		return m.closeExpiredFn(ctx, timeout)
	}
	return 0, nil
}

type mockCommandRepo struct {
	createFn  func(ctx context.Context, cmd *Command) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*Command, error)
	updateFn  func(ctx context.Context, cmd *Command) error
	listFn    func(ctx context.Context, filter CommandFilter) (*model.ListResponse[Command], error)
}

func (m *mockCommandRepo) Create(ctx context.Context, cmd *Command) error {
	if m.createFn != nil {
		return m.createFn(ctx, cmd)
	}
	cmd.ID = uuid.New()
	cmd.CreatedAt = time.Now()
	return nil
}

func (m *mockCommandRepo) GetByID(ctx context.Context, id uuid.UUID) (*Command, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}

func (m *mockCommandRepo) Update(ctx context.Context, cmd *Command) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, cmd)
	}
	return nil
}

func (m *mockCommandRepo) List(ctx context.Context, filter CommandFilter) (*model.ListResponse[Command], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return model.NewListResponse([]Command{}, 0, 1, 20), nil
}

// Mock device-level repos needed by device.DeviceService
type mockDeviceRepo struct {
	devices map[string]*model.Device
}

func newMockDeviceRepo() *mockDeviceRepo {
	return &mockDeviceRepo{devices: make(map[string]*model.Device)}
}

func (m *mockDeviceRepo) Create(ctx context.Context, d *model.Device) error {
	m.devices[d.SerialNumber] = d
	return nil
}
func (m *mockDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	for _, d := range m.devices {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}
func (m *mockDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	d, ok := m.devices[sn]
	if !ok {
		return nil, nil
	}
	return d, nil
}
func (m *mockDeviceRepo) Update(ctx context.Context, d *model.Device) error { return nil }
func (m *mockDeviceRepo) Delete(ctx context.Context, id uuid.UUID) error    { return nil }
func (m *mockDeviceRepo) List(ctx context.Context, filter device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *mockDeviceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	return nil
}

// T-0162 新接口方法
func (m *mockDeviceRepo) UpdateLifecycle(_ context.Context, _ uuid.UUID, _ model.DeviceLifecycle) error {
	return nil
}

func (m *mockDeviceRepo) UpdateOnlineStatus(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *mockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	return nil
}
func (m *mockDeviceRepo) RecordBoot(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockDeviceRepo) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListGeo(_ context.Context, _ device.GeoDeviceFilter) ([]device.GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *mockDeviceRepo) GetGeoStats(_ context.Context, _ []string) (*device.GeoStats, error) {
	return &device.GeoStats{}, nil
}
func (m *mockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int) ([]device.GeoDevice, error) {
	return nil, nil
}
func (m *mockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) UpdateLastParamSyncAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *mockDeviceRepo) ListRecycleBin(_ context.Context, _ device.RecycleBinFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *mockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}

type mockParamRepo struct {
	params map[uuid.UUID][]model.DeviceParameter
}

func newMockParamRepo() *mockParamRepo {
	return &mockParamRepo{params: make(map[uuid.UUID][]model.DeviceParameter)}
}

func (m *mockParamRepo) BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error {
	m.params[deviceID] = params
	return nil
}
func (m *mockParamRepo) GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	return m.params[deviceID], nil
}
func (m *mockParamRepo) GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error) {
	return nil, nil
}
func (m *mockParamRepo) DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error { return nil }
func (m *mockParamRepo) GetByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (m *mockParamRepo) CountByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) (int, error) {
	return 0, nil
}
func (m *mockParamRepo) SearchByKeyword(ctx context.Context, deviceID uuid.UUID, keyword string, limit int) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (m *mockParamRepo) GetDirectChildLeaves(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]model.DeviceParameter, int, error) {
	return nil, 0, nil
}

func (m *mockParamRepo) GetByGroup(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *mockParamRepo) GetByFAPInstance(_ context.Context, _ uuid.UUID, _ int) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *mockParamRepo) GetByFAPInstanceAndGroup(_ context.Context, _ uuid.UUID, _ int, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

type mockAlarmStore struct{}

func (m *mockAlarmStore) SaveActive(ctx context.Context, a *model.Alarm) error { return nil }
func (m *mockAlarmStore) GetActiveByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) GetActiveByDeviceAndIdentifier(ctx context.Context, deviceSN, alarmIdentifier string) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) GetActiveByDeviceSN(_ context.Context, _ string) ([]*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) GetActiveByDeviceAndCode(_ context.Context, _, _ string) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) UpdateActive(ctx context.Context, a *model.Alarm) error { return nil }
func (m *mockAlarmStore) RemoveActive(ctx context.Context, id uuid.UUID) error   { return nil }
func (m *mockAlarmStore) ListActive(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return model.NewListResponse([]model.Alarm{}, 0, 1, 20), nil
}
func (m *mockAlarmStore) Archive(ctx context.Context, a *model.Alarm) error { return nil }
func (m *mockAlarmStore) ListHistory(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return model.NewListResponse([]model.Alarm{}, 0, 1, 20), nil
}
func (m *mockAlarmStore) Statistics(ctx context.Context, filter alarm.AlarmFilter) (*alarm.AlarmStatistics, error) {
	return &alarm.AlarmStatistics{}, nil
}
func (m *mockAlarmStore) BatchAcknowledge(_ context.Context, _ []uuid.UUID, _ string, _ string) error { return nil }
func (m *mockAlarmStore) BatchClear(_ context.Context, _ []uuid.UUID, _ string, _ string) error { return nil }
func (m *mockAlarmStore) HistoryStatistics(_ context.Context, _ alarm.AlarmFilter) (*alarm.AlarmStatistics, error) { return nil, nil }
func (m *mockAlarmStore) BatchUnacknowledge(_ context.Context, _ []uuid.UUID) error { return nil }
func (m *mockAlarmStore) BatchHistoryAcknowledge(_ context.Context, _ []uuid.UUID, _ string, _ string) error { return nil }
func (m *mockAlarmStore) BatchHistoryUnacknowledge(_ context.Context, _ []uuid.UUID) error { return nil }
func (m *mockAlarmStore) BatchHistoryDelete(_ context.Context, _ []uuid.UUID) error { return nil }
func (m *mockAlarmStore) MarkRead(_ context.Context, _ uuid.UUID) error { return nil }

type mockEventBus struct {
	published []event.Event
}

func (m *mockEventBus) Publish(ctx context.Context, subject string, e event.Event) error {
	m.published = append(m.published, e)
	return nil
}
func (m *mockEventBus) Subscribe(subject string, handler event.EventHandler) (event.Subscription, error) {
	return &mockSubscription{}, nil
}
func (m *mockEventBus) QueueSubscribe(subject, queue string, handler event.EventHandler) (event.Subscription, error) {
	return &mockSubscription{}, nil
}
func (m *mockEventBus) Close() error { return nil }

type mockSubscription struct{}

func (m *mockSubscription) Unsubscribe() error { return nil }

// --- Test helpers ---

func newTestService(
	sessionRepo *mockSessionRepo,
	commandRepo *mockCommandRepo,
	existingDevices map[string]*model.Device,
) (*Service, *mockEventBus) {
	deviceRepo := newMockDeviceRepo()
	if existingDevices != nil {
		deviceRepo.devices = existingDevices
	}
	paramRepo := newMockParamRepo()
	alarmStore := &mockAlarmStore{}
	alarmEngine := alarm.NewAlarmEngine(alarmStore, nil, nil, nil, zap.NewNop())
	eventBus := &mockEventBus{}
	deviceService := device.NewDeviceService(deviceRepo, paramRepo, nil, nil, zap.NewNop())

	svc := NewService(sessionRepo, commandRepo, deviceService, alarmEngine, eventBus, zap.NewNop())
	return svc, eventBus
}

// --- Tests: Connect ---

func TestService_Connect_NewSession(t *testing.T) {
	deviceID := uuid.New()
	devices := map[string]*model.Device{
		"SN-001": {
			ID:           deviceID,
			SerialNumber: "SN-001",
			Status:       model.DeviceActive,
			IPAddress:    "10.0.0.1",
		},
	}

	sessionRepo := &mockSessionRepo{}
	svc, eventBus := newTestService(sessionRepo, &mockCommandRepo{}, devices)

	session, err := svc.Connect(context.Background(), "SN-001", "user1", "admin")

	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, "SN-001", session.DeviceSN)
	assert.Equal(t, "user1", session.UserID)
	assert.Equal(t, "admin", session.Username)
	assert.Equal(t, SessionActive, session.Status)
	assert.Equal(t, deviceID, session.DeviceID)

	// Connect event published
	require.Len(t, eventBus.published, 1)
	assert.Equal(t, event.SubjectNEDirectConnect, eventBus.published[0].Subject)
}

func TestService_Connect_ExistingSession(t *testing.T) {
	deviceID := uuid.New()
	sessionID := uuid.New()
	devices := map[string]*model.Device{
		"SN-001": {
			ID:           deviceID,
			SerialNumber: "SN-001",
			Status:       model.DeviceActive,
			IPAddress:    "10.0.0.1",
		},
	}

	existingSession := &Session{
		ID:           sessionID,
		DeviceID:     deviceID,
		DeviceSN:     "SN-001",
		UserID:       "user1",
		Username:     "admin",
		Status:       SessionActive,
		ConnectedAt:  time.Now().Add(-10 * time.Minute),
		LastActiveAt: time.Now().Add(-5 * time.Minute),
	}

	sessionRepo := &mockSessionRepo{
		getActiveByDeviceUserFn: func(ctx context.Context, deviceSN, userID string) (*Session, error) {
			if deviceSN == "SN-001" && userID == "user1" {
				return existingSession, nil
			}
			return nil, nil
		},
	}

	svc, eventBus := newTestService(sessionRepo, &mockCommandRepo{}, devices)

	session, err := svc.Connect(context.Background(), "SN-001", "user1", "admin")

	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, sessionID, session.ID)
	// No connect event for existing session
	assert.Empty(t, eventBus.published)
}

func TestService_Connect_DeviceNotFound(t *testing.T) {
	svc, _ := newTestService(&mockSessionRepo{}, &mockCommandRepo{}, nil)

	session, err := svc.Connect(context.Background(), "NONEXISTENT", "user1", "admin")

	require.Error(t, err)
	assert.Nil(t, session)
}

// --- Tests: Disconnect ---

func TestService_Disconnect_Success(t *testing.T) {
	sessionID := uuid.New()
	activeSession := &Session{
		ID:       sessionID,
		DeviceSN: "SN-001",
		UserID:   "user1",
		Status:   SessionActive,
	}

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Session, error) {
			if id == sessionID {
				return activeSession, nil
			}
			return nil, commonerrors.ErrNotFound
		},
	}

	svc, eventBus := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	err := svc.Disconnect(context.Background(), sessionID)

	require.NoError(t, err)
	assert.Equal(t, SessionDisconnected, activeSession.Status)
	require.NotNil(t, activeSession.DisconnectAt)

	// Disconnect event published
	require.Len(t, eventBus.published, 1)
	assert.Equal(t, event.SubjectNEDirectDisconnect, eventBus.published[0].Subject)
}

func TestService_Disconnect_AlreadyDisconnected(t *testing.T) {
	sessionID := uuid.New()
	session := &Session{
		ID:     sessionID,
		Status: SessionDisconnected,
	}

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Session, error) {
			return session, nil
		},
	}

	svc, eventBus := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	err := svc.Disconnect(context.Background(), sessionID)

	require.NoError(t, err)
	assert.Empty(t, eventBus.published) // No event for already disconnected
}

// --- Tests: SendCommand ---

func TestService_SendCommand_Success(t *testing.T) {
	sessionID := uuid.New()
	session := &Session{
		ID:       sessionID,
		DeviceSN: "SN-001",
		Status:   SessionActive,
	}

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Session, error) {
			if id == sessionID {
				return session, nil
			}
			return nil, commonerrors.ErrNotFound
		},
	}

	svc, eventBus := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	cmd, err := svc.SendCommand(context.Background(), sessionID, "DSP CELLINFO;")

	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, sessionID, cmd.SessionID)
	assert.Equal(t, "SN-001", cmd.DeviceSN)
	assert.Equal(t, "DSP CELLINFO;", cmd.CommandStr)
	assert.Equal(t, CommandPending, cmd.Status)

	// Command event published
	require.Len(t, eventBus.published, 1)
	assert.Equal(t, event.SubjectNEDirectCommand, eventBus.published[0].Subject)
}

func TestService_SendCommand_InactiveSession(t *testing.T) {
	sessionID := uuid.New()
	session := &Session{
		ID:     sessionID,
		Status: SessionDisconnected,
	}

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Session, error) {
			return session, nil
		},
	}

	svc, _ := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	cmd, err := svc.SendCommand(context.Background(), sessionID, "DSP CELLINFO;")

	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

// --- Tests: UpdateCommandResult ---

func TestService_UpdateCommandResult_Success(t *testing.T) {
	cmdID := uuid.New()
	existingCmd := &Command{
		ID:     cmdID,
		Status: CommandPending,
	}

	commandRepo := &mockCommandRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Command, error) {
			if id == cmdID {
				return existingCmd, nil
			}
			return nil, commonerrors.ErrNotFound
		},
	}

	svc, _ := newTestService(&mockSessionRepo{}, commandRepo, nil)

	result, err := svc.UpdateCommandResult(context.Background(), cmdID, "OK\nCell Info: ...", "")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, CommandCompleted, result.Status)
	assert.Equal(t, "OK\nCell Info: ...", result.Response)
	assert.NotNil(t, result.RespondAt)
}

func TestService_UpdateCommandResult_WithError(t *testing.T) {
	cmdID := uuid.New()
	existingCmd := &Command{
		ID:     cmdID,
		Status: CommandPending,
	}

	commandRepo := &mockCommandRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Command, error) {
			return existingCmd, nil
		},
	}

	svc, _ := newTestService(&mockSessionRepo{}, commandRepo, nil)

	result, err := svc.UpdateCommandResult(context.Background(), cmdID, "", "command not supported")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, CommandFailed, result.Status)
	assert.Equal(t, "command not supported", result.ErrorMsg)
}

// --- Tests: RegisterDevice ---

func TestService_RegisterDevice_ExistingDevice(t *testing.T) {
	deviceID := uuid.New()
	devices := map[string]*model.Device{
		"SN-EXISTS": {
			ID:           deviceID,
			SerialNumber: "SN-EXISTS",
			Status:       model.DeviceActive,
		},
	}

	svc, eventBus := newTestService(&mockSessionRepo{}, &mockCommandRepo{}, devices)

	dev, isNew, err := svc.RegisterDevice(context.Background(), RegisterRequest{SerialNumber: "SN-EXISTS"})

	require.NoError(t, err)
	assert.False(t, isNew)
	require.NotNil(t, dev)
	assert.Equal(t, deviceID, dev.ID)
	assert.Empty(t, eventBus.published) // No event for existing device
}

func TestService_RegisterDevice_NewDevice(t *testing.T) {
	svc, eventBus := newTestService(&mockSessionRepo{}, &mockCommandRepo{}, nil)

	dev, isNew, err := svc.RegisterDevice(context.Background(), RegisterRequest{SerialNumber: "SN-NEW"})

	require.NoError(t, err)
	assert.True(t, isNew)
	assert.Nil(t, dev)
	require.Len(t, eventBus.published, 1)
	assert.Equal(t, event.SubjectNEDirectRegister, eventBus.published[0].Subject)
}

// --- Tests: ReportFault ---

func TestService_ReportFault_Success(t *testing.T) {
	deviceID := uuid.New()
	devices := map[string]*model.Device{
		"SN-FAULT": {
			ID:           deviceID,
			SerialNumber: "SN-FAULT",
			Status:       model.DeviceActive,
		},
	}

	svc, eventBus := newTestService(&mockSessionRepo{}, &mockCommandRepo{}, devices)

	err := svc.ReportFault(context.Background(), FaultReport{
		SerialNumber: "SN-FAULT",
		AlarmCode:    "TEMP_HIGH",
		Severity:     1,
		Description:  "Temperature exceeded",
	})

	require.NoError(t, err)
	require.Len(t, eventBus.published, 1)
	assert.Equal(t, event.SubjectNEDirectFault, eventBus.published[0].Subject)
}

// --- Tests: GetDeviceStatus ---

func TestService_GetDeviceStatus_Found(t *testing.T) {
	deviceID := uuid.New()
	devices := map[string]*model.Device{
		"SN-001": {
			ID:           deviceID,
			SerialNumber: "SN-001",
			Status:       model.DeviceActive,
			IPAddress:    "10.0.0.1",
		},
	}

	svc, _ := newTestService(&mockSessionRepo{}, &mockCommandRepo{}, devices)

	dev, err := svc.GetDeviceStatus(context.Background(), "SN-001")

	require.NoError(t, err)
	require.NotNil(t, dev)
	assert.Equal(t, "SN-001", dev.SerialNumber)
}

func TestService_GetDeviceStatus_NotFound(t *testing.T) {
	svc, _ := newTestService(&mockSessionRepo{}, &mockCommandRepo{}, nil)

	dev, err := svc.GetDeviceStatus(context.Background(), "NONEXISTENT")

	require.Error(t, err)
	assert.Nil(t, dev)
}

// --- Tests: CloseExpiredSessions ---

func TestService_CloseExpiredSessions(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		closeExpiredFn: func(ctx context.Context, timeout int) (int64, error) {
			assert.Equal(t, DefaultSessionTimeoutMinutes, timeout)
			return 3, nil
		},
	}

	svc, _ := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	count, err := svc.CloseExpiredSessions(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

// --- Tests: ListSessions ---

func TestService_ListSessions(t *testing.T) {
	expected := model.NewListResponse([]Session{
		{ID: uuid.New(), DeviceSN: "SN-001", Status: SessionActive},
	}, 1, 1, 20)

	sessionRepo := &mockSessionRepo{
		listFn: func(ctx context.Context, filter SessionFilter) (*model.ListResponse[Session], error) {
			return expected, nil
		},
	}

	svc, _ := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	result, err := svc.ListSessions(context.Background(), SessionFilter{})

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// --- Tests: ListCommands ---

func TestService_ListCommands(t *testing.T) {
	expected := model.NewListResponse([]Command{
		{ID: uuid.New(), CommandStr: "DSP CELLINFO;", Status: CommandCompleted},
	}, 1, 1, 20)

	commandRepo := &mockCommandRepo{
		listFn: func(ctx context.Context, filter CommandFilter) (*model.ListResponse[Command], error) {
			return expected, nil
		},
	}

	svc, _ := newTestService(&mockSessionRepo{}, commandRepo, nil)

	result, err := svc.ListCommands(context.Background(), CommandFilter{})

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

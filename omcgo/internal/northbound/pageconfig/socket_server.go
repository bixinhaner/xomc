package pageconfig

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

const (
	socketDefaultReloadInterval = 30 * time.Second
	socketMaxFrameBody          = 1 << 20
	socketWriteTimeout          = 5 * time.Second

	ctccFrameStart uint16 = 0x7ee7
	cuccFrameStart uint16 = 0xffff

	socketMessageFormatString = 1
	socketMessageFormatJSON   = 2

	ctccMsgLogin             = 1
	ctccMsgLoginResult       = 2
	ctccMsgHeartbeat         = 3
	ctccMsgHeartbeatResult   = 4
	ctccMsgSyncAlarm         = 5
	ctccMsgSyncAlarmResult   = 6
	ctccMsgRealtimeAlarm     = 10
	ctccMsgSyncStart         = 11
	ctccMsgSyncEnd           = 12
	ctccMsgDisconnect        = 13
	cuccMsgRealtimeAlarm     = 0
	cuccMsgLogin             = 1
	cuccMsgLoginAck          = 2
	cuccMsgSyncAlarm         = 3
	cuccMsgSyncAlarmAck      = 4
	cuccMsgSyncAlarmFile     = 5
	cuccMsgSyncAlarmFileAck  = 6
	cuccMsgSyncAlarmFileDone = 7
	cuccMsgHeartbeat         = 8
	cuccMsgHeartbeatAck      = 9
	cuccMsgClose             = 10
)

type socketProtocolFrame struct {
	MessageType   int
	MessageFormat int
	Body          []byte
}

type SocketAlarmServerManager struct {
	svc            *Service
	bus            event.EventBus
	logger         *zap.Logger
	reloadInterval time.Duration

	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	servers   map[string]*socketAlarmServer
	sessions  map[*socketAlarmSession]struct{}
	subs      []event.Subscription
	wg        sync.WaitGroup
	reloadNow chan struct{}
}

type socketAlarmServer struct {
	mu       sync.RWMutex
	config   SocketAlarmConfig
	listener net.Listener
}

type socketAlarmSession struct {
	manager   *SocketAlarmServerManager
	config    SocketAlarmConfig
	conn      net.Conn
	remote    string
	sessionID string

	authMu        sync.RWMutex
	authenticated bool
	account       SocketAccount

	writeMu   sync.Mutex
	syncMu    sync.Mutex
	syncing   bool
	syncQueue []socketProtocolFrame
	closeOnce sync.Once
	closed    chan struct{}
}

type socketBroadcastBucket struct {
	config   SocketAlarmConfig
	sessions []*socketAlarmSession
}

func NewSocketAlarmServerManager(svc *Service, bus event.EventBus, logger *zap.Logger) *SocketAlarmServerManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SocketAlarmServerManager{
		svc:            svc,
		bus:            bus,
		logger:         logger.Named("page-config-socket-server"),
		reloadInterval: socketDefaultReloadInterval,
		servers:        map[string]*socketAlarmServer{},
		sessions:       map[*socketAlarmSession]struct{}{},
		reloadNow:      make(chan struct{}, 1),
	}
}

func (m *SocketAlarmServerManager) SetReloadInterval(interval time.Duration) {
	if interval <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadInterval = interval
}

func (m *SocketAlarmServerManager) RequestReload() {
	if m == nil {
		return
	}
	m.mu.RLock()
	running := m.ctx != nil && m.cancel != nil
	reloadNow := m.reloadNow
	m.mu.RUnlock()
	if !running || reloadNow == nil {
		return
	}
	select {
	case reloadNow <- struct{}{}:
	default:
	}
}

func (m *SocketAlarmServerManager) Start(ctx context.Context) error {
	if m == nil || m.svc == nil {
		return nil
	}
	m.mu.Lock()
	if m.cancel != nil {
		m.mu.Unlock()
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if m.reloadNow == nil {
		m.reloadNow = make(chan struct{}, 1)
	}
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	if m.bus != nil {
		for _, subject := range []string{event.SubjectAlarmRaised, event.SubjectAlarmCleared} {
			subject := subject
			sub, err := m.bus.Subscribe(subject, func(ctx context.Context, evt event.Event) error {
				return m.handleAlarmEvent(ctx, subject, evt)
			})
			if err != nil {
				_ = m.Stop()
				return fmt.Errorf("subscribe socket alarm server %s: %w", subject, err)
			}
			m.mu.Lock()
			m.subs = append(m.subs, sub)
			m.mu.Unlock()
		}
	}

	if err := m.reload(m.ctx); err != nil {
		m.logger.Warn("initial northbound socket server reload failed", zap.Error(err))
	}
	m.wg.Add(1)
	go m.reloadLoop()
	return nil
}

func (m *SocketAlarmServerManager) Stop() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	subs := append([]event.Subscription(nil), m.subs...)
	m.subs = nil
	servers := make(map[string]*socketAlarmServer, len(m.servers))
	for key, server := range m.servers {
		servers[key] = server
		delete(m.servers, key)
	}
	sessions := make([]*socketAlarmSession, 0, len(m.sessions))
	for session := range m.sessions {
		sessions = append(sessions, session)
		delete(m.sessions, session)
	}
	m.mu.Unlock()

	var firstErr error
	for _, sub := range subs {
		if err := sub.Unsubscribe(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, server := range servers {
		_ = server.listener.Close()
	}
	for _, session := range sessions {
		session.close()
	}
	m.wg.Wait()
	return firstErr
}

func (m *SocketAlarmServerManager) ListenerAddr(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	server, ok := m.servers[key]
	if !ok || server.listener == nil {
		return "", false
	}
	return server.listener.Addr().String(), true
}

func (m *SocketAlarmServerManager) HandleAlarmEvent(ctx context.Context, subject string, evt event.Event) error {
	return m.handleAlarmEvent(ctx, subject, evt)
}

func (m *SocketAlarmServerManager) reloadLoop() {
	defer m.wg.Done()
	for {
		m.mu.RLock()
		ctx := m.ctx
		interval := m.reloadInterval
		reloadNow := m.reloadNow
		m.mu.RUnlock()
		if ctx == nil {
			return
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-reloadNow:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			if err := m.reload(ctx); err != nil {
				m.logger.Warn("northbound socket server reload failed", zap.Error(err))
			}
		case <-timer.C:
			if err := m.reload(ctx); err != nil {
				m.logger.Warn("northbound socket server reload failed", zap.Error(err))
			}
		}
	}
}

func (m *SocketAlarmServerManager) reload(ctx context.Context) error {
	configs, err := m.svc.ListActiveSocketAlarmConfigsForServe(ctx)
	if err != nil {
		return err
	}
	wanted := make(map[string]SocketAlarmConfig, len(configs))
	for _, config := range configs {
		config = normalizeSocketAlarmConfig(config.Key, config)
		if !config.Enabled || config.Mode != "server" {
			continue
		}
		wanted[config.Key] = config
	}

	var lifecycleEvents []PageConfigEvent
	var firstErr error
	m.mu.Lock()
	for key, server := range m.servers {
		config, ok := wanted[key]
		if !ok || socketServerListenChanged(server.currentConfig(), config) {
			stopped := server.currentConfig()
			m.stopServerLocked(key)
			lifecycleEvents = append(lifecycleEvents, eventFromSocketRuntime(stopped, "server_stopped", RunStatusTerminated, "socket listener stopped", "", map[string]any{
				"reason": "disabled_or_changed",
			}))
			continue
		}
		current := server.currentConfig()
		server.setConfig(config)
		if socketServerSessionPolicyChanged(current, config) {
			closed := m.closeSessionsLocked(key)
			if closed > 0 {
				lifecycleEvents = append(lifecycleEvents, eventFromSocketRuntime(config, "server_sessions_reloaded", RunStatusTerminated, "socket sessions closed for config reload", "", map[string]any{
					"closed_sessions": closed,
					"reason":          "policy_changed",
				}))
			}
		}
		delete(wanted, key)
	}
	for _, config := range wanted {
		server, err := m.startServerLocked(ctx, config)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			lifecycleEvents = append(lifecycleEvents, eventFromSocketRuntime(config, "server_start_failed", RunStatusFailed, err.Error(), err.Error(), map[string]any{
				"enabled_account_count": enabledSocketAccountCount(config.Accounts),
			}))
			continue
		}
		started := config
		if tcpAddr, ok := server.listener.Addr().(*net.TCPAddr); ok {
			started.ListenPort = tcpAddr.Port
		}
		lifecycleEvents = append(lifecycleEvents, eventFromSocketRuntime(started, "server_started", RunStatusSuccess, "socket listener started", "", map[string]any{
			"enabled_account_count": enabledSocketAccountCount(config.Accounts),
			"listen_addr":           server.listener.Addr().String(),
		}))
	}
	m.mu.Unlock()

	for _, evt := range lifecycleEvents {
		m.recordEvent(evt)
	}
	return firstErr
}

func (m *SocketAlarmServerManager) startServerLocked(ctx context.Context, config SocketAlarmConfig) (*socketAlarmServer, error) {
	listenIP := firstNonEmpty(config.ListenIP, "0.0.0.0")
	addr := net.JoinHostPort(listenIP, strconv.Itoa(config.ListenPort))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s socket alarm server %s: %w", config.Profile, addr, err)
	}
	server := &socketAlarmServer{config: config, listener: listener}
	m.servers[config.Key] = server
	m.wg.Add(1)
	go m.acceptLoop(ctx, server)
	m.logger.Info("northbound socket alarm server started",
		zap.String("key", config.Key),
		zap.String("profile", config.Profile),
		zap.String("addr", listener.Addr().String()))
	return server, nil
}

func (m *SocketAlarmServerManager) stopServerLocked(key string) {
	server, ok := m.servers[key]
	if !ok {
		return
	}
	delete(m.servers, key)
	_ = server.listener.Close()
	m.closeSessionsLocked(key)
}

func (m *SocketAlarmServerManager) closeSessionsLocked(key string) int {
	closed := 0
	for session := range m.sessions {
		if session.config.Key == key {
			session.close()
			delete(m.sessions, session)
			closed++
		}
	}
	return closed
}

func socketServerListenChanged(current, next SocketAlarmConfig) bool {
	return !strings.EqualFold(current.Profile, next.Profile) ||
		!strings.EqualFold(firstNonEmpty(current.ListenIP, "0.0.0.0"), firstNonEmpty(next.ListenIP, "0.0.0.0")) ||
		current.ListenPort != next.ListenPort
}

func socketServerSessionPolicyChanged(current, next SocketAlarmConfig) bool {
	return current.MaxClients != next.MaxClients ||
		current.RealtimePushEnabled != next.RealtimePushEnabled ||
		current.ClientSyncEnabled != next.ClientSyncEnabled ||
		current.HeartbeatSeconds != next.HeartbeatSeconds ||
		current.HeartbeatTimes != next.HeartbeatTimes ||
		current.IdleTimeoutSeconds != next.IdleTimeoutSeconds ||
		!reflect.DeepEqual(current.Accounts, next.Accounts)
}

func (s *socketAlarmServer) currentConfig() SocketAlarmConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *socketAlarmServer) setConfig(config SocketAlarmConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

func (m *SocketAlarmServerManager) acceptLoop(ctx context.Context, server *socketAlarmServer) {
	defer m.wg.Done()
	for {
		conn, err := server.listener.Accept()
		if err != nil {
			if ctx != nil && ctx.Err() != nil {
				return
			}
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			m.logger.Warn("accept northbound socket alarm connection failed", zap.Error(err))
			continue
		}
		config := server.currentConfig()
		maxClients := config.MaxClients
		if maxClients <= 0 {
			maxClients = 20
		}
		if m.sessionCountForConfig(config.Key) >= maxClients {
			m.recordEvent(eventFromSocketRuntime(config, "connection_rejected", RunStatusFailed, "max clients reached", "max clients reached", map[string]any{
				"remote_addr": conn.RemoteAddr().String(),
				"max_clients": maxClients,
			}))
			_ = conn.Close()
			continue
		}
		session := &socketAlarmSession{
			manager:   m,
			config:    config,
			conn:      conn,
			remote:    conn.RemoteAddr().String(),
			sessionID: fmt.Sprintf("%s-%d", config.Key, time.Now().UnixNano()),
			closed:    make(chan struct{}),
		}
		m.registerSession(session)
		m.wg.Add(1)
		go session.serve(ctx)
	}
}

func (m *SocketAlarmServerManager) sessionCountForConfig(key string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for session := range m.sessions {
		if session.config.Key == key {
			count++
		}
	}
	return count
}

func (m *SocketAlarmServerManager) registerSession(session *socketAlarmSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session] = struct{}{}
}

func (m *SocketAlarmServerManager) unregisterSession(session *socketAlarmSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, session)
}

func (s *socketAlarmSession) serve(ctx context.Context) {
	defer s.manager.wg.Done()
	defer s.manager.unregisterSession(s)
	defer s.close()
	for {
		timeout := time.Duration(s.config.IdleTimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 180 * time.Second
		}
		_ = s.conn.SetReadDeadline(time.Now().Add(timeout))
		frame, err := readSocketFrame(s.conn, s.config.Profile)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) && !strings.Contains(err.Error(), "use of closed network connection") {
				s.manager.logger.Debug("northbound socket alarm session closed",
					zap.String("config_key", s.config.Key),
					zap.String("remote", s.remote),
					zap.Error(err))
			}
			return
		}
		if err := s.handleFrame(ctx, frame); err != nil {
			if !errors.Is(err, io.EOF) {
				s.manager.logger.Debug("northbound socket alarm frame handling failed",
					zap.String("config_key", s.config.Key),
					zap.String("remote", s.remote),
					zap.Error(err))
			}
			return
		}
	}
}

func (s *socketAlarmSession) close() {
	s.closeOnce.Do(func() {
		close(s.closed)
		_ = s.conn.Close()
	})
}

func (s *socketAlarmSession) writeFrame(messageType int, messageFormat int, body []byte) error {
	data, err := encodeSocketFrame(s.config.Profile, messageType, messageFormat, body)
	if err != nil {
		return err
	}
	select {
	case <-s.closed:
		return net.ErrClosed
	default:
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(socketWriteTimeout))
	if _, err := s.conn.Write(data); err != nil {
		s.close()
		return err
	}
	return nil
}

func (s *socketAlarmSession) beginSync() bool {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	if s.syncing {
		return false
	}
	s.syncing = true
	s.syncQueue = nil
	return true
}

func (s *socketAlarmSession) finishSync() []socketProtocolFrame {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	s.syncing = false
	pending := append([]socketProtocolFrame(nil), s.syncQueue...)
	s.syncQueue = nil
	return pending
}

func (s *socketAlarmSession) queueOrWriteRealtimeFrame(frame socketProtocolFrame) error {
	s.syncMu.Lock()
	if s.syncing {
		copied := socketProtocolFrame{
			MessageType:   frame.MessageType,
			MessageFormat: frame.MessageFormat,
			Body:          append([]byte(nil), frame.Body...),
		}
		s.syncQueue = append(s.syncQueue, copied)
		s.syncMu.Unlock()
		return nil
	}
	s.syncMu.Unlock()
	return s.writeFrame(frame.MessageType, frame.MessageFormat, frame.Body)
}

func (s *socketAlarmSession) flushQueuedRealtimeFrames(frames []socketProtocolFrame) error {
	for _, frame := range frames {
		if err := s.writeFrame(frame.MessageType, frame.MessageFormat, frame.Body); err != nil {
			return err
		}
	}
	return nil
}

func (s *socketAlarmSession) handleFrame(ctx context.Context, frame socketProtocolFrame) error {
	switch strings.ToUpper(s.config.Profile) {
	case "CUCC":
		return s.handleCUCCFrame(ctx, frame)
	default:
		return s.handleCTCCFrame(ctx, frame)
	}
}

func (s *socketAlarmSession) handleCTCCFrame(ctx context.Context, frame socketProtocolFrame) error {
	switch frame.MessageType {
	case ctccMsgLogin:
		var req struct {
			User     string `json:"user"`
			Passwd   string `json:"passwd"`
			Password string `json:"password"`
			Key      string `json:"key"`
		}
		_ = json.Unmarshal(frame.Body, &req)
		account, ok := authenticateSocketAccount(s.config, req.User, firstNonEmpty(req.Passwd, req.Password, req.Key), "msg")
		resp := map[string]any{
			"is_success":        ok,
			"server_time":       time.Now().Format("20060102150405"),
			"heart_beat_period": s.config.HeartbeatSeconds,
			"info":              "success",
		}
		status := RunStatusSuccess
		errMessage := ""
		if !ok {
			resp["info"] = "invalid user or password"
			status = RunStatusFailed
			errMessage = "invalid CTCC socket account"
		} else {
			s.markAuthenticated(account)
		}
		payload := mustMarshalEventPayload(resp)
		s.manager.recordEvent(eventFromSocketRuntime(s.config, "login", status, payload, errMessage, map[string]any{
			"remote_addr":  s.remote,
			"account_key":  account.Key,
			"account_type": "msg",
			"session_id":   s.sessionID,
		}))
		if err := s.writeCTCCJSON(ctccMsgLoginResult, resp); err != nil {
			return err
		}
		if !ok {
			return io.EOF
		}
		return nil
	case ctccMsgHeartbeat:
		resp := map[string]any{
			"server_time": time.Now().Format("20060102150405"),
			"result":      0,
		}
		s.manager.recordEvent(eventFromSocketRuntime(s.config, "heartbeat", RunStatusSuccess, mustMarshalEventPayload(resp), "", map[string]any{
			"remote_addr": s.remote,
			"session_id":  s.sessionID,
		}))
		return s.writeCTCCJSON(ctccMsgHeartbeatResult, resp)
	case ctccMsgSyncAlarm:
		req := socketAlarmSyncRequest{
			ReqID:    jsonTextValue(frame.Body, "reqId", "req_id"),
			AlarmSeq: jsonTextValue(frame.Body, "alarmSeq", "alarm_seq"),
		}
		reqID := req.ReqID
		alarmSeq := req.AlarmSeq
		if reqID == "" {
			reqID = fmt.Sprintf("CTCC-%d", time.Now().UnixNano())
			req.ReqID = reqID
		}
		if !s.isAuthenticatedType("msg") || !s.config.ClientSyncEnabled {
			resp := map[string]any{"reqId": reqID, "result": 1, "count": 0, "info": "client sync is disabled or session is not authenticated"}
			s.manager.recordEvent(eventFromSocketRuntime(s.config, "sync_request", RunStatusFailed, mustMarshalEventPayload(resp), "client sync rejected", map[string]any{
				"remote_addr": s.remote,
				"request_id":  reqID,
				"alarm_seq":   alarmSeq,
			}))
			return s.writeCTCCJSON(ctccMsgSyncAlarmResult, resp)
		}
		if msg := validateSocketMessageSyncRequest(req, false); msg != "" {
			resp := map[string]any{"reqId": reqID, "result": 1, "count": 0, "alarmSeq": alarmSeq, "info": msg}
			s.manager.recordEvent(eventFromSocketRuntime(s.config, "sync_request", RunStatusFailed, mustMarshalEventPayload(resp), msg, map[string]any{
				"remote_addr": s.remote,
				"request_id":  reqID,
				"alarm_seq":   alarmSeq,
			}))
			return s.writeCTCCJSON(ctccMsgSyncAlarmResult, resp)
		}
		if !s.beginSync() {
			resp := map[string]any{"reqId": reqID, "result": 1, "count": 0, "alarmSeq": alarmSeq, "info": "Synchronization is being performed"}
			s.manager.recordEvent(eventFromSocketRuntime(s.config, "sync_request", RunStatusFailed, mustMarshalEventPayload(resp), "Synchronization is being performed", map[string]any{
				"remote_addr": s.remote,
				"request_id":  reqID,
				"alarm_seq":   alarmSeq,
			}))
			return s.writeCTCCJSON(ctccMsgSyncAlarmResult, resp)
		}
		replayItems, replayErr := s.loadSocketSyncItems(ctx, req, socketAlarmMessageLimit)
		replayEvents := []PageConfigEvent(nil)
		if replayErr != nil && strings.Contains(replayErr.Error(), "alarm store is not configured") {
			replayEvents, replayErr = s.loadSocketReplayEvents(ctx, 50)
		}
		replayCount := len(replayItems) + len(replayEvents)
		info := "historical alarm replay uses alarm store"
		if len(replayEvents) > 0 {
			info = "historical alarm replay uses page-config socket alarm_push events"
		}
		if replayErr != nil {
			replayCount = 0
			info = replayErr.Error()
		}
		start := map[string]any{"reqId": reqID, "alarmSeq": alarmSeq, "server_time": time.Now().Format("20060102150405")}
		resultStatus := 0
		if replayErr != nil {
			resultStatus = 1
		}
		result := map[string]any{"reqId": reqID, "result": resultStatus, "count": replayCount, "alarmSeq": alarmSeq, "info": info}
		end := map[string]any{"reqId": reqID, "count": replayCount}
		if err := s.writeCTCCJSON(ctccMsgSyncStart, start); err != nil {
			_ = s.finishSync()
			return err
		}
		if replayErr == nil {
			for _, item := range replayItems {
				frame := socketCTCCSyncFrameFromItem(item)
				if err := s.writeFrame(ctccMsgSyncAlarmResult, socketMessageFormatJSON, frame.Body); err != nil {
					_ = s.finishSync()
					return err
				}
			}
			for _, event := range replayEvents {
				if strings.TrimSpace(event.Payload) == "" {
					continue
				}
				if err := s.writeFrame(ctccMsgSyncAlarmResult, socketMessageFormatJSON, []byte(event.Payload)); err != nil {
					_ = s.finishSync()
					return err
				}
			}
		}
		if err := s.writeCTCCJSON(ctccMsgSyncAlarmResult, result); err != nil {
			_ = s.finishSync()
			return err
		}
		if err := s.writeCTCCJSON(ctccMsgSyncEnd, end); err != nil {
			_ = s.finishSync()
			return err
		}
		pending := s.finishSync()
		if err := s.flushQueuedRealtimeFrames(pending); err != nil {
			return err
		}
		status := RunStatusSuccess
		errMessage := ""
		if replayErr != nil {
			status = RunStatusFailed
			errMessage = replayErr.Error()
		}
		s.manager.recordEvent(eventFromSocketRuntime(s.config, "sync_request", status, mustMarshalEventPayload(result), errMessage, map[string]any{
			"remote_addr":                s.remote,
			"request_id":                 reqID,
			"alarm_seq":                  alarmSeq,
			"replay_count":               replayCount,
			"replay_source":              info,
			"queued_realtime_after_sync": len(pending),
		}))
		return nil
	case ctccMsgDisconnect:
		return io.EOF
	default:
		return fmt.Errorf("unsupported CTCC socket message type %d", frame.MessageType)
	}
}

func (s *socketAlarmSession) handleCUCCFrame(ctx context.Context, frame socketProtocolFrame) error {
	command, fields := parseCUCCCommand(string(frame.Body))
	switch frame.MessageType {
	case cuccMsgLogin:
		accountType := firstNonEmpty(fields["type"], "msg")
		account, ok := authenticateSocketAccount(s.config, fields["user"], firstNonEmpty(fields["key"], fields["passwd"], fields["password"]), accountType)
		result := "succ"
		desc := "success"
		status := RunStatusSuccess
		errMessage := ""
		if !ok {
			result = "fail"
			desc = "invalid user or password"
			status = RunStatusFailed
			errMessage = "invalid CUCC socket account"
		} else {
			s.markAuthenticated(account)
		}
		body := cuccCommand("ackLoginAlarm", [][2]string{
			{"result", result},
			{"resDesc", desc},
		})
		s.manager.recordEvent(eventFromSocketRuntime(s.config, "login", status, body, errMessage, map[string]any{
			"remote_addr":  s.remote,
			"account_key":  account.Key,
			"account_type": accountType,
			"command":      command,
			"session_id":   s.sessionID,
		}))
		if err := s.writeFrame(cuccMsgLoginAck, socketMessageFormatString, []byte(body)); err != nil {
			return err
		}
		if !ok {
			return io.EOF
		}
		return nil
	case cuccMsgHeartbeat:
		reqID := fields["reqid"]
		body := cuccCommand("ackHeartBeat", [][2]string{{"reqId", reqID}})
		s.manager.recordEvent(eventFromSocketRuntime(s.config, "heartbeat", RunStatusSuccess, body, "", map[string]any{
			"remote_addr": s.remote,
			"request_id":  reqID,
			"session_id":  s.sessionID,
		}))
		return s.writeFrame(cuccMsgHeartbeatAck, socketMessageFormatString, []byte(body))
	case cuccMsgSyncAlarm:
		req := socketAlarmSyncRequest{
			ReqID:    fields["reqid"],
			AlarmSeq: fields["alarmseq"],
		}
		reqID := req.ReqID
		alarmSeq := req.AlarmSeq
		if reqID == "" {
			reqID = strconv.FormatInt(time.Now().UnixNano(), 10)
			req.ReqID = reqID
		}
		result := "0"
		desc := "success"
		status := RunStatusSuccess
		errMessage := ""
		if !s.isAuthenticatedType("msg") || !s.config.ClientSyncEnabled {
			result = "1"
			desc = "client sync is disabled or session is not authenticated"
			status = RunStatusFailed
			errMessage = "client sync rejected"
		}
		if result == "0" {
			if msg := validateSocketMessageSyncRequest(req, true); msg != "" {
				result = "1"
				desc = msg
				status = RunStatusFailed
				errMessage = msg
			}
		}
		syncStarted := false
		if result == "0" {
			syncStarted = s.beginSync()
			if !syncStarted {
				result = "1"
				desc = "Synchronization is being performed"
				status = RunStatusFailed
				errMessage = desc
			}
		}
		var replayItems []socketAlarmSyncItem
		var replayEvents []PageConfigEvent
		var replayErr error
		replaySource := "alarm store"
		if result == "0" {
			replayItems, replayErr = s.loadSocketSyncItems(ctx, req, socketAlarmMessageLimit)
			if replayErr != nil && strings.Contains(replayErr.Error(), "alarm store is not configured") {
				replayEvents, replayErr = s.loadSocketReplayEvents(ctx, 50)
				replaySource = "northbound_page_config_events.alarm_push"
			}
			if replayErr != nil {
				result = "1"
				desc = replayErr.Error()
				status = RunStatusFailed
				errMessage = replayErr.Error()
			}
		}
		replayCount := len(replayItems) + len(replayEvents)
		body := cuccCommand("ackSyncAlarmMsg", [][2]string{
			{"reqId", reqID},
			{"result", result},
			{"alarmSeq", alarmSeq},
			{"count", strconv.Itoa(replayCount)},
			{"resDesc", desc},
		})
		s.manager.recordEvent(eventFromSocketRuntime(s.config, "sync_request", status, body, errMessage, map[string]any{
			"remote_addr":   s.remote,
			"request_id":    reqID,
			"alarm_seq":     alarmSeq,
			"replay_count":  replayCount,
			"replay_source": replaySource,
		}))
		if err := s.writeFrame(cuccMsgSyncAlarmAck, socketMessageFormatString, []byte(body)); err != nil {
			if syncStarted {
				_ = s.finishSync()
			}
			return err
		}
		if replayErr == nil {
			for _, item := range replayItems {
				frame, _, _ := socketRealtimeFrameFromItem(s.config, item)
				if err := s.writeFrame(frame.MessageType, frame.MessageFormat, frame.Body); err != nil {
					if syncStarted {
						_ = s.finishSync()
					}
					return err
				}
			}
			for _, event := range replayEvents {
				if strings.TrimSpace(event.Payload) == "" {
					continue
				}
				if err := s.writeFrame(cuccMsgRealtimeAlarm, socketMessageFormatString, []byte(event.Payload)); err != nil {
					if syncStarted {
						_ = s.finishSync()
					}
					return err
				}
			}
		}
		if syncStarted {
			pending := s.finishSync()
			if err := s.flushQueuedRealtimeFrames(pending); err != nil {
				return err
			}
		}
		return nil
	case cuccMsgSyncAlarmFile:
		return s.handleCUCCFileSync(ctx, fields)
	case cuccMsgClose:
		return io.EOF
	default:
		return fmt.Errorf("unsupported CUCC socket message type %d", frame.MessageType)
	}
}

func (s *socketAlarmSession) handleCUCCFileSync(ctx context.Context, fields map[string]string) error {
	req, validationMessage := validateSocketFileSyncRequest(fields)
	reqID := req.ReqID
	if validationMessage != "" {
		body := cuccCommand("ackSyncAlarmFile", [][2]string{
			{"reqId", reqID},
			{"result", "fail"},
			{"resDesc", validationMessage},
		})
		s.manager.recordEvent(eventFromSocketRuntime(s.config, "file_sync", RunStatusFailed, body, validationMessage, map[string]any{
			"remote_addr": s.remote,
			"request_id":  reqID,
		}))
		return s.writeFrame(cuccMsgSyncAlarmFileAck, socketMessageFormatString, []byte(body))
	}
	if !s.isAuthenticatedType("ftp") || !s.config.ClientSyncEnabled {
		body := cuccCommand("ackSyncAlarmFile", [][2]string{
			{"reqId", reqID},
			{"result", "fail"},
			{"resDesc", "No permissions."},
		})
		s.manager.recordEvent(eventFromSocketRuntime(s.config, "file_sync", RunStatusFailed, body, "No permissions.", map[string]any{
			"remote_addr": s.remote,
			"request_id":  reqID,
		}))
		return s.writeFrame(cuccMsgSyncAlarmFileAck, socketMessageFormatString, []byte(body))
	}
	ack := cuccCommand("ackSyncAlarmFile", [][2]string{
		{"reqId", reqID},
		{"result", "succ"},
		{"resDesc", "accepted"},
	})
	if err := s.writeFrame(cuccMsgSyncAlarmFileAck, socketMessageFormatString, []byte(ack)); err != nil {
		return err
	}

	items, err := s.loadSocketSyncItems(ctx, req, socketAlarmFileMaxRows)
	if err != nil && strings.Contains(err.Error(), "alarm store is not configured") {
		items = nil
		err = nil
	}
	artifactName := ""
	var artifact []byte
	content := ""
	if err == nil {
		artifactName, artifact, content, err = socketAlarmFileContent(s.config, req, items)
	}
	var remotePaths []string
	var failures []string
	if err != nil {
		failures = append(failures, err.Error())
	} else if s.manager.svc == nil || s.manager.svc.repo == nil {
		failures = append(failures, "northbound page-config repository is not configured")
	} else {
		targets, err := s.manager.svc.repo.ListActiveDeliveryTargets(ctx, DeliveryScopeSocket, s.config.Key)
		if err != nil {
			failures = append(failures, err.Error())
		}
		if len(targets) == 0 {
			failures = append(failures, "no enabled socket file-sync delivery target")
		}
		run := FileRun{
			ID:                 "socket-sync-" + safeSocketToken(reqID),
			ProfileKind:        ProfileKind("socket"),
			ProfileCode:        s.config.Key,
			GroupID:            "alarm-file-sync",
			Domain:             DomainLOG,
			ObjectCode:         "ALARM",
			Status:             RunStatusSuccess,
			ArtifactPath:       "/northupload/socket/" + time.Now().Format("20060102150405") + "/",
			ArtifactName:       artifactName,
			ArtifactContent:    content,
			ArtifactSize:       int64(len(artifact)),
			RowCount:           len(items),
			CompressionEnabled: true,
			CompressionFormat:  CompressionGz,
		}
		for _, target := range targets {
			result := uploadRunArtifact(ctx, target, run, artifact)
			if result.Success {
				remotePaths = append(remotePaths, result.RemotePath)
			} else {
				failures = append(failures, fmt.Sprintf("%s: %s", target.Key, result.Message))
			}
			s.manager.recordEvent(eventFromDeliveryRun(target, result))
		}
	}
	result := "succ"
	desc := "success"
	if len(failures) > 0 {
		result = "fail"
		desc = strings.Join(failures, ", ")
	}
	filePath := ""
	if len(remotePaths) > 0 {
		filePath = remotePaths[0]
	}
	done := cuccCommand("ackSyncAlarmFileResult", [][2]string{
		{"reqId", reqID},
		{"result", result},
		{"fileName", artifactName},
		{"filePath", filePath},
		{"resDesc", desc},
	})
	s.manager.recordEvent(eventFromSocketFileSync(s.config, reqID, artifactName, remotePaths, failures, done))
	return s.writeFrame(cuccMsgSyncAlarmFileDone, socketMessageFormatString, []byte(done))
}

func (s *socketAlarmSession) loadSocketReplayEvents(ctx context.Context, limit int) ([]PageConfigEvent, error) {
	if s.manager == nil || s.manager.svc == nil || s.manager.svc.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	result, err := s.manager.svc.repo.ListEvents(ctx, EventFilter{
		Capability: "socket",
		OwnerCode:  s.config.Key,
		TargetKey:  s.config.Key,
		EventType:  "alarm_push",
		Status:     RunStatusSuccess,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	events := make([]PageConfigEvent, 0, len(result.Items))
	for i := len(result.Items) - 1; i >= 0; i-- {
		event := result.Items[i]
		if strings.TrimSpace(event.Payload) == "" {
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

func (s *socketAlarmSession) writeCTCCJSON(messageType int, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.writeFrame(messageType, socketMessageFormatJSON, body)
}

func (s *socketAlarmSession) markAuthenticated(account SocketAccount) {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.authenticated = true
	s.account = account
}

func (s *socketAlarmSession) isAuthenticatedType(accountType string) bool {
	s.authMu.RLock()
	defer s.authMu.RUnlock()
	return s.authenticated && strings.EqualFold(s.account.Type, accountType)
}

func (s *socketAlarmSession) realtimeAccount() (SocketAccount, bool) {
	s.authMu.RLock()
	defer s.authMu.RUnlock()
	if !s.authenticated || !strings.EqualFold(s.account.Type, "msg") {
		return SocketAccount{}, false
	}
	return s.account, true
}

func authenticateSocketAccount(config SocketAlarmConfig, username, credential, accountType string) (SocketAccount, bool) {
	username = strings.TrimSpace(username)
	credential = strings.TrimSpace(credential)
	accountType = strings.TrimSpace(accountType)
	if accountType == "" {
		accountType = "msg"
	}
	if username == "" || credential == "" {
		return SocketAccount{}, false
	}
	for _, account := range config.Accounts {
		if !account.Enabled || !strings.EqualFold(account.Type, accountType) {
			continue
		}
		if strings.TrimSpace(account.Username) != username {
			continue
		}
		if strings.TrimSpace(account.Credential) == "" || strings.TrimSpace(account.Credential) != credential {
			continue
		}
		return account, true
	}
	return SocketAccount{}, false
}

func readSocketFrame(r io.Reader, profile string) (socketProtocolFrame, error) {
	if strings.EqualFold(profile, "CUCC") {
		var header [9]byte
		if _, err := io.ReadFull(r, header[:]); err != nil {
			return socketProtocolFrame{}, err
		}
		if binary.BigEndian.Uint16(header[0:2]) != cuccFrameStart {
			return socketProtocolFrame{}, fmt.Errorf("invalid CUCC socket frame start")
		}
		bodyLen := int(binary.BigEndian.Uint16(header[7:9]))
		if bodyLen > socketMaxFrameBody {
			return socketProtocolFrame{}, fmt.Errorf("CUCC socket frame body too large: %d", bodyLen)
		}
		body := make([]byte, bodyLen)
		if _, err := io.ReadFull(r, body); err != nil {
			return socketProtocolFrame{}, err
		}
		return socketProtocolFrame{MessageType: int(header[2]), MessageFormat: socketMessageFormatString, Body: body}, nil
	}
	var header [16]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return socketProtocolFrame{}, err
	}
	if binary.BigEndian.Uint16(header[0:2]) != ctccFrameStart {
		return socketProtocolFrame{}, fmt.Errorf("invalid CTCC socket frame start")
	}
	bodyLen := int(binary.BigEndian.Uint16(header[14:16]))
	if bodyLen > socketMaxFrameBody {
		return socketProtocolFrame{}, fmt.Errorf("CTCC socket frame body too large: %d", bodyLen)
	}
	body := make([]byte, bodyLen)
	if _, err := io.ReadFull(r, body); err != nil {
		return socketProtocolFrame{}, err
	}
	return socketProtocolFrame{
		MessageType:   int(binary.BigEndian.Uint16(header[8:10])),
		MessageFormat: int(binary.BigEndian.Uint16(header[10:12])),
		Body:          body,
	}, nil
}

func encodeSocketFrame(profile string, messageType int, messageFormat int, body []byte) ([]byte, error) {
	if len(body) > 0xffff {
		return nil, fmt.Errorf("socket frame body exceeds uint16 length: %d", len(body))
	}
	if strings.EqualFold(profile, "CUCC") {
		out := make([]byte, 9+len(body))
		binary.BigEndian.PutUint16(out[0:2], cuccFrameStart)
		out[2] = byte(messageType)
		binary.BigEndian.PutUint32(out[3:7], uint32(time.Now().Unix()))
		binary.BigEndian.PutUint16(out[7:9], uint16(len(body)))
		copy(out[9:], body)
		return out, nil
	}
	if messageFormat == 0 {
		messageFormat = socketMessageFormatJSON
	}
	out := make([]byte, 16+len(body))
	binary.BigEndian.PutUint16(out[0:2], ctccFrameStart)
	writeCTCCTimestamp(out[2:8], time.Now())
	binary.BigEndian.PutUint16(out[8:10], uint16(messageType))
	binary.BigEndian.PutUint16(out[10:12], uint16(messageFormat))
	binary.BigEndian.PutUint16(out[14:16], uint16(len(body)))
	copy(out[16:], body)
	return out, nil
}

func writeCTCCTimestamp(dst []byte, now time.Time) {
	value := uint64(now.UnixMilli()) & 0x0000ffffffffffff
	for i := 5; i >= 0; i-- {
		dst[i] = byte(value)
		value >>= 8
	}
}

func (m *SocketAlarmServerManager) handleAlarmEvent(ctx context.Context, subject string, evt event.Event) error {
	var alarm model.Alarm
	if err := evt.DecodePayload(&alarm); err != nil {
		m.logger.Warn("decode alarm event for socket forward failed",
			zap.String("subject", subject),
			zap.String("event_id", evt.ID),
			zap.Error(err))
		return nil
	}
	buckets := m.messageSessionBuckets()
	if len(buckets) == 0 {
		return nil
	}
	for _, bucket := range buckets {
		frame, payload, alarmID := socketRealtimeFrameFromModel(bucket.config, alarm, subject)
		sent := 0
		failed := 0
		for _, session := range bucket.sessions {
			if err := session.queueOrWriteRealtimeFrame(frame); err != nil {
				failed++
				session.close()
				m.unregisterSession(session)
				continue
			}
			sent++
		}
		status := RunStatusSuccess
		errMessage := ""
		if failed > 0 && sent == 0 {
			status = RunStatusFailed
			errMessage = "socket alarm push failed for all sessions"
		} else if failed > 0 {
			status = RunStatusFailed
			errMessage = "socket alarm push failed for some sessions"
		}
		m.recordEvent(eventFromSocketRuntime(bucket.config, "alarm_push", status, payload, errMessage, map[string]any{
			"alarm_id":        alarmID,
			"subject":         subject,
			"sent_sessions":   sent,
			"failed_sessions": failed,
		}))
	}
	return nil
}

func (m *SocketAlarmServerManager) messageSessionBuckets() []socketBroadcastBucket {
	m.mu.RLock()
	defer m.mu.RUnlock()
	buckets := map[string]socketBroadcastBucket{}
	for session := range m.sessions {
		if !session.config.RealtimePushEnabled {
			continue
		}
		if _, ok := session.realtimeAccount(); !ok {
			continue
		}
		bucket := buckets[session.config.Key]
		bucket.config = session.config
		bucket.sessions = append(bucket.sessions, session)
		buckets[session.config.Key] = bucket
	}
	out := make([]socketBroadcastBucket, 0, len(buckets))
	for _, bucket := range buckets {
		out = append(out, bucket)
	}
	return out
}

func socketRealtimeFrameFromModel(config SocketAlarmConfig, alarm model.Alarm, subject string) (socketProtocolFrame, string, string) {
	alarmID := socketAlarmID(alarm)
	if strings.EqualFold(config.Profile, "CUCC") {
		body := cuccCommand("realTimeAlarm", socketAlarmCUCCFields(alarm, subject))
		return socketProtocolFrame{MessageType: cuccMsgRealtimeAlarm, MessageFormat: socketMessageFormatString, Body: []byte(body)}, body, alarmID
	}
	payload := socketAlarmCTCCPayload(alarm, subject)
	body, _ := json.Marshal(payload)
	return socketProtocolFrame{MessageType: ctccMsgRealtimeAlarm, MessageFormat: socketMessageFormatJSON, Body: body}, string(body), alarmID
}

func socketAlarmID(alarm model.Alarm) string {
	return firstNonEmpty(alarm.AlarmIdentifier, alarm.ID.String())
}

func socketAlarmSequence(alarm model.Alarm) string {
	for _, key := range []string{"alarmSeq", "alarm_seq", "notificationID", "notification_id", "sequence_id"} {
		if alarm.AdditionalInfo != nil && strings.TrimSpace(alarm.AdditionalInfo[key]) != "" {
			raw := strings.TrimSpace(alarm.AdditionalInfo[key])
			if _, ok := parseSocketAlarmSeq(raw); ok {
				return raw
			}
			if digits := digitsOnly(raw); digits != "" {
				return digits
			}
		}
	}
	t := socketAlarmEventTime(alarm, eventSubjectForAlarm(alarm))
	if !t.IsZero() {
		return strconv.FormatInt(t.UnixMilli(), 10)
	}
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

func digitsOnly(value string) string {
	var b strings.Builder
	for _, ch := range value {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func parseCUCCCommand(body string) (string, map[string]string) {
	parts := strings.Split(strings.TrimSpace(body), ";")
	fields := map[string]string{}
	command := ""
	if len(parts) > 0 {
		command = strings.TrimSpace(parts[0])
	}
	for _, part := range parts[1:] {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		fields[key] = value
		fields[strings.ToLower(key)] = value
	}
	return command, fields
}

func cuccCommand(name string, fields [][2]string) string {
	var b strings.Builder
	b.WriteString(name)
	for _, field := range fields {
		b.WriteByte(';')
		b.WriteString(field[0])
		b.WriteByte('=')
		b.WriteString(sanitizeCUCCValue(field[1]))
	}
	return b.String()
}

func sanitizeCUCCValue(value string) string {
	value = strings.ReplaceAll(value, ";", ",")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func jsonTextValue(body []byte, keys ...string) string {
	var values map[string]any
	if err := json.Unmarshal(body, &values); err != nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

func safeSocketToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "NA"
	}
	var b strings.Builder
	for _, ch := range value {
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			b.WriteRune(ch)
		}
	}
	if b.Len() == 0 {
		return "NA"
	}
	if b.Len() > 48 {
		return b.String()[:48]
	}
	return b.String()
}

func (m *SocketAlarmServerManager) recordEvent(evt PageConfigEvent) {
	if m == nil || m.svc == nil || m.svc.repo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := m.svc.repo.CreateEvent(ctx, evt); err != nil {
		m.logger.Warn("record northbound socket page-config event failed",
			zap.String("config_key", evt.OwnerCode),
			zap.String("event_type", evt.EventType),
			zap.Error(err))
	}
}

package acs

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/rpc"
<<<<<<< HEAD
	"github.com/omcgo/omcgo/internal/acs/upload"
=======
	"github.com/omcgo/omcgo/internal/core/components/logger"
>>>>>>> c2a509c (feat(components): 实现 Request ID 中间件和 SQL 日志功能)
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/pkg/soap"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// RequestIDHeader is the header key for request ID
const RequestIDHeader = "X-Request-ID"

// connSessionEntry tracks a connection-level session binding with creation time for TTL cleanup.
type connSessionEntry struct {
	DeviceSN  string
	CreatedAt time.Time
}

// Handler processes TR069/CWMP HTTP requests.
type Handler struct {
	sessionStore     SessionStore
	commandQueue     cmdqueue.CommandQueue
	eventBus         event.EventBus
	authenticator    auth.DeviceAuthenticator
	rpcDispatcher    *rpc.Dispatcher
	rateLimiter      *DeviceRateLimiter
	admission        *AdmissionController
	metrics          *ACSMetrics
	logger           *zap.Logger
	requestIDPrefix  string // prefix for request IDs, e.g., "acs"
	// connSessions maps HTTP RemoteAddr → connSessionEntry for connection-level session tracking.
	// Entries are cleaned up on session completion or by the background reaper.
	connSessions sync.Map
}

// startSessionReaper launches a background goroutine that periodically cleans up
// stale connSessions entries (e.g., from dropped TCP connections).
// It releases admission slots and decrements metrics for reaped sessions.
func (h *Handler) startSessionReaper(interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			h.connSessions.Range(func(key, value interface{}) bool {
				entry := value.(connSessionEntry)
				if now.Sub(entry.CreatedAt) > maxAge {
					h.connSessions.Delete(key)
					h.admission.Release()
					h.metrics.ActiveSessions.Dec()
					h.metrics.SessionDuration.Observe(now.Sub(entry.CreatedAt).Seconds())
					h.logger.Warn("reaped stale session",
						zap.String("device_sn", entry.DeviceSN),
						zap.String("remote_addr", key.(string)),
						zap.Duration("age", now.Sub(entry.CreatedAt)))
				}
				return true
			})
		}
	}()
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Generate or propagate Request ID
	requestID := r.Header.Get(RequestIDHeader)
	if requestID == "" {
		prefix := h.requestIDPrefix
		if prefix == "" {
			prefix = "acs" // default prefix
		}
		requestID = generateRequestIDWithPrefix(prefix)
	}

	// Store Request ID in context for logger and downstream services
	ctx := logger.WithRequestID(r.Context(), requestID)
	r = r.WithContext(ctx)

	// Set Request ID in response header for client correlation
	w.Header().Set(RequestIDHeader, requestID)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("read request body", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Detect method from body
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) == 0 {
		h.handleEmpty(w, r)
		return
	}

	method := soap.DetectRPCMethod(body)

	switch method {
	case soap.MethodInform:
		h.handleInform(w, r, body)
	case soap.MethodTransferComplete:
		h.handleTransferComplete(w, r, body)
	case soap.MethodAutonomousTransferComplete:
		h.handleAutonomousTransferComplete(w, r, body)
	case soap.MethodGetParameterValuesResp,
		soap.MethodSetParameterValuesResp,
		soap.MethodGetParameterNamesResp,
		soap.MethodGetParameterAttributesResp,
		soap.MethodSetParameterAttributesResp,
		soap.MethodAddObjectResp,
		soap.MethodDeleteObjectResp,
		soap.MethodDownloadResp,
		soap.MethodUploadResp,
		soap.MethodRebootResp,
		soap.MethodFactoryResetResp:
		h.handleRPCResponse(w, r, body, method)
	default:
		h.logger.Warn("unknown SOAP method", zap.String("method", string(method)))
		http.Error(w, "Unknown method", http.StatusBadRequest)
	}
}

// handleInform processes an Inform message from a CPE device.
// Per TR069 spec: Inform → InformResponse (always). RPC dispatch happens on the
// subsequent Empty POST via handleEmpty().
func (h *Handler) handleInform(w http.ResponseWriter, r *http.Request, body []byte) {
	// Log raw request body for debugging
	h.logger.Debug("ACS received raw Inform body",
		zap.String("remote_addr", r.RemoteAddr),
		zap.Int("body_len", len(body)),
		zap.String("body_preview", truncateString(string(body), 500)),
	)

	// Parse Inform
	inform, cwmpID, err := soap.DecodeInform(bytes.NewReader(body))
	if err != nil {
		h.logger.Error("decode Inform", zap.Error(err),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("body_preview", truncateString(string(body), 200)),
		)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	deviceSN := inform.DeviceId.SerialNumber

	// Log parsed Inform details
	h.logger.Info("ACS parsed Inform",
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("device_sn", deviceSN),
		zap.String("oui", inform.DeviceId.OUI),
		zap.String("product_class", inform.DeviceId.ProductClass),
		zap.String("manufacturer", inform.DeviceId.Manufacturer),
		zap.String("cwmp_id", cwmpID),
	)

	// Rate limiting
	if !h.rateLimiter.Allow(deviceSN) {
		h.metrics.RateLimitRejected.Inc()
		h.logger.Warn("rate limited", zap.String("device_sn", deviceSN))
		http.Error(w, "Too Many Requests", http.StatusServiceUnavailable)
		return
	}

	// Admission control — slot is held until session completes (via completeSession)
	// or the background reaper cleans it up.
	if !h.admission.Acquire() {
		h.logger.Warn("admission denied", zap.String("device_sn", deviceSN))
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	// Track active session — will be decremented by completeSession() or reaper.
	h.metrics.ActiveSessions.Inc()

	// Record metrics
	eventCodes := tr069.EventCodes(inform.Event)
	for _, code := range eventCodes {
		h.metrics.InformTotal.WithLabelValues(code).Inc()
	}

	h.logger.Info("ACS Inform processing",
		zap.String("device_sn", deviceSN),
		zap.String("oui", inform.DeviceId.OUI),
		zap.String("product_class", inform.DeviceId.ProductClass),
		zap.Strings("events", eventCodes),
		zap.Int("param_count", len(inform.ParameterList)),
	)

	// Create/update session
	session := &Session{
		DeviceSN:     deviceSN,
		State:        StateInformReceived,
		InstanceID:   r.RemoteAddr,
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		InformEvents: eventCodes,
		CWMPId:       cwmpID,
	}

	if err := h.sessionStore.Create(r.Context(), deviceSN, session); err != nil {
		h.logger.Error("create session", zap.Error(err), zap.String("device_sn", deviceSN))
	}

	// Bind connection (RemoteAddr) → deviceSN for session tracking.
	h.connSessions.Store(r.RemoteAddr, connSessionEntry{
		DeviceSN:  deviceSN,
		CreatedAt: time.Now(),
	})

	// Publish events
	h.publishInformEvents(r.Context(), inform, eventCodes)

	// Per TR069 spec: Always send InformResponse first.
	// Command queue will be checked on the subsequent Empty POST.
	h.logger.Info("ACS Inform done, sending InformResponse",
		zap.String("device_sn", deviceSN),
		zap.String("cwmp_id", cwmpID))
	h.sendInformResponse(w, cwmpID)
}

// handleEmpty processes an empty POST from the CPE.
// Per TR069 spec, after InformResponse the CPE sends an empty POST.
// The ACS should then either send an RPC request or an empty response to close the session.
func (h *Handler) handleEmpty(w http.ResponseWriter, r *http.Request) {
	// Look up the device SN from the connection binding.
	val, ok := h.connSessions.Load(r.RemoteAddr)
	if !ok {
		// No session bound to this connection — just close.
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	entry := val.(connSessionEntry)
	deviceSN := entry.DeviceSN

	session, _ := h.sessionStore.Get(r.Context(), deviceSN)
	if session == nil {
		h.completeSession(r.Context(), deviceSN, r.RemoteAddr, nil)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Transition from InformReceived → Processing
	if session.State == StateInformReceived {
		session.State = StateProcessing
		session.UpdatedAt = time.Now()
		h.sessionStore.Update(r.Context(), deviceSN, session)
	}

	// Check command queue for pending commands
	cmd, err := h.commandQueue.Pop(r.Context(), deviceSN)
	if err != nil {
		h.logger.Error("pop command queue", zap.Error(err))
	}

	if cmd != nil {
		session.State = StateRPCPending
		session.LastRPC = cmd.Method
		session.UpdatedAt = time.Now()
		h.sessionStore.Update(r.Context(), deviceSN, session)

		respData, err := h.rpcDispatcher.BuildRequest(cmd, session.CWMPId)
		if err != nil {
			h.logger.Error("build RPC request", zap.Error(err))
			h.metrics.RPCErrorsTotal.WithLabelValues(cmd.Method).Inc()
		} else {
			h.sendSOAPResponse(w, respData)
			return
		}
	}

	// No more commands — complete the session.
	h.completeSession(r.Context(), deviceSN, r.RemoteAddr, session)

	// Send truly empty response to signal end of session (no body per TR069 spec).
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRPCResponse(w http.ResponseWriter, r *http.Request, body []byte, method soap.RPCMethod) {
	// Extract CWMP ID from the SOAP response.
	_, cwmpID, _, _ := soap.DetectMethod(bytes.NewReader(body))

	h.logger.Info("RPC response received",
		zap.String("method", string(method)),
		zap.String("cwmp_id", cwmpID),
	)

	// Find the device SN from the connection context.
	val, ok := h.connSessions.Load(r.RemoteAddr)
	if !ok {
		// Fallback: no connection binding found.
		h.logger.Warn("no connection binding for RPC response", zap.String("remote_addr", r.RemoteAddr))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	entry := val.(connSessionEntry)
	deviceSN := entry.DeviceSN

	session, _ := h.sessionStore.Get(r.Context(), deviceSN)
	if session != nil {
		// Record RPC duration (approximate: time since last state update).
		rpcDuration := time.Since(session.UpdatedAt).Seconds()
		h.metrics.RPCDuration.WithLabelValues(string(method)).Observe(rpcDuration)

		session.State = StateRPCResponse
		session.UpdatedAt = time.Now()
		h.sessionStore.Update(r.Context(), deviceSN, session)

		// Publish RPC response event for provisioning engine.
		h.publishRPCResponseEvent(r.Context(), deviceSN, method)

		// Check if there are more commands in the queue (multi-step RPC).
		cmd, err := h.commandQueue.Pop(r.Context(), deviceSN)
		if err != nil {
			h.logger.Error("pop command queue", zap.Error(err))
		}

		if cmd != nil {
			session.State = StateRPCPending
			session.LastRPC = cmd.Method
			session.UpdatedAt = time.Now()
			h.sessionStore.Update(r.Context(), deviceSN, session)

			respData, err := h.rpcDispatcher.BuildRequest(cmd, cwmpID)
			if err != nil {
				h.logger.Error("build next RPC request", zap.Error(err))
				h.metrics.RPCErrorsTotal.WithLabelValues(cmd.Method).Inc()
			} else {
				h.sendSOAPResponse(w, respData)
				return
			}
		}

		// No more commands — complete the session.
		h.completeSession(r.Context(), deviceSN, r.RemoteAddr, session)
	}

	// Send truly empty response to signal end of session (no body per TR069 spec).
	w.WriteHeader(http.StatusNoContent)
}

// completeSession releases admission, decrements metrics, records session duration,
// cleans up connSessions, and marks the session as complete in the store.
func (h *Handler) completeSession(ctx context.Context, deviceSN, remoteAddr string, session *Session) {
	h.connSessions.Delete(remoteAddr)
	h.admission.Release()
	h.metrics.ActiveSessions.Dec()

	if session != nil {
		duration := time.Since(session.StartedAt).Seconds()
		h.metrics.SessionDuration.Observe(duration)

		session.State = StateComplete
		session.UpdatedAt = time.Now()
		h.sessionStore.Update(ctx, deviceSN, session)
	}
}

func (h *Handler) handleTransferComplete(w http.ResponseWriter, r *http.Request, body []byte) {
	tc, cwmpID, err := soap.DecodeTransferComplete(bytes.NewReader(body))
	if err != nil {
		h.logger.Error("decode TransferComplete", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Get deviceSN from connection binding for logging/events.
	deviceSN := ""
	if val, ok := h.connSessions.Load(r.RemoteAddr); ok {
		deviceSN = val.(connSessionEntry).DeviceSN
	}

	h.logger.Info("TransferComplete received",
		zap.String("device_sn", deviceSN),
		zap.String("command_key", tc.CommandKey))

	// Publish event
	evt, _ := event.NewEvent(event.SubjectDeviceTransferComplete, tc)
	h.eventBus.Publish(r.Context(), event.SubjectDeviceTransferComplete, evt)

	// Send TransferCompleteResponse
	resp, err := soap.RenderResponse(soap.TransferCompleteRespTmpl, soap.InformResponseData{ID: cwmpID})
	if err != nil {
		h.logger.Error("render TransferCompleteResponse", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	h.sendSOAPResponse(w, resp)
}

// handleAutonomousTransferComplete processes an AutonomousTransferComplete message
// from a CPE device (e.g., PM/MR file upload completion).
func (h *Handler) handleAutonomousTransferComplete(w http.ResponseWriter, r *http.Request, body []byte) {
	atc, cwmpID, err := soap.DecodeAutonomousTransferComplete(bytes.NewReader(body))
	if err != nil {
		h.logger.Error("decode AutonomousTransferComplete", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Get deviceSN from connection binding.
	deviceSN := ""
	if val, ok := h.connSessions.Load(r.RemoteAddr); ok {
		deviceSN = val.(connSessionEntry).DeviceSN
	}

	h.logger.Info("AutonomousTransferComplete received",
		zap.String("device_sn", deviceSN),
		zap.String("file_type", atc.FileType),
		zap.String("transfer_url", atc.TransferURL),
		zap.Bool("is_download", atc.IsDownload),
	)

	// Publish event
	payload := map[string]interface{}{
		"device_sn":       deviceSN,
		"announce_url":    atc.AnnounceURL,
		"transfer_url":    atc.TransferURL,
		"is_download":     atc.IsDownload,
		"file_type":       atc.FileType,
		"file_size":       atc.FileSize,
		"target_filename": atc.TargetFileName,
		"start_time":      atc.StartTime,
		"complete_time":   atc.CompleteTime,
	}
	if atc.FaultStruct != nil {
		payload["fault"] = atc.FaultStruct
	}
	evt, _ := event.NewEvent(event.SubjectDeviceAutonomousTransferComplete, payload)
	h.eventBus.Publish(r.Context(), event.SubjectDeviceAutonomousTransferComplete, evt)

	// Send AutonomousTransferCompleteResponse
	resp, err := soap.RenderResponse(soap.AutonomousTransferCompleteRespTmpl, soap.InformResponseData{ID: cwmpID})
	if err != nil {
		h.logger.Error("render AutonomousTransferCompleteResponse", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	h.sendSOAPResponse(w, resp)
}

func (h *Handler) publishInformEvents(ctx context.Context, inform *tr069.InformMessage, eventCodes []string) {
	// Build event payload
	payload := map[string]interface{}{
		"device_id":      inform.DeviceId,
		"events":         eventCodes,
		"parameter_list": inform.ParameterList,
		"current_time":   inform.CurrentTime,
		"retry_count":    inform.RetryCount,
	}

	// Determine primary subject based on event codes (priority order).
	var subject string
	switch {
	case tr069.IsBootstrap(inform.Event):
		subject = event.SubjectDeviceBootstrap
	case tr069.IsAlarm(inform.Event):
		subject = event.SubjectDeviceAlarm
	case tr069.IsRebootComplete(inform.Event):
		subject = event.SubjectDeviceRebootComplete
	case tr069.IsConnectionRequest(inform.Event):
		subject = event.SubjectDeviceConnectionRequest
	case tr069.HasEvent(inform.Event, tr069.EventTransferComplete):
		subject = event.SubjectDeviceTransferComplete
	case tr069.IsValueChange(inform.Event):
		subject = event.SubjectDeviceValueChange
	case tr069.IsPeriodic(inform.Event):
		subject = event.SubjectDevicePeriodic
	default:
		subject = event.SubjectDevicePeriodic
	}

	evt, err := event.NewEvent(subject, payload)
	if err != nil {
		h.logger.Error("create event failed",
			zap.Error(err),
			zap.String("device_sn", inform.DeviceId.SerialNumber),
			zap.String("subject", subject))
		return
	}

	if err := h.eventBus.Publish(ctx, subject, evt); err != nil {
		h.logger.Error("publish event failed",
			zap.Error(err),
			zap.String("device_sn", inform.DeviceId.SerialNumber),
			zap.String("subject", subject),
			zap.String("event_id", evt.ID))
		return
	}

	h.logger.Info("event published to bus",
		zap.String("device_sn", inform.DeviceId.SerialNumber),
		zap.String("subject", subject),
		zap.String("event_id", evt.ID),
		zap.Strings("event_codes", eventCodes))
}

func (h *Handler) publishRPCResponseEvent(ctx context.Context, deviceSN string, method soap.RPCMethod) {
	var subject string
	switch method {
	case soap.MethodGetParameterValuesResp:
		subject = event.SubjectCommandGetParamsResponse
	case soap.MethodSetParameterValuesResp:
		subject = event.SubjectCommandSetParamsResponse
	case soap.MethodDownloadResp:
		subject = event.SubjectCommandDownloadResponse
	case soap.MethodUploadResp:
		subject = event.SubjectCommandUploadResponse
	case soap.MethodGetParameterNamesResp:
		subject = event.SubjectCommandGetNamesResponse
	case soap.MethodAddObjectResp:
		subject = event.SubjectCommandAddObjectResponse
	case soap.MethodDeleteObjectResp:
		subject = event.SubjectCommandDeleteObjectResponse
	case soap.MethodRebootResp:
		subject = event.SubjectCommandRebootResponse
	case soap.MethodFactoryResetResp:
		subject = event.SubjectCommandFactoryResetResponse
	case soap.MethodGetParameterAttributesResp:
		subject = event.SubjectCommandGetAttrsResponse
	case soap.MethodSetParameterAttributesResp:
		subject = event.SubjectCommandSetAttrsResponse
	default:
		return
	}

	payload := map[string]interface{}{
		"device_sn": deviceSN,
		"method":    string(method),
	}

	evt, err := event.NewEvent(subject, payload)
	if err != nil {
		h.logger.Error("create RPC response event", zap.Error(err))
		return
	}
	if err := h.eventBus.Publish(ctx, subject, evt); err != nil {
		h.logger.Error("publish RPC response event", zap.Error(err), zap.String("subject", subject))
	}
}

func (h *Handler) sendInformResponse(w http.ResponseWriter, cwmpID string) {
	resp, err := soap.RenderResponse(soap.InformResponseTmpl, soap.InformResponseData{ID: cwmpID})
	if err != nil {
		h.logger.Error("render InformResponse", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	h.sendSOAPResponse(w, resp)
}

func (h *Handler) sendSOAPResponse(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// truncateString truncates a string to maxLen characters for logging purposes.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// generateRequestIDWithPrefix generates a unique request ID with a custom prefix.
// Format: {prefix}-{timestamp}-{random}
// Example: acs-20260319150430-a1b2c3d4
func generateRequestIDWithPrefix(prefix string) string {
	timestamp := time.Now().Format("20060102150405")
	random := make([]byte, 4)
	rand.Read(random)
	return prefix + "-" + timestamp + "-" + hex.EncodeToString(random)
}

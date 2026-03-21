package acs

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/soap"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// RequestIDHeader is the header key for request ID
const RequestIDHeader = "X-Request-ID"

// SessionCookieName is the cookie name for TR069 session ID
const SessionCookieName = "SESSION"

// connSessionEntry tracks a connection-level session binding with creation time for TTL cleanup.
type connSessionEntry struct {
	DeviceSN  string
	CreatedAt time.Time
}

// Handler processes TR069/CWMP HTTP requests.
type Handler struct {
	sessionStore    SessionStore
	commandQueue    cmdqueue.CommandQueue // deprecated: use taskService instead
	taskService     *task.TaskService     // new task management service
	eventBus        event.EventBus
	authenticator   auth.DeviceAuthenticator
	rpcDispatcher   *rpc.Dispatcher
	rateLimiter     *DeviceRateLimiter
	admission       *AdmissionController
	metrics         *ACSMetrics
	logger          *zap.Logger
	requestIDPrefix string // prefix for request IDs, e.g., "acs"
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

	// Create context-aware logger with request_id
	log := logger.L(ctx)

	if r.Method != http.MethodPost {
		log.Warn("method not allowed", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("read request body", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log complete request XML
	log.Debug("ACS received request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.Int("body_len", len(body)),
		zap.String("xml", string(body)),
	)

	// Detect method from body
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) == 0 {
		h.handleEmpty(w, r, log)
		return
	}

	method := soap.DetectRPCMethod(body)
	log.Info("ACS detected RPC method", zap.String("method", string(method)))

	switch method {
	case soap.MethodInform:
		h.handleInform(w, r, body, log)
	case soap.MethodTransferComplete:
		h.handleTransferComplete(w, r, body, log)
	case soap.MethodAutonomousTransferComplete:
		h.handleAutonomousTransferComplete(w, r, body, log)
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
		h.handleRPCResponse(w, r, body, method, log)
	default:
		log.Warn("unknown SOAP method", zap.String("method", string(method)))
		http.Error(w, "Unknown method", http.StatusBadRequest)
	}
}

// handleInform processes an Inform message from a CPE device.
// Per TR069 spec: Inform → InformResponse (always). RPC dispatch happens on the
// subsequent Empty POST via handleEmpty().
func (h *Handler) handleInform(w http.ResponseWriter, r *http.Request, body []byte, log *zap.Logger) {
	// Log complete request XML
	log.Debug("ACS received Inform request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.Int("body_len", len(body)),
		zap.String("xml", string(body)),
	)

	// Parse Inform
	inform, cwmpID, err := soap.DecodeInform(bytes.NewReader(body))
	if err != nil {
		log.Error("decode Inform", zap.Error(err),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("xml", string(body)),
		)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	deviceSN := inform.DeviceId.SerialNumber

	// Log parsed Inform details
	log.Info("ACS parsed Inform",
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
		log.Warn("rate limited", zap.String("device_sn", deviceSN))
		http.Error(w, "Too Many Requests", http.StatusServiceUnavailable)
		return
	}

	// Admission control — slot is held until session completes (via completeSession)
	// or the background reaper cleans it up.
	if !h.admission.Acquire() {
		log.Warn("admission denied", zap.String("device_sn", deviceSN))
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

	log.Info("ACS Inform processing",
		zap.String("device_sn", deviceSN),
		zap.String("oui", inform.DeviceId.OUI),
		zap.String("product_class", inform.DeviceId.ProductClass),
		zap.Strings("events", eventCodes),
		zap.Int("param_count", len(inform.ParameterList)),
	)

	// Create/update session with new Session ID
	sessionID := generateSessionID()
	session := &Session{
		ID:           sessionID,
		DeviceSN:     deviceSN,
		State:        StateInformReceived,
		InstanceID:   r.RemoteAddr,
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		InformEvents: eventCodes,
		CWMPId:       cwmpID,
	}

	// Store session by Session ID (Cookie-based lookup)
	if err := h.sessionStore.CreateWithID(r.Context(), sessionID, session); err != nil {
		log.Error("create session by id", zap.Error(err), zap.String("device_sn", deviceSN))
	}

	// Publish events
	h.publishInformEvents(r.Context(), inform, eventCodes, log)

	// Per TR069 spec: Always send InformResponse first.
	// Command queue will be checked on the subsequent Empty POST.
	log.Info("ACS Inform done, sending InformResponse",
		zap.String("device_sn", deviceSN),
		zap.String("cwmp_id", cwmpID),
		zap.String("session_id", sessionID))

	// Set Session Cookie in response header
	h.setSessionCookie(w, sessionID)

	h.sendInformResponse(w, cwmpID, log)
}

// handleEmpty processes an empty POST from the CPE.
// Per TR069 spec, after InformResponse the CPE sends an empty POST.
// The ACS should then either send an RPC request or an empty response to close the session.
func (h *Handler) handleEmpty(w http.ResponseWriter, r *http.Request, log *zap.Logger) {
	// Look up session from Cookie
	session, sessionID := h.getSessionFromCookie(r, log)
	if session == nil {
		// No valid session — just close.
		log.Warn("empty POST without valid session cookie", zap.String("remote_addr", r.RemoteAddr))
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	deviceSN := session.DeviceSN

	// Transition from InformReceived → Processing
	if session.State == StateInformReceived {
		session.State = StateProcessing
		session.UpdatedAt = time.Now()
		h.sessionStore.UpdateByID(r.Context(), sessionID, session)
	}

	// Priority 1: Try new TaskService
	if h.taskService != nil {
		taskItem, err := h.taskService.PopTask(r.Context(), deviceSN)
		if err != nil {
			log.Error("pop task from queue", zap.Error(err))
		} else if taskItem != nil {
			// Generate CWMP ID for this task
			cwmpID := task.GenerateCWMPID(taskItem.Method)

			// Mark task as sent
			if err := h.taskService.MarkTaskSent(r.Context(), taskItem.ID, cwmpID); err != nil {
				log.Error("mark task sent", zap.Error(err), zap.String("task_id", taskItem.ID))
			}

			// Update session state
			session.State = StateRPCPending
			session.LastRPC = taskItem.Method
			session.UpdatedAt = time.Now()
			h.sessionStore.UpdateByID(r.Context(), sessionID, session)

			// Build RPC request with CWMP ID
			cmd := &cmdqueue.Command{
				ID:         taskItem.ID,
				Method:     taskItem.Method,
				Params:     taskItem.Params,
				CommandKey: taskItem.CommandKey,
			}
			respData, err := h.rpcDispatcher.BuildRequest(cmd, cwmpID)
			if err != nil {
				log.Error("build RPC request from task", zap.Error(err))
				h.metrics.RPCErrorsTotal.WithLabelValues(taskItem.Method).Inc()
				// Mark task as failed
				h.taskService.MarkTaskFailed(r.Context(), taskItem.ID, 0, err.Error())
			} else {
				log.Info("ACS sending RPC request from task",
					zap.String("device_sn", deviceSN),
					zap.String("method", taskItem.Method),
					zap.String("task_id", taskItem.ID),
					zap.String("cwmp_id", cwmpID),
				)
				// Set session cookie in response
				h.setSessionCookie(w, sessionID)
				h.sendSOAPResponse(w, respData, log)
				return
			}
		}
	}

	// Priority 2: Fallback to legacy command queue (for backward compatibility)
	cmd, err := h.commandQueue.Pop(r.Context(), deviceSN)
	if err != nil {
		log.Error("pop command queue", zap.Error(err))
	}

	if cmd != nil {
		session.State = StateRPCPending
		session.LastRPC = cmd.Method
		session.UpdatedAt = time.Now()
		h.sessionStore.UpdateByID(r.Context(), sessionID, session)

		respData, err := h.rpcDispatcher.BuildRequest(cmd, session.CWMPId)
		if err != nil {
			log.Error("build RPC request", zap.Error(err))
			h.metrics.RPCErrorsTotal.WithLabelValues(cmd.Method).Inc()
		} else {
			log.Debug("ACS sending RPC request",
				zap.String("device_sn", deviceSN),
				zap.String("method", cmd.Method),
				zap.String("xml", string(respData)),
			)
			// Set session cookie in response
			h.setSessionCookie(w, sessionID)
			h.sendSOAPResponse(w, respData, log)
			return
		}
	}

	// No more commands — complete the session.
	h.completeSession(r.Context(), session)

	// Send truly empty response to signal end of session (no body per TR069 spec).
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRPCResponse(w http.ResponseWriter, r *http.Request, body []byte, method soap.RPCMethod, log *zap.Logger) {
	// Extract CWMP ID from the SOAP response.
	_, cwmpID, _, _ := soap.DetectMethod(bytes.NewReader(body))

	// Log complete response XML
	log.Debug("ACS received RPC response",
		zap.String("method", string(method)),
		zap.String("cwmp_id", cwmpID),
		zap.String("xml", string(body)),
	)

	log.Info("RPC response received",
		zap.String("method", string(method)),
		zap.String("cwmp_id", cwmpID),
	)

	// Look up session from Cookie
	session, sessionID := h.getSessionFromCookie(r, log)
	if session == nil {
		log.Warn("no valid session cookie for RPC response", zap.String("remote_addr", r.RemoteAddr))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	deviceSN := session.DeviceSN

	// Record RPC duration (approximate: time since last state update).
	rpcDuration := time.Since(session.UpdatedAt).Seconds()
	h.metrics.RPCDuration.WithLabelValues(string(method)).Observe(rpcDuration)

	session.State = StateRPCResponse
	session.UpdatedAt = time.Now()
	h.sessionStore.UpdateByID(r.Context(), sessionID, session)

	// Check if this is a task-based RPC (new task queue system)
	if h.taskService != nil && cwmpID != "" {
		taskItem, err := h.taskService.GetTaskByCWMPID(r.Context(), cwmpID)
		if err != nil {
			log.Warn("get task by cwmp_id", zap.Error(err), zap.String("cwmp_id", cwmpID))
		} else if taskItem != nil {
			// Check for SOAP fault in response
			if isFault, faultCode, faultMsg := detectSOAPFault(body); isFault {
				// Task failed with SOAP fault
				if markErr := h.taskService.MarkTaskFailed(r.Context(), taskItem.ID, faultCode, faultMsg); markErr != nil {
					log.Error("mark task failed", zap.Error(markErr), zap.String("task_id", taskItem.ID))
				}
				log.Warn("task failed with SOAP fault",
					zap.String("task_id", taskItem.ID),
					zap.Int("fault_code", faultCode),
					zap.String("fault_msg", faultMsg))
			} else {
				// Task completed successfully - store raw response as result
				resultJSON, _ := json.Marshal(map[string]interface{}{
					"method":       string(method),
					"raw_response": string(body),
				})
				if markErr := h.taskService.MarkTaskCompleted(r.Context(), taskItem.ID, resultJSON); markErr != nil {
					log.Error("mark task completed", zap.Error(markErr), zap.String("task_id", taskItem.ID))
				}
				log.Info("task completed", zap.String("task_id", taskItem.ID), zap.String("method", taskItem.Method))
			}
		}
	}

	// Publish RPC response event for provisioning engine.
	h.publishRPCResponseEvent(r.Context(), deviceSN, method, log)

	// Priority 1: Try new TaskService for next task
	if h.taskService != nil {
		nextTask, err := h.taskService.PopTask(r.Context(), deviceSN)
		if err != nil {
			log.Error("pop task from queue", zap.Error(err))
		} else if nextTask != nil {
			// Generate CWMP ID for this task
			newCWMPID := task.GenerateCWMPID(nextTask.Method)

			// Mark task as sent
			if err := h.taskService.MarkTaskSent(r.Context(), nextTask.ID, newCWMPID); err != nil {
				log.Error("mark task sent", zap.Error(err), zap.String("task_id", nextTask.ID))
			}

			session.State = StateRPCPending
			session.LastRPC = nextTask.Method
			session.UpdatedAt = time.Now()
			h.sessionStore.UpdateByID(r.Context(), sessionID, session)

			cmd := &cmdqueue.Command{
				ID:         nextTask.ID,
				Method:     nextTask.Method,
				Params:     nextTask.Params,
				CommandKey: nextTask.CommandKey,
			}
			respData, err := h.rpcDispatcher.BuildRequest(cmd, newCWMPID)
			if err != nil {
				log.Error("build next RPC request from task", zap.Error(err))
				h.metrics.RPCErrorsTotal.WithLabelValues(nextTask.Method).Inc()
				h.taskService.MarkTaskFailed(r.Context(), nextTask.ID, 0, err.Error())
			} else {
				h.setSessionCookie(w, sessionID)
				h.sendSOAPResponse(w, respData, log)
				return
			}
		}
	}

	// Priority 2: Fallback to legacy command queue (for backward compatibility)
	cmd, err := h.commandQueue.Pop(r.Context(), deviceSN)
	if err != nil {
		log.Error("pop command queue", zap.Error(err))
	}

	if cmd != nil {
		session.State = StateRPCPending
		session.LastRPC = cmd.Method
		session.UpdatedAt = time.Now()
		h.sessionStore.UpdateByID(r.Context(), sessionID, session)

		respData, err := h.rpcDispatcher.BuildRequest(cmd, cwmpID)
		if err != nil {
			log.Error("build next RPC request", zap.Error(err))
			h.metrics.RPCErrorsTotal.WithLabelValues(cmd.Method).Inc()
		} else {
			// Set session cookie in response
			h.setSessionCookie(w, sessionID)
			h.sendSOAPResponse(w, respData, log)
			return
		}
	}

	// No more commands — complete the session.
	h.completeSession(r.Context(), session)

	// Send truly empty response to signal end of session (no body per TR069 spec).
	w.WriteHeader(http.StatusNoContent)
}

// completeSession 完成 TR069 会话，释放所有相关资源。
// 根据 Session.ID 删除 Redis 中的会话数据。
func (h *Handler) completeSession(ctx context.Context, session *Session) {
	// 释放准入槽位
	h.admission.Release()

	// 递减活跃会话计数
	h.metrics.ActiveSessions.Dec()

	if session == nil {
		return
	}

	// 记录会话时长指标
	duration := time.Since(session.StartedAt).Seconds()
	h.metrics.SessionDuration.Observe(duration)

	// 更新会话状态为 COMPLETE
	session.State = StateComplete
	session.UpdatedAt = time.Now()

	// 删除 Cookie-based Session
	if session.ID != "" {
		h.sessionStore.DeleteByID(ctx, session.ID)
	}
}

func (h *Handler) handleTransferComplete(w http.ResponseWriter, r *http.Request, body []byte, log *zap.Logger) {
	// Log complete request XML
	log.Debug("ACS received TransferComplete request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("xml", string(body)),
	)

	tc, cwmpID, err := soap.DecodeTransferComplete(bytes.NewReader(body))
	if err != nil {
		log.Error("decode TransferComplete", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Get deviceSN from Cookie session
	deviceSN := ""
	var sessionID string
	if session, sid := h.getSessionFromCookie(r, log); session != nil {
		deviceSN = session.DeviceSN
		sessionID = sid
	}

	log.Info("TransferComplete received",
		zap.String("device_sn", deviceSN),
		zap.String("command_key", tc.CommandKey))

	// Publish event
	evt, _ := event.NewEvent(event.SubjectDeviceTransferComplete, tc)
	h.eventBus.Publish(r.Context(), event.SubjectDeviceTransferComplete, evt)

	// Send TransferCompleteResponse
	resp, err := soap.RenderResponse(soap.TransferCompleteRespTmpl, soap.InformResponseData{ID: cwmpID})
	if err != nil {
		log.Error("render TransferCompleteResponse", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Debug("ACS sending TransferCompleteResponse",
		zap.String("device_sn", deviceSN),
		zap.String("xml", string(resp)),
	)
	// Set session cookie in response if available
	if sessionID != "" {
		h.setSessionCookie(w, sessionID)
	}
	h.sendSOAPResponse(w, resp, log)
}

// handleAutonomousTransferComplete processes an AutonomousTransferComplete message
// from a CPE device (e.g., PM/MR file upload completion).
func (h *Handler) handleAutonomousTransferComplete(w http.ResponseWriter, r *http.Request, body []byte, log *zap.Logger) {
	// Log complete request XML
	log.Debug("ACS received AutonomousTransferComplete request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("xml", string(body)),
	)

	atc, cwmpID, err := soap.DecodeAutonomousTransferComplete(bytes.NewReader(body))
	if err != nil {
		log.Error("decode AutonomousTransferComplete", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Get deviceSN from Cookie session
	deviceSN := ""
	var sessionID string
	if session, sid := h.getSessionFromCookie(r, log); session != nil {
		deviceSN = session.DeviceSN
		sessionID = sid
	}

	log.Info("AutonomousTransferComplete received",
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
		log.Error("render AutonomousTransferCompleteResponse", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Debug("ACS sending AutonomousTransferCompleteResponse",
		zap.String("device_sn", deviceSN),
		zap.String("xml", string(resp)),
	)
	// Set session cookie in response if available
	if sessionID != "" {
		h.setSessionCookie(w, sessionID)
	}
	h.sendSOAPResponse(w, resp, log)
}

func (h *Handler) publishInformEvents(ctx context.Context, inform *tr069.InformMessage, eventCodes []string, log *zap.Logger) {
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
		log.Error("create event failed",
			zap.Error(err),
			zap.String("device_sn", inform.DeviceId.SerialNumber),
			zap.String("subject", subject))
		return
	}

	if err := h.eventBus.Publish(ctx, subject, evt); err != nil {
		log.Error("publish event failed",
			zap.Error(err),
			zap.String("device_sn", inform.DeviceId.SerialNumber),
			zap.String("subject", subject),
			zap.String("event_id", evt.ID))
		return
	}

	log.Info("event published to bus",
		zap.String("device_sn", inform.DeviceId.SerialNumber),
		zap.String("subject", subject),
		zap.String("event_id", evt.ID),
		zap.Strings("event_codes", eventCodes))
}

func (h *Handler) publishRPCResponseEvent(ctx context.Context, deviceSN string, method soap.RPCMethod, log *zap.Logger) {
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
		log.Error("create RPC response event", zap.Error(err))
		return
	}
	if err := h.eventBus.Publish(ctx, subject, evt); err != nil {
		log.Error("publish RPC response event", zap.Error(err), zap.String("subject", subject))
	}
}

func (h *Handler) sendInformResponse(w http.ResponseWriter, cwmpID string, log *zap.Logger) {
	// CurrentTime is formatted as ISO 8601 dateTime per TR069 spec
	// This helps CPE devices synchronize their clocks with the ACS
	currentTime := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	resp, err := soap.RenderResponse(soap.InformResponseTmpl, soap.InformResponseData{
		ID:          cwmpID,
		CurrentTime: currentTime,
	})
	if err != nil {
		log.Error("render InformResponse", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Debug("ACS sending InformResponse",
		zap.String("cwmp_id", cwmpID),
		zap.String("xml", string(resp)),
	)
	h.sendSOAPResponse(w, resp, log)
}

func (h *Handler) sendSOAPResponse(w http.ResponseWriter, data []byte, log *zap.Logger) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// getSessionFromCookie retrieves the session from the Cookie header.
// Returns nil if no valid session cookie found.
// The log parameter should be a context-aware logger with request_id.
func (h *Handler) getSessionFromCookie(r *http.Request, log *zap.Logger) (*Session, string) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		log.Debug("no session cookie", zap.Error(err), zap.String("remote_addr", r.RemoteAddr))
		return nil, ""
	}

	sessionID := cookie.Value
	session, err := h.sessionStore.GetByID(r.Context(), sessionID)
	if err != nil {
		log.Error("get session by cookie", zap.Error(err), zap.String("session_id", sessionID))
		return nil, sessionID
	}

	return session, sessionID
}

// setSessionCookie sets the SESSION cookie in the response header.
func (h *Handler) setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
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

// generateSessionID generates a UUID-based session ID for Cookie.
func generateSessionID() string {
	uuidBytes := make([]byte, 16)
	rand.Read(uuidBytes)
	// Set version (4) and variant bits per RFC 4122
	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x40
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80
	return hex.EncodeToString(uuidBytes)
}

// detectSOAPFault 检查 SOAP 响应中是否包含 Fault
// 返回: (isFault, faultCode, faultString)
func detectSOAPFault(body []byte) (bool, int, string) {
	// 简单检查 XML 中是否包含 Fault 元素
	bodyStr := string(body)
	if strings.Contains(bodyStr, "<Fault>") || strings.Contains(bodyStr, "<soap:Fault>") || strings.Contains(bodyStr, "<SOAP-ENV:Fault>") {
		// 提取 faultcode 和 faultstring (简化处理)
		faultCode := 0
		faultString := "SOAP fault"

		// 尝试提取 faultcode
		if codeStart := strings.Index(bodyStr, "<faultcode>"); codeStart != -1 {
			codeStart += len("<faultcode>")
			if codeEnd := strings.Index(bodyStr[codeStart:], "</faultcode>"); codeEnd != -1 {
				faultString = strings.TrimSpace(bodyStr[codeStart : codeStart+codeEnd])
			}
		}

		// 尝试提取 faultstring
		if strStart := strings.Index(bodyStr, "<faultstring>"); strStart != -1 {
			strStart += len("<faultstring>")
			if strEnd := strings.Index(bodyStr[strStart:], "</faultstring>"); strEnd != -1 {
				faultString = strings.TrimSpace(bodyStr[strStart : strStart+strEnd])
			}
		}

		return true, faultCode, faultString
	}
	return false, 0, ""
}

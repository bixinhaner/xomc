package acs

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/acs/stun"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/core/middleware"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/soap"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// ConnectionRequester sends a Connection Request to wake a device.
// Used by the Handler for post-session wake when the command queue is not empty.
type ConnectionRequester interface {
	// Send sends a Connection Request. httpURL is the device's HTTP CR URL (may be empty).
	Send(ctx context.Context, deviceSN, httpURL string) error
}


// SessionCookieName is the cookie name for TR069 session ID
const SessionCookieName = "SESSION"

// connSessionEntry tracks a connection-level session binding with creation time for TTL cleanup.
type connSessionEntry struct {
	DeviceSN  string
	CreatedAt time.Time
}

// deviceSessionEntry tracks the active session for a device.
// Used to detect and clean up orphaned sessions when a new Inform arrives
// before the previous session completed (e.g., CPE didn't respond to RPC,
// CPE rebooted mid-session, or PERIODIC timer fired during active session).
type deviceSessionEntry struct {
	SessionID string
	DeviceSN  string
	CreatedAt time.Time
}

// Handler processes TR069/CWMP HTTP requests.
type Handler struct {
	sessionStore            SessionStore
	commandQueue            cmdqueue.CommandQueue // deprecated: use taskService instead
	taskService             TaskService           // new task management service (interface)
	eventBus                event.EventBus
	authenticator           auth.DeviceAuthenticator
	rpcDispatcher           *rpc.Dispatcher
	rateLimiter             *DeviceRateLimiter
	admission               *AdmissionController
	metrics                 *ACSMetrics
	logger                  *zap.Logger
	requestIDPrefix         string                  // prefix for request IDs, e.g., "acs"
	enableTestTaskInjection bool                    // enable random test task injection (for testing only)
	uploadConfig            *appconfig.UploadConfig // upload server configuration for generating upload URLs
	// maxRPCPerSession limits the number of RPC interactions per TR069 session.
	// When reached, the session is gracefully completed; remaining commands stay
	// in the queue and are dispatched in subsequent sessions via post-session wake.
	// 0 means no limit. Recommended: ≤15 to avoid triggering CPE per-session limits.
	maxRPCPerSession int
	// Post-session wake: send Connection Request when session ends with remaining commands.
	connReqSender       ConnectionRequester
	postSessionWakeCfg  appconfig.PostSessionWakeConfig
	redisClient         redis.Cmdable  // for continuous wake counter
	stunStore           *stun.Store    // for caching device STUN addresses from Inform
	connReqURLCache     sync.Map       // deviceSN → ConnectionRequestURL (from Inform)
	// connSessions maps HTTP RemoteAddr → connSessionEntry for connection-level session tracking.
	// Entries are cleaned up on session completion or by the background reaper.
	connSessions sync.Map
	// deviceSessions maps deviceSN → *deviceSessionEntry for device-level session tracking.
	// Used to detect orphaned sessions: when a new Inform arrives, any existing session
	// for the same device is cleaned up via completeSession() before creating a new one.
	deviceSessions sync.Map
}

// sessionRPCLimitReached returns true if the session has reached the per-session RPC limit.
func (h *Handler) sessionRPCLimitReached(session *Session) bool {
	return h.maxRPCPerSession > 0 && session.RPCCount >= h.maxRPCPerSession
}

// startSessionReaper launches a background goroutine that periodically cleans up
// stale sessions from both connSessions and deviceSessions maps.
// For connSessions: releases admission slots and decrements metrics.
// For deviceSessions: loads the orphaned session from Redis, calls completeSession()
// (which triggers postSessionWake if queue has remaining commands), and cleans up.
func (h *Handler) startSessionReaper(interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			// Clean up stale connection-level sessions.
			h.connSessions.Range(func(key, value interface{}) bool {
				entry := value.(connSessionEntry)
				if now.Sub(entry.CreatedAt) > maxAge {
					h.connSessions.Delete(key)
					h.admission.Release()
					h.metrics.ActiveSessions.Dec()
					h.metrics.SessionDuration.Observe(now.Sub(entry.CreatedAt).Seconds())
					h.logger.Warn("reaped stale conn session",
						zap.String("device_sn", entry.DeviceSN),
						zap.String("remote_addr", key.(string)),
						zap.Duration("age", now.Sub(entry.CreatedAt)))
				}
				return true
			})
			// Clean up stale device-level sessions (orphaned sessions).
			// This is the safety net for sessions that were never completed
			// because the CPE didn't respond to an RPC or rebooted mid-session.
			h.deviceSessions.Range(func(key, value interface{}) bool {
				entry := value.(*deviceSessionEntry)
				if now.Sub(entry.CreatedAt) > maxAge {
					h.deviceSessions.Delete(key)
					h.reapOrphanedSession(entry, "reaper")
				}
				return true
			})
		}
	}()
}

// reapOrphanedSession cleans up an orphaned device session.
// It loads the session from Redis (if still exists), calls completeSession to release
// resources and trigger postSessionWake, then logs the cleanup.
func (h *Handler) reapOrphanedSession(entry *deviceSessionEntry, reason string) {
	ctx := context.Background()

	// Try to load the session from Redis to get full session data for completeSession.
	session, err := h.sessionStore.GetByID(ctx, entry.SessionID)
	if err != nil {
		h.logger.Warn("reap orphaned session: failed to load from store",
			zap.String("device_sn", entry.DeviceSN),
			zap.String("session_id", entry.SessionID),
			zap.String("reason", reason),
			zap.Error(err))
	}

	if session != nil {
		h.logger.Info("reap orphaned session: completing",
			zap.String("device_sn", entry.DeviceSN),
			zap.String("session_id", entry.SessionID),
			zap.String("old_state", string(session.State)),
			zap.Duration("age", time.Since(entry.CreatedAt)),
			zap.String("reason", reason))
		h.completeSession(ctx, session)
	} else {
		// Session already expired in Redis (TTL). Still release admission + metrics.
		h.logger.Info("reap orphaned session: session expired in store, releasing resources",
			zap.String("device_sn", entry.DeviceSN),
			zap.String("session_id", entry.SessionID),
			zap.Duration("age", time.Since(entry.CreatedAt)),
			zap.String("reason", reason))
		h.admission.Release()
		h.metrics.ActiveSessions.Dec()
		h.metrics.SessionDuration.Observe(time.Since(entry.CreatedAt).Seconds())
		// Still trigger postSessionWake — even though session is gone,
		// the device may have pending commands in its queue.
		if h.connReqSender != nil && h.postSessionWakeCfg.Enabled && entry.DeviceSN != "" {
			go h.postSessionWake(entry.DeviceSN)
		}
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Generate or propagate Request ID
	requestID := r.Header.Get(middleware.RequestIDHeader)
	if requestID == "" {
		prefix := h.requestIDPrefix
		if prefix == "" {
			prefix = "acs" // default prefix
		}
		requestID = middleware.GenerateRequestIDWithPrefix(prefix)
	}

	// Store Request ID in context for logger and downstream services
	ctx := logger.WithRequestID(r.Context(), requestID)
	r = r.WithContext(ctx)

	// Set Request ID in response header for client correlation
	w.Header().Set(middleware.RequestIDHeader, requestID)

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

	// Log request info (Info level for production visibility)
	log.Info("ACS received request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.Int("body_len", len(body)),
	)

	// Detect method from body
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) == 0 {
		h.handleEmpty(w, r, log)
		return
	}

	method := soap.DetectRPCMethod(body)
	log.Info("ACS detected RPC method", zap.String("method", string(method)))

	// Check for SOAP Fault first (can be in response to any RPC)
	if isFault, faultCode, faultMsg := detectSOAPFault(body); isFault {
		h.handleSOAPFault(w, r, body, faultCode, faultMsg, log)
		return
	}

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
	ctx, span := tracing.StartSpan(r.Context(), tracing.ACSTracerName, "ACS HandleInform",
		attribute.String("acs.remote_addr", r.RemoteAddr),
		attribute.Int("acs.body_len", len(body)),
	)
	defer span.End()
	r = r.WithContext(ctx)

	// Log complete request XML
	log.Debug("ACS received Inform request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.Int("body_len", len(body)),
		zap.String("xml", string(body)),
	)

	// Parse Inform
	inform, cwmpID, err := soap.DecodeInform(bytes.NewReader(body))
	if err != nil {
		tracing.RecordError(span, err)
		log.Error("decode Inform", zap.Error(err),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("xml", string(body)),
		)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	deviceSN := inform.DeviceId.SerialNumber
	span.SetAttributes(
		attribute.String("acs.device_sn", deviceSN),
		attribute.String("acs.oui", inform.DeviceId.OUI),
		attribute.String("acs.product_class", inform.DeviceId.ProductClass),
		attribute.String("acs.cwmp_id", cwmpID),
	)

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

	// Clean up orphaned session for this device (if any).
	// When a CPE sends a new Inform while the previous session was still pending
	// (e.g., CPE didn't respond to an RPC, rebooted, or PERIODIC timer fired),
	// the old session is never completed. We detect and clean it up here to:
	// 1) Release the old admission slot (prevents slot leak)
	// 2) Trigger postSessionWake for any remaining queued commands
	// 3) Keep ActiveSessions metric accurate
	if old, loaded := h.deviceSessions.LoadAndDelete(deviceSN); loaded {
		oldEntry := old.(*deviceSessionEntry)
		log.Info("cleaning orphaned session before new Inform",
			zap.String("device_sn", deviceSN),
			zap.String("old_session_id", oldEntry.SessionID),
			zap.Duration("old_session_age", time.Since(oldEntry.CreatedAt)))
		h.reapOrphanedSession(oldEntry, "new_inform")
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

	// Cache connection request addresses from Inform for post-session wake.
	for _, p := range inform.ParameterList {
		if strings.HasSuffix(p.Name, ".UDPConnectionRequestAddress") && p.Value != "" {
			if h.stunStore != nil {
				if err := h.stunStore.SetFromInform(ctx, deviceSN, p.Value); err != nil {
					log.Warn("cache STUN address from Inform",
						zap.String("device_sn", deviceSN),
						zap.String("udp_addr", p.Value),
						zap.Error(err))
				} else {
					log.Debug("cached STUN address from Inform",
						zap.String("device_sn", deviceSN),
						zap.String("udp_addr", p.Value))
				}
			}
		}
		if strings.HasSuffix(p.Name, ".ConnectionRequestURL") && p.Value != "" {
			h.connReqURLCache.Store(deviceSN, p.Value)
			log.Debug("cached ConnectionRequestURL from Inform",
				zap.String("device_sn", deviceSN),
				zap.String("cr_url", p.Value))
		}
	}

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

	// Register device → session mapping for orphan detection.
	h.deviceSessions.Store(deviceSN, &deviceSessionEntry{
		SessionID: sessionID,
		DeviceSN:  deviceSN,
		CreatedAt: time.Now(),
	})

	// Inject random test tasks for this device (TEST FEATURE)
	// Skip injection for TransferComplete/AutonomousTransferComplete sessions —
	// those sessions have a specific purpose and should not be polluted with test tasks.
	isTC := tr069.HasEvent(inform.Event, tr069.EventTransferComplete)
	isATC := tr069.IsAutonomousTransferComplete(inform.Event)
	log.Info("ACS task injection check",
		zap.String("device_sn", deviceSN),
		zap.Bool("test_task_injection_enabled", h.enableTestTaskInjection),
		zap.Bool("task_service_available", h.taskService != nil),
		zap.Bool("is_tc", isTC),
		zap.Bool("is_atc", isATC),
	)
	if !isTC && !isATC {
		h.injectRandomTestTasks(r, deviceSN, log)
	} else {
		log.Debug("skipped test task injection for TC/ATC session",
			zap.String("device_sn", deviceSN),
			zap.Strings("events", eventCodes))
	}

	// Publish events
	h.publishInformEvents(r.Context(), inform, eventCodes, log)

	// Reset continuous wake counter — device has connected, allow new wake cycle.
	h.resetContinuousWake(r.Context(), deviceSN)

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
	ctx, span := tracing.StartSpan(r.Context(), tracing.ACSTracerName, "ACS HandleEmpty")
	defer span.End()
	r = r.WithContext(ctx)

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
	span.SetAttributes(
		attribute.String("acs.device_sn", deviceSN),
		attribute.String("acs.session_state", string(session.State)),
	)

	log.Info("ACS HandleEmpty started",
		zap.String("device_sn", deviceSN),
		zap.String("session_id", sessionID),
		zap.String("session_state", string(session.State)),
		zap.Bool("task_service_available", h.taskService != nil),
		zap.Bool("command_queue_available", h.commandQueue != nil),
	)

	// Transition from InformReceived → Processing
	if session.State == StateInformReceived {
		session.State = StateProcessing
		session.UpdatedAt = time.Now()
		h.sessionStore.UpdateByID(r.Context(), sessionID, session)
	}

	// Check per-session RPC limit before dispatching more commands.
	// This prevents overwhelming CPEs that have per-session interaction limits.
	if h.sessionRPCLimitReached(session) {
		log.Info("ACS session RPC limit reached, completing session",
			zap.String("device_sn", deviceSN),
			zap.String("session_id", sessionID),
			zap.Int("rpc_count", session.RPCCount),
			zap.Int("max_rpc_per_session", h.maxRPCPerSession),
		)
		// Complete session; post-session wake will trigger a new session
		// for remaining commands in the queue.
		h.completeSession(r.Context(), session)
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusNoContent)
		return
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
			session.RPCCount++
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
		} else {
			log.Info("ACS TaskService.PopTask returned nil",
				zap.String("device_sn", deviceSN),
				zap.String("reason", "no pending tasks in queue"))
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
		session.RPCCount++
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
	log.Info("ACS HandleEmpty no tasks found, completing session",
		zap.String("device_sn", deviceSN),
		zap.String("session_id", sessionID),
		zap.String("session_state", string(session.State)),
	)
	h.completeSession(r.Context(), session)

	// Send truly empty response to signal end of session (no body per TR069 spec).
	// Connection: close tells CPE to close the TCP connection.
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRPCResponse(w http.ResponseWriter, r *http.Request, body []byte, method soap.RPCMethod, log *zap.Logger) {
	ctx, span := tracing.StartSpan(r.Context(), tracing.ACSTracerName, "ACS HandleRPCResponse",
		attribute.String("acs.rpc_method", string(method)),
	)
	defer span.End()
	r = r.WithContext(ctx)

	// Extract CWMP ID from the SOAP response.
	_, cwmpID, _, _ := soap.DetectMethod(bytes.NewReader(body))
	span.SetAttributes(attribute.String("acs.cwmp_id", cwmpID))

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

	// Publish RPC response event for provisioning engine (includes raw body for GPN/GPV processing).
	h.publishRPCResponseEvent(r.Context(), deviceSN, method, body, log)

	// Check per-session RPC limit before dispatching next command.
	if h.sessionRPCLimitReached(session) {
		log.Info("ACS session RPC limit reached after response, completing session",
			zap.String("device_sn", deviceSN),
			zap.String("session_id", sessionID),
			zap.Int("rpc_count", session.RPCCount),
			zap.Int("max_rpc_per_session", h.maxRPCPerSession),
		)
		h.completeSession(r.Context(), session)
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusNoContent)
		return
	}

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
			session.RPCCount++
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
		session.RPCCount++
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
	log.Info("ACS RPC loop done, completing session",
		zap.String("device_sn", deviceSN),
		zap.String("session_id", sessionID),
		zap.String("session_state", string(session.State)),
	)
	h.completeSession(r.Context(), session)

	// Send truly empty response to signal end of session (no body per TR069 spec).
	w.Header().Set("Connection", "close")
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

	// 清除设备→会话映射。仅当映射的 SessionID 与当前 session 一致时才删除，
	// 避免误删已被新 Inform 覆盖的映射。
	if session.DeviceSN != "" && session.ID != "" {
		if v, ok := h.deviceSessions.Load(session.DeviceSN); ok {
			if entry := v.(*deviceSessionEntry); entry.SessionID == session.ID {
				h.deviceSessions.Delete(session.DeviceSN)
			}
		}
	}

	// 异步检查队列并续唤设备
	if h.connReqSender != nil && h.postSessionWakeCfg.Enabled && session.DeviceSN != "" {
		go h.postSessionWake(session.DeviceSN)
	}
}

// postSessionWake checks if the device still has pending commands after a session ends.
// If so, it sends a Connection Request to trigger a new Inform immediately,
// instead of waiting for the device's next periodic Inform (~60s+).
// This dramatically speeds up command queue drain (e.g., parameter discovery).
func (h *Handler) postSessionWake(deviceSN string) {
	// Recover from any panic to prevent silent goroutine death.
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("post-session wake: panic recovered",
				zap.String("device_sn", deviceSN),
				zap.Any("panic", r))
		}
	}()

	h.logger.Info("post-session wake: goroutine started",
		zap.String("device_sn", deviceSN),
		zap.Bool("has_connReqSender", h.connReqSender != nil),
		zap.Bool("enabled", h.postSessionWakeCfg.Enabled),
		zap.Bool("has_taskService", h.taskService != nil),
		zap.Bool("has_commandQueue", h.commandQueue != nil),
		zap.Bool("has_redisClient", h.redisClient != nil))

	if h.connReqSender == nil || !h.postSessionWakeCfg.Enabled {
		return
	}

	ctx := context.Background()

	// Short delay FIRST: let CPE close the previous TCP connection and let the
	// provisioning engine (EventBus subscriber) finish adding new commands to
	// the queue from RPC response events. Without this delay, the queue appears
	// empty because the async subscribers haven't processed yet.
	delay := h.postSessionWakeCfg.DelayAfter
	if delay <= 0 {
		delay = time.Second
	}
	time.Sleep(delay)

	// Check combined queue depth AFTER delay (task queue + legacy command queue).
	var remaining int64
	if h.taskService != nil {
		if n, err := h.taskService.GetQueueLength(ctx, deviceSN); err == nil {
			remaining += n
		}
		h.logger.Debug("post-session wake: task queue check",
			zap.String("device_sn", deviceSN),
			zap.Int64("task_remaining", remaining))
	}
	if h.commandQueue != nil {
		cmdLen := int64(0)
		if n, err := h.commandQueue.Len(ctx, deviceSN); err == nil {
			cmdLen = n
			remaining += n
		}
		h.logger.Debug("post-session wake: cmd queue check",
			zap.String("device_sn", deviceSN),
			zap.Int64("cmd_remaining", cmdLen),
			zap.Int64("total_remaining", remaining))
	}

	if remaining == 0 {
		h.logger.Info("post-session wake: queue empty, skip",
			zap.String("device_sn", deviceSN))
		return // 队列已空，无需续唤
	}

	// Check continuous wake counter (prevent infinite loop).
	maxContinuous := h.postSessionWakeCfg.MaxContinuous
	if maxContinuous <= 0 {
		maxContinuous = 200
	}
	cooldownTTL := h.postSessionWakeCfg.CooldownTTL
	if cooldownTTL <= 0 {
		cooldownTTL = 5 * time.Minute
	}

	counterKey := "acs:continuous_wake:" + deviceSN
	count, err := h.redisClient.Incr(ctx, counterKey).Result()
	if err != nil {
		h.logger.Warn("post-session wake: incr counter failed",
			zap.String("device_sn", deviceSN), zap.Error(err))
		return
	}
	h.redisClient.Expire(ctx, counterKey, cooldownTTL)

	if int(count) > maxContinuous {
		h.logger.Warn("post-session wake: max continuous reached, backing off",
			zap.String("device_sn", deviceSN),
			zap.Int64("remaining", remaining),
			zap.Int64("continuous_count", count),
			zap.Int("max_continuous", maxContinuous))
		return
	}

	// Send Connection Request.
	// Look up cached ConnectionRequestURL for HTTP fallback.
	var httpURL string
	if v, ok := h.connReqURLCache.Load(deviceSN); ok {
		httpURL, _ = v.(string)
	}

	// Use a short timeout to avoid blocking on unreachable devices.
	crCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.connReqSender.Send(crCtx, deviceSN, httpURL); err != nil {
		h.logger.Warn("post-session wake: CR failed",
			zap.String("device_sn", deviceSN),
			zap.Int64("remaining", remaining),
			zap.Error(err))
	} else {
		h.logger.Info("post-session wake: CR sent",
			zap.String("device_sn", deviceSN),
			zap.Int64("remaining", remaining),
			zap.Int64("continuous_count", count))
		h.metrics.PostSessionWakeTotal.Inc()
	}
}

// resetContinuousWake resets the continuous wake counter when a device sends an Inform.
// This allows the counter to restart when the device reconnects.
func (h *Handler) resetContinuousWake(ctx context.Context, deviceSN string) {
	if h.redisClient == nil || !h.postSessionWakeCfg.Enabled {
		return
	}
	h.redisClient.Del(ctx, "acs:continuous_wake:"+deviceSN)
}

// handleSOAPFault handles SOAP Fault responses from CPE.
// It looks up the associated task by CWMP ID and marks it as failed.
func (h *Handler) handleSOAPFault(w http.ResponseWriter, r *http.Request, body []byte, faultCode int, faultMsg string, log *zap.Logger) {
	// Extract CWMP ID from the SOAP Header
	_, cwmpID, _, _ := soap.DetectMethod(bytes.NewReader(body))

	log.Warn("ACS received SOAP Fault",
		zap.String("cwmp_id", cwmpID),
		zap.Int("fault_code", faultCode),
		zap.String("fault_msg", faultMsg),
	)

	// Look up session from cookie for device context
	session, sessionID := h.getSessionFromCookie(r, log)
	if session == nil {
		log.Warn("SOAP Fault without valid session cookie")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Mark task as failed if task service is available
	if h.taskService != nil && cwmpID != "" {
		taskItem, err := h.taskService.GetTaskByCWMPID(r.Context(), cwmpID)
		if err != nil {
			log.Warn("get task by cwmp_id for fault", zap.Error(err), zap.String("cwmp_id", cwmpID))
		} else if taskItem != nil {
			if markErr := h.taskService.MarkTaskFailed(r.Context(), taskItem.ID, faultCode, faultMsg); markErr != nil {
				log.Error("mark task failed on fault", zap.Error(markErr), zap.String("task_id", taskItem.ID))
			}
			log.Info("task marked as failed on SOAP fault",
				zap.String("task_id", taskItem.ID),
				zap.String("method", taskItem.Method),
				zap.String("fault_msg", faultMsg))
		} else {
			log.Warn("no task found for cwmp_id in fault", zap.String("cwmp_id", cwmpID))
		}
	}

	// Check for more tasks in queue
	if h.taskService != nil {
		deviceSN := session.DeviceSN
		nextTask, err := h.taskService.PopTask(r.Context(), deviceSN)
		if err != nil {
			log.Error("pop next task after fault", zap.Error(err))
		} else if nextTask != nil {
			newCWMPID := task.GenerateCWMPID(nextTask.Method)
			if err := h.taskService.MarkTaskSent(r.Context(), nextTask.ID, newCWMPID); err != nil {
				log.Error("mark next task sent", zap.Error(err), zap.String("task_id", nextTask.ID))
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
				log.Error("build next RPC request after fault", zap.Error(err))
				h.metrics.RPCErrorsTotal.WithLabelValues(nextTask.Method).Inc()
				h.taskService.MarkTaskFailed(r.Context(), nextTask.ID, 0, err.Error())
			} else {
				h.setSessionCookie(w, sessionID)
				h.sendSOAPResponse(w, respData, log)
				return
			}
		}
	}

	// No more commands — complete the session
	h.completeSession(r.Context(), session)
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusNoContent)
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
	case tr069.IsBootstrap(inform.Event), tr069.IsBoot(inform.Event):
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

func (h *Handler) publishRPCResponseEvent(ctx context.Context, deviceSN string, method soap.RPCMethod, body []byte, log *zap.Logger) {
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

	// For GPN/GPV responses, parse the SOAP body and include structured data
	// instead of raw XML to avoid NATS message size limits.
	switch method {
	case soap.MethodGetParameterNamesResp:
		paramInfos, _, parseErr := soap.DecodeGetParameterNamesResponse(bytes.NewReader(body))
		if parseErr != nil {
			log.Warn("parse GPN response for event", zap.Error(parseErr))
		} else {
			payload["parameter_infos"] = paramInfos
			log.Info("GPN response parsed for event",
				zap.String("device_sn", deviceSN),
				zap.Int("parameter_count", len(paramInfos)),
			)
		}
	case soap.MethodGetParameterValuesResp:
		paramValues, _, parseErr := soap.DecodeGetParameterValuesResponse(bytes.NewReader(body))
		if parseErr != nil {
			log.Warn("parse GPV response for event", zap.Error(parseErr))
		} else {
			payload["parameter_values"] = paramValues
			log.Info("GPV response parsed for event",
				zap.String("device_sn", deviceSN),
				zap.Int("parameter_count", len(paramValues)),
			)
		}
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
	// TR-069 spec: InformResponse only contains MaxEnvelopes.
	// CurrentTime is not a standard field and has been removed for protocol compliance.
	resp, err := soap.RenderResponse(soap.InformResponseTmpl, soap.InformResponseData{
		ID: cwmpID,
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


// generateSessionID generates a UUID-based session ID for Cookie.
func generateSessionID() string {
	uuidBytes := make([]byte, 16)
	cryptorand.Read(uuidBytes)
	// Set version (4) and variant bits per RFC 4122
	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x40
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80
	return hex.EncodeToString(uuidBytes)
}

// ============================================================================
// TEST FEATURE: Random Task Injection
// ============================================================================
// The following method is for testing purposes only.
// It injects random RPC tasks when a CPE sends an Inform.
// To disable: comment out the call to injectRandomTestTasks in handleInform.
// ============================================================================

// rpcTaskTemplate defines a template for generating random RPC tasks
type rpcTaskTemplate struct {
	method   string
	params   json.RawMessage
	priority int
}

// testRPCTaskTemplates contains all available RPC task templates for random generation
var testRPCTaskTemplates = []rpcTaskTemplate{
	{method: "GetParameterValues", params: json.RawMessage(`{"names":["Device.DeviceInfo.SoftwareVersion","Device.DeviceInfo.HardwareVersion"]}`), priority: 10},
	{method: "GetParameterValues", params: json.RawMessage(`{"names":["Device.X_0000B9_Config.AntennaConfig"]}`), priority: 10},
	{method: "SetParameterValues", params: json.RawMessage(`{"values":[{"name":"Device.X_0000B9_Config.TestParam","value":"test_value_123","type":"xsd:string"}]}`), priority: 10},
	{method: "GetParameterNames", params: json.RawMessage(`{"path":"Device.DeviceInfo.","next_level":false}`), priority: 10},
	{method: "GetParameterNames", params: json.RawMessage(`{"path":"Device.X_0000B9_Config.","next_level":true}`), priority: 10},
	{method: "GetParameterAttributes", params: json.RawMessage(`{"names":["Device.DeviceInfo.SoftwareVersion"]}`), priority: 10},
	{method: "SetParameterAttributes", params: json.RawMessage(`{"attributes":[{"name":"Device.X_Test.Param","notification_change":true,"notification":1}]}`), priority: 10},
	{method: "Download", params: json.RawMessage(`{"file_type":"1 Firmware Upgrade Image","url":"http://acs.example.com/firmware/v1.0.0.bin","file_size":10485760}`), priority: 5},
	{method: "Upload", params: json.RawMessage(`{"file_type":"2 Vendor Log File","url":"http://acs.example.com/upload/logs","delay_seconds":0}`), priority: 10},
	{method: "Reboot", params: json.RawMessage(`{}`), priority: 100}, // High priority but should be last
	{method: "FactoryReset", params: json.RawMessage(`{}`), priority: 100},
}

// injectRandomTestTasks injects random RPC tasks for testing purposes.
// This method generates 3-10 random tasks for the device, plus a fixed PM upload task.
// If Reboot task is generated, it will be moved to the end of the queue.
// NOTE: This is a test feature - disable by setting enableTestTaskInjection to false.
func (h *Handler) injectRandomTestTasks(r *http.Request, deviceSN string, log *zap.Logger) {
	// Skip if test task injection is disabled
	if !h.enableTestTaskInjection {
		return
	}

	// Skip if TaskService is not available
	if h.taskService == nil {
		return
	}

	ctx := r.Context()

	// Generate random number of tasks (3-10)
	numTasks := 3 + rand.Intn(8) // 3 + 0-7 = 3-10

	// Randomly select tasks
	selectedIndices := make(map[int]bool)
	var tasks []rpcTaskTemplate

	for len(tasks) < numTasks {
		idx := rand.Intn(len(testRPCTaskTemplates))
		if selectedIndices[idx] {
			continue
		}
		selectedIndices[idx] = true
		tasks = append(tasks, testRPCTaskTemplates[idx])
	}

	// Fixed: Always inject a PM upload task with real upload URL
	pmUploadTask := h.createPMUploadTask(r, deviceSN)
	if pmUploadTask != nil {
		tasks = append(tasks, *pmUploadTask)
	}

	// Sort tasks: Reboot should be last
	var normalTasks []rpcTaskTemplate
	var rebootTask *rpcTaskTemplate

	for _, t := range tasks {
		if t.method == "Reboot" {
			rebootTask = &t
		} else {
			normalTasks = append(normalTasks, t)
		}
	}

	// Combine: normal tasks first, then reboot (if exists)
	finalTasks := normalTasks
	if rebootTask != nil {
		finalTasks = append(finalTasks, *rebootTask)
	}

	// Create tasks in TaskService
	createdCount := 0
	for _, t := range finalTasks {
		req := &task.CreateTaskRequest{
			DeviceSN:    deviceSN,
			Method:      t.method,
			Params:      t.params,
			Priority:    t.priority,
			Source:      task.TaskSourceSystem,
			Description: fmt.Sprintf("Test task: %s", t.method),
		}

		_, err := h.taskService.CreateTask(ctx, req)
		if err != nil {
			log.Warn("failed to create test task",
				zap.String("device_sn", deviceSN),
				zap.String("method", t.method),
				zap.Error(err))
			continue
		}
		createdCount++
	}

	log.Info("injected random test tasks",
		zap.String("device_sn", deviceSN),
		zap.Int("total_selected", len(tasks)),
		zap.Int("created", createdCount),
		zap.Bool("has_reboot", rebootTask != nil),
		zap.Bool("has_pm_upload", pmUploadTask != nil))
}

// createPMUploadTask creates a PM file upload task with a real upload URL from config.
// If base_url is localhost, it will be replaced with the request's host.
// Returns nil if upload config is not available.
func (h *Handler) createPMUploadTask(r *http.Request, deviceSN string) *rpcTaskTemplate {
	if h.uploadConfig == nil || h.uploadConfig.BaseURL == "" {
		return nil
	}

	// Generate upload URL: {BaseURL}{Path}?fileType=PM&filename={deviceSN}_{timestamp}.xml.gz
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.xml.gz", deviceSN, timestamp)

	// Get base URL, replace localhost with request host if needed
	baseURL := h.uploadConfig.BaseURL
	if isLocalhost(baseURL) {
		// Use the request's host instead of localhost
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, r.Host)
	}

	// Build upload URL
	uploadURL := fmt.Sprintf("%s%s?fileType=PM&filename=%s",
		baseURL,
		h.uploadConfig.Path,
		filename,
	)

	// Create Upload task params
	// TR069 Upload RPC parameters:
	// - FileType: "1 Vendor Configuration File" (1) or "2 Vendor Log File" (2) etc.
	// - URL: The URL where the CPE should upload the file
	// - Username/Password: HTTP Basic Auth credentials (optional)
	params := map[string]interface{}{
		"file_type":      "1 Vendor Configuration File", // PM data as vendor config
		"url":            uploadURL,
		"delay_seconds":  0,
	}
	if h.uploadConfig.Username != "" {
		params["username"] = h.uploadConfig.Username
		params["password"] = h.uploadConfig.Password
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil
	}

	return &rpcTaskTemplate{
		method:   "Upload",
		params:   json.RawMessage(paramsJSON),
		priority: 10,
	}
}

// ============================================================================
// END TEST FEATURE
// ============================================================================

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

// isLocalhost checks if a URL contains localhost or 127.0.0.1
func isLocalhost(url string) bool {
	return strings.Contains(url, "localhost") ||
		strings.Contains(url, "127.0.0.1") ||
		strings.Contains(url, "[::1]") ||
		strings.Contains(url, "::1")
}

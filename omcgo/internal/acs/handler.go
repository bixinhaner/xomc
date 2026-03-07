package acs

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/common/event"
	"github.com/omcgo/omcgo/pkg/soap"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// Handler processes TR069/CWMP HTTP requests.
type Handler struct {
	sessionStore  SessionStore
	commandQueue  cmdqueue.CommandQueue
	eventBus      event.EventBus
	authenticator auth.DeviceAuthenticator
	rpcDispatcher *rpc.Dispatcher
	rateLimiter   *DeviceRateLimiter
	admission     *AdmissionController
	metrics       *ACSMetrics
	logger        *zap.Logger
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	case soap.MethodGetParameterValuesResp,
		soap.MethodSetParameterValuesResp,
		soap.MethodGetParameterNamesResp,
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

func (h *Handler) handleInform(w http.ResponseWriter, r *http.Request, body []byte) {
	// Parse Inform
	inform, cwmpID, err := soap.DecodeInform(bytes.NewReader(body))
	if err != nil {
		h.logger.Error("decode Inform", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	deviceSN := inform.DeviceId.SerialNumber

	// Rate limiting
	if !h.rateLimiter.Allow(deviceSN) {
		h.logger.Warn("rate limited", zap.String("device_sn", deviceSN))
		http.Error(w, "Too Many Requests", http.StatusServiceUnavailable)
		return
	}

	// Admission control
	if !h.admission.Acquire() {
		h.logger.Warn("admission denied", zap.String("device_sn", deviceSN))
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	defer h.admission.Release()

	h.metrics.ActiveSessions.Inc()
	defer h.metrics.ActiveSessions.Dec()

	// Record metrics
	eventCodes := tr069.EventCodes(inform.Event)
	for _, code := range eventCodes {
		h.metrics.InformTotal.WithLabelValues(code).Inc()
	}

	h.logger.Info("Inform received",
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

	// Publish events
	h.publishInformEvents(r.Context(), inform, eventCodes)

	// Check command queue for pending commands
	cmd, err := h.commandQueue.Peek(r.Context(), deviceSN)
	if err != nil {
		h.logger.Error("peek command queue", zap.Error(err))
	}

	if cmd != nil {
		// Pop and send the command after InformResponse
		cmd, _ = h.commandQueue.Pop(r.Context(), deviceSN)
		if cmd != nil {
			session.State = StateRPCPending
			session.LastRPC = cmd.Method
			h.sessionStore.Update(r.Context(), deviceSN, session)

			respData, err := h.rpcDispatcher.BuildRequest(cmd, cwmpID)
			if err != nil {
				h.logger.Error("build RPC request", zap.Error(err))
				h.sendInformResponse(w, cwmpID)
				return
			}
			h.sendSOAPResponse(w, respData)
			return
		}
	}

	// No pending commands, send InformResponse and close session
	h.sendInformResponse(w, cwmpID)
	session.State = StateComplete
	h.sessionStore.Update(r.Context(), deviceSN, session)
}

func (h *Handler) handleRPCResponse(w http.ResponseWriter, r *http.Request, body []byte, method soap.RPCMethod) {
	// Extract CWMP ID from the SOAP response.
	_, cwmpID, _, _ := soap.DetectMethod(bytes.NewReader(body))

	h.logger.Info("RPC response received",
		zap.String("method", string(method)),
		zap.String("cwmp_id", cwmpID),
	)

	// Find the device SN from the connection context.
	// In TR069, the session persists over the same HTTP connection from Inform.
	deviceSN := r.Header.Get("X-Device-SN")

	if deviceSN != "" {
		session, _ := h.sessionStore.Get(r.Context(), deviceSN)
		if session != nil {
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
				h.sessionStore.Update(r.Context(), deviceSN, session)

				respData, err := h.rpcDispatcher.BuildRequest(cmd, cwmpID)
				if err != nil {
					h.logger.Error("build next RPC request", zap.Error(err))
				} else {
					h.sendSOAPResponse(w, respData)
					return
				}
			}

			// No more commands — complete the session.
			session.State = StateComplete
			h.sessionStore.Update(r.Context(), deviceSN, session)
		}
	}

	// Send empty response to signal end of session.
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	resp, _ := soap.RenderResponse(soap.EmptyResponseTmpl, nil)
	w.Write(resp)
}

func (h *Handler) handleTransferComplete(w http.ResponseWriter, r *http.Request, body []byte) {
	tc, cwmpID, err := soap.DecodeTransferComplete(bytes.NewReader(body))
	if err != nil {
		h.logger.Error("decode TransferComplete", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	h.logger.Info("TransferComplete received",
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

func (h *Handler) handleEmpty(w http.ResponseWriter, r *http.Request) {
	// Empty POST indicates the CPE has no more data to send.
	// This is the session completion signal.
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusNoContent)
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

	// Determine primary subject based on event codes
	var subject string
	switch {
	case tr069.IsBootstrap(inform.Event):
		subject = event.SubjectDeviceBootstrap
	case tr069.IsPeriodic(inform.Event):
		subject = event.SubjectDevicePeriodic
	case tr069.IsValueChange(inform.Event):
		subject = event.SubjectDeviceValueChange
	case tr069.HasEvent(inform.Event, tr069.EventTransferComplete):
		subject = event.SubjectDeviceTransferComplete
	default:
		subject = event.SubjectDevicePeriodic
	}

	evt, err := event.NewEvent(subject, payload)
	if err != nil {
		h.logger.Error("create event", zap.Error(err))
		return
	}

	if err := h.eventBus.Publish(ctx, subject, evt); err != nil {
		h.logger.Error("publish event", zap.Error(err), zap.String("subject", subject))
	}
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

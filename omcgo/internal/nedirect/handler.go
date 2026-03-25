package nedirect

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"go.uber.org/zap"
)

// RegisterRequest represents a NE direct registration request from a device.
type RegisterRequest struct {
	SerialNumber string `json:"serial_number"`
	IPAddress    string `json:"ip_address"`
	Manufacturer string `json:"manufacturer"`
	ModelName    string `json:"model_name"`
}

// ConfigRequest represents a NE direct configuration query.
type ConfigRequest struct {
	SerialNumber string `json:"serial_number"`
}

// FaultReport represents a NE direct fault report from a device.
type FaultReport struct {
	SerialNumber string `json:"serial_number"`
	AlarmCode    string `json:"alarm_code"`
	Severity     int    `json:"severity"`
	Description  string `json:"description"`
}

// ConnectRequest represents a request to establish a NE direct session.
type ConnectRequest struct {
	DeviceSN string `json:"device_sn"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// SendCommandRequest represents a request to send a command through a session.
type SendCommandRequest struct {
	SessionID string `json:"session_id"`
	Command   string `json:"command"`
}

// Handler provides net/http stdlib handlers for NE Direct connections.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new NE Direct Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers NE Direct HTTP routes on a standard ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/nedirect/register", h.HandleRegister)
	mux.HandleFunc("/nedirect/config", h.HandleConfig)
	mux.HandleFunc("/nedirect/status", h.HandleStatus)
	mux.HandleFunc("/nedirect/fault", h.HandleFault)
	mux.HandleFunc("/nedirect/connect", h.HandleConnect)
	mux.HandleFunc("/nedirect/disconnect", h.HandleDisconnect)
	mux.HandleFunc("/nedirect/command", h.HandleCommand)
	mux.HandleFunc("/nedirect/sessions", h.HandleListSessions)
}

// HandleRegister handles NE direct device registration.
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	if req.SerialNumber == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "serial_number is required"})
		return
	}

	existing, isNew, err := h.service.RegisterDevice(r.Context(), req)
	if err != nil {
		logger.L(r.Context()).Error("ne-direct register failed", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if !isNew {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "device already registered",
			"device_id": existing.ID.String(),
			"status":    string(existing.Status),
		})
	} else {
		writeJSON(w, http.StatusAccepted, map[string]string{
			"message": "registration accepted",
		})
	}
}

// HandleConfig handles NE direct configuration query.
func (h *Handler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	if req.SerialNumber == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "serial_number is required"})
		return
	}

	params, err := h.service.GetDeviceConfig(r.Context(), req.SerialNumber)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "device not found"})
		return
	}

	dev, _ := h.service.GetDeviceStatus(r.Context(), req.SerialNumber)
	deviceID := ""
	if dev != nil {
		deviceID = dev.ID.String()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"device_id":  deviceID,
		"parameters": params,
		"total":      len(params),
	})
}

// HandleStatus handles NE direct device status query.
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	sn := r.URL.Query().Get("serial_number")
	if sn == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "serial_number is required"})
		return
	}

	dev, err := h.service.GetDeviceStatus(r.Context(), sn)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "device not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"device_id":        dev.ID.String(),
		"serial_number":    dev.SerialNumber,
		"status":           string(dev.Status),
		"firmware_version": dev.FirmwareVersion,
		"ip_address":       dev.IPAddress,
		"last_inform_at":   dev.LastInformAt,
	})
}

// HandleFault handles NE direct fault reports.
func (h *Handler) HandleFault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req FaultReport
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	if req.SerialNumber == "" || req.AlarmCode == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "serial_number and alarm_code are required"})
		return
	}

	if err := h.service.ReportFault(r.Context(), req); err != nil {
		logger.L(r.Context()).Error("ne-direct fault processing failed",
			zap.String("serial_number", req.SerialNumber),
			zap.String("alarm_code", req.AlarmCode),
			zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fault processing failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "fault reported"})
}

// HandleConnect handles establishing a NE direct session.
func (h *Handler) HandleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	if req.DeviceSN == "" || req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "device_sn and user_id are required"})
		return
	}

	session, err := h.service.Connect(r.Context(), req.DeviceSN, req.UserID, req.Username)
	if err != nil {
		logger.L(r.Context()).Error("ne-direct connect failed", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "connect failed"})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// HandleDisconnect handles closing a NE direct session.
func (h *Handler) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var body struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	sessionID, err := uuid.Parse(body.SessionID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session_id"})
		return
	}

	if err := h.service.Disconnect(r.Context(), sessionID); err != nil {
		logger.L(r.Context()).Error("ne-direct disconnect failed", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "disconnect failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "disconnected"})
}

// HandleCommand handles sending a CLI/MML command through a NE direct session.
func (h *Handler) HandleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req SendCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session_id"})
		return
	}

	if req.Command == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "command is required"})
		return
	}

	cmd, err := h.service.SendCommand(r.Context(), sessionID, req.Command)
	if err != nil {
		logger.L(r.Context()).Error("ne-direct command failed", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "command failed"})
		return
	}

	writeJSON(w, http.StatusOK, cmd)
}

// HandleListSessions handles listing NE direct sessions.
func (h *Handler) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	filter := SessionFilter{}
	if sn := r.URL.Query().Get("device_sn"); sn != "" {
		filter.DeviceSN = &sn
	}
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		filter.UserID = &uid
	}
	if status := r.URL.Query().Get("status"); status != "" {
		s := SessionStatus(status)
		filter.Status = &s
	}
	filter.ListRequest.Page = 1
	filter.ListRequest.PageSize = 20

	result, err := h.service.ListSessions(r.Context(), filter)
	if err != nil {
		logger.L(r.Context()).Error("ne-direct list sessions failed", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

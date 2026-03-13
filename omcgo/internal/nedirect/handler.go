package nedirect

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
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

// Handler provides net/http stdlib handlers for NE Direct connections.
type Handler struct {
	deviceService *device.DeviceService
	alarmEngine   *alarm.AlarmEngine
	eventBus      event.EventBus
	logger        *zap.Logger
}

// NewHandler creates a new NE Direct Handler.
func NewHandler(
	deviceService *device.DeviceService,
	alarmEngine *alarm.AlarmEngine,
	eventBus event.EventBus,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		deviceService: deviceService,
		alarmEngine:   alarmEngine,
		eventBus:      eventBus,
		logger:        logger,
	}
}

// RegisterRoutes registers NE Direct HTTP routes on a standard ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/nedirect/register", h.HandleRegister)
	mux.HandleFunc("/nedirect/config", h.HandleConfig)
	mux.HandleFunc("/nedirect/status", h.HandleStatus)
	mux.HandleFunc("/nedirect/fault", h.HandleFault)
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

	// Check if device already exists
	ctx := r.Context()
	existing, err := h.deviceService.GetBySerialNumber(ctx, req.SerialNumber)
	if err != nil {
		h.logger.Error("ne-direct register lookup failed", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if existing != nil {
		// Device already registered, return its info
		h.logger.Info("ne-direct device already registered",
			zap.String("serial_number", req.SerialNumber))
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "device already registered",
			"device_id": existing.ID.String(),
			"status":    string(existing.Status),
		})
	} else {
		// Publish registration event for the provisioning pipeline to handle
		h.publishEvent(ctx, event.SubjectNEDirectRegister, req)
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

	ctx := r.Context()
	dev, err := h.deviceService.GetBySerialNumber(ctx, req.SerialNumber)
	if err != nil || dev == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "device not found"})
		return
	}

	params, err := h.deviceService.GetDeviceParameters(ctx, dev.ID)
	if err != nil {
		h.logger.Error("ne-direct config query failed", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"device_id":  dev.ID.String(),
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

	dev, err := h.deviceService.GetBySerialNumber(r.Context(), sn)
	if err != nil || dev == nil {
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

	ctx := r.Context()

	// Build alarm and process through alarm engine
	alarmObj := &model.Alarm{
		DeviceSN:    req.SerialNumber,
		Carrier:     model.CarrierCMCC, // NE Direct is CMCC-specific
		Severity:    model.AlarmSeverity(req.Severity),
		AlarmType:   "ne_direct",
		AlarmCode:   req.AlarmCode,
		Description: req.Description,
		Status:      model.AlarmActive,
		RaisedAt:    time.Now(),
	}

	// Lookup device to get device_id
	dev, err := h.deviceService.GetBySerialNumber(ctx, req.SerialNumber)
	if err == nil && dev != nil {
		alarmObj.DeviceID = dev.ID
	}

	if err := h.alarmEngine.Process(ctx, alarmObj); err != nil {
		h.logger.Error("ne-direct fault processing failed",
			zap.String("serial_number", req.SerialNumber),
			zap.String("alarm_code", req.AlarmCode),
			zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fault processing failed"})
		return
	}

	// Publish NE Direct fault event
	h.publishEvent(ctx, event.SubjectNEDirectFault, req)

	h.logger.Info("ne-direct fault reported",
		zap.String("serial_number", req.SerialNumber),
		zap.String("alarm_code", req.AlarmCode))

	writeJSON(w, http.StatusOK, map[string]string{"message": "fault reported"})
}

func (h *Handler) publishEvent(ctx context.Context, subject string, payload interface{}) {
	evt, err := event.NewEvent(subject, payload)
	if err != nil {
		h.logger.Warn("create event failed", zap.String("subject", subject), zap.Error(err))
		return
	}
	if err := h.eventBus.Publish(ctx, subject, evt); err != nil {
		h.logger.Warn("publish event failed", zap.String("subject", subject), zap.Error(err))
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

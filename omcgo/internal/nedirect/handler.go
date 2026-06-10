package nedirect

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"go.uber.org/zap"
)

// snFromQuery 返回一个从 URL query 提取设备 SN 的函数（用于 per-device 限流 / 审计）。
func snFromQuery(param string) func(*http.Request) string {
	return func(r *http.Request) string {
		return r.URL.Query().Get(param)
	}
}

// snFromJSONBody 返回一个从 JSON 请求体提取设备 SN 字段的函数。
//
// 读 body 后会用 bytes.Reader 复原 r.Body，确保下游业务 handler 仍能正常解码 ——
// 这是限流/审计中间件需要窥探 body 的标准做法。读失败 / 字段缺失返回空串
// （此时中间件跳过 per-device 限流，不影响主流程）。
func snFromJSONBody(field string) func(*http.Request) string {
	return func(r *http.Request) string {
		if r.Body == nil {
			return ""
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MiB 上限，防滥用
		_ = r.Body.Close()
		// 复原 body 供下游 handler 解码。
		r.Body = io.NopCloser(bytes.NewReader(raw))
		if err != nil {
			return ""
		}
		var m map[string]any
		if json.Unmarshal(raw, &m) != nil {
			return ""
		}
		if v, ok := m[field].(string); ok {
			return v
		}
		return ""
	}
}

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
	mw      *Middleware
	logger  *zap.Logger
}

// NewHandler creates a new NE Direct Handler.
//
// mw 为安全中间件（认证 + 限流 + 审计）。允许为 nil（dev/test 退化为无防护），
// 但生产装配（cmd/app/provider）始终注入 —— 见 NewMiddleware 调用点。
func NewHandler(service *Service, mw *Middleware, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		mw:      mw,
		logger:  logger,
	}
}

// wrap 按是否注入了中间件，决定给 handler 套上「认证 + 限流 + 审计」还是直接挂载。
// endpoint 为审计/限流用的端点标识；deviceSNFn 从请求提取设备 SN（per-device 限流 + 审计）。
func (h *Handler) wrap(endpoint string, deviceSNFn func(*http.Request) string, next http.HandlerFunc) http.HandlerFunc {
	if h.mw == nil {
		return next
	}
	return h.mw.Wrap(endpoint, deviceSNFn, next)
}

// RegisterRoutes registers NE Direct HTTP routes on a standard ServeMux.
// 所有端点都经 h.wrap 套上强制认证 + per-endpoint/per-device 限流 + 审计日志。
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/nedirect/register", h.wrap("register", snFromJSONBody("serial_number"), h.HandleRegister))
	mux.HandleFunc("/nedirect/config", h.wrap("config", snFromJSONBody("serial_number"), h.HandleConfig))
	mux.HandleFunc("/nedirect/status", h.wrap("status", snFromQuery("serial_number"), h.HandleStatus))
	mux.HandleFunc("/nedirect/fault", h.wrap("fault", snFromJSONBody("serial_number"), h.HandleFault))
	mux.HandleFunc("/nedirect/connect", h.wrap("connect", snFromJSONBody("device_sn"), h.HandleConnect))
	mux.HandleFunc("/nedirect/disconnect", h.wrap("disconnect", nil, h.HandleDisconnect))
	mux.HandleFunc("/nedirect/command", h.wrap("command", nil, h.HandleCommand))
	mux.HandleFunc("/nedirect/sessions", h.wrap("sessions", snFromQuery("device_sn"), h.HandleListSessions))
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

	if req.DeviceSN == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "device_sn is required"})
		return
	}

	// 会话归属取自已认证身份，覆盖请求体里的 user_id/username —— 杜绝伪造身份建会话。
	// 中间件已注入 Principal；principal 为 nil 仅出现在未挂中间件的 dev/test，退化为
	// 沿用请求体（与旧行为兼容）。
	userID, username := req.UserID, req.Username
	if p, ok := PrincipalFromContext(r.Context()); ok && p != nil {
		userID = p.UserID.String()
		username = p.Username
	}
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id is required"})
		return
	}

	session, err := h.service.Connect(r.Context(), req.DeviceSN, userID, username)
	if err != nil {
		logger.L(r.Context()).Error("ne-direct connect failed", zap.Error(err))
		writeJSON(w, serviceErrStatus(err), map[string]string{"error": "connect failed"})
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

	principal, _ := PrincipalFromContext(r.Context())
	if err := h.service.Disconnect(r.Context(), principal, sessionID); err != nil {
		logger.L(r.Context()).Warn("ne-direct disconnect failed", zap.Error(err))
		writeJSON(w, serviceErrStatus(err), map[string]string{"error": "disconnect failed"})
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

	principal, _ := PrincipalFromContext(r.Context())
	cmd, err := h.service.SendCommand(r.Context(), principal, sessionID, req.Command)
	if err != nil {
		logger.L(r.Context()).Warn("ne-direct command failed", zap.Error(err))
		writeJSON(w, serviceErrStatus(err), map[string]string{"error": "command failed"})
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

	principal, _ := PrincipalFromContext(r.Context())
	result, err := h.service.ListSessions(r.Context(), principal, filter)
	if err != nil {
		logger.L(r.Context()).Error("ne-direct list sessions failed", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// serviceErrStatus 把 service 层错误映射到 HTTP 状态码。IDOR 越权（ErrSessionOwnership）
// 与 ErrForbidden 映射为 403；ErrNotFound 映射 404；ErrInvalidInput 映射 400；
// 其余视为 500 内部错误（不向客户端暴露细节）。
func serviceErrStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, ErrSessionOwnership), errors.Is(err, commonerrors.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, commonerrors.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, commonerrors.ErrInvalidInput):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

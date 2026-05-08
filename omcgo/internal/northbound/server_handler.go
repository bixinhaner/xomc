package northbound

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// ServerHandler 北向 OSS 主备服务器 HTTP handler。
type ServerHandler struct {
	svc    *ServerService
	logger *zap.Logger
}

// NewServerHandler 构造 handler。
func NewServerHandler(svc *ServerService, logger *zap.Logger) *ServerHandler {
	return &ServerHandler{svc: svc, logger: logger}
}

// ListServers 处理 GET /admin/northbound/servers。
//
// 返回主备两行 + total。前端 NorthboundSettings 用此回显主备 IP/端口/激活状态。
func (h *ServerHandler) ListServers(c *gin.Context) {
	servers, err := h.svc.List(c.Request.Context())
	if err != nil {
		h.logger.Error("list northbound servers failed", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, "list northbound servers failed")
		return
	}
	response.OK(c, gin.H{"items": servers, "total": len(servers)})
}

// SwitchActive 处理 PUT /admin/northbound/servers/active。
//
// 请求体：{"role": "primary"|"standby"}。已激活 role 视为 noop（200 OK）。
// 不存在的 role 返回 400。
func (h *ServerHandler) SwitchActive(c *gin.Context) {
	var req SwitchActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if !req.Role.IsValid() {
		response.Fail(c, http.StatusBadRequest, "role must be 'primary' or 'standby'")
		return
	}

	if err := h.svc.SetActive(c.Request.Context(), req.Role); err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "server with given role not found")
			return
		}
		h.logger.Error("switch active northbound server failed",
			zap.String("role", string(req.Role)), zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, "switch active server failed")
		return
	}
	response.OKWithMsg(c, gin.H{"role": req.Role}, "active northbound server switched")
}

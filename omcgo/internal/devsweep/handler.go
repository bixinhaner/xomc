package devsweep

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler 是 devsweep 的 REST handler。
//
// 路由：POST /api/v1/devices/:id/sweep-paths（URL 段实际承载设备 SN；
// gin 路由树要求 /api/v1/devices/ 之下同位置参数名必须统一为 :id）
// 鉴权：与 device 路由组共享 RequireAPIPermission（router.go 挂在 permGroup("devices") 下）
//
// 设计：omcctl 子命令是 thin client，所有业务逻辑在 Service.Run 里。
type Handler struct {
	svc *Service
}

// NewHandler 构造 Handler。svc 必填，不允许 nil。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 挂到给定 router group 下（必须是已带 auth 中间件的组）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/devices/:id/sweep-paths", h.SweepPaths)
}

// sweepRequest 是 SweepPaths 的 JSON body。
//
// 字段一一对应 omcctl device sweep-paths 子命令的 CLI flag，
// 见 cmd/omcctl/device_sweep.go。
type sweepRequest struct {
	Apply                 bool    `json:"apply"`
	Prefix                string  `json:"prefix"`
	BatchSize             int     `json:"batch_size"`
	RPCTimeoutSeconds     int     `json:"rpc_timeout_seconds"`
	RPCRate               float64 `json:"rpc_rate"`
	Operator              string  `json:"operator"`
	ConfirmParamModelWide bool    `json:"confirm_param_model_wide"`
	Force                 bool    `json:"force"`
	Verbose               bool    `json:"verbose"`
}

// SweepPaths handles POST /devices/:id/sweep-paths.
//
// 路径段虽然命名为 :id（gin 路由树约束），实际承载的是设备 SN，
// 与 omcctl device sweep-paths <SN> 客户端契约一致。
//
// 响应信封：success/abort 都用 200 + ret=1，错误细节进 data.error_code。
// 真正 4xx/5xx 只在解析失败/DB 错误时返回。
func (h *Handler) SweepPaths(c *gin.Context) {
	sn := c.Param("id")
	if sn == "" {
		response.Fail(c, http.StatusBadRequest, "missing device sn in path")
		return
	}

	var req sweepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 空 body 允许 — 等价 dry-run + 所有默认值
		req = sweepRequest{}
	}

	opts := Options{
		DeviceSN:              sn,
		Prefix:                req.Prefix,
		BatchSize:             req.BatchSize,
		RPCTimeout:            time.Duration(req.RPCTimeoutSeconds) * time.Second,
		RPCRate:               req.RPCRate,
		Apply:                 req.Apply,
		Force:                 req.Force,
		ConfirmParamModelWide: req.ConfirmParamModelWide,
		Operator:              req.Operator,
		Safety:                DefaultSafetyConfig(),
	}
	if opts.Operator == "" {
		opts.Operator = "api"
	}

	// 用较长的请求级 timeout 兜底（探测可能跑几分钟）。
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Minute)
	defer cancel()

	if !req.Verbose {
		// 默认不返回 per_path（量大）
		defer func(orig []ProbeRecord) {
			// no-op; cleanup happens after Run
			_ = orig
		}(nil)
	}

	res, err := h.svc.Run(ctx, opts)
	if err != nil {
		// res 可能部分填好（含 ErrorCode/Aborted） — 仍按 200 + 数据返回，让 CLI
		// 拿到 ErrorCode 决定 exit code。系统级错误（无 res 字段）才走 4xx/5xx。
		if res == nil {
			response.Fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		// 对 device_not_found / 离线 / 安全门，返回 200 + envelope，data 含详情
		if res.Aborted {
			res.PerPath = nil
			if !req.Verbose {
				res.PerPath = nil
			}
			response.OKWithMsg(c, res, err.Error())
			return
		}
		// device_service 之类错（DB 挂了）走 500
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	if !req.Verbose {
		res.PerPath = nil
	}
	response.OK(c, res)
}

package northbound

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/ratelimit"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/northbound/pageconfig"
	"github.com/omcgo/omcgo/internal/northbound/push"
)

// Router registers northbound/OSS API routes.
type Router struct {
	svc               *NorthboundService
	pmHandler         *PMHandler
	alarmHandler      *AlarmHandler
	configHandler     *ConfigHandler
	pageConfigHandler *pageconfig.Handler
	pageConfigService *pageconfig.Service
	serverHandler     *ServerHandler                // 主备服务器配置 + 切换（system/config 北向设置）
	outboxRepo        push.OutboxRepository         // may be nil if outbox is not configured
	scoper            *Scoper                       // 多租户数据隔离；nil 时退化为不隔离（dev/test）
	rateLimiter       *ratelimit.FixedWindowLimiter // per-endpoint 限流；nil 时不限流
	deviceService     northboundDeviceService       // optional device business facade for legacy northbound API overlap
	taskService       northboundTaskService         // optional task facade for legacy async job overlap
	regService        northboundRegistrationService // optional device pre-registration facade
	groupService      northboundDeviceGroupService  // optional current device-group facade
	transferService   northboundTransferTaskService // optional UFTE facade for station log collection
}

// NewRouter creates a new northbound Router.
//
// serverHandler 通过 SetServerService 在 ServerService 装配完成后注入。
func NewRouter(svc *NorthboundService) *Router {
	logger := svc.logger
	pageConfigService := pageconfig.NewService(pageconfig.NewDefaultCatalog())
	return &Router{
		svc:               svc,
		pmHandler:         NewPMHandler(svc, logger),
		alarmHandler:      NewAlarmHandler(svc, logger),
		configHandler:     NewConfigHandler(svc, logger),
		pageConfigHandler: pageconfig.NewHandler(pageConfigService, logger.Named("page-config")),
		pageConfigService: pageConfigService,
	}
}

// SetServerService 注入主备服务器 service（创建 ServerHandler）。
// 在 provider/modules.go 完成 northbound_servers repo + service 装配后调用。
func (r *Router) SetServerService(svc *ServerService) {
	r.serverHandler = NewServerHandler(svc, r.svc.logger)
}

// SetPageConfigRepository wires page-config persistence. Without this injection
// the page-config handler serves the built-in read-only catalog for tests.
func (r *Router) SetPageConfigRepository(repo pageconfig.Repository) {
	r.SetPageConfigService(pageconfig.NewServiceWithRepository(pageconfig.NewDefaultCatalog(), repo))
}

// SetPageConfigService wires page-config service. It is used by app provider
// so the HTTP handler and background scheduler share one service/repository.
func (r *Router) SetPageConfigService(svc *pageconfig.Service) {
	r.pageConfigService = svc
	r.pageConfigHandler = pageconfig.NewHandler(
		svc,
		r.svc.logger.Named("page-config"),
	)
}

// SetOutboxRepo sets the outbox repository for dead letter queue endpoints.
func (r *Router) SetOutboxRepo(repo push.OutboxRepository) {
	r.outboxRepo = repo
}

// SetScoper 注入多租户数据权限 Scoper（在 provider 装配 PermService + DeviceService 后调用）。
// 同时下传给 PM/alarm/config 子 handler，使按设备维度的导出走同一套隔离。
func (r *Router) SetScoper(s *Scoper) {
	r.scoper = s
	r.pmHandler.scoper = s
	r.alarmHandler.scoper = s
	r.configHandler.scoper = s
}

// SetRateLimiter 注入 per-endpoint Redis 固定窗口限流器。
func (r *Router) SetRateLimiter(l *ratelimit.FixedWindowLimiter) {
	r.rateLimiter = l
}

// SetDeviceTaskServices wires current xomc device/task capabilities into the
// northbound facade. These routes are thin wrappers so the original device and
// task APIs keep their behavior unchanged.
func (r *Router) SetDeviceTaskServices(deviceSvc northboundDeviceService, taskSvc northboundTaskService) {
	r.deviceService = deviceSvc
	r.taskService = taskSvc
}

// SetDeviceRegistrationService wires current device pre-registration into the
// legacy-shaped northbound facade.
func (r *Router) SetDeviceRegistrationService(svc northboundRegistrationService) {
	r.regService = svc
}

// SetDeviceGroupService wires current topology device-group management into the
// legacy-shaped northbound facade.
func (r *Router) SetDeviceGroupService(svc northboundDeviceGroupService) {
	r.groupService = svc
}

// SetTransferTaskService wires UFTE file-transfer tasks into the northbound
// facade. It is used for real station running/fault log collection.
func (r *Router) SetTransferTaskService(svc northboundTransferTaskService) {
	r.transferService = svc
}

// rateLimit 返回一个按端点名限流的 Gin 中间件。endpoint 为限流 key（端点标识）。
// limiter 未注入时为 no-op。超限返回 429。
func (r *Router) rateLimit(endpoint string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if r.rateLimiter == nil {
			c.Next()
			return
		}
		allowed, _, err := r.rateLimiter.Allow(c.Request.Context(), endpoint)
		if err != nil {
			// fail-open：Redis 抖动不拒外部接口，仅靠日志暴露。
			c.Next()
			return
		}
		if !allowed {
			response.Fail(c, http.StatusTooManyRequests, "too many requests, please retry later")
			return
		}
		c.Next()
	}
}

func (r *Router) pageConfiguredAPI(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.requirePageConfiguredAPI(c, apiKey) {
			return
		}
		if !r.requirePageConfiguredAPIClient(c, apiKey) {
			return
		}
		c.Next()
	}
}

func (r *Router) requirePageConfiguredAPI(c *gin.Context, apiKey string) bool {
	if r.pageConfigService == nil {
		return true
	}
	enabled, err := r.pageConfigService.IsAPIConfigEnabled(c.Request.Context(), apiKey)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "northbound API page-config check failed")
		return false
	}
	if !enabled {
		response.Fail(c, http.StatusForbidden, "northbound API is disabled by page-config")
		return false
	}
	return true
}

func (r *Router) requirePageConfiguredAPIClient(c *gin.Context, apiKey string) bool {
	if r.pageConfigService == nil {
		return true
	}
	client, err := r.pageConfigService.AuthenticateAPIClient(
		c.Request.Context(),
		northboundAPIClientCredential(c),
		c.ClientIP(),
		apiKey,
	)
	if err != nil {
		switch {
		case errors.Is(err, commonerrors.ErrUnauthorized):
			response.Fail(c, http.StatusUnauthorized, err.Error())
		case errors.Is(err, commonerrors.ErrForbidden):
			response.Fail(c, http.StatusForbidden, err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, "northbound API user check failed")
		}
		return false
	}
	if client != nil {
		c.Set("northbound_api_user", client.ClientKey)
		c.Set("northbound_api_client", client.ClientKey)
	}
	return true
}

func northboundAPIClientCredential(c *gin.Context) string {
	if token := strings.TrimSpace(c.GetHeader("X-Northbound-Token")); token != "" {
		return token
	}
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func northboundResponseContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(response.ContextKeyNorthbound, true)
		c.Next()
	}
}

func (r *Router) pageConfiguredOnly(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.requirePageConfiguredAPI(c, apiKey) {
			return
		}
		c.Next()
	}
}

func (r *Router) apiUserLogin(c *gin.Context) {
	if r.pageConfigHandler == nil {
		response.Fail(c, http.StatusInternalServerError, "northbound API user service is not configured")
		return
	}
	r.pageConfigHandler.LoginAPIUser(c)
}

// RegisterPublicRoutes registers northbound API routes that are called by
// external OSS systems with northbound-only users and tokens.
func (r *Router) RegisterPublicRoutes(rg *gin.RouterGroup) {
	nb := rg.Group("/northbound")
	nb.Use(northboundResponseContext())
	legacy := nb.Group("/v1")
	legacy.POST("/access/token", r.rateLimit("nb:legacy:auth:token"), r.pageConfiguredOnly("auth-login"), r.apiUserLogin)
	r.registerLegacyV1Routes(legacy)
}

// RegisterRoutes registers northbound API routes on the given router group.
func (r *Router) RegisterRoutes(rg *gin.RouterGroup) {
	nb := rg.Group("/northbound")
	// 北向响应对上游 OSS 必须保持 UTC 标准格式（issue #457）：打上下文标记，
	// 统一响应出口（response.renderJSON）识别后整体跳过系统时区转换。
	nb.Use(northboundResponseContext())
	{
		// Push target management
		nb.GET("/push/targets", r.listTargets)
		nb.POST("/push/targets", r.addTarget)
		nb.DELETE("/push/targets/:id", r.removeTarget)

		// Circuit breaker status
		nb.GET("/push/targets/:id/circuit", r.getCircuitBreaker)
		nb.POST("/push/targets/:id/circuit/reset", r.resetCircuitBreaker)

		// Dead letter queue
		nb.GET("/push/deadletter", r.listDeadLetters)
		nb.POST("/push/deadletter/:id/replay", r.replayDeadLetter)

		// Sync endpoints（per-endpoint 限流：同步是重查询，限速防滥用）
		nb.GET("/sync/full", r.rateLimit("nb:sync:full"), r.fullSync)
		nb.GET("/sync/incremental", r.rateLimit("nb:sync:incremental"), r.incrementalSync)

		// Export endpoints（per-endpoint 限流）
		nb.POST("/export/pm", r.rateLimit("nb:export:pm"), r.pmHandler.ExportPM)
		nb.POST("/export/alarms", r.rateLimit("nb:export:alarms"), r.alarmHandler.ExportAlarms)
		nb.GET("/export/config/:deviceId", r.rateLimit("nb:export:config"), r.pageConfiguredAPI("nb-export-config"), r.configHandler.ExportConfig)

		// Page-config control plane. P0 exposes default catalogs and validation
		// without changing existing northbound push/sync/export behavior.
		if r.pageConfigHandler != nil {
			r.pageConfigHandler.RegisterRoutes(nb)
		}

		// 主备服务器配置 + 切换（system/config 北向设置）。
		// 仅在 SetServerService 注入后挂载，未注入时这三条端点 404，避免 nil deref。
		// PUT /servers/:role 是"独立编辑权限"端点：默认仅 admin / operator 角色拥有
		// （seed/000070 显式 grant），viewer 不可改配置。
		if r.serverHandler != nil {
			nb.GET("/servers", r.serverHandler.ListServers)
			nb.PUT("/servers/active", r.serverHandler.SwitchActive)
			nb.PUT("/servers/:role", r.serverHandler.UpdateServer)
		}
	}
}

func (r *Router) registerLegacyV1Routes(legacy *gin.RouterGroup) {
	// Legacy-shaped northboundApi routes. They live under the northbound
	// namespace so existing xomc business endpoints are not shadowed.
	legacy.GET("/sync/full", r.rateLimit("nb:legacy:sync:full"), r.pageConfiguredAPI("nb-sync-full-device"), r.fullSync)
	legacy.GET("/export/config/:deviceId", r.rateLimit("nb:legacy:export:config"), r.pageConfiguredAPI("nb-export-config"), r.configHandler.ExportConfig)
	legacy.POST("/device/query", r.rateLimit("nb:legacy:device:query"), r.pageConfiguredAPI("device-list"), r.legacyDeviceQuery)
	legacy.GET("/device/status/:sn", r.rateLimit("nb:legacy:device:status"), r.pageConfiguredAPI("device-status"), r.legacyDeviceStatus)
	legacy.GET("/device/infos/:sn", r.rateLimit("nb:legacy:device:infos"), r.pageConfiguredAPI("device-detail"), r.legacyDeviceInfo)
	legacy.GET("/enodeb/infos/status/:sn", r.rateLimit("nb:legacy:enodeb:status"), r.pageConfiguredAPI("device-status"), r.legacyDeviceStatus)
	legacy.POST("/device/register", r.rateLimit("nb:legacy:device:register:create"), r.pageConfiguredAPI("device-register-create"), r.legacyCreateRegistration)
	legacy.GET("/device/register/page", r.rateLimit("nb:legacy:device:register:list"), r.pageConfiguredAPI("device-register-list"), r.legacyListRegistrations)
	legacy.POST("/device/register/page", r.rateLimit("nb:legacy:device:register:list"), r.pageConfiguredAPI("device-register-list"), r.legacyListRegistrations)
	legacy.DELETE("/device/register/:id", r.rateLimit("nb:legacy:device:register:delete"), r.pageConfiguredAPI("device-register-delete"), r.legacyDeleteRegistration)
	legacy.GET("/device/group", r.rateLimit("nb:legacy:device:group:tree"), r.pageConfiguredAPI("device-group-tree"), r.legacyDeviceGroupTree)
	legacy.POST("/device/group", r.rateLimit("nb:legacy:device:group:create"), r.pageConfiguredAPI("device-group-create"), r.legacyCreateDeviceGroup)
	legacy.PUT("/device/group", r.rateLimit("nb:legacy:device:group:update"), r.pageConfiguredAPI("device-group-update"), r.legacyUpdateDeviceGroup)
	legacy.PUT("/device/group/:id", r.rateLimit("nb:legacy:device:group:update"), r.pageConfiguredAPI("device-group-update"), r.legacyUpdateDeviceGroup)
	legacy.DELETE("/device/group", r.rateLimit("nb:legacy:device:group:delete"), r.pageConfiguredAPI("device-group-delete"), r.legacyDeleteDeviceGroup)
	legacy.DELETE("/device/group/:id", r.rateLimit("nb:legacy:device:group:delete"), r.pageConfiguredAPI("device-group-delete"), r.legacyDeleteDeviceGroup)
	legacy.POST("/device/group/:id/devices", r.rateLimit("nb:legacy:device:group:add-devices"), r.pageConfiguredAPI("device-group-add-devices"), r.legacyAddDevicesToGroup)
	legacy.GET("/device/parameters/:sn", r.rateLimit("nb:legacy:parameters:get"), r.pageConfiguredAPI("parameter-tree"), r.legacyGetDeviceParameters)
	legacy.POST("/device/parameters/query/:sn", r.rateLimit("nb:legacy:parameters:query"), r.pageConfiguredAPI("config-pull"), r.legacyQueryDeviceParameters)
	legacy.PUT("/device/parameters/cellname/:sn", r.rateLimit("nb:legacy:cellname:set"), r.pageConfiguredAPI("parameter-cellname"), r.legacySetDeviceName)
	legacy.PUT("/device/parameters/:sn", r.rateLimit("nb:legacy:parameters:set"), r.pageConfiguredAPI("parameter-set"), r.legacySetDeviceParameters)
	legacy.POST("/device/task/:sn", r.rateLimit("nb:legacy:device:task:create"), r.pageConfiguredAPI("device-task-create"), r.legacyCreateDeviceTask)
	legacy.POST("/device/reboot/:sn", r.rateLimit("nb:legacy:device:reboot"), r.pageConfiguredAPI("device-reboot"), r.legacyRebootDevice)
	legacy.POST("/device/reset/:sn", r.rateLimit("nb:legacy:device:reset"), r.pageConfiguredAPI("device-reset"), r.legacyResetDevice)
	legacy.POST("/device/log/collect/:sn", r.rateLimit("nb:legacy:device:log-collect"), r.pageConfiguredAPI("device-log-collect"), r.legacyCollectDeviceLog)
	legacy.GET("/job/result/:taskId", r.rateLimit("nb:legacy:job:result"), r.pageConfiguredAPI("task-detail"), r.legacyGetTask)
	legacy.POST("/job/result/page", r.rateLimit("nb:legacy:job:page"), r.pageConfiguredAPI("task-page"), r.legacyListTasks)
}

func (r *Router) listTargets(c *gin.Context) {
	targets := r.svc.PushEngine().ListTargets()
	response.OK(c, gin.H{"items": targets, "total": len(targets)})
}

func (r *Router) addTarget(c *gin.Context) {
	var req AddPushTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	target := &push.Target{
		ID:         req.ID,
		URL:        req.URL,
		AuthType:   req.AuthType,
		AuthToken:  req.AuthToken,
		DataTypes:  req.DataTypes,
		Format:     req.Format,
		BatchSize:  req.BatchSize,
		RetryCount: req.RetryCount,
		Enabled:    req.Enabled,
	}

	r.svc.PushEngine().AddTarget(target)
	response.OKWithStatus(c, http.StatusCreated, gin.H{"message": "push target added", "id": req.ID})
}

func (r *Router) removeTarget(c *gin.Context) {
	id := c.Param("id")
	if !r.svc.PushEngine().RemoveTarget(id) {
		response.Fail(c, http.StatusNotFound, "push target not found")
		return
	}
	response.OKWithMsg(c, gin.H{"id": id}, "push target removed")
}

func (r *Router) getCircuitBreaker(c *gin.Context) {
	id := c.Param("id")
	cb := r.svc.PushEngine().GetCircuitBreaker(id)
	if cb == nil {
		response.Fail(c, http.StatusNotFound, "push target not found")
		return
	}

	failureCount, threshold := cb.Counters()
	state := cb.State()
	response.OK(c, gin.H{
		"target_id":     id,
		"state":         state.String(),
		"failure_count": failureCount,
		"threshold":     threshold,
	})
}

func (r *Router) resetCircuitBreaker(c *gin.Context) {
	id := c.Param("id")
	cb := r.svc.PushEngine().GetCircuitBreaker(id)
	if cb == nil {
		response.Fail(c, http.StatusNotFound, "push target not found")
		return
	}

	cb.Reset()
	response.OKWithMsg(c, gin.H{"target_id": id, "state": reliability.StateClosed.String()}, "circuit breaker reset")
}

func (r *Router) listDeadLetters(c *gin.Context) {
	if r.outboxRepo == nil {
		response.Fail(c, http.StatusServiceUnavailable, "outbox not configured")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	entries, total, err := r.outboxRepo.ListDead(c.Request.Context(), limit, offset)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"items": entries, "total": total})
}

func (r *Router) replayDeadLetter(c *gin.Context) {
	if r.outboxRepo == nil {
		response.Fail(c, http.StatusServiceUnavailable, "outbox not configured")
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid uuid")
		return
	}

	if err := r.outboxRepo.Replay(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	response.OKWithMsg(c, gin.H{"id": idStr}, "dead letter replayed")
}

func (r *Router) fullSync(c *gin.Context) {
	// 过滤参数白名单：拒绝未知 query 参数。
	if bad := unknownFilterParams(c.Request.URL.Query(), allowedSyncParams); len(bad) > 0 {
		response.Fail(c, http.StatusBadRequest, "unknown filter params: "+joinFields(bad))
		return
	}
	// 全量同步无设备维度：非超管不得跨租户拉取全量 → 403。
	if r.scoper != nil && !r.scoper.RequireDeviceScope(c, false) {
		return
	}

	dataType := c.DefaultQuery("data_type", "device")
	if dataType == "device" {
		if !r.requirePageConfiguredAPI(c, "nb-sync-full-device") {
			return
		}
		if !r.requirePageConfiguredAPIClient(c, "nb-sync-full-device") {
			return
		}
	}
	result, err := r.svc.SyncService().FullSync(c.Request.Context(), dataType)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, result)
}

func (r *Router) incrementalSync(c *gin.Context) {
	// 过滤参数白名单：拒绝未知 query 参数。
	if bad := unknownFilterParams(c.Request.URL.Query(), allowedSyncParams); len(bad) > 0 {
		response.Fail(c, http.StatusBadRequest, "unknown filter params: "+joinFields(bad))
		return
	}
	// 增量同步无设备维度：非超管不得跨租户拉取全量 → 403。
	if r.scoper != nil && !r.scoper.RequireDeviceScope(c, false) {
		return
	}

	dataType := c.DefaultQuery("data_type", "alarm")
	sinceStr := c.Query("since")
	if sinceStr == "" {
		response.Fail(c, http.StatusBadRequest, "since parameter is required (RFC3339 format)")
		return
	}

	since, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid since format, use RFC3339")
		return
	}

	result, err := r.svc.SyncService().IncrementalSync(c.Request.Context(), dataType, since)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, result)
}

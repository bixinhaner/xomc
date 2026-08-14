package pageconfig

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	admin "github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/admin/audit"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type Handler struct {
	svc    *Service
	logger *zap.Logger
}

func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	if svc == nil {
		svc = NewService(nil)
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) RegisterRoutes(nb *gin.RouterGroup) {
	pc := nb.Group("/page-config")
	{
		pc.GET("/overview", h.Overview)
		pc.GET("/file/profiles", h.ListFileProfiles)
		pc.POST("/file/profiles", h.CreateFileProfile)
		pc.GET("/file/profiles/:id/preview", h.PreviewFileProfile)
		pc.POST("/file/profiles/:id/run", h.RunFileProfile)
		pc.PUT("/file/profiles/:id", h.UpdateFileProfile)
		pc.GET("/inventory/profiles", h.ListInventoryProfiles)
		pc.POST("/inventory/profiles/:id/run", h.RunInventoryProfile)
		pc.PUT("/inventory/profiles/:id", h.UpdateInventoryProfile)
		pc.GET("/runs", h.ListRuns)
		pc.GET("/runs/:id", h.GetRun)
		pc.GET("/runs/:id/download", h.DownloadRun)
		pc.GET("/delivery/targets", h.ListDeliveryTargets)
		pc.PUT("/delivery/targets", h.ReplaceDeliveryTargets)
		pc.POST("/delivery/targets/test", h.TestDeliveryTarget)
		pc.GET("/alarm/snmp/targets", h.ListSNMPAlarmTargets)
		pc.POST("/alarm/snmp/targets/:key/test", h.TestSNMPAlarmTarget)
		pc.PUT("/alarm/snmp/targets/:key", h.UpdateSNMPAlarmTarget)
		pc.GET("/alarm/socket/configs", h.ListSocketAlarmConfigs)
		pc.POST("/alarm/socket/configs/:key/test", h.TestSocketAlarmConfig)
		pc.PUT("/alarm/socket/configs/:key", h.UpdateSocketAlarmConfig)
		pc.GET("/api/configs", h.ListAPIConfigs)
		pc.PUT("/api/configs", h.UpdateAllAPIConfigs)
		pc.POST("/api/configs/:key/test", h.TestAPIConfig)
		pc.PUT("/api/configs/:key", h.UpdateAPIConfig)
		pc.GET("/api/invocation-logs", h.ListAPIInvocationLogs)
		pc.GET("/api/users", h.ListAPIUsers)
		pc.PUT("/api/users", h.ReplaceAPIUsers)
		pc.GET("/api/clients", h.ListAPIClients)
		pc.PUT("/api/clients", h.ReplaceAPIClients)
		pc.GET("/events", h.ListEvents)
		pc.GET("/events/:id", h.GetEvent)
		pc.GET("/fields", h.ListFields)
		pc.POST("/validate", h.Validate)
	}
}

func (h *Handler) Overview(c *gin.Context) {
	overview, err := h.svc.Overview(c.Request.Context())
	if err != nil {
		h.logger.Error("load northbound page-config overview failed", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, "load northbound page-config overview failed")
		return
	}
	response.OK(c, overview)
}

func (h *Handler) ListFileProfiles(c *gin.Context) {
	items, err := h.svc.ListFileProfiles(c.Request.Context())
	if err != nil {
		h.logger.Error("list northbound file profiles failed", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, "list northbound file profiles failed")
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) CreateFileProfile(c *gin.Context) {
	var req FileProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	profile, err := h.svc.CreateFileProfile(c.Request.Context(), req)
	h.auditPageConfig(c, "create_file_profile", req.Code, err == nil, err, map[string]interface{}{
		"capability":  "file",
		"profile":     req.Code,
		"enabled":     req.Enabled,
		"status":      req.Status,
		"group_count": len(req.Groups),
	})
	if err != nil {
		h.handleUpdateError(c, "create northbound file profile failed", err)
		return
	}
	response.OK(c, profile)
}

func (h *Handler) ListInventoryProfiles(c *gin.Context) {
	items, err := h.svc.ListInventoryProfiles(c.Request.Context())
	if err != nil {
		h.logger.Error("list northbound inventory profiles failed", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, "list northbound inventory profiles failed")
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ListFields(c *gin.Context) {
	filter := FieldFilter{
		Domain:     Domain(c.Query("domain")),
		ObjectCode: c.Query("object"),
		Tech:       c.Query("tech"),
		Profile:    c.Query("profile"),
	}
	items := h.svc.ListFields(c.Request.Context(), filter)
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) Validate(c *gin.Context) {
	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	result := h.svc.Validate(c.Request.Context(), req)
	if !result.Valid {
		response.FailWithData(c, http.StatusBadRequest, "northbound page config validation failed", result)
		return
	}
	response.OK(c, result)
}

func (h *Handler) UpdateFileProfile(c *gin.Context) {
	var req UpdateFileProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	profile, err := h.svc.UpdateFileProfile(c.Request.Context(), c.Param("id"), req)
	h.auditPageConfig(c, "update_file_profile", c.Param("id"), err == nil, err, map[string]interface{}{
		"capability":  "file",
		"profile":     c.Param("id"),
		"status":      req.Status,
		"group_count": len(req.Groups),
	})
	if err != nil {
		h.handleUpdateError(c, "update northbound file profile failed", err)
		return
	}
	response.OK(c, profile)
}

func (h *Handler) PreviewFileProfile(c *gin.Context) {
	preview, err := h.svc.PreviewFileProfile(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleUpdateError(c, "preview northbound file profile failed", err)
		return
	}
	response.OK(c, preview)
}

func (h *Handler) RunFileProfile(c *gin.Context) {
	req, ok := h.bindRunProfileRequest(c)
	if !ok {
		return
	}
	req.TriggerReason = runTriggerManual
	result, err := h.svc.RunFileProfile(c.Request.Context(), c.Param("id"), req)
	h.auditPageConfig(c, "run_file_profile", c.Param("id"), err == nil, err, map[string]interface{}{
		"capability": "file",
		"profile":    c.Param("id"),
		"group_id":   req.GroupID,
	})
	if err != nil {
		h.handleUpdateError(c, "run northbound file profile failed", err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) UpdateInventoryProfile(c *gin.Context) {
	var req UpdateInventoryProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	profile, err := h.svc.UpdateInventoryProfile(c.Request.Context(), c.Param("id"), req)
	h.auditPageConfig(c, "update_inventory_profile", c.Param("id"), err == nil, err, map[string]interface{}{
		"capability":  "inventory",
		"profile":     c.Param("id"),
		"object_code": req.ObjectCode,
		"period":      req.Period,
		"status":      req.Status,
	})
	if err != nil {
		h.handleUpdateError(c, "update northbound inventory profile failed", err)
		return
	}
	response.OK(c, profile)
}

func (h *Handler) RunInventoryProfile(c *gin.Context) {
	req, ok := h.bindRunProfileRequest(c)
	if !ok {
		return
	}
	req.TriggerReason = runTriggerManual
	result, err := h.svc.RunInventoryProfile(c.Request.Context(), c.Param("id"), req)
	h.auditPageConfig(c, "run_inventory_profile", c.Param("id"), err == nil, err, map[string]interface{}{
		"capability": "inventory",
		"profile":    c.Param("id"),
	})
	if err != nil {
		h.handleUpdateError(c, "run northbound inventory profile failed", err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListRuns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	latestPerProfile, _ := strconv.ParseBool(c.DefaultQuery("latest_per_profile", "false"))
	result, err := h.svc.ListRuns(c.Request.Context(), RunFilter{
		ProfileKind:      ProfileKind(c.Query("profile_kind")),
		ProfileCode:      c.Query("profile_code"),
		Status:           RunStatus(c.Query("status")),
		LatestPerProfile: latestPerProfile,
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		h.handleUpdateError(c, "list northbound file runs failed", err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) GetRun(c *gin.Context) {
	run, err := h.svc.GetRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleUpdateError(c, "get northbound file run failed", err)
		return
	}
	response.OK(c, run)
}

func (h *Handler) DownloadRun(c *gin.Context) {
	run, err := h.svc.GetRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleUpdateError(c, "download northbound file run failed", err)
		return
	}
	download, err := h.svc.DownloadRunArtifact(c.Request.Context(), run.ID)
	h.auditPageConfig(c, "download_run_artifact", run.ID, err == nil, err, map[string]interface{}{
		"capability":    string(run.ProfileKind),
		"profile_code":  run.ProfileCode,
		"group_id":      run.GroupID,
		"artifact":      run.ArtifactName,
		"artifact_size": run.ArtifactSize,
	})
	if err != nil {
		h.handleUpdateError(c, "download northbound file run failed", err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(download.FileName, `"`, "")+`"`)
	c.Data(http.StatusOK, download.ContentType, download.Content)
}

func (h *Handler) ListDeliveryTargets(c *gin.Context) {
	items, err := h.svc.ListDeliveryTargets(c.Request.Context(), DeliveryTargetFilter{
		Scope:     DeliveryScope(c.Query("scope")),
		OwnerCode: c.Query("owner_code"),
	})
	if err != nil {
		h.handleUpdateError(c, "list northbound delivery targets failed", err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ReplaceDeliveryTargets(c *gin.Context) {
	var req ReplaceDeliveryTargetsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	items, err := h.svc.ReplaceDeliveryTargets(c.Request.Context(), req)
	h.auditPageConfig(c, "replace_delivery_targets", string(req.Scope)+":"+req.OwnerCode, err == nil, err, map[string]interface{}{
		"capability":    "delivery",
		"scope":         req.Scope,
		"owner_code":    req.OwnerCode,
		"target_count":  len(req.Items),
		"enabled_count": enabledDeliveryTargetCount(req.Items),
	})
	if err != nil {
		h.handleUpdateError(c, "replace northbound delivery targets failed", err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) TestDeliveryTarget(c *gin.Context) {
	var req DeliveryTarget
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	event, err := h.svc.TestDeliveryTarget(c.Request.Context(), req)
	h.auditPageConfig(c, "test_delivery_target", string(req.Scope)+":"+req.OwnerCode+":"+req.Key, err == nil, err, map[string]interface{}{
		"capability": "delivery",
		"scope":      req.Scope,
		"owner_code": req.OwnerCode,
		"target_key": req.Key,
		"protocol":   req.Protocol,
		"host":       req.Host,
		"port":       req.Port,
	})
	if err != nil {
		h.handleUpdateError(c, "test northbound delivery target failed", err)
		return
	}
	response.OK(c, event)
}

func (h *Handler) ListSNMPAlarmTargets(c *gin.Context) {
	items, err := h.svc.ListSNMPAlarmTargets(c.Request.Context())
	if err != nil {
		h.handleUpdateError(c, "list northbound SNMP targets failed", err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items), "mib_fields": defaultSNMPAlarmFields()})
}

func (h *Handler) TestSNMPAlarmTarget(c *gin.Context) {
	event, err := h.svc.TestSNMPAlarmTarget(c.Request.Context(), c.Param("key"))
	h.auditPageConfig(c, "test_snmp_alarm_target", c.Param("key"), err == nil, err, map[string]interface{}{
		"capability": "snmp",
		"target_key": c.Param("key"),
	})
	if err != nil {
		h.handleUpdateError(c, "test northbound SNMP target failed", err)
		return
	}
	response.OK(c, event)
}

func (h *Handler) UpdateSNMPAlarmTarget(c *gin.Context) {
	var req SNMPAlarmTarget
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	item, err := h.svc.UpdateSNMPAlarmTarget(c.Request.Context(), c.Param("key"), req)
	h.auditPageConfig(c, "update_snmp_alarm_target", c.Param("key"), err == nil, err, map[string]interface{}{
		"capability":        "snmp",
		"target_key":        c.Param("key"),
		"enabled":           req.Enabled,
		"version":           req.Version,
		"notification_type": req.NotificationType,
		"target_host":       req.TargetHost,
		"target_port":       req.TargetPort,
	})
	if err != nil {
		h.handleUpdateError(c, "update northbound SNMP target failed", err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) ListSocketAlarmConfigs(c *gin.Context) {
	items, err := h.svc.ListSocketAlarmConfigs(c.Request.Context())
	if err != nil {
		h.handleUpdateError(c, "list northbound socket configs failed", err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) TestSocketAlarmConfig(c *gin.Context) {
	event, err := h.svc.TestSocketAlarmConfig(c.Request.Context(), c.Param("key"))
	h.auditPageConfig(c, "test_socket_alarm_config", c.Param("key"), err == nil, err, map[string]interface{}{
		"capability": "socket",
		"config_key": c.Param("key"),
	})
	if err != nil {
		h.handleUpdateError(c, "test northbound socket config failed", err)
		return
	}
	response.OK(c, event)
}

func (h *Handler) UpdateSocketAlarmConfig(c *gin.Context) {
	var req SocketAlarmConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	item, err := h.svc.UpdateSocketAlarmConfig(c.Request.Context(), c.Param("key"), req)
	h.auditPageConfig(c, "update_socket_alarm_config", c.Param("key"), err == nil, err, map[string]interface{}{
		"capability":            "socket",
		"config_key":            c.Param("key"),
		"enabled":               req.Enabled,
		"profile":               req.Profile,
		"listen_ip":             req.ListenIP,
		"listen_port":           req.ListenPort,
		"max_clients":           req.MaxClients,
		"realtime_push_enabled": req.RealtimePushEnabled,
		"client_sync_enabled":   req.ClientSyncEnabled,
		"heartbeat_seconds":     req.HeartbeatSeconds,
		"heartbeat_times":       req.HeartbeatTimes,
		"account_count":         len(req.Accounts),
		"enabled_account_count": enabledSocketAccountCount(req.Accounts),
	})
	if err != nil {
		h.handleUpdateError(c, "update northbound socket config failed", err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) ListAPIConfigs(c *gin.Context) {
	items, err := h.svc.ListAPIConfigs(c.Request.Context())
	if err != nil {
		h.handleUpdateError(c, "list northbound API configs failed", err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) TestAPIConfig(c *gin.Context) {
	event, err := h.svc.TestAPIConfig(c.Request.Context(), c.Param("key"))
	h.auditPageConfig(c, "test_api_config", c.Param("key"), err == nil, err, map[string]interface{}{
		"capability": "api",
		"api_key":    c.Param("key"),
	})
	if err != nil {
		h.handleUpdateError(c, "test northbound API config failed", err)
		return
	}
	response.OK(c, event)
}

func (h *Handler) UpdateAPIConfig(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	item, err := h.svc.UpdateAPIConfig(c.Request.Context(), c.Param("key"), req.Enabled)
	h.auditPageConfig(c, "update_api_config", c.Param("key"), err == nil, err, map[string]interface{}{
		"capability": "api",
		"api_key":    c.Param("key"),
		"enabled":    req.Enabled,
	})
	if err != nil {
		h.handleUpdateError(c, "update northbound API config failed", err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) UpdateAllAPIConfigs(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	items, err := h.svc.UpdateAllAPIConfigs(c.Request.Context(), req.Enabled)
	h.auditPageConfig(c, "update_all_api_configs", "api-configs", err == nil, err, map[string]interface{}{
		"capability": "api",
		"enabled":    req.Enabled,
		"total":      len(items),
	})
	if err != nil {
		h.handleUpdateError(c, "update northbound API configs failed", err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ListAPIClients(c *gin.Context) {
	items, err := h.svc.ListAPIClients(c.Request.Context())
	if err != nil {
		h.handleUpdateError(c, "list northbound API clients failed", err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ListAPIUsers(c *gin.Context) {
	items, err := h.svc.ListAPIUsers(c.Request.Context())
	if err != nil {
		h.handleUpdateError(c, "list northbound API users failed", err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ListAPIInvocationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(firstNonEmpty(c.Query("page_size"), c.Query("pageSize"), c.Query("limit"), "20"))
	pageSize = normalizeLimit(pageSize)

	offset, err := strconv.Atoi(c.Query("offset"))
	if err != nil || offset < 0 {
		offset = (page - 1) * pageSize
	}

	result, err := h.svc.ListAPIInvocationLogs(c.Request.Context(), APIInvocationLogFilter{
		APIKey:     c.Query("api_key"),
		Name:       c.Query("name"),
		Method:     c.Query("method"),
		Path:       c.Query("path"),
		Status:     c.Query("status"),
		CreateUser: c.Query("create_user"),
		IPAddress:  c.Query("ip_address"),
		Keyword:    c.Query("keyword"),
		StartTime:  c.Query("start_time"),
		EndTime:    c.Query("end_time"),
		Limit:      pageSize,
		Offset:     offset,
	})
	if err != nil {
		h.handleUpdateError(c, "list northbound API invocation logs failed", err)
		return
	}
	response.OK(c, gin.H{
		"items":       result.Items,
		"total":       result.Total,
		"limit":       result.Limit,
		"offset":      result.Offset,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages(result.Total, pageSize),
	})
}

func totalPages(total, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	pages := total / pageSize
	if total%pageSize != 0 {
		pages++
	}
	return pages
}

func (h *Handler) ReplaceAPIUsers(c *gin.Context) {
	var req ReplaceAPIUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	items, err := h.svc.ReplaceAPIUsers(c.Request.Context(), req)
	h.auditPageConfig(c, "replace_api_users", "api-users", err == nil, err, map[string]interface{}{
		"capability":    "api",
		"user_count":    len(req.Items),
		"enabled_count": enabledAPIUserCount(req.Items),
	})
	if err != nil {
		h.handleUpdateError(c, "replace northbound API users failed", err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) LoginAPIUser(c *gin.Context) {
	var req APIUserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	token, err := h.svc.LoginAPIUser(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, commonerrors.ErrUnauthorized) {
			response.Fail(c, http.StatusUnauthorized, err.Error())
			return
		}
		h.handleUpdateError(c, "northbound API login failed", err)
		return
	}
	response.OK(c, token)
}

func (h *Handler) ReplaceAPIClients(c *gin.Context) {
	var req ReplaceAPIClientsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	items, err := h.svc.ReplaceAPIClients(c.Request.Context(), req)
	h.auditPageConfig(c, "replace_api_clients", "api-clients", err == nil, err, map[string]interface{}{
		"capability":     "api",
		"client_count":   len(req.Items),
		"enabled_count":  enabledAPIClientCount(req.Items),
		"allowed_scopes": summarizeAPIClientScopes(req.Items),
	})
	if err != nil {
		h.handleUpdateError(c, "replace northbound API clients failed", err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ListEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	includePayload := !isFalseQueryValue(c.DefaultQuery("include_payload", "true"))
	result, err := h.svc.ListEvents(c.Request.Context(), EventFilter{
		Capability: c.Query("capability"),
		OwnerCode:  c.Query("owner_code"),
		TargetKey:  c.Query("target_key"),
		EventType:  c.Query("event_type"),
		Status:     RunStatus(c.Query("status")),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		h.handleUpdateError(c, "list northbound page-config events failed", err)
		return
	}
	if !includePayload {
		for i := range result.Items {
			result.Items[i].Payload = ""
		}
	}
	response.OK(c, result)
}

func isFalseQueryValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}

func (h *Handler) GetEvent(c *gin.Context) {
	event, err := h.svc.GetEvent(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleUpdateError(c, "get northbound page-config event failed", err)
		return
	}
	response.OK(c, event)
}

func (h *Handler) bindRunProfileRequest(c *gin.Context) (RunProfileRequest, bool) {
	var req RunProfileRequest
	if c.Request.Body == nil {
		return req, true
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return RunProfileRequest{}, false
	}
	return req, true
}

func contentTypeForArtifact(fileName string) string {
	name := strings.ToLower(fileName)
	switch {
	case strings.Contains(name, ".zip"):
		return "application/zip"
	case strings.Contains(name, ".gz"):
		return "application/gzip"
	case strings.Contains(name, ".csv"):
		return "text/csv; charset=utf-8"
	case strings.Contains(name, ".xml"):
		return "application/xml; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}

func (h *Handler) handleUpdateError(c *gin.Context, msg string, err error) {
	switch {
	case errors.Is(err, commonerrors.ErrNotFound):
		response.Fail(c, http.StatusNotFound, "northbound page-config profile not found")
	case errors.Is(err, commonerrors.ErrInvalidInput):
		response.Fail(c, http.StatusBadRequest, err.Error())
	default:
		h.logger.Error(msg, zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, msg)
	}
}

func (h *Handler) auditPageConfig(c *gin.Context, action string, resourceID string, success bool, err error, details map[string]interface{}) {
	entry := admin.AuditContextFromGin(c)
	entry.Action = audit.ActionConfig + ".northbound_page_config." + action
	entry.ResourceType = audit.ResourceConfig
	entry.ResourceID = "northbound-page-config"
	if strings.TrimSpace(resourceID) != "" {
		entry.ResourceID += ":" + resourceID
	}
	entry.Success = success
	entry.ErrorMessage = errorString(err)
	entry.Details = sanitizePageConfigAuditDetails(details)
	audit.LogAsync(entry)
}

func sanitizePageConfigAuditDetails(details map[string]interface{}) map[string]interface{} {
	if len(details) == 0 {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(details))
	for key, value := range details {
		if isSensitiveAuditKey(key) {
			out[key] = "<redacted>"
			continue
		}
		out[key] = value
	}
	return out
}

func isSensitiveAuditKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(key, "password") ||
		strings.Contains(key, "credential") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "community") ||
		strings.Contains(key, "auth_credential") ||
		strings.Contains(key, "priv_credential")
}

func enabledDeliveryTargetCount(targets []DeliveryTarget) int {
	total := 0
	for _, target := range targets {
		if target.Enabled {
			total++
		}
	}
	return total
}

func enabledAPIClientCount(clients []APIClient) int {
	total := 0
	for _, client := range clients {
		if client.Enabled {
			total++
		}
	}
	return total
}

func enabledAPIUserCount(users []APIUser) int {
	total := 0
	for _, user := range users {
		if user.Enabled {
			total++
		}
	}
	return total
}

func summarizeAPIClientScopes(clients []APIClient) []string {
	out := make([]string, 0, len(clients))
	for _, client := range clients {
		out = append(out, client.ClientKey+":"+strings.Join(client.AllowedAPIKeys, ","))
	}
	return out
}

package geofence

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/admin/audit"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
	"go.uber.org/zap"
)

const (
	auditActionCreate         = audit.ActionConfig + "_geofence_create"
	auditActionModify         = audit.ActionConfig + "_geofence_modify"
	auditActionPublish        = audit.ActionConfig + "_geofence_publish"
	auditActionEnable         = audit.ActionConfig + "_geofence_enable"
	auditActionDisable        = audit.ActionConfig + "_geofence_disable"
	auditActionArchive        = audit.ActionConfig + "_geofence_archive"
	auditActionSettingsUpdate = audit.ActionConfig + "_geofence_settings_update"
	auditActionBatchBind      = audit.ActionConfig + "_geofence_bind"
	auditActionBindingSuspend = audit.ActionConfig + "_geofence_binding_suspend"
	auditActionBindingResume  = audit.ActionConfig + "_geofence_binding_resume"
	auditActionBindingRemove  = audit.ActionConfig + "_geofence_binding_remove"
)

type Handler struct {
	service  *Service
	resolver *authz.Resolver
	logger   *zap.Logger
}

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{service: service, logger: logger}
}

func (h *Handler) SetPermissionService(perm authz.VisibleGroupsResolver) {
	h.resolver = authz.NewResolver(perm)
}

func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	geofences := group.Group("/geofences")
	geofences.Use(h.resolveVisibleGroups)
	geofences.GET("", h.ListDefinitions)
	geofences.POST("", h.CreateDefinition)
	geofences.GET("/settings", h.GetSettings)
	geofences.POST("/settings/preview", h.PreviewSettings)
	geofences.PUT("/settings", h.UpdateSettings)
	geofences.GET("/map", h.ListMapDefinitions)
	definition := geofences.Group("/:id")
	definition.Use(h.authorizeDefinitionRoute)
	definition.GET("", h.GetDefinition)
	definition.PATCH("", h.RenameDefinition)
	definition.GET("/versions", h.ListVersions)
	definition.POST("/versions", h.CreateDraftVersion)
	definition.POST("/publish", h.PublishDraft)
	definition.POST("/enable-preview", h.PreviewEnable)
	definition.POST("/enable", h.EnableDefinition)
	definition.POST("/disable-preview", h.PreviewDisable)
	definition.POST("/disable", h.DisableDefinition)
	definition.POST("/archive-preview", h.PreviewArchive)
	definition.POST("/archive", h.ArchiveDefinition)
	definition.POST("/binding-preview", h.PreviewManualBindings)
	definition.POST("/candidate-preview", h.PreviewGeofenceCandidates)
	definition.POST("/bindings", h.CreateManualBindJob)
	definition.GET("/bindings/export", h.ExportBindings)
	definition.GET("/bindings", h.ListBindings)
	definition.GET("/control-actions", h.ListControlActions)

	bindings := group.Group("/geofence-bindings")
	bindings.Use(h.resolveVisibleGroups, h.authorizeBindingRoute)
	bindings.POST("/:id/suspend", h.SuspendBinding)
	bindings.POST("/:id/resume", h.ResumeBinding)
	bindings.DELETE("/:id", h.RemoveBinding)

	jobs := group.Group("/geofence-jobs")
	jobs.GET("/:id", h.GetManualBindJob)
	jobs.GET("/:id/items", h.ListManualBindItems)
}

// RegisterAvailabilityRoute is intentionally registered on the authenticated
// group before endpoint-level authorization. Every signed-in GIS user may
// discover whether the system feature is enabled; actual geofence data and
// mutations remain protected by API permissions in RegisterRoutes.
func (h *Handler) RegisterAvailabilityRoute(group *gin.RouterGroup) {
	group.GET("/geofences/availability", h.GetAvailability)
}

func (h *Handler) GetAvailability(c *gin.Context) {
	availability, err := h.service.GetAvailability(c.Request.Context())
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, availability)
}

func (h *Handler) ListControlActions(c *gin.Context) {
	geofenceID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	actions, err := h.service.ListControlActions(
		c.Request.Context(),
		geofenceID,
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, actions)
}

func (h *Handler) PreviewGeofenceCandidates(c *gin.Context) {
	admin.SkipAudit(c)
	if _, ok := h.requireActor(c); !ok {
		return
	}
	geofenceID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	preview, err := h.service.PreviewGeofenceCandidates(
		c.Request.Context(),
		geofenceID,
		visibleGroupsFromContext(c),
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, preview)
}

func (h *Handler) RegisterThirdPartyRoutes(group *gin.RouterGroup) {
	group.POST("/fence/batchUpdateDeviceLocation", h.BatchUpdateDeviceLocation)
}

type thirdPartyLocationDevicePayload struct {
	VesselName       string `json:"vesselName"`
	LegacySerialName string `json:"serialName"`
	SerialNumber     string `json:"serialNumber"`
	Longitude        string `json:"longitude"`
	Latitude         string `json:"latitude"`
	UpdateTime       string `json:"updateTime"`
	LegacyUpdateTime string `json:"updatetime"`
}

type thirdPartyLocationRequestPayload struct {
	Devices []thirdPartyLocationDevicePayload `json:"devices"`
}

func (payload thirdPartyLocationRequestPayload) canonicalRequest() ThirdPartyLocationRequest {
	request := ThirdPartyLocationRequest{
		Devices: make([]ThirdPartyLocationDevice, 0, len(payload.Devices)),
	}
	for _, item := range payload.Devices {
		vesselName := item.VesselName
		if strings.TrimSpace(vesselName) == "" {
			vesselName = item.LegacySerialName
		}
		updateTime := item.UpdateTime
		if strings.TrimSpace(updateTime) == "" {
			updateTime = item.LegacyUpdateTime
		}
		request.Devices = append(request.Devices, ThirdPartyLocationDevice{
			VesselName:   vesselName,
			SerialNumber: item.SerialNumber,
			Longitude:    item.Longitude,
			Latitude:     item.Latitude,
			UpdateTime:   updateTime,
		})
	}
	return request
}

func (h *Handler) BatchUpdateDeviceLocation(c *gin.Context) {
	var payload thirdPartyLocationRequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Request parameter error",
		})
		return
	}
	request := payload.canonicalRequest()
	result, replayed, err := h.service.UpdateThirdPartyLocationsIdempotent(
		c.Request.Context(),
		request,
		c.GetHeader("Idempotency-Key"),
	)
	if err != nil {
		if errors.Is(err, ErrThirdPartyLocationBatchConflict) {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
			return
		}
		if errors.Is(err, ErrThirdPartyLocationBatchInProgress) {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": func() string {
			if replayed {
				return "Batch update result replayed"
			}
			return "Batch update completed successfully"
		}(),
		"data": result,
	})
}

func (h *Handler) ListVersions(c *gin.Context) {
	geofenceID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	versions, err := h.service.ListVersions(
		c.Request.Context(),
		geofenceID,
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, VersionList{Items: versions})
}

func (h *Handler) ListMapDefinitions(c *gin.Context) {
	visibleGroups := visibleGroupsFromContext(c)
	bounds, err := ParseMapBounds(c.Query("bounds"))
	if err != nil {
		h.abort(c, err)
		return
	}
	items, err := h.service.ListMapDefinitions(
		c.Request.Context(),
		MapDefinitionFilter{
			Bounds:        bounds,
			Carrier:       strings.ToLower(strings.TrimSpace(c.Query("carrier"))),
			Status:        DefinitionStatus(strings.TrimSpace(c.Query("status"))),
			Name:          strings.TrimSpace(c.Query("name")),
			VisibleGroups: visibleGroups,
		},
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, MapDefinitionList{Items: items})
}

type manualBindingBody struct {
	DeviceIDs          []uuid.UUID `json:"device_ids"`
	DeviceSNs          []string    `json:"device_sns"`
	PreviewFingerprint string      `json:"preview_fingerprint"`
	Reason             string      `json:"reason"`
	ScheduledAt        time.Time   `json:"scheduled_at"`
}

func (h *Handler) PreviewManualBindings(c *gin.Context) {
	admin.SkipAudit(c)
	if _, ok := h.requireActor(c); !ok {
		return
	}
	visibleGroups := visibleGroupsFromContext(c)
	geofenceID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	var body manualBindingBody
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	preview, err := h.service.PreviewManualBindings(
		c.Request.Context(),
		ManualBindingPreviewRequest{
			GeofenceID: geofenceID,
			Inputs: BindingInputRequest{
				DeviceIDs: body.DeviceIDs,
				DeviceSNs: body.DeviceSNs,
			},
			VisibleGroups: visibleGroups,
		},
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, preview)
}

func (h *Handler) CreateManualBindJob(c *gin.Context) {
	actorID, ok := h.requireActor(c)
	if !ok {
		return
	}
	visibleGroups := visibleGroupsFromContext(c)
	geofenceID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	auditEntry := beginGeofenceAudit(
		c,
		auditActionBatchBind,
		audit.ResourceGeofence,
		geofenceID.String(),
		nil,
	)
	var body manualBindingBody
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	accepted, err := h.service.CreateManualBindJob(
		c.Request.Context(),
		CreateManualBindJobRequest{
			GeofenceID:    geofenceID,
			ActorID:       actorID,
			VisibleGroups: visibleGroups,
			Inputs: BindingInputRequest{
				DeviceIDs: body.DeviceIDs,
				DeviceSNs: body.DeviceSNs,
			},
			PreviewFingerprint: body.PreviewFingerprint,
			Reason:             body.Reason,
			ScheduledAt:        body.ScheduledAt,
		},
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	if auditEntry != nil {
		auditEntry.Details = map[string]interface{}{
			"job_id":          accepted.JobID.String(),
			"reason":          strings.TrimSpace(body.Reason),
			"device_id_count": len(body.DeviceIDs),
			"device_sn_count": len(body.DeviceSNs),
		}
	}
	response.OKWithStatus(c, http.StatusAccepted, accepted)
}

func (h *Handler) GetManualBindJob(c *gin.Context) {
	actorID, ok := h.requireActor(c)
	if !ok {
		return
	}
	jobID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	job, err := h.service.GetManualBindJob(
		c.Request.Context(),
		jobID,
		actorID,
		superAdministrator(c),
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, job)
}

func (h *Handler) ListManualBindItems(c *gin.Context) {
	actorID, ok := h.requireActor(c)
	if !ok {
		return
	}
	visibleGroups, ok := h.requireVisibleGroups(c)
	if !ok {
		return
	}
	jobID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	page, ok := parsePositiveQueryInt(c, "page")
	if !ok {
		return
	}
	pageSize, ok := parsePositiveQueryInt(c, "page_size")
	if !ok {
		return
	}
	items, err := h.service.ListManualBindItems(
		c.Request.Context(),
		BatchItemFilter{
			JobID:         jobID,
			Status:        BatchItemStatus(c.Query("status")),
			Page:          page,
			PageSize:      pageSize,
			VisibleGroups: visibleGroups,
		},
		actorID,
		superAdministrator(c),
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, items)
}

func (h *Handler) PreviewEnable(c *gin.Context) {
	h.previewDefinitionTransition(c, DefinitionStatusEnabled)
}

func (h *Handler) PreviewDisable(c *gin.Context) {
	h.previewDefinitionTransition(c, DefinitionStatusDisabled)
}

func (h *Handler) PreviewArchive(c *gin.Context) {
	h.previewDefinitionTransition(c, DefinitionStatusArchived)
}

func (h *Handler) previewDefinitionTransition(
	c *gin.Context,
	target DefinitionStatus,
) {
	admin.SkipAudit(c)
	geofenceID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	impact, err := h.service.PreviewDefinitionTransition(
		c.Request.Context(),
		geofenceID,
		target,
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, impact)
}

func (h *Handler) EnableDefinition(c *gin.Context) {
	h.transitionDefinition(c, DefinitionStatusEnabled)
}

func (h *Handler) DisableDefinition(c *gin.Context) {
	h.transitionDefinition(c, DefinitionStatusDisabled)
}

func (h *Handler) ArchiveDefinition(c *gin.Context) {
	h.transitionDefinition(c, DefinitionStatusArchived)
}

func (h *Handler) transitionDefinition(
	c *gin.Context,
	target DefinitionStatus,
) {
	actorID, ok := authenticatedActorID(c)
	if !ok {
		commonerrors.AbortWithError(
			c,
			http.StatusUnauthorized,
			errors.New("authenticated actor is required"),
		)
		return
	}
	geofenceID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	auditEntry := beginGeofenceAudit(
		c,
		definitionTransitionAuditAction(target),
		audit.ResourceGeofence,
		geofenceID.String(),
		map[string]interface{}{"target_status": target},
	)
	var body struct {
		Reason             string `json:"reason" binding:"required"`
		PreviewFingerprint string `json:"preview_fingerprint" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if auditEntry != nil {
		auditEntry.Details["reason"] = strings.TrimSpace(body.Reason)
	}
	if err := h.service.TransitionDefinition(
		c.Request.Context(),
		geofenceID,
		target,
		actorID,
		body.Reason,
		body.PreviewFingerprint,
	); err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, gin.H{"status": "executed"})
}

func (h *Handler) CreateDraftVersion(c *gin.Context) {
	actorID, ok := authenticatedActorID(c)
	if !ok {
		commonerrors.AbortWithError(
			c,
			http.StatusUnauthorized,
			errors.New("authenticated actor is required"),
		)
		return
	}
	geofenceID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	auditEntry := beginGeofenceAudit(
		c,
		auditActionModify,
		audit.ResourceGeofence,
		geofenceID.String(),
		nil,
	)
	var body struct {
		Geometry json.RawMessage `json:"geometry" binding:"required"`
		Policy   json.RawMessage `json:"policy" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	version, err := h.service.CreateDraftVersion(
		c.Request.Context(),
		CreateVersionRequest{
			GeofenceID: geofenceID,
			Geometry:   body.Geometry,
			Policy:     body.Policy,
			ActorID:    actorID,
		},
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	if auditEntry != nil {
		auditEntry.Details = map[string]interface{}{
			"version_id": version.ID.String(),
			"version":    version.Version,
		}
	}
	response.OKWithStatus(c, http.StatusCreated, version)
}

func (h *Handler) SuspendBinding(c *gin.Context) {
	actorID, ok := h.requireActor(c)
	if !ok {
		return
	}
	bindingID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	auditEntry := beginGeofenceAudit(
		c,
		auditActionBindingSuspend,
		audit.ResourceGeofenceBinding,
		bindingID.String(),
		map[string]interface{}{"target_status": BindingStatusSuspended},
	)
	var body struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if auditEntry != nil {
		auditEntry.Details["reason"] = strings.TrimSpace(body.Reason)
	}
	binding, err := h.service.SuspendBinding(
		c.Request.Context(),
		bindingID,
		actorID,
		body.Reason,
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, binding)
}

func (h *Handler) ResumeBinding(c *gin.Context) {
	actorID, ok := h.requireActor(c)
	if !ok {
		return
	}
	bindingID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	auditEntry := beginGeofenceAudit(
		c,
		auditActionBindingResume,
		audit.ResourceGeofenceBinding,
		bindingID.String(),
		map[string]interface{}{"target_status": BindingStatusActive},
	)
	var body struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if auditEntry != nil {
		auditEntry.Details["reason"] = strings.TrimSpace(body.Reason)
	}
	binding, err := h.service.ResumeBinding(
		c.Request.Context(),
		bindingID,
		actorID,
		body.Reason,
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, binding)
}

func (h *Handler) RemoveBinding(c *gin.Context) {
	actorID, ok := h.requireActor(c)
	if !ok {
		return
	}
	bindingID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	auditEntry := beginGeofenceAudit(
		c,
		auditActionBindingRemove,
		audit.ResourceGeofenceBinding,
		bindingID.String(),
		map[string]interface{}{"target_status": BindingStatusRemoved},
	)
	var body struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if auditEntry != nil {
		auditEntry.Details["reason"] = strings.TrimSpace(body.Reason)
	}
	binding, err := h.service.RemoveBinding(
		c.Request.Context(),
		bindingID,
		actorID,
		body.Reason,
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, binding)
}

func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.service.GetSettingsForVisibleGroups(
		c.Request.Context(),
		visibleGroupsFromContext(c),
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, settings)
}

func (h *Handler) PreviewSettings(c *gin.Context) {
	admin.SkipAudit(c)
	var settings Settings
	if err := c.ShouldBindJSON(&settings); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := ValidateSettings(settings); err != nil {
		h.abort(c, err)
		return
	}
	if err := h.authorizeSettingsChange(c, settings); err != nil {
		h.abort(c, err)
		return
	}
	preview, err := h.service.PreviewSettings(c.Request.Context(), settings)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, preview)
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	actorID, ok := authenticatedActorID(c)
	if !ok {
		commonerrors.AbortWithError(
			c,
			http.StatusUnauthorized,
			errors.New("authenticated actor is required"),
		)
		return
	}
	auditEntry := beginGeofenceAudit(
		c,
		auditActionSettingsUpdate,
		audit.ResourceGeofenceSettings,
		"system",
		nil,
	)
	var settings Settings
	if err := c.ShouldBindJSON(&settings); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := ValidateSettings(settings); err != nil {
		h.abort(c, err)
		return
	}
	if err := h.authorizeSettingsChange(c, settings); err != nil {
		h.abort(c, err)
		return
	}
	if auditEntry != nil {
		auditEntry.Details = settingsAuditDetails(settings)
	}
	updated, err := h.service.UpdateSettings(
		c.Request.Context(),
		settings,
		actorID,
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	if auditEntry != nil {
		auditEntry.Details = settingsAuditDetails(updated)
	}
	response.OK(c, updated)
}

func (h *Handler) authorizeSettingsChange(c *gin.Context, proposed Settings) error {
	if superAdministrator(c) {
		return nil
	}
	current, err := h.service.GetSettings(c.Request.Context())
	if err != nil {
		return err
	}
	if proposed.SystemMode != current.SystemMode {
		return commonerrors.ErrForbidden
	}
	for _, setting := range proposed.Carriers {
		if err := h.service.AuthorizeCarrierAccess(
			c.Request.Context(),
			setting.Carrier,
			visibleGroupsFromContext(c),
		); err != nil {
			return err
		}
	}
	return nil
}

type createDefinitionBody struct {
	Name          string          `json:"name" binding:"required"`
	Carrier       string          `json:"carrier" binding:"required"`
	RuleType      RuleType        `json:"rule_type" binding:"required"`
	OwnerDeviceID *uuid.UUID      `json:"owner_device_id"`
	Geometry      json.RawMessage `json:"geometry" binding:"required"`
	Policy        json.RawMessage `json:"policy" binding:"required"`
}

func (h *Handler) CreateDefinition(c *gin.Context) {
	actorID, ok := authenticatedActorID(c)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, errors.New("authenticated actor is required"))
		return
	}
	auditEntry := beginGeofenceAudit(
		c,
		auditActionCreate,
		audit.ResourceGeofence,
		"",
		nil,
	)
	var body createDefinitionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	visibleGroups := visibleGroupsFromContext(c)
	if auditEntry != nil {
		auditEntry.Details = map[string]interface{}{
			"name":      strings.TrimSpace(body.Name),
			"carrier":   strings.TrimSpace(body.Carrier),
			"rule_type": body.RuleType,
		}
		if body.OwnerDeviceID != nil {
			auditEntry.Details["owner_device_id"] = body.OwnerDeviceID.String()
		}
	}
	result, err := h.service.CreateDefinition(c.Request.Context(), CreateDefinitionRequest{
		Name: body.Name, Carrier: body.Carrier, RuleType: body.RuleType,
		OwnerDeviceID: body.OwnerDeviceID, Geometry: body.Geometry,
		Policy: body.Policy, ActorID: actorID,
		VisibleGroups: visibleGroups,
	})
	if err != nil {
		h.abort(c, err)
		return
	}
	if auditEntry != nil {
		auditEntry.ResourceID = result.Definition.ID.String()
		auditEntry.Details["name"] = result.Definition.Name
		auditEntry.Details["carrier"] = result.Definition.Carrier
		auditEntry.Details["rule_type"] = result.Definition.RuleType
		auditEntry.Details["draft_version_id"] =
			result.DraftVersion.ID.String()
	}
	response.OKWithStatus(c, http.StatusCreated, result)
}

func (h *Handler) ListDefinitions(c *gin.Context) {
	visibleGroups := visibleGroupsFromContext(c)
	definitions, err := h.service.ListDefinitions(c.Request.Context(), DefinitionFilter{
		Carrier:       c.Query("carrier"),
		Status:        DefinitionStatus(c.Query("status")),
		Name:          c.Query("name"),
		VisibleGroups: visibleGroups,
	})
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, definitions)
}

func (h *Handler) GetDefinition(c *gin.Context) {
	id, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	definition, err := h.service.GetDefinition(c.Request.Context(), id)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, definition)
}

func (h *Handler) RenameDefinition(c *gin.Context) {
	actorID, ok := h.requireActor(c)
	if !ok {
		return
	}
	geofenceID, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	auditEntry := beginGeofenceAudit(
		c,
		auditActionModify,
		audit.ResourceGeofence,
		geofenceID.String(),
		map[string]interface{}{},
	)
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if auditEntry != nil {
		auditEntry.Details["name"] = strings.TrimSpace(body.Name)
	}
	if err := h.service.RenameDefinition(c.Request.Context(), geofenceID, body.Name, actorID); err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, gin.H{"status": "executed"})
}

func (h *Handler) PublishDraft(c *gin.Context) {
	actorID, ok := authenticatedActorID(c)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, errors.New("authenticated actor is required"))
		return
	}
	id, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	auditEntry := beginGeofenceAudit(
		c,
		auditActionPublish,
		audit.ResourceGeofence,
		id.String(),
		nil,
	)
	var body struct {
		VersionID uuid.UUID `json:"version_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if auditEntry != nil {
		auditEntry.Details = map[string]interface{}{
			"version_id": body.VersionID.String(),
		}
	}
	if err := h.service.PublishDraft(c.Request.Context(), id, body.VersionID, actorID); err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, gin.H{"status": "executed"})
}

func (h *Handler) ListBindings(c *gin.Context) {
	id, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	visibleGroups := visibleGroupsFromContext(c)
	page, ok := parsePositiveQueryInt(c, "page")
	if !ok {
		return
	}
	pageSize, ok := parsePositiveQueryInt(c, "page_size")
	if !ok {
		return
	}
	bindings, err := h.service.ListBindingDetails(
		c.Request.Context(),
		BindingDetailFilter{
			GeofenceID:    id,
			Status:        BindingStatus(strings.TrimSpace(c.Query("status"))),
			Keyword:       strings.TrimSpace(c.Query("keyword")),
			Page:          page,
			PageSize:      pageSize,
			VisibleGroups: visibleGroups,
		},
	)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, bindings)
}

func (h *Handler) ExportBindings(c *gin.Context) {
	id, ok := parsePathUUID(c, "id")
	if !ok {
		return
	}
	filter := BindingDetailFilter{
		GeofenceID:    id,
		Status:        BindingStatus(strings.TrimSpace(c.Query("status"))),
		Keyword:       strings.TrimSpace(c.Query("keyword")),
		Page:          1,
		PageSize:      MaxBindingPageSize,
		VisibleGroups: visibleGroupsFromContext(c),
	}
	items := make([]BindingDetail, 0)
	for {
		page, err := h.service.ListBindingDetails(c.Request.Context(), filter)
		if err != nil {
			h.abort(c, err)
			return
		}
		items = append(items, page.Items...)
		if len(page.Items) == 0 || int64(len(items)) >= page.Total {
			break
		}
		filter.Page++
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header(
		"Content-Disposition",
		`attachment; filename="geofence-bindings.csv"`,
	)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{
		"device_sn",
		"device_name",
		"carrier",
		"device_group",
		"binding_status",
		"bound_at",
		"confirmed_state",
		"evaluation_health",
	})
	for _, item := range items {
		boundAt := ""
		if !item.BoundAt.IsZero() {
			boundAt = item.BoundAt.UTC().Format(time.RFC3339)
		}
		_ = writer.Write([]string{
			item.DeviceSN,
			item.DeviceName,
			item.DeviceCarrier,
			item.DeviceGroupName,
			string(item.Status),
			boundAt,
			string(item.Evaluation.ConfirmedState),
			item.Evaluation.EvaluationHealth,
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		h.logger.Warn("write geofence bindings export", zap.Error(err))
	}
}

func (h *Handler) abort(c *gin.Context, err error) {
	admin.SetBusinessAuditError(c, err)
	status := commonerrors.HTTPStatusFromError(err)
	if errors.Is(err, ErrInvalidGeometry) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, ErrNoEligibleBindingInputs) {
		status = http.StatusUnprocessableEntity
	}
	h.logger.Warn("geofence request failed", zap.Int("status", status), zap.Error(err))
	if status >= http.StatusInternalServerError {
		err = commonerrors.ErrInternal
	}
	commonerrors.AbortWithError(c, status, err)
}

func beginGeofenceAudit(
	c *gin.Context,
	action string,
	resourceType string,
	resourceID string,
	details map[string]interface{},
) *audit.Entry {
	return admin.SetBusinessAudit(c, audit.Entry{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
	})
}

func definitionTransitionAuditAction(target DefinitionStatus) string {
	switch target {
	case DefinitionStatusEnabled:
		return auditActionEnable
	case DefinitionStatusDisabled:
		return auditActionDisable
	case DefinitionStatusArchived:
		return auditActionArchive
	default:
		return audit.ActionConfig + "_geofence_transition"
	}
}

func settingsAuditDetails(settings Settings) map[string]interface{} {
	carriers := make(map[string]RuntimeMode, len(settings.Carriers))
	for _, setting := range settings.Carriers {
		carriers[setting.Carrier] = setting.Mode
	}
	return map[string]interface{}{
		"system_mode": settings.SystemMode,
		"carriers":    carriers,
	}
}

func (h *Handler) requireVisibleGroups(
	c *gin.Context,
) ([]uuid.UUID, bool) {
	groups, err := h.resolver.ResolveFromContext(c)
	if err != nil {
		h.abort(c, err)
		return nil, false
	}
	if groups != nil && len(groups) == 0 {
		commonerrors.AbortWithError(
			c,
			http.StatusForbidden,
			commonerrors.ErrForbidden,
		)
		return nil, false
	}
	return groups, true
}

const geofenceVisibleGroupsContextKey = "geofence_visible_groups"

func (h *Handler) resolveVisibleGroups(c *gin.Context) {
	visibleGroups, ok := h.requireVisibleGroups(c)
	if !ok {
		c.Abort()
		return
	}
	c.Set(geofenceVisibleGroupsContextKey, visibleGroups)
	c.Next()
}

func visibleGroupsFromContext(c *gin.Context) []uuid.UUID {
	value, _ := c.Get(geofenceVisibleGroupsContextKey)
	visibleGroups, _ := value.([]uuid.UUID)
	return visibleGroups
}

func (h *Handler) authorizeDefinitionRoute(c *gin.Context) {
	geofenceID, ok := parsePathUUID(c, "id")
	if !ok {
		c.Abort()
		return
	}
	if err := h.service.AuthorizeDefinitionAccess(
		c.Request.Context(),
		geofenceID,
		visibleGroupsFromContext(c),
	); err != nil {
		h.abort(c, err)
		c.Abort()
		return
	}
	c.Next()
}

func (h *Handler) authorizeBindingRoute(c *gin.Context) {
	bindingID, ok := parsePathUUID(c, "id")
	if !ok {
		c.Abort()
		return
	}
	if err := h.service.AuthorizeBindingAccess(
		c.Request.Context(),
		bindingID,
		visibleGroupsFromContext(c),
	); err != nil {
		h.abort(c, err)
		c.Abort()
		return
	}
	c.Next()
}

func (h *Handler) requireActor(c *gin.Context) (uuid.UUID, bool) {
	actorID, ok := authenticatedActorID(c)
	if !ok {
		commonerrors.AbortWithError(
			c,
			http.StatusUnauthorized,
			errors.New("authenticated actor is required"),
		)
		return uuid.Nil, false
	}
	return actorID, true
}

func superAdministrator(c *gin.Context) bool {
	value, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	superAdministrator, _ := value.(bool)
	return superAdministrator
}

func parsePositiveQueryInt(c *gin.Context, name string) (int, bool) {
	raw, exists := c.GetQuery(name)
	if !exists {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		commonerrors.AbortWithError(
			c,
			http.StatusBadRequest,
			commonerrors.ErrInvalidInput,
		)
		return 0, false
	}
	return value, true
}

func authenticatedActorID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(admin.CtxKeyUserID)
	id, ok := value.(uuid.UUID)
	return id, exists && ok && id != uuid.Nil
}

func parsePathUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		commonerrors.AbortWithError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("invalid %s: %w", name, commonerrors.ErrInvalidInput),
		)
		return uuid.Nil, false
	}
	return id, true
}

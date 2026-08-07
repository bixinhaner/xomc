package provision

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/quicksettings"
	"github.com/omcgo/omcgo/internal/storageprotection"
	devtask "github.com/omcgo/omcgo/internal/task"
)

// deviceExistenceChecker is the minimal read-only dependency the handler needs
// to verify that a device exists before creating a provisioning task for it.
// In production it is satisfied by *device.DeviceService.GetDevice; tests inject
// a stub. GetDevice returns (nil, nil) when the device is absent.
type deviceExistenceChecker interface {
	GetDevice(ctx context.Context, id uuid.UUID) (*model.Device, error)
}

// Handler provides HTTP handlers for provisioning REST API.
type Handler struct {
	repo           ProvisioningTaskRepository
	engine         *ProvisioningEngine
	deviceChecker  deviceExistenceChecker
	policyRepo     PlugAndPlayPolicyRepository
	xmlRepo        ProvisioningXMLRepository
	deviceService  plugAndPlayDeviceService
	taskEnqueuer   plugAndPlayTaskEnqueuer
	quickSettings  *quicksettings.Registry
	paramModels    plugAndPlayParamModelLookup
	paramMappings  plugAndPlayParamMappingLookup
	fileWorkflow   plugAndPlayFileWorkflow
	xmlObjectStore plugAndPlayXMLObjectStore
	xmlBucket      string
	admission      storageprotection.WriteAdmission
}

type plugAndPlayDeviceService interface {
	GetDevice(context.Context, uuid.UUID) (*model.Device, error)
	ListDevices(context.Context, device.DeviceFilter) (*model.ListResponse[model.Device], error)
}

type plugAndPlayTaskEnqueuer interface {
	CreateTask(context.Context, *devtask.CreateTaskRequest) (*devtask.Task, error)
}

type plugAndPlayParamModelLookup interface {
	LookupParamModelNameByID(context.Context, uuid.UUID) (string, error)
}

type plugAndPlayParamMappingLookup interface {
	GetByProduct(context.Context, uuid.UUID, string) (*parammodel.MappingSet, error)
	GetByParamModel(context.Context, uuid.UUID) (*parammodel.MappingSet, error)
}

// plugAndPlayFileWorkflow reuses unified file management for firmware and
// license only. Final parameter configuration is dispatched directly as a
// product-specific TR-069 Download by enqueueXMLDownload.
type plugAndPlayFileWorkflow interface {
	ExecuteUpgrade(
		context.Context,
		string,
		string,
		bool,
		[]*model.Device,
		string,
	) (uuid.UUID, error)
	ExecuteLicense(context.Context, string, []*model.Device, string) (uuid.UUID, error)
}

type plugAndPlayXMLObjectStore interface {
	PutObject(
		context.Context,
		string,
		string,
		io.Reader,
		int64,
		minio.PutObjectOptions,
	) (minio.UploadInfo, error)
}

// NewHandler creates a new provisioning REST API handler.
func NewHandler(repo ProvisioningTaskRepository, engine *ProvisioningEngine) *Handler {
	return &Handler{repo: repo, engine: engine}
}

// SetDeviceChecker wires the read-only device existence checker used by Create
// to reject provisioning tasks for non-existent devices (returns 404 instead of
// silently creating an orphan task). When nil the precheck is skipped.
func (h *Handler) SetDeviceChecker(c deviceExistenceChecker) {
	h.deviceChecker = c
}

func (h *Handler) SetPlugAndPlayDependencies(
	policies PlugAndPlayPolicyRepository,
	xmlFiles ProvisioningXMLRepository,
	devices plugAndPlayDeviceService,
	tasks plugAndPlayTaskEnqueuer,
) {
	h.policyRepo, h.xmlRepo, h.deviceService, h.taskEnqueuer = policies, xmlFiles, devices, tasks
}

// SetPlugAndPlayXMLDependencies wires both parameter sources used by XML
// compilation: quick-settings first, then the product parameter mappings.
func (h *Handler) SetPlugAndPlayXMLDependencies(
	registry *quicksettings.Registry,
	paramModels plugAndPlayParamModelLookup,
	paramMappings plugAndPlayParamMappingLookup,
) {
	h.quickSettings, h.paramModels, h.paramMappings = registry, paramModels, paramMappings
}

// SetPlugAndPlayXMLStorage makes generated XML use the same MinIO-backed ACS
// Download path as firmware and license files.
func (h *Handler) SetPlugAndPlayXMLStorage(store plugAndPlayXMLObjectStore, bucket string) {
	h.xmlObjectStore, h.xmlBucket = store, strings.TrimSpace(bucket)
}

func (h *Handler) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	h.admission = admission
}

// SetPlugAndPlayFileWorkflow wires firmware and license execution services into
// plug-and-play without routing final parameter configuration through them.
func (h *Handler) SetPlugAndPlayFileWorkflow(workflow plugAndPlayFileWorkflow) {
	h.fileWorkflow = workflow
}

// RegisterRoutes registers provisioning routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	prov := rg.Group("/provisioning/tasks")
	{
		prov.GET("", h.List)
		prov.GET("/:id", h.Get)
		prov.POST("", h.Create)
		prov.POST("/:id/retry", h.Retry)
	}
	policies := rg.Group("/provisioning/policies")
	{
		policies.GET("", h.ListPolicies)
		policies.POST("", h.CreatePolicy)
		policies.GET("/:id", h.GetPolicy)
		policies.PUT("/:id", h.UpdatePolicy)
		policies.PATCH("/:id/enabled", h.SetPolicyEnabled)
		policies.DELETE("/:id", h.DeletePolicy)
		policies.GET("/:id/devices", h.DetectDevices)
		policies.POST("/:id/execute", h.ExecutePolicy)
	}
	rg.GET("/provisioning/xml-files/:id", h.GetXML)
	rg.GET("/provisioning/xml-files/:id/download", h.DownloadXML)
}

// RegisterPublicRoutes exposes the one-time CPE-facing XML download URL. It is
// outside JWT auth because a TR-069 device cannot send an OMC bearer token;
// access is guarded by the per-file checksum token embedded in the Download RPC.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/provisioning/xml-downloads/:id", h.DownloadXMLPublic)
}

// List handles GET /api/v1/provisioning/tasks.
func (h *Handler) List(c *gin.Context) {
	filter := ProvisioningTaskFilter{
		Page:     1,
		PageSize: 20,
	}
	if status := c.Query("status"); status != "" {
		filter.Status = ProvisioningState(status)
	}
	filter.PolicyOnly = c.Query("policy_only") == "true"
	if value, err := strconv.Atoi(c.Query("page")); err == nil && value > 0 {
		filter.Page = value
	}
	if value, err := strconv.Atoi(c.Query("page_size")); err == nil && value > 0 {
		filter.PageSize = value
	}
	if deviceID := c.Query("device_id"); deviceID != "" {
		id, err := uuid.Parse(deviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		filter.DeviceID = &id
	}

	items, total, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": total})
}

func (h *Handler) requirePolicyRepo(c *gin.Context) bool {
	if h.policyRepo == nil {
		response.Fail(c, http.StatusServiceUnavailable, "plug and play service unavailable")
		return false
	}
	return true
}

func (h *Handler) ListPolicies(c *gin.Context) {
	if !h.requirePolicyRepo(c) {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.policyRepo.ListPolicies(c.Request.Context(), PolicyFilter{
		ProductClass: strings.TrimSpace(c.Query("product_class")),
		Search:       strings.TrimSpace(c.Query("search")), Page: page, PageSize: size,
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}

func bindPolicy(c *gin.Context) (*PlugAndPlayPolicy, bool) {
	var policy PlugAndPlayPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return nil, false
	}
	policy.Name = strings.TrimSpace(policy.Name)
	normalizePolicyProductClasses(&policy)
	if policy.Name == "" || policy.ProductClass == "" {
		response.Fail(c, http.StatusBadRequest, "name and product_classes are required")
		return nil, false
	}
	if policy.ExecuteType != "auto" && policy.ExecuteType != "manual" {
		response.Fail(c, http.StatusBadRequest, "execute_type must be auto or manual")
		return nil, false
	}
	if len(policy.Config) == 0 || !json.Valid(policy.Config) {
		policy.Config = []byte(`{}`)
	}
	return &policy, true
}

func (h *Handler) CreatePolicy(c *gin.Context) {
	if !h.requirePolicyRepo(c) {
		return
	}
	policy, ok := bindPolicy(c)
	if !ok {
		return
	}
	if err := h.policyRepo.CreatePolicy(c.Request.Context(), policy); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, policy)
}

func (h *Handler) policyID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid policy ID")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) GetPolicy(c *gin.Context) {
	if !h.requirePolicyRepo(c) {
		return
	}
	id, ok := h.policyID(c)
	if !ok {
		return
	}
	policy, err := h.policyRepo.GetPolicy(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, policy)
}

func (h *Handler) UpdatePolicy(c *gin.Context) {
	if !h.requirePolicyRepo(c) {
		return
	}
	id, ok := h.policyID(c)
	if !ok {
		return
	}
	policy, ok := bindPolicy(c)
	if !ok {
		return
	}
	policy.ID = id
	if err := h.policyRepo.UpdatePolicy(c.Request.Context(), policy); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, policy)
}

func (h *Handler) SetPolicyEnabled(c *gin.Context) {
	if !h.requirePolicyRepo(c) {
		return
	}
	id, ok := h.policyID(c)
	if !ok {
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	policy, err := h.policyRepo.GetPolicy(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	policy.Enabled = req.Enabled
	if err := h.policyRepo.UpdatePolicy(c.Request.Context(), policy); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, policy)
}

func (h *Handler) DeletePolicy(c *gin.Context) {
	if !h.requirePolicyRepo(c) {
		return
	}
	id, ok := h.policyID(c)
	if !ok {
		return
	}
	if err := h.policyRepo.DeletePolicy(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DetectDevices(c *gin.Context) {
	if !h.requirePolicyRepo(c) || h.deviceService == nil {
		return
	}
	id, ok := h.policyID(c)
	if !ok {
		return
	}
	policy, err := h.policyRepo.GetPolicy(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))
	search := strings.TrimSpace(c.Query("search"))
	normalizePolicyProductClasses(policy)
	productClasses := strings.Join(policy.ProductClasses, ",")
	filter := device.DeviceFilter{ProductClass: &productClasses}
	filter.Page, filter.PageSize = page, size
	if search != "" {
		filter.Search = &search
	}
	result, err := h.deviceService.ListDevices(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ExecutePolicy(c *gin.Context) {
	if !h.requirePolicyRepo(c) || h.deviceService == nil {
		response.Fail(c, http.StatusServiceUnavailable, "plug and play execution unavailable")
		return
	}
	policyID, ok := h.policyID(c)
	if !ok {
		return
	}
	policy, err := h.policyRepo.GetPolicy(c.Request.Context(), policyID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	if !policy.Enabled {
		response.Fail(c, http.StatusConflict, "plug and play policy is disabled")
		return
	}
	if policy.SelfConfigEnabled && h.xmlRepo == nil {
		response.Fail(c, http.StatusServiceUnavailable, "plug and play parameter configuration unavailable")
		return
	}
	var req struct {
		DeviceIDs []uuid.UUID `json:"device_ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if len(req.DeviceIDs) > 50 {
		response.Fail(c, http.StatusBadRequest, "one execution batch supports at most 50 devices")
		return
	}
	seenDevices := make(map[uuid.UUID]struct{}, len(req.DeviceIDs))
	devices := make([]*model.Device, 0, len(req.DeviceIDs))
	for _, deviceID := range req.DeviceIDs {
		if _, exists := seenDevices[deviceID]; exists {
			response.Fail(c, http.StatusBadRequest, "device_ids contains duplicates")
			return
		}
		seenDevices[deviceID] = struct{}{}
		dev, getErr := h.deviceService.GetDevice(c.Request.Context(), deviceID)
		if getErr != nil {
			commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(getErr), getErr)
			return
		}
		if dev == nil {
			commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
			return
		}
		if !policySupportsProductClass(policy, dev.ProductClass) {
			response.Fail(c, http.StatusUnprocessableEntity,
				fmt.Sprintf("device %s product class %q does not match policy %q",
					dev.SerialNumber, dev.ProductClass, policy.ProductClasses))
			return
		}
		devices = append(devices, dev)
	}

	operator := "system"
	if raw, exists := c.Get("username"); exists {
		if username, valid := raw.(string); valid && strings.TrimSpace(username) != "" {
			operator = username
		}
	}
	if policy.UpgradeEnabled || policy.LicenseEnabled {
		if h.fileWorkflow == nil {
			response.Fail(c, http.StatusServiceUnavailable, "plug and play file workflow unavailable")
			return
		}
		if h.repo == nil {
			response.Fail(c, http.StatusServiceUnavailable, "plug and play task tracking unavailable")
			return
		}
		if policy.UpgradeEnabled && strings.TrimSpace(policy.TargetVersion) == "" {
			response.Fail(c, http.StatusUnprocessableEntity, "target version is required when upgrade is enabled")
			return
		}
	}
	if policy.SelfConfigEnabled {
		if h.taskEnqueuer == nil {
			response.Fail(c, http.StatusServiceUnavailable, "plug and play XML download unavailable")
			return
		}
	}

	tasks := make([]*ProvisioningTask, 0, len(req.DeviceIDs))
	fileTaskIDs := make(map[string]uuid.UUID, 1)
	modules := enabledPolicyModules(policy)
	if len(modules) > 0 {
		module := modules[0]
		if module == policyModuleSelfConfig {
			for _, deviceID := range req.DeviceIDs {
				item, executeErr := h.executeXML(c.Request.Context(), policy, deviceID, operator)
				if executeErr != nil {
					abortPolicyExecution(c, executeErr)
					return
				}
				tasks = append(tasks, item)
			}
		} else {
			delegated, fileTaskID, executeErr := h.executePolicyFileModule(
				c.Request.Context(), policy, module, devices, operator,
			)
			if executeErr != nil {
				abortPolicyExecution(c, executeErr)
				return
			}
			tasks = append(tasks, delegated...)
			fileTaskIDs[string(module)] = fileTaskID
		}
	}
	payload := gin.H{"items": tasks, "total": len(tasks)}
	if len(fileTaskIDs) == 1 {
		for _, id := range fileTaskIDs {
			payload["file_task_id"] = id
		}
	} else if len(fileTaskIDs) > 1 {
		payload["file_task_ids"] = fileTaskIDs
	}
	response.OKWithStatus(c, http.StatusCreated, payload)
}

// ExecuteAutomaticPolicy matches and starts the highest-priority enabled auto
// policy for a device. It is called only after the existing first-registration
// full parameter sync has completed; that sync is deliberately not represented
// as a provisioning step.
func (h *Handler) ExecuteAutomaticPolicy(ctx context.Context, deviceID uuid.UUID) error {
	if h.policyRepo == nil || h.deviceService == nil || h.repo == nil {
		return fmt.Errorf("automatic plug and play execution unavailable")
	}
	dev, err := h.deviceService.GetDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get automatic plug and play device: %w", err)
	}
	if dev == nil {
		return nil
	}
	policies, _, err := h.policyRepo.ListPolicies(ctx, PolicyFilter{
		ProductClass: dev.ProductClass, Page: 1, PageSize: 100,
	})
	if err != nil {
		return fmt.Errorf("list automatic plug and play policies: %w", err)
	}
	var matched *PlugAndPlayPolicy
	for i := range policies {
		candidate := &policies[i]
		if candidate.Enabled && candidate.ExecuteType == "auto" &&
			policySupportsProductClass(candidate, dev.ProductClass) &&
			policyMatchesFirmware(candidate, dev.FirmwareVersion) {
			matched = candidate
			break
		}
	}
	if matched == nil {
		return nil
	}

	// The registered-device sync completion event is at-least-once. A policy
	// task already created for this device is the durable idempotency record.
	latest, getErr := h.repo.GetByDeviceID(ctx, deviceID)
	if getErr == nil && latest != nil && latest.PolicyID != nil && *latest.PolicyID == matched.ID {
		return nil
	}

	modules := enabledPolicyModules(matched)
	if len(modules) == 0 {
		return nil
	}
	first := modules[0]
	if first == policyModuleSelfConfig {
		if h.xmlRepo == nil || h.taskEnqueuer == nil {
			return fmt.Errorf("automatic plug and play parameter configuration unavailable")
		}
		_, err = h.executeXML(ctx, matched, deviceID, "system")
		return err
	}
	if h.fileWorkflow == nil {
		return fmt.Errorf("automatic plug and play file workflow unavailable")
	}
	if first == policyModuleUpgrade && strings.TrimSpace(matched.TargetVersion) == "" {
		return fmt.Errorf("target version is required when upgrade is enabled")
	}
	_, _, err = h.executePolicyFileModule(ctx, matched, first, []*model.Device{dev}, "system")
	return err
}

func policyMatchesFirmware(policy *PlugAndPlayPolicy, firmwareVersion string) bool {
	if policy == nil || len(policy.Config) == 0 {
		return true
	}
	var config struct {
		OriginalVersion any `json:"originalVersion"`
	}
	if err := json.Unmarshal(policy.Config, &config); err != nil || config.OriginalVersion == nil {
		return true
	}
	want := strings.TrimSpace(firmwareVersion)
	switch value := config.OriginalVersion.(type) {
	case string:
		value = strings.TrimSpace(value)
		return value == "" || strings.EqualFold(value, "all") || value == want
	case []any:
		if len(value) == 0 {
			return true
		}
		for _, item := range value {
			version := strings.TrimSpace(fmt.Sprint(item))
			if strings.EqualFold(version, "all") || version == want {
				return true
			}
		}
	}
	return false
}

func abortPolicyExecution(c *gin.Context, err error) {
	var validation *ConfigValidationError
	if errors.As(err, &validation) {
		commonerrors.AbortWithError(c, http.StatusUnprocessableEntity, err)
		return
	}
	commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
}

func (h *Handler) executePolicyFileModule(
	ctx context.Context,
	policy *PlugAndPlayPolicy,
	module policyModule,
	devices []*model.Device,
	operator string,
) ([]*ProvisioningTask, uuid.UUID, error) {
	now := time.Now()
	delegatedTasks := make([]*ProvisioningTask, 0, len(devices))
	for _, dev := range devices {
		item := NewProvisioningTask(dev.ID)
		item.MaxRetries = 0
		item.PolicyID = &policy.ID
		item.Status = StateConfiguring
		item.CurrentStepName = string(module) + "_task_submitted"
		item.TotalSteps = 1
		item.StartedAt = &now
		if err := h.repo.Create(ctx, item); err != nil {
			return nil, uuid.Nil, err
		}
		delegatedTasks = append(delegatedTasks, item)
	}

	var (
		fileTaskID uuid.UUID
		err        error
	)
	if module == policyModuleUpgrade {
		var config struct {
			PreserveSetting bool `json:"preserveSetting"`
		}
		_ = json.Unmarshal(policy.Config, &config)
		fileTaskID, err = h.fileWorkflow.ExecuteUpgrade(
			ctx, policy.Name, policy.TargetVersion, config.PreserveSetting, devices, operator,
		)
	} else {
		fileTaskID, err = h.fileWorkflow.ExecuteLicense(ctx, policy.Name, devices, operator)
	}
	if err != nil {
		for _, item := range delegatedTasks {
			_ = h.repo.UpdateStatus(ctx, item.ID, StateFailed, err.Error())
		}
		return nil, uuid.Nil, err
	}
	for _, item := range delegatedTasks {
		item.DeviceTaskID = &fileTaskID
		if err := h.repo.Update(ctx, item); err != nil {
			return nil, uuid.Nil, err
		}
	}
	return delegatedTasks, fileTaskID, nil
}

// ContinuePolicy is invoked by the provisioning engine after the existing
// upgrade/license workflow publishes its terminal result for one device.
func (h *Handler) ContinuePolicy(
	ctx context.Context,
	policyID uuid.UUID,
	deviceID uuid.UUID,
	completed policyModule,
) error {
	policy, err := h.policyRepo.GetPolicy(ctx, policyID)
	if err != nil {
		return err
	}
	dev, err := h.deviceService.GetDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	modules := enabledPolicyModules(policy)
	for index, module := range modules {
		if module != completed || index+1 >= len(modules) {
			continue
		}
		next := modules[index+1]
		if next == policyModuleSelfConfig {
			_, err = h.executeXML(ctx, policy, deviceID, "system")
			return err
		}
		_, _, err = h.executePolicyFileModule(ctx, policy, next, []*model.Device{dev}, "system")
		return err
	}
	return nil
}

func (h *Handler) executeXML(ctx context.Context, policy *PlugAndPlayPolicy, deviceID uuid.UUID, operator string) (*ProvisioningTask, error) {
	dev, err := h.deviceService.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get plug and play device %s: %w", deviceID, err)
	}
	if dev == nil {
		return nil, fmt.Errorf("get plug and play device %s: %w", deviceID, commonerrors.ErrNotFound)
	}
	if !policySupportsProductClass(policy, dev.ProductClass) {
		return nil, fmt.Errorf("device %s product class %q does not match policy %q", dev.SerialNumber, dev.ProductClass, policy.ProductClasses)
	}
	var (
		paramModel    string
		groups        []quicksettings.Group
		mappings      []parammodel.ParamMapping
		definitionErr error
	)
	if dev.ParamModelID != nil && h.paramModels != nil {
		paramModel, definitionErr = h.paramModels.LookupParamModelNameByID(ctx, *dev.ParamModelID)
		if definitionErr == nil && h.quickSettings != nil {
			groups = h.quickSettings.GetByParamModel(paramModel)
		}
	}
	if h.paramMappings != nil {
		var set *parammodel.MappingSet
		switch {
		case dev.ProductID != nil:
			set, err = h.paramMappings.GetByProduct(ctx, *dev.ProductID, dev.FirmwareVersion)
		case dev.ParamModelID != nil:
			set, err = h.paramMappings.GetByParamModel(ctx, *dev.ParamModelID)
		}
		if err != nil && !errors.Is(err, parammodel.ErrNoMapping) && !errors.Is(err, parammodel.ErrNoParamModel) {
			return nil, fmt.Errorf("lookup device product parameter mappings: %w", err)
		}
		if set != nil {
			mappings = set.Mappings
		}
	}
	if len(groups) == 0 && len(mappings) == 0 && definitionErr != nil {
		return nil, fmt.Errorf("lookup device parameter model: %w", definitionErr)
	}
	compiled, err := CompilePolicyParametersWithMappings(
		policy, dev, paramModel, groups, mappings,
	)
	if err != nil {
		return nil, err
	}
	generatedAt := time.Now().UTC()
	content, err := GenerateAutoStartXML(AutoStartXMLDocument{
		NetworkType: compiled.NetworkType, Vendor: dev.Manufacturer,
		SerialNumber: dev.SerialNumber, GeneratedAt: generatedAt,
		DataModelVersion: compiled.DataModelVersion, Parameters: compiled.Parameters,
		VendorSpecific: compiled.VendorSpecific,
	})
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(content)
	xmlFile := &ProvisioningXMLFile{PolicyID: policy.ID, DeviceID: dev.ID,
		FileName: automaticStartFileName(dev, compiled.DataModelVersion, generatedAt),
		Content:  string(content), Checksum: hex.EncodeToString(sum[:])}
	if err := h.xmlRepo.CreateXML(ctx, xmlFile); err != nil {
		return nil, err
	}
	return h.enqueueXMLDownload(ctx, dev, policy.ID, xmlFile, 0)
}

func plugAndPlayXMLFileType(dev *model.Device) string {
	if dev != nil && dev.Technology == model.TechNR {
		return "103 Base Station Startup File"
	}
	return "Auto Start File"
}

func (h *Handler) enqueueXMLDownload(
	ctx context.Context,
	dev *model.Device,
	policyID uuid.UUID,
	xmlFile *ProvisioningXMLFile,
	retryCount int,
) (*ProvisioningTask, error) {
	pt := NewProvisioningTask(dev.ID)
	pt.MaxRetries = 0
	pt.PolicyID, pt.XMLFileID = &policyID, &xmlFile.ID
	pt.RetryCount = retryCount
	pt.Status, pt.CurrentStep, pt.CurrentStepName, pt.TotalSteps = StateConfiguring, 5, "download_xml", 12
	now := time.Now()
	pt.StartedAt = &now
	downloadURL, err := h.stageXMLDownload(ctx, xmlFile)
	if err != nil {
		return nil, err
	}
	if err := h.repo.Create(ctx, pt); err != nil {
		return nil, err
	}
	md5sum := md5.Sum([]byte(xmlFile.Content))
	noRetries := 0
	params, _ := json.Marshal(map[string]any{
		"file_type": plugAndPlayXMLFileType(dev), "url": downloadURL,
		"target_file_name": xmlFile.FileName, "md5": hex.EncodeToString(md5sum[:]),
		"sha256": xmlFile.Checksum,
	})
	dt, err := h.taskEnqueuer.CreateTask(ctx, &devtask.CreateTaskRequest{
		DeviceSN: dev.SerialNumber, Method: "Download", Params: params,
		Source: devtask.TaskSourceSystem, SourceID: pt.ID.String(),
		CommandKey: "PNPXML_" + pt.ID.String(), Description: "plug and play XML auto start",
		MaxRetries: &noRetries,
	})
	if err != nil {
		_ = h.repo.UpdateStatus(ctx, pt.ID, StateFailed, err.Error())
		return nil, fmt.Errorf("enqueue plug and play XML Download: %w", err)
	}
	if id, parseErr := uuid.Parse(dt.ID); parseErr == nil {
		pt.DeviceTaskID = &id
	}
	if err := h.repo.Update(ctx, pt); err != nil {
		return nil, err
	}
	return pt, nil
}

func automaticStartFileName(dev *model.Device, dataModelVersion string, generatedAt time.Time) string {
	if dev != nil && (dev.Technology == model.TechLTE || dev.Technology == model.TechGSM) {
		return fmt.Sprintf("auto_start_%s.xml", sanitizeFileComponent(dev.SerialNumber))
	}
	return fmt.Sprintf("%s_%s_%s_%s.xml",
		sanitizeFileComponent(dev.SerialNumber),
		sanitizeFileComponent(dev.Manufacturer),
		sanitizeFileComponent(dataModelVersion),
		generatedAt.UTC().Format("20060102150405"),
	)
}

func sanitizeFileComponent(value string) string {
	value = strings.TrimSpace(value)
	var result strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			result.WriteRune(r)
		default:
			result.WriteByte('_')
		}
	}
	if result.Len() == 0 {
		return "unknown"
	}
	return result.String()
}

func (h *Handler) stageXMLDownload(
	ctx context.Context,
	xmlFile *ProvisioningXMLFile,
) (string, error) {
	if h.xmlObjectStore == nil || h.xmlBucket == "" {
		return "", fmt.Errorf("plug and play XML object storage is not configured")
	}
	if xmlFile == nil || xmlFile.ID == uuid.Nil {
		return "", fmt.Errorf("plug and play XML file ID is required")
	}
	if h.admission != nil {
		decision, err := h.admission.Check(ctx, storageprotection.TargetFilesystem, storageprotection.UnifiedStorageTargetID, storageprotection.WriteScopeUpload)
		if err != nil {
			return "", fmt.Errorf("storage admission check: %w", err)
		}
		if !decision.Allowed {
			return "", fmt.Errorf("storage write protected: %s", decision.Reason)
		}
	}
	objectName := fmt.Sprintf("plug-and-play/%s.xml", xmlFile.ID)
	if _, err := h.xmlObjectStore.PutObject(
		ctx,
		h.xmlBucket,
		objectName,
		strings.NewReader(xmlFile.Content),
		int64(len(xmlFile.Content)),
		minio.PutObjectOptions{ContentType: "application/xml; charset=utf-8"},
	); err != nil {
		return "", fmt.Errorf("store plug and play XML for Download: %w", err)
	}
	return h.xmlBucket + "/" + objectName, nil
}

func (h *Handler) GetXML(c *gin.Context) {
	h.serveXML(c, false)
}

func (h *Handler) DownloadXML(c *gin.Context) {
	h.serveXML(c, true)
}

func (h *Handler) DownloadXMLPublic(c *gin.Context) {
	h.serveXML(c, true)
}

func (h *Handler) serveXML(c *gin.Context, attachment bool) {
	if h.xmlRepo == nil {
		response.Fail(c, http.StatusServiceUnavailable, "XML service unavailable")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid XML file ID")
		return
	}
	file, err := h.xmlRepo.GetXML(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	if strings.Contains(c.FullPath(), "/xml-downloads/") &&
		subtle.ConstantTimeCompare([]byte(c.Query("token")), []byte(file.DownloadToken.String())) != 1 {
		response.Fail(c, http.StatusForbidden, "invalid XML download token")
		return
	}
	if attachment {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.FileName))
	}
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(file.Content))
}

// Get handles GET /api/v1/provisioning/tasks/:id.
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid task ID")
		return
	}

	task, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "task not found")
		return
	}
	response.OK(c, task)
}

type createTaskRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// Create handles POST /api/v1/provisioning/tasks (manual provisioning trigger).
func (h *Handler) Create(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid device_id")
		return
	}

	// Existence precheck: provisioning_tasks has no FK on device_id, so a
	// well-formed-but-unknown UUID would otherwise create an orphan task that
	// the 15-minute reaper later fails. Reject up front with 404 (issue #126
	// item 10). GetDevice returns (nil, nil) for an absent device.
	if h.deviceChecker != nil {
		dev, derr := h.deviceChecker.GetDevice(c.Request.Context(), deviceID)
		if derr != nil {
			commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(derr), derr)
			return
		}
		if dev == nil {
			commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
			return
		}
	}

	task := NewProvisioningTask(deviceID)
	if err := h.repo.Create(c.Request.Context(), task); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, task)
}

// Retry handles POST /api/v1/provisioning/tasks/:id/retry.
func (h *Handler) Retry(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid task ID")
		return
	}

	task, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "task not found")
		return
	}

	if task.Status != StateFailed {
		response.Fail(c, http.StatusBadRequest, "only failed tasks can be retried")
		return
	}
	if task.PolicyID != nil {
		response.Fail(c, http.StatusBadRequest,
			"plug-and-play module tasks are not retried; start a new policy execution")
		return
	}
	if task.RetryCount >= task.MaxRetries {
		response.Fail(c, http.StatusBadRequest, "maximum retry count reached")
		return
	}

	// Create a new task for retry.
	newTask := NewProvisioningTask(task.DeviceID)
	newTask.RetryCount = task.RetryCount + 1
	if err := h.repo.Create(c.Request.Context(), newTask); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, newTask)
}

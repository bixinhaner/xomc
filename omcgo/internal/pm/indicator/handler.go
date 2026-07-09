package indicator

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// IndicatorHandler provides REST API endpoints for KPI indicator management.
// It holds a reference to the IndicatorManagementService and delegates all
// business logic to the service layer.
type IndicatorHandler struct {
	svc    *IndicatorManagementService
	logger *zap.Logger
}

// NewIndicatorHandler creates a new indicator management handler.
func NewIndicatorHandler(svc *IndicatorManagementService, logger *zap.Logger) *IndicatorHandler {
	return &IndicatorHandler{svc: svc, logger: logger}
}

// RegisterRoutes registers indicator management API routes.
// Three route groups are created:
//   - /pm/indicatormg       — ENB/GSM indicator CRUD
//   - /gnb/pm/indicatormg   — GNB indicator CRUD
//   - /cell/perfmgmt/kpimanage — enable/disable indicators (ENB/GSM/GNB)
func (h *IndicatorHandler) RegisterRoutes(rg *gin.RouterGroup) {
	h.RegisterReadRoutes(rg)
	h.RegisterWriteRoutes(rg)
}

// RegisterReadRoutes registers indicator read endpoints for authenticated users.
func (h *IndicatorHandler) RegisterReadRoutes(rg *gin.RouterGroup) {
	// ENB/GSM indicator management
	enbGsm := rg.Group("/pm/indicatormg")
	{
		enbGsm.POST("/getIndicatorGroupTree", h.GetIndicatorGroupTree)
		enbGsm.POST("/getIndicatorGroupInfo", h.GetGroupInfo)
		enbGsm.POST("/getIndicatorListByPage", h.GetIndicatorListByPage)
		enbGsm.POST("/getEffectiveIndicators", h.GetEffectiveIndicators)
		enbGsm.POST("/getIndicatorInfo", h.GetIndicatorInfo)
		enbGsm.POST("/exportAllIndicator", h.ExportAllIndicator)
		enbGsm.GET("/getIndicatorUnitList", h.GetIndicatorUnitList)
		enbGsm.POST("/getIndicatorGroupList", h.GetIndicatorGroupList)
		enbGsm.POST("/getIndicatorTypes", h.GetIndicatorTypes)
	}

	// GNB indicator management
	gnb := rg.Group("/gnb/pm/indicatormg")
	{
		gnb.POST("/getIndicatorGroupTree", h.GetGNBIndicatorGroupTree)
		gnb.POST("/getIndicatorGroupInfo", h.GetGNBGroupInfo)
		gnb.POST("/getIndicatorListByPage", h.GetGNBIndicatorListPageData)
		gnb.POST("/getIndicatorInfo", h.GetGNBIndicatorInfo)
		gnb.POST("/exportAllIndicator", h.ExportGNBAllIndicator)
	}

	// Enable/disable indicators (shared by ENB/GSM/GNB)
	kpiManage := rg.Group("/cell/perfmgmt/kpimanage")
	{
		kpiManage.GET("/isIndicatorInTemplate", h.IsIndicatorInTemplate)
	}
}

// RegisterWriteRoutes registers indicator write endpoints for super-admin users.
func (h *IndicatorHandler) RegisterWriteRoutes(rg *gin.RouterGroup) {
	// ENB/GSM indicator management
	enbGsm := rg.Group("/pm/indicatormg")
	{
		enbGsm.POST("/addIndicatorGroup", h.CreateGroup)
		enbGsm.POST("/modifyIndicatorGroup", h.ModifyGroup)
		enbGsm.POST("/delIndicatorGroup", h.DeleteGroup)
		enbGsm.POST("/addOrModifyIndicator", h.AddOrModifyIndicator)
		enbGsm.POST("/delIndicator", h.DeleteIndicator)
		enbGsm.POST("/updateBaseKpiCustName", h.UpdateBaseKpiCustName)
		enbGsm.POST("/updateEnbIndicatorsName", h.UpdateEnbIndicatorsName)
		enbGsm.POST("/updateGsmIndicatorsName", h.UpdateGsmIndicatorsName)
	}

	// GNB indicator management
	gnb := rg.Group("/gnb/pm/indicatormg")
	{
		gnb.POST("/addIndicatorGroup", h.CreateGNBGroup)
		gnb.POST("/modifyIndicatorGroup", h.ModifyGNBGroup)
		gnb.POST("/delIndicatorGroup", h.DeleteGNBGroup)
		gnb.POST("/addOrModifyIndicator", h.AddOrModifyGNBIndicator)
		gnb.POST("/delIndicator", h.DeleteGNBIndicator)
		gnb.POST("/updateBaseKpiCustName", h.UpdateGNBBaseKpiCustName)
		gnb.POST("/updateGnbIndicatorsName", h.UpdateGnbIndicatorsName)
	}

	// Enable/disable indicators (shared by ENB/GSM/GNB)
	kpiManage := rg.Group("/cell/perfmgmt/kpimanage")
	{
		kpiManage.POST("/enableIndicator", h.EnableIndicator)
		kpiManage.POST("/disableIndicator", h.DisableIndicator)
	}
}

// ── Request DTOs ──────────────────────────────────────────────────────────────

// groupTreeRequest is the request body for getIndicatorGroupTree.
type groupTreeRequest struct {
	DeviceType   string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	OperatorCode string `json:"operator_code"`
}

// groupInfoRequest is the request body for getIndicatorGroupInfo.
type groupInfoRequest struct {
	DeviceType string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	ID         string `json:"id" binding:"required"`
}

// deleteGroupRequest is the request body for delIndicatorGroup.
type deleteGroupRequest struct {
	DeviceType string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	ID         string `json:"id" binding:"required"`
}

// indicatorInfoRequest is the request body for getIndicatorInfo.
type indicatorInfoRequest struct {
	DeviceType string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	ID         string `json:"id" binding:"required"`
}

// addOrModifyIndicatorRequest is the request body for addOrModifyIndicator.
// If ID is provided, it's an update; otherwise it's a create.
type addOrModifyIndicatorRequest struct {
	ID            string `json:"id"`
	DeviceType    string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	EnName        string `json:"en_name" binding:"required"`
	CnName        string `json:"cn_name" binding:"required"`
	EnDescription string `json:"en_description"`
	CnDescription string `json:"cn_description"`
	GroupID       string `json:"group_id" binding:"required"`
	OperatorCode  string `json:"operator_code"`
	DataType      string `json:"data_type"`
	UnitID        string `json:"unit_id"`
	Updator       string `json:"updator"`
	IsCounter     string `json:"is_counter"`
	Arithmetic    string `json:"arithmetic"`
	// 对齐老 OMC perf_indicators.statis_type 业务枚举（sum/avg/max/min/pct），驱动 G5 cron 聚合算子。
	StatisType     string `json:"statis_type" binding:"omitempty,oneof=sum avg max min pct"`
	ProductTypes   string `json:"product_types"`
	IndicatorLevel string `json:"indicator_level"`
}

func resolveUpdator(c *gin.Context, requestUpdator string) string {
	if trimmed := strings.TrimSpace(requestUpdator); trimmed != "" {
		return trimmed
	}
	if username, ok := c.Get("username"); ok {
		if name, ok := username.(string); ok {
			if trimmed := strings.TrimSpace(name); trimmed != "" {
				return trimmed
			}
		}
	}
	return "system"
}

// deleteIndicatorRequest is the request body for delIndicator.
type deleteIndicatorRequest struct {
	DeviceType string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	ID         string `json:"id" binding:"required"`
}

// counterNameUpdateRequest is the request body for updateEnbIndicatorsName/updateGsmIndicatorsName/updateGnbIndicatorsName.
type counterNameUpdateRequest struct {
	ID      string `json:"id" binding:"required"`
	NewName string `json:"new_name" binding:"required"`
}

// effectiveIndicatorsRequest is the request body for getEffectiveIndicators.
type effectiveIndicatorsRequest struct {
	DeviceType   string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	OperatorCode string `json:"operator_code" binding:"required"`
}

// isIndicatorInTemplateQuery is the query for isIndicatorInTemplate.
type isIndicatorInTemplateQuery struct {
	IndicatorID string `form:"indicator_id" binding:"required"`
}

// ── ENB/GSM Handlers ─────────────────────────────────────────────────────────

// GetIndicatorGroupTree handles POST /pm/indicatormg/getIndicatorGroupTree.
func (h *IndicatorHandler) GetIndicatorGroupTree(c *gin.Context) {
	var req groupTreeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	tree, err := h.svc.GetGroupTree(c.Request.Context(), IndicatorGroupTreeRequest{
		DeviceType:   req.DeviceType,
		OperatorCode: req.OperatorCode,
	})
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, tree)
}

// CreateGroup handles POST /pm/indicatormg/addIndicatorGroup.
func (h *IndicatorHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	group, err := h.svc.CreateGroup(c.Request.Context(), &req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, group)
}

// GetGroupInfo handles POST /pm/indicatormg/getIndicatorGroupInfo.
func (h *IndicatorHandler) GetGroupInfo(c *gin.Context) {
	var req groupInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	group, err := h.svc.GetGroupByID(c.Request.Context(), dt, req.ID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, group)
}

// ModifyGroup handles POST /pm/indicatormg/modifyIndicatorGroup.
func (h *IndicatorHandler) ModifyGroup(c *gin.Context) {
	var req struct {
		DeviceType  string  `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
		ID          string  `json:"id" binding:"required"`
		EnName      *string `json:"en_name"`
		CnName      *string `json:"cn_name"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	updateReq := &UpdateGroupRequest{
		EnName:      req.EnName,
		CnName:      req.CnName,
		Description: req.Description,
	}
	if err := h.svc.UpdateGroup(c.Request.Context(), dt, req.ID, updateReq); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

// DeleteGroup handles POST /pm/indicatormg/delIndicatorGroup.
func (h *IndicatorHandler) DeleteGroup(c *gin.Context) {
	var req deleteGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.DeleteGroup(c.Request.Context(), dt, req.ID); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

// GetIndicatorListByPage handles POST /pm/indicatormg/getIndicatorListByPage.
func (h *IndicatorHandler) GetIndicatorListByPage(c *gin.Context) {
	var filter IndicatorListFilter
	filter.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindJSON(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.svc.ListIndicators(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

// GetEffectiveIndicators handles POST /pm/indicatormg/getEffectiveIndicators.
func (h *IndicatorHandler) GetEffectiveIndicators(c *gin.Context) {
	var req effectiveIndicatorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	ids, err := h.svc.GetEnabledIndicatorIDs(c.Request.Context(), dt, req.OperatorCode)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, ids)
}

// GetIndicatorInfo handles POST /pm/indicatormg/getIndicatorInfo.
func (h *IndicatorHandler) GetIndicatorInfo(c *gin.Context) {
	var req indicatorInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	indicator, err := h.svc.GetIndicatorInfo(c.Request.Context(), dt, req.ID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, indicator)
}

// AddOrModifyIndicator handles POST /pm/indicatormg/addOrModifyIndicator.
// If the request contains an ID, it updates; otherwise it creates.
func (h *IndicatorHandler) AddOrModifyIndicator(c *gin.Context) {
	var req addOrModifyIndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if req.ID != "" {
		updator := resolveUpdator(c, req.Updator)
		// Update existing indicator
		updateReq := &UpdateIndicatorRequest{
			EnName:        &req.EnName,
			CnName:        &req.CnName,
			EnDescription: &req.EnDescription,
			CnDescription: &req.CnDescription,
			GroupID:       &req.GroupID,
			DataType:      &req.DataType,
			UnitID:        &req.UnitID,
			Updator:       &updator,
			Arithmetic:    &req.Arithmetic,
			StatisType:    &req.StatisType,
		}
		if dt.HasProductTypes() {
			updateReq.ProductTypes = &req.ProductTypes
			updateReq.IndicatorLevel = &req.IndicatorLevel
		}
		if err := h.svc.UpdateIndicator(c.Request.Context(), dt, req.ID, updateReq); err != nil {
			commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
			return
		}
		response.OKWithMsg(c, nil, "success")
		return
	}

	// Create new indicator
	updator := resolveUpdator(c, req.Updator)
	createReq := &CreateIndicatorRequest{
		DeviceType:     req.DeviceType,
		EnName:         req.EnName,
		CnName:         req.CnName,
		EnDescription:  req.EnDescription,
		CnDescription:  req.CnDescription,
		GroupID:        req.GroupID,
		DataType:       req.DataType,
		UnitID:         req.UnitID,
		Updator:        updator,
		IsCounter:      req.IsCounter,
		Arithmetic:     req.Arithmetic,
		StatisType:     req.StatisType,
		ProductTypes:   req.ProductTypes,
		IndicatorLevel: req.IndicatorLevel,
		OperatorCode:   req.OperatorCode,
	}
	indicator, err := h.svc.CreateIndicator(c.Request.Context(), createReq)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, indicator)
}

// DeleteIndicator handles POST /pm/indicatormg/delIndicator.
func (h *IndicatorHandler) DeleteIndicator(c *gin.Context) {
	var req deleteIndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.DeleteIndicator(c.Request.Context(), dt, req.ID); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

// ExportAllIndicator handles POST /pm/indicatormg/exportAllIndicator.
func (h *IndicatorHandler) ExportAllIndicator(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.svc.ExportIndicators(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	filename := fmt.Sprintf("indicators_%s.csv", req.DeviceType)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}

// UpdateBaseKpiCustName handles POST /pm/indicatormg/updateBaseKpiCustName.
func (h *IndicatorHandler) UpdateBaseKpiCustName(c *gin.Context) {
	var req CustNameUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.UpdateCustName(c.Request.Context(), &req); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

// UpdateEnbIndicatorsName handles POST /pm/indicatormg/updateEnbIndicatorsName.
func (h *IndicatorHandler) UpdateEnbIndicatorsName(c *gin.Context) {
	h.updateCounterName(c, DeviceTypeENB)
}

// UpdateGsmIndicatorsName handles POST /pm/indicatormg/updateGsmIndicatorsName.
func (h *IndicatorHandler) UpdateGsmIndicatorsName(c *gin.Context) {
	h.updateCounterName(c, DeviceTypeGSM)
}

// GetIndicatorUnitList handles GET /pm/indicatormg/getIndicatorUnitList.
func (h *IndicatorHandler) GetIndicatorUnitList(c *gin.Context) {
	// Return static unit list. In the future, this could be loaded from a table.
	units := []IndicatorUnit{
		{ID: "1", EnName: "none", CnName: "无"},
		{ID: "2", EnName: "count", CnName: "次数"},
		{ID: "3", EnName: "percent", CnName: "百分比"},
		{ID: "4", EnName: "dbm", CnName: "dBm"},
		{ID: "5", EnName: "db", CnName: "dB"},
		{ID: "6", EnName: "kbps", CnName: "kbps"},
		{ID: "7", EnName: "mbps", CnName: "Mbps"},
		{ID: "8", EnName: "ms", CnName: "毫秒"},
	}
	response.OK(c, units)
}

// GetIndicatorGroupList handles POST /pm/indicatormg/getIndicatorGroupList.
func (h *IndicatorHandler) GetIndicatorGroupList(c *gin.Context) {
	var req struct {
		DeviceType string `json:"device_type" binding:"required,oneof=ENB GSM GNB"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	groups, err := h.svc.GetGroupList(c.Request.Context(), dt)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, groups)
}

// GetIndicatorTypes handles POST /pm/indicatormg/getIndicatorTypes.
func (h *IndicatorHandler) GetIndicatorTypes(c *gin.Context) {
	types := []gin.H{
		{"id": "0", "name": "Counter"},
		{"id": "1", "name": "KPI"},
	}
	response.OK(c, types)
}

// ── Enable/Disable Handlers ──────────────────────────────────────────────────

// EnableIndicator handles POST /cell/perfmgmt/kpimanage/enableIndicator.
func (h *IndicatorHandler) EnableIndicator(c *gin.Context) {
	var req EnableIndicatorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.EnableIndicators(c.Request.Context(), &req); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

// DisableIndicator handles POST /cell/perfmgmt/kpimanage/disableIndicator.
func (h *IndicatorHandler) DisableIndicator(c *gin.Context) {
	var req EnableIndicatorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.DisableIndicators(c.Request.Context(), &req); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

// IsIndicatorInTemplate handles GET /cell/perfmgmt/kpimanage/isIndicatorInTemplate.
func (h *IndicatorHandler) IsIndicatorInTemplate(c *gin.Context) {
	var q isIndicatorInTemplateQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	exists, err := h.svc.IsIndicatorInTemplate(c.Request.Context(), q.IndicatorID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, exists)
}

// ── GNB Handlers ─────────────────────────────────────────────────────────────

// GetGNBIndicatorGroupTree handles POST /gnb/pm/indicatormg/getIndicatorGroupTree.
func (h *IndicatorHandler) GetGNBIndicatorGroupTree(c *gin.Context) {
	var req groupTreeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Override to GNB
	req.DeviceType = string(DeviceTypeGNB)

	tree, err := h.svc.GetGroupTree(c.Request.Context(), IndicatorGroupTreeRequest{
		DeviceType: req.DeviceType,
	})
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, tree)
}

// CreateGNBGroup handles POST /gnb/pm/indicatormg/addIndicatorGroup.
func (h *IndicatorHandler) CreateGNBGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	req.DeviceType = string(DeviceTypeGNB)

	group, err := h.svc.CreateGroup(c.Request.Context(), &req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, group)
}

// GetGNBGroupInfo handles POST /gnb/pm/indicatormg/getIndicatorGroupInfo.
func (h *IndicatorHandler) GetGNBGroupInfo(c *gin.Context) {
	var req groupInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	group, err := h.svc.GetGroupByID(c.Request.Context(), DeviceTypeGNB, req.ID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, group)
}

// ModifyGNBGroup handles POST /gnb/pm/indicatormg/modifyIndicatorGroup.
func (h *IndicatorHandler) ModifyGNBGroup(c *gin.Context) {
	var req struct {
		ID          string  `json:"id" binding:"required"`
		EnName      *string `json:"en_name"`
		CnName      *string `json:"cn_name"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	updateReq := &UpdateGroupRequest{
		EnName:      req.EnName,
		CnName:      req.CnName,
		Description: req.Description,
	}
	if err := h.svc.UpdateGroup(c.Request.Context(), DeviceTypeGNB, req.ID, updateReq); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

// DeleteGNBGroup handles POST /gnb/pm/indicatormg/delIndicatorGroup.
func (h *IndicatorHandler) DeleteGNBGroup(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.DeleteGroup(c.Request.Context(), DeviceTypeGNB, req.ID); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

// GetGNBIndicatorListPageData handles POST /gnb/pm/indicatormg/getIndicatorListPageData.
func (h *IndicatorHandler) GetGNBIndicatorListPageData(c *gin.Context) {
	var filter IndicatorListFilter
	filter.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindJSON(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter.DeviceType = string(DeviceTypeGNB)

	result, err := h.svc.ListIndicators(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

// GetGNBIndicatorInfo handles POST /gnb/pm/indicatormg/getIndicatorInfo.
func (h *IndicatorHandler) GetGNBIndicatorInfo(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	indicator, err := h.svc.GetIndicatorInfo(c.Request.Context(), DeviceTypeGNB, req.ID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, indicator)
}

// AddOrModifyGNBIndicator handles POST /gnb/pm/indicatormg/addOrModifyIndicator.
func (h *IndicatorHandler) AddOrModifyGNBIndicator(c *gin.Context) {
	var req addOrModifyIndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	req.DeviceType = string(DeviceTypeGNB)

	if req.ID != "" {
		updator := resolveUpdator(c, req.Updator)
		updateReq := &UpdateIndicatorRequest{
			EnName:        &req.EnName,
			CnName:        &req.CnName,
			EnDescription: &req.EnDescription,
			CnDescription: &req.CnDescription,
			GroupID:       &req.GroupID,
			DataType:      &req.DataType,
			UnitID:        &req.UnitID,
			Updator:       &updator,
			Arithmetic:    &req.Arithmetic,
			StatisType:    &req.StatisType,
		}
		if err := h.svc.UpdateIndicator(c.Request.Context(), DeviceTypeGNB, req.ID, updateReq); err != nil {
			commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
			return
		}
		response.OKWithMsg(c, nil, "success")
		return
	}

	updator := resolveUpdator(c, req.Updator)
	createReq := &CreateIndicatorRequest{
		DeviceType:    req.DeviceType,
		EnName:        req.EnName,
		CnName:        req.CnName,
		EnDescription: req.EnDescription,
		CnDescription: req.CnDescription,
		GroupID:       req.GroupID,
		DataType:      req.DataType,
		UnitID:        req.UnitID,
		Updator:       updator,
		IsCounter:     req.IsCounter,
		Arithmetic:    req.Arithmetic,
		StatisType:    req.StatisType,
		OperatorCode:  req.OperatorCode,
	}
	indicator, err := h.svc.CreateIndicator(c.Request.Context(), createReq)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, indicator)
}

// DeleteGNBIndicator handles POST /gnb/pm/indicatormg/delIndicator.
func (h *IndicatorHandler) DeleteGNBIndicator(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.DeleteIndicator(c.Request.Context(), DeviceTypeGNB, req.ID); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

// ExportGNBAllIndicator handles POST /gnb/pm/indicatormg/exportAllIndicator.
func (h *IndicatorHandler) ExportGNBAllIndicator(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	req.DeviceType = string(DeviceTypeGNB)

	data, err := h.svc.ExportIndicators(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=indicators_GNB.csv")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}

// UpdateGNBBaseKpiCustName handles POST /gnb/pm/indicatormg/updateBaseKpiCustName.
func (h *IndicatorHandler) UpdateGNBBaseKpiCustName(c *gin.Context) {
	var req CustNameUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	req.DeviceType = string(DeviceTypeGNB)

	if err := h.svc.UpdateCustName(c.Request.Context(), &req); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

// UpdateGnbIndicatorsName handles POST /gnb/pm/indicatormg/updateGnbIndicatorsName.
func (h *IndicatorHandler) UpdateGnbIndicatorsName(c *gin.Context) {
	h.updateCounterName(c, DeviceTypeGNB)
}

// ── Shared Helpers ────────────────────────────────────────────────────────────

// updateCounterName is a shared handler for updateEnbIndicatorsName, updateGsmIndicatorsName, updateGnbIndicatorsName.
func (h *IndicatorHandler) updateCounterName(c *gin.Context, dt DeviceType) {
	var req counterNameUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.UpdateCounterName(c.Request.Context(), dt, req.ID, req.NewName); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "success")
}

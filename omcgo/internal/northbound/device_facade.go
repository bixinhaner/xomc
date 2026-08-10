package northbound

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/topology"
	"github.com/omcgo/omcgo/internal/ufte"
)

type northboundDeviceService interface {
	ListDevicesWithInfo(context.Context, device.DeviceFilter) (*model.ListResponse[device.DeviceWithInfo], error)
	GetDevice(context.Context, uuid.UUID) (*model.Device, error)
	GetBySerialNumber(context.Context, string) (*model.Device, error)
	GetDeviceWithInfo(context.Context, uuid.UUID) (*device.DeviceWithInfo, error)
	GetDeviceParameters(context.Context, uuid.UUID) ([]model.DeviceParameter, error)
	SetParameters(context.Context, uuid.UUID, []device.ParameterValueItem, string) (string, error)
	SyncDeviceParamsManualDetailed(context.Context, uuid.UUID, string, []string) (*device.ManualParamSyncStart, *model.Device, error)
	RenameDevice(context.Context, uuid.UUID, string, string) (*device.RenameDeviceResult, error)
}

type northboundTaskService interface {
	CreateTask(context.Context, *task.CreateTaskRequest) (*task.Task, error)
	GetTask(context.Context, string) (*task.Task, error)
	GetTaskHistory(context.Context, string, *task.TaskHistoryOptions) (*task.TaskListResponse, error)
}

type northboundRegistrationService interface {
	Register(context.Context, device.CreateRegistrationRequest, string) (*device.DeviceRegistration, error)
	List(context.Context, device.RegistrationFilter) (*model.ListResponse[device.DeviceRegistration], error)
	Delete(context.Context, uuid.UUID) error
}

type northboundDeviceGroupService interface {
	GetTree(context.Context) ([]topology.DeviceGroup, error)
	CreateGroup(context.Context, topology.CreateGroupRequest, string) (*topology.DeviceGroup, error)
	UpdateGroup(context.Context, uuid.UUID, topology.UpdateGroupRequest, string) (*topology.DeviceGroup, error)
	DeleteGroup(context.Context, uuid.UUID) error
	BatchAddDevices(context.Context, uuid.UUID, []uuid.UUID) (int64, error)
}

type northboundTransferTaskService interface {
	CreateTask(context.Context, ufte.CreateTaskRequest, string, []uuid.UUID) (*ufte.Task, error)
	GetTask(context.Context, string, []uuid.UUID) (*ufte.Task, error)
	ListTasks(context.Context, ufte.TaskListFilter, []uuid.UUID) (*model.ListResponse[ufte.Task], error)
}

type northboundSetParametersRequest struct {
	Parameters []device.ParameterValueItem `json:"parameters"`
	PathList   []device.ParameterValueItem `json:"pathList"`
	PathList2  []device.ParameterValueItem `json:"path_list"`
}

type northboundSyncParametersRequest struct {
	ParameterNames []string `json:"parameter_names"`
	ParameterPaths []string `json:"parameter_paths"`
	PathList       []string `json:"pathList"`
	PathList2      []string `json:"path_list"`
}

func (r *Router) requireDeviceFacade(c *gin.Context) bool {
	if r.deviceService == nil {
		response.Fail(c, http.StatusServiceUnavailable, "northbound device facade is not configured")
		return false
	}
	return true
}

func (r *Router) requireTaskFacade(c *gin.Context) bool {
	if r.taskService == nil {
		response.Fail(c, http.StatusServiceUnavailable, "northbound task facade is not configured")
		return false
	}
	return true
}

func (r *Router) requireRegistrationFacade(c *gin.Context) bool {
	if r.regService == nil {
		response.Fail(c, http.StatusServiceUnavailable, "northbound device registration facade is not configured")
		return false
	}
	return true
}

func (r *Router) requireGroupFacade(c *gin.Context) bool {
	if r.groupService == nil {
		response.Fail(c, http.StatusServiceUnavailable, "northbound device group facade is not configured")
		return false
	}
	return true
}

func (r *Router) requireTransferFacade(c *gin.Context) bool {
	if r.transferService == nil {
		response.Fail(c, http.StatusServiceUnavailable, "northbound file-transfer facade is not configured")
		return false
	}
	return true
}

func (r *Router) listDevices(c *gin.Context) {
	if !r.requireDeviceFacade(c) {
		return
	}

	filter := device.DeviceFilter{ListRequest: model.DefaultListRequest()}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if carrier := strings.TrimSpace(c.Query("carrier")); carrier != "" {
		cc := model.CarrierCode(carrier)
		filter.Carrier = &cc
	}
	if tech := strings.TrimSpace(c.Query("technology")); tech != "" {
		techs := device.SplitCSV(tech)
		if len(techs) == 1 {
			t := model.NormalizeTechnology(techs[0])
			filter.Technology = &t
		} else {
			filter.Technologies = make([]model.Technology, 0, len(techs))
			for _, item := range techs {
				filter.Technologies = append(filter.Technologies, model.NormalizeTechnology(item))
			}
		}
	}
	if sn := strings.TrimSpace(c.Query("sn")); sn != "" {
		filter.SN = &sn
	}
	if snList := strings.TrimSpace(c.Query("sn_list")); snList != "" {
		filter.SNList = device.SplitCSV(snList)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		filter.Search = &search
	}
	if productClass := strings.TrimSpace(c.Query("product_class")); productClass != "" {
		filter.ProductClass = &productClass
	}
	if lifecycle := strings.TrimSpace(c.Query("lifecycle_state")); lifecycle != "" {
		for _, item := range device.SplitCSV(lifecycle) {
			filter.LifecycleState = append(filter.LifecycleState, model.DeviceLifecycle(item))
		}
	}
	if online := strings.TrimSpace(c.Query("is_online")); online != "" {
		value := online == "true" || online == "1"
		filter.IsOnline = &value
	}
	if groupID := strings.TrimSpace(c.Query("group_id")); groupID != "" {
		id, err := uuid.Parse(groupID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid group_id")
			return
		}
		filter.GroupID = &id
	}

	if !r.authorizeDeviceListScope(c, filter) {
		return
	}

	result, err := r.deviceService.ListDevicesWithInfo(c.Request.Context(), filter)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OK(c, result)
}

func (r *Router) getDevice(c *gin.Context) {
	id, ok := parseDeviceIDParam(c)
	if !ok || !r.requireDeviceFacade(c) {
		return
	}
	if r.scoper != nil && !r.scoper.AuthorizeDevice(c, id) {
		return
	}
	dev, err := r.deviceService.GetDeviceWithInfo(c.Request.Context(), id)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	if dev == nil {
		response.Fail(c, http.StatusNotFound, "device not found")
		return
	}
	response.OK(c, dev)
}

func (r *Router) getDeviceParameters(c *gin.Context) {
	id, ok := parseDeviceIDParam(c)
	if !ok || !r.requireDeviceFacade(c) {
		return
	}
	if r.scoper != nil && !r.scoper.AuthorizeDevice(c, id) {
		return
	}
	params, err := r.deviceService.GetDeviceParameters(c.Request.Context(), id)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	root := strings.TrimSpace(c.Query("root"))
	if root != "" {
		filtered := params[:0]
		for _, item := range params {
			if strings.HasPrefix(item.ParameterPath, root) {
				filtered = append(filtered, item)
			}
		}
		params = filtered
	}
	response.OK(c, gin.H{"items": params, "total": len(params)})
}

func (r *Router) setDeviceParameters(c *gin.Context) {
	id, ok := parseDeviceIDParam(c)
	if !ok || !r.requireDeviceFacade(c) {
		return
	}
	if r.scoper != nil && !r.scoper.AuthorizeDevice(c, id) {
		return
	}

	var req northboundSetParametersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	params := req.Parameters
	if len(params) == 0 {
		params = req.PathList
	}
	if len(params) == 0 {
		params = req.PathList2
	}
	if msg := validateParameterValueItems(params); msg != "" {
		response.Fail(c, http.StatusBadRequest, msg)
		return
	}
	dev, err := r.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	if dev == nil {
		response.Fail(c, http.StatusNotFound, "device not found")
		return
	}

	taskID, err := r.deviceService.SetParameters(c.Request.Context(), id, params, admin.UserIDStringFromCtx(c))
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, asyncTaskPayload(taskID, dev.SerialNumber, gin.H{
		"message":    "set parameter values command queued",
		"parameters": len(params),
	}))
}

func (r *Router) syncDeviceParameters(c *gin.Context) {
	id, ok := parseDeviceIDParam(c)
	if !ok || !r.requireDeviceFacade(c) {
		return
	}
	if r.scoper != nil && !r.scoper.AuthorizeDevice(c, id) {
		return
	}

	var req northboundSyncParametersRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	paths := firstNonEmptyStringList(req.ParameterNames, req.ParameterPaths, req.PathList, req.PathList2)
	sourceID := "northbound-manual:" + uuid.NewString()
	start, dev, err := r.deviceService.SyncDeviceParamsManualDetailed(c.Request.Context(), id, sourceID, paths)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	if dev == nil {
		response.Fail(c, http.StatusNotFound, "device not found")
		return
	}
	payload := gin.H{
		"message":         "parameter sync queued",
		"source_id":       sourceID,
		"parameter_paths": len(paths),
	}
	if start != nil {
		payload["used"] = start.Used
		payload["task_count"] = start.TaskCount
		payload["request_id"] = start.RequestID
		payload["run_id"] = start.RunID
		payload["status"] = start.Status
		payload["result_code"] = start.ResultCode
	}
	response.OKWithStatus(c, http.StatusAccepted, asyncTaskPayload(sourceID, dev.SerialNumber, payload))
}

func (r *Router) rebootDevice(c *gin.Context) {
	id, ok := parseDeviceIDParam(c)
	if !ok || !r.requireDeviceFacade(c) || !r.requireTaskFacade(c) {
		return
	}
	if r.scoper != nil && !r.scoper.AuthorizeDevice(c, id) {
		return
	}
	dev, err := r.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	if dev == nil {
		response.Fail(c, http.StatusNotFound, "device not found")
		return
	}
	created, err := r.taskService.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "Reboot",
		Priority:   0,
		CommandKey: "northbound-reboot-" + uuid.NewString()[:8],
		Source:     task.TaskSourceAPI,
		CreatorID:  admin.UserIDStringFromCtx(c),
	})
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, asyncTaskPayload(created.ID, dev.SerialNumber, gin.H{
		"message": "reboot command queued",
	}))
}

func (r *Router) getTask(c *gin.Context) {
	if !r.requireTaskFacade(c) {
		return
	}
	taskID := strings.TrimSpace(c.Param("taskId"))
	if taskID == "" {
		response.Fail(c, http.StatusBadRequest, "task_id is required")
		return
	}
	item, err := r.taskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	if item == nil {
		response.Fail(c, http.StatusNotFound, "task not found")
		return
	}
	if r.scoper != nil && item.DeviceSN != "" && !r.scoper.AuthorizeDeviceBySN(c, item.DeviceSN) {
		return
	}
	response.OK(c, taskResultPayload(item))
}

func (r *Router) authorizeDeviceListScope(c *gin.Context, filter device.DeviceFilter) bool {
	if r.scoper == nil {
		return true
	}
	hasSNScope := filter.SN != nil || len(filter.SNList) > 0
	if !r.scoper.RequireDeviceScope(c, hasSNScope) {
		return false
	}
	if filter.SN != nil && !r.scoper.AuthorizeDeviceBySN(c, *filter.SN) {
		return false
	}
	for _, sn := range filter.SNList {
		if !r.scoper.AuthorizeDeviceBySN(c, sn) {
			return false
		}
	}
	return true
}

func parseDeviceIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid device_id")
		return uuid.Nil, false
	}
	return id, true
}

func validateParameterValueItems(params []device.ParameterValueItem) string {
	if len(params) == 0 {
		return "parameters is required"
	}
	if len(params) > 10 {
		return "parameters cannot exceed 10 items"
	}
	for i, item := range params {
		if strings.TrimSpace(item.Path) == "" {
			return fmt.Sprintf("parameters[%d].path is required", i)
		}
	}
	return ""
}

func firstNonEmptyStringList(values ...[]string) []string {
	for _, items := range values {
		out := make([]string, 0, len(items))
		for _, item := range items {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func asyncTaskPayload(taskID string, sn string, extra gin.H) gin.H {
	payload := gin.H{
		"task_id":  taskID,
		"jobId":    taskID,
		"sn":       sn,
		"waitTime": 5,
	}
	for key, value := range extra {
		payload[key] = value
	}
	return payload
}

func taskResultPayload(item *task.Task) gin.H {
	details := ""
	if len(item.Result) > 0 && json.Valid(item.Result) {
		details = string(item.Result)
	}
	return gin.H{
		"jobId":         item.ID,
		"task_id":       item.ID,
		"name":          item.Method,
		"method":        item.Method,
		"status":        string(item.Status),
		"legacy_status": legacyJobStatus(item.Status),
		"sn":            item.DeviceSN,
		"device_sn":     item.DeviceSN,
		"errorMessage":  item.ErrorMessage,
		"error_code":    item.ErrorCode,
		"createTime":    item.CreatedAt,
		"created_at":    item.CreatedAt,
		"completeTime":  item.CompletedAt,
		"completed_at":  item.CompletedAt,
		"details":       details,
		"result":        item.Result,
		"command_key":   item.CommandKey,
		"cwmp_id":       item.CWMPID,
		"retry_count":   item.RetryCount,
		"max_retries":   item.MaxRetries,
		"source":        item.Source,
		"source_id":     item.SourceID,
	}
}

func legacyJobStatus(status task.TaskStatus) string {
	switch status {
	case task.TaskStatusPending:
		return "0"
	case task.TaskStatusSent:
		return "1"
	case task.TaskStatusCompleted:
		return "2"
	case task.TaskStatusFailed, task.TaskStatusExpired, task.TaskStatusCancelled:
		return "3"
	default:
		return ""
	}
}

func failNorthboundFacade(c *gin.Context, err error) {
	status := commonerrors.HTTPStatusFromError(err)
	if status == http.StatusInternalServerError {
		response.Fail(c, status, "northbound API request failed")
		return
	}
	response.Fail(c, status, err.Error())
}

package northbound

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/topology"
	"github.com/omcgo/omcgo/internal/ufte"
)

type northboundCreateDeviceTaskRequest struct {
	Method               string          `json:"method"`
	Params               json.RawMessage `json:"params"`
	Priority             int             `json:"priority"`
	ExpiresIn            int             `json:"expires_in"`
	MaxRetries           *int            `json:"max_retries"`
	RetryIntervalSeconds int             `json:"retry_interval_seconds"`
	CommandKey           string          `json:"command_key"`
	Description          string          `json:"description"`
	SourceID             string          `json:"source_id"`
}

func (r *Router) legacyDeviceQuery(c *gin.Context) {
	if !r.requireDeviceFacade(c) {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	filter := device.DeviceFilter{ListRequest: model.DefaultListRequest()}
	filter.Page = intFromMap(body, "page", filter.Page)
	filter.Page = intFromMap(body, "pageNo", filter.Page)
	filter.PageSize = intFromMap(body, "rows", filter.PageSize)
	filter.PageSize = intFromMap(body, "pageSize", filter.PageSize)
	filter.PageSize = intFromMap(body, "page_size", filter.PageSize)
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	if sn := firstStringFromMap(body, "sn", "serialNumber", "serial_number"); sn != "" {
		filter.SN = &sn
	}
	if snList := firstStringFromMap(body, "snList", "sn_list"); snList != "" {
		filter.SNList = device.SplitCSV(snList)
	}
	if search := firstStringFromMap(body, "search", "keyword", "smallCellCode", "cellCode"); search != "" {
		filter.Search = &search
	}
	if tech := firstStringFromMap(body, "technology", "rat", "mode"); tech != "" {
		t := model.NormalizeTechnology(tech)
		filter.Technology = &t
	}
	if productClass := firstStringFromMap(body, "productClass", "product_class"); productClass != "" {
		filter.ProductClass = &productClass
	}
	if groupID := firstStringFromMap(body, "groupId", "group_id"); groupID != "" {
		id, err := uuid.Parse(groupID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid groupId")
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

func (r *Router) legacyCreateRegistration(c *gin.Context) {
	if !r.requireRegistrationFacade(c) {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	req := device.CreateRegistrationRequest{
		SerialNumber: firstStringFromMap(body, "serial_number", "serialNumber", "sn"),
		GroupID:      firstStringFromMap(body, "group_id", "groupId"),
		Carrier:      firstStringFromMap(body, "carrier", "operatorCode", "operator_code"),
		SiteName:     firstStringFromMap(body, "site_name", "siteName", "deviceName", "device_name"),
		Longitude:    floatPtrFromMap(body, "longitude", "lng"),
		Latitude:     floatPtrFromMap(body, "latitude", "lat"),
		Remark:       firstStringFromMap(body, "remark", "description"),
	}
	if req.SerialNumber == "" {
		response.Fail(c, http.StatusBadRequest, "serial_number is required")
		return
	}
	if req.GroupID == "" {
		response.Fail(c, http.StatusBadRequest, "group_id is required")
		return
	}
	if req.Carrier == "" {
		response.Fail(c, http.StatusBadRequest, "carrier is required")
		return
	}
	created, err := r.regService.Register(c.Request.Context(), req, northboundOperator(c))
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, created)
}

func (r *Router) legacyListRegistrations(c *gin.Context) {
	if !r.requireRegistrationFacade(c) {
		return
	}
	filter := device.RegistrationFilter{ListRequest: model.DefaultListRequest()}
	var body map[string]any
	if c.Request.Method == http.MethodPost {
		parsed, ok := readJSONMap(c)
		if !ok {
			return
		}
		body = parsed
		filter.Page = intFromMap(body, "page", filter.Page)
		filter.Page = intFromMap(body, "pageNo", filter.Page)
		filter.PageSize = intFromMap(body, "rows", filter.PageSize)
		filter.PageSize = intFromMap(body, "pageSize", filter.PageSize)
		filter.PageSize = intFromMap(body, "page_size", filter.PageSize)
		if status := firstStringFromMap(body, "status"); status != "" {
			s := global.RegistrationStatus(status)
			filter.Status = &s
		}
		if sn := firstStringFromMap(body, "serial_number", "serialNumber", "sn"); sn != "" {
			filter.SerialNumber = &sn
		}
		if carrier := firstStringFromMap(body, "carrier", "operatorCode", "operator_code"); carrier != "" {
			cc := model.CarrierCode(carrier)
			filter.Carrier = &cc
		}
	} else {
		filter.Page = positiveIntQuery(c, "page", filter.Page)
		filter.Page = positiveIntQuery(c, "pageNo", filter.Page)
		filter.PageSize = positiveIntQuery(c, "rows", filter.PageSize)
		filter.PageSize = positiveIntQuery(c, "pageSize", filter.PageSize)
		filter.PageSize = positiveIntQuery(c, "page_size", filter.PageSize)
		if status := strings.TrimSpace(c.Query("status")); status != "" {
			s := global.RegistrationStatus(status)
			filter.Status = &s
		}
		if sn := firstNonEmptyString(c.Query("serial_number"), c.Query("serialNumber"), c.Query("sn")); sn != "" {
			filter.SerialNumber = &sn
		}
		if carrier := firstNonEmptyString(c.Query("carrier"), c.Query("operatorCode"), c.Query("operator_code")); carrier != "" {
			cc := model.CarrierCode(carrier)
			filter.Carrier = &cc
		}
	}
	result, err := r.regService.List(c.Request.Context(), filter)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OK(c, result)
}

func (r *Router) legacyDeleteRegistration(c *gin.Context) {
	if !r.requireRegistrationFacade(c) {
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid registration id")
		return
	}
	if err := r.regService.Delete(c.Request.Context(), id); err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (r *Router) legacyDeviceGroupTree(c *gin.Context) {
	if !r.requireGroupFacade(c) {
		return
	}
	tree, err := r.groupService.GetTree(c.Request.Context())
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OK(c, gin.H{"items": tree, "total": len(tree)})
}

func (r *Router) legacyCreateDeviceGroup(c *gin.Context) {
	if !r.requireGroupFacade(c) {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	req := topology.CreateGroupRequest{
		Name:          firstStringFromMap(body, "name", "groupName", "group_name"),
		ParentID:      firstStringFromMap(body, "parent_id", "parentId", "pid"),
		Carrier:       firstStringFromMap(body, "carrier", "operatorCode", "operator_code"),
		Remark:        firstStringFromMap(body, "remark", "description"),
		SortOrder:     intFromMap(body, "sort_order", 0),
		MatchingMode:  firstStringFromMap(body, "matching_mode", "matchingMode"),
		SourceGroupID: firstStringFromMap(body, "source_group_id", "sourceGroupId"),
	}
	if req.Name == "" {
		response.Fail(c, http.StatusBadRequest, "name is required")
		return
	}
	created, err := r.groupService.CreateGroup(c.Request.Context(), req, northboundOperator(c))
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, created)
}

func (r *Router) legacyUpdateDeviceGroup(c *gin.Context) {
	if !r.requireGroupFacade(c) {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	groupID, ok := legacyGroupID(c, body)
	if !ok {
		return
	}
	var req topology.UpdateGroupRequest
	if name := firstStringFromMap(body, "name", "groupName", "group_name"); name != "" {
		req.Name = &name
	}
	if parentID := firstStringFromMap(body, "parent_id", "parentId", "pid"); parentID != "" {
		req.ParentID = &parentID
	}
	if remark := firstStringFromMap(body, "remark", "description"); remark != "" {
		req.Remark = &remark
	}
	if sortOrder, exists := intFromMapIfPresent(body, "sort_order", "sortOrder"); exists {
		req.SortOrder = &sortOrder
	}
	if mode := firstStringFromMap(body, "matching_mode", "matchingMode"); mode != "" {
		req.MatchingMode = &mode
	}
	if sourceID := firstStringFromMap(body, "source_group_id", "sourceGroupId"); sourceID != "" {
		req.SourceGroupID = &sourceID
	}
	updated, err := r.groupService.UpdateGroup(c.Request.Context(), groupID, req, northboundOperator(c))
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OK(c, updated)
}

func (r *Router) legacyDeleteDeviceGroup(c *gin.Context) {
	if !r.requireGroupFacade(c) {
		return
	}
	body := map[string]any{}
	if strings.TrimSpace(c.Param("id")) == "" {
		var ok bool
		body, ok = readJSONMap(c)
		if !ok {
			return
		}
	}
	groupID, ok := legacyGroupID(c, body)
	if !ok {
		return
	}
	if err := r.groupService.DeleteGroup(c.Request.Context(), groupID); err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OK(c, gin.H{"group_id": groupID.String()})
}

func (r *Router) legacyAddDevicesToGroup(c *gin.Context) {
	if !r.requireGroupFacade(c) {
		return
	}
	groupID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group id")
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	deviceIDs, ok := r.deviceIDsFromLegacyBody(c, body)
	if !ok {
		return
	}
	affected, err := r.groupService.BatchAddDevices(c.Request.Context(), groupID, deviceIDs)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OK(c, gin.H{"group_id": groupID.String(), "affected": affected, "device_count": len(deviceIDs)})
}

func (r *Router) legacyDeviceStatus(c *gin.Context) {
	dev, info, ok := r.legacyDeviceBySN(c)
	if !ok {
		return
	}
	connectionStatus := "off"
	if dev.IsOnline {
		connectionStatus = "on"
	}
	payload := gin.H{
		"sn":               dev.SerialNumber,
		"serialNumber":     dev.SerialNumber,
		"smallCellCode":    dev.SiteID,
		"connectionStatus": connectionStatus,
		"is_online":        dev.IsOnline,
		"technology":       dev.Technology,
	}
	if info != nil {
		payload["cellStatus"] = valueOrEmpty(info.CellStatus)
		payload["mmeStatus"] = valueOrEmpty(info.MMEStatus)
		payload["rfStatus"] = valueOrEmpty(info.RFStatus)
		payload["syncStatus"] = valueOrEmpty(info.SyncStatus)
		payload["lockStatus"] = valueOrEmpty(info.LockStatus)
		payload["ueCount"] = info.UECount
	}
	response.OK(c, payload)
}

func (r *Router) legacyDeviceInfo(c *gin.Context) {
	_, info, ok := r.legacyDeviceBySN(c)
	if !ok {
		return
	}
	response.OK(c, info)
}

func (r *Router) legacyGetDeviceParameters(c *gin.Context) {
	dev, ok := r.legacyDeviceBySNOnly(c)
	if !ok {
		return
	}
	params, err := r.deviceService.GetDeviceParameters(c.Request.Context(), dev.ID)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	values := make(map[string]string, len(params))
	for _, item := range params {
		values[item.ParameterPath] = item.ParameterValue
	}
	response.OK(c, gin.H{
		"sn":         dev.SerialNumber,
		"parameters": values,
		"items":      params,
		"total":      len(params),
	})
}

func (r *Router) legacyQueryDeviceParameters(c *gin.Context) {
	dev, ok := r.legacyDeviceBySNOnly(c)
	if !ok {
		return
	}
	paths, ok := readStringListBody(c)
	if !ok {
		return
	}
	sourceID := "northbound-legacy-query:" + uuid.NewString()
	start, _, err := r.deviceService.SyncDeviceParamsManualDetailed(c.Request.Context(), dev.ID, sourceID, paths)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	payload := gin.H{"message": "parameter query queued", "parameter_paths": len(paths)}
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

func (r *Router) legacySetDeviceParameters(c *gin.Context) {
	dev, ok := r.legacyDeviceBySNOnly(c)
	if !ok {
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
	taskID, err := r.deviceService.SetParameters(c.Request.Context(), dev.ID, params, admin.UserIDStringFromCtx(c))
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, asyncTaskPayload(taskID, dev.SerialNumber, gin.H{
		"message":    "set parameter values command queued",
		"parameters": len(params),
	}))
}

func (r *Router) legacySetDeviceName(c *gin.Context) {
	dev, ok := r.legacyDeviceBySNOnly(c)
	if !ok {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	name := firstStringFromMap(body, "cellName", "cell_name", "deviceName", "device_name", "name")
	if name == "" {
		response.Fail(c, http.StatusBadRequest, "cellName is required")
		return
	}
	result, err := r.deviceService.RenameDevice(c.Request.Context(), dev.ID, name, admin.UserIDStringFromCtx(c))
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	payload := gin.H{"message": "device name updated", "sn": dev.SerialNumber, "cellName": name}
	if result != nil && result.TaskID != "" {
		payload["task_id"] = result.TaskID
		payload["jobId"] = result.TaskID
		payload["waitTime"] = 5
	}
	response.OK(c, payload)
}

func (r *Router) legacyCreateDeviceTask(c *gin.Context) {
	dev, ok := r.legacyDeviceBySNOnly(c)
	if !ok || !r.requireTaskFacade(c) {
		return
	}
	var req northboundCreateDeviceTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	req.Method = strings.TrimSpace(req.Method)
	if req.Method == "" {
		response.Fail(c, http.StatusBadRequest, "method is required")
		return
	}
	r.queueLegacyDeviceTask(c, dev, req, "device command queued", "northbound-task")
}

func (r *Router) legacyRebootDevice(c *gin.Context) {
	dev, ok := r.legacyDeviceBySNOnly(c)
	if !ok || !r.requireTaskFacade(c) {
		return
	}
	r.queueLegacyDeviceTask(c, dev, northboundCreateDeviceTaskRequest{Method: "Reboot"}, "reboot command queued", "northbound-reboot")
}

func (r *Router) legacyResetDevice(c *gin.Context) {
	dev, ok := r.legacyDeviceBySNOnly(c)
	if !ok || !r.requireTaskFacade(c) {
		return
	}
	r.queueLegacyDeviceTask(c, dev, northboundCreateDeviceTaskRequest{Method: "FactoryReset"}, "factory reset command queued", "northbound-reset")
}

func (r *Router) legacyCollectDeviceLog(c *gin.Context) {
	dev, ok := r.legacyDeviceBySNOnly(c)
	if !ok || !r.requireTransferFacade(c) {
		return
	}
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	typeCode := normalizeLogCollectType(
		firstNonEmptyString(
			firstStringFromMap(body, "typeCode", "type_code", "logType", "log_type"),
			c.Query("typeCode"),
			c.Query("type_code"),
			c.Query("logType"),
			c.Query("log_type"),
		),
	)
	taskName := firstStringFromMap(body, "taskName", "task_name", "name")
	if taskName == "" {
		taskName = "Northbound " + typeCode + " " + dev.SerialNumber
	}
	executionMode := firstStringFromMap(body, "executionMode", "execution_mode")
	if executionMode == "" {
		executionMode = "immediate"
	}
	req := ufte.CreateTaskRequest{
		TaskName:      taskName,
		TypeCode:      typeCode,
		ProductType:   dev.ProductClass,
		DeviceIDs:     []uuid.UUID{dev.ID},
		DeviceCount:   1,
		ExecutionMode: executionMode,
		ScheduledAt:   firstStringFromMap(body, "scheduledAt", "scheduled_at"),
		Note:          firstStringFromMap(body, "note", "remark", "description"),
	}
	created, err := r.transferService.CreateTask(c.Request.Context(), req, northboundOperator(c), nil)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, asyncTaskPayload(created.ID, dev.SerialNumber, gin.H{
		"message":        "device log collect queued",
		"typeCode":       created.TypeCode,
		"taskName":       created.TaskName,
		"status":         created.Status,
		"execution_mode": created.ExecutionMode,
	}))
}

func (r *Router) legacyGetTask(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("taskId"))
	if taskID == "" {
		response.Fail(c, http.StatusBadRequest, "task_id is required")
		return
	}
	if r.taskService != nil {
		item, err := r.taskService.GetTask(c.Request.Context(), taskID)
		if err == nil && item != nil {
			if r.scoper != nil && item.DeviceSN != "" && !r.scoper.AuthorizeDeviceBySN(c, item.DeviceSN) {
				return
			}
			response.OK(c, taskResultPayload(item))
			return
		}
		if err != nil && !errors.Is(err, task.ErrTaskNotFound) && !errors.Is(err, commonerrors.ErrNotFound) {
			failNorthboundFacade(c, err)
			return
		}
	}
	if r.transferService != nil {
		item, err := r.transferService.GetTask(c.Request.Context(), taskID, nil)
		if err == nil && item != nil {
			response.OK(c, transferTaskResultPayload(item))
			return
		}
		if err != nil && !errors.Is(err, commonerrors.ErrNotFound) {
			failNorthboundFacade(c, err)
			return
		}
	}
	response.Fail(c, http.StatusNotFound, "task not found")
}

func (r *Router) legacyListTasks(c *gin.Context) {
	body, ok := readJSONMap(c)
	if !ok {
		return
	}
	sn := firstStringFromMap(body, "sn", "device_sn", "serialNumber", "serial_number")
	if sn == "" {
		if r.transferService == nil {
			response.Fail(c, http.StatusBadRequest, "sn or device_sn is required")
			return
		}
		filter := ufte.TaskListFilter{
			Category: firstStringFromMap(body, "category"),
			TypeCode: firstStringFromMap(body, "typeCode", "type_code"),
			Status:   firstStringFromMap(body, "status"),
			Keyword:  firstStringFromMap(body, "keyword", "search"),
			Page:     intFromMap(body, "page", 1),
			PageSize: intFromMap(body, "rows", 20),
		}
		filter.Page = intFromMap(body, "pageNo", filter.Page)
		filter.PageSize = intFromMap(body, "pageSize", filter.PageSize)
		filter.PageSize = intFromMap(body, "page_size", filter.PageSize)
		if filter.Category == "" {
			filter.Category = "station_log"
		}
		result, err := r.transferService.ListTasks(c.Request.Context(), filter, nil)
		if err != nil {
			failNorthboundFacade(c, err)
			return
		}
		response.OK(c, result)
		return
	}
	if !r.requireTaskFacade(c) {
		return
	}
	if r.scoper != nil && !r.scoper.AuthorizeDeviceBySN(c, sn) {
		return
	}
	page := intFromMap(body, "page", 1)
	page = intFromMap(body, "pageNo", page)
	rows := intFromMap(body, "rows", 20)
	rows = intFromMap(body, "pageSize", rows)
	status := firstStringFromMap(body, "status")
	opts := &task.TaskHistoryOptions{Page: page, PageSize: rows}
	if status != "" {
		opts.Status = task.TaskStatus(status)
	}
	result, err := r.taskService.GetTaskHistory(c.Request.Context(), sn, opts)
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OK(c, result)
}

func (r *Router) queueLegacyDeviceTask(c *gin.Context, dev *model.Device, req northboundCreateDeviceTaskRequest, message, commandPrefix string) {
	commandKey := strings.TrimSpace(req.CommandKey)
	if commandKey == "" {
		commandKey = commandPrefix + "-" + uuid.NewString()[:8]
	}
	created, err := r.taskService.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN:               dev.SerialNumber,
		Method:                 req.Method,
		Params:                 req.Params,
		Priority:               req.Priority,
		ExpiresIn:              req.ExpiresIn,
		MaxRetries:             req.MaxRetries,
		RetryIntervalSeconds:   req.RetryIntervalSeconds,
		CommandKey:             commandKey,
		Source:                 task.TaskSourceAPI,
		CreatorID:              admin.UserIDStringFromCtx(c),
		Description:            req.Description,
		SourceID:               req.SourceID,
		HasPathTranslationMiss: false,
	})
	if err != nil {
		failNorthboundFacade(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, asyncTaskPayload(created.ID, dev.SerialNumber, gin.H{
		"message": message,
		"method":  created.Method,
	}))
}

func (r *Router) legacyDeviceBySN(c *gin.Context) (*model.Device, *device.DeviceWithInfo, bool) {
	dev, ok := r.legacyDeviceBySNOnly(c)
	if !ok {
		return nil, nil, false
	}
	info, err := r.deviceService.GetDeviceWithInfo(c.Request.Context(), dev.ID)
	if err != nil {
		failNorthboundFacade(c, err)
		return nil, nil, false
	}
	if info == nil {
		response.Fail(c, http.StatusNotFound, "device not found")
		return nil, nil, false
	}
	return dev, info, true
}

func (r *Router) legacyDeviceBySNOnly(c *gin.Context) (*model.Device, bool) {
	if !r.requireDeviceFacade(c) {
		return nil, false
	}
	sn := strings.TrimSpace(c.Param("sn"))
	if sn == "" {
		response.Fail(c, http.StatusBadRequest, "sn is required")
		return nil, false
	}
	if r.scoper != nil && !r.scoper.AuthorizeDeviceBySN(c, sn) {
		return nil, false
	}
	dev, err := r.deviceService.GetBySerialNumber(c.Request.Context(), sn)
	if err != nil {
		failNorthboundFacade(c, err)
		return nil, false
	}
	if dev == nil {
		response.Fail(c, http.StatusNotFound, "sn not exist")
		return nil, false
	}
	return dev, true
}

func readJSONMap(c *gin.Context) (map[string]any, bool) {
	if c.Request == nil || c.Request.Body == nil {
		return map[string]any{}, true
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	_ = c.Request.Body.Close()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid JSON body")
		return nil, false
	}
	return body, true
}

func readStringListBody(c *gin.Context) ([]string, bool) {
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	_ = c.Request.Body.Close()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, true
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return firstNonEmptyStringList(list), true
	}
	var body northboundSyncParametersRequest
	if err := json.Unmarshal(raw, &body); err != nil && !errors.Is(err, io.EOF) {
		response.Fail(c, http.StatusBadRequest, "invalid JSON body")
		return nil, false
	}
	return firstNonEmptyStringList(body.ParameterNames, body.ParameterPaths, body.PathList, body.PathList2), true
}

func firstStringFromMap(body map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := body[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		case float64:
			return strconv.FormatInt(int64(v), 10)
		case json.Number:
			return v.String()
		}
	}
	return ""
}

func intFromMap(body map[string]any, key string, fallback int) int {
	value, ok := body[key]
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case float64:
		if v > 0 {
			return int(v)
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil && parsed > 0 {
			return parsed
		}
	case json.Number:
		parsed, err := strconv.Atoi(v.String())
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func intFromMapIfPresent(body map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		value, ok := body[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case float64:
			return int(v), true
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(v))
			if err == nil {
				return parsed, true
			}
		case json.Number:
			parsed, err := strconv.Atoi(v.String())
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func positiveIntQuery(c *gin.Context, key string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(c.Query(key)))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func floatPtrFromMap(body map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		value, ok := body[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case float64:
			return &v
		case string:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err == nil {
				return &parsed
			}
		case json.Number:
			parsed, err := strconv.ParseFloat(v.String(), 64)
			if err == nil {
				return &parsed
			}
		}
	}
	return nil
}

func legacyGroupID(c *gin.Context, body map[string]any) (uuid.UUID, bool) {
	raw := strings.TrimSpace(c.Param("id"))
	if raw == "" {
		raw = firstStringFromMap(body, "group_id", "groupId", "id")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group id")
		return uuid.Nil, false
	}
	return id, true
}

func (r *Router) deviceIDsFromLegacyBody(c *gin.Context, body map[string]any) ([]uuid.UUID, bool) {
	values := stringListFromMap(body, "device_ids", "deviceIds", "ids")
	out := make([]uuid.UUID, 0, len(values))
	for _, raw := range values {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device id: "+raw)
			return nil, false
		}
		out = append(out, id)
	}
	sns := stringListFromMap(body, "sns", "snList", "sn_list", "serialNumbers", "serial_numbers")
	if len(sns) > 0 {
		if !r.requireDeviceFacade(c) {
			return nil, false
		}
		for _, sn := range sns {
			dev, err := r.deviceService.GetBySerialNumber(c.Request.Context(), sn)
			if err != nil {
				failNorthboundFacade(c, err)
				return nil, false
			}
			if dev == nil {
				response.Fail(c, http.StatusNotFound, "sn not exist: "+sn)
				return nil, false
			}
			out = append(out, dev.ID)
		}
	}
	if len(out) == 0 {
		response.Fail(c, http.StatusBadRequest, "device_ids or sns is required")
		return nil, false
	}
	return out, true
}

func stringListFromMap(body map[string]any, keys ...string) []string {
	for _, key := range keys {
		value, ok := body[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case []string:
			return trimStringList(v)
		case []any:
			out := make([]string, 0, len(v))
			for _, item := range v {
				switch typed := item.(type) {
				case string:
					out = append(out, typed)
				case float64:
					out = append(out, strconv.FormatInt(int64(typed), 10))
				case json.Number:
					out = append(out, typed.String())
				}
			}
			return trimStringList(out)
		case string:
			return trimStringList(device.SplitCSV(v))
		}
	}
	return nil
}

func trimStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalizeLogCollectType(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "FAULT", "FAULT_LOG", "FAULT_LOG_COLLECT", "8":
		return "FAULT_LOG_COLLECT"
	default:
		return "RUNTIME_LOG_COLLECT"
	}
}

func northboundOperator(c *gin.Context) string {
	if v, ok := c.Get(admin.CtxKeyUsername); ok {
		if username, ok := v.(string); ok && strings.TrimSpace(username) != "" {
			return strings.TrimSpace(username)
		}
	}
	if id := admin.UserIDStringFromCtx(c); id != "" {
		return id
	}
	return "northbound"
}

func transferTaskResultPayload(item *ufte.Task) gin.H {
	return gin.H{
		"jobId":          item.ID,
		"task_id":        item.ID,
		"name":           item.TaskName,
		"method":         item.TypeCode,
		"typeCode":       item.TypeCode,
		"category":       item.Category,
		"status":         item.Status,
		"legacy_status":  legacyTransferTaskStatus(item.Status),
		"result":         item.Result,
		"progress":       item.Progress,
		"totalCount":     item.TotalCount,
		"successCount":   item.SuccessCount,
		"failCount":      item.FailCount,
		"currentStep":    item.CurrentStep,
		"execution_mode": item.ExecutionMode,
		"createTime":     item.CreatedAt,
		"completeTime":   item.EndedAt,
	}
}

func legacyTransferTaskStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "0"
	case "in_progress", "suspended":
		return "1"
	case "ended":
		return "2"
	case "failed", "terminated":
		return "3"
	default:
		return ""
	}
}

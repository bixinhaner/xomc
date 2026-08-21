package northbound

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
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
	"github.com/omcgo/omcgo/internal/paramsync"
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

var (
	legacyRunTimeTokenRegex       = regexp.MustCompile(`(?i)(\d+)\s*(days?|d|天|hours?|hrs?|hr|h|小时|minutes?|mins?|min|m|分钟|seconds?|secs?|sec|s|秒)`)
	legacyRunTimeColonRegex       = regexp.MustCompile(`^(\d+):(\d{1,2})(?::(\d{1,2}))?$`)
	legacyLicenseCapacityPathExpr = regexp.MustCompile(`(?i)^(.*X_COM_LICENSE\.Capacity\.\d+)\.(State|RemainingPeriod|RemainDays|RemainingDays)$`)
)

type legacyLicenseCapacityParams struct {
	state     string
	remain    int
	hasRemain bool
}

// legacyParamSyncParameterResult 是 job/result 载荷 parameters 映射的内部组装单元，
// 仅 Path（对外键，优先请求路径形态）与 Value 会出现在响应里。
type legacyParamSyncParameterResult struct {
	Path  string
	Value string
}

type legacyParamSyncCoverageMapping struct {
	standardPath string
	privatePath  string
}

type legacyParamSyncParameterMatch struct {
	requestedPath string
	standardPath  string
	privatePath   string
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
	if result != nil {
		for i := range result.Items {
			r.refreshLegacyDeviceRadioFrequencyFromParameters(c, &result.Items[i])
			legacyNormalizeDeviceInfoForNorthbound(&result.Items[i])
		}
	}
	response.OK(c, legacyListPayload(result))
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
	if result != nil && result.Total > 0 && result.TotalPages > 0 && len(result.Items) == 0 && result.Page > result.TotalPages {
		filter.Page = result.TotalPages
		result, err = r.regService.List(c.Request.Context(), filter)
		if err != nil {
			failNorthboundFacade(c, err)
			return
		}
	}
	response.OK(c, legacyListPayload(result))
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
	r.refreshLegacyDeviceInfoFromParameters(c, info)
	legacyNormalizeDeviceInfoForNorthbound(info)
	response.OK(c, info)
}

func (r *Router) refreshLegacyDeviceInfoFromParameters(c *gin.Context, info *device.DeviceWithInfo) {
	if info == nil || r.deviceService == nil {
		return
	}
	params, err := r.deviceService.GetDeviceParameters(c.Request.Context(), info.ID)
	if err != nil {
		return
	}
	values := legacyDeviceParameterValues(params)
	if seconds, ok := legacyResolveRunTimeSeconds(values); ok {
		info.RunTime = int64Ptr(seconds)
	}
	if legacyHasLicenseCapacityParameters(values) {
		status := legacyCalcLicenseStatus(values)
		info.LicenseStatus = &status
	}
	legacyApplyRadioFrequencyProjection(info, device.ProjectRadioFrequencyValues(values, info.Technology, info.ProductClass))
}

func (r *Router) refreshLegacyDeviceRadioFrequencyFromParameters(c *gin.Context, info *device.DeviceWithInfo) {
	if info == nil || r.deviceService == nil {
		return
	}
	params, err := r.deviceService.GetDeviceParameters(c.Request.Context(), info.ID)
	if err != nil {
		return
	}
	legacyApplyRadioFrequencyProjection(
		info,
		device.ProjectRadioFrequencyValues(legacyDeviceParameterValues(params), info.Technology, info.ProductClass),
	)
}

func legacyApplyRadioFrequencyProjection(info *device.DeviceWithInfo, projected device.RadioFrequencyProjection) {
	if info == nil {
		return
	}
	if projected.DLObserved && strings.TrimSpace(projected.DLValue) != "" {
		info.FreqPoint = stringValuePtr(projected.DLValue)
	}
	if projected.ULObserved && strings.TrimSpace(projected.ULValue) != "" {
		info.ULEarfcn = stringValuePtr(projected.ULValue)
	}
}

func legacyNormalizeDeviceInfoForNorthbound(info *device.DeviceWithInfo) {
	if info == nil {
		return
	}
	if total, ok := legacyAccumulatedOnlineDuration(info); ok {
		// Old northbound inventory calls this field online_duration. It means
		// accumulated online time, not only the current/last online interval.
		info.OnlineDuration = int64Ptr(total)
		info.CumulativeOnlineDuration = int64Ptr(total)
	}
	info.FreqPointNoDL = info.FreqPoint
	info.FreqPointNoUL = info.ULEarfcn
}

func legacyListPayload[T any](result *model.ListResponse[T]) gin.H {
	items := []T{}
	if result == nil {
		return gin.H{
			"items":       items,
			"rows":        items,
			"total":       int64(0),
			"totalRows":   int64(0),
			"page":        1,
			"pageNo":      1,
			"page_size":   20,
			"pageSize":    20,
			"total_pages": 0,
			"totalPages":  0,
		}
	}
	if result.Items != nil {
		items = result.Items
	}
	payload := gin.H{
		"items":       items,
		"rows":        items,
		"total":       result.Total,
		"totalRows":   result.Total,
		"page":        result.Page,
		"pageNo":      result.Page,
		"page_size":   result.PageSize,
		"pageSize":    result.PageSize,
		"total_pages": result.TotalPages,
		"totalPages":  result.TotalPages,
	}
	if result.Stats != nil {
		payload["stats"] = result.Stats
	}
	return payload
}

func legacyAccumulatedOnlineDuration(info *device.DeviceWithInfo) (int64, bool) {
	if info == nil {
		return 0, false
	}
	var total int64
	hasValue := false
	if info.CumulativeOnlineDuration != nil {
		total += *info.CumulativeOnlineDuration
		hasValue = true
	}
	if info.IsOnline && info.OnlineDuration != nil {
		total += *info.OnlineDuration
		hasValue = true
	}
	if !hasValue && info.OnlineDuration != nil {
		return *info.OnlineDuration, true
	}
	return total, hasValue
}

func legacyDeviceParameterValues(params []model.DeviceParameter) map[string]string {
	values := make(map[string]string, len(params))
	for _, param := range params {
		values[param.ParameterPath] = param.ParameterValue
	}
	return values
}

func legacyHasLicenseCapacityParameters(values map[string]string) bool {
	return len(legacyLicenseCapacities(values)) > 0
}

func legacyResolveRunTimeSeconds(values map[string]string) (int64, bool) {
	if val := strings.TrimSpace(values[device.ParamUpTime]); val != "" {
		if seconds, ok := legacyParseRunTimeValue(val); ok {
			return seconds, true
		}
	}
	if val := strings.TrimSpace(values[device.ParamStationRunTime]); val != "" {
		if seconds, ok := legacyParseRunTimeValue(val); ok {
			return seconds, true
		}
	}
	return 0, false
}

func legacyParseRunTimeValue(raw string) (int64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return seconds, true
	}
	if matches := legacyRunTimeColonRegex.FindStringSubmatch(trimmed); matches != nil {
		first, _ := strconv.ParseInt(matches[1], 10, 64)
		second, _ := strconv.ParseInt(matches[2], 10, 64)
		if matches[3] == "" {
			return first*3600 + second*60, true
		}
		third, _ := strconv.ParseInt(matches[3], 10, 64)
		return first*3600 + second*60 + third, true
	}

	var totalSeconds int64
	matches := legacyRunTimeTokenRegex.FindAllStringSubmatch(trimmed, -1)
	if len(matches) == 0 {
		return 0, false
	}
	for _, match := range matches {
		n, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(match[2]) {
		case "d", "day", "days", "天":
			totalSeconds += n * 86400
		case "h", "hr", "hrs", "hour", "hours", "小时":
			totalSeconds += n * 3600
		case "m", "min", "mins", "minute", "minutes", "分钟":
			totalSeconds += n * 60
		case "s", "sec", "secs", "second", "seconds", "秒":
			totalSeconds += n
		}
	}
	return totalSeconds, true
}

func legacyCalcLicenseStatus(values map[string]string) string {
	hasActive := false
	minRemain := int(^uint(0) >> 1)
	for _, capacity := range legacyLicenseCapacities(values) {
		if !legacyLicenseCapacityActive(capacity.state, capacity.remain, capacity.hasRemain) {
			continue
		}
		hasActive = true
		if capacity.hasRemain && capacity.remain < minRemain {
			minRemain = capacity.remain
		}
	}
	switch {
	case !hasActive:
		return "expired"
	case minRemain <= 30:
		return "expiring"
	default:
		return "active"
	}
}

func legacyLicenseCapacities(values map[string]string) map[string]legacyLicenseCapacityParams {
	capacities := map[string]legacyLicenseCapacityParams{}
	for path, value := range values {
		matches := legacyLicenseCapacityPathExpr.FindStringSubmatch(path)
		if matches == nil {
			continue
		}
		key := strings.ToLower(matches[1])
		field := strings.ToLower(matches[2])
		capacity := capacities[key]
		switch field {
		case "state":
			capacity.state = value
		case "remainingperiod", "remaindays", "remainingdays":
			if remain, ok := legacyLicenseRemainingPeriod(value); ok {
				capacity.remain = remain
				capacity.hasRemain = true
			}
		}
		capacities[key] = capacity
	}
	return capacities
}

func legacyLicenseRemainingPeriod(raw string) (int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}
	remain, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, false
	}
	return remain, true
}

func legacyLicenseCapacityActive(state string, remain int, hasRemain bool) bool {
	normalized := strings.ToLower(strings.TrimSpace(state))
	switch normalized {
	case "1", "active", "true", "enabled", "valid":
		return true
	case "0", "inactive", "false", "disabled", "expired", "invalid":
		return false
	}
	return hasRemain && remain > 0
}

func int64Ptr(value int64) *int64 {
	return &value
}

func stringValuePtr(value string) *string {
	return &value
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
	if parsedID, err := uuid.Parse(taskID); err == nil {
		if r.paramSyncService != nil {
			if payload, ok, err := r.lookupParamSyncJob(c, parsedID); err != nil {
				failNorthboundFacade(c, err)
				return
			} else if ok {
				if payload != nil {
					response.OK(c, payload)
				}
				return
			}
		}
	} else if r.paramSyncService != nil && isNorthboundParamSyncSourceID(taskID) {
		req, findErr := r.findParamSyncRequestBySourceID(c, taskID)
		if findErr == nil && req != nil {
			if r.scoper != nil && req.DeviceSN != "" && !r.scoper.AuthorizeDeviceBySN(c, req.DeviceSN) {
				return
			}
			payload := paramSyncRequestResultPayload(req)
			var run *paramsync.SyncRun
			if req.RunID != nil {
				if loadedRun, runErr := r.paramSyncService.GetRun(c.Request.Context(), *req.RunID); runErr == nil && loadedRun != nil {
					run = loadedRun
					payload = paramSyncRunResultPayload(req, run)
				}
			}
			if err := r.attachLegacyParamSyncParameterResults(c, payload, req, run); err != nil {
				failNorthboundFacade(c, err)
				return
			}
			response.OK(c, payload)
			return
		}
		if findErr != nil && !isNotFoundLike(findErr) {
			failNorthboundFacade(c, findErr)
			return
		}
		response.Fail(c, http.StatusNotFound, "task not found")
		return
	} else if _, err := uuid.Parse(taskID); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid task_id")
		return
	}
	invalidTaskID := false
	if r.taskService != nil {
		item, err := r.taskService.GetTask(c.Request.Context(), taskID)
		if err == nil && item != nil {
			if r.scoper != nil && item.DeviceSN != "" && !r.scoper.AuthorizeDeviceBySN(c, item.DeviceSN) {
				return
			}
			response.OK(c, taskResultPayload(item))
			return
		}
		if isInvalidTaskIdentifierError(err) {
			invalidTaskID = true
		} else if err != nil && !errors.Is(err, task.ErrTaskNotFound) && !errors.Is(err, commonerrors.ErrNotFound) {
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
		if isInvalidTaskIdentifierError(err) {
			invalidTaskID = true
		} else if err != nil && !errors.Is(err, commonerrors.ErrNotFound) {
			failNorthboundFacade(c, err)
			return
		}
	}
	if invalidTaskID {
		response.Fail(c, http.StatusBadRequest, "invalid task_id")
		return
	}
	response.Fail(c, http.StatusNotFound, "task not found")
}

func (r *Router) findParamSyncRequestBySourceID(c *gin.Context, taskID string) (*paramsync.SyncRequest, error) {
	for _, callerType := range paramSyncSourceIDCallerTypes(taskID) {
		req, err := r.paramSyncService.FindRequestByIdempotency(c.Request.Context(), callerType, taskID)
		if err == nil || !isNotFoundLike(err) {
			return req, err
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *Router) lookupParamSyncJob(c *gin.Context, id uuid.UUID) (gin.H, bool, error) {
	req, err := r.paramSyncService.GetRequest(c.Request.Context(), id)
	if err == nil && req != nil {
		if r.scoper != nil && req.DeviceSN != "" && !r.scoper.AuthorizeDeviceBySN(c, req.DeviceSN) {
			return nil, true, nil
		}
		payload := paramSyncRequestResultPayload(req)
		var run *paramsync.SyncRun
		if req.RunID != nil {
			if loadedRun, runErr := r.paramSyncService.GetRun(c.Request.Context(), *req.RunID); runErr == nil && loadedRun != nil {
				run = loadedRun
				payload = paramSyncRunResultPayload(req, run)
			}
		}
		if err := r.attachLegacyParamSyncParameterResults(c, payload, req, run); err != nil {
			return nil, true, err
		}
		return payload, true, nil
	}
	if err != nil && !isNotFoundLike(err) {
		return nil, false, err
	}

	run, err := r.paramSyncService.GetRun(c.Request.Context(), id)
	if err == nil && run != nil {
		req, reqErr := r.paramSyncService.GetRequest(c.Request.Context(), run.RequestID)
		if reqErr != nil && !isNotFoundLike(reqErr) {
			return nil, false, reqErr
		}
		if r.scoper != nil && run.DeviceSN != "" && !r.scoper.AuthorizeDeviceBySN(c, run.DeviceSN) {
			return nil, true, nil
		}
		payload := paramSyncRunResultPayload(req, run)
		if err := r.attachLegacyParamSyncParameterResults(c, payload, req, run); err != nil {
			return nil, true, err
		}
		return payload, true, nil
	}
	if err != nil && !isNotFoundLike(err) {
		return nil, false, err
	}
	return nil, false, nil
}

func (r *Router) attachLegacyParamSyncParameterResults(c *gin.Context, payload gin.H, req *paramsync.SyncRequest, run *paramsync.SyncRun) error {
	if payload == nil {
		return nil
	}
	items, err := r.legacyParamSyncParameterResultItems(c, req, run)
	if err != nil {
		return err
	}
	parameters := make(map[string]string, len(items))
	for _, item := range items {
		parameters[item.Path] = item.Value
	}
	payload["parameters"] = parameters
	payload["total"] = len(parameters)
	return nil
}

func (r *Router) legacyParamSyncParameterResultItems(c *gin.Context, req *paramsync.SyncRequest, run *paramsync.SyncRun) ([]legacyParamSyncParameterResult, error) {
	// 仅路径级任务返回值；full-sync 无 requested_paths，直接短路，
	// 避免无谓拉取全量暂存值/设备参数。
	if len(legacyParamSyncRequestedPaths(req)) == 0 {
		return nil, nil
	}
	if run != nil && run.ID != uuid.Nil && r.paramSyncService != nil {
		values, err := r.paramSyncService.ListRunValues(c.Request.Context(), run.ID)
		if err != nil {
			return nil, err
		}
		if items := legacyParamSyncRunValueResults(req, run, values); len(items) > 0 {
			return items, nil
		}
	}
	if !legacyParamSyncCanFallbackToDeviceValues(req, run) || r.deviceService == nil {
		return nil, nil
	}
	deviceID := uuid.Nil
	if req != nil {
		deviceID = req.DeviceID
	}
	if deviceID == uuid.Nil && run != nil {
		deviceID = run.DeviceID
	}
	if deviceID == uuid.Nil {
		return nil, nil
	}
	params, err := r.deviceService.GetDeviceParameters(c.Request.Context(), deviceID)
	if err != nil {
		return nil, err
	}
	return legacyParamSyncDeviceParameterResults(req, run, params), nil
}

func legacyParamSyncCanFallbackToDeviceValues(req *paramsync.SyncRequest, run *paramsync.SyncRun) bool {
	if run != nil {
		return run.Status == paramsync.RunStatusSucceeded
	}
	return req != nil && req.Status == paramsync.RequestStatusSucceeded
}

func legacyParamSyncRunValueResults(req *paramsync.SyncRequest, run *paramsync.SyncRun, values []paramsync.RunValue) []legacyParamSyncParameterResult {
	requestedPaths := legacyParamSyncRequestedPaths(req)
	if len(requestedPaths) == 0 {
		return nil
	}
	mappings := legacyParamSyncCoverageMappings(run)
	items := make([]legacyParamSyncParameterResult, 0, len(requestedPaths))
	for _, value := range values {
		match, ok := legacyMatchParamSyncValue(value.ParameterPath, value.PrivatePath, requestedPaths, mappings)
		if !ok {
			continue
		}
		standardPath := firstNonEmptyString(match.standardPath, value.ParameterPath)
		privatePath := firstNonEmptyString(match.privatePath, value.PrivatePath)
		path := firstNonEmptyString(match.requestedPath, privatePath, standardPath)
		items = append(items, legacyParamSyncParameterResult{
			Path:  path,
			Value: value.Value,
		})
	}
	return items
}

func legacyParamSyncDeviceParameterResults(req *paramsync.SyncRequest, run *paramsync.SyncRun, params []model.DeviceParameter) []legacyParamSyncParameterResult {
	requestedPaths := legacyParamSyncRequestedPaths(req)
	if len(requestedPaths) == 0 {
		return nil
	}
	mappings := legacyParamSyncCoverageMappings(run)
	items := make([]legacyParamSyncParameterResult, 0, len(requestedPaths))
	for _, param := range params {
		match, ok := legacyMatchParamSyncParameter(param.ParameterPath, requestedPaths, mappings)
		if !ok {
			continue
		}
		standardPath := firstNonEmptyString(match.standardPath, param.ParameterPath)
		path := firstNonEmptyString(match.requestedPath, match.privatePath, standardPath)
		items = append(items, legacyParamSyncParameterResult{
			Path:  path,
			Value: param.ParameterValue,
		})
	}
	return items
}

func legacyMatchParamSyncValue(standardPath, privatePath string, requestedPaths []string, mappings []legacyParamSyncCoverageMapping) (legacyParamSyncParameterMatch, bool) {
	if match, ok := legacyMatchParamSyncParameter(standardPath, requestedPaths, mappings); ok {
		if match.privatePath == "" {
			match.privatePath = privatePath
		}
		return match, true
	}
	for _, requestedPath := range requestedPaths {
		if legacyParamSyncPathCovers(requestedPath, privatePath) {
			return legacyParamSyncParameterMatch{
				requestedPath: requestedPath,
				standardPath:  standardPath,
				privatePath:   privatePath,
			}, true
		}
	}
	return legacyParamSyncParameterMatch{}, false
}

func legacyParamSyncRequestedPaths(req *paramsync.SyncRequest) []string {
	if req == nil {
		return nil
	}
	out := make([]string, 0, len(req.RequestedPaths))
	seen := make(map[string]struct{}, len(req.RequestedPaths))
	for _, path := range req.RequestedPaths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		key := strings.TrimSuffix(path, ".")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, path)
	}
	return out
}

func legacyParamSyncCoverageMappings(run *paramsync.SyncRun) []legacyParamSyncCoverageMapping {
	if run == nil {
		return nil
	}
	mappings := make([]legacyParamSyncCoverageMapping, 0)
	for _, scope := range run.Coverage {
		for _, mapping := range scope.Mappings {
			if strings.TrimSpace(mapping.StandardPath) == "" {
				continue
			}
			mappings = append(mappings, legacyParamSyncCoverageMapping{
				standardPath: strings.TrimSpace(mapping.StandardPath),
				privatePath:  strings.TrimSpace(mapping.PrivatePath),
			})
		}
	}
	return mappings
}

func legacyMatchParamSyncParameter(path string, requestedPaths []string, mappings []legacyParamSyncCoverageMapping) (legacyParamSyncParameterMatch, bool) {
	for _, requestedPath := range requestedPaths {
		if legacyParamSyncPathCovers(requestedPath, path) {
			return legacyParamSyncParameterMatch{
				requestedPath: requestedPath,
				standardPath:  path,
			}, true
		}
	}
	for _, mapping := range mappings {
		if !legacyParamSyncPathCovers(mapping.standardPath, path) {
			continue
		}
		standardPath := legacyInstantiateParamSyncPath(mapping.standardPath, mapping.standardPath, path)
		privatePath := legacyInstantiateParamSyncPath(mapping.privatePath, mapping.standardPath, path)
		for _, requestedPath := range requestedPaths {
			if legacyParamSyncRequestMatchesMapping(requestedPath, path, standardPath, privatePath, mapping) {
				return legacyParamSyncParameterMatch{
					requestedPath: requestedPath,
					standardPath:  standardPath,
					privatePath:   privatePath,
				}, true
			}
		}
	}
	return legacyParamSyncParameterMatch{}, false
}

func legacyParamSyncRequestMatchesMapping(requestedPath, actualPath, standardPath, privatePath string, mapping legacyParamSyncCoverageMapping) bool {
	return legacyParamSyncPathCovers(requestedPath, actualPath) ||
		legacyParamSyncPathCovers(requestedPath, standardPath) ||
		legacyParamSyncPathCovers(requestedPath, privatePath) ||
		legacyParamSyncPathCovers(requestedPath, mapping.standardPath) ||
		legacyParamSyncPathCovers(requestedPath, mapping.privatePath)
}

func legacyParamSyncPathCovers(pattern, actual string) bool {
	pattern = strings.TrimSpace(pattern)
	actual = strings.TrimSpace(actual)
	if pattern == "" || actual == "" {
		return false
	}
	patternIsPrefix := strings.HasSuffix(pattern, ".")
	pattern = strings.TrimSuffix(pattern, ".")
	actual = strings.TrimSuffix(actual, ".")
	if !legacyParamSyncPathHasInstanceWildcard(pattern) {
		if patternIsPrefix {
			return actual == pattern || strings.HasPrefix(actual, pattern+".")
		}
		return actual == pattern
	}
	normalizedPattern := legacyNormalizeParamSyncPath(pattern)
	normalizedActual := legacyNormalizeParamSyncPath(actual)
	if patternIsPrefix {
		return normalizedActual == normalizedPattern || strings.HasPrefix(normalizedActual, normalizedPattern+".")
	}
	return normalizedActual == normalizedPattern
}

func legacyParamSyncPathHasInstanceWildcard(path string) bool {
	for _, part := range strings.Split(path, ".") {
		if part == "{i}" {
			return true
		}
	}
	return false
}

func legacyInstantiateParamSyncPath(template, standardTemplate, actualStandard string) string {
	template = strings.TrimSpace(template)
	if template == "" {
		return ""
	}
	standardParts := strings.Split(strings.TrimSpace(standardTemplate), ".")
	actualParts := strings.Split(strings.TrimSpace(actualStandard), ".")
	if len(standardParts) != len(actualParts) {
		return template
	}
	instances := make([]string, 0, 2)
	for i, part := range standardParts {
		if part == "{i}" && i < len(actualParts) {
			instances = append(instances, actualParts[i])
		}
	}
	if len(instances) == 0 {
		return template
	}
	out := strings.Split(template, ".")
	instanceIndex := 0
	for i, part := range out {
		if part != "{i}" || instanceIndex >= len(instances) {
			continue
		}
		out[i] = instances[instanceIndex]
		instanceIndex++
	}
	return strings.Join(out, ".")
}

func legacyNormalizeParamSyncPath(path string) string {
	parts := strings.Split(path, ".")
	for i, part := range parts {
		if part == "" || part == "{i}" {
			continue
		}
		allDigits := true
		for _, r := range part {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			parts[i] = "{i}"
		}
	}
	return strings.Join(parts, ".")
}

// paramSyncRequestResultPayload / paramSyncRunResultPayload 保持旧系统
// job/result 契约的最小字段集：任务标识 + 新旧两套状态 + 结果由
// attachLegacyParamSyncParameterResults 追加（parameters/total）。
func paramSyncRequestResultPayload(req *paramsync.SyncRequest) gin.H {
	if req == nil {
		return gin.H{}
	}
	status := string(req.Status)
	return gin.H{
		"jobId":        req.ID.String(),
		"name":         "GetParameterValues",
		"status":       status,
		"sn":           req.DeviceSN,
		"errorMessage": req.ErrorMessage,
		"createTime":   req.CreatedAt,
		"completeTime": req.CompletedAt,
	}
}

func paramSyncRunResultPayload(req *paramsync.SyncRequest, run *paramsync.SyncRun) gin.H {
	if run == nil {
		return paramSyncRequestResultPayload(req)
	}
	status := string(run.Status)
	errorMessage := run.ErrorMessage
	if req != nil && req.ErrorMessage != "" && run.ErrorMessage == "" {
		errorMessage = req.ErrorMessage
	}
	return gin.H{
		"jobId":        run.ID.String(),
		"name":         "GetParameterValues",
		"status":       status,
		"sn":           run.DeviceSN,
		"errorMessage": errorMessage,
		"createTime":   run.StartedAt,
		"completeTime": run.CompletedAt,
	}
}

func isNorthboundParamSyncSourceID(taskID string) bool {
	return strings.HasPrefix(taskID, "northbound-manual:") ||
		strings.HasPrefix(taskID, "northbound-legacy-query:")
}

func paramSyncSourceIDCallerTypes(taskID string) []string {
	preferred := []string{}
	if strings.HasPrefix(taskID, "northbound-manual:") ||
		strings.HasPrefix(taskID, "northbound-legacy-query:") {
		preferred = append(preferred, "manual")
	}
	for _, callerType := range []string{"manual", "provision", "config", "acs", "license", "system"} {
		seen := false
		for _, existing := range preferred {
			if existing == callerType {
				seen = true
				break
			}
		}
		if !seen {
			preferred = append(preferred, callerType)
		}
	}
	return preferred
}

func isInvalidTaskIdentifierError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, commonerrors.ErrInvalidInput) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid input syntax for type uuid") ||
		strings.Contains(msg, "sqlstate 22p02") ||
		strings.Contains(msg, "invalid task id")
}

func isNotFoundLike(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, task.ErrTaskNotFound) || errors.Is(err, commonerrors.ErrNotFound) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "no rows")
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

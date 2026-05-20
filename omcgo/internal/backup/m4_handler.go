package backup

// M4: 运营商规范 API 补全 (backup-restore-alignment-plan M4)
//
// 本文件实现 /task/enb/config/backupRestore/* 别名路由所需的新处理函数：
//   - ListTasksAlias    POST /queryTaskList        — 包装现有 ListTasks（POST body → query）
//   - TerminateTask     POST /terminateTask        — POST body {task_id} 调用 CancelTask
//   - ExportFile        GET  /single/exportFile    — 按 SN 返回 MinIO presigned GET URL
//   - QueryCellInfos    POST /queryCellInfos       — 按运营商/产品类型/关键字过滤基站，分页
//   - QueryTaskDeviceList POST /queryTaskDeviceList — 按任务列出目标设备视图（device repo join）
//   - GetProductType    GET  /getProductType       — 枚举 device.product_class distinct 列表
//
// QueryCellInfos / QueryTaskDeviceList / GetProductType 依赖 deviceReader，
// 未注入时返回 503。

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/device"
)

// TerminateTaskRequest defines the POST body for terminateTask.
type TerminateTaskRequest struct {
	TaskID string `json:"task_id" binding:"required"`
}

// TerminateTask POST /task/enb/config/backupRestore/terminateTask
// 别名 CancelTask — 从 POST body 读 task_id（而非 URL path param），
// 调用 service.CancelTask 终止任务。
func (h *Handler) TerminateTask(c *gin.Context) {
	var req TerminateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	id, err := uuid.Parse(req.TaskID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if err := h.service.CancelTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"task_id": req.TaskID, "status": "cancelled"})
}

// ListTasksAliasRequest defines the POST body for queryTaskList.
type ListTasksAliasRequest struct {
	model.ListRequest
	Status   string `json:"status"`
	TaskType string `json:"task_type"`
}

// ListTasksAlias POST /task/enb/config/backupRestore/queryTaskList
// 别名 ListTasks — 规范要求 POST body，直接转发给 service.ListTasks。
func (h *Handler) ListTasksAlias(c *gin.Context) {
	var req ListTasksAliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// body 完全缺失时也当作空筛选处理
		req = ListTasksAliasRequest{ListRequest: model.DefaultListRequest()}
	}
	if req.Page == 0 {
		req.ListRequest = model.DefaultListRequest()
	}

	filter := TaskFilter{ListRequest: req.ListRequest}
	if req.Status != "" {
		s := TaskStatus(req.Status)
		filter.Status = &s
	}
	if req.TaskType != "" {
		t := TaskType(req.TaskType)
		filter.TaskType = &t
	}

	result, err := h.service.ListTasks(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

// ExportFileResponse is the JSON payload returned by ExportFile.
type ExportFileResponse struct {
	SerialNumber string `json:"serial_number"`
	FileName     string `json:"file_name"`
	DownloadURL  string `json:"download_url"`
	ExpiresIn    int    `json:"expires_in_seconds"` // always 3600
}

// ExportFile GET /task/enb/config/backupRestore/single/exportFile?sn=<sn>
//
// 按 SN 从 backup_restore_file 表查出最新备份文件元数据，
// 通过 MinIO PresignedGetObject 生成 1h 有效下载链接后返回。
//
// 依赖：h.fileRepo 和 h.minioClient 均需注入；否则返回 503。
func (h *Handler) ExportFile(c *gin.Context) {
	if h.fileRepo == nil || h.minioClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ret":  0,
			"msg":  "export file not configured: fileRepo or minioClient not wired",
			"data": nil,
		})
		return
	}

	sn := strings.TrimSpace(c.Query("sn"))
	if sn == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	files, err := h.fileRepo.ListBySerial(c.Request.Context(), sn)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	if len(files) == 0 {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	// 取最新一条（ListBySerial 已按 update_time DESC 排序）
	latest := files[0]
	if latest.ObjectPath == "" {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	bucket, objectPath, parseErr := splitBucketAndPath(latest.ObjectPath)
	if parseErr != nil {
		h.logger.Warn("export file: invalid object_path",
			zap.String("sn", sn),
			zap.String("object_path", latest.ObjectPath),
			zap.Error(parseErr))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, parseErr)
		return
	}

	presigned, presignErr := h.minioClient.PresignedGetObject(
		c.Request.Context(), bucket, objectPath, time.Hour, url.Values{},
	)
	if presignErr != nil {
		h.logger.Warn("export file: presign failed",
			zap.String("sn", sn), zap.Error(presignErr))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, presignErr)
		return
	}

	response.OK(c, ExportFileResponse{
		SerialNumber: sn,
		FileName:     latest.FileName,
		DownloadURL:  presigned.String(),
		ExpiresIn:    3600,
	})
}

// QueryCellInfosRequest defines the POST body for queryCellInfos.
// All fields optional; empty filters return the first page of all visible
// devices (subject to ListRequest pagination).
type QueryCellInfosRequest struct {
	model.ListRequest
	OperatorCode string `json:"operator_code"` // 运营商编码 CMCC/CTCC/CUCC（可空）
	ProductType  string `json:"product_type"`  // device.product_class（可空）
	Search       string `json:"search"`        // 模糊匹配 SN/site_name 等
}

// CellInfo 是基站精简视图，供前端基站选择面板使用。
type CellInfo struct {
	ID           uuid.UUID `json:"id"`
	SerialNumber string    `json:"serial_number"`
	SiteName     string    `json:"site_name"`
	SiteID       string    `json:"site_id"`
	Carrier      string    `json:"carrier"`
	ProductClass string    `json:"product_class"`
	Status       string    `json:"status"`
}

// QueryCellInfos POST /task/enb/config/backupRestore/queryCellInfos
//
// 按运营商 / 产品类型 / 关键字过滤基站列表，分页返回。
func (h *Handler) QueryCellInfos(c *gin.Context) {
	if h.deviceReader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ret":  0,
			"msg":  "queryCellInfos not configured: deviceReader not wired",
			"data": nil,
		})
		return
	}

	var req QueryCellInfosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = QueryCellInfosRequest{ListRequest: model.DefaultListRequest()}
	}
	if req.Page == 0 {
		req.ListRequest = model.DefaultListRequest()
	}

	filter := device.DeviceFilter{ListRequest: req.ListRequest}
	if req.OperatorCode != "" {
		cc := model.CarrierCode(strings.ToLower(req.OperatorCode))
		filter.Carrier = &cc
	}
	if req.ProductType != "" {
		filter.ProductClass = &req.ProductType
	}
	if req.Search != "" {
		filter.Search = &req.Search
	}

	resp, err := h.deviceReader.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	cells := make([]CellInfo, 0, len(resp.Items))
	for i := range resp.Items {
		d := resp.Items[i]
		cells = append(cells, CellInfo{
			ID:           d.ID,
			SerialNumber: d.SerialNumber,
			SiteName:     d.DeviceName,
			SiteID:       d.SiteID,
			Carrier:      string(d.Carrier),
			ProductClass: d.ProductClass,
			Status:       string(d.Status),
		})
	}

	response.OK(c, model.NewListResponse(cells, resp.Total, resp.Page, resp.PageSize))
}

// QueryTaskDeviceListRequest defines the POST body for queryTaskDeviceList.
type QueryTaskDeviceListRequest struct {
	TaskID string `json:"task_id" binding:"required"`
}

// TaskDeviceInfo 描述某次备份/恢复任务下一台目标设备的执行视图。
// 当前 v1 仅展示设备静态信息 + 任务整体状态；按设备粒度的执行细节
// 需要 device_tasks join 后补齐（M5 范畴）。
type TaskDeviceInfo struct {
	SerialNumber string `json:"serial_number"`
	SiteName     string `json:"site_name"`
	SiteID       string `json:"site_id"`
	Carrier      string `json:"carrier"`
	ProductClass string `json:"product_class"`
	DeviceStatus string `json:"device_status"`
	// 任务整体状态镜像 — v1 简化为同一字段，未来按 device_tasks 精细化。
	TaskStatus string `json:"task_status"`
}

// QueryTaskDeviceList POST /task/enb/config/backupRestore/queryTaskDeviceList
//
// 列出指定备份/恢复任务下每台目标设备的视图。v1 通过 BackupTask.TargetIDs
// 反查 devices 表，task_status 镜像 BackupTask.Status。
func (h *Handler) QueryTaskDeviceList(c *gin.Context) {
	if h.deviceReader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ret":  0,
			"msg":  "queryTaskDeviceList not configured: deviceReader not wired",
			"data": nil,
		})
		return
	}

	var req QueryTaskDeviceListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	taskID, err := uuid.Parse(req.TaskID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), taskID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	out := make([]TaskDeviceInfo, 0, len(task.TargetIDs))
	for _, sn := range task.TargetIDs {
		info := TaskDeviceInfo{
			SerialNumber: sn,
			TaskStatus:   string(task.Status),
		}
		dev, getErr := h.deviceReader.GetBySerialNumber(c.Request.Context(), sn)
		if getErr == nil && dev != nil {
			info.SiteName = dev.DeviceName
			info.SiteID = dev.SiteID
			info.Carrier = string(dev.Carrier)
			info.ProductClass = dev.ProductClass
			info.DeviceStatus = string(dev.Status)
		} else if getErr != nil {
			h.logger.Debug("queryTaskDeviceList: device lookup failed",
				zap.String("sn", sn), zap.Error(getErr))
		}
		out = append(out, info)
	}

	response.OK(c, gin.H{
		"task_id": task.ID.String(),
		"status":  string(task.Status),
		"devices": out,
	})
}

// GetProductType GET /task/enb/config/backupRestore/getProductType
//
// 枚举 devices.product_class 的 distinct 列表（按字母序）。
func (h *Handler) GetProductType(c *gin.Context) {
	if h.deviceReader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ret":  0,
			"msg":  "getProductType not configured: deviceReader not wired",
			"data": nil,
		})
		return
	}
	classes, err := h.deviceReader.ListProductClasses(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"product_types": classes})
}

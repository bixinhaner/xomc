package rebootrecord

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type Handler struct {
	svc      *Service
	resolver *authz.Resolver
	logger   *zap.Logger
}

func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger.Named("rebootrecord-handler")}
}

// SetPermissionService 注入数据权限解析器（#63 设备组可见性强制层）。未注入时
// FromContext 走 nil-safe 退化（不过滤），与 device/alarm 模块语义一致。
func (h *Handler) SetPermissionService(perm authz.VisibleGroupsResolver) {
	h.resolver = authz.NewResolver(perm)
}

// RegisterRoutes 挂载于 /api/v1 下：
//
//	GET /reboot-records             统一重启记录列表（普通 + 异常，按重启类型过滤）
//	GET /reboot-records/statistics  按设备聚合（总次数 / 异常次数，跟随过滤）
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/reboot-records")
	g.GET("", h.List)
	g.GET("/statistics", h.Statistics)
}

// parseFilter 从 query 解析统一过滤条件（List 与 Statistics 共用）。
func parseFilter(c *gin.Context) Filter {
	f := Filter{Page: 1, PageSize: 20}
	if v := strings.TrimSpace(c.Query("device_sn")); v != "" {
		f.DeviceSN = v
	}
	if v := strings.TrimSpace(c.Query("device_type")); v != "" {
		f.DeviceType = v
	}
	f.RebootType = ParseRebootType(strings.TrimSpace(c.Query("reboot_type")))
	if v := c.Query("start_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.StartTime = &t
		}
	}
	if v := c.Query("end_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.EndTime = &t
		}
	}
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			f.Page = p
		}
	}
	if v := c.Query("page_size"); v != "" {
		if s, err := strconv.Atoi(v); err == nil && s > 0 {
			f.PageSize = s
		}
	}
	return f
}

// List godoc
// @Summary 统一重启记录列表（event_logs ∪ station_fault_logs）
// @Param device_sn   query string false "设备 SN（模糊匹配 ILIKE）"
// @Param device_type query string false "设备类型：eNB / gNB / GSM / UPS"
// @Param reboot_type query string false "重启类型：normal / abnormal / 空=全部"
// @Param start_time  query string false "起始时间 RFC3339"
// @Param end_time    query string false "结束时间 RFC3339"
// @Param page        query int    false "页码 默认 1"
// @Param page_size   query int    false "每页 默认 20"
func (h *Handler) List(c *gin.Context) {
	f := parseFilter(c)
	groups, ok := h.resolver.FromContext(c)
	if !ok {
		return
	}
	f.VisibleGroups = groups
	items, total, err := h.svc.List(c.Request.Context(), f)
	if err != nil {
		h.logger.Error("list reboot records", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{
		"items": items,
		"total": total,
		"page":  f.Page,
		"size":  f.PageSize,
	})
}

// Statistics godoc
// @Summary 按设备聚合重启次数（总次数 / 异常次数，跟随过滤、不分页）
func (h *Handler) Statistics(c *gin.Context) {
	f := parseFilter(c)
	f.Page = 1
	f.PageSize = 0 // 统计不分页

	groups, ok := h.resolver.FromContext(c)
	if !ok {
		return
	}
	f.VisibleGroups = groups

	items, err := h.svc.StatByDevice(c.Request.Context(), f)
	if err != nil {
		h.logger.Error("stat reboot records by device", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{
		"items": items,
		"total": len(items),
	})
}

package pm

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// T-0193 自选设备下钻到小区/PLMN —— 列设备小区/PLMN 只读接口。
//
// GET /pm/metrics/objects?device_sns=SN1,SN2&technology=lte
//
// 入参：
//   - device_sns（必填）：一批设备 SN，逗号分隔或重复 query（device_sns=A&device_sns=B）
//   - technology（可选）：lte / nr / gsm，限定到该制式设备；空 = 不限制式
//
// 出参：这些设备在最细原始表 pm_metrics 里实际出现过的 distinct object_ldn 清单，
// 每项附解析出的 cell_id / plmn（供前端做友好名）。
// 空设备列表 / 跨制式无数据 → 返回空清单（不报错）。

// objectItem 是单个小区/PLMN 项的响应体。
type objectItem struct {
	ObjectLDN string `json:"object_ldn"`        // 原始字符串（白名单用它）
	CellID    string `json:"cell_id,omitempty"` // 解析出的小区号（友好名用）
	PLMN      string `json:"plmn,omitempty"`    // 解析出的 PLMN（友好名用）
}

// ListMetricObjects handles GET /pm/metrics/objects（T-0193）。
//
// #18 分层收敛：SQL 查询下沉到 DeviceQueryService（handler → service →
// repository），Handler 只做参数解析 + 响应组装。
func (h *Handler) ListMetricObjects(c *gin.Context) {
	deviceSNs := parseCSVQuery(c, "device_sns")
	// 空设备列表 → 直接返回空清单（不报错，不打 DB）。
	if len(deviceSNs) == 0 {
		response.OK(c, gin.H{"items": []objectItem{}, "total": 0})
		return
	}

	var technologies []string
	if tech := strings.ToLower(strings.TrimSpace(c.Query("technology"))); tech != "" {
		technologies = []string{tech}
	}

	startTime, endTime := parseOptionalObjectTimeRange(c)

	objs, err := h.deviceQuery.ListMetricObjects(c.Request.Context(), deviceSNs, technologies, startTime, endTime)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	items := make([]objectItem, 0, len(objs))
	for _, o := range objs {
		items = append(items, objectItem{ObjectLDN: o.ObjectLDN, CellID: o.CellID, PLMN: o.PLMN})
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func parseOptionalObjectTimeRange(c *gin.Context) (time.Time, time.Time) {
	var startTime, endTime time.Time
	if v := c.Query("start_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			startTime = t
		}
	}
	if v := c.Query("end_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			endTime = t
		}
	}
	return startTime, endTime
}

// parseCSVQuery 取一个可逗号分隔或重复出现的 query 参数，拆成去空去重的字符串切片。
func parseCSVQuery(c *gin.Context, key string) []string {
	raw := c.QueryArray(key) // 支持重复 query：device_sns=A&device_sns=B
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{})
	for _, v := range raw {
		for _, part := range strings.Split(v, ",") {
			if p := strings.TrimSpace(part); p != "" {
				if _, dup := seen[p]; !dup {
					seen[p] = struct{}{}
					out = append(out, p)
				}
			}
		}
	}
	return out
}

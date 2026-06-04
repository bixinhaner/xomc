package pm

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/pm/metrics"
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

// parseObjectLDN 从原始 object_ldn 拆出 cell_id / plmn（缺段则留空）。
// 委托给 metrics.ParseObjectLDN 单一真值源（与 aggregator KPI 跨层级配对同口径），不另造解析。
func parseObjectLDN(ldn string) (cellID, plmn string) {
	return metrics.ParseObjectLDN(ldn)
}

// buildObjectsQuery 纯函数：拼"列设备小区/PLMN"查询 SQL + 占位参数。
//
//   - $1 = device_sns（TEXT[]）
//   - 制式过滤（technologies 非空时）照 applyCommonFilters 范式：
//     (device_oui, device_sn) IN (SELECT oui, serial_number FROM devices WHERE technology = ANY($2))
//
// 抽出便于单测断言（device 过滤 + 制式过滤 + distinct）。
func buildObjectsQuery(deviceSNs, technologies []string) (string, []any) {
	q := `SELECT DISTINCT object_ldn
FROM pm_metrics
WHERE device_sn = ANY($1)
  AND object_ldn <> ''`
	args := []any{deviceSNs}
	if len(technologies) > 0 {
		q += `
  AND (device_oui, device_sn) IN (SELECT oui, serial_number FROM devices WHERE technology = ANY($2))`
		args = append(args, technologies)
	}
	q += `
ORDER BY object_ldn`
	return q, args
}

// ListMetricObjects handles GET /pm/metrics/objects（T-0193）。
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

	q, args := buildObjectsQuery(deviceSNs, technologies)
	rows, err := h.pool.Query(c.Request.Context(), q, args...)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	items := make([]objectItem, 0)
	for rows.Next() {
		var ldn string
		if err := rows.Scan(&ldn); err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		cellID, plmn := parseObjectLDN(ldn)
		items = append(items, objectItem{ObjectLDN: ldn, CellID: cellID, PLMN: plmn})
	}
	if err := rows.Err(); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
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

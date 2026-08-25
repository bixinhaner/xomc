package pm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// fakeDeviceQuery 是 DeviceQueryService 的测试替身（#18 收敛后 Handler 经此接口
// 而非直连 SQL 池）。
type fakeDeviceQuery struct {
	lookupFn func(ctx context.Context, deviceID uuid.UUID) (string, string, error)
	listFn   func(ctx context.Context, deviceSNs, technologies []string, startTime, endTime time.Time) ([]MetricObject, error)
	groupsFn func(ctx context.Context, deviceID uuid.UUID) ([]uuid.UUID, error)
}

func (f *fakeDeviceQuery) LookupDeviceOUISN(ctx context.Context, deviceID uuid.UUID) (string, string, error) {
	if f.lookupFn != nil {
		return f.lookupFn(ctx, deviceID)
	}
	return "", "", nil
}

func (f *fakeDeviceQuery) ListMetricObjects(ctx context.Context, deviceSNs, technologies []string, startTime, endTime time.Time) ([]MetricObject, error) {
	if f.listFn != nil {
		return f.listFn(ctx, deviceSNs, technologies, startTime, endTime)
	}
	return nil, nil
}

func (f *fakeDeviceQuery) DeviceGroupIDs(ctx context.Context, deviceID uuid.UUID) ([]uuid.UUID, error) {
	if f.groupsFn != nil {
		return f.groupsFn(ctx, deviceID)
	}
	return nil, nil
}

func objSetupRouter(dq DeviceQueryService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &Handler{deviceQuery: dq, logger: zap.NewNop()}
	h.RegisterRoutes(r.Group(""))
	return r
}

// 成功路径：Handler 把 query 拆出的 SN/制式透传给 Service，并把返回项原样组装。
func Test_ListMetricObjects_RoutesThroughService(t *testing.T) {
	var gotSNs, gotTechs []string
	var gotStart, gotEnd time.Time
	dq := &fakeDeviceQuery{
		listFn: func(_ context.Context, deviceSNs, technologies []string, startTime, endTime time.Time) ([]MetricObject, error) {
			gotSNs, gotTechs = deviceSNs, technologies
			gotStart, gotEnd = startTime, endTime
			return []MetricObject{
				{ObjectLDN: "Cellid=111,PLMN=46068", CellID: "111", PLMN: "46068"},
			}, nil
		},
	}
	router := objSetupRouter(dq)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/metrics/objects?device_sns=SN-1,SN-2&technology=LTE&start_time=2026-08-22T11:51:54Z&end_time=2026-08-23T11:51:54Z", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"SN-1", "SN-2"}, gotSNs)
	assert.Equal(t, []string{"lte"}, gotTechs, "technology 小写归一后传给 service")
	assert.Equal(t, "2026-08-22T11:51:54Z", gotStart.Format(time.RFC3339))
	assert.Equal(t, "2026-08-23T11:51:54Z", gotEnd.Format(time.RFC3339))

	var body struct {
		Items []objectItem `json:"items"`
		Total int          `json:"total"`
	}
	response.DecodeData(t, w.Body, &body)
	assert.Equal(t, 1, body.Total)
	assert.Equal(t, "111", body.Items[0].CellID)
}

// 空设备列表：不打 Service，直接返回空清单。
func Test_ListMetricObjects_EmptyDevices_NoServiceCall(t *testing.T) {
	called := false
	dq := &fakeDeviceQuery{
		listFn: func(_ context.Context, _, _ []string, _, _ time.Time) ([]MetricObject, error) {
			called = true
			return nil, nil
		},
	}
	router := objSetupRouter(dq)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/metrics/objects", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, called, "空设备列表应短路，不调用 service")
}

// 失败路径：Service 报错 → 500。
func Test_ListMetricObjects_ServiceError(t *testing.T) {
	dq := &fakeDeviceQuery{
		listFn: func(_ context.Context, _, _ []string, _, _ time.Time) ([]MetricObject, error) {
			return nil, errors.New("db down")
		},
	}
	router := objSetupRouter(dq)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/metrics/objects?device_sns=SN-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// device 过滤 + distinct + 排序：不带制式/时间时只有 $1，无额外过滤。
func Test_buildObjectsQuery_DeviceOnly(t *testing.T) {
	q, args := buildObjectsQuery([]string{"SN-1", "SN-2"}, nil, time.Time{}, time.Time{})

	assert.Contains(t, q, "WITH target_devices AS MATERIALIZED")
	assert.Contains(t, q, "SELECT DISTINCT object_ldn")
	assert.Contains(t, q, "JOIN pm_measurement_anchors")
	assert.Contains(t, q, "serial_number = ANY($1)")
	assert.Contains(t, q, "a.granularity = '15min'")
	assert.Contains(t, q, "a.object_ldn <> ''")
	assert.NotContains(t, q, "technology = ANY")
	assert.NotContains(t, q, `a."time" >=`)
	assert.Contains(t, q, "ORDER BY object_ldn")
	assert.Equal(t, []any{[]string{"SN-1", "SN-2"}}, args)
}

// 带制式：叠加 devices 子查询制式过滤（$2），照 applyCommonFilters 范式。
func Test_buildObjectsQuery_WithTechnology(t *testing.T) {
	q, args := buildObjectsQuery([]string{"SN-1"}, []string{"lte"}, time.Time{}, time.Time{})

	assert.Contains(t, q, "serial_number = ANY($1)")
	assert.Contains(t, q, "a.granularity = '15min'")
	assert.Contains(t, q, "technology = ANY($2)")
	assert.Len(t, args, 2)
	assert.Equal(t, []string{"SN-1"}, args[0])
	assert.Equal(t, []string{"lte"}, args[1])
}

func Test_buildObjectsQuery_WithTimeRange(t *testing.T) {
	start := time.Date(2026, 8, 22, 11, 51, 54, 0, time.UTC)
	end := time.Date(2026, 8, 23, 11, 51, 54, 0, time.UTC)
	q, args := buildObjectsQuery([]string{"SN-1"}, []string{"lte"}, start, end)

	assert.Contains(t, q, "technology = ANY($2)")
	assert.Contains(t, q, `a."time" >= $3`)
	assert.Contains(t, q, `a."time" < $4`)
	assert.Equal(t, []string{"SN-1"}, args[0])
	assert.Equal(t, []string{"lte"}, args[1])
	assert.Equal(t, start, args[2])
	assert.Equal(t, end, args[3])
}

// 空入参：SQL 仍合法（device_sn = ANY($1) 对空切片 → 无命中），返回空清单由 handler 兜底。
func Test_buildObjectsQuery_EmptyDevices(t *testing.T) {
	q, args := buildObjectsQuery([]string{}, nil, time.Time{}, time.Time{})

	assert.Contains(t, q, "serial_number = ANY($1)")
	assert.NotContains(t, q, "technology")
	assert.Len(t, args, 1)
}

// parseObjectLDN 拆 Cellid / PLMN。
func Test_parseObjectLDN(t *testing.T) {
	tests := []struct {
		name       string
		ldn        string
		wantCellID string
		wantPLMN   string
	}{
		{"full", "Cellid=111172245,PLMN=46068", "111172245", "46068"},
		{"cell only", "Cellid=12345", "12345", ""},
		{"plmn only", "PLMN=46000", "", "46000"},
		{"none", "Foo=bar", "", ""},
		{"empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cellID, plmn := parseObjectLDN(tt.ldn)
			assert.Equal(t, tt.wantCellID, cellID)
			assert.Equal(t, tt.wantPLMN, plmn)
		})
	}
}

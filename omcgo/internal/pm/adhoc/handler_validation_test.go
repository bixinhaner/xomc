package adhoc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-0182：建任务校验单测。
//
// 这些 case 全部在 handler 进 repo / DB 之前的纯校验阶段命中，故用 nil pool 的 stub repo 即可：
//   - granularities len != 1 → 400（成功 = 恰好 1 个；失败 = 0 个 / 多个）
//   - technology DTO binding oneof=lte nr gsm → 非法制式值 400（建任务拒非法制式入口）
//
// 跨制式（设备制式与任务制式不一致）的"全链路拒绝"需真实 devices 表查询（rejectCrossTechnology
// 走 h.pool），属集成/真机验证范畴；本单测覆盖到 DTO 制式合法性这层入口校验。

func postCreate(t *testing.T, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	r := newTestRouter(&handlerStubRepo{})
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/adhoc/tasks", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func baseCreateBody() map[string]any {
	return map[string]any{
		"name": "t", "mode": "oneshot",
		"device_sns":   []string{"S1"},
		"metric_paths": []string{"M1"},
		"window_start": "2026-05-22T10:00:00Z",
		"window_end":   "2026-05-22T11:00:00Z",
	}
}

// 成功路径：granularities 恰好 1 个 → 201。
func Test_Handler_Create_SingleGranularity_OK(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 失败路径：granularities 多个 → 400（设计 §2.4/§2.6 单粒度）。
func Test_Handler_Create_MultiGranularity_Rejected(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly", "daily"}
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 失败路径：granularities 为空 → 400（binding required,min=1 命中）。
func Test_Handler_Create_EmptyGranularity_Rejected(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{}
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 成功路径：technology 合法值（lte）+ 单粒度 → 201（无 pool 时跨制式校验因 device_sns 仍会触发，
// 这里用空 device_sns 以隔离到 DTO 校验层；建任务允许空 device_sns by binding? device_sns required,min=1）。
//
// 注意：technology != "" 且 device_sns 非空时 handler 会调 rejectCrossTechnology（需 pool）。
// 为隔离 DTO 制式合法性校验，这里只断言"非法制式值在 bind 阶段即被拒"，合法值的全链路放真机验证。

// 失败路径：technology 非法值（5g）→ 400（DTO binding oneof=lte nr gsm 命中，建任务拒非法制式）。
func Test_Handler_Create_InvalidTechnology_Rejected(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["technology"] = "5g" // 非 lte/nr/gsm
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 失败路径：dimension 非法值 → 400（DTO binding oneof=device aggregate_group product band 命中）。
func Test_Handler_Create_InvalidDimension_Rejected(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "site" // 非合法维度
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 成功路径：dimension=product 合法 → 201（product 已入枚举，建任务可选）。
func Test_Handler_Create_ProductDimension_OK(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "product"
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 成功路径：dimension=network 合法 → 201（T-0184 全网维度已入 binding oneof）。
func Test_Handler_Create_NetworkDimension_OK(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "network"
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 成功路径：dimension=device_group 合法 → 201（T-0184 设备组维度已入 binding oneof）。
func Test_Handler_Create_DeviceGroupDimension_OK(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "device_group"
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ── #363：(粒度, 维度) 组合守门 ────────────────────────────────────────────

// 失败路径：dimension=device_group + granularities=['15min'] → 400（无 15min 级设备组聚合源，
// 前置守门拦下，不再落库等 worker 跑才失败）。错误消息明确指向设备组不支持 15min。
func Test_Handler_Create_DeviceGroup15Min_Rejected(t *testing.T) {
	var created bool
	repo := &handlerStubRepo{
		create: func(CreateRequest) (uuid.UUID, error) { created = true; return uuid.New(), nil },
	}
	b := baseCreateBody()
	b["granularities"] = []string{"15min"}
	b["dimension"] = "device_group"
	w := postCreateWithRepo(t, repo, b)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "device_group")
	assert.Contains(t, w.Body.String(), "15min")
	assert.False(t, created, "非法组合不应落库")
}

// 成功路径：dimension=device_group + granularities=['hourly'] → 201（不误伤合法组合）。
func Test_Handler_Create_DeviceGroupHourly_OK(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "device_group"
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 成功路径：dimension=device（默认）+ granularities=['15min'] → 201（设备维度 15min 合法）。
func Test_Handler_Create_Device15Min_OK(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"15min"}
	// dimension 不填 → 默认 device
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 失败路径：编辑自建 device_group 任务，把粒度改成 15min → 400（编辑分支同样守门）。
func Test_Handler_Update_DeviceGroup15Min_Rejected(t *testing.T) {
	id := uuid.New()
	var updated bool
	repo := &handlerStubRepo{
		get: func(uuid.UUID) (*Task, error) {
			return &Task{ID: id, IsBuiltin: false, Mode: ModeContinuous, Dimension: DimensionDeviceGroup, Creator: "anonymous"}, nil
		},
		update: func(uuid.UUID, UpdateRequest) error { updated = true; return nil },
	}
	b := map[string]any{
		"metric_paths":  []string{"M1"},
		"granularities": []string{"15min"},
	}
	w := patchUpdate(t, repo, id, b)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "device_group")
	assert.False(t, updated, "非法组合不应更新")
}

// ── T-0185：device_sns / window 放宽校验 ──────────────────────────────────

// postCreateWithRepo 用注入的 stub repo 跑建任务（用于捕获 CreateRequest 验证派生值）。
func postCreateWithRepo(t *testing.T, repo Repository, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	r := newTestRouter(repo)
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/adhoc/tasks", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// 成功路径：非 device 维度（network）+ 空 device_sns → 201（T-0185 放宽：
// network/product/band/device_group 按制式全量聚合，不必填设备）。
func Test_Handler_Create_NonDeviceDimension_EmptyDeviceSNs_OK(t *testing.T) {
	b := baseCreateBody()
	delete(b, "device_sns")
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "network"
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 失败路径：device 维度（默认）+ 空 device_sns → 400（自选设备维度必须给设备）。
func Test_Handler_Create_DeviceDimension_EmptyDeviceSNs_Rejected(t *testing.T) {
	b := baseCreateBody()
	delete(b, "device_sns")
	b["granularities"] = []string{"hourly"}
	// dimension 不填 → 默认 device
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 失败路径：aggregate_group 维度 + 空 device_sns → 400（与 device 同，临时组也要设备）。
func Test_Handler_Create_AggregateGroupDimension_EmptyDeviceSNs_Rejected(t *testing.T) {
	b := baseCreateBody()
	delete(b, "device_sns")
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "aggregate_group"
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 成功路径：continuous + 无 window → 201，且 cron 按粒度派生、window 清零（存 NULL 开窗滚动）。
func Test_Handler_Create_Continuous_NoWindow_DerivesCron_ClearsWindow(t *testing.T) {
	var captured CreateRequest
	repo := &handlerStubRepo{
		create: func(req CreateRequest) (uuid.UUID, error) {
			captured = req
			return uuid.New(), nil
		},
	}
	b := map[string]any{
		"name": "c", "mode": "continuous",
		"dimension":    "network",
		"metric_paths": []string{"M1"},
		"granularities": []string{"daily"},
		// 不带 device_sns / window
	}
	w := postCreateWithRepo(t, repo, b)
	assert.Equal(t, http.StatusCreated, w.Code)
	// continuous 强制清零 window → 存 NULL 开窗
	assert.True(t, captured.WindowStart.IsZero(), "continuous window_start 应清零")
	assert.True(t, captured.WindowEnd.IsZero(), "continuous window_end 应清零")
	// cron 按粒度派生（daily → "10 0 * * *"）
	require.NotNil(t, captured.CronExpr)
	assert.Equal(t, cronForGranularity("daily"), *captured.CronExpr)
}

// 失败路径：oneshot + 无 window → 400（oneshot 必须给有效时间窗）。
func Test_Handler_Create_Oneshot_NoWindow_Rejected(t *testing.T) {
	b := map[string]any{
		"name": "o", "mode": "oneshot",
		"dimension":     "network",
		"metric_paths":  []string{"M1"},
		"granularities": []string{"hourly"},
		// 不带 window
	}
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 成功路径：oneshot + 有效 window（end>start）→ 201。
func Test_Handler_Create_Oneshot_ValidWindow_OK(t *testing.T) {
	b := map[string]any{
		"name": "o", "mode": "oneshot",
		"dimension":     "network",
		"metric_paths":  []string{"M1"},
		"granularities": []string{"hourly"},
		"window_start":  "2026-05-22T10:00:00Z",
		"window_end":    "2026-05-22T11:00:00Z",
	}
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 失败路径：oneshot + window_end <= window_start → 400。
func Test_Handler_Create_Oneshot_InvalidWindow_Rejected(t *testing.T) {
	b := map[string]any{
		"name": "o", "mode": "oneshot",
		"dimension":     "network",
		"metric_paths":  []string{"M1"},
		"granularities": []string{"hourly"},
		"window_start":  "2026-05-22T11:00:00Z",
		"window_end":    "2026-05-22T10:00:00Z",
	}
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── T-0194：编辑任务（PATCH /pm/adhoc/tasks/:id）校验 ─────────────────────────

// patchUpdate 用注入 get/update 的 stub repo 跑编辑任务。
func patchUpdate(t *testing.T, repo Repository, id uuid.UUID, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	r := newTestRouter(repo)
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/pm/adhoc/tasks/"+id.String(), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// 成功路径：自建任务（device 维度 oneshot）改指标/设备/粒度/时窗 → 200，捕获的 UpdateRequest 字段正确。
func Test_Handler_Update_Adhoc_Success(t *testing.T) {
	id := uuid.New()
	var captured UpdateRequest
	repo := &handlerStubRepo{
		get: func(uuid.UUID) (*Task, error) {
			return &Task{ID: id, IsBuiltin: false, Mode: ModeOneshot, Dimension: DimensionDevice, Technology: "", Creator: "anonymous"}, nil
		},
		update: func(_ uuid.UUID, req UpdateRequest) error { captured = req; return nil },
	}
	b := map[string]any{
		"name":          "edited",
		"device_sns":    []string{"S1", "S2"},
		"metric_paths":  []string{"K1001", "K1002"},
		"granularities": []string{"daily"},
		"window_start":  "2026-05-22T10:00:00Z",
		"window_end":    "2026-05-22T11:00:00Z",
	}
	w := patchUpdate(t, repo, id, b)
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, captured.IsBuiltin)
	assert.Equal(t, "edited", captured.Name)
	assert.Equal(t, []string{"S1", "S2"}, captured.DeviceSNs)
	assert.Equal(t, []string{"K1001", "K1002"}, captured.MetricPaths)
	assert.Equal(t, []string{"daily"}, captured.Granularities)
	assert.False(t, captured.WindowStart.IsZero())
}

// 失败路径：自建 device 维度 + 空 device_sns → 400（必填设备）。
func Test_Handler_Update_Adhoc_EmptyDeviceSNs_Rejected(t *testing.T) {
	id := uuid.New()
	repo := &handlerStubRepo{
		get: func(uuid.UUID) (*Task, error) {
			return &Task{ID: id, IsBuiltin: false, Mode: ModeOneshot, Dimension: DimensionDevice, Creator: "anonymous"}, nil
		},
	}
	b := map[string]any{
		"metric_paths":  []string{"M1"},
		"granularities": []string{"hourly"},
		"window_start":  "2026-05-22T10:00:00Z",
		"window_end":    "2026-05-22T11:00:00Z",
		// 不带 device_sns
	}
	w := patchUpdate(t, repo, id, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 失败路径：自建 oneshot + window_end <= window_start → 400。
func Test_Handler_Update_Adhoc_InvalidWindow_Rejected(t *testing.T) {
	id := uuid.New()
	repo := &handlerStubRepo{
		get: func(uuid.UUID) (*Task, error) {
			return &Task{ID: id, IsBuiltin: false, Mode: ModeOneshot, Dimension: DimensionNetwork, Creator: "anonymous"}, nil
		},
	}
	b := map[string]any{
		"metric_paths":  []string{"M1"},
		"granularities": []string{"hourly"},
		"window_start":  "2026-05-22T11:00:00Z",
		"window_end":    "2026-05-22T10:00:00Z",
	}
	w := patchUpdate(t, repo, id, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 失败路径：自建多粒度 → 400（单粒度约束）。
func Test_Handler_Update_Adhoc_MultiGranularity_Rejected(t *testing.T) {
	id := uuid.New()
	repo := &handlerStubRepo{
		get: func(uuid.UUID) (*Task, error) {
			return &Task{ID: id, IsBuiltin: false, Mode: ModeOneshot, Dimension: DimensionNetwork, Creator: "anonymous"}, nil
		},
	}
	b := map[string]any{
		"metric_paths":  []string{"M1"},
		"granularities": []string{"hourly", "daily"},
		"window_start":  "2026-05-22T10:00:00Z",
		"window_end":    "2026-05-22T11:00:00Z",
	}
	w := patchUpdate(t, repo, id, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 成功路径：内置任务只取 metric_paths，传入的结构性字段（device_sns/granularities/window）被忽略；
// IsBuiltin 透传到 UpdateRequest，且不跑结构性校验（多粒度/缺设备都不影响）。
func Test_Handler_Update_Builtin_OnlyMetricPaths(t *testing.T) {
	id := uuid.New()
	var captured UpdateRequest
	repo := &handlerStubRepo{
		get: func(uuid.UUID) (*Task, error) {
			return &Task{ID: id, IsBuiltin: true, Mode: ModeContinuous, Dimension: DimensionNetwork, Technology: "lte"}, nil
		},
		update: func(_ uuid.UUID, req UpdateRequest) error { captured = req; return nil },
	}
	b := map[string]any{
		"name":          "should-be-ignored",
		"device_sns":    []string{"X1"},
		"metric_paths":  []string{"K2001"},
		"granularities": []string{"hourly", "daily"}, // 多粒度对内置无所谓（不校验）
	}
	w := patchUpdate(t, repo, id, b)
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, captured.IsBuiltin)
	assert.Equal(t, []string{"K2001"}, captured.MetricPaths)
	// 内置守门：handler 不把结构性字段塞进 UpdateRequest（保持零值）
	assert.Empty(t, captured.Name)
	assert.Empty(t, captured.DeviceSNs)
	assert.Empty(t, captured.Granularities)
}

// 失败路径：缺 metric_paths → 400（binding required,min=1）。
func Test_Handler_Update_MissingMetricPaths_Rejected(t *testing.T) {
	id := uuid.New()
	repo := &handlerStubRepo{
		get: func(uuid.UUID) (*Task, error) {
			return &Task{ID: id, IsBuiltin: true}, nil
		},
	}
	b := map[string]any{"granularities": []string{"hourly"}}
	w := patchUpdate(t, repo, id, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 失败路径：任务不存在 → 404。
func Test_Handler_Update_NotFound(t *testing.T) {
	id := uuid.New()
	repo := &handlerStubRepo{
		get: func(uuid.UUID) (*Task, error) { return nil, ErrNotFound },
	}
	b := map[string]any{"metric_paths": []string{"M1"}}
	w := patchUpdate(t, repo, id, b)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

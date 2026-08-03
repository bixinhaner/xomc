package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

type recordingBuiltInPermissionReconciler struct {
	calls  int
	result BuiltInAPIPermissionGrantResult
	err    error
}

type recordingBatchApiEndpointRepository struct {
	ApiEndpointRepository
	inputs  []ApiEndpointUpsertInput
	created int
}

func (r *recordingBatchApiEndpointRepository) UpsertBatch(_ context.Context, inputs []ApiEndpointUpsertInput) (int, error) {
	r.inputs = inputs
	return r.created, nil
}

func (r *recordingBuiltInPermissionReconciler) ReconcileBuiltInAPIPermissions(context.Context) (BuiltInAPIPermissionGrantResult, error) {
	r.calls++
	return r.result, r.err
}

func strPtr(s string) *string { return &s }

// TestUpdateApiEndpoint_PathMethodPassThrough 验证：
// 子①——更新请求携带 Path/Method 时，被原样透传到 repo.Update（不被吞掉），
// 且 service 返回 repo 回读出的新值（死判 pathmethod_save）。
func TestUpdateApiEndpoint_PathMethodPassThrough(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name       string
		req        UpdateApiEndpointRequest
		repoReturn *ApiEndpointDB
		repoErr    error
		wantErr    bool
		wantPath   string
		wantMethod string
	}{
		{
			name: "success path+method updated and read back",
			req: UpdateApiEndpointRequest{
				Path:   strPtr("/api/v1/new-path"),
				Method: strPtr("PUT"),
				Name:   strPtr("新名称"),
			},
			repoReturn: &ApiEndpointDB{
				ID:     id,
				Path:   "/api/v1/new-path",
				Method: "PUT",
				Name:   "新名称",
			},
			wantErr:    false,
			wantPath:   "/api/v1/new-path",
			wantMethod: "PUT",
		},
		{
			name: "failure path propagates repo error",
			req: UpdateApiEndpointRequest{
				Path:   strPtr("/api/v1/dup"),
				Method: strPtr("GET"),
			},
			repoReturn: nil,
			repoErr:    errors.New("unique violation"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := NewMockApiEndpointRepository(ctrl)
			// 关键断言：service 收到的 req（含 Path/Method）原样传给 repo.Update。
			repo.EXPECT().
				Update(gomock.Any(), gomock.Eq(id), gomock.Eq(tt.req)).
				Return(tt.repoReturn, tt.repoErr)

			svc := NewApiEndpointService(repo, zap.NewNop())
			got, err := svc.UpdateApiEndpoint(context.Background(), id, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Path != tt.wantPath {
				t.Errorf("回读 Path = %q, want %q", got.Path, tt.wantPath)
			}
			if got.Method != tt.wantMethod {
				t.Errorf("回读 Method = %q, want %q", got.Method, tt.wantMethod)
			}
		})
	}
}

func TestInferApiGroup(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		// /admin/* 下钻到二级模块
		{"admin users", "/api/v1/admin/users", "users"},
		{"admin user detail", "/api/v1/admin/users/abc", "users"},
		{"admin roles nested", "/api/v1/admin/roles/abc/menus", "roles"},
		{"admin sysConfig", "/api/v1/admin/sysConfig", "sysConfig"},
		{"admin sysDictionary", "/api/v1/admin/sysDictionary/list", "sysDictionary"},
		{"admin api-endpoints", "/api/v1/admin/api-endpoints", "api-endpoints"},
		{"admin audit-logs", "/api/v1/admin/audit-logs", "audit-logs"},
		{"admin logs login", "/api/v1/admin/logs/login", "logs"},
		{"admin permissions", "/api/v1/admin/permissions", "permissions"},
		{"admin menus", "/api/v1/admin/menus", "menus"},

		// 非 /admin/* 取第一段
		{"auth login", "/api/v1/auth/login", "auth"},
		{"devices flat", "/api/v1/devices", "devices"},
		{"devices detail", "/api/v1/devices/123", "devices"},
		{"device-groups tree", "/api/v1/device-groups/tree", "device-groups"},
		{"topology", "/api/v1/topology/nodes", "topology"},
		{"alarms", "/api/v1/alarms/active", "alarms"},
		{"pm files", "/api/v1/pm/files", "pm"},

		// 边界
		{"admin without sub", "/api/v1/admin", "admin"},
		{"admin trailing slash", "/api/v1/admin/", "admin"},
		{"v2 prefix", "/api/v2/admin/users", "users"},
		{"empty", "", ""},
		{"only api/v1", "/api/v1", ""},
		{"only api/v1 trailing", "/api/v1/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inferApiGroup(tt.path); got != tt.want {
				t.Errorf("inferApiGroup(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestInferRouteMetadata(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		path            string
		wantName        string
		wantDescription string
	}{
		{
			name:            "list users",
			method:          "GET",
			path:            "/api/v1/admin/users",
			wantName:        "用户 - 列表",
			wantDescription: "用户：按筛选条件查询资源列表",
		},
		{
			name:            "update role detail",
			method:          "PUT",
			path:            "/api/v1/admin/roles/:id",
			wantName:        "角色 - 更新",
			wantDescription: "角色：更新指定资源的完整配置",
		},
		{
			name:            "sync api endpoints",
			method:          "POST",
			path:            "/api/v1/admin/api-endpoints/sync",
			wantName:        "API 端点 - 同步",
			wantDescription: "API 端点：从运行态或外部系统同步最新数据",
		},
		{
			name:            "login",
			method:          "POST",
			path:            "/api/v1/auth/login",
			wantName:        "认证 - 登录",
			wantDescription: "认证：提交账号凭据并获取访问令牌",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inferRouteName(tt.method, tt.path); got != tt.wantName {
				t.Fatalf("inferRouteName(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.wantName)
			}
			if got := inferRouteDescription(tt.method, tt.path); got != tt.wantDescription {
				t.Fatalf("inferRouteDescription(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.wantDescription)
			}
		})
	}
}

func TestSyncApiEndpointsPassesReadableDescription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMockApiEndpointRepository(ctrl)
	repo.EXPECT().
		Upsert(
			gomock.Any(),
			"/api/v1/admin/api-endpoints/sync",
			"POST",
			"API 端点 - 同步",
			"API 端点：从运行态或外部系统同步最新数据",
			"api-endpoints",
		).
		Return(true, nil)

	svc := NewApiEndpointService(repo, zap.NewNop())
	result, err := svc.SyncApiEndpoints(context.Background(), []gin.RouteInfo{{
		Method: "POST",
		Path:   "/api/v1/admin/api-endpoints/sync",
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created != 1 || result.Updated != 0 || result.Total != 1 {
		t.Fatalf("unexpected sync result: %+v", result)
	}
}

func TestSyncApiEndpointsUsesBatchUpsertAndDeduplicatesRoutes(t *testing.T) {
	repo := &recordingBatchApiEndpointRepository{created: 2}
	svc := NewApiEndpointService(repo, zap.NewNop())

	result, err := svc.SyncApiEndpoints(context.Background(), gin.RoutesInfo{
		{Method: "GET", Path: "/api/v1/devices"},
		{Method: "GET", Path: "/api/v1/devices"},
		{Method: "POST", Path: "/api/v1/devices"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.inputs) != 2 {
		t.Fatalf("batch inputs = %d, want 2", len(repo.inputs))
	}
	if result.Total != 3 || result.Created != 2 || result.Updated != 1 {
		t.Fatalf("unexpected sync result: %+v", result)
	}
}

func TestSyncApiEndpointsReconcilesBuiltInPermissions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMockApiEndpointRepository(ctrl)
	repo.EXPECT().
		Upsert(
			gomock.Any(),
			"/api/v1/admin/sysConfig/apply-batches/:id",
			"GET",
			gomock.Any(),
			gomock.Any(),
			"sysConfig",
		).
		Return(true, nil)

	reconciler := &recordingBuiltInPermissionReconciler{}
	svc := NewApiEndpointService(repo, zap.NewNop())
	svc.SetBuiltInPermissionReconciler(reconciler)

	_, err := svc.SyncApiEndpoints(context.Background(), []gin.RouteInfo{{
		Method: "GET",
		Path:   "/api/v1/admin/sysConfig/apply-batches/:id",
	}})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reconciler.calls != 1 {
		t.Fatalf("reconcile calls = %d, want 1", reconciler.calls)
	}
}

func TestSyncApiEndpointsReturnsBuiltInPermissionReconcileError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMockApiEndpointRepository(ctrl)
	repo.EXPECT().
		Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(false, nil)

	reconciler := &recordingBuiltInPermissionReconciler{err: errors.New("database unavailable")}
	svc := NewApiEndpointService(repo, zap.NewNop())
	svc.SetBuiltInPermissionReconciler(reconciler)

	_, err := svc.SyncApiEndpoints(context.Background(), []gin.RouteInfo{{
		Method: "GET",
		Path:   "/api/v1/admin/sysConfig",
	}})

	if err == nil {
		t.Fatal("expected reconcile error, got nil")
	}
	if !errors.Is(err, reconciler.err) {
		t.Fatalf("error = %v, want wrapped %v", err, reconciler.err)
	}
}

func TestSyncApiEndpointsReturnsRouteUpsertError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoErr := errors.New("database unavailable")
	repo := NewMockApiEndpointRepository(ctrl)
	repo.EXPECT().
		Upsert(gomock.Any(), "/api/v1/admin/new-route", "GET", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(false, repoErr)

	reconciler := &recordingBuiltInPermissionReconciler{}
	svc := NewApiEndpointService(repo, zap.NewNop())
	svc.SetBuiltInPermissionReconciler(reconciler)

	_, err := svc.SyncApiEndpoints(context.Background(), []gin.RouteInfo{{
		Method: "GET",
		Path:   "/api/v1/admin/new-route",
	}})

	if !errors.Is(err, repoErr) {
		t.Fatalf("error = %v, want wrapped %v", err, repoErr)
	}
	if reconciler.calls != 0 {
		t.Fatalf("reconcile calls = %d, want 0 after route persistence failure", reconciler.calls)
	}
}

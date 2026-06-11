package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

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

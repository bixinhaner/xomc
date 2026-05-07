package admin

import "testing"

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

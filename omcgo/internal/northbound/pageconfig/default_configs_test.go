package pageconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultAPIConfigsCoverLegacyNorthboundOverlap(t *testing.T) {
	configs := defaultAPIConfigs()
	byKey := make(map[string]APIConfig, len(configs))
	for _, config := range configs {
		byKey[config.Key] = config
	}

	for _, key := range []string{
		"auth-login",
		"api-user-list",
		"api-user-create",
		"api-user-update",
		"api-user-delete",
		"api-log-page",
		"api-log-export",
		"nb-sync-full-device",
		"nb-export-config",
		"device-list",
		"device-status",
		"device-detail",
		"device-register-create",
		"device-register-list",
		"device-register-delete",
		"device-group-tree",
		"device-group-create",
		"device-group-update",
		"device-group-delete",
		"device-group-add-devices",
		"device-group-sub-create",
		"device-group-sub-update",
		"device-group-sub-delete",
		"parameter-tree",
		"parameter-set",
		"parameter-cellname",
		"config-pull",
		"task-detail",
		"task-page",
		"device-task-create",
		"device-reboot",
		"device-reset",
		"device-log-collect",
	} {
		config, ok := byKey[key]
		require.True(t, ok, "missing northbound API config %s", key)
		require.True(t, config.OldSystemSupported, "legacy overlap must stay visible: %s", key)
		require.True(t, config.CurrentSupported, "legacy overlap must stay current-supported: %s", key)
		require.NotEmpty(t, config.Path)
	}

	for key, path := range map[string]string{
		"auth-login":               "/api/v1/northbound/v1/access/token",
		"api-user-list":            "/api/v1/northbound/v1/user/users",
		"api-user-create":          "/api/v1/northbound/v1/user",
		"api-user-update":          "/api/v1/northbound/v1/user/update",
		"api-user-delete":          "/api/v1/northbound/v1/user/{id}",
		"api-log-page":             "/api/v1/northbound/v1/log/page",
		"api-log-export":           "/api/v1/northbound/v1/log/exportLogToCsvFile",
		"nb-sync-full-device":      "/api/v1/northbound/v1/sync/full?data_type=device&format=json",
		"nb-export-config":         "/api/v1/northbound/v1/export/config/{deviceId}",
		"device-list":              "/api/v1/northbound/v1/device/query",
		"device-status":            "/api/v1/northbound/v1/device/status/{sn}",
		"device-detail":            "/api/v1/northbound/v1/device/infos/{sn}",
		"device-register-create":   "/api/v1/northbound/v1/device/register",
		"device-register-list":     "/api/v1/northbound/v1/device/register/page",
		"device-register-delete":   "/api/v1/northbound/v1/device/register/{id}",
		"device-group-tree":        "/api/v1/northbound/v1/device/group",
		"device-group-create":      "/api/v1/northbound/v1/device/group",
		"device-group-update":      "/api/v1/northbound/v1/device/group/{id}",
		"device-group-delete":      "/api/v1/northbound/v1/device/group/{id}",
		"device-group-add-devices": "/api/v1/northbound/v1/device/group/{id}/devices",
		"device-group-sub-create":  "/api/v1/northbound/v1/device/group/sub",
		"device-group-sub-update":  "/api/v1/northbound/v1/device/group/sub",
		"device-group-sub-delete":  "/api/v1/northbound/v1/device/group/sub",
		"parameter-tree":           "/api/v1/northbound/v1/device/parameters/{sn}",
		"parameter-set":            "/api/v1/northbound/v1/device/parameters/{sn}",
		"parameter-cellname":       "/api/v1/northbound/v1/device/parameters/cellname/{sn}",
		"config-pull":              "/api/v1/northbound/v1/device/parameters/query/{sn}",
		"task-detail":              "/api/v1/northbound/v1/job/result/{jobId}",
		"task-page":                "/api/v1/northbound/v1/job/result/page",
		"device-task-create":       "/api/v1/northbound/v1/device/task/{sn}",
		"device-reboot":            "/api/v1/northbound/v1/device/reboot/{sn}",
		"device-reset":             "/api/v1/northbound/v1/device/reset/{sn}",
		"device-log-collect":       "/api/v1/northbound/v1/device/log/collect/{sn}",
	} {
		require.Equal(t, path, byKey[key].Path, "northbound API should expose legacy-shaped path for %s", key)
	}
}

func TestDefaultSNMPV2TargetUsesFieldCommunity(t *testing.T) {
	targets := defaultSNMPAlarmTargets()
	require.NotEmpty(t, targets)
	require.Equal(t, "snmp-v2-primary", targets[0].Key)
	require.Equal(t, defaultSNMPV2Community, targets[0].Community)
	require.True(t, targets[0].MIBQueryEnabled)
}

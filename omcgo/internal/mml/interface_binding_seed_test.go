package mml

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterfaceBindingLSTCommandsAreUpserted(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)

	const startMarker = "WITH canonical(command_code, group_code, logical_zh, logical_en, command_zh, command_en, target_paths) AS ("
	const endMarker = "WITH merged_fields(command_code, standard_path, mml_code, sort_order) AS ("
	start := strings.Index(string(seed), startMarker)
	require.NotEqual(t, -1, start, "interface-binding LST command block must exist")
	endOffset := strings.Index(string(seed)[start:], endMarker)
	require.NotEqual(t, -1, endOffset, "interface-binding LST field block must follow command block")
	block := string(seed)[start : start+endOffset]

	require.Contains(t, block, "INSERT INTO public.mml_commands", "fresh databases must create interface-binding LST commands")
	require.Contains(t, block, "ON CONFLICT (command_code) DO UPDATE", "interface-binding LST command seed must be idempotent")
	require.Contains(t, block, "'GetParameterValues'")
	require.Contains(t, block, "'LST'")
	require.Contains(t, block, "'LST MML350_DEVICE_LAN_HOSTCONFIGMANAGEMENT__IPINTERFACE_NRCU'")
	require.Contains(t, block, "'LST MML350_DEVICE_LAN_HOSTCONFIGMANAGEMENT__IPINTERFACE_NGAPMGMT'")

	const cleanupStartMarker = "-- Consolidate legacy seed dump rows so a fresh deployment starts with one"
	cleanupStart := strings.Index(string(seed), cleanupStartMarker)
	require.NotEqual(t, -1, cleanupStart)
	cleanup := string(seed)[cleanupStart:]
	require.NotContains(t, cleanup, "'LST MML350_DEVICE_LAN_HOSTCONFIGMANAGEMENT__IPINTERFACE_NRCU',")
	require.NotContains(t, cleanup, "'LST MML350_DEVICE_LAN_HOSTCONFIGMANAGEMENT__IPINTERFACE_NGAPMGMT',")
}

func TestAmfsStatusBelongsToNRCellStatusOnly(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	contents := string(seed)

	require.Contains(t, contents,
		"('LST SF_NR_SJ_SUB_04', 'AMFSSTATUS', 'AMF状态', 'Device.Services.FAPService.{i}.AmfsStatus', 2)")
	require.NotContains(t, contents,
		"('LST STD_TRPATH_G08', 'Device.Services.FAPService.{i}.AmfsStatus', 4)")
	require.NotContains(t, contents,
		"('LST STD_TRPATH_G08', 'Device.Services.FAPService.{i}.AmfsStatus', 'AMFS_STATUS', 'AmfsStatus', 4)")
	require.Contains(t, contents,
		"('LST STD_TRPATH_G08', 'Device.Services.FAPService.{i}.AmfsStatus')")
}

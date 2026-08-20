package mml

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssue274SeedRestoresNeighborObjectCommandsAndLabels(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	sql := string(seed)

	assert.Contains(t, sql, "-- issue #274: restore neighbor and inter-frequency MML commands.")
	assert.Contains(t, sql, "('ADD LTE_INTER_FREQ_CARRIER', '添加 LTE邻频参数管理'")
	assert.Contains(t, sql, "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.")
	assert.Contains(t, sql, "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.")
	assert.Contains(t, sql, "COALESCE(NULLIF(sp.description, ''), leaf_name)")
	assert.NotContains(t, issue274CorrectionBlock(sql), "'zh-CN', n.mml_code")
}

func TestIssue274SeedRepairsOtherMultiInstanceCommands(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	block := issue274CorrectionBlock(string(seed))

	assert.Contains(t, block, "('ADD SJ_CONN_EUTRA_CARRIER', 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.EUTRA.Carrier.')")
	assert.Contains(t, block, "('RMV SJ_CONN_NR_INTER_FREQ_CARRIER', 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.InterFreq.Carrier.')")
	assert.Contains(t, block, "('ADD INTERFACE_I_PV4_ADDRESS', 'Device.Ethernet.Interface.{i}.IPv4Address.')")
	assert.Contains(t, block, "('ADD VLAN_INTERFACE_I_PV6_ADDRESS', 'Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.')")
	assert.Contains(t, block, "('ADD PLMN_LIST', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.')")
	assert.Contains(t, block, "('ADD PDCP_INIT_PARAM', 'Device.Services.FAPService.{i}.CellConfig.LTE.VoLTE.PdcpInitParam.')")
	assert.Contains(t, block, "sp.standard_path LIKE '%{i}%'")
	assert.Contains(t, block, "WHERE c.command_code IN ('ADD 5G_CELL', 'RMV 5G_CELL')")
	assert.Contains(t, block, "deprecated_at = NULL")
	assert.Contains(t, block, "'zh-CN', CASE c.operation_type WHEN 'ADD' THEN '新增 NR邻区参数管理' ELSE '删除 NR邻区参数管理' END")
	assert.Contains(t, block, "numeric_5g_neighbor_types(leaf_name, data_type)")
	assert.Contains(t, block, "('GnbIdLength', 'U_INT')")
	assert.Contains(t, block, "('QOFFSET', 'INT')")
	assert.Contains(t, block, "UPDATE public.param_mappings pm")
	assert.NotContains(t, block, "('PLMNID', 'U_INT')")
}

func TestIssue274SeedCopiesWritableFieldsToEveryMultiInstanceAdd(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	block := issue274CorrectionBlock(string(seed))

	assert.Contains(t, block, "-- Every multi-instance ADD must expose the writable fields of its MOD counterpart.")
	assert.Contains(t, block, "mod.command_code = 'MOD ' || substr(add.command_code, 5)")
	assert.Contains(t, block, "source_sf.access_type = 'RW'")
	assert.Contains(t, block, "regexp_count(source_sp.standard_path, '\\{i\\}')")
	assert.Contains(t, block, "regexp_count(add.target_object, '\\{i\\}') + 1")
	assert.Contains(t, block, "regexp_replace(add.target_object, '\\{i\\}\\.', '', 'g')")
	assert.Contains(t, block, "add.operation_type = 'ADD'")
	assert.Contains(t, block, "-- Commands without a MOD counterpart fall back to the standard writable fields.")
	assert.Contains(t, block, "NOT EXISTS (")
	assert.Contains(t, block, "candidate_sp.access = 'READ_WRITE'")
}

func TestIssue274SeedKeepsMODInheritedAddFieldsAuthoritative(t *testing.T) {
	paths := []string{
		"../../migrations/seed/000001_init_seed.sql",
		"../../scripts/mml_apply_config_updates_20260721.sql",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			contents, err := os.ReadFile(path)
			require.NoError(t, err)
			sql := string(contents)

			// BSC overrides many DeviceGSM.Bts.{i} leaves to READ_WRITE while the shared
			// standard model keeps them READ_ONLY. ADD must inherit that effective set
			// from MOD, and no later global-access cleanup may remove those bindings.
			assert.Contains(t, sql, "mod.command_code = 'MOD ' || substr(add.command_code, 5)")
			assert.Contains(t, sql, "source_sf.access_type = 'RW'")
			assert.Contains(t, sql, "NOT EXISTS (")
			assert.NotRegexp(t, `(?m)^\s*sp\.access <> 'READ_WRITE'`, sql)
		})
	}
}

func TestIssue274SeedRepairsX2AndMMEAddObjectFamilies(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	block := issue274CorrectionBlock(string(seed))

	assert.Contains(t, block, "-- Repair X2 and MME multi-instance command families before generic ADD field inheritance.")
	assert.Contains(t, block, "Device.Services.FAPService.{i}.FAPControl.X2IpAddrMapInfo.")
	assert.Contains(t, block, "Device.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.")
	assert.Contains(t, block, "'LST X2_IP_ADDR_MAP_INFO', 'MOD X2_IP_ADDR_MAP_INFO',")
	assert.Contains(t, block, "x.operation_type = 'LST' OR canonical_sp.access = 'READ_WRITE'")
	assert.Contains(t, block, "mme_sp.access <> 'READ_WRITE'")
}

func TestIssue274SeedMakes5GCellSSBOptionalForAdd(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	block := issue274CorrectionBlock(string(seed))

	assert.Contains(t, block, "sp.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.{i}.SSB'")
	assert.Contains(t, block, "c.command_code = 'ADD 5G_CELL'")
	assert.Contains(t, block, "SET is_required = false")
}

func TestSeedDisablesMRObjectCommandsForEveryBaseStation(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	sql := string(seed)

	assert.Contains(t, sql, "WHERE command_code IN ('ADD MR_MGMT_CONFIG', 'RMV MR_MGMT_CONFIG')")
	assert.NotContains(t, issue274CorrectionBlock(sql), "'ADD MR_MGMT_CONFIG',\n            '添加 MR参数管理'")
}

func TestIssue274SeedAddsNRDeviceLTENeighborObjectCommands(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	block := issue274CorrectionBlock(string(seed))

	assert.Contains(t, block, "('ADD NR_LTE_CELL', 'ADD', 'AddObject'")
	assert.Contains(t, block, "('RMV NR_LTE_CELL', 'RMV', 'DeleteObject'")
	assert.Contains(t, block, "Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.NeighborList.LTECell.")
	assert.Contains(t, block, "dst.command_code IN ('ADD NR_LTE_CELL', 'RMV NR_LTE_CELL')")
}

func TestKeepalivedSeedRestoresPermissionDrivenCommandSet(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	block := keepalivedCorrectionBlock(string(seed))

	assert.Contains(t, block, "('MOD MML350_DEVICE_KEEPALIVEDMGMT__KEEPALIVEDMGMT', '修改热备配置'")
	assert.Contains(t, block, "('MOD MML350_DEVICE_KEEPALIVEDMGMT__VRRPMGMT', '修改VRRP实例'")
	assert.Contains(t, block, "('ADD MML350_DEVICE_KEEPALIVEDMGMT__VRRPMGMT', '添加VRRP实例'")
	assert.Contains(t, block, "('RMV MML350_DEVICE_KEEPALIVEDMGMT__VRRPMGMT', '删除VRRP实例'")
	assert.Contains(t, block, "'Device.KeepalivedMgmt.VrrpMgmt.{i}.', 'object', 'READ_WRITE'")
	assert.Contains(t, block, "INSERT INTO public.param_mappings")
	assert.Contains(t, block, "WHERE pm.name = 'BSC'")
	assert.NotContains(t, block, "ADD MML350_DEVICE_KEEPALIVEDMGMT__KEEPALIVEDMGMT")
	assert.NotContains(t, block, "RMV MML350_DEVICE_KEEPALIVEDMGMT__KEEPALIVEDMGMT")
	assert.Contains(t, block, "AND bpm.access = 'READ_WRITE'")
	assert.Contains(t, block, "regexp_count(fp.standard_path, '\\{i\\}') = 1")
	assert.Contains(t, block, "UPDATE public.param_mappings mapping")
	assert.Contains(t, block, "'Device.KeepalivedMgmt.VrrpMgmt.{i}.DstIpAddr', 'READ_ONLY', 'STRING'")
	assert.Contains(t, block, "'Device.KeepalivedMgmt.VrrpMgmt.{i}.SrcIpAddr', 'READ_ONLY', 'STRING'")
	assert.Contains(t, block, "'Device.KeepalivedMgmt.VrrpMgmt.{i}.VirtualIpList.{i}.IP', 'READ_ONLY', 'STRING'")
	assert.Contains(t, block, "'Device.KeepalivedMgmt.MaxVrrpEntries', 'READ_ONLY', 'U_INT'")
	assert.Contains(t, block, "'Device.KeepalivedMgmt.VrrpMgmt.{i}.MaxVirtualIpEntries', 'READ_ONLY', 'U_INT'")
	assert.Contains(t, block, "'Device.KeepalivedMgmt.VrrpMgmt.{i}.VirtualIpNumberOfEntries', 'READ_ONLY', 'U_INT'")
	assert.Contains(t, block, "'Device.KeepalivedMgmt.VrrpMgmt.'")
	assert.Contains(t, block, "WHEN c.operation_type IN ('ADD', 'RMV') AND COALESCE(c.target_object, '') <> ''")
}

func issue274CorrectionBlock(sql string) string {
	start := strings.Index(sql, "-- issue #274:")
	if start < 0 {
		return ""
	}
	return sql[start:]
}

func keepalivedCorrectionBlock(sql string) string {
	start := strings.Index(sql, "-- Keepalived/VRRP: restore HA write commands")
	if start < 0 {
		return ""
	}
	return sql[start:]
}

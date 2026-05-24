package specparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeCatalog 构造一个最小 SpecCatalog 用于 differ 单测。
func makeCatalog() *SpecCatalog {
	return &SpecCatalog{
		Version: "test",
		Groups: []*SpecGroup{
			{
				Chapter:       "SA",
				GroupCode:     "Device.DeviceInfo.*",
				CommandZhName: "设备基本信息",
				HasInstance:   false,
				Paths: []*SpecPath{
					{StandardPath: "Device.DeviceInfo.UserLabel", ParamName: "UserLabel", ChineseName: "用户友好名", Access: AccessReadWrite, DataType: "string"},
					{StandardPath: "Device.DeviceInfo.UpTime", ParamName: "UpTime", ChineseName: "运行时间", Access: AccessReadOnly, DataType: "unsignedInt"},
				},
			},
			{
				Chapter:       "SF",
				GroupCode:     "Device.X2.{i}.*",
				CommandZhName: "X2 接口",
				HasInstance:   true,
				Paths: []*SpecPath{
					{StandardPath: "Device.X2.{i}.Id", ParamName: "Id", ChineseName: "标识", Access: AccessReadWrite, DataType: "string"},
				},
			},
		},
	}
}

func TestDeriveCommands_LSTRWMod(t *testing.T) {
	cat := makeCatalog()
	cmds := DeriveCommands(cat)
	// SA: LST + MOD（UserLabel 是 RW）/ SF: LST + MOD + ADD + RMV (HasInstance + 非 blacklist)
	require.Len(t, cmds, 6)

	// SA 派生
	assert.Equal(t, "LST DEVICE_INFO", cmds[0].CommandCode)
	assert.Equal(t, OpLST, cmds[0].OperationType)
	assert.Len(t, cmds[0].TargetPaths, 2)

	assert.Equal(t, "MOD DEVICE_INFO", cmds[1].CommandCode)
	assert.Len(t, cmds[1].TargetPaths, 1, "MOD 只含 RW path")
	assert.Equal(t, "Device.DeviceInfo.UserLabel", cmds[1].TargetPaths[0])
}

func TestDeriveCommands_ADDRMVOnInstance(t *testing.T) {
	cat := makeCatalog()
	cmds := DeriveCommands(cat)
	// SF group 派生 4 个 op
	var addCmd, rmvCmd *SpecCommand
	for _, c := range cmds {
		if c.CommandCode == "ADD X2" {
			addCmd = c
		}
		if c.CommandCode == "RMV X2" {
			rmvCmd = c
		}
	}
	require.NotNil(t, addCmd)
	require.NotNil(t, rmvCmd)
	assert.Equal(t, RPCAddObject, addCmd.RPCMethod)
	assert.Equal(t, RPCDeleteObject, rmvCmd.RPCMethod)
	assert.Equal(t, "Device.X2.", addCmd.TargetObject)
}

func TestDeriveCommands_BlacklistBlocksADD(t *testing.T) {
	cat := makeCatalog()
	cat.Blacklist = []string{"Device.X2.{i}.*"}
	cmds := DeriveCommands(cat)
	for _, c := range cmds {
		if strings.HasPrefix(c.CommandCode, "ADD ") || strings.HasPrefix(c.CommandCode, "RMV ") {
			assert.NotEqual(t, "Device.X2.{i}.*", c.GroupCode, "Blacklist 命中 group 不应派生 ADD/RMV")
		}
	}
}

func TestDiff_AllNewWhenEmptyDB(t *testing.T) {
	cat := makeCatalog()
	empty := &DBSnapshot{
		StandardParams:   map[string]*StandardParamRow{},
		Commands:         map[string]*MMLCommandRow{},
		CommandSubFields: map[string]map[string]*SubFieldRow{},
	}
	rep := Diff(cat, empty)
	assert.Equal(t, 3, rep.Summary.NewStandardParams, "SA 2 path + SF 1 path")
	assert.Equal(t, 6, rep.Summary.NewCommands, "SA 2 op + SF 4 op")
	// SubFieldLinks 只对 LST/MOD：SA LST(2) + SA MOD(1) + SF LST(1) + SF MOD(1) = 5
	assert.Equal(t, 5, rep.Summary.NewSubFieldLinks)
	assert.Equal(t, 0, rep.Summary.OrphanCommands)
}

func TestDiff_OrphanDetection(t *testing.T) {
	cat := makeCatalog()
	snap := &DBSnapshot{
		StandardParams:   map[string]*StandardParamRow{},
		Commands: map[string]*MMLCommandRow{
			"LST OBSOLETE_CMD": {CommandCode: "LST OBSOLETE_CMD", OperationType: OpLST, Source: SourceStandard},
		},
		CommandSubFields: map[string]map[string]*SubFieldRow{},
	}
	rep := Diff(cat, snap)
	assert.Contains(t, rep.OrphanCommands, "LST OBSOLETE_CMD")
}

func TestGroupCodeToLogicalCode(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Device.DeviceInfo.*", "DEVICE_INFO"},
		{"Device.DeviceInfo.SwUpgrade.*", "DEVICE_INFO_SW_UPGRADE"},
		{"Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*", "X2_IP_ADDR_MAP_INFO"},
		{"Device.FAP.GPS.*", "FAP_GPS"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, groupCodeToLogicalCode(tt.in))
		})
	}
}

func TestCamelToSnakeUpper(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"UserLabel", "USER_LABEL"},
		{"ManufacturerOUI", "MANUFACTURER_OUI"},
		{"3GPPSpecVersion", "3GPP_SPEC_VERSION"},
		{"IpAddr", "IP_ADDR"},
		{"X2", "X2"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, camelToSnakeUpper(tt.in))
		})
	}
}

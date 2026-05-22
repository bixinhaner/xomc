package catalogloader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseV2_ProductionFile 用真实 catalog 文件跑端到端，
// 校验所有 v2 schema 校验通过、统计字段与 §R-2.4 命令中文名表一致。
func TestParseV2_ProductionFile(t *testing.T) {
	cat, err := ParseFile("../../../datamodels/mml-catalog/cmcc-tdlte-v2.3.json")
	require.NoError(t, err, "real v2 catalog should parse cleanly")

	assert.Equal(t, "cmcc-tdlte-v2.3", cat.SpecVersion)
	assert.Equal(t, "v2", cat.SchemaVersion)
	assert.Equal(t, "cmcc", cat.Carrier)
	assert.Equal(t, "lte", cat.Tech)

	// §R-1: 18 chapter groups
	assert.Len(t, cat.Groups, 18, "expected 18 SA-SR chapters")
	for i, g := range cat.Groups {
		assert.Truef(t, len(g.GroupCode) > len("chapter:") && g.GroupCode[:8] == "chapter:",
			"groups[%d].groupCode %q must start with 'chapter:'", i, g.GroupCode)
		assert.NotEmptyf(t, g.NameI18n["zh-CN"], "groups[%d] (%s) needs zh-CN name", i, g.GroupCode)
		assert.NotEmptyf(t, g.Commands, "groups[%d] (%s) needs commands", i, g.GroupCode)
	}

	// Total op leaves 应与 stats 一致（§R-3 派生）
	if cat.Stats != nil {
		var total int
		for _, g := range cat.Groups {
			total += len(g.Commands)
		}
		assert.Equal(t, cat.Stats.CommandLeavesTotal, total,
			"sum(groups[].commands) should match stats.commandLeavesTotal")
		assert.Equal(t, 18, cat.Stats.Chapters)
	}

	// §R-3.1 合并：v2.3 仅触发 1 条（FAPService.{i}.* SF+SH→SF）
	assert.Len(t, cat.MergeLog, 1, "expected exactly 1 merge event in v2.3")
	if len(cat.MergeLog) == 1 {
		assert.Equal(t, "Device.Services.FAPService.{i}.*", cat.MergeLog[0].GroupCode)
		assert.Equal(t, "SF", cat.MergeLog[0].OwnerChapter)
		assert.Equal(t, "SH", cat.MergeLog[0].MergedChapter)
	}

	// §R-3.2 非可创建：17 个对象被过滤掉 ADD/RMV
	assert.Len(t, cat.NonCreatableHits, 17, "expected 17 non-creatable hits")

	// §R-2.4 命令中文名 zh-CN 全 71 个 standard 命令唯一（跨章节）
	seenZh := make(map[string]string)
	for _, g := range cat.Groups {
		for _, cmd := range g.Commands {
			zh := cmd.LogicalNameI18n["zh-CN"]
			require.NotEmpty(t, zh, "command %s missing zh-CN", cmd.CommandCode)
			// 同 zh-CN 在 LST/MOD/ADD/RMV 之间共享是允许的，只要 groupCodeObject 相同
			if prev, ok := seenZh[zh]; ok {
				assert.Equal(t, prev, cmd.GroupCodeObject,
					"§R-2.4 跨 op 同 zh-CN 必须指向同一 groupCodeObject; %q saw %s vs %s",
					zh, prev, cmd.GroupCodeObject)
			} else {
				seenZh[zh] = cmd.GroupCodeObject
			}
		}
	}
	assert.Len(t, seenZh, 71, "expected 71 unique command_zh_name (§R-2.4)")

	// Spot-check: SA 设备信息参数管理 should have 3 leaves (LST + MOD 设备基本信息, LST 设备软件升级状态)
	var saGroup *Group
	for i := range cat.Groups {
		if cat.Groups[i].GroupCode == "chapter:SA" {
			saGroup = &cat.Groups[i]
			break
		}
	}
	require.NotNil(t, saGroup, "SA chapter group should exist")
	assert.Equal(t, "设备信息参数管理", saGroup.NameI18n["zh-CN"])
	assert.Len(t, saGroup.Commands, 3)

	// Verify SA's LST 设备基本信息 has 17 treeNodeRefs (matches spec table)
	var lstDevInfo *Command
	for i := range saGroup.Commands {
		if saGroup.Commands[i].CommandCode == "LST:Device.DeviceInfo.*" {
			lstDevInfo = &saGroup.Commands[i]
			break
		}
	}
	require.NotNil(t, lstDevInfo)
	assert.Equal(t, "设备基本信息", lstDevInfo.LogicalNameI18n["zh-CN"])
	assert.Equal(t, "GetParameterValues", lstDevInfo.RPCMethod)
	assert.Len(t, lstDevInfo.TreeNodeRefs, 17)

	// Verify SD's "当前告警实例" carries parsed instanceRangeMeta (dynamic, nSource=MaxCurrentAlarmEntries)
	var lstCurAlarm *Command
	for _, g := range cat.Groups {
		if g.GroupCode != "chapter:SD" {
			continue
		}
		for i := range g.Commands {
			if g.Commands[i].CommandCode == "LST:Device.FaultMgmt.CurrentAlarm.{i}.*" {
				lstCurAlarm = &g.Commands[i]
				break
			}
		}
	}
	require.NotNil(t, lstCurAlarm, "SD's LST CurrentAlarm must exist")
	require.Len(t, lstCurAlarm.InstanceRangeMeta, 1, "single-layer {i} → 1 InstanceRange entry")
	r := lstCurAlarm.InstanceRangeMeta[0]
	assert.Equal(t, 1, r.Layer)
	assert.Equal(t, "0~N", r.RangeExpr)
	assert.True(t, r.Dynamic)
	assert.Equal(t, "MaxCurrentAlarmEntries", r.NSource)

	// Verify SF's merged FAPService.{i}.* command has > 24 paths (24 SF + 10 SH union)
	var lstFAPService *Command
	for _, g := range cat.Groups {
		if g.GroupCode != "chapter:SF" {
			continue
		}
		for i := range g.Commands {
			if g.Commands[i].CommandCode == "LST:Device.Services.FAPService.{i}.*" {
				lstFAPService = &g.Commands[i]
				break
			}
		}
	}
	require.NotNil(t, lstFAPService, "SF's LST FAPService 载波 should exist (merged from SH per §R-3.1)")
	assert.GreaterOrEqual(t, len(lstFAPService.TreeNodeRefs), 30,
		"FAPService.{i}.* merged set should be ~34 paths (SF 24 + SH 10 deduped)")

	// Verify §R-3.2 filter: FAPService.{i}.* should NOT have ADD/RMV commands
	for _, g := range cat.Groups {
		for _, cmd := range g.Commands {
			if cmd.GroupCodeObject == "Device.Services.FAPService.{i}.*" {
				assert.NotContainsf(t, []string{"ADD", "RMV"}, cmd.OperationType,
					"§R-3.2: FAPService.{i}.* is non-creatable, ADD/RMV should be filtered; saw %s",
					cmd.CommandCode)
			}
		}
	}
}

// TestParseV2_RejectMissingTreeNodeRefs 校验 v2 schema 拒绝 treeNodeRefs 为空的命令。
func TestParseV2_RejectMissingTreeNodeRefs(t *testing.T) {
	raw := []byte(`{
		"specVersion": "test-v2.3",
		"schemaVersion": "v2",
		"carrier": "cmcc",
		"tech": "lte",
		"groups": [
			{
				"groupCode": "chapter:SA",
				"chapterCode": "SA",
				"nameI18n": {"zh-CN": "设备信息参数管理"},
				"displayOrder": 1,
				"commands": [
					{
						"commandCode": "LST:Device.DeviceInfo.*",
						"operationType": "LST",
						"rpcMethod": "GetParameterValues",
						"logicalNameI18n": {"zh-CN": "设备基本信息"},
						"groupCodeObject": "Device.DeviceInfo.*",
						"treeNodeRefs": []
					}
				]
			}
		]
	}`)
	_, err := Parse(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty treeNodeRefs")
}

// TestParseV2_RejectNonChapterGroupCode 校验 v2 schema 拒绝非 chapter:* 的 groupCode。
func TestParseV2_RejectNonChapterGroupCode(t *testing.T) {
	raw := []byte(`{
		"specVersion": "test-v2.3",
		"schemaVersion": "v2",
		"carrier": "cmcc",
		"tech": "lte",
		"groups": [
			{
				"groupCode": "Device.DeviceInfo.*",
				"chapterCode": "SA",
				"nameI18n": {"zh-CN": "设备信息"},
				"displayOrder": 1,
				"commands": [
					{
						"commandCode": "LST:Device.DeviceInfo.*",
						"operationType": "LST",
						"logicalNameI18n": {"zh-CN": "设备基本信息"},
						"groupCodeObject": "Device.DeviceInfo.*",
						"treeNodeRefs": ["Device.DeviceInfo.UserLabel"]
					}
				]
			}
		]
	}`)
	_, err := Parse(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must have 'chapter:' prefix")
}

// TestParseV2_RejectDuplicateZhAcrossGroups 校验 §R-2.4 跨章节同名禁令。
func TestParseV2_RejectDuplicateZhAcrossGroups(t *testing.T) {
	raw := []byte(`{
		"specVersion": "test-v2.3",
		"schemaVersion": "v2",
		"carrier": "cmcc",
		"tech": "lte",
		"groups": [
			{
				"groupCode": "chapter:SA",
				"chapterCode": "SA",
				"nameI18n": {"zh-CN": "设备信息参数管理"},
				"displayOrder": 1,
				"commands": [
					{
						"commandCode": "LST:Device.X.*",
						"operationType": "LST",
						"logicalNameI18n": {"zh-CN": "撞名命令"},
						"groupCodeObject": "Device.X.*",
						"treeNodeRefs": ["Device.X.A"]
					}
				]
			},
			{
				"groupCode": "chapter:SB",
				"chapterCode": "SB",
				"nameI18n": {"zh-CN": "软件版本参数管理"},
				"displayOrder": 2,
				"commands": [
					{
						"commandCode": "LST:Device.Y.*",
						"operationType": "LST",
						"logicalNameI18n": {"zh-CN": "撞名命令"},
						"groupCodeObject": "Device.Y.*",
						"treeNodeRefs": ["Device.Y.B"]
					}
				]
			}
		]
	}`)
	_, err := Parse(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "§R-2.4")
	assert.Contains(t, err.Error(), "撞名命令")
}


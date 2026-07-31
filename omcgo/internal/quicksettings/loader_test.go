package quicksettings

import (
	"context"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

func writeXML(t *testing.T, dir, name, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
}

func TestLoader_LoadOnce_HappyPath(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "quicksettings")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	writeXML(t, dir, "BLQ.xml", `<?xml version="1.0" encoding="UTF-8"?>
<quickSettings paramModel="BLQ">
  <group id="enb-cell" titleZh="小区参数" titleEn="Cell Parameters">
    <param name="ECI" titleZh="ECI" titleEn="ECI"
           standardPath="Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity"/>
  </group>
  <group id="enb-nbr" titleZh="邻区" titleEn="Neighbor List"
         multiInstance="true"
         objectPath="Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.">
    <param name="CellID" titleZh="小区ID" titleEn="Cell ID" leaf="CID"/>
  </group>
</quickSettings>`)

	writeXML(t, dir, "BaiBNQ.xml", `<?xml version="1.0" encoding="UTF-8"?>
<quickSettings paramModel="BaiBNQ">
  <group id="gnb-cell" titleZh="小区参数" titleEn="Cell Parameters">
    <param name="PCI" titleZh="PCI" titleEn="PCI"
           standardPath="Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RF.PhyCellID"/>
  </group>
</quickSettings>`)

	reg := NewRegistry()
	loader := NewLoader(appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"}, tmp, reg, nil)

	rep, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, rep.FilesLoaded)
	assert.Equal(t, 0, rep.FilesSkipped)
	assert.Equal(t, 3, rep.RowsAffected) // 2 BLQ groups + 1 BaiBNQ group

	blq := reg.GetByParamModel("BLQ")
	require.Len(t, blq, 2)
	assert.Equal(t, "enb-cell", blq[0].ID)
	assert.False(t, blq[0].MultiInstance)
	assert.Equal(t, "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity", blq[0].Params[0].StandardPath)
	assert.Equal(t, "enb-nbr", blq[1].ID)
	assert.True(t, blq[1].MultiInstance)
	assert.NotEmpty(t, blq[1].ObjectPath)
	assert.Equal(t, "CID", blq[1].Params[0].Leaf)

	nr := reg.GetByParamModel("BaiBNQ")
	require.Len(t, nr, 1)
	assert.Equal(t, "gnb-cell", nr[0].ID)
	assert.False(t, nr[0].MultiInstance)

	known := reg.KnownParamModels()
	assert.ElementsMatch(t, []string{"BLQ", "BaiBNQ"}, known)
}

func TestLoader_EmptyDirectory_DoesNotFail(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "quicksettings")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	reg := NewRegistry()
	loader := NewLoader(appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"}, tmp, reg, nil)
	rep, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, rep.FilesLoaded)
	assert.Equal(t, 0, len(reg.KnownParamModels()))
}

func TestLoader_DirectoryNotFound_DoesNotFail(t *testing.T) {
	tmp := t.TempDir()
	// 不创建 quicksettings 子目录
	reg := NewRegistry()
	loader := NewLoader(appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"}, tmp, reg, nil)
	rep, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, rep.FilesLoaded)
	assert.NotEmpty(t, rep.Errors) // 目录不存在 记为 non-fatal
}

func TestLoader_BadXML_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "quicksettings")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	writeXML(t, dir, "BLQ.xml", `<quickSettings><bad-xml`)

	reg := NewRegistry()
	loader := NewLoader(appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"}, tmp, reg, nil)
	_, err := loader.LoadOnce(context.Background())
	assert.Error(t, err)
}

func TestLoader_FilenameParamModelMismatch_Rejects(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "quicksettings")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	// 文件名是 BLQ.xml 但 XML 属性 paramModel="BM" — 必须拒绝
	writeXML(t, dir, "BLQ.xml", `<?xml version="1.0"?>
<quickSettings paramModel="BM">
  <group id="x" titleZh="x" titleEn="x">
    <param name="x" titleZh="x" titleEn="x" standardPath="Device.X"/>
  </group>
</quickSettings>`)

	reg := NewRegistry()
	loader := NewLoader(appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"}, tmp, reg, nil)
	_, err := loader.LoadOnce(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mismatches file name")
}

func TestLoader_ValidationCatches_MissingLeafForMultiInstance(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "quicksettings")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	writeXML(t, dir, "BLQ.xml", `<?xml version="1.0" encoding="UTF-8"?>
<quickSettings paramModel="BLQ">
  <group id="enb-nbr" titleZh="邻区" titleEn="Neighbor"
         multiInstance="true"
         objectPath="Device.X.{i}.">
    <param name="CellID" titleZh="小区ID" titleEn="Cell ID"/>
  </group>
</quickSettings>`)

	reg := NewRegistry()
	loader := NewLoader(appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"}, tmp, reg, nil)
	_, err := loader.LoadOnce(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing leaf or standardPath")
}

// TestLoader_MultiInstance_StandardPathDerivesLeaf 验证多实例 group 可以
// 只写 standardPath，loader 自动按 objectPath 前缀反推 Leaf。
func TestLoader_MultiInstance_StandardPathDerivesLeaf(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "quicksettings")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	writeXML(t, dir, "BSC.xml", `<?xml version="1.0" encoding="UTF-8"?>
<quickSettings paramModel="BSC">
  <group id="bsc-bts-access" titleZh="BTS 接入" titleEn="BTS Access"
         multiInstance="true" objectPath="DeviceGSM.Bts.{i}.">
    <param name="OmlIpaStreamId" titleZh="IPA" titleEn="IPA"
           standardPath="DeviceGSM.Bts.{i}.OmlIpaStreamId"/>
  </group>
</quickSettings>`)

	reg := NewRegistry()
	loader := NewLoader(appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"}, tmp, reg, nil)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)
	bsc := reg.GetByParamModel("BSC")
	require.Len(t, bsc, 1)
	require.Len(t, bsc[0].Params, 1)
	assert.Equal(t, "DeviceGSM.Bts.{i}.OmlIpaStreamId", bsc[0].Params[0].StandardPath)
	assert.Equal(t, "OmlIpaStreamId", bsc[0].Params[0].Leaf, "loader 应按 objectPath 前缀反推 leaf")
}

// TestLoader_MultiInstance_StandardPathPrefixMismatch_Rejects 验证 standardPath
// 与所在 group 的 objectPath 不一致时 loader 拒绝加载。
func TestLoader_MultiInstance_StandardPathPrefixMismatch_Rejects(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "quicksettings")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	writeXML(t, dir, "BSC.xml", `<?xml version="1.0" encoding="UTF-8"?>
<quickSettings paramModel="BSC">
  <group id="g" titleZh="g" titleEn="g"
         multiInstance="true" objectPath="DeviceGSM.Bts.{i}.">
    <param name="x" titleZh="x" titleEn="x" standardPath="Device.Other.X"/>
  </group>
</quickSettings>`)

	reg := NewRegistry()
	loader := NewLoader(appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"}, tmp, reg, nil)
	_, err := loader.LoadOnce(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must start with group objectPath")
}

func TestBuiltinBSC_HandoverOptionsUseDeviceValues(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "data", "quicksettings", "BSC.xml"))
	require.NoError(t, err)

	var doc xmlQuickSettings
	require.NoError(t, xml.Unmarshal(data, &doc))

	var handover *xmlParam
	for gi := range doc.Groups {
		for pi := range doc.Groups[gi].Params {
			p := &doc.Groups[gi].Params[pi]
			if p.StandardPath == "DeviceGSM.Bts.{i}.handover" {
				handover = p
				break
			}
		}
	}
	require.NotNil(t, handover)
	require.Len(t, handover.EnumOptions, 2)
	assert.Equal(t, "1", handover.EnumOptions[0].Value)
	assert.Equal(t, "Allow", handover.EnumOptions[0].Label)
	assert.Equal(t, "0", handover.EnumOptions[1].Value)
	assert.Equal(t, "Forbid", handover.EnumOptions[1].Label)
}

func TestBuiltinBaiBNQ_IncludesStringSyncSourceSettings(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "data", "quicksettings", "BaiBNQ.xml"))
	require.NoError(t, err)

	var doc xmlQuickSettings
	require.NoError(t, xml.Unmarshal(data, &doc))

	var syncGroup *xmlGroup
	for gi := range doc.Groups {
		if doc.Groups[gi].ID == "gnb-sync-source" {
			syncGroup = &doc.Groups[gi]
			break
		}
	}
	require.NotNil(t, syncGroup)
	require.Len(t, syncGroup.Params, 11)

	mode := syncGroup.Params[0]
	assert.Equal(t, "PpsTimeMode", mode.Name)
	assert.Equal(t, "string", mode.Type)
	assert.Equal(t, "Device.FAP.Synchronization.PpsTimeMode", mode.StandardPath)
	assert.Equal(t, []xmlEnumOption{
		{Value: "FREE_OSCILLATION", Label: "FREE_OSCILLATION"},
		{Value: "GPS_PPS", Label: "GPS_PPS"},
		{Value: "LOCAL_CLOCK_HOLDOVER_GPS_PPS", Label: "LOCAL_CLOCK_HOLDOVER_GPS_PPS"},
		{Value: "OCXO_PPS", Label: "OCXO_PPS"},
		{Value: "1588_PPS", Label: "1588_PPS"},
		{Value: "GPS_AND_PTP", Label: "GPS_AND_PTP"},
	}, mode.EnumOptions)

	source := syncGroup.Params[1]
	assert.Equal(t, "SyncSource", source.Name)
	assert.Equal(t, "string", source.Type)
	assert.Equal(t, "Device.FAP.GPS.SyncSource", source.StandardPath)
	assert.Equal(t, []xmlEnumOption{
		{Value: "GPS", Label: "GPS"},
		{Value: "GLONASS", Label: "GLONASS"},
		{Value: "BEIDOU", Label: "BEIDOU"},
		{Value: "GALILEO", Label: "GALILEO"},
		{Value: "QZSS", Label: "QZSS"},
	}, source.EnumOptions)

	forcedSync := syncGroup.Params[2]
	assert.Equal(t, "ForcedSync", forcedSync.Name)
	assert.Equal(t, "unsignedInt", forcedSync.Type)
	assert.Equal(t, "Device.DeviceInfo.iForcedSyncControlSwitch", forcedSync.StandardPath)
	assert.Equal(t, "1", forcedSync.DefaultValue)
	assert.Equal(t, "true", forcedSync.HideRangeHint)
	assert.Equal(t, []xmlEnumOption{
		{Value: "1", Label: "1"},
		{Value: "0", Label: "0"},
	}, forcedSync.EnumOptions)

	ptpStatus := syncGroup.Params[3]
	assert.Equal(t, "PTPSyncStatus", ptpStatus.Name)
	assert.Equal(t, "string", ptpStatus.Type)
	assert.Equal(t, "Device.FAP.PTP1588.SyncStatus", ptpStatus.StandardPath)
	assert.Equal(t, "true", ptpStatus.Readonly)

	ptpProfile := syncGroup.Params[4]
	assert.Equal(t, "PTPProfile", ptpProfile.Name)
	assert.Equal(t, "Device.FAP.PTP1588.Profile", ptpProfile.StandardPath)
	assert.Equal(t, []xmlEnumOption{{Value: "1588v2", Label: "1588v2"}}, ptpProfile.EnumOptions)

	ptpDelayInterval := syncGroup.Params[10]
	assert.Equal(t, "PTPDelayInterval", ptpDelayInterval.Name)
	assert.Equal(t, "Device.FAP.PTP1588.DelayInterval", ptpDelayInterval.StandardPath)
	assert.Contains(t, ptpDelayInterval.EnumOptions, xmlEnumOption{Value: "-4", Label: "-4"})
}

func TestBuiltinMLN_IncludesIndependentPLMNList(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "data", "quicksettings", "MLN.xml"))
	require.NoError(t, err)

	var doc xmlQuickSettings
	require.NoError(t, xml.Unmarshal(data, &doc))

	var plmnGroup *xmlGroup
	for i := range doc.Groups {
		if doc.Groups[i].ID == "enb-plmn" {
			plmnGroup = &doc.Groups[i]
			break
		}
	}
	require.NotNil(t, plmnGroup)
	require.False(t, plmnGroup.MultiInstance == "true")
	require.Len(t, plmnGroup.Params, 1)
	assert.Equal(t, "ExistPlmnidList", plmnGroup.Params[0].Name)
	assert.Equal(t,
		"Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.ExistPlmnidList",
		plmnGroup.Params[0].StandardPath,
	)
	assert.Equal(t, "6", plmnGroup.Params[0].MaxValue)
}

func TestBuiltinBM_IncludesIndependentPLMNList(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "data", "quicksettings", "BM.xml"))
	require.NoError(t, err)

	var doc xmlQuickSettings
	require.NoError(t, xml.Unmarshal(data, &doc))

	var plmnGroup *xmlGroup
	for i := range doc.Groups {
		if doc.Groups[i].ID == "enb-plmn" {
			plmnGroup = &doc.Groups[i]
			break
		}
	}
	require.NotNil(t, plmnGroup)
	require.False(t, plmnGroup.MultiInstance == "true")
	require.Len(t, plmnGroup.Params, 1)
	assert.Equal(t, "ExistPlmnidList", plmnGroup.Params[0].Name)
	assert.Equal(t,
		"Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.ExistPlmnidList",
		plmnGroup.Params[0].StandardPath,
	)
	assert.Equal(t, "6", plmnGroup.Params[0].MaxValue)
}

func TestBuiltinBLQ_IncludesIndependentPLMNList(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "data", "quicksettings", "BLQ.xml"))
	require.NoError(t, err)

	var doc xmlQuickSettings
	require.NoError(t, xml.Unmarshal(data, &doc))

	var plmnGroup *xmlGroup
	for i := range doc.Groups {
		if doc.Groups[i].ID == "enb-plmn" {
			plmnGroup = &doc.Groups[i]
			break
		}
	}
	require.NotNil(t, plmnGroup)
	assert.Equal(t, "true", plmnGroup.MultiInstance)
	assert.Equal(t, 6, plmnGroup.MaxInstances)
	assert.Equal(t,
		"Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.",
		plmnGroup.ObjectPath,
	)
	require.Len(t, plmnGroup.Params, 1)
	assert.Equal(t, "PLMNID", plmnGroup.Params[0].Name)
	assert.Equal(t, "PLMNID", plmnGroup.Params[0].Leaf)
}

func TestBuiltinMLQ_IncludesIndependentPLMNList(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "data", "quicksettings", "MLQ.xml"))
	require.NoError(t, err)

	var doc xmlQuickSettings
	require.NoError(t, xml.Unmarshal(data, &doc))

	var plmnGroup *xmlGroup
	for i := range doc.Groups {
		if doc.Groups[i].ID == "enb-plmn" {
			plmnGroup = &doc.Groups[i]
			break
		}
	}
	require.NotNil(t, plmnGroup)
	assert.Equal(t, "true", plmnGroup.MultiInstance)
	assert.Equal(t, 6, plmnGroup.MaxInstances)
	assert.Equal(t,
		"Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.",
		plmnGroup.ObjectPath,
	)
	require.Len(t, plmnGroup.Params, 1)
	assert.Equal(t, "PLMNID", plmnGroup.Params[0].Name)
	assert.Equal(t, "PLMNID", plmnGroup.Params[0].Leaf)
}

func TestBuiltinBLN_QuickSettingsReferenceParamModel(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "data", "quicksettings", "BLN.xml"))
	require.NoError(t, err)

	var doc xmlQuickSettings
	require.NoError(t, xml.Unmarshal(data, &doc))
	assert.Equal(t, "BLN", doc.ParamModel)

	groupIDs := make([]string, 0, len(doc.Groups))
	for _, group := range doc.Groups {
		groupIDs = append(groupIDs, group.ID)
	}
	assert.Equal(t, []string{"enb-cell", "enb-plmn", "device-ipsec", "enb-neighbor-freq", "enb-neighbor-cell"}, groupIDs)

	paramModelData, err := os.ReadFile(filepath.Join("..", "..", "data", "param-mappings", "BLN.xml"))
	require.NoError(t, err)

	var paramModel struct {
		Objects []struct {
			StandardPath string `xml:"standardPath,attr"`
		} `xml:"objects>object"`
		Params []struct {
			StandardPath string `xml:"standardPath,attr"`
		} `xml:"parameters>param"`
	}
	require.NoError(t, xml.Unmarshal(paramModelData, &paramModel))

	standardPaths := make(map[string]struct{}, len(paramModel.Objects)+len(paramModel.Params))
	for _, object := range paramModel.Objects {
		standardPaths[object.StandardPath] = struct{}{}
	}
	for _, param := range paramModel.Params {
		standardPaths[param.StandardPath] = struct{}{}
	}

	checked := 0
	for _, group := range doc.Groups {
		for _, param := range group.Params {
			standardPath := param.StandardPath
			if standardPath == "" {
				standardPath = group.ObjectPath + param.Leaf
			}
			checked++
			assert.Containsf(t, standardPaths, standardPath, "quicksettings group %s param %s must reference BLN param model", group.ID, param.Name)
		}
	}
	assert.Equal(t, 54, checked)
}

func TestBuiltinLTENeighborCellIncludesRequiredTACAndNumericConstraints(t *testing.T) {
	for _, model := range []string{"BM", "BLQ", "MLN", "MLQ", "ENB_DEFAULT_181"} {
		t.Run(model, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "data", "quicksettings", model+".xml"))
			require.NoError(t, err)

			var doc xmlQuickSettings
			require.NoError(t, xml.Unmarshal(data, &doc))

			var neighborGroup *xmlGroup
			for i := range doc.Groups {
				if doc.Groups[i].ID == "enb-neighbor-cell" {
					neighborGroup = &doc.Groups[i]
					break
				}
			}
			require.NotNil(t, neighborGroup)

			params := make(map[string]xmlParam, len(neighborGroup.Params))
			for _, param := range neighborGroup.Params {
				params[param.Name] = param
			}

			tac, ok := params["TAC"]
			require.True(t, ok)
			assert.Equal(t, "true", tac.Required)
			assert.Equal(t, "unsignedInt", tac.Type)
			assert.Equal(t, "0", tac.MinValue)
			assert.Equal(t, "65535", tac.MaxValue)

			for name, want := range map[string]struct {
				typ string
				min string
				max string
			}{
				"EARFCN":  {typ: "unsignedInt", min: "0", max: "65535"},
				"PCI":     {typ: "unsignedInt", min: "0", max: "503"},
				"QOffset": {typ: "int", min: "-24", max: "24"},
				"CIO":     {typ: "int", min: "-24", max: "24"},
			} {
				param, exists := params[name]
				require.True(t, exists, "%s metadata missing", name)
				assert.Equal(t, want.typ, param.Type, "%s type", name)
				assert.Equal(t, want.min, param.MinValue, "%s minValue", name)
				assert.Equal(t, want.max, param.MaxValue, "%s maxValue", name)
			}

			if model == "BLQ" {
				cellID := params["CellID"]
				assert.Equal(t, "true", cellID.Required)
				assert.Equal(t, "unsignedInt", cellID.Type)
				assert.Equal(t, "0", cellID.MinValue)
				assert.Equal(t, "268435455", cellID.MaxValue)
				assert.Equal(t, "true", params["PCI"].Required)
				assert.Equal(t, "0", params["QOffset"].DefaultValue)
				assert.Equal(t, "0", params["CIO"].DefaultValue)

				enbType := params["NeighborCellEnbType"]
				assert.Equal(t, "NeighCellEnbType", enbType.Leaf)
				assert.Equal(t, "1", enbType.DefaultValue)
				assert.Equal(t, []xmlEnumOption{
					{Value: "1", Label: "Home"},
					{Value: "0", Label: "Macro"},
				}, enbType.EnumOptions)

				x2Flag := params["X2Flag"]
				assert.Equal(t, "0", x2Flag.DefaultValue)
				assert.Equal(t, []xmlEnumOption{
					{Value: "0", Label: "SON"},
					{Value: "1", Label: "Manual"},
				}, x2Flag.EnumOptions)
			}

			if model == "BM" {
				cellID := params["CellID"]
				assert.Equal(t, "true", cellID.Required)
				assert.Equal(t, "unsignedInt", cellID.Type)
				assert.Equal(t, "0", cellID.MinValue)
				assert.Equal(t, "268435455", cellID.MaxValue)
				assert.Equal(t, "true", params["PCI"].Required)
				assert.Equal(t, "0", params["QOffset"].DefaultValue)
				assert.Equal(t, "0", params["CIO"].DefaultValue)

				enbType := params["NeighborCellEnbType"]
				assert.Equal(t, "NeighCellTypeContainer", enbType.Leaf)
				assert.Equal(t, "1", enbType.DefaultValue)
				assert.Equal(t, []xmlEnumOption{
					{Value: "1", Label: "Home"},
					{Value: "0", Label: "Macro"},
				}, enbType.EnumOptions)

				x2Flag := params["X2Flag"]
				assert.Equal(t, "X2Flag", x2Flag.Leaf)
				assert.Equal(t, "0", x2Flag.DefaultValue)
				assert.Equal(t, []xmlEnumOption{
					{Value: "0", Label: "SON"},
					{Value: "1", Label: "Manual"},
				}, x2Flag.EnumOptions)
			}
		})
	}
}

func TestBuiltinLTEInterFrequencyQRxLevMinUsesNumericConstraints(t *testing.T) {
	reg := NewRegistry()
	loader := NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		filepath.Join("..", "..", "data"),
		reg,
		nil,
	)

	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	for _, model := range []string{"BLQ", "MLN", "MLQ", "BM", "BLN"} {
		t.Run(model, func(t *testing.T) {
			groups := reg.GetByParamModel(model)
			var interFrequencyGroup *Group
			for i := range groups {
				group := &groups[i]
				if group.ID == "enb-neighbor-freq" {
					interFrequencyGroup = group
					break
				}
			}
			require.NotNil(t, interFrequencyGroup)

			var qRxLevMin *Param
			for i := range interFrequencyGroup.Params {
				param := &interFrequencyGroup.Params[i]
				if param.Leaf == "QRxLevMinSIB5" {
					qRxLevMin = param
					break
				}
			}
			require.NotNil(t, qRxLevMin)
			assert.Equal(t, "int", qRxLevMin.Type)
			require.NotNil(t, qRxLevMin.MinValue)
			require.NotNil(t, qRxLevMin.MaxValue)
			assert.EqualValues(t, -70, *qRxLevMin.MinValue)
			assert.EqualValues(t, -22, *qRxLevMin.MaxValue)
		})
	}
}

func TestRegistry_GetByParamModel_IsCopy(t *testing.T) {
	reg := NewRegistry()
	reg.Replace("BLQ", []Group{{ID: "g1"}})
	got := reg.GetByParamModel("BLQ")
	got[0].ID = "mutated"
	again := reg.GetByParamModel("BLQ")
	assert.Equal(t, "g1", again[0].ID, "Registry should return a defensive copy")
}

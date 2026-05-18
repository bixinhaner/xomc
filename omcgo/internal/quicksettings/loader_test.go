package quicksettings

import (
	"context"
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
	assert.Contains(t, err.Error(), "missing leaf")
}

func TestRegistry_GetByParamModel_IsCopy(t *testing.T) {
	reg := NewRegistry()
	reg.Replace("BLQ", []Group{{ID: "g1"}})
	got := reg.GetByParamModel("BLQ")
	got[0].ID = "mutated"
	again := reg.GetByParamModel("BLQ")
	assert.Equal(t, "g1", again[0].ID, "Registry should return a defensive copy")
}

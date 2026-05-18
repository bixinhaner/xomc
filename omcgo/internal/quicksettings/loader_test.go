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

	writeXML(t, dir, "enb.xml", `<?xml version="1.0" encoding="UTF-8"?>
<quickSettings tech="lte">
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

	writeXML(t, dir, "gnb.xml", `<?xml version="1.0" encoding="UTF-8"?>
<quickSettings tech="nr">
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
	assert.Equal(t, 3, rep.RowsAffected) // 2 enb groups + 1 gnb group

	lte := reg.GetByTech(TechLTE)
	require.Len(t, lte, 2)
	assert.Equal(t, "enb-cell", lte[0].ID)
	assert.False(t, lte[0].MultiInstance)
	assert.Equal(t, "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity", lte[0].Params[0].StandardPath)
	assert.Equal(t, "enb-nbr", lte[1].ID)
	assert.True(t, lte[1].MultiInstance)
	assert.NotEmpty(t, lte[1].ObjectPath)
	assert.Equal(t, "CID", lte[1].Params[0].Leaf)

	nr := reg.GetByTech(TechNR)
	require.Len(t, nr, 1)
	assert.Equal(t, "gnb-cell", nr[0].ID)
	assert.False(t, nr[0].MultiInstance)
}

func TestLoader_MissingFile_DoesNotFail(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "quicksettings")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	// 只放 enb.xml，gnb.xml 缺失应跳过而非报错
	writeXML(t, dir, "enb.xml", `<?xml version="1.0" encoding="UTF-8"?>
<quickSettings tech="lte">
  <group id="enb-cell" titleZh="小区" titleEn="Cell">
    <param name="ECI" titleZh="ECI" titleEn="ECI" standardPath="Device.X.CellIdentity"/>
  </group>
</quickSettings>`)

	reg := NewRegistry()
	loader := NewLoader(appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"}, tmp, reg, nil)
	rep, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, rep.FilesLoaded)
	assert.Equal(t, 1, rep.FilesSkipped)
	assert.Len(t, reg.GetByTech(TechNR), 0)
}

func TestLoader_BadXML_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "quicksettings")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	writeXML(t, dir, "enb.xml", `<quickSettings><bad-xml`)

	reg := NewRegistry()
	loader := NewLoader(appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"}, tmp, reg, nil)
	_, err := loader.LoadOnce(context.Background())
	assert.Error(t, err)
}

func TestLoader_ValidationCatches_MissingLeafForMultiInstance(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "quicksettings")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	writeXML(t, dir, "enb.xml", `<?xml version="1.0" encoding="UTF-8"?>
<quickSettings tech="lte">
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

func TestRegistry_GetByTech_IsCopy(t *testing.T) {
	reg := NewRegistry()
	reg.Replace(TechLTE, []Group{{ID: "g1"}})
	got := reg.GetByTech(TechLTE)
	got[0].ID = "mutated"
	again := reg.GetByTech(TechLTE)
	assert.Equal(t, "g1", again[0].ID, "Registry should return a defensive copy")
}

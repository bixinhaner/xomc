package indicator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// writeXML 是测试辅助:在指定 dir 下写一份最小合法的 indicator XML 文件(无 deviceType)。
func writeXML(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	body := []byte(`<indicatorModel platform="X" indicatorCount="0"></indicatorModel>`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// writeXMLDevice 同 writeXML,但根元素带 deviceType 属性(根级制式分类用)。
func writeXMLDevice(t *testing.T, path, deviceType string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	body := []byte(`<indicatorModel platform="X" deviceType="` + deviceType + `" indicatorCount="0"></indicatorModel>`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestResolveENBSources_SingleTree: 单目录树扫描 enb/*.xml(builtin + custom 同住)。
func TestResolveENBSources_SingleTree(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/enb/ALL.xml"))
	writeXML(t, filepath.Join(base, "indicator-library/enb/BLQ.xml"))

	got, err := resolveENBSources(base, "indicator-library/enb")
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "indicator-library/enb/ALL.xml", got[0].LoadedFrom)
	assert.Equal(t, "indicator-library/enb/BLQ.xml", got[1].LoadedFrom)
	for _, src := range got {
		assert.FileExists(t, src.AbsPath)
	}
}

// TestResolveENBSources_SkipsSidecar: sidecar 标记文件(X.xml.custom)被跳过。
func TestResolveENBSources_SkipsSidecar(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/enb/MY.xml"))
	// sidecar 标记
	if err := os.WriteFile(filepath.Join(base, "indicator-library/enb/MY.xml.custom"), []byte{}, 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	got, err := resolveENBSources(base, "indicator-library/enb")
	assert.NoError(t, err)
	assert.Len(t, got, 1, "sidecar 不应被当作 XML 文件扫描")
	assert.Equal(t, "indicator-library/enb/MY.xml", got[0].LoadedFrom)
}

// TestResolveENBSources_DirMissing: ENB 子目录不存在 → 返错(部署/镜像问题)。
func TestResolveENBSources_DirMissing(t *testing.T) {
	base := t.TempDir()
	_, err := resolveENBSources(base, "indicator-library/enb")
	assert.Error(t, err, "enb 子目录不存在应返错")
}

// TestResolveRootTechSources_BuiltinAndCustom: 根级出厂单文件(文件名分类)+
// 根级自定义上传(deviceType 分类)并存;异制式文件被排除。
func TestResolveRootTechSources_BuiltinAndCustom(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/GSM.xml"))                 // 出厂,无 deviceType 也按文件名归 gsm
	writeXMLDevice(t, filepath.Join(base, "indicator-library/MY_GSM.xml"), "GSM") // 自定义,deviceType 归 gsm
	writeXMLDevice(t, filepath.Join(base, "indicator-library/MY_GNB.xml"), "GNB") // 异制式 → 排除

	got, err := resolveRootTechSources(base, "indicator-library", "gsm")
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "indicator-library/GSM.xml", got[0].LoadedFrom)
	assert.Equal(t, "indicator-library/MY_GSM.xml", got[1].LoadedFrom)
}

// TestResolveRootTechSources_DirMissing: 根级目录不存在 → 返空,无错。
func TestResolveRootTechSources_DirMissing(t *testing.T) {
	base := t.TempDir()
	got, err := resolveRootTechSources(base, "indicator-library", "gsm")
	assert.NoError(t, err)
	assert.Empty(t, got)
}

// TestResolveRootTechSources_OnlyBuiltin: 仅出厂单文件 → 1 条。
func TestResolveRootTechSources_OnlyBuiltin(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/GNB.xml"))
	got, err := resolveRootTechSources(base, "indicator-library", "gnb")
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "indicator-library/GNB.xml", got[0].LoadedFrom)
}

// TestResolveRootTechSources_NoDeviceTypeSkipped: 根级文件缺 deviceType 且非出厂
// 文件名 → 不属于任何制式,跳过(enb 子目录文件不受影响,由 resolveENBSources 承载)。
func TestResolveRootTechSources_NoDeviceTypeSkipped(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/ORPHAN.xml"))
	for _, tech := range []string{"gsm", "gnb"} {
		got, err := resolveRootTechSources(base, "indicator-library", tech)
		assert.NoError(t, err)
		assert.Empty(t, got, "缺 deviceType 的根级文件不应归入 %s", tech)
	}
}

// TestClassifyRootIndicatorTech: 出厂文件名映射 > deviceType 属性;不可识别返空。
func TestClassifyRootIndicatorTech(t *testing.T) {
	base := t.TempDir()
	gsmBuiltin := filepath.Join(base, "GSM.xml")
	writeXML(t, gsmBuiltin)
	custom := filepath.Join(base, "MY.xml")
	writeXMLDevice(t, custom, "gnb") // 大小写不敏感
	orphan := filepath.Join(base, "ORPHAN.xml")
	writeXML(t, orphan)

	assert.Equal(t, "gsm", classifyRootIndicatorTech(gsmBuiltin, "GSM.xml"))
	assert.Equal(t, "gnb", classifyRootIndicatorTech(custom, "MY.xml"))
	assert.Equal(t, "", classifyRootIndicatorTech(orphan, "ORPHAN.xml"))
	assert.Equal(t, "", classifyRootIndicatorTech(filepath.Join(base, "GONE.xml"), "GONE.xml"))
}

// TestScanXMLBasenames_IgnoresNonXMLAndSidecar: 仅 .xml 被采纳;隐藏/子目录/sidecar 跳过。
func TestScanXMLBasenames_IgnoresNonXMLAndSidecar(t *testing.T) {
	dir := t.TempDir()
	writeXML(t, filepath.Join(dir, "A.xml"))
	writeXML(t, filepath.Join(dir, "B.xml"))
	_ = os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("not xml"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, ".hidden.xml"), []byte("hidden"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "A.xml.custom"), []byte{}, 0o644)
	_ = os.Mkdir(filepath.Join(dir, "subdir"), 0o755)

	got, err := scanXMLBasenames(dir)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"A.xml", "B.xml"}, got)
}

// TestScanXMLBasenamesOptional_MissingDir: 目录不存在静默返空 + nil error。
func TestScanXMLBasenamesOptional_MissingDir(t *testing.T) {
	got, err := scanXMLBasenamesOptional("/nonexistent/path/xyz")
	assert.NoError(t, err)
	assert.Empty(t, got)
}

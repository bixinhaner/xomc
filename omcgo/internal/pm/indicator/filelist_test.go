package indicator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// writeXML 是测试辅助:在指定 dir 下写一份最小合法的 indicator XML 文件。
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

// TestResolveSingleTechSources_RootAndSubdir: 根级单文件 + 同制式子目录并存。
func TestResolveSingleTechSources_RootAndSubdir(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/GSM.xml"))
	writeXML(t, filepath.Join(base, "indicator-library/gsm/MY_GSM.xml"))

	got, err := resolveSingleTechSources(base, "indicator-library/GSM.xml", "indicator-library/gsm")
	assert.NoError(t, err)
	assert.Len(t, got, 2)

	loadedFromList := []string{got[0].LoadedFrom, got[1].LoadedFrom}
	assert.Contains(t, loadedFromList, "indicator-library/GSM.xml")
	assert.Contains(t, loadedFromList, "indicator-library/gsm/MY_GSM.xml")
}

// TestResolveSingleTechSources_BothMissing: 根级单文件 + 子目录都不存在 → 返空,无错。
func TestResolveSingleTechSources_BothMissing(t *testing.T) {
	base := t.TempDir()
	got, err := resolveSingleTechSources(base, "indicator-library/GSM.xml", "indicator-library/gsm")
	assert.NoError(t, err)
	assert.Empty(t, got)
}

// TestResolveSingleTechSources_OnlyRoot: 仅根级单文件 → 1 条。
func TestResolveSingleTechSources_OnlyRoot(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/GNB.xml"))
	got, err := resolveSingleTechSources(base, "indicator-library/GNB.xml", "indicator-library/gnb")
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "indicator-library/GNB.xml", got[0].LoadedFrom)
}

// TestResolveSingleTechSources_OnlySubdir: 仅子目录(根级单文件不存在) → 全部子目录文件。
func TestResolveSingleTechSources_OnlySubdir(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/gnb/A.xml"))
	writeXML(t, filepath.Join(base, "indicator-library/gnb/B.xml"))
	got, err := resolveSingleTechSources(base, "indicator-library/GNB.xml", "indicator-library/gnb")
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	// 按 LoadedFrom 字典序排序
	assert.Equal(t, "indicator-library/gnb/A.xml", got[0].LoadedFrom)
	assert.Equal(t, "indicator-library/gnb/B.xml", got[1].LoadedFrom)
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

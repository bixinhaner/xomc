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

// TestResolveENBSources_BuiltinOnly: 仅 builtin 存在 → 输出全部 builtin 文件。
func TestResolveENBSources_BuiltinOnly(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/enb/ALL.xml"))
	writeXML(t, filepath.Join(base, "indicator-library/enb/BLQ.xml"))

	got, err := resolveENBSources(base, "indicator-library/enb", "indicator-library-custom/enb", true)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "indicator-library/enb/ALL.xml", got[0].LoadedFrom)
	assert.Equal(t, "indicator-library/enb/BLQ.xml", got[1].LoadedFrom)
	for _, src := range got {
		assert.FileExists(t, src.AbsPath)
	}
}

// TestResolveENBSources_CustomOnly: 仅 custom 存在(builtin 缺) → 返错(builtin 必须存在)。
func TestResolveENBSources_BuiltinMissing(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library-custom/enb/MY.xml"))
	// builtin 子目录不存在
	_, err := resolveENBSources(base, "indicator-library/enb", "indicator-library-custom/enb", true)
	assert.Error(t, err, "builtin enb 不存在应返错(部署/镜像问题)")
}

// TestResolveENBSources_CustomMissing: builtin 存在 + custom 不存在 → 仅 builtin。
func TestResolveENBSources_CustomMissing(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/enb/ALL.xml"))
	got, err := resolveENBSources(base, "indicator-library/enb", "indicator-library-custom/enb", true)
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "indicator-library/enb/ALL.xml", got[0].LoadedFrom)
}

// TestResolveENBSources_CustomOverrideBuiltin: 同名 + customWins=true → custom 胜出(D1)。
func TestResolveENBSources_CustomOverridesBuiltin(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/enb/ALL.xml"))
	writeXML(t, filepath.Join(base, "indicator-library-custom/enb/ALL.xml")) // 同名
	writeXML(t, filepath.Join(base, "indicator-library/enb/BLQ.xml"))

	got, err := resolveENBSources(base, "indicator-library/enb", "indicator-library-custom/enb", true)
	assert.NoError(t, err)
	assert.Len(t, got, 2, "ALL 合并为 1 + BLQ 1 = 2")

	// 按字典序排序后:custom 前缀 > builtin 前缀,所以 BLQ(builtin) 排前面,ALL(custom) 排后面
	// 实际排序:"indicator-library-custom/..." 大于 "indicator-library/..." 因为 '-' < '/'
	// (ASCII '-'=0x2D, '/'=0x2F → '-' < '/' → custom 排序在 builtin 之前)
	loadedFromList := []string{got[0].LoadedFrom, got[1].LoadedFrom}
	assert.Contains(t, loadedFromList, "indicator-library-custom/enb/ALL.xml", "custom ALL 胜出")
	assert.Contains(t, loadedFromList, "indicator-library/enb/BLQ.xml", "BLQ 仅 builtin")
	assert.NotContains(t, loadedFromList, "indicator-library/enb/ALL.xml", "builtin ALL 被压制")
}

// TestResolveENBSources_BuiltinWinsWhenCustomDisabled: customWins=false → builtin 胜出。
func TestResolveENBSources_BuiltinWinsWhenCustomDisabled(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/enb/ALL.xml"))
	writeXML(t, filepath.Join(base, "indicator-library-custom/enb/ALL.xml"))
	writeXML(t, filepath.Join(base, "indicator-library-custom/enb/MY.xml")) // custom 独有,无碰撞

	got, err := resolveENBSources(base, "indicator-library/enb", "indicator-library-custom/enb", false)
	assert.NoError(t, err)
	assert.Len(t, got, 2, "ALL=builtin 胜出 + MY=custom 独有 = 2")

	loadedFromList := []string{got[0].LoadedFrom, got[1].LoadedFrom}
	assert.Contains(t, loadedFromList, "indicator-library/enb/ALL.xml", "customWins=false → builtin ALL 胜出")
	assert.Contains(t, loadedFromList, "indicator-library-custom/enb/MY.xml", "MY 仅 custom,无碰撞")
	assert.NotContains(t, loadedFromList, "indicator-library-custom/enb/ALL.xml", "custom ALL 被压制")
}

// TestResolveSingleTechSources_BuiltinAndCustom: builtin 单文件 + custom 子目录都存在 → 并存。
func TestResolveSingleTechSources_BuiltinAndCustom(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/GSM.xml"))
	writeXML(t, filepath.Join(base, "indicator-library-custom/gsm/MY_GSM.xml"))

	got, err := resolveSingleTechSources(base, "indicator-library/GSM.xml", "indicator-library-custom/gsm")
	assert.NoError(t, err)
	assert.Len(t, got, 2)

	loadedFromList := []string{got[0].LoadedFrom, got[1].LoadedFrom}
	assert.Contains(t, loadedFromList, "indicator-library/GSM.xml")
	assert.Contains(t, loadedFromList, "indicator-library-custom/gsm/MY_GSM.xml")
}

// TestResolveSingleTechSources_BothMissing: builtin 单文件 + custom 子目录都不存在 → 返空,无错。
func TestResolveSingleTechSources_BothMissing(t *testing.T) {
	base := t.TempDir()
	got, err := resolveSingleTechSources(base, "indicator-library/GSM.xml", "indicator-library-custom/gsm")
	assert.NoError(t, err)
	assert.Empty(t, got)
}

// TestResolveSingleTechSources_OnlyBuiltin: 仅 builtin 单文件 → 1 条。
func TestResolveSingleTechSources_OnlyBuiltin(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library/GNB.xml"))
	got, err := resolveSingleTechSources(base, "indicator-library/GNB.xml", "indicator-library-custom/gnb")
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "indicator-library/GNB.xml", got[0].LoadedFrom)
}

// TestResolveSingleTechSources_OnlyCustom: 仅 custom 子目录(builtin 单文件不存在) → 全部 custom 文件。
func TestResolveSingleTechSources_OnlyCustom(t *testing.T) {
	base := t.TempDir()
	writeXML(t, filepath.Join(base, "indicator-library-custom/gnb/A.xml"))
	writeXML(t, filepath.Join(base, "indicator-library-custom/gnb/B.xml"))
	got, err := resolveSingleTechSources(base, "indicator-library/GNB.xml", "indicator-library-custom/gnb")
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	// 按 LoadedFrom 字典序排序
	assert.Equal(t, "indicator-library-custom/gnb/A.xml", got[0].LoadedFrom)
	assert.Equal(t, "indicator-library-custom/gnb/B.xml", got[1].LoadedFrom)
}

// TestScanXMLBasenames_IgnoresNonXML: 仅 .xml 后缀文件被采纳;隐藏与子目录跳过。
func TestScanXMLBasenames_IgnoresNonXML(t *testing.T) {
	dir := t.TempDir()
	writeXML(t, filepath.Join(dir, "A.xml"))
	writeXML(t, filepath.Join(dir, "B.xml"))
	_ = os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("not xml"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, ".hidden.xml"), []byte("hidden"), 0o644)
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

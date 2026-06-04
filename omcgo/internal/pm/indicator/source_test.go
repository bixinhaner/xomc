package indicator

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSidecarClassification 验证 2026-06-04 sidecar 来源判定:
// X.xml 旁存在 X.xml.custom ⇒ custom(可删);否则 builtin;loadedFrom 空 ⇒ unknown。
func TestSidecarClassification(t *testing.T) {
	base := t.TempDir()
	mk := func(rel string) {
		abs := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	mk("indicator-library/enb/ALL.xml")       // builtin
	mk("indicator-library/GSM.xml")           // builtin 根级单文件
	mk("indicator-library/enb/MY.xml")        // custom:XML + sidecar
	mk("indicator-library/enb/MY.xml.custom") // sidecar 标记

	cases := []struct {
		name       string
		loadedFrom string
		wantSrc    Source
		wantDel    bool
	}{
		{"builtin enb no sidecar", "indicator-library/enb/ALL.xml", SourceBuiltin, false},
		{"builtin gsm root no sidecar", "indicator-library/GSM.xml", SourceBuiltin, false},
		{"custom enb with sidecar", "indicator-library/enb/MY.xml", SourceCustom, true},
		{"empty -> unknown", "", SourceUnknown, false},
		{"missing file no sidecar -> builtin", "indicator-library/enb/Ghost.xml", SourceBuiltin, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifySource(base, tc.loadedFrom); got != tc.wantSrc {
				t.Errorf("ClassifySource(%q) = %q, want %q", tc.loadedFrom, got, tc.wantSrc)
			}
			if got := IsDeletable(base, tc.loadedFrom); got != tc.wantDel {
				t.Errorf("IsDeletable(%q) = %v, want %v", tc.loadedFrom, got, tc.wantDel)
			}
		})
	}
}

// TestSourceConstants 防御性测试:确保导出常量值不被无意修改。
func TestSourceConstants(t *testing.T) {
	if SourceBuiltin != "builtin" {
		t.Errorf("SourceBuiltin = %q, want %q", SourceBuiltin, "builtin")
	}
	if SourceCustom != "custom" {
		t.Errorf("SourceCustom = %q, want %q", SourceCustom, "custom")
	}
	if SourceUnknown != "unknown" {
		t.Errorf("SourceUnknown = %q, want %q", SourceUnknown, "unknown")
	}
	if CustomMarkerSuffix != ".custom" {
		t.Errorf("CustomMarkerSuffix = %q, want %q", CustomMarkerSuffix, ".custom")
	}
}

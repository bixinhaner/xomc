package parammodel

import (
	"os"
	"path/filepath"
	"testing"
)

// touchData 在 baseDir 下按 rel(slash 路径)建一个非空文件(含父目录),供 sidecar 测试。
func touchData(t *testing.T, baseDir, rel string) {
	t.Helper()
	abs := filepath.Join(baseDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// TestSidecarClassification 验证 2026-06-04 sidecar 来源判定:
// X.xml 旁存在 X.xml.custom ⇒ custom(可删);否则 builtin;loadedFrom 空 ⇒ unknown。
func TestSidecarClassification(t *testing.T) {
	base := t.TempDir()
	touchData(t, base, "param-mappings/BTS.xml")            // builtin:仅 XML
	touchData(t, base, "param-mappings/MyModel.xml")        // custom:XML + sidecar
	touchData(t, base, "param-mappings/MyModel.xml.custom") // sidecar 标记

	cases := []struct {
		name       string
		loadedFrom string
		wantSrc    Source
		wantDel    bool
	}{
		{"builtin no sidecar", "param-mappings/BTS.xml", SourceBuiltin, false},
		{"custom with sidecar", "param-mappings/MyModel.xml", SourceCustom, true},
		{"empty -> unknown", "", SourceUnknown, false},
		{"missing file no sidecar -> builtin", "param-mappings/Ghost.xml", SourceBuiltin, false},
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

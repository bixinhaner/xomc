package parammodel

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestResolveLoadedFrom 覆盖路径前缀规范化(对应 source.go 的 ClassifySource 输入契约)。
func TestResolveLoadedFrom(t *testing.T) {
	cases := []struct {
		name    string
		base    string
		absPath string
		want    string
	}{
		{
			name:    "builtin relative",
			base:    "/etc/omcgo/data",
			absPath: "/etc/omcgo/data/param-mappings/BTS.xml",
			want:    "param-mappings/BTS.xml",
		},
		{
			name:    "base is dev relative",
			base:    "data",
			absPath: "data/param-mappings/BLQ.xml",
			want:    "param-mappings/BLQ.xml",
		},
		{
			name:    "same dir",
			base:    "/x",
			absPath: "/x/BLQ.xml",
			want:    "BLQ.xml",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveLoadedFrom(tc.base, tc.absPath)
			if got != tc.want {
				t.Errorf("resolveLoadedFrom(%q, %q) = %q, want %q",
					tc.base, tc.absPath, got, tc.want)
			}
		})
	}
}

// TestResolveLoadedFrom_ClassifierIntegration 验证 resolveLoadedFrom 输出
// 被 sidecar 版 ClassifySource 正确分类(两个函数构成的完整契约链)。
func TestResolveLoadedFrom_ClassifierIntegration(t *testing.T) {
	base := t.TempDir()
	touchData(t, base, "param-mappings/BTS.xml")            // builtin:仅 XML
	touchData(t, base, "param-mappings/MyModel.xml")        // custom:XML + sidecar
	touchData(t, base, "param-mappings/MyModel.xml.custom") // sidecar 标记
	cases := []struct {
		rel      string
		wantSrc  Source
		wantDel  bool
		nickName string
	}{
		{"param-mappings/BTS.xml", SourceBuiltin, false, "builtin BTS"},
		{"param-mappings/MyModel.xml", SourceCustom, true, "custom MyModel"},
	}
	for _, tc := range cases {
		t.Run(tc.nickName, func(t *testing.T) {
			absPath := filepath.Join(base, filepath.FromSlash(tc.rel))
			loadedFrom := resolveLoadedFrom(base, absPath)
			if got := ClassifySource(base, loadedFrom); got != tc.wantSrc {
				t.Errorf("ClassifySource(%q) = %q, want %q", loadedFrom, got, tc.wantSrc)
			}
			if got := IsDeletable(base, loadedFrom); got != tc.wantDel {
				t.Errorf("IsDeletable(%q) = %v, want %v", loadedFrom, got, tc.wantDel)
			}
		})
	}
}

// TestResolveLoaderFiles_Whitelist 覆盖白名单模式:builtin 严格按列表。
func TestResolveLoaderFiles_Whitelist(t *testing.T) {
	files, err := resolveLoaderFiles(
		"/b",
		[]string{"BTS.xml", "BLQ.xml"},
		[]string{"standard-model.xml"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"/b/BTS.xml", "/b/BLQ.xml"}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("files = %v, want %v", files, want)
	}
}

// TestResolveLoaderFiles_AutoScan 覆盖单目录自动扫描:剔除 reserved,
// 跳过 sidecar(X.xml.custom),按 basename 字典序输出。
func TestResolveLoaderFiles_AutoScan(t *testing.T) {
	base := t.TempDir()
	builtinDir := filepath.Join(base, "param-mappings")
	if err := os.Mkdir(builtinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStub(t, builtinDir, "BTS.xml")
	writeStub(t, builtinDir, "BLQ.xml")
	writeStub(t, builtinDir, "standard-model.xml")    // reserved,应被排除
	writeStub(t, builtinDir, "Custom.xml")            // custom XML(仍应加载)
	writeStub(t, builtinDir, "Custom.xml.custom")     // sidecar,应被跳过
	writeStub(t, builtinDir, "BTS.xml.deleted.20260604120000") // 备份残留,非 .xml,应被跳过

	files, err := resolveLoaderFiles(
		builtinDir,
		nil,
		[]string{"standard-model.xml", "products.xml"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotBases := basenames(files)
	sort.Strings(gotBases)
	want := []string{"BLQ.xml", "BTS.xml", "Custom.xml"}
	if !reflect.DeepEqual(gotBases, want) {
		t.Errorf("basenames = %v, want %v", gotBases, want)
	}
}

// TestResolveLoaderFiles_BuiltinDirMissing 覆盖 builtin dir 不存在 → error。
// builtin 是镜像层必备,不该容忍。
func TestResolveLoaderFiles_BuiltinDirMissing(t *testing.T) {
	base := t.TempDir()
	builtinDir := filepath.Join(base, "no-such")
	if _, err := resolveLoaderFiles(builtinDir, nil, nil); err == nil {
		t.Errorf("expected error when builtin dir missing, got nil")
	}
}

// ── 辅助 ───────────────────────────────────────────────────────────────

func writeStub(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("<paramModel/>"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func basenames(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = filepath.Base(p)
	}
	return out
}

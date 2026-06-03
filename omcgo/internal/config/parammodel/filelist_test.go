package parammodel

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestMergeFileLists 覆盖 T-0178 §9.3 合并算法的所有分支:
//   - 仅 builtin / 仅 custom / 双都为空
//   - 同名冲突 + customOverrides=true(默认,custom 胜)
//   - 同名冲突 + customOverrides=false(builtin 胜)
//   - 非冲突文件并存
//   - 输出按 basename 字典序稳定排序
//   - 输入切片不被修改(纯函数防御)
func TestMergeFileLists(t *testing.T) {
	type want struct {
		paths []string
	}
	cases := []struct {
		name           string
		builtinDir     string
		builtinFiles   []string
		customDir      string
		customFiles    []string
		customOverride bool
		want           want
	}{
		{
			name:         "builtin only",
			builtinDir:   "/b",
			builtinFiles: []string{"BTS.xml", "BLQ.xml"},
			customDir:    "/c",
			customFiles:  nil,
			want: want{paths: []string{
				"/b/BLQ.xml",
				"/b/BTS.xml",
			}},
		},
		{
			name:         "custom only",
			builtinDir:   "/b",
			builtinFiles: nil,
			customDir:    "/c",
			customFiles:  []string{"CBQQ.xml"},
			want: want{paths: []string{
				"/c/CBQQ.xml",
			}},
		},
		{
			name:         "both empty",
			builtinDir:   "/b",
			builtinFiles: nil,
			customDir:    "/c",
			customFiles:  nil,
			want:         want{paths: []string{}},
		},
		{
			name:         "no conflict — both contribute",
			builtinDir:   "/b",
			builtinFiles: []string{"BTS.xml", "BLQ.xml"},
			customDir:    "/c",
			customFiles:  []string{"CBQQ.xml", "ANQ.xml"},
			want: want{paths: []string{
				"/c/ANQ.xml",
				"/b/BLQ.xml",
				"/b/BTS.xml",
				"/c/CBQQ.xml",
			}},
		},
		{
			name:           "same name — custom overrides true",
			builtinDir:     "/b",
			builtinFiles:   []string{"BTS.xml"},
			customDir:      "/c",
			customFiles:    []string{"BTS.xml"},
			customOverride: true,
			want: want{paths: []string{
				"/c/BTS.xml",
			}},
		},
		{
			name:           "same name — custom overrides false",
			builtinDir:     "/b",
			builtinFiles:   []string{"BTS.xml"},
			customDir:      "/c",
			customFiles:    []string{"BTS.xml"},
			customOverride: false,
			want: want{paths: []string{
				"/b/BTS.xml",
			}},
		},
		{
			name:           "mixed conflict + non-conflict — custom overrides true",
			builtinDir:     "/b",
			builtinFiles:   []string{"BTS.xml", "BLQ.xml"},
			customDir:      "/c",
			customFiles:    []string{"BTS.xml", "CBQQ.xml"},
			customOverride: true,
			want: want{paths: []string{
				"/b/BLQ.xml",
				"/c/BTS.xml", // custom wins
				"/c/CBQQ.xml",
			}},
		},
		{
			name:           "mixed conflict + non-conflict — custom overrides false",
			builtinDir:     "/b",
			builtinFiles:   []string{"BTS.xml", "BLQ.xml"},
			customDir:      "/c",
			customFiles:    []string{"BTS.xml", "CBQQ.xml"},
			customOverride: false,
			want: want{paths: []string{
				"/b/BLQ.xml",
				"/b/BTS.xml", // builtin wins
				"/c/CBQQ.xml",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 防御性测试:保存输入副本,验证 mergeFileLists 不修改输入
			builtinIn := append([]string(nil), tc.builtinFiles...)
			customIn := append([]string(nil), tc.customFiles...)

			got := mergeFileLists(
				tc.builtinDir, tc.builtinFiles,
				tc.customDir, tc.customFiles,
				tc.customOverride,
			)
			if !reflect.DeepEqual(got, tc.want.paths) {
				t.Errorf("mergeFileLists =\n  %v\nwant\n  %v", got, tc.want.paths)
			}
			if !reflect.DeepEqual(builtinIn, tc.builtinFiles) {
				t.Errorf("builtinFiles was mutated: got %v, want %v",
					tc.builtinFiles, builtinIn)
			}
			if !reflect.DeepEqual(customIn, tc.customFiles) {
				t.Errorf("customFiles was mutated: got %v, want %v",
					tc.customFiles, customIn)
			}
		})
	}
}

// TestMergeFileLists_StableSort 验证排序按 basename 而非完整路径
// (即使两个文件的目录不同,basename 同名时合并只剩一个,验证字典序)。
func TestMergeFileLists_StableSort(t *testing.T) {
	// basename 排序应是: ANQ < BLQ < CBQQ
	got := mergeFileLists(
		"/zzz", []string{"BLQ.xml"}, // /zzz/BLQ.xml — 路径靠后但 basename 靠中
		"/aaa", []string{"CBQQ.xml", "ANQ.xml"}, // /aaa/* — 路径靠前
		true,
	)
	want := []string{
		"/aaa/ANQ.xml",
		"/zzz/BLQ.xml",
		"/aaa/CBQQ.xml",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unstable sort:\n  got  %v\n  want %v", got, want)
	}
}

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
			name:    "custom relative",
			base:    "/etc/omcgo/data",
			absPath: "/etc/omcgo/data/param-mappings-custom/CBQQ.xml",
			want:    "param-mappings-custom/CBQQ.xml",
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
// 被 ClassifySource 正确分类(两个函数构成的完整契约链)。
func TestResolveLoadedFrom_ClassifierIntegration(t *testing.T) {
	base := "/etc/omcgo/data"
	cases := []struct {
		absPath  string
		wantSrc  Source
		wantDel  bool
		nickName string
	}{
		{"/etc/omcgo/data/param-mappings/BTS.xml", SourceBuiltin, true, "builtin BTS"},
		{"/etc/omcgo/data/param-mappings-custom/CBQQ.xml", SourceCustom, true, "custom CBQQ"},
		{"/etc/omcgo/data/param-mappings-custom/BTS.xml", SourceCustom, true, "custom override BTS"},
	}
	for _, tc := range cases {
		t.Run(tc.nickName, func(t *testing.T) {
			loadedFrom := resolveLoadedFrom(base, tc.absPath)
			if got := ClassifySource(loadedFrom); got != tc.wantSrc {
				t.Errorf("ClassifySource(%q) = %q, want %q", loadedFrom, got, tc.wantSrc)
			}
			if got := IsDeletable(loadedFrom); got != tc.wantDel {
				t.Errorf("IsDeletable(%q) = %v, want %v", loadedFrom, got, tc.wantDel)
			}
		})
	}
}

// TestResolveLoaderFiles_Whitelist 覆盖白名单模式:custom 不参与,builtin 严格按列表。
func TestResolveLoaderFiles_Whitelist(t *testing.T) {
	files, _, warnings, err := resolveLoaderFiles(
		"/b", "/c",
		[]string{"BTS.xml", "BLQ.xml"},
		[]string{"standard-model.xml"},
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	want := []string{"/b/BTS.xml", "/b/BLQ.xml"}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("files = %v, want %v", files, want)
	}
}

// TestResolveLoaderFiles_AutoScan_BuiltinOnly 覆盖 custom dir 不存在场景(首次部署)。
func TestResolveLoaderFiles_AutoScan_BuiltinOnly(t *testing.T) {
	base := t.TempDir()
	builtinDir := filepath.Join(base, "param-mappings")
	customDir := filepath.Join(base, "param-mappings-custom") // 故意不创建

	if err := os.Mkdir(builtinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStub(t, builtinDir, "BTS.xml")
	writeStub(t, builtinDir, "BLQ.xml")
	writeStub(t, builtinDir, "standard-model.xml") // reserved,应被排除

	files, _, warnings, err := resolveLoaderFiles(
		builtinDir, customDir,
		nil,
		[]string{"standard-model.xml", "products.xml"},
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("ENOENT custom dir 不该出 warning: %v", warnings)
	}
	gotBases := basenames(files)
	sort.Strings(gotBases)
	want := []string{"BLQ.xml", "BTS.xml"}
	if !reflect.DeepEqual(gotBases, want) {
		t.Errorf("basenames = %v, want %v", gotBases, want)
	}
}

// TestResolveLoaderFiles_AutoScan_CustomOverride 覆盖同名 custom 覆盖 builtin。
func TestResolveLoaderFiles_AutoScan_CustomOverride(t *testing.T) {
	base := t.TempDir()
	builtinDir := filepath.Join(base, "param-mappings")
	customDir := filepath.Join(base, "param-mappings-custom")
	if err := os.Mkdir(builtinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(customDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStub(t, builtinDir, "BTS.xml")
	writeStub(t, builtinDir, "BLQ.xml")
	writeStub(t, customDir, "BTS.xml")  // 同名覆盖
	writeStub(t, customDir, "CBQQ.xml") // 新增

	files, _, warnings, err := resolveLoaderFiles(
		builtinDir, customDir,
		nil,
		[]string{"standard-model.xml"},
		true, // customOverrides
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	// 期望:BLQ 来自 builtin,BTS 来自 custom(覆盖),CBQQ 来自 custom
	wantPaths := map[string]string{
		"BLQ.xml":  builtinDir, // 不冲突,留 builtin
		"BTS.xml":  customDir,  // 冲突 + override=true → custom 胜
		"CBQQ.xml": customDir,  // 仅 custom 有
	}
	if len(files) != len(wantPaths) {
		t.Fatalf("file count = %d, want %d (files=%v)", len(files), len(wantPaths), files)
	}
	for _, f := range files {
		base := filepath.Base(f)
		gotDir := filepath.Dir(f)
		wantDir, ok := wantPaths[base]
		if !ok {
			t.Errorf("unexpected file %q", f)
			continue
		}
		if gotDir != wantDir {
			t.Errorf("file %q: dir = %q, want %q", base, gotDir, wantDir)
		}
	}
}

// TestResolveLoaderFiles_AutoScan_BuiltinWinsOnFalse 覆盖
// customOverrides=false 同名冲突 builtin 胜出。
func TestResolveLoaderFiles_AutoScan_BuiltinWinsOnFalse(t *testing.T) {
	base := t.TempDir()
	builtinDir := filepath.Join(base, "param-mappings")
	customDir := filepath.Join(base, "param-mappings-custom")
	if err := os.Mkdir(builtinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(customDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStub(t, builtinDir, "BTS.xml")
	writeStub(t, customDir, "BTS.xml")
	writeStub(t, customDir, "CBQQ.xml")

	files, _, _, err := resolveLoaderFiles(
		builtinDir, customDir,
		nil,
		[]string{"standard-model.xml"},
		false, // builtin wins on conflict
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantPaths := map[string]string{
		"BTS.xml":  builtinDir, // builtin 胜
		"CBQQ.xml": customDir,  // 仅 custom,无冲突
	}
	if len(files) != len(wantPaths) {
		t.Fatalf("file count = %d, want %d (files=%v)", len(files), len(wantPaths), files)
	}
	for _, f := range files {
		base := filepath.Base(f)
		gotDir := filepath.Dir(f)
		wantDir, ok := wantPaths[base]
		if !ok {
			t.Errorf("unexpected file %q", f)
			continue
		}
		if gotDir != wantDir {
			t.Errorf("file %q: dir = %q, want %q", base, gotDir, wantDir)
		}
	}
}

// TestResolveLoaderFiles_AutoScan_CustomIsFile 覆盖 customDir 路径存在但是文件不是目录,
// 应进 warnings 但不阻塞 builtin 加载。
func TestResolveLoaderFiles_AutoScan_CustomIsFile(t *testing.T) {
	base := t.TempDir()
	builtinDir := filepath.Join(base, "param-mappings")
	customDir := filepath.Join(base, "param-mappings-custom") // 注意:作为文件创建,非目录
	if err := os.Mkdir(builtinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStub(t, builtinDir, "BTS.xml")
	if err := os.WriteFile(customDir, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, _, warnings, err := resolveLoaderFiles(
		builtinDir, customDir,
		nil,
		[]string{"standard-model.xml"},
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 1 {
		t.Errorf("want 1 warning(not-a-dir), got %d: %v", len(warnings), warnings)
	}
	// builtin 仍正常加载
	if len(files) != 1 || filepath.Base(files[0]) != "BTS.xml" {
		t.Errorf("builtin BTS.xml missing or extra files: %v", files)
	}
}

// TestResolveLoaderFiles_BuiltinDirMissing 覆盖 builtin dir 不存在 → error。
// builtin 是镜像层必备,不该容忍。
func TestResolveLoaderFiles_BuiltinDirMissing(t *testing.T) {
	base := t.TempDir()
	builtinDir := filepath.Join(base, "no-such")
	customDir := filepath.Join(base, "param-mappings-custom")
	_, _, _, err := resolveLoaderFiles(builtinDir, customDir, nil, nil, true)
	if err == nil {
		t.Errorf("expected error when builtin dir missing, got nil")
	}
}

// TestFindShadowedCustom 覆盖 T-0178 R-NEW-T0178-8 检测函数(纯字符串集合操作)。
func TestFindShadowedCustom(t *testing.T) {
	cases := []struct {
		name    string
		builtin []string
		custom  []string
		want    []string
	}{
		{"both empty", nil, nil, nil},
		{"only builtin", []string{"BTS.xml"}, nil, nil},
		{"only custom", nil, []string{"CBQQ.xml"}, nil},
		{"no overlap", []string{"BTS.xml"}, []string{"CBQQ.xml"}, nil},
		{"full overlap", []string{"BTS.xml", "BLQ.xml"}, []string{"BLQ.xml", "BTS.xml"},
			[]string{"BLQ.xml", "BTS.xml"}},
		{"partial overlap order preserved",
			[]string{"BTS.xml", "BLQ.xml"},
			[]string{"CBQQ.xml", "BTS.xml", "NEW.xml"},
			[]string{"BTS.xml"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findShadowedCustom(tc.builtin, tc.custom)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("findShadowedCustom(%v, %v) = %v, want %v",
					tc.builtin, tc.custom, got, tc.want)
			}
		})
	}
}

// TestResolveLoaderFiles_ShadowedCustom 验证 resolveLoaderFiles 在双目录扫描下
// 返回同名 custom 文件清单(供 Loader.run 决定是否 WARN)。
func TestResolveLoaderFiles_ShadowedCustom(t *testing.T) {
	base := t.TempDir()
	builtinDir := filepath.Join(base, "param-mappings")
	customDir := filepath.Join(base, "param-mappings-custom")
	if err := os.Mkdir(builtinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(customDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStub(t, builtinDir, "BTS.xml")
	writeStub(t, builtinDir, "BLQ.xml")
	writeStub(t, customDir, "BTS.xml")  // shadow
	writeStub(t, customDir, "CBQQ.xml") // not shadow

	// customOverrides=true 时 shadowed 仍返(算法不依赖该 flag,由 caller 决定是否 log)
	_, shadowed, _, err := resolveLoaderFiles(
		builtinDir, customDir, nil, nil, true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"BTS.xml"}
	if !reflect.DeepEqual(shadowed, want) {
		t.Errorf("shadowed = %v, want %v", shadowed, want)
	}

	// customOverrides=false 时一样返(算法不依赖该 flag)
	_, shadowed, _, err = resolveLoaderFiles(
		builtinDir, customDir, nil, nil, false,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(shadowed, want) {
		t.Errorf("shadowed (override=false) = %v, want %v", shadowed, want)
	}

	// 白名单模式 shadowed 应为 nil(白名单不扫 custom)
	_, shadowed, _, err = resolveLoaderFiles(
		builtinDir, customDir,
		[]string{"BLQ.xml"}, nil, true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shadowed != nil {
		t.Errorf("whitelist mode shadowed should be nil, got %v", shadowed)
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

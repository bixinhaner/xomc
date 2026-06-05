package parammodel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidateUploadFilename 验证文件名白名单守门(T-0178 §9.4 安全边界 4 + 11)。
func TestValidateUploadFilename(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// 通过
		{"basic name", "CBQQ.xml", false},
		{"with underscore", "my_model.xml", false},
		{"with hyphen", "my-model.xml", false},
		{"alphanumeric", "BLQ123.xml", false},
		{"single char", "A.xml", false},

		// 拒绝 — 扩展名
		{"no extension", "CBQQ", true},
		{"wrong extension", "CBQQ.txt", true},
		{"uppercase extension", "CBQQ.XML", true}, // 严格小写,避免大小写敏感系统的混淆
		{"multi extension", "CBQQ.xml.bak", true},
		{"prefix dot", "CBQQ.tar.xml", true}, // 多扩展名变种,正则只允许一个 .

		// 拒绝 — 路径分隔符 / 遍历
		{"slash", "dir/CBQQ.xml", true},
		{"backslash", "dir\\CBQQ.xml", true},
		{"path traversal", "../CBQQ.xml", true},
		{"absolute", "/etc/CBQQ.xml", true},

		// 拒绝 — 隐藏 / 空白 / 特殊字符
		{"dot prefix", ".hidden.xml", true},
		{"space", "C BQQ.xml", true},
		{"chinese", "中文.xml", true},
		{"emoji", "🎉.xml", true},
		{"semicolon", "X;Y.xml", true},

		// 拒绝 — 长度边界
		{"empty", "", true},
		{"too long", strings.Repeat("A", 65) + ".xml", true},
		{"max-1 ok", strings.Repeat("A", 64) + ".xml", false},

		// 拒绝 — 保留名
		{"reserved standard-model", "standard-model.xml", true},
		{"reserved products", "products.xml", true},
		{"reserved product-routing", "product-name-routing.xml", true},
		{"reserved param-routing", "param-model-routing.xml", true},
		// 大小写不敏感的保留名检查
		{"reserved uppercase rejected", "Products.xml", true}, // 大小写不同的保留名,白名单正则要求扩展名小写,所以这条会先在正则被拦
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUploadFilename(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateUploadFilename(%q) err=%v, wantErr=%v",
					tc.input, err, tc.wantErr)
			}
		})
	}
}

// TestValidateUploadXML 验证 XML 内容校验(T-0178 §9.4 校验 3)。
func TestValidateUploadXML(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// 通过 — 真值源 model.go xmlParameterModel `xml:"parameterModel"`,
		// builtin BLQ.xml 等所有真实 XML 根元素均为 <parameterModel paramModel="..." ...>。
		{"valid empty parameterModel", `<parameterModel/>`, false},
		{"valid with attrs", `<parameterModel paramModel="CBQQ" totalEntries="100"/>`, false},
		{"valid with children", `<parameterModel><object name="x"/></parameterModel>`, false},
		{"valid with xml decl", `<?xml version="1.0"?><parameterModel/>`, false},
		{"valid with comment before root", `<!-- header --><parameterModel/>`, false},
		{"valid with whitespace", "  \n  <parameterModel/>  ", false},

		// 拒绝 — 形态错
		{"empty body", "", true},
		{"malformed not closed", `<parameterModel`, true},
		{"malformed mismatched", `<parameterModel></parameterModl>`, true},
		{"not xml html", `<html><body/></html>`, true},
		{"wrong root products", `<products/>`, true},
		// 历史踩坑:曾误把 paramModel(attribute 名)当作 root,导致 Upload 拒绝
		// 所有真实 builtin XML(BLQ.xml 等)。Root 必须是 parameterModel。
		{"wrong root paramModel", `<paramModel/>`, true},
		// namespace 不参与判定(与 Loader.xml.Unmarshal 默认行为一致,
		// xml struct tag `xml:"parameterModel"` 也只匹配 Local Name):
		{"namespaced root accepted", `<x:parameterModel xmlns:x="urn:test"/>`, false},
		{"plain text", `just text content`, true},
		{"json content", `{"name":"X"}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUploadXML([]byte(tc.input))
			if (err != nil) != tc.wantErr {
				t.Errorf("validateUploadXML err=%v, wantErr=%v\ninput=%q",
					err, tc.wantErr, tc.input)
			}
		})
	}
}

// TestPathContainedIn 验证路径包含守门(T-0178 §9.4 校验 4 安全边界)。
func TestPathContainedIn(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "param-mappings-custom")
	cases := []struct {
		name   string
		target string
		want   bool
	}{
		{"direct child", filepath.Join(base, "CBQQ.xml"), true},
		{"same dir", base, true},
		{"trailing slash", base + string(filepath.Separator), true},
		{"parent escape", filepath.Join(base, "..", "..", "etc", "passwd"), false},
		{"sibling dir", filepath.Join(tmp, "other-dir", "X.xml"), false},
		{"different prefix", "/var/lib/something/x.xml", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pathContainedIn(base, tc.target)
			if got != tc.want {
				t.Errorf("pathContainedIn(%q, %q) = %v, want %v",
					base, tc.target, got, tc.want)
			}
		})
	}
}

// TestToModelView_SourceAndDeletable 验证 modelView DTO 的 source/deletable
// 由 sidecar 派生(2026-06-04)。
func TestToModelView_SourceAndDeletable(t *testing.T) {
	base := t.TempDir()
	touchData(t, base, "param-mappings/BTS.xml")           // builtin
	touchData(t, base, "param-mappings/Custom.xml")        // custom
	touchData(t, base, "param-mappings/Custom.xml.custom") // sidecar
	cases := []struct {
		name          string
		loadedFrom    string
		wantSource    Source
		wantDeletable bool
	}{
		{"builtin BTS", "param-mappings/BTS.xml", SourceBuiltin, false},
		{"custom with sidecar", "param-mappings/Custom.xml", SourceCustom, true},
		{"empty", "", SourceUnknown, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &ParamModel{Name: "X", LoadedFrom: tc.loadedFrom}
			view := toModelView(base, m)
			if view.Source != tc.wantSource {
				t.Errorf("Source = %q, want %q", view.Source, tc.wantSource)
			}
			if view.Deletable != tc.wantDeletable {
				t.Errorf("Deletable = %v, want %v", view.Deletable, tc.wantDeletable)
			}
			if view.LoadedFrom != tc.loadedFrom {
				t.Errorf("LoadedFrom roundtrip failed: %q != %q", view.LoadedFrom, tc.loadedFrom)
			}
		})
	}
}

// TestValidateUploadFilename_NamePath 验证 name 上传路径(目标 = <name>.xml)
// 复用同一个 validateUploadFilename 守门(三库 XML 导入重构)。
func TestValidateUploadFilename_NamePath(t *testing.T) {
	cases := []struct {
		name    string
		input   string // 用户提交的 name(不含扩展名)
		wantErr bool
	}{
		{"basic name", "CBQQ", false},
		{"underscore", "my_model", false},
		{"hyphen", "my-model", false},
		{"empty name", "", true},
		{"slash in name", "dir/CBQQ", true},
		{"traversal", "../CBQQ", true},
		{"reserved products", "products", true},
		{"chinese", "中文", true},
		{"too long", strings.Repeat("A", 65), true},
		{"max ok", strings.Repeat("A", 64), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUploadFilename(tc.input + ".xml")
			if (err != nil) != tc.wantErr {
				t.Errorf("validateUploadFilename(%q+.xml) err=%v, wantErr=%v",
					tc.input, err, tc.wantErr)
			}
		})
	}
}

// TestSidecarWriteAndDelete 验证上传写 sidecar / 删除移除 sidecar 的物理契约
// (用 temp dir,不依赖 DB / HTTP)。
func TestSidecarWriteAndDelete(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "MyModel.xml")
	sidecar := xmlPath + CustomMarkerSuffix

	// 写 XML + sidecar(模拟 UploadXML 落地)
	if err := os.WriteFile(xmlPath, []byte("<parameterModel name=\"MyModel\"/>"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sidecar, nil, 0o640); err != nil {
		t.Fatal(err)
	}

	// sidecar 存在 → IsCustom / IsDeletable 为 true
	loadedFrom := resolveLoadedFrom(dir, xmlPath)
	if !IsCustom(dir, loadedFrom) {
		t.Fatalf("IsCustom = false after sidecar write, want true")
	}
	if !IsDeletable(dir, loadedFrom) {
		t.Fatalf("IsDeletable = false after sidecar write, want true")
	}

	// 移除 sidecar(模拟 DeleteModel)→ IsCustom 变 false
	if err := os.Remove(sidecar); err != nil {
		t.Fatal(err)
	}
	if IsCustom(dir, loadedFrom) {
		t.Errorf("IsCustom = true after sidecar removed, want false")
	}
}

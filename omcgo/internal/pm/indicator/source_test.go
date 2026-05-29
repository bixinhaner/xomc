package indicator

import "testing"

// TestClassifySource 覆盖 T-0180 PRD §3 source 三态契约。
// 表驱动:每条用例对应 PRD GWT-1 / GWT-4 / GWT-5 之一,
// 加上历史数据 / 边界 / 安全输入等防御场景。
func TestClassifySource(t *testing.T) {
	cases := []struct {
		name       string
		loadedFrom string
		want       Source
	}{
		// 正常路径(builtin 含 enb/ 子目录 + GSM.xml / GNB.xml 单文件)
		{"builtin LTE multi-file", "indicator-library/enb/ALL.xml", SourceBuiltin},
		{"builtin LTE BLQ", "indicator-library/enb/BLQ.xml", SourceBuiltin},
		{"builtin GSM single-file", "indicator-library/GSM.xml", SourceBuiltin},
		{"builtin GNB single-file", "indicator-library/GNB.xml", SourceBuiltin},

		// custom 路径(三制式均子目录化)
		{"custom LTE", "indicator-library-custom/enb/MY_ENB.xml", SourceCustom},
		{"custom GSM", "indicator-library-custom/gsm/MY_GSM.xml", SourceCustom},
		{"custom GNB", "indicator-library-custom/gnb/MY_GNB.xml", SourceCustom},
		{"custom same name as builtin", "indicator-library-custom/enb/ALL.xml", SourceCustom},

		// 历史数据(migration 000217 之前 Loader 入库无 loaded_from;NULL 由 DB 层处理为 "")
		{"legacy empty string", "", SourceUnknown},
		{"legacy bare filename", "ALL.xml", SourceUnknown},
		{"legacy nested no prefix", "enb/ALL.xml", SourceUnknown},

		// 边界 / 安全(前缀必须严格匹配,防绕过)
		{"prefix-like no slash", "indicator-library", SourceUnknown},
		{"custom prefix-like no slash", "indicator-library-custom", SourceUnknown},
		{"path traversal attempt", "../../etc/passwd", SourceUnknown},
		{"unrelated nested path", "other-dir/X.xml", SourceUnknown},

		// 前缀大小写敏感(filepath.Rel + ToSlash 保证大小写规范)
		{"case mismatch builtin", "Indicator-Library/GSM.xml", SourceUnknown},
		{"case mismatch custom", "Indicator-Library-Custom/enb/X.xml", SourceUnknown},

		// 容易撞前缀的相邻字符串(防 BuiltinDirPrefix 误判 CustomDirPrefix 子串)
		// indicator-library-custom/ 开头同时也满足 strings.HasPrefix 是 "indicator-library/"?
		// 不,因为 "indicator-library/" 含末尾 / 而 "indicator-library-custom/" 之后第 17 字符是 '-' 不是 '/'。
		// 这条用例就是验证这点。
		{"custom must not classify as builtin", "indicator-library-custom/enb/X.xml", SourceCustom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifySource(tc.loadedFrom); got != tc.want {
				t.Errorf("ClassifySource(%q) = %q, want %q", tc.loadedFrom, got, tc.want)
			}
		})
	}
}

// TestIsDeletable 验证守门规则:仅 SourceCustom 可删。
// PRD §3 GWT-4(可删)+ GWT-5(内置拒删)行为基础。
func TestIsDeletable(t *testing.T) {
	cases := []struct {
		name       string
		loadedFrom string
		want       bool
	}{
		{"custom enb is deletable", "indicator-library-custom/enb/MY.xml", true},
		{"custom gsm is deletable", "indicator-library-custom/gsm/MY.xml", true},
		{"custom gnb is deletable", "indicator-library-custom/gnb/MY.xml", true},
		{"custom same-name override is deletable", "indicator-library-custom/enb/ALL.xml", true},

		{"builtin LTE multi-file NOT deletable", "indicator-library/enb/ALL.xml", false},
		{"builtin GSM single-file NOT deletable", "indicator-library/GSM.xml", false},
		{"builtin GNB single-file NOT deletable", "indicator-library/GNB.xml", false},

		{"legacy bare filename NOT deletable", "ALL.xml", false},
		{"empty NOT deletable", "", false},
		{"path traversal NOT deletable", "../../etc/passwd", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsDeletable(tc.loadedFrom); got != tc.want {
				t.Errorf("IsDeletable(%q) = %v, want %v", tc.loadedFrom, got, tc.want)
			}
		})
	}
}

// TestSourceConstants 防御性测试:确保导出常量值不被无意修改。
// 这些字符串是 DB 持久化字段 + 前端 mapping 的契约,改动需要数据迁移。
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
	if BuiltinDirPrefix != "indicator-library/" {
		t.Errorf("BuiltinDirPrefix = %q, want %q", BuiltinDirPrefix, "indicator-library/")
	}
	if CustomDirPrefix != "indicator-library-custom/" {
		t.Errorf("CustomDirPrefix = %q, want %q", CustomDirPrefix, "indicator-library-custom/")
	}
	if BuiltinDirSubdir != "indicator-library" {
		t.Errorf("BuiltinDirSubdir = %q, want %q", BuiltinDirSubdir, "indicator-library")
	}
	if CustomDirSubdir != "indicator-library-custom" {
		t.Errorf("CustomDirSubdir = %q, want %q", CustomDirSubdir, "indicator-library-custom")
	}
}

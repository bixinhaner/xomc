package parammodel

import "testing"

// TestClassifySource 覆盖 T-0178 §9.2 source 三态契约。
// 表驱动:每条用例对应 PRD §3 GWT-2 / GWT-3 / GWT-4 之一,
// 加上历史数据 / 边界 / 安全输入等防御场景。
func TestClassifySource(t *testing.T) {
	cases := []struct {
		name       string
		loadedFrom string
		want       Source
	}{
		// 正常路径
		{"builtin BTS", "param-mappings/BTS.xml", SourceBuiltin},
		{"builtin BLQ deep", "param-mappings/BLQ.xml", SourceBuiltin},
		{"custom CBQQ", "param-mappings-custom/CBQQ.xml", SourceCustom},
		{"custom same name as builtin", "param-mappings-custom/BTS.xml", SourceCustom},

		// 历史数据(T-0098 P1-06 入库无前缀;T-0178 迁移前可能存在)
		{"legacy bare filename", "BTS.xml", SourceUnknown},
		{"legacy empty string", "", SourceUnknown},

		// 边界 / 安全(前缀必须严格匹配,防绕过)
		{"prefix-like but no slash", "param-mappings", SourceUnknown},
		{"custom prefix-like no slash", "param-mappings-custom", SourceUnknown},
		{"path traversal attempt", "../../etc/passwd", SourceUnknown},
		{"unrelated nested path", "other-dir/X.xml", SourceUnknown},

		// 前缀大小写敏感(filepath.Rel + ToSlash 保证大小写规范)
		{"case mismatch", "Param-Mappings/BTS.xml", SourceUnknown},
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
// PRD §3 GWT-3 + GWT-4 + GWT-5 行为基础。
func TestIsDeletable(t *testing.T) {
	cases := []struct {
		name       string
		loadedFrom string
		want       bool
	}{
		{"custom is deletable", "param-mappings-custom/CBQQ.xml", true},
		{"builtin is NOT deletable", "param-mappings/BTS.xml", false},
		{"legacy bare filename is NOT deletable", "BTS.xml", false},
		{"empty is NOT deletable", "", false},
		{"path traversal is NOT deletable", "../../etc/passwd", false},
		{"custom same-name override is deletable", "param-mappings-custom/BTS.xml", true},
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
	if BuiltinDirPrefix != "param-mappings/" {
		t.Errorf("BuiltinDirPrefix = %q, want %q", BuiltinDirPrefix, "param-mappings/")
	}
	if CustomDirPrefix != "param-mappings-custom/" {
		t.Errorf("CustomDirPrefix = %q, want %q", CustomDirPrefix, "param-mappings-custom/")
	}
}

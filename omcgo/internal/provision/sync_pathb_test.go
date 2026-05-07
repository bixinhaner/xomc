package provision

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/omcgo/omcgo/internal/config/parammodel"
)

func TestExtractStorablePrefixes_HappyPath(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.WiFi.SSID.{i}.Enabled", IsStorable: true, EntryType: "parameter"},
		{PrivatePath: "Dev.WiFi.SSID.{i}.Name", IsStorable: true, EntryType: "parameter"},
		{PrivatePath: "Dev.WiFi.Radio.{i}.Channel", IsStorable: true, EntryType: "parameter"},
		{PrivatePath: "Dev.System.Mode", IsStorable: true, EntryType: "parameter"},
		{PrivatePath: "Dev.NotStorable", IsStorable: false, EntryType: "parameter"}, // 应被过滤
	}
	got := extractStorablePrefixes(mappings)
	want := []string{"Dev.System.", "Dev.WiFi.Radio.", "Dev.WiFi.SSID."}
	sort.Strings(got)
	assert.Equal(t, want, got)
}

func TestExtractStorablePrefixes_DedupesIdenticalPrefixes(t *testing.T) {
	// 多条同前缀 → 只保留一条
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.A.{i}.X", IsStorable: true, EntryType: "parameter"},
		{PrivatePath: "Dev.A.{i}.Y", IsStorable: true, EntryType: "parameter"},
		{PrivatePath: "Dev.A.{i}.Z", IsStorable: true, EntryType: "parameter"},
	}
	got := extractStorablePrefixes(mappings)
	assert.Equal(t, []string{"Dev.A."}, got)
}

func TestExtractStorablePrefixes_AllNonStorable_Empty(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.A.B", IsStorable: false},
		{PrivatePath: "Dev.X.Y", IsStorable: false},
	}
	got := extractStorablePrefixes(mappings)
	assert.Empty(t, got)
}

func TestExtractStorablePrefixes_ObjectPaths(t *testing.T) {
	// 末尾 "." 的 object 路径原样使用
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "Dev.WiFi.", IsStorable: true, EntryType: "object"},
		{PrivatePath: "Dev.System.", IsStorable: true, EntryType: "object"},
	}
	got := extractStorablePrefixes(mappings)
	sort.Strings(got)
	assert.Equal(t, []string{"Dev.System.", "Dev.WiFi."}, got)
}

func TestExtractStorablePrefixes_EmptyAndInvalid(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{PrivatePath: "", IsStorable: true},
		{PrivatePath: "NoDot", IsStorable: true}, // 无 "." → basePrefix 返回空 → 过滤
	}
	got := extractStorablePrefixes(mappings)
	assert.Empty(t, got)
}

func TestBasePrefix_Cases(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"Dev.WiFi.SSID.{i}.Enabled", "Dev.WiFi.SSID."},
		{"Dev.WiFi.SSID.{i}.{i}.Foo", "Dev.WiFi.SSID."}, // 截到第一个 {i}
		{"Dev.WiFi.SSID.", "Dev.WiFi.SSID."},
		{"Dev.System.Mode", "Dev.System."},
		{"Dev", ""},
		{"", ""},
		{"Dev.", "Dev."},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.out, basePrefix(c.in))
		})
	}
}

func TestInstantiateStandardPath_NoPlaceholder(t *testing.T) {
	// 模板 standardPath 无 {i} → 直接返回
	got := instantiateStandardPath("Dev.System.Mode", "Dev.System.Mode", "Device.System.Mode")
	assert.Equal(t, "Device.System.Mode", got)
}

func TestInstantiateStandardPath_SingleInstance(t *testing.T) {
	got := instantiateStandardPath(
		"Dev.WiFi.SSID.7.Enabled",
		"Dev.WiFi.SSID.{i}.Enabled",
		"Device.WiFi.SSID.{i}.Enable",
	)
	assert.Equal(t, "Device.WiFi.SSID.7.Enable", got)
}

func TestInstantiateStandardPath_MultipleInstances(t *testing.T) {
	got := instantiateStandardPath(
		"Foo.1.Bar.2.Baz",
		"Foo.{i}.Bar.{i}.Baz",
		"Foo.{i}.Bar.{i}.Baz",
	)
	assert.Equal(t, "Foo.1.Bar.2.Baz", got)
}

func TestInstantiateStandardPath_PartialOverlap_LeftFirst(t *testing.T) {
	// privatePath 首个 {i} 段对应实例号"3"，第二个对应"5"
	got := instantiateStandardPath(
		"P.3.Q.5.R",
		"P.{i}.Q.{i}.R",
		"S.{i}.T.{i}.U",
	)
	assert.Equal(t, "S.3.T.5.U", got)
}

func TestInstantiateStandardPath_LengthMismatch_Fallback(t *testing.T) {
	// 段数不一致 → 容错返回模板原文
	got := instantiateStandardPath(
		"Foo.1",
		"Foo.{i}.Bar",
		"Device.{i}.Bar",
	)
	assert.Equal(t, "Device.{i}.Bar", got)
}

func TestInstantiateStandardPath_NoPlaceholderInTemplate(t *testing.T) {
	// 即便 actualPrivate 含数字段，templateStandard 无 {i} → 原样返回
	got := instantiateStandardPath(
		"Dev.WiFi.SSID.7.Enabled",
		"Dev.WiFi.SSID.{i}.Enabled",
		"Device.WiFi.SSID.Enable",
	)
	assert.Equal(t, "Device.WiFi.SSID.Enable", got)
}

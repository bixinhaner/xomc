package export

import (
	"testing"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

func TestAdhocFirstColHeader(t *testing.T) {
	cases := map[string]string{
		"device": "设备 SN", "device_group": "设备组", "product": "产品",
		"band": "频段", "network": "全网", "aggregate_group": "聚合组",
		"": "设备 SN", "unknown": "设备 SN",
	}
	for dim, want := range cases {
		if got := adhocFirstColHeader(dim, appcontext.LocaleZH); got != want {
			t.Errorf("header(%q)=%q want %q", dim, got, want)
		}
	}
}

func TestAdhocFirstColHeader_English(t *testing.T) {
	cases := map[string]string{
		"device": "Device SN", "device_group": "Device Group", "product": "Product",
		"band": "Band", "network": "Network", "aggregate_group": "Aggregate Group",
		"": "Device SN", "unknown": "Device SN",
	}
	for dim, want := range cases {
		if got := adhocFirstColHeader(dim, appcontext.LocaleEN); got != want {
			t.Errorf("header(%q)=%q want %q", dim, got, want)
		}
	}
}

func TestAdhocIncludesCell(t *testing.T) {
	if !adhocIncludesCell("device") {
		t.Error("device 应含小区列")
	}
	for _, d := range []string{"device_group", "product", "band", "network", "aggregate_group", ""} {
		if adhocIncludesCell(d) {
			t.Errorf("%q 不应含小区列", d)
		}
	}
}

func TestAdhocObjectLabel(t *testing.T) {
	// device：只用 SN（与页面表格「设备 SN」列一致，去掉 OUI 前缀）
	if got := adhocObjectLabel("device", "OUI1", "SN1", "", "", "", "", 0); got != "SN1" {
		t.Errorf("device label=%q", got)
	}
	// device：仅 SN
	if got := adhocObjectLabel("device", "", "SN1", "", "", "", "", 0); got != "SN1" {
		t.Errorf("device label sn-only=%q", got)
	}
	// device_group：组名优先
	if got := adhocObjectLabel("device_group", "", "", "", "", "DeviceGroup=abcdef1234-xx", "华东A组", 0); got != "华东A组" {
		t.Errorf("group label=%q", got)
	}
	// device_group：组名缺失回退 uuid 前 8（剥 DeviceGroup= 前缀）
	if got := adhocObjectLabel("device_group", "", "", "", "", "DeviceGroup=abcdef1234-xxxx", "", 0); got != "abcdef12" {
		t.Errorf("group fallback=%q", got)
	}
	// device_group：object_ldn 带 ,Tech= 后缀时，回退也只取逗号前段剥前缀截 8（B2 修复：不把后缀混进 uuid）
	if got := adhocObjectLabel("device_group", "", "", "", "", "DeviceGroup=abcdef1234-xxxx,Tech=lte", "", 0); got != "abcdef12" {
		t.Errorf("group fallback with tech suffix=%q", got)
	}
	// product：产品名优先
	if got := adhocObjectLabel("product", "", "", "11112222-3333-4444", "NR-Pico", "", "", 0); got != "NR-Pico" {
		t.Errorf("product label=%q", got)
	}
	// product：缺失回退 id 前 8
	if got := adhocObjectLabel("product", "", "", "11112222-3333-4444", "", "", "", 0); got != "11112222" {
		t.Errorf("product fallback=%q", got)
	}
	// band：剥 Band= 前缀
	if got := adhocObjectLabel("band", "", "AGGREGATED", "", "", "Band=42", "", 0); got != "42" {
		t.Errorf("band label=%q", got)
	}
	// network
	if got := adhocObjectLabel("network", "", "AGGREGATED", "", "", "", "", 0); got != "全网" {
		t.Errorf("network label=%q", got)
	}
	// aggregate_group：聚合组(N个设备)
	if got := adhocObjectLabel("aggregate_group", "", "AGGREGATED", "", "", "", "", 3); got != "聚合组(3个设备)" {
		t.Errorf("aggregate_group label=%q", got)
	}
}

func TestAdhocObjectLabel_EnglishSystemValues(t *testing.T) {
	if got := adhocObjectLabel("network", "", "AGGREGATED", "", "", "", "", 0, appcontext.LocaleEN); got != "Network" {
		t.Errorf("network english label=%q", got)
	}
	if got := adhocObjectLabel("aggregate_group", "", "AGGREGATED", "", "", "", "", 3, appcontext.LocaleEN); got != "Aggregate Group (3 devices)" {
		t.Errorf("aggregate_group english label=%q", got)
	}
	if got := adhocObjectLabel("device_group", "", "", "", "", "DeviceGroup=abcdef1234-xx", "华东A组", 0, appcontext.LocaleEN); got != "华东A组" {
		t.Errorf("user-authored group name should stay unchanged, got %q", got)
	}
}

// adhocTechnology 从 device_group 维度 object_ldn 解析制式（设备组制式治本 B 方案），
// 与前端 adhocObjectColumn.ts 同口径（取 ,Tech= 后段、大写；无段返空）。
func TestAdhocTechnology(t *testing.T) {
	cases := map[string]string{
		"DeviceGroup=11112222-3333-4444-5555-666677778888,Tech=lte": "LTE",
		"DeviceGroup=11112222-3333-4444-5555-666677778888,Tech=nr":  "NR",
		"DeviceGroup=11112222-3333-4444-5555-666677778888,Tech=gsm": "GSM",
		// 老行 / 非设备组维度：无 ,Tech= 段 → 空串（调用方据此不渲染制式列值）
		"DeviceGroup=11112222-3333-4444-5555-666677778888": "",
		"Band=42": "",
		"":        "",
	}
	for ldn, want := range cases {
		if got := adhocTechnology(ldn); got != want {
			t.Errorf("adhocTechnology(%q)=%q want %q", ldn, got, want)
		}
	}
}

func TestFirst8(t *testing.T) {
	if got := first8("abcdef1234"); got != "abcdef12" {
		t.Errorf("first8 long=%q", got)
	}
	if got := first8("abc"); got != "abc" {
		t.Errorf("first8 short=%q", got)
	}
	if got := first8(""); got != "" {
		t.Errorf("first8 empty=%q", got)
	}
}

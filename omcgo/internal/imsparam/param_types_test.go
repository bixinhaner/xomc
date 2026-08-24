package imsparam

import (
	"strings"
	"testing"
)

func TestParamTypesRegistry(t *testing.T) {
	types := ParamTypes()
	if len(types) != 19 {
		t.Fatalf("expect 19 file types, got %d", len(types))
	}
	// imscore_filetask.txt 顺序：7 种 UD 双向 → 10 种仅上传 → 2 种仅下发。
	want := []struct {
		code string
		name string
		up   bool
		down bool
	}{
		{"FT_ImsCore_User_Setting_UD", "开户数据", true, true},
		{"FT_ImsCore_Ims_User_Setting_UD", "IMS用户数据", true, true},
		{"FT_ImsCore_Pbx_User_Setting_UD", "PBX用户数据", true, true},
		{"FT_ImsCore_User_Apn_Setting_UD", "用户APN设置", true, true},
		{"FT_ImsCore_Ue_IMEI_UD", "UE IMEI", true, true},
		{"FT_ImsCore_Ue_Route_Setting_UD", "UE 后路由设置", true, true},
		{"FT_ImsCore_Pcrf_Policy_Setting_UD", "流量策略设置", true, true},
		{"FT_ImsCore_User_Location_Info_U", "用户位置信息", true, false},
		{"FT_ImsCore_eNBgNB_Location_Info_U", "基站位置信息", true, false},
		{"FT_ImsCore_Signaling_Events_U", "信令事件", true, false},
		{"FT_ImsCore_Sip_Events_U", "SIP事件", true, false},
		{"FT_ImsCore_Cdr_U", "呼叫详单", true, false},
		{"FT_ImsCore_Operation_Logs_U", "操作日志", true, false},
		{"FT_ImsCore_Download_Auth_U", "授权认证文件", true, false},
		{"FT_ImsCore_Backups_U", "核心网服务备份", true, false},
		{"FT_ImsCore_Core_Logs_U", "核心网Core日志", true, false},
		{"FT_ImsCore_Web_Logs_U", "核心网Web日志", true, false},
		{"FT_ImsCore_Upload_License_D", "授权License", false, true},
		{"FT_ImsCore_Recovery_D", "核心网服务恢复", false, true},
	}
	for i, item := range types {
		if item.Code != want[i].code {
			t.Fatalf("types[%d].Code = %q, want %q", i, item.Code, want[i].code)
		}
		if item.Name != want[i].name {
			t.Fatalf("types[%d].Name = %q, want %q", i, item.Name, want[i].name)
		}
		if item.UploadSupported != want[i].up || item.DownloadSupported != want[i].down {
			t.Fatalf("types[%d] (%s) up/down = %v/%v, want %v/%v", i, item.Code,
				item.UploadSupported, item.DownloadSupported, want[i].up, want[i].down)
		}
		if !strings.HasPrefix(item.Code, "FT_ImsCore") {
			t.Fatalf("types[%d].Code %q should have FT_ImsCore prefix", i, item.Code)
		}
	}
}

func TestLookup(t *testing.T) {
	if def, ok := Lookup("FT_ImsCore_Ims_User_Setting_UD"); !ok || def.Name != "IMS用户数据" {
		t.Fatalf("Lookup(FT_ImsCore_Ims_User_Setting_UD) = %+v, %v", def, ok)
	}
	// 大小写不敏感
	if def, ok := Lookup("ft_imscore_cdr_u"); !ok || def.Code != "FT_ImsCore_Cdr_U" {
		t.Fatalf("Lookup(ft_imscore_cdr_u) = %+v, %v", def, ok)
	}
	if def, ok := Lookup("FT_ImsCore_Recovery_D"); !ok || !def.DownloadSupported || def.UploadSupported {
		t.Fatalf("Lookup(FT_ImsCore_Recovery_D) = %+v, %v", def, ok)
	}
	// 历史 code（已更名/已下线）应 miss
	for _, legacy := range []string{"FT_ImsCore_Policy_Setting_UD", "FT_ImsCore_Ue_Route_UD", "FT_ImsCore_Download_Auth_D", "FT_ImsCore_Backups_D", "FT_ImsCore_Upload_License_U"} {
		if _, ok := Lookup(legacy); ok {
			t.Fatalf("Lookup(%q) should miss (legacy code)", legacy)
		}
	}
	if _, ok := Lookup(""); ok {
		t.Fatal("Lookup(\"\") should miss")
	}
}

func TestNormalizeParamType(t *testing.T) {
	if got := NormalizeParamType("ft_imscore_ue_imei_ud"); got != "FT_ImsCore_Ue_IMEI_UD" {
		t.Fatalf("NormalizeParamType = %q, want canonical code", got)
	}
	if got := NormalizeParamType("  "); got != "" {
		t.Fatalf("NormalizeParamType(blank) = %q, want empty", got)
	}
}

func TestDisplayName(t *testing.T) {
	if got := DisplayName("FT_ImsCore_Ue_Route_Setting_UD"); got != "UE 后路由设置" {
		t.Fatalf("DisplayName = %q, want UE 后路由设置", got)
	}
	if got := DisplayName("FT_ImsCore_Core_Logs_U"); got != "核心网Core日志" {
		t.Fatalf("DisplayName = %q, want 核心网Core日志", got)
	}
	if got := DisplayName("FT_ImsCore_Unknown"); got != "FT_ImsCore_Unknown" {
		t.Fatalf("DisplayName(unknown) = %q, want passthrough", got)
	}
}

func TestIsLogType(t *testing.T) {
	for _, code := range []string{"FT_ImsCore_Operation_Logs_U", "FT_ImsCore_Core_Logs_U", "FT_ImsCore_Web_Logs_U"} {
		if !IsLogType(code) {
			t.Fatalf("IsLogType(%q) should be true", code)
		}
	}
	for _, code := range []string{"FT_ImsCore_Cdr_U", "FT_ImsCore_User_Setting_UD", ""} {
		if IsLogType(code) {
			t.Fatalf("IsLogType(%q) should be false", code)
		}
	}
}

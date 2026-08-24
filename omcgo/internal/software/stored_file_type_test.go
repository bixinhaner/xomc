package software

import "testing"

// splitStoredFileType：任务行 download_file_type 落库值带 ":FT_ImsCore_*" 尾段时，
// 拆成 CWMP FileType + ParamType；其它（含冒号但尾段不是 FT_ImsCore*）原样返回空 ParamType。
func TestSplitStoredFileType(t *testing.T) {
	cases := []struct {
		in        string
		wantFile  string
		wantParam string
	}{
		{"ImsCore Parameters File:FT_ImsCore_Policy_Setting_UD", "ImsCore Parameters File", "FT_ImsCore_Policy_Setting_UD"},
		{"ImsCore Parameters File:FT_ImsCore_Cdr_U", "ImsCore Parameters File", "FT_ImsCore_Cdr_U"},
		{"ImsCore Parameters File", "ImsCore Parameters File", ""},
		{"10 {OUI} Configuration File", "10 {OUI} Configuration File", ""},
		{"8", "8", ""},
		{"", "", ""},
		{"weird:FT1x", "weird:FT1x", ""}, // 尾段不是 FT_ImsCore* → 不拆
		{"a:b:FT_ImsCore_Ue_Route_UD", "a:b", "FT_ImsCore_Ue_Route_UD"}, // 只看最后一段
		{"IMS_PARAM_DISTRIBUTE:FT_ImsCore_Ue_IMEI_UD", "IMS_PARAM_DISTRIBUTE", "FT_ImsCore_Ue_IMEI_UD"},
		{"x:FT_IMSCORE_SIP_EVENTS_U", "x", "FT_IMSCORE_SIP_EVENTS_U"}, // 大小写不敏感
	}
	for _, tc := range cases {
		gotFile, gotParam := splitStoredFileType(tc.in)
		if gotFile != tc.wantFile || gotParam != tc.wantParam {
			t.Fatalf("splitStoredFileType(%q) = (%q, %q), want (%q, %q)",
				tc.in, gotFile, gotParam, tc.wantFile, tc.wantParam)
		}
	}
}

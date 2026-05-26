package acs

import "testing"

func TestExtractBadPathFromFaultString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "BLQ 实测格式：including 关键字带对象前缀（尾点）",
			in:   "Invalid Parameter Names [1], including: Device.Services.FAPService.2.CellConfig.LTE.RAN.NeighborList.LTECell.",
			want: "Device.Services.FAPService.2.CellConfig.LTE.RAN.NeighborList.LTECell.",
		},
		{
			name: "标准 9005 消息 + path 叶子",
			in:   "Invalid parameter name: Device.WiFi.SSID.7.Enable",
			want: "Device.WiFi.SSID.7.Enable",
		},
		{
			name: "Parameter 关键字带引号",
			in:   "Parameter 'Device.X_VENDOR.Foo.Bar' is not supported",
			want: "Device.X_VENDOR.Foo.Bar",
		},
		{
			name: "无关键字时兜底找最长 dot-separated token",
			in:   "Something went wrong with Device.System.Mode while processing",
			want: "Device.System.Mode",
		},
		{
			name: "纯描述无 path → 空串",
			in:   "Internal server error",
			want: "",
		},
		{
			name: "空字符串 → 空串",
			in:   "",
			want: "",
		},
		{
			name: "末尾逗号需 trim",
			in:   "Invalid parameter name: Device.X.Y,",
			want: "Device.X.Y",
		},
		{
			name: "[1] 计数器不被误抓",
			in:   "Invalid Parameter Names [1], including: Device.A.B.C",
			want: "Device.A.B.C",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := extractBadPathFromFaultString(tc.in)
			if got != tc.want {
				t.Errorf("extractBadPathFromFaultString(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

package software

import (
	"testing"
	"time"
)

// imsParamUploadFileName：paramType 去 FT_ImsCore_ 前缀 + 年月日时分秒；
// 日志类（*_Logs_U）.log 后缀，其余 .dat；非规范前缀用原值兜底。
func TestImsParamUploadFileName(t *testing.T) {
	now := time.Date(2026, 8, 20, 15, 32, 1, 0, time.UTC)
	cases := []struct {
		in   string
		want string
	}{
		{"FT_ImsCore_Pcrf_Policy_Setting_UD", "Pcrf_Policy_Setting_UD_20260820153201.dat"},
		{"FT_ImsCore_Cdr_U", "Cdr_U_20260820153201.dat"},
		{"FT_ImsCore_Operation_Logs_U", "Operation_Logs_U_20260820153201.log"},
		{"FT_ImsCore_Web_Logs_U", "Web_Logs_U_20260820153201.log"},
		{"FT_ImsCore_Core_Logs_U ", "Core_Logs_U_20260820153201.log"}, // 前后空白容忍
		{"Unknown_Type", "Unknown_Type_20260820153201.dat"},           // 非规范前缀兜底原值
	}
	for _, tc := range cases {
		if got := imsParamUploadFileName(tc.in, now); got != tc.want {
			t.Fatalf("imsParamUploadFileName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

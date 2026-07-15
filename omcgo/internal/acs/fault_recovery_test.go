package acs

import (
	"testing"

	"github.com/omcgo/omcgo/internal/task"
)

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
			name: "厂商 Get failed 格式支持两段 GSM 顶层路径",
			in:   "Get failed: DeviceGSM.NriNullDel",
			want: "DeviceGSM.NriNullDel",
		},
		{
			name: "BU1810 共享内存错误提取具体坏叶子",
			in:   "Failed to get parameter from shared memory for 'Device.DeviceInfo.EU.0.RouteIndex' (MIB DN: FAP.0.BBU_EU.0): Parameter with ExtendedKey not found in hash table (hash=0xfc2486138815d049)",
			want: "Device.DeviceInfo.EU.0.RouteIndex",
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

func TestRemoveFaultedGPVRequest(t *testing.T) {
	tests := []struct {
		name      string
		names     []string
		badPath   string
		want      []string
		wantFound bool
	}{
		{
			name:      "exact leaf is removed",
			names:     []string{"Device.Time.Enable", "Device.Time.NTPServer1"},
			badPath:   "Device.Time.Enable",
			want:      []string{"Device.Time.NTPServer1"},
			wantFound: true,
		},
		{
			name:      "object request covering vendor fault leaf is removed",
			names:     []string{"Device.DeviceInfo.EU.", "Device.Time.Enable"},
			badPath:   "Device.DeviceInfo.EU.0.RouteIndex",
			want:      []string{"Device.Time.Enable"},
			wantFound: true,
		},
		{
			name:      "sibling prefix is not removed",
			names:     []string{"Device.DeviceInfo.RU."},
			badPath:   "Device.DeviceInfo.EU.0.RouteIndex",
			want:      []string{"Device.DeviceInfo.RU."},
			wantFound: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := removeFaultedGPVRequest(tt.names, tt.badPath)
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("remaining = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("remaining = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestPlanGPVFaultRecoveryFallsBackToSkippingTranslatedBatch(t *testing.T) {
	names := []string{
		"Device.Services.FAPService.1.FAPControl.Halob.L2.Apn.2.apnDefault",
		"Device.Services.FAPService.1.FAPControl.LTE.Gateway.MultiS1ConnectionMode",
	}
	badPath := "Device.Services.FAPService.1.FAPControl.EMBEDDED_EPC.L2.Apn.2.apnDefault"

	remaining, skipped, ok := planGPVFaultRecovery(names, badPath, true)

	if !ok {
		t.Fatal("durable 9005 with a translated private path must be recoverable")
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining = %v, want empty batch", remaining)
	}
	if len(skipped) != len(names) {
		t.Fatalf("skipped = %v, want the original request batch", skipped)
	}
	for i := range names {
		if skipped[i] != names[i] {
			t.Fatalf("skipped = %v, want %v", skipped, names)
		}
	}

	if _, _, legacyOK := planGPVFaultRecovery(names, badPath, false); legacyOK {
		t.Fatal("legacy GPV must retain its existing failure semantics")
	}
}

func TestDurableObjectPrefix9005IsToleratedAsIncompleteCoverage(t *testing.T) {
	durable := &task.Task{Source: task.TaskSourceParamSync, CommandKey: "param-sync-run-6"}
	if !isToleratedDurableGPVBadPath(durable, "Device.KeepalivedMgmt.VrrpMgmt.", 9005) {
		t.Fatal("durable object-prefix 9005 must not cancel unrelated GPV batches")
	}
	legacy := &task.Task{Source: task.TaskSourceSystem, CommandKey: "sync-gpv-SN-6"}
	if isToleratedDurableGPVBadPath(legacy, "Device.KeepalivedMgmt.VrrpMgmt.", 9005) {
		t.Fatal("legacy recovery semantics must remain unchanged")
	}
}

func TestIsRecoverableGPVBadPath(t *testing.T) {
	cases := []struct {
		name      string
		badPath   string
		faultCode int
		want      bool
	}{
		{
			name:      "具体叶子参数 9005 可恢复",
			badPath:   "Device.WiFi.SSID.7.Enable",
			faultCode: 9005,
			want:      true,
		},
		{
			name:      "对象实例前缀 9005 不做逐个 recovery",
			badPath:   "DeviceGSM.Bts.90.",
			faultCode: 9005,
			want:      false,
		},
		{
			name:      "对象子树前缀 9005 不做 recovery",
			badPath:   "Device.Services.FAPService.2.CellConfig.LTE.RAN.NeighborList.LTECell.",
			faultCode: 9005,
			want:      false,
		},
		{
			name:      "非 9005 不做 GPV recovery",
			badPath:   "Device.WiFi.SSID.7.Enable",
			faultCode: 9002,
			want:      false,
		},
		{
			name:      "空 path 不恢复",
			badPath:   "",
			faultCode: 9005,
			want:      false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := isRecoverableGPVBadPath(tc.badPath, tc.faultCode)
			if got != tc.want {
				t.Errorf("isRecoverableGPVBadPath(%q, %d) = %v, want %v", tc.badPath, tc.faultCode, got, tc.want)
			}
		})
	}
}

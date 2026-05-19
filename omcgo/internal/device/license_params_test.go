package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLicensePartialPrefix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "private X_COM_LICENSE leaf → father object prefix",
			path: "Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Author",
			want: "Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.",
		},
		{
			name: "nested capacity instance under private LICENSE",
			path: "Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}.RemainDays",
			want: "Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.",
		},
		{
			name: "standard LICENSE leaf → father object prefix",
			path: "Device.Services.FAPService.{i}.FAPControl.LTE.LICENSE.Author",
			want: "Device.Services.FAPService.{i}.FAPControl.LTE.LICENSE.",
		},
		{
			name: "private keyword preferred over standard when both present",
			path: "Device.X_COM_LICENSE.LICENSE.Foo",
			want: "Device.X_COM_LICENSE.",
		},
		{
			name: "path without license keyword → empty",
			path: "Device.WiFi.SSID.1.Enabled",
			want: "",
		},
		{
			// 我们认 '.LICENSE.' 为父对象，所以 path 必须有前导 dot+LICENSE
			name: "leading LICENSE without dot prefix returns empty",
			path: "LICENSE.Author",
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := licensePartialPrefix(tc.path)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsLicensePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Author", true},
		{"Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Capacity.1.RemainDays", true},
		{"Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Author", true},
		{"Device.WiFi.SSID.1.Enabled", false},
		{"Device.X_LICENSE_AGREEMENT.Status", false}, // 不含 ".LICENSE." 严格父对象
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, isLicensePath(tc.path))
		})
	}
}

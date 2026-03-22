package pathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeParameterPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		rootVersion string
		want        string
	}{
		{
			name:        "already has Device prefix",
			path:        "Device.ManagementServer.URL",
			rootVersion: "2.8",
			want:        "Device.ManagementServer.URL",
		},
		{
			name:        "already has IGD prefix",
			path:        "InternetGatewayDevice.ManagementServer.URL",
			rootVersion: "1.0",
			want:        "InternetGatewayDevice.ManagementServer.URL",
		},
		{
			name:        "relative path with version 2.x",
			path:        "ManagementServer.URL",
			rootVersion: "2.8",
			want:        "Device.ManagementServer.URL",
		},
		{
			name:        "relative path with version 2.0",
			path:        "ManagementServer.URL",
			rootVersion: "2.0",
			want:        "Device.ManagementServer.URL",
		},
		{
			name:        "relative path with version 2.12",
			path:        "ManagementServer.URL",
			rootVersion: "2.12",
			want:        "Device.ManagementServer.URL",
		},
		{
			name:        "relative path with version 1.x defaults to Device",
			path:        "ManagementServer.URL",
			rootVersion: "1.0",
			want:        "Device.ManagementServer.URL",
		},
		{
			name:        "relative path with empty version defaults to Device",
			path:        "ManagementServer.URL",
			rootVersion: "",
			want:        "Device.ManagementServer.URL",
		},
		{
			name:        "empty path with version",
			path:        "",
			rootVersion: "2.8",
			want:        "Device.",
		},
		{
			name:        "vendor extension path relative",
			path:        "X_VENDOR_Custom.Param",
			rootVersion: "2.8",
			want:        "Device.X_VENDOR_Custom.Param",
		},
		{
			name:        "Device prefix is not confused with DeviceSomething",
			path:        "DeviceInfo.ModelName",
			rootVersion: "2.8",
			want:        "Device.DeviceInfo.ModelName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeParameterPath(tt.path, tt.rootVersion)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsRelativePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "Device prefix is absolute",
			path: "Device.ManagementServer.URL",
			want: false,
		},
		{
			name: "IGD prefix is absolute",
			path: "InternetGatewayDevice.ManagementServer.URL",
			want: false,
		},
		{
			name: "no prefix is relative",
			path: "ManagementServer.URL",
			want: true,
		},
		{
			name: "empty path is relative",
			path: "",
			want: true,
		},
		{
			name: "partial Device prefix is relative",
			path: "Devic.Something",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRelativePath(tt.path))
		})
	}
}

func TestGetRootPrefix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "Device root",
			path: "Device.ManagementServer.URL",
			want: "Device.",
		},
		{
			name: "IGD root",
			path: "InternetGatewayDevice.DeviceInfo.ModelName",
			want: "InternetGatewayDevice.",
		},
		{
			name: "relative path returns empty",
			path: "ManagementServer.URL",
			want: "",
		},
		{
			name: "empty path returns empty",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetRootPrefix(tt.path))
		})
	}
}

func TestStripRootPrefix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "strip Device prefix",
			path: "Device.ManagementServer.URL",
			want: "ManagementServer.URL",
		},
		{
			name: "strip IGD prefix",
			path: "InternetGatewayDevice.DeviceInfo.ModelName",
			want: "DeviceInfo.ModelName",
		},
		{
			name: "relative path unchanged",
			path: "ManagementServer.URL",
			want: "ManagementServer.URL",
		},
		{
			name: "Device root only",
			path: "Device.",
			want: "",
		},
		{
			name: "IGD root only",
			path: "InternetGatewayDevice.",
			want: "",
		},
		{
			name: "empty path unchanged",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StripRootPrefix(tt.path))
		})
	}
}

func TestStripAndNormalize_Roundtrip(t *testing.T) {
	// Stripping and re-normalizing should give back the original path
	original := "Device.ManagementServer.URL"
	stripped := StripRootPrefix(original)
	restored := NormalizeParameterPath(stripped, "2.8")
	assert.Equal(t, original, restored)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "Device.", RootDataModelDevice)
	assert.Equal(t, "InternetGatewayDevice.", RootDataModelIGD)
}

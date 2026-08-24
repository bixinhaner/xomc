package provider

import (
	"os"
	"strings"
	"testing"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

func TestTransferPublicBaseURL(t *testing.T) {
	// 无 OMC_PUBLIC_HOST → 自动探测宿主机 IP 或回退 localhost，应能拼出合法 http URL
	os.Unsetenv("OMC_PUBLIC_HOST")
	os.Unsetenv("OMC_PUBLIC_ACS_PORT")
	got := transferPublicBaseURL()
	if !strings.HasPrefix(got, "http://") || !strings.Contains(got, ":7557") {
		t.Fatalf("default = %q, want http://<host>:7557", got)
	}
	// OMC_PUBLIC_HOST 注入优先
	os.Setenv("OMC_PUBLIC_HOST", "172.19.1.132")
	if got := transferPublicBaseURL(); got != "http://172.19.1.132:7557" {
		t.Fatalf("with host = %q, want http://172.19.1.132:7557", got)
	}
	// 自定义端口
	os.Setenv("OMC_PUBLIC_ACS_PORT", "8081")
	if got := transferPublicBaseURL(); got != "http://172.19.1.132:8081" {
		t.Fatalf("with port = %q, want http://172.19.1.132:8081", got)
	}
	os.Unsetenv("OMC_PUBLIC_HOST")
	os.Unsetenv("OMC_PUBLIC_ACS_PORT")
}

func TestNewSoftwareTransferDefaultsFallsBackToPublicHost(t *testing.T) {
	os.Unsetenv("OMC_PUBLIC_HOST")
	os.Unsetenv("OMC_PUBLIC_ACS_PORT")
	cfg := appconfig.UpgradeConfig{}
	snap := newSoftwareTransferDefaults(cfg)
	if !strings.HasPrefix(snap.Upload.BaseURL, "http://") || !strings.HasSuffix(snap.Upload.BaseURL, ":7557") {
		t.Fatalf("fallback Upload.BaseURL = %q, want http://<host>:7557", snap.Upload.BaseURL)
	}
	if snap.Download.BaseURL != snap.Upload.BaseURL {
		t.Fatalf("fallback Download.BaseURL = %q, want equal to Upload %q", snap.Download.BaseURL, snap.Upload.BaseURL)
	}
	// 显式 yaml 优先
	cfg.ACSUploadBaseURL = "http://app.example.com:8080"
	snap2 := newSoftwareTransferDefaults(cfg)
	if snap2.Upload.BaseURL != "http://app.example.com:8080" {
		t.Fatalf("explicit Upload.BaseURL = %q, want http://app.example.com:8080", snap2.Upload.BaseURL)
	}
	os.Unsetenv("OMC_PUBLIC_HOST")
	os.Unsetenv("OMC_PUBLIC_ACS_PORT")
}

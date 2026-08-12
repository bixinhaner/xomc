package transfercfg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateBaseURL_ProductionRejectsLocalOnlyAddresses(t *testing.T) {
	t.Setenv("OMCGO_ENV", "production")
	t.Setenv("GIN_MODE", "release")

	for _, raw := range []string{
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://169.254.1.10:8080",
		"http://0.0.0.0:8080",
		"http://127.1:8080",
		"http://2130706433:8080",
		"http://0:8080",
		"http://[::1]:8080",
		"http://[fe80::1]:8080",
	} {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateBaseURL(raw); err == nil {
				t.Fatalf("ValidateBaseURL(%q) unexpectedly accepted a production-local address", raw)
			}
		})
	}
}

func TestValidateBaseURL_ProductionAllowsDeviceReachablePrivateAddresses(t *testing.T) {
	t.Setenv("OMCGO_ENV", "production")
	t.Setenv("GIN_MODE", "release")

	for _, raw := range []string{
		"http://10.10.0.1:8080",
		"http://172.17.9.239:8081",
		"http://192.168.1.10:8080",
		"http://[fd00::10]:8080",
	} {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateBaseURL(raw); err != nil {
				t.Fatalf("ValidateBaseURL(%q) returned error: %v", raw, err)
			}
		})
	}
}

func TestValidateBaseURL_DevelopmentAllowsLocalAndPreservesValidHTTPSURL(t *testing.T) {
	t.Setenv("OMCGO_ENV", "dev")
	t.Setenv("GIN_MODE", "debug")

	for _, raw := range []string{
		"http://localhost:8080",
		"https://edge.example.com:9443/custom/transfer/",
	} {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateBaseURL(raw); err != nil {
				t.Fatalf("ValidateBaseURL(%q) returned error: %v", raw, err)
			}
		})
	}
}

func TestValidateBaseURL_RejectsNonAbsoluteOrUnsupportedURL(t *testing.T) {
	t.Setenv("OMCGO_ENV", "production")
	t.Setenv("GIN_MODE", "release")

	for _, raw := range []string{
		"edge.example.com:8080",
		"ftp://edge.example.com/file",
		"/smallcell/FileUploadService",
		"https://user:password@edge.example.com/transfer",
		"https://edge.example.com/transfer?token=secret",
		"https://edge.example.com/transfer#section",
	} {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateBaseURL(raw); err == nil {
				t.Fatalf("ValidateBaseURL(%q) unexpectedly accepted an invalid endpoint", raw)
			}
		})
	}
}

func TestValidateBaseURL_RejectsMalformedPortsAndRawDelimiters(t *testing.T) {
	t.Setenv("OMCGO_ENV", "dev")
	t.Setenv("GIN_MODE", "debug")

	for _, raw := range []string{
		"http://edge.example.com:",
		"https://edge.example.com:",
		"http://edge.example.com:abc",
		"https://edge.example.com:abc",
		"http://edge.example.com:0",
		"https://edge.example.com:0",
		"http://edge.example.com:65536",
		"https://edge.example.com:65536",
		"http://edge.example.com/transfer?",
		"https://edge.example.com/transfer?",
		"http://edge.example.com/transfer#",
		"https://edge.example.com/transfer#",
	} {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateBaseURL(raw); err == nil {
				t.Fatalf("ValidateBaseURL(%q) unexpectedly accepted an invalid endpoint", raw)
			}
		})
	}
}

func TestValidateBaseURL_AllowsBracketedIPv6WithValidPort(t *testing.T) {
	t.Setenv("OMCGO_ENV", "dev")
	t.Setenv("GIN_MODE", "debug")

	for _, raw := range []string{
		"http://[fd00::10]:8080/upload",
		"https://[fd00::10]:8443/download",
	} {
		t.Run(raw, func(t *testing.T) {
			require.NoError(t, ValidateBaseURL(raw))
		})
	}
}

func TestValidateServicePath_AcceptsAbsoluteServicePath(t *testing.T) {
	for _, raw := range []string{
		"/smallcell/FileUploadService",
		"/smallcell/FileDownloadService",
		"/vendor/v1/files",
	} {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateServicePath(raw); err != nil {
				t.Fatalf("ValidateServicePath(%q) returned error: %v", raw, err)
			}
		})
	}
}

func TestValidateServicePath_EmptyUsesConfiguredFallback(t *testing.T) {
	if err := ValidateServicePath(""); err != nil {
		t.Fatalf("ValidateServicePath(empty) returned error: %v", err)
	}
}

func TestValidateServicePath_RejectsAmbiguousOrEscapingPath(t *testing.T) {
	for _, raw := range []string{
		"smallcell/FileUploadService",
		"//edge.example.com/FileUploadService",
		"https://edge.example.com/FileUploadService",
		"/smallcell/../admin",
		"/smallcell/%2e%2e/admin",
		"/smallcell/FileUploadService?token=secret",
		"/smallcell/FileUploadService#section",
		`/smallcell\FileUploadService`,
	} {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateServicePath(raw); err == nil {
				t.Fatalf("ValidateServicePath(%q) unexpectedly accepted an invalid service path", raw)
			}
		})
	}
}

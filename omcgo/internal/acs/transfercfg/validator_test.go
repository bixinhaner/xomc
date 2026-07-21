package transfercfg

import "testing"

func TestValidateBaseURL_ProductionRejectsPrivateAndLocalAddresses(t *testing.T) {
	t.Setenv("OMCGO_ENV", "production")
	t.Setenv("GIN_MODE", "release")

	for _, raw := range []string{
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://169.254.1.10:8080",
		"http://0.0.0.0:8080",
		"http://10.10.0.1:8080",
		"http://172.16.0.1:8080",
		"http://192.168.1.10:8080",
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

func TestValidateServicePath_RejectsAmbiguousOrEscapingPath(t *testing.T) {
	for _, raw := range []string{
		"",
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

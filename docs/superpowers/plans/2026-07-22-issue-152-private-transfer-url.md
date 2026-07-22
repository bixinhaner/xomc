# Issue 152 Private Transfer URL Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore production use of device-reachable private transfer addresses and optional service paths without removing URL structure and path-traversal protections.

**Architecture:** Keep ACS transfer configuration validation at the existing `transfercfg` boundary. Treat reachability as deployment topology rather than an IP-class property, while continuing to reject loopback, link-local, unspecified, malformed numeric, and structurally unsafe endpoints; keep blank configured paths as “use startup fallback.” Exercise the shared URL builder used by log collection and backup, and retain the existing KPI private-host regression test.

**Tech Stack:** Go 1.x, `net/url`, Go `testing`, Testify.

## Global Constraints

- Base-station-reachable RFC1918 IPv4 and IPv6 ULA addresses are valid in production.
- Do not add network probes, DNS resolution, environment bypass switches, or unrelated refactors.
- Preserve HTTP(S)-only, credential/query/fragment, path traversal, and escaping validation.
- Blank upload/download service paths retain the configured startup fallback.
- Do not modify or commit the user's existing `AGENTS.md` or unrelated issue 147 plan.

---

### Task 1: Restore transfer validator business compatibility

**Files:**
- Modify: `omcgo/internal/acs/transfercfg/validator_test.go:5-99`
- Modify: `omcgo/internal/acs/transfercfg/validator.go:13-117`

**Interfaces:**
- Consumes: `ValidateBaseURL(value string) error`, `ValidateServicePath(value string) error`
- Produces: production validation that accepts RFC1918/ULA hosts and blank optional paths while preserving existing structural guards

- [ ] **Step 1: Write failing validator tests**

```go
func TestValidateBaseURL_ProductionAllowsDeviceReachablePrivateAddresses(t *testing.T) {
	t.Setenv("OMCGO_ENV", "production")
	t.Setenv("GIN_MODE", "release")
	for _, raw := range []string{
		"http://10.10.0.1:8080",
		"http://172.17.9.239:8081",
		"http://192.168.1.10:8080",
		"http://[fd00::10]:8080",
	} {
		if err := ValidateBaseURL(raw); err != nil {
			t.Fatalf("ValidateBaseURL(%q) returned error: %v", raw, err)
		}
	}
}

func TestValidateServicePath_EmptyUsesConfiguredFallback(t *testing.T) {
	if err := ValidateServicePath(""); err != nil {
		t.Fatalf("ValidateServicePath(empty) returned error: %v", err)
	}
}
```

- [ ] **Step 2: Run tests and verify the compatibility assertions fail**

Run: `cd omcgo && go test ./internal/acs/transfercfg -run 'TestValidateBaseURL_ProductionAllowsDeviceReachablePrivateAddresses|TestValidateServicePath_EmptyUsesConfiguredFallback' -count=1`

Expected: FAIL because production currently rejects private addresses and `ValidateServicePath("")` returns `service path is required`.

- [ ] **Step 3: Implement the minimal validator correction**

```go
// Remove ip.IsPrivate() from the final return expression in isLocalOnlyHost.
return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()

// At the start of ValidateServicePath, preserve the documented fallback contract.
if value == "" {
	return nil
}
```

Update comments so they state that operator private networks are valid and reachability cannot be inferred from IP address class.

- [ ] **Step 4: Run the focused validator package tests**

Run: `cd omcgo && go test ./internal/acs/transfercfg -count=1`

Expected: PASS, including all existing unsafe URL/path cases.

### Task 2: Cover affected file-transfer consumers and verify backend

**Files:**
- Modify: `omcgo/internal/software/transfer_url_test.go:10-36`
- Modify: `omcgo/internal/backup/executor_test.go:21-43`
- Verify: `omcgo/internal/pm/online_subscriber_test.go:198-218`

**Interfaces:**
- Consumes: `buildTransferUploadURL(baseURL, resolvedTransport string) (string, error)`, `buildBackupUploadURL(upload transfercfg.UploadSettings, spec *BackupTypeSpec, sn, taskID, filename string) (string, error)`
- Produces: regression evidence for log collection, configuration backup, and the existing KPI private-host override

- [ ] **Step 1: Add production-private integration cases**

```go
func TestBuildTransferUploadURL_ProductionAllowsDeviceReachablePrivateBase(t *testing.T) {
	t.Setenv("OMCGO_ENV", "production")
	got, err := buildTransferUploadURL("http://172.17.9.239:8081", "/smallcell/FileUploadService?fileType=LOG")
	require.NoError(t, err)
	require.Equal(t, "http://172.17.9.239:8081/smallcell/FileUploadService?fileType=LOG", got)
}

func TestBuildBackupUploadURL_ProductionAllowsDeviceReachablePrivateBase(t *testing.T) {
	t.Setenv("OMCGO_ENV", "production")
	got, err := buildBackupUploadURL(transfercfg.UploadSettings{BaseURL: "http://172.17.9.239:8081"}, &BackupTypeSpec{URLFileTypeParam: "CONFIGBACKUP_XML"}, "SN100", "task-100", "backup.xml")
	require.NoError(t, err)
	require.Contains(t, got, "http://172.17.9.239:8081/smallcell/FileUploadService")
}
```

- [ ] **Step 2: Run affected package tests**

Run: `cd omcgo && go test ./internal/acs/transfercfg ./internal/software ./internal/backup ./internal/pm -count=1`

Expected: PASS; the PM package retains `Test_OnlineSubscriber_BaseURLResolverOverridesHost` with `172.19.1.173`.

- [ ] **Step 3: Format and run backend verification**

Run: `gofmt -w omcgo/internal/acs/transfercfg/validator.go omcgo/internal/acs/transfercfg/validator_test.go omcgo/internal/software/transfer_url_test.go omcgo/internal/backup/executor_test.go`

Run: `cd omcgo && go build ./... && go test ./...`

Expected: both commands exit successfully. If sandboxed tests cannot bind local listeners, rerun the same command with the required permission and distinguish environment failures from code failures.

- [ ] **Step 4: Review the scoped diff**

Run: `git diff --check && git diff -- omcgo/internal/acs/transfercfg/validator.go omcgo/internal/acs/transfercfg/validator_test.go omcgo/internal/software/transfer_url_test.go omcgo/internal/backup/executor_test.go docs/superpowers/plans/2026-07-22-issue-152-private-transfer-url.md`

Expected: only issue 152 compatibility, regression tests, and this plan are changed.

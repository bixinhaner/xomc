package systimezone

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteRuntimeTimezone(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}

	path := filepath.Join(t.TempDir(), "nested", "system-timezone")
	if err := WriteRuntimeTimezone(path, loc); err != nil {
		t.Fatalf("WriteRuntimeTimezone: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read runtime timezone: %v", err)
	}
	if got := strings.TrimSpace(string(content)); got != "Asia/Shanghai" {
		t.Fatalf("runtime timezone = %q, want Asia/Shanghai", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat runtime timezone: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("runtime timezone mode = %o, want 644", got)
	}
}

func TestWriteRuntimeTimezone_NilFallsBackToUTC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "system-timezone")
	if err := WriteRuntimeTimezone(path, nil); err != nil {
		t.Fatalf("WriteRuntimeTimezone: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read runtime timezone: %v", err)
	}
	if got := strings.TrimSpace(string(content)); got != DefaultTimezone {
		t.Fatalf("runtime timezone = %q, want %s", got, DefaultTimezone)
	}
}

func TestWriteRuntimeTimezoneName_EmptyMeansSystemTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "system-timezone")
	if err := WriteRuntimeTimezoneName(path, ""); err != nil {
		t.Fatalf("WriteRuntimeTimezoneName: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read runtime timezone: %v", err)
	}
	if got := strings.TrimSpace(string(content)); got != RuntimeSystemTimezone {
		t.Fatalf("runtime timezone = %q, want %s", got, RuntimeSystemTimezone)
	}
}

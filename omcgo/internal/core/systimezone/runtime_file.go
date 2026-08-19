package systimezone

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RuntimeTimezonePath is the shared path used by containers that need the OMC timezone.
const RuntimeTimezonePath = "/var/lib/omcgo/timezone/system-timezone"

// RuntimeSystemTimezone marks an empty OMC timezone, which means use system local time.
const RuntimeSystemTimezone = "SYSTEM"

// WriteRuntimeTimezone atomically publishes a validated timezone name for local consumers.
func WriteRuntimeTimezone(path string, loc *time.Location) error {
	name := DefaultTimezone
	if loc != nil && strings.TrimSpace(loc.String()) != "" {
		name = loc.String()
	}
	return WriteRuntimeTimezoneName(path, name)
}

// WriteRuntimeTimezoneName atomically publishes a runtime timezone marker.
func WriteRuntimeTimezoneName(path, name string) error {
	if path == "" {
		path = RuntimeTimezonePath
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = RuntimeSystemTimezone
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create timezone runtime directory %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".system-timezone.*")
	if err != nil {
		return fmt.Errorf("create timezone runtime file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	removeTemp := func() { _ = os.Remove(tmpPath) }

	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		removeTemp()
		return fmt.Errorf("chmod timezone runtime file: %w", err)
	}
	if _, err := tmp.WriteString(name + "\n"); err != nil {
		_ = tmp.Close()
		removeTemp()
		return fmt.Errorf("write timezone runtime file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		removeTemp()
		return fmt.Errorf("sync timezone runtime file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		removeTemp()
		return fmt.Errorf("close timezone runtime file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		removeTemp()
		return fmt.Errorf("publish timezone runtime file %s: %w", path, err)
	}
	return nil
}

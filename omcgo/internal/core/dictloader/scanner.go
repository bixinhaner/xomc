// Package dictloader provides shared infrastructure for startup-time XML
// dictionary loading and cross-instance cache invalidation.
//
// Each business domain (param-model / indicator / alarm-definition / product)
// implements the Loader interface; this package supplies:
//
//   - Scanner       — directory scanning with mtime+size fingerprints
//   - Registry      — Loader lifecycle orchestration with bounded parallelism
//   - CacheVersion  — Redis-backed counter + L1 invalidation watchdog
//   - Report        — per-Loader load result aggregation
//
// Concrete XML parsing and DB persistence are owned by each domain
// (T-0098 P1-06).
package dictloader

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileFingerprint identifies an XML file by its modification time and size.
// Two scans compare fingerprints to detect added/removed/modified files
// without parsing content.
type FileFingerprint struct {
	Path    string
	Size    int64
	ModTime time.Time
}

// Scanner walks a base directory tree looking for XML files.
type Scanner struct {
	BaseDir string
}

// NewScanner returns a Scanner rooted at baseDir.
func NewScanner(baseDir string) *Scanner {
	return &Scanner{BaseDir: baseDir}
}

// ScanDirectory returns sorted FileFingerprints for *.xml files (case-insensitive
// extension) under filepath.Join(s.BaseDir, relDir). When whitelist is non-empty,
// only files whose basename appears in the whitelist are returned (case-sensitive,
// extension included). Hidden entries (basename starting with ".") and
// subdirectories are skipped — the scan is non-recursive.
func (s *Scanner) ScanDirectory(relDir string, whitelist []string) ([]FileFingerprint, error) {
	fullDir := filepath.Join(s.BaseDir, relDir)
	entries, err := os.ReadDir(fullDir)
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", fullDir, err)
	}

	var allow map[string]struct{}
	if len(whitelist) > 0 {
		allow = make(map[string]struct{}, len(whitelist))
		for _, name := range whitelist {
			allow[name] = struct{}{}
		}
	}

	out := make([]FileFingerprint, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.EqualFold(filepath.Ext(name), ".xml") {
			continue
		}
		if allow != nil {
			if _, ok := allow[name]; !ok {
				continue
			}
		}
		info, err := e.Info()
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", filepath.Join(fullDir, name), err)
		}
		out = append(out, FileFingerprint{
			Path:    filepath.Join(fullDir, name),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// DiffFingerprints reports the differences between a previous and current
// snapshot. A file is considered modified when either Size or ModTime changes.
// Both inputs may be in any order; matching is by Path.
func DiffFingerprints(prev, cur []FileFingerprint) (added, removed, modified []FileFingerprint) {
	prevByPath := make(map[string]FileFingerprint, len(prev))
	for _, fp := range prev {
		prevByPath[fp.Path] = fp
	}
	seen := make(map[string]struct{}, len(cur))
	for _, fp := range cur {
		seen[fp.Path] = struct{}{}
		old, ok := prevByPath[fp.Path]
		switch {
		case !ok:
			added = append(added, fp)
		case old.Size != fp.Size || !old.ModTime.Equal(fp.ModTime):
			modified = append(modified, fp)
		}
	}
	for _, fp := range prev {
		if _, ok := seen[fp.Path]; !ok {
			removed = append(removed, fp)
		}
	}
	return added, removed, modified
}

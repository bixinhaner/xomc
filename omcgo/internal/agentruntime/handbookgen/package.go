package handbookgen

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const handbookPackageRoot = "references"

// BuildPackage creates a deterministic archive containing only handbook data.
// Executable Skill instructions are intentionally excluded from OMC-delivered artifacts.
func BuildPackage(skillRoot string) ([]byte, error) {
	referencesDir := filepath.Join(skillRoot, handbookPackageRoot)
	entries := make([]string, 0, 1024)
	files := make(map[string][]byte, 1024)
	err := filepath.WalkDir(referencesDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("handbook package contains non-regular file %s", path)
		}
		relative, err := filepath.Rel(skillRoot, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if !strings.HasPrefix(name, handbookPackageRoot+"/") {
			return fmt.Errorf("handbook file escapes references: %s", name)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read handbook file %s: %w", relative, err)
		}
		entries = append(entries, name)
		files[name] = raw
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan handbook references: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("handbook references are empty")
	}
	sort.Strings(entries)
	return buildPackageFiles(entries, files)
}

func buildPackageFiles(entries []string, files map[string][]byte) ([]byte, error) {
	var output bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&output, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("create handbook gzip writer: %w", err)
	}
	gzipWriter.Header.ModTime = time.Unix(0, 0).UTC()
	tarWriter := tar.NewWriter(gzipWriter)
	for _, relative := range entries {
		name := filepath.ToSlash(relative)
		if !strings.HasPrefix(name, handbookPackageRoot+"/") {
			return nil, closePackageWriters(tarWriter, gzipWriter, fmt.Errorf("handbook file escapes references: %s", name))
		}
		raw, ok := files[name]
		if !ok {
			return nil, closePackageWriters(tarWriter, gzipWriter, fmt.Errorf("handbook file %s is missing", name))
		}
		header := &tar.Header{
			Name:       name,
			Mode:       0o644,
			Size:       int64(len(raw)),
			ModTime:    time.Unix(0, 0).UTC(),
			AccessTime: time.Unix(0, 0).UTC(),
			ChangeTime: time.Unix(0, 0).UTC(),
			Typeflag:   tar.TypeReg,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, closePackageWriters(tarWriter, gzipWriter, fmt.Errorf("write handbook header %s: %w", name, err))
		}
		if _, err := tarWriter.Write(raw); err != nil {
			return nil, closePackageWriters(tarWriter, gzipWriter, fmt.Errorf("write handbook file %s: %w", name, err))
		}
	}
	if err := tarWriter.Close(); err != nil {
		_ = gzipWriter.Close()
		return nil, fmt.Errorf("close handbook tar writer: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close handbook gzip writer: %w", err)
	}
	return output.Bytes(), nil
}

func WritePackage(skillRoot, packagePath string) error {
	raw, err := BuildPackage(skillRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(packagePath), 0o755); err != nil {
		return fmt.Errorf("create handbook package directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(packagePath), ".handbook-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary handbook package: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(raw); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary handbook package: %w", err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set handbook package permissions: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary handbook package: %w", err)
	}
	if err := os.Rename(temporaryPath, packagePath); err != nil {
		return fmt.Errorf("publish handbook package: %w", err)
	}
	return nil
}

func CheckPackage(skillRoot, packagePath string) error {
	expected, err := BuildPackage(skillRoot)
	if err != nil {
		return err
	}
	actual, err := os.ReadFile(packagePath)
	if err != nil {
		return fmt.Errorf("read handbook package: %w", err)
	}
	if !bytes.Equal(actual, expected) {
		return fmt.Errorf("handbook package is stale; regenerate it from %s", filepath.Join(skillRoot, handbookPackageRoot))
	}
	return nil
}

func closePackageWriters(tarWriter *tar.Writer, gzipWriter *gzip.Writer, cause error) error {
	_ = tarWriter.Close()
	_ = gzipWriter.Close()
	return cause
}

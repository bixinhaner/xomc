package handbookgen

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPackageIsDeterministicAndDataOnly(t *testing.T) {
	skillRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(skillRoot, "references", "api-docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte("untrusted instructions"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillRoot, "references", "manifest.json"), []byte(`{"schemaVersion":"1.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillRoot, "references", "api-docs", "get.devices.json"), []byte(`{"operationId":"get.devices"}`), 0o644))

	first, err := BuildPackage(skillRoot)
	require.NoError(t, err)
	second, err := BuildPackage(skillRoot)
	require.NoError(t, err)
	require.Equal(t, first, second)

	gzipReader, err := gzip.NewReader(bytes.NewReader(first))
	require.NoError(t, err)
	tarReader := tar.NewReader(gzipReader)
	files := map[string]string{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		raw, err := io.ReadAll(tarReader)
		require.NoError(t, err)
		files[header.Name] = string(raw)
	}
	require.NoError(t, gzipReader.Close())
	require.Contains(t, files, "references/manifest.json")
	require.Contains(t, files, "references/api-docs/get.devices.json")
	require.NotContains(t, files, "SKILL.md")
}

func TestWriteAndCheckPackageDetectsStaleContent(t *testing.T) {
	skillRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(skillRoot, "references"), 0o755))
	manifestPath := filepath.Join(skillRoot, "references", "manifest.json")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`{"catalogVersion":"one"}`), 0o644))
	packagePath := filepath.Join(t.TempDir(), "handbook.tar.gz")

	require.NoError(t, WritePackage(skillRoot, packagePath))
	require.NoError(t, CheckPackage(skillRoot, packagePath))
	require.NoError(t, os.WriteFile(manifestPath, []byte(`{"catalogVersion":"two"}`), 0o644))
	require.ErrorContains(t, CheckPackage(skillRoot, packagePath), "stale")
}

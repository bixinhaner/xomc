package dictloader

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanner_ScanDirectory(t *testing.T) {
	base := t.TempDir()
	sub := filepath.Join(base, "domain")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	write := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(sub, name), []byte(content), 0o644))
	}
	write("alpha.xml", "alpha")
	write("beta.xml", "beta")
	write("gamma.xml", "gamma")
	write(".hidden.xml", "hidden")
	write("notes.txt", "notes")
	require.NoError(t, os.MkdirAll(filepath.Join(sub, "nested"), 0o755))

	sc := NewScanner(base)

	t.Run("no whitelist returns all xml sorted", func(t *testing.T) {
		fps, err := sc.ScanDirectory("domain", nil)
		require.NoError(t, err)
		require.Len(t, fps, 3)
		assert.Equal(t, filepath.Join(sub, "alpha.xml"), fps[0].Path)
		assert.Equal(t, filepath.Join(sub, "beta.xml"), fps[1].Path)
		assert.Equal(t, filepath.Join(sub, "gamma.xml"), fps[2].Path)
		assert.Greater(t, fps[0].Size, int64(0))
	})

	t.Run("whitelist filters basenames", func(t *testing.T) {
		fps, err := sc.ScanDirectory("domain", []string{"alpha.xml", "gamma.xml", "missing.xml"})
		require.NoError(t, err)
		require.Len(t, fps, 2)
		assert.Equal(t, "alpha.xml", filepath.Base(fps[0].Path))
		assert.Equal(t, "gamma.xml", filepath.Base(fps[1].Path))
	})

	t.Run("missing directory returns error", func(t *testing.T) {
		_, err := sc.ScanDirectory("does-not-exist", nil)
		require.Error(t, err)
	})

	t.Run("uppercase XML extension included", func(t *testing.T) {
		write("CAPS.XML", "caps")
		t.Cleanup(func() { _ = os.Remove(filepath.Join(sub, "CAPS.XML")) })
		fps, err := sc.ScanDirectory("domain", nil)
		require.NoError(t, err)
		var names []string
		for _, fp := range fps {
			names = append(names, filepath.Base(fp.Path))
		}
		assert.Contains(t, names, "CAPS.XML")
	})

	t.Run("hidden files and subdirectories skipped", func(t *testing.T) {
		fps, err := sc.ScanDirectory("domain", nil)
		require.NoError(t, err)
		for _, fp := range fps {
			base := filepath.Base(fp.Path)
			assert.False(t, base == ".hidden.xml")
			assert.False(t, base == "nested")
		}
	})
}

func TestDiffFingerprints(t *testing.T) {
	now := time.Now()
	prev := []FileFingerprint{
		{Path: "/a.xml", Size: 10, ModTime: now},
		{Path: "/b.xml", Size: 20, ModTime: now},
		{Path: "/c.xml", Size: 30, ModTime: now},
	}
	cur := []FileFingerprint{
		{Path: "/a.xml", Size: 10, ModTime: now},
		{Path: "/b.xml", Size: 25, ModTime: now},
		{Path: "/d.xml", Size: 40, ModTime: now},
	}

	added, removed, modified := DiffFingerprints(prev, cur)
	require.Len(t, added, 1)
	assert.Equal(t, "/d.xml", added[0].Path)
	require.Len(t, removed, 1)
	assert.Equal(t, "/c.xml", removed[0].Path)
	require.Len(t, modified, 1)
	assert.Equal(t, "/b.xml", modified[0].Path)
}

func TestDiffFingerprints_ModTimeChange(t *testing.T) {
	base := time.Now()
	prev := []FileFingerprint{{Path: "/a.xml", Size: 10, ModTime: base}}
	cur := []FileFingerprint{{Path: "/a.xml", Size: 10, ModTime: base.Add(time.Second)}}

	_, _, modified := DiffFingerprints(prev, cur)
	require.Len(t, modified, 1)
	assert.Equal(t, "/a.xml", modified[0].Path)
}

func TestDiffFingerprints_NoChange(t *testing.T) {
	now := time.Now()
	snap := []FileFingerprint{{Path: "/a.xml", Size: 10, ModTime: now}}
	added, removed, modified := DiffFingerprints(snap, snap)
	assert.Empty(t, added)
	assert.Empty(t, removed)
	assert.Empty(t, modified)
}

func TestDiffFingerprints_AllRemoved(t *testing.T) {
	now := time.Now()
	prev := []FileFingerprint{
		{Path: "/a.xml", Size: 10, ModTime: now},
		{Path: "/b.xml", Size: 20, ModTime: now},
	}
	added, removed, modified := DiffFingerprints(prev, nil)
	assert.Empty(t, added)
	assert.Empty(t, modified)
	assert.Len(t, removed, 2)
}

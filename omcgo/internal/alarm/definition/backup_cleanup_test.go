package definition

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClassifyAlarmBackupFile 覆盖文件名分类规则。
func TestClassifyAlarmBackupFile(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		wantKind string
		wantOK   bool
	}{
		{"deleted", "ENB.xml.deleted.20260101030405", "deleted", true},
		{"bak", "ENB.xml.bak.20260601000000", "bak", true},
		{"tmp", "ENB.xml.tmp.1234567890", "tmp", true},
		{"normal xml", "ENB.xml", "", false},
		{"sidecar not a backup", "ENB.xml.custom", "", false},
		{"short ts", "ENB.xml.deleted.2026", "", false},
		{"empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, ok := classifyAlarmBackupFile(tc.filename)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantKind, kind)
		})
	}
}

func TestBackupCleanup_NoDir(t *testing.T) {
	bc := NewBackupCleanup(filepath.Join(t.TempDir(), "no-such"), 30, nil, nil)
	swept, err := bc.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, swept)
}

// TestBackupCleanup_ExpiryAndOrphanSidecar 验证:
//   - 过期 .deleted/.bak 被清,未过期保留
//   - 孤儿 sidecar(X.xml.custom 而 X.xml 不在)被清
//   - 仍有主文件的 sidecar 保留;合法 X.xml 保留
func TestBackupCleanup_ExpiryAndOrphanSidecar(t *testing.T) {
	dir := t.TempDir()
	write := func(name string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte{}, 0o644))
	}

	// fixed "now" = 2026-06-05 00:00:00 local
	now := time.Date(2026, 6, 5, 0, 0, 0, 0, time.Local)

	// 过期备份(35 天前)
	write("OLD.xml.deleted." + now.Add(-35*24*time.Hour).Format("20060102150405"))
	write("OLD.xml.bak." + now.Add(-40*24*time.Hour).Format("20060102150405"))
	// 未过期备份(5 天前)
	write("NEW.xml.deleted." + now.Add(-5*24*time.Hour).Format("20060102150405"))

	// 孤儿 sidecar(无对应 .xml)
	write("GONE.xml" + CustomMarkerSuffix)
	// 有主文件的 sidecar(应保留)
	write("ALIVE.xml")
	write("ALIVE.xml" + CustomMarkerSuffix)
	// 合法 XML(应保留)
	write("BUILTIN.xml")

	bc := NewBackupCleanup(dir, 30, nil, nil)
	bc.now = func() time.Time { return now }

	swept, err := bc.Run(context.Background())
	require.NoError(t, err)
	// 清掉:OLD.deleted, OLD.bak, GONE sidecar = 3
	assert.Equal(t, 3, swept)

	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	assert.False(t, exists("GONE.xml"+CustomMarkerSuffix), "孤儿 sidecar 应被清")
	assert.True(t, exists("ALIVE.xml"+CustomMarkerSuffix), "有主文件的 sidecar 应保留")
	assert.True(t, exists("ALIVE.xml"))
	assert.True(t, exists("BUILTIN.xml"))
	assert.True(t, exists("NEW.xml.deleted."+now.Add(-5*24*time.Hour).Format("20060102150405")), "未过期备份保留")
}

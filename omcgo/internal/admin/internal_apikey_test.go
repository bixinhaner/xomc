package admin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_writeAPIKeyAtomic_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", ".api-key")

	require.NoError(t, writeAPIKeyAtomic(path, "omk_abcdef0123456789"))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "omk_abcdef0123456789", string(got))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(internalAPIKeyFileMode), info.Mode().Perm(),
		"file should be 0640")

	dirInfo, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())
}

func Test_writeAPIKeyAtomic_Overwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".api-key")

	require.NoError(t, writeAPIKeyAtomic(path, "first"))
	require.NoError(t, writeAPIKeyAtomic(path, "second"))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "second", string(got))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no leftover .tmp file allowed")
}

func Test_LoadInternalAPIKey_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".api-key")
	require.NoError(t, os.WriteFile(path, []byte("  omk_xyz  \n"), 0o600))

	key, err := LoadInternalAPIKey(path)
	require.NoError(t, err)
	assert.Equal(t, "omk_xyz", key)
}

func Test_LoadInternalAPIKey_MissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent")

	key, err := LoadInternalAPIKey(path)
	require.NoError(t, err, "missing file is not an error — caller falls back")
	assert.Empty(t, key)
}

func Test_LoadInternalAPIKey_DefaultPathWhenEmpty(t *testing.T) {
	// 默认路径在测试环境一般不存在,应返回空串无错误
	key, err := LoadInternalAPIKey("")
	require.NoError(t, err)
	// 不强断言空 — 测试机器可能恰好挂了文件;但 err 必须 nil
	_ = key
}

func Test_Constants_Stable(t *testing.T) {
	// 防止未来手抖改了 UUID / name —— DB 迁移与代码必须同步
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", SystemInternalUserID,
		"must match seed/000204_system_internal_user.sql")
	assert.Equal(t, "omc-internal", InternalAPIKeyName)
	assert.Equal(t, "/var/lib/omcgo/secrets/.api-key", DefaultInternalAPIKeyPath)
}

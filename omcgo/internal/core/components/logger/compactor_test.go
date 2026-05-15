package logger

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// truncateToMinute 单测：lumberjack 秒.毫秒 → 分钟精度。
func TestTruncateToMinute(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"raw log second precision", "/var/log/acs-2026-05-15T03-55-16.790.log", "/var/log/acs-2026-05-15T03-55.log"},
		{"gz second precision", "/var/log/acs-2026-05-15T03-55-16.790.log.gz", "/var/log/acs-2026-05-15T03-55.log.gz"},
		{"already minute precision raw", "/var/log/acs-2026-05-15T03-55.log", "/var/log/acs-2026-05-15T03-55.log"},
		{"already minute precision gz", "/var/log/acs-2026-05-15T03-55.log.gz", "/var/log/acs-2026-05-15T03-55.log.gz"},
		{"with collision suffix unchanged", "/var/log/acs-2026-05-15T03-55-1.log", "/var/log/acs-2026-05-15T03-55-1.log"},
		{"unrelated filename unchanged", "/var/log/something-else.log", "/var/log/something-else.log"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, truncateToMinute(tc.in))
		})
	}
}

// gzipFile：压缩成功 → .gz 内容正确 + 源文件已删。
func TestGzipFile_Success(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.log")
	content := []byte("hello\nworld\n这是中文\n")
	require.NoError(t, os.WriteFile(src, content, 0o644))

	require.NoError(t, gzipFile(src))

	// 源文件已删
	_, err := os.Stat(src)
	assert.True(t, os.IsNotExist(err), "源文件应被删除")

	// .gz 存在 + 解压后内容一致
	gzPath := src + ".gz"
	f, err := os.Open(gzPath)
	require.NoError(t, err)
	defer f.Close()

	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	got, err := io.ReadAll(gz)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

// gzipFile：源文件不存在 → 返错，不留 .gz 半成品。
func TestGzipFile_MissingSource(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "ghost.log")

	err := gzipFile(missing)
	assert.Error(t, err)
	_, err = os.Stat(missing + ".gz")
	assert.True(t, os.IsNotExist(err), "失败时不应留 .gz 半成品")
}

// safeRename：目标不存在 → 直接 rename。
func TestSafeRename_NoCollision(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.log")
	require.NoError(t, os.WriteFile(src, []byte("x"), 0o644))

	target := filepath.Join(dir, "target.log")
	final, ok := safeRename(src, target)
	assert.True(t, ok)
	assert.Equal(t, target, final)
	_, err := os.Stat(target)
	assert.NoError(t, err)
}

// safeRename：目标已存在 → 附加 -N 后缀避免覆盖。
func TestSafeRename_CollisionAppendsSuffix(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "x.log")
	require.NoError(t, os.WriteFile(target, []byte("existing"), 0o644)) // 预占

	src := filepath.Join(dir, "src.log")
	require.NoError(t, os.WriteFile(src, []byte("new"), 0o644))

	final, ok := safeRename(src, target)
	assert.True(t, ok)
	assert.Equal(t, filepath.Join(dir, "x-1.log"), final, "首次冲突应落在 x-1.log")

	// 原 target 内容不变
	got, _ := os.ReadFile(target)
	assert.Equal(t, "existing", string(got))
}

// compactOnce 整体场景：写若干 mock 归档，验证三档分流。
// 文件 mtime 通过 os.Chtimes 模拟"老"文件。
func TestCompactOnce_KeepUncompressedAndExpire(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "acs.log")

	now := time.Now()

	// 准备 5 个归档（最新→最旧）+ 1 个超期：
	//   0: 2 min 前（最新，应保持 .log）
	//   1: 5 min 前（应保持 .log）
	//   2: 10 min 前（keepUncompressed=2 → 应压缩）
	//   3: 1 hour 前（应压缩）
	//   4: 6 days 前（应压缩，未超期）
	//   5: 8 days 前（应整删）
	files := []struct {
		name    string
		ageBack time.Duration
		gz      bool
	}{
		{"acs-2026-05-15T10-58-12.345.log", 2 * time.Minute, false},
		{"acs-2026-05-15T10-55-30.111.log", 5 * time.Minute, false},
		{"acs-2026-05-15T10-50-00.000.log", 10 * time.Minute, false},
		{"acs-2026-05-15T10-00-00.000.log", 1 * time.Hour, false},
		{"acs-2026-05-09T11-00-00.000.log", 6 * 24 * time.Hour, false},
		{"acs-2026-05-07T11-00-00.000.log", 8 * 24 * time.Hour, false},
	}
	for _, f := range files {
		full := filepath.Join(dir, f.name)
		require.NoError(t, os.WriteFile(full, []byte("logline\n"), 0o644))
		mt := now.Add(-f.ageBack)
		require.NoError(t, os.Chtimes(full, mt, mt))
	}

	compactOnce(logPath, 2 /*keepUncompressed*/, 7*24*time.Hour /*maxAge*/)

	// 收集 dir 下所有文件
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	// 期望：
	//   * 最新 2 个 .log（rename 到分钟精度）
	//   * 3 个 .log.gz（压缩，分钟精度）
	//   * 1 个被删（超 7 天）

	// 计数 .log 与 .log.gz
	var rawLogs, gzLogs int
	for _, n := range names {
		switch {
		case strings.HasSuffix(n, ".log.gz"):
			gzLogs++
		case strings.HasSuffix(n, ".log"):
			rawLogs++
		}
	}
	assert.Equal(t, 2, rawLogs, "应保留 2 个未压缩 .log")
	assert.Equal(t, 3, gzLogs, "应有 3 个压缩 .log.gz")

	// 验证最新 2 个已 rename 到分钟精度（无秒.毫秒部分）
	for _, n := range names {
		if !strings.HasSuffix(n, ".log") || strings.HasSuffix(n, ".log.gz") {
			continue
		}
		assert.Regexp(t, `^acs-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}\.log$`, n,
			"未压缩归档名应为分钟精度（无秒.毫秒），实际 %s", n)
	}

	// 验证压缩文件名也是分钟精度
	for _, n := range names {
		if !strings.HasSuffix(n, ".log.gz") {
			continue
		}
		assert.Regexp(t, `^acs-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}\.log\.gz$`, n,
			"压缩归档名应为分钟精度（无秒.毫秒），实际 %s", n)
	}

	// 验证超 7 天的文件确已删除（原名不应存在）
	_, err = os.Stat(filepath.Join(dir, "acs-2026-05-07T11-00-00.000.log"))
	assert.True(t, os.IsNotExist(err), "8 天前的归档应被删")
}

// compactOnce 空目录场景：不 panic。
func TestCompactOnce_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "acs.log")
	// 不创建任何归档
	assert.NotPanics(t, func() {
		compactOnce(logPath, 10, 7*24*time.Hour)
	})
}

// compactOnce active log 不被误删/误压：active 文件名（无 dash 时间戳）glob 不命中。
func TestCompactOnce_DoesNotTouchActiveLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "acs.log")
	require.NoError(t, os.WriteFile(logPath, []byte("active content\n"), 0o644))

	compactOnce(logPath, 10, 7*24*time.Hour)

	info, err := os.Stat(logPath)
	require.NoError(t, err, "active 文件不应被动")
	assert.Equal(t, int64(len("active content\n")), info.Size())
}

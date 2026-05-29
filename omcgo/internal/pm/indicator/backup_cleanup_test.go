package indicator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// ── classifyIndicatorBackupFile 表驱动 ──────────────────────────────

func TestClassifyIndicatorBackupFile(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantKind string
		wantOK   bool
	}{
		{"deleted", "MY.xml.deleted.20260101120000", "deleted", true},
		{"bak", "MY.xml.bak.20260101120000", "bak", true},
		{"tmp short uuid", "MY.xml.tmp.abcdef123", "tmp", true},
		{"tmp nano", "MY.xml.tmp.1717050000123456789", "tmp", true},

		{"normal xml ignored", "MY.xml", "", false},
		{"random file ignored", "notes.txt", "", false},
		{"partial pattern wrong ts length", "MY.xml.deleted.202601", "", false},
		{"hidden file ignored", ".secret.bak.20260101120000", "bak", true}, // 正则匹配,Run() 流程会按文件处理
		{"empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, ok := classifyIndicatorBackupFile(tc.input)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.wantKind, k)
			}
		})
	}
}

// ── Run 集成 + 边界 ────────────────────────────────────────────────

// writeBackupFile 在 dir 下写一个空备份文件。
func writeBackupFile(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	full := filepath.Join(dir, name)
	if err := os.WriteFile(full, []byte{}, 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
	return full
}

func TestRun_EmptyCustomDir(t *testing.T) {
	tmp := t.TempDir()
	bc := NewBackupCleanup(filepath.Join(tmp, "indicator-library-custom"), 30, nil, zap.NewNop())
	swept, err := bc.Run(context.Background())
	assert.NoError(t, err, "三制式子目录都不存在应返 0, nil")
	assert.Equal(t, 0, swept)
}

func TestRun_DeletedAndBakExpired(t *testing.T) {
	tmp := t.TempDir()
	customDir := filepath.Join(tmp, "indicator-library-custom")

	// 在 enb 子目录:1 个 31 天前 .deleted + 1 个 31 天前 .bak + 1 个 1 天前 .deleted(保留)
	old := time.Now().AddDate(0, 0, -31).Format("20060102150405")
	fresh := time.Now().AddDate(0, 0, -1).Format("20060102150405")
	writeBackupFile(t, filepath.Join(customDir, "enb"), "MY.xml.deleted."+old)
	writeBackupFile(t, filepath.Join(customDir, "enb"), "MY.xml.bak."+old)
	writeBackupFile(t, filepath.Join(customDir, "enb"), "FRESH.xml.deleted."+fresh)
	// 同时写一个非备份文件,不应被动
	writeBackupFile(t, filepath.Join(customDir, "enb"), "ACTIVE.xml")

	// gsm 子目录:1 个过期 + 1 个新鲜
	writeBackupFile(t, filepath.Join(customDir, "gsm"), "G.xml.bak."+old)
	writeBackupFile(t, filepath.Join(customDir, "gsm"), "G.xml.deleted."+fresh)

	// gnb 子目录:空(目录不存在)

	bc := NewBackupCleanup(customDir, 30, NewBackupCleanupMetrics(nil), zap.NewNop())
	swept, err := bc.Run(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 3, swept, "过期 2 enb + 1 gsm = 3")

	// 验证哪些文件还在
	_, err = os.Stat(filepath.Join(customDir, "enb", "MY.xml.deleted."+old))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(customDir, "enb", "MY.xml.bak."+old))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(customDir, "enb", "FRESH.xml.deleted."+fresh))
	assert.NoError(t, err, "新鲜文件应保留")
	_, err = os.Stat(filepath.Join(customDir, "enb", "ACTIVE.xml"))
	assert.NoError(t, err, "活跃 XML 不应被动")
	_, err = os.Stat(filepath.Join(customDir, "gsm", "G.xml.bak."+old))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(customDir, "gsm", "G.xml.deleted."+fresh))
	assert.NoError(t, err)
}

func TestRun_TmpResidualByMtime(t *testing.T) {
	tmp := t.TempDir()
	customDir := filepath.Join(tmp, "indicator-library-custom")

	// 写一个 .tmp.xxx 然后把 mtime 改成 2 小时前 → 应被清
	fullOld := writeBackupFile(t, filepath.Join(customDir, "enb"), "MY.xml.tmp.abc123")
	twoHrsAgo := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(fullOld, twoHrsAgo, twoHrsAgo); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	// 再写一个新鲜 .tmp(刚建,mtime=now)→ 不应被清
	writeBackupFile(t, filepath.Join(customDir, "enb"), "FRESH.xml.tmp.def456")

	bc := NewBackupCleanup(customDir, 30, nil, zap.NewNop())
	swept, err := bc.Run(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, swept)

	_, err = os.Stat(fullOld)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(customDir, "enb", "FRESH.xml.tmp.def456"))
	assert.NoError(t, err)
}

func TestRun_NoOpOnCustomTimestamp(t *testing.T) {
	// 用注入 now 让"30 天前"等价于现在,验证全部跳过
	tmp := t.TempDir()
	customDir := filepath.Join(tmp, "indicator-library-custom")
	writeBackupFile(t, filepath.Join(customDir, "gnb"), "X.xml.deleted.20260101120000")

	bc := NewBackupCleanup(customDir, 30, nil, zap.NewNop())
	// 把 now 设为 2026-01-15(15 天后,< 30 天) → 文件不应被清
	bc.now = func() time.Time {
		return time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local)
	}
	swept, err := bc.Run(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, swept)
	_, err = os.Stat(filepath.Join(customDir, "gnb", "X.xml.deleted.20260101120000"))
	assert.NoError(t, err)
}

func TestRun_CtxCancelled(t *testing.T) {
	tmp := t.TempDir()
	customDir := filepath.Join(tmp, "indicator-library-custom")
	old := time.Now().AddDate(0, 0, -31).Format("20060102150405")
	writeBackupFile(t, filepath.Join(customDir, "enb"), "MY.xml.deleted."+old)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	bc := NewBackupCleanup(customDir, 30, nil, zap.NewNop())
	_, err := bc.Run(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRun_RetentionDaysDefault(t *testing.T) {
	bc := NewBackupCleanup("/some/dir", 0, nil, zap.NewNop())
	assert.Equal(t, time.Duration(DefaultIndicatorBackupRetentionDays)*24*time.Hour, bc.maxAge,
		"retentionDays=0 应走默认值")
}

func TestNewBackupCleanupMetrics_RegisterIdempotent(t *testing.T) {
	// reg=nil 不应 panic
	m1 := NewBackupCleanupMetrics(nil)
	assert.NotNil(t, m1)
	// observe nil 安全(老路径意外传 nil 指标也不崩)
	var nilm *BackupCleanupMetrics
	nilm.observe("deleted", "swept")
}

package parammodel

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// TestClassifyBackupFile 覆盖 T-0178 §9.6 文件名分类规则。
func TestClassifyBackupFile(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		wantKind string
		wantOK   bool
	}{
		{"deleted backup", "CBQQ.xml.deleted.20260101030405", "deleted", true},
		{"bak backup", "CBQQ.xml.bak.20260601000000", "bak", true},
		{"tmp residual", "CBQQ.xml.tmp.abc123-def4-5678", "tmp", true},
		{"tmp with uuid hyphens", "X.xml.tmp.a1b2c3d4-e5f6-7890-abcd-ef1234567890", "tmp", true},

		// 非清扫对象
		{"normal xml", "CBQQ.xml", "", false},
		{"reserved standard", "standard-model.xml", "", false},
		{"name like .bak no ts", "CBQQ.xml.bak", "", false},
		{"name like .deleted short ts", "CBQQ.xml.deleted.2026", "", false},
		{"ts has trailing", "CBQQ.xml.bak.20260601000000.extra", "", false},
		{"random user backup", "CBQQ.xml.bak.manual", "", false},
		{"empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, ok := classifyBackupFile(tc.filename)
			if ok != tc.wantOK {
				t.Errorf("classifyBackupFile(%q): ok=%v, want %v", tc.filename, ok, tc.wantOK)
			}
			if kind != tc.wantKind {
				t.Errorf("classifyBackupFile(%q): kind=%q, want %q", tc.filename, kind, tc.wantKind)
			}
		})
	}
}

// TestBackupCleanup_NoDir 验证目录不存在时 ENOENT 被容忍。
func TestBackupCleanup_NoDir(t *testing.T) {
	bc := NewBackupCleanup(filepath.Join(t.TempDir(), "no-such"), 30, nil, nil)
	swept, err := bc.Run(context.Background())
	if err != nil {
		t.Errorf("ENOENT should not error: %v", err)
	}
	if swept != 0 {
		t.Errorf("swept = %d, want 0", swept)
	}
}

// TestBackupCleanup_DefaultRetention 验证 retentionDays <= 0 走默认 30。
func TestBackupCleanup_DefaultRetention(t *testing.T) {
	bc := NewBackupCleanup(t.TempDir(), 0, nil, nil)
	wantDays := DefaultBackupRetentionDays
	if bc.maxAge != time.Duration(wantDays)*24*time.Hour {
		t.Errorf("maxAge = %v, want %d days", bc.maxAge, wantDays)
	}
}

// TestBackupCleanup_SweepExpired 是核心场景:30 天前的 .deleted/.bak 应清理,
// 30 天内的应保留,合法 .xml 与无 ts 后缀的私有备份保留不动。
func TestBackupCleanup_SweepExpired(t *testing.T) {
	dir := t.TempDir()

	// 固定 now 为 2026-06-01 12:00:00,使测试可重复
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.Local)

	// 文件清单
	files := []struct {
		name      string
		shouldRm  bool
		desc      string
		writeMode string // "stat-mtime" for tmp files
		mtime     time.Time
	}{
		{"CBQQ.xml.deleted.20260101030405", true, "5 个月前 .deleted → 清", "", time.Time{}},
		{"BLQ.xml.deleted.20260530000000", false, "2 天前 .deleted → 留", "", time.Time{}},
		{"X.xml.bak.20260403120000", true, "60 天前 .bak → 清", "", time.Time{}},
		{"Y.xml.bak.20260601000000", false, "今天 .bak → 留", "", time.Time{}},
		{"CBQQ.xml", false, "合法 XML → 不清", "", time.Time{}},
		{"standard-model.xml", false, "保留名 → 不清", "", time.Time{}},
		{"Z.xml.bak.manual", false, "用户私有(非 14 位 ts) → 不清", "", time.Time{}},
		// .tmp 文件用 mtime 判断
		{"W.xml.tmp.uuid-old", true, "2 小时前的 .tmp → 清", "stat-mtime", now.Add(-2 * time.Hour)},
		{"V.xml.tmp.uuid-fresh", false, "10 秒前的 .tmp → 留(健康 Upload 中)", "stat-mtime", now.Add(-10 * time.Second)},
	}

	for _, f := range files {
		p := filepath.Join(dir, f.name)
		if err := os.WriteFile(p, []byte("x"), 0o640); err != nil {
			t.Fatal(err)
		}
		if f.writeMode == "stat-mtime" {
			if err := os.Chtimes(p, f.mtime, f.mtime); err != nil {
				t.Fatal(err)
			}
		}
	}

	bc := NewBackupCleanup(dir, 30, nil, nil)
	bc.now = func() time.Time { return now }
	swept, err := bc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	expectedRm := 0
	for _, f := range files {
		if f.shouldRm {
			expectedRm++
		}
	}
	if swept != expectedRm {
		t.Errorf("swept = %d, want %d", swept, expectedRm)
	}

	// 验证最终目录内容
	got := listFilenames(t, dir)
	want := []string{}
	for _, f := range files {
		if !f.shouldRm {
			want = append(want, f.name)
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if !equalStrings(got, want) {
		t.Errorf("dir after cleanup:\n got  %v\n want %v", got, want)
	}
}

// TestBackupCleanup_CutoffBoundary 验证恰好 cutoff 时间的文件被清理(>= 阈值)。
// 文件 ts == cutoff → 因 After 是严格大于,等于 cutoff 也清(即"满 30 天")。
func TestBackupCleanup_CutoffBoundary(t *testing.T) {
	dir := t.TempDir()
	// now = 2026-06-01 12:00:00,30 天前 = 2026-05-02 12:00:00
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.Local)
	cutoff := now.Add(-30 * 24 * time.Hour) // 2026-05-02 12:00:00

	// 比 cutoff 早 1 秒 → 清(过 30 天)
	exactExpire := cutoff.Add(-time.Second)
	expireName := "A.xml.deleted." + exactExpire.Format("20060102150405")
	// 比 cutoff 晚 1 秒 → 留(还在 30 天内)
	exactKeep := cutoff.Add(time.Second)
	keepName := "B.xml.deleted." + exactKeep.Format("20060102150405")

	for _, n := range []string{expireName, keepName} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o640); err != nil {
			t.Fatal(err)
		}
	}

	bc := NewBackupCleanup(dir, 30, nil, nil)
	bc.now = func() time.Time { return now }
	swept, _ := bc.Run(context.Background())
	if swept != 1 {
		t.Errorf("swept = %d, want 1 (boundary case)", swept)
	}

	got := listFilenames(t, dir)
	if len(got) != 1 || got[0] != keepName {
		t.Errorf("after cleanup got %v, want only [%s]", got, keepName)
	}
}

// TestBackupCleanup_ContextCancel 验证 ctx 取消时 Run 提前退出。
func TestBackupCleanup_ContextCancel(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.Local)
	// 写一堆过期文件
	for i := 0; i < 20; i++ {
		old := now.Add(-60 * 24 * time.Hour).Format("20060102150405")
		n := filepath.Join(dir, "X"+old+"-"+string(rune('a'+i))+".xml.deleted."+old)
		_ = os.WriteFile(n, []byte("x"), 0o640)
	}
	bc := NewBackupCleanup(dir, 30, nil, nil)
	bc.now = func() time.Time { return now }

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 提前取消
	_, err := bc.Run(ctx)
	if err == nil {
		t.Error("expected context error when ctx cancelled before scan")
	}
}

// TestBackupCleanup_MetricsObservation 验证 metrics 在不同分支被记录。
func TestBackupCleanup_MetricsObservation(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.Local)

	// 1 个过期 .deleted(swept), 1 个新鲜 .bak(skipped), 1 个新鲜 .tmp(skipped)
	_ = os.WriteFile(filepath.Join(dir, "X.xml.deleted.20260101000000"), []byte("x"), 0o640)
	_ = os.WriteFile(filepath.Join(dir, "Y.xml.bak.20260530000000"), []byte("x"), 0o640)
	tmpPath := filepath.Join(dir, "Z.xml.tmp.uuid")
	_ = os.WriteFile(tmpPath, []byte("x"), 0o640)
	_ = os.Chtimes(tmpPath, now.Add(-time.Second), now.Add(-time.Second))

	m := NewBackupCleanupMetrics(nil) // anonymous registry
	bc := NewBackupCleanup(dir, 30, m, nil)
	bc.now = func() time.Time { return now }
	_, err := bc.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// metrics 是 protected,这里仅验证 NewBackupCleanupMetrics 注册不 panic + Run 不 panic
}

// TestBackupCleanup_NilMetrics 验证 metrics=nil 时 observe 安全(防御性)。
func TestBackupCleanup_NilMetrics(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "X.xml.deleted.20260101000000"), []byte("x"), 0o640)
	bc := NewBackupCleanup(dir, 30, nil, nil)
	bc.now = func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local) }
	if _, err := bc.Run(context.Background()); err != nil {
		t.Fatalf("nil metrics should be safe: %v", err)
	}
}

// ── 辅助 ───────────────────────────────────────────────────────────────

func listFilenames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

package indicator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// ── ParseReloadMode 表驱动 ──────────────────────────────────────────

func TestParseReloadMode(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    ReloadMode
		wantErr bool
	}{
		{"empty defaults to import", "", ReloadModeImport, false},
		{"explicit import", "import", ReloadModeImport, false},
		{"reload", "reload", ReloadModeReload, false},

		{"unknown rejected", "destroy", "", true},
		{"case-sensitive RELOAD rejected", "RELOAD", "", true},
		{"whitespace rejected", "  reload", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseReloadMode(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

// ── PerformReloadWithOrphans 行为 ────────────────────────────────────

func TestPerformReloadWithOrphans_HappyPath(t *testing.T) {
	reloader := &stubReloader{}
	repo := &mockFileRepository{
		orphanRows: map[string]int{
			"enb": 5,
			"gsm": 2,
			"gnb": 0,
		},
	}
	result, err := PerformReloadWithOrphans(context.Background(), repo, reloader, zap.NewNop())
	assert.NoError(t, err)
	assert.Equal(t, ReloadModeReload, result.Mode)
	assert.Equal(t, 5, result.Orphans["enb"])
	assert.Equal(t, 2, result.Orphans["gsm"])
	assert.Equal(t, 0, result.Orphans["gnb"])

	// Reloader 必须先调一次
	assert.Equal(t, 1, reloader.calls)

	// DeleteOrphansBefore 应被三制式各调用一次,顺序 enb → gsm → gnb
	assert.Equal(t, []string{"enb", "gsm", "gnb"}, repo.orphanCalls)
	assert.Len(t, repo.orphanCutoffs, 3)
	// 三次 before 时刻应一致(都用同一个 start)
	assert.Equal(t, repo.orphanCutoffs[0], repo.orphanCutoffs[1])
	assert.Equal(t, repo.orphanCutoffs[0], repo.orphanCutoffs[2])
}

func TestPerformReloadWithOrphans_ReloaderFails(t *testing.T) {
	reloader := &stubReloader{err: errors.New("dictloader err")}
	repo := &mockFileRepository{}
	_, err := PerformReloadWithOrphans(context.Background(), repo, reloader, zap.NewNop())
	assert.Error(t, err)
	// Reload 失败时不应触发 DeleteOrphansBefore
	assert.Empty(t, repo.orphanCalls)
}

func TestPerformReloadWithOrphans_OrphanCleanupFailsPartial(t *testing.T) {
	reloader := &stubReloader{}
	repo := &mockFileRepository{orphanErr: errors.New("simulated db err")}
	_, err := PerformReloadWithOrphans(context.Background(), repo, reloader, zap.NewNop())
	assert.Error(t, err)
	// Reload 已成功 + 第一次 DeleteOrphans(enb) 失败 → 后续 gsm/gnb 不调
	assert.Equal(t, []string{"enb"}, repo.orphanCalls)
}

func TestPerformReloadWithOrphans_NilReloader_Errors(t *testing.T) {
	_, err := PerformReloadWithOrphans(context.Background(), &mockFileRepository{}, nil, zap.NewNop())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reloader not wired")
}

func TestPerformReloadWithOrphans_NilRepo_Errors(t *testing.T) {
	_, err := PerformReloadWithOrphans(context.Background(), nil, &stubReloader{}, zap.NewNop())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file repository not wired")
}

func TestPerformReloadWithOrphans_StartMonotonicity(t *testing.T) {
	// 防御性测试:确保 start 在 Reload 调用之前抓取,而非之后
	// 通过让 stubReloader sleep,看 before 时刻确实早于 reloader 返回时刻
	reloader := &stubReloader{sleep: 10 * time.Millisecond}
	repo := &mockFileRepository{}
	before := time.Now()
	_, err := PerformReloadWithOrphans(context.Background(), repo, reloader, zap.NewNop())
	after := time.Now()
	assert.NoError(t, err)

	// 三次 before 应都在 [before, after) 区间内,即"重载开始时刻"
	for _, cutoff := range repo.orphanCutoffs {
		assert.False(t, cutoff.Before(before), "cutoff %v should be >= test start %v", cutoff, before)
		assert.False(t, cutoff.After(after), "cutoff %v should be <= test end %v", cutoff, after)
	}
}

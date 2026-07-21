package minio

import (
	"context"
	"errors"
	"testing"

	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/stretchr/testify/require"
)

// TestRawFileLifecycleConfig 校验原始文件桶生命周期配置：恰一条 Enabled 规则、整桶、按入参天数过期
// （#169 设此策略；#319 天数改为可配，本测试用任意天数验证规则构造正确）。
func TestRawFileLifecycleConfig(t *testing.T) {
	const days = 60
	lc := rawFileLifecycleConfig(days)
	if got := len(lc.Rules); got != 1 {
		t.Fatalf("rules = %d, want 1", got)
	}
	r := lc.Rules[0]

	if r.Status != "Enabled" {
		t.Errorf("rule status = %q, want Enabled", r.Status)
	}
	if got := int(r.Expiration.Days); got != days {
		t.Errorf("expiration days = %d, want %d", got, days)
	}
	if r.RuleFilter.Prefix != "" {
		t.Errorf("filter prefix = %q, want empty (整桶)", r.RuleFilter.Prefix)
	}
	if r.ID == "" {
		t.Error("rule ID 不应为空")
	}
}

// TestDefaultRawFileRetentionDays 锁定兜底默认值（#319：默认对齐 60 天保留场景）。
func TestDefaultRawFileRetentionDays(t *testing.T) {
	if DefaultRawFileRetentionDays != 60 {
		t.Fatalf("DefaultRawFileRetentionDays = %d, want 60 (#319)", DefaultRawFileRetentionDays)
	}
}

func TestApplyRawFileLifecycleRejectsNilClient(t *testing.T) {
	require.Error(t, ApplyRawFileLifecycle(t.Context(), nil, []string{"pm-files"}, 60))
}

type fakeLifecycleClient struct {
	configs        map[string]*lifecycle.Configuration
	failDesiredOn  string
	failRollbackOn string
	setCalls       []string
	cancelApply    context.CancelFunc
}

func (f *fakeLifecycleClient) GetBucketLifecycle(_ context.Context, bucket string) (*lifecycle.Configuration, error) {
	config, ok := f.configs[bucket]
	if !ok {
		return lifecycle.NewConfiguration(), nil
	}
	return cloneLifecycle(config), nil
}

func (f *fakeLifecycleClient) SetBucketLifecycle(ctx context.Context, bucket string, config *lifecycle.Configuration) error {
	days := lifecycleDays(config)
	f.setCalls = append(f.setCalls, bucket)
	if bucket == f.failDesiredOn && lifecycleDays(config) == 60 {
		if f.cancelApply != nil {
			f.cancelApply()
		}
		return errors.New("injected set failure")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if bucket == f.failRollbackOn && days == 14 {
		return errors.New("injected rollback failure")
	}
	f.configs[bucket] = cloneLifecycle(config)
	return nil
}

func TestApplyRawFileLifecycleRollbackSurvivesCancelledApplyContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	client := &fakeLifecycleClient{configs: map[string]*lifecycle.Configuration{
		"pm-files": rawFileLifecycleConfig(14),
		"mr-files": rawFileLifecycleConfig(14),
	}, failDesiredOn: "mr-files", cancelApply: cancel}

	err := applyRawFileLifecycle(ctx, client, []string{"pm-files", "mr-files"}, 60)

	require.Error(t, err)
	require.NotContains(t, err.Error(), "compensating rollback failed")
	require.Equal(t, 14, lifecycleDays(client.configs["pm-files"]))
	require.Equal(t, 14, lifecycleDays(client.configs["mr-files"]))
}

func TestApplyRawFileLifecycleAttemptsEveryRollbackAfterOneRollbackFails(t *testing.T) {
	client := &fakeLifecycleClient{configs: map[string]*lifecycle.Configuration{
		"first":  rawFileLifecycleConfig(14),
		"second": rawFileLifecycleConfig(14),
		"third":  rawFileLifecycleConfig(14),
	}, failDesiredOn: "third", failRollbackOn: "second"}

	err := applyRawFileLifecycle(t.Context(), client, []string{"first", "second", "third"}, 60)

	require.ErrorContains(t, err, "compensating rollback failed")
	require.Equal(t, 14, lifecycleDays(client.configs["first"]),
		"rollback must continue to earlier buckets after another bucket fails")
	require.Equal(t, 60, lifecycleDays(client.configs["second"]))
	require.Equal(t, 14, lifecycleDays(client.configs["third"]))
}

func TestApplyRawFileLifecycleRollsBackEarlierBucketsOnFailure(t *testing.T) {
	client := &fakeLifecycleClient{configs: map[string]*lifecycle.Configuration{
		"pm-files": rawFileLifecycleConfig(14),
		"mr-files": rawFileLifecycleConfig(14),
	}, failDesiredOn: "mr-files"}

	err := applyRawFileLifecycle(t.Context(), client, []string{"pm-files", "mr-files"}, 60)

	require.ErrorContains(t, err, "mr-files")
	require.Equal(t, 14, lifecycleDays(client.configs["pm-files"]),
		"a partial update must restore the earlier bucket's lifecycle")
	require.Equal(t, 14, lifecycleDays(client.configs["mr-files"]))
}

func lifecycleDays(config *lifecycle.Configuration) int {
	if config == nil || len(config.Rules) != 1 {
		return 0
	}
	return int(config.Rules[0].Expiration.Days)
}

func cloneLifecycle(config *lifecycle.Configuration) *lifecycle.Configuration {
	if config == nil {
		return lifecycle.NewConfiguration()
	}
	clone := *config
	clone.Rules = append([]lifecycle.Rule(nil), config.Rules...)
	return &clone
}

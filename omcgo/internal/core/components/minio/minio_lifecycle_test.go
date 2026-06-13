package minio

import "testing"

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

// TestApplyRawFileLifecycle_NilClientNoop 校验降级：client 为 nil 或 days<=0 时不报错、不调用。
func TestApplyRawFileLifecycle_NilClientNoop(t *testing.T) {
	if err := ApplyRawFileLifecycle(nil, nil, []string{"pm-files"}, 60); err != nil {
		t.Errorf("nil client should noop, got %v", err)
	}
}

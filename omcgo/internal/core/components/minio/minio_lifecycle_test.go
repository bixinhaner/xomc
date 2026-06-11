package minio

import "testing"

// TestRawFileLifecycleConfig 校验原始文件桶生命周期配置：恰一条 Enabled 规则、整桶、14 天过期（#169）。
func TestRawFileLifecycleConfig(t *testing.T) {
	if rawFileRetentionDays != 14 {
		t.Fatalf("rawFileRetentionDays = %d, want 14 (#169 决定)", rawFileRetentionDays)
	}

	lc := rawFileLifecycleConfig()
	if got := len(lc.Rules); got != 1 {
		t.Fatalf("rules = %d, want 1", got)
	}
	r := lc.Rules[0]

	if r.Status != "Enabled" {
		t.Errorf("rule status = %q, want Enabled", r.Status)
	}
	if got := int(r.Expiration.Days); got != rawFileRetentionDays {
		t.Errorf("expiration days = %d, want %d", got, rawFileRetentionDays)
	}
	if r.RuleFilter.Prefix != "" {
		t.Errorf("filter prefix = %q, want empty (整桶)", r.RuleFilter.Prefix)
	}
	if r.ID == "" {
		t.Error("rule ID 不应为空")
	}
}

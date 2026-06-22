package minio

import (
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

func newTestCfg(publicEndpoint string) appconfig.MinIOConfig {
	return appconfig.MinIOConfig{
		Endpoint:       "minio:9000",
		PublicEndpoint: publicEndpoint,
		AccessKey:      "ak",
		SecretKey:      "sk",
		UseSSL:         false,
	}
}

// TestPresignBridge_Initial_UsesEnvFallback：cfg.PublicEndpoint 非空时初值就是它。
func TestPresignBridge_Initial_UsesEnvFallback(t *testing.T) {
	cfg := newTestCfg("public-host:9000")
	b, err := NewPresignBridge(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewPresignBridge err=%v", err)
	}
	if got := b.CurrentEndpoint(); got != "public-host:9000" {
		t.Fatalf("initial endpoint = %q, want public-host:9000", got)
	}
	if c := b.Get(); c == nil {
		t.Fatalf("Get() returned nil client")
	}
}

// TestPresignBridge_Initial_FallsBackToInternal：cfg.PublicEndpoint 空时回退内部 endpoint + Warn 一次。
func TestPresignBridge_Initial_FallsBackToInternal(t *testing.T) {
	core, recorded := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	cfg := newTestCfg("")
	b, err := NewPresignBridge(cfg, logger)
	if err != nil {
		t.Fatalf("NewPresignBridge err=%v", err)
	}
	if got := b.CurrentEndpoint(); got != "minio:9000" {
		t.Fatalf("internal fallback endpoint = %q, want minio:9000", got)
	}
	warns := recorded.FilterMessageSnippet("回退使用内部 endpoint").All()
	if len(warns) != 1 {
		t.Fatalf("expected 1 Warn about internal fallback, got %d (all logs: %+v)", len(warns), recorded.All())
	}
}

// TestPresignBridge_Priority_SysConfigOverridesEnv：sys_configs 值生效后覆盖 envFallback。
func TestPresignBridge_Priority_SysConfigOverridesEnv(t *testing.T) {
	b, err := NewPresignBridge(newTestCfg("env-host:9000"), zap.NewNop())
	if err != nil {
		t.Fatalf("NewPresignBridge err=%v", err)
	}
	if err := b.SetPublicEndpoint("ui-vip.example:9100"); err != nil {
		t.Fatalf("SetPublicEndpoint err=%v", err)
	}
	if got := b.CurrentEndpoint(); got != "ui-vip.example:9100" {
		t.Fatalf("after Set, endpoint = %q, want ui-vip.example:9100", got)
	}
}

// TestPresignBridge_Set_EmptyFallsBackToEnv：清掉 sys_configs（传 ""）回退到 envFallback。
func TestPresignBridge_Set_EmptyFallsBackToEnv(t *testing.T) {
	b, err := NewPresignBridge(newTestCfg("env-host:9000"), zap.NewNop())
	if err != nil {
		t.Fatalf("NewPresignBridge err=%v", err)
	}
	if err := b.SetPublicEndpoint("ui-vip.example:9100"); err != nil {
		t.Fatalf("SetPublicEndpoint err=%v", err)
	}
	if err := b.SetPublicEndpoint(""); err != nil {
		t.Fatalf("SetPublicEndpoint('') err=%v", err)
	}
	if got := b.CurrentEndpoint(); got != "env-host:9000" {
		t.Fatalf("after clear, endpoint = %q, want env-host:9000", got)
	}
}

// TestPresignBridge_Set_EmptyAndEnvEmpty_FallsBackToInternal：全空时落到内部 endpoint。
func TestPresignBridge_Set_EmptyAndEnvEmpty_FallsBackToInternal(t *testing.T) {
	core, _ := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	b, err := NewPresignBridge(newTestCfg(""), logger)
	if err != nil {
		t.Fatalf("NewPresignBridge err=%v", err)
	}
	// 初值已是内部 endpoint；Set("") 再次落到内部应保持幂等（不报错）。
	if err := b.SetPublicEndpoint(""); err != nil {
		t.Fatalf("SetPublicEndpoint('') err=%v", err)
	}
	if got := b.CurrentEndpoint(); got != "minio:9000" {
		t.Fatalf("endpoint = %q, want internal minio:9000", got)
	}
}

// TestPresignBridge_Set_InvalidIsRejectedAndCurrentUnchanged：非法值拒绝 + 当前 client 不变。
func TestPresignBridge_Set_InvalidIsRejectedAndCurrentUnchanged(t *testing.T) {
	b, err := NewPresignBridge(newTestCfg("env-host:9000"), zap.NewNop())
	if err != nil {
		t.Fatalf("NewPresignBridge err=%v", err)
	}
	before := b.Get()
	beforeEP := b.CurrentEndpoint()

	cases := []string{
		"http://x:9000",
		"https://x",
		"x:9000/foo",
		"x:9000?a=1",
		"x:9000#frag",
		":9000",
		"x:abc",
		"x:0",
		"x:65536",
	}
	for _, bad := range cases {
		err := b.SetPublicEndpoint(bad)
		if err == nil {
			t.Fatalf("SetPublicEndpoint(%q) want err, got nil", bad)
		}
		if !strings.Contains(err.Error(), "public_endpoint") {
			t.Fatalf("err msg lacks 'public_endpoint': %v", err)
		}
		if b.Get() != before {
			t.Fatalf("invalid Set(%q) replaced client (want unchanged)", bad)
		}
		if b.CurrentEndpoint() != beforeEP {
			t.Fatalf("invalid Set(%q) changed CurrentEndpoint to %q", bad, b.CurrentEndpoint())
		}
	}
}

// TestPresignBridge_Set_IdempotentSameValue_NoRebuild：相同 effective endpoint 不重建 client。
func TestPresignBridge_Set_IdempotentSameValue_NoRebuild(t *testing.T) {
	b, err := NewPresignBridge(newTestCfg("env-host:9000"), zap.NewNop())
	if err != nil {
		t.Fatalf("NewPresignBridge err=%v", err)
	}
	c1 := b.Get()
	if err := b.SetPublicEndpoint("env-host:9000"); err != nil { // 与 envFallback 一致
		t.Fatalf("SetPublicEndpoint err=%v", err)
	}
	if b.Get() != c1 {
		t.Fatalf("same effective endpoint should be idempotent (client pointer unchanged)")
	}
}

// TestPresignBridge_Concurrent_GetAndSet：并发读写不 race（要 -race 跑才有意义）。
func TestPresignBridge_Concurrent_GetAndSet(t *testing.T) {
	b, err := NewPresignBridge(newTestCfg("env-host:9000"), zap.NewNop())
	if err != nil {
		t.Fatalf("NewPresignBridge err=%v", err)
	}

	const writers = 8
	const readers = 16
	const iters = 200

	endpoints := []string{
		"a.example:9000",
		"b.example:9000",
		"c.example:9000",
		"", // 触发回退到 envFallback
	}

	var wg sync.WaitGroup
	wg.Add(writers + readers)
	for i := 0; i < writers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				_ = b.SetPublicEndpoint(endpoints[(id+j)%len(endpoints)])
			}
		}(i)
	}
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				if c := b.Get(); c == nil {
					t.Errorf("Get() returned nil under concurrency")
					return
				}
				_ = b.CurrentEndpoint()
			}
		}()
	}
	wg.Wait()
}

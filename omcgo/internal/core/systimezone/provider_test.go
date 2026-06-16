package systimezone

import (
	"context"
	"testing"
	"time"
)

// fetcherFunc 把闭包适配成 Fetcher，并记录调用次数（验缓存命中）。
type fetcherStub struct {
	value string
	ok    bool
	calls int
}

func (f *fetcherStub) fetch(_ context.Context, _ string, _ string) (string, bool) {
	f.calls++
	return f.value, f.ok
}

func TestProvider_Location(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name     string
		value    string
		ok       bool
		wantName string
	}{
		{name: "正常IANA-上海", value: "Asia/Shanghai", ok: true, wantName: "Asia/Shanghai"},
		{name: "正常IANA-UTC", value: "UTC", ok: true, wantName: "UTC"},
		{name: "空值默认UTC", value: "", ok: true, wantName: "UTC"},
		{name: "读不到默认UTC", value: "", ok: false, wantName: "UTC"},
		{name: "纯空白默认UTC", value: "   ", ok: true, wantName: "UTC"},
		{name: "非法值回落UTC", value: "Not/A_Real_Zone", ok: true, wantName: "UTC"},
		{name: "非法值乱码回落UTC", value: "@@@", ok: true, wantName: "UTC"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &fetcherStub{value: tc.value, ok: tc.ok}
			p := New(stub.fetch, nil)
			got := p.Location(ctx)
			if got == nil {
				t.Fatalf("Location() = nil, 期望非 nil")
			}
			if got.String() != tc.wantName {
				t.Fatalf("Location() = %q, 期望 %q", got.String(), tc.wantName)
			}
		})
	}
}

// 正常 IANA 值应与标准库 LoadLocation 等价（瞬时偏移一致）。
func TestProvider_Location_OffsetMatches(t *testing.T) {
	stub := &fetcherStub{value: "Asia/Shanghai", ok: true}
	p := New(stub.fetch, nil)
	loc := p.Location(context.Background())

	want, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	ref := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	if got := ref.In(loc).Format("-0700"); got != ref.In(want).Format("-0700") {
		t.Fatalf("offset = %s, 期望 %s", got, ref.In(want).Format("-0700"))
	}
}

// 缓存命中：TTL 内多次取只打一次配置源。
func TestProvider_Cache(t *testing.T) {
	stub := &fetcherStub{value: "UTC", ok: true}
	p := New(stub.fetch, nil)

	for i := 0; i < 5; i++ {
		_ = p.Location(context.Background())
	}
	if stub.calls != 1 {
		t.Fatalf("fetch 调用次数 = %d, 期望 1（缓存命中）", stub.calls)
	}
}

// 失效刷新：Invalidate 后下次 Location 重读配置源，能拿到新值。
func TestProvider_Invalidate(t *testing.T) {
	stub := &fetcherStub{value: "UTC", ok: true}
	p := New(stub.fetch, nil)

	if got := p.Location(context.Background()).String(); got != "UTC" {
		t.Fatalf("首次 = %q, 期望 UTC", got)
	}
	// 运维改了系统时区
	stub.value = "Asia/Shanghai"
	// 未失效：仍命中旧缓存
	if got := p.Location(context.Background()).String(); got != "UTC" {
		t.Fatalf("失效前 = %q, 期望仍 UTC（缓存）", got)
	}
	p.Invalidate()
	if got := p.Location(context.Background()).String(); got != "Asia/Shanghai" {
		t.Fatalf("失效后 = %q, 期望 Asia/Shanghai", got)
	}
}

// TTL 过期后自动重读。
func TestProvider_TTLExpiry(t *testing.T) {
	stub := &fetcherStub{value: "UTC", ok: true}
	p := New(stub.fetch, nil, WithTTL(time.Millisecond))

	_ = p.Location(context.Background())
	stub.value = "Asia/Shanghai"
	time.Sleep(5 * time.Millisecond)
	if got := p.Location(context.Background()).String(); got != "Asia/Shanghai" {
		t.Fatalf("TTL 过期后 = %q, 期望 Asia/Shanghai", got)
	}
}

package redisx_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

func newMiniRedis(t *testing.T) (redis.UniversalClient, *miniredis.Miniredis) {
	t.Helper()
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client, s
}

func TestScanner_CollectsAllMatches(t *testing.T) {
	client, _ := newMiniRedis(t)
	ctx := context.Background()

	// Seed 7 keys under the same pattern + 3 noise keys.
	for i := 0; i < 7; i++ {
		if err := client.Set(ctx, fmt.Sprintf("acs:taskq:SN%02d", i), "x", 0).Err(); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	for i := 0; i < 3; i++ {
		if err := client.Set(ctx, fmt.Sprintf("noise:%d", i), "x", 0).Err(); err != nil {
			t.Fatalf("seed noise: %v", err)
		}
	}

	keys, err := redisx.ScanKeys(ctx, client, redisx.Keys.ACSTaskQueuePattern())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(keys) != 7 {
		t.Fatalf("len(keys) = %d, want 7; keys = %v", len(keys), keys)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if k[:len("acs:taskq:")] != "acs:taskq:" {
			t.Errorf("unexpected key in result: %s", k)
		}
	}
}

func TestScanner_CallbackCanStop(t *testing.T) {
	client, _ := newMiniRedis(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_ = client.Set(ctx, fmt.Sprintf("acs:taskq:%d", i), "x", 0).Err()
	}

	sentinel := fmt.Errorf("stop")
	count := 0
	err := redisx.Scan(ctx, client, redisx.Keys.ACSTaskQueuePattern(),
		func(_ context.Context, _ string) error {
			count++
			if count == 3 {
				return sentinel
			}
			return nil
		})
	if err != sentinel {
		t.Fatalf("err = %v, want sentinel", err)
	}
	if count != 3 {
		t.Fatalf("count = %d, want 3 (callback should have stopped)", count)
	}
}

func TestPurgePattern_DeletesAllMatches(t *testing.T) {
	client, _ := newMiniRedis(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_ = client.Set(ctx, fmt.Sprintf("acs:taskq:SN%d", i), "x", 0).Err()
	}
	_ = client.Set(ctx, "keepme:1", "x", 0).Err()

	n, err := redisx.PurgePattern(ctx, client, redisx.Keys.ACSTaskQueuePattern(), 100)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if n != 5 {
		t.Fatalf("purged = %d, want 5", n)
	}
	// "keepme:1" 仍然存在。
	if _, err := client.Get(ctx, "keepme:1").Result(); err != nil {
		t.Fatalf("keepme:1 should survive purge, got err = %v", err)
	}
}

func TestTouchTTL_RefreshesKeys(t *testing.T) {
	client, s := newMiniRedis(t)
	ctx := context.Background()

	_ = client.Set(ctx, "acs:session:id:A", "x", 0).Err()
	_ = client.Set(ctx, "acs:session:id:B", "x", 0).Err()

	if err := redisx.TouchTTL(ctx, client,
		5*time.Minute,
		"acs:session:id:A", "acs:session:id:B"); err != nil {
		t.Fatalf("touch: %v", err)
	}

	// miniredis TTL 是近似值，这里仅断言 TTL > 0。
	if ttl := s.TTL("acs:session:id:A"); ttl <= 0 {
		t.Fatalf("A TTL not refreshed: %v", ttl)
	}
	if ttl := s.TTL("acs:session:id:B"); ttl <= 0 {
		t.Fatalf("B TTL not refreshed: %v", ttl)
	}
}

func TestTouchTTL_ZeroArgsAreNoop(t *testing.T) {
	client, _ := newMiniRedis(t)
	// 空 keys / 零 ttl 应当是 no-op，不返回错误。
	if err := redisx.TouchTTL(context.Background(), client, time.Minute); err != nil {
		t.Fatalf("empty keys: %v", err)
	}
	if err := redisx.TouchTTL(context.Background(), client, 0, "k"); err != nil {
		t.Fatalf("zero ttl: %v", err)
	}
}

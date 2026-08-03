package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/stretchr/testify/require"
)

func TestWireUnknownAlarmFallback_InjectsBothReceivers(t *testing.T) {
	alarmReceiver := &alarm.AlarmReceiver{}
	expeditedReceiver := &alarm.ExpeditedEventReceiver{}
	alarmDefRegistry := &definition.Registry{}
	productRegistry := &product.Registry{}

	alarmReceiver, expeditedReceiver, resolver := wireUnknownAlarmFallback(
		alarmReceiver,
		expeditedReceiver,
		alarmDefRegistry,
		productRegistry,
	)

	require.NotNil(t, resolver)
	require.False(t, nilField(t, alarmReceiver, "alarmDefRegistry"))
	require.False(t, nilField(t, alarmReceiver, "productResolver"))
	require.False(t, nilField(t, expeditedReceiver, "alarmDefRegistry"))
	require.False(t, nilField(t, expeditedReceiver, "productResolver"))
}

func nilField(t *testing.T, target any, fieldName string) bool {
	t.Helper()
	value := reflect.ValueOf(target)
	require.Equal(t, reflect.Ptr, value.Kind())
	field := value.Elem().FieldByName(fieldName)
	require.Truef(t, field.IsValid(), "field %s should exist", fieldName)
	return field.IsNil()
}

func TestEnabledIndicatorLookupCachesByDeviceTypeWithinTTL(t *testing.T) {
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	repo := &fakeEnabledIndicatorRepo{
		ids: map[indicator.DeviceType][]string{
			indicator.DeviceTypeENB: {"C0001", "", "K0001"},
			indicator.DeviceTypeGSM: {"CGSM0010001"},
		},
	}
	lookup := &enabledIndicatorLookup{
		repo:  repo,
		ttl:   5 * time.Minute,
		now:   func() time.Time { return now },
		cache: make(map[indicator.DeviceType]enabledIndicatorCacheEntry),
	}

	first, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, first, "C0001")
	require.Contains(t, first, "K0001")
	require.NotContains(t, first, "")
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeENB])

	delete(first, "C0001")
	first["MUTATED"] = struct{}{}
	second, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, second, "C0001", "调用方修改返回 map 不应污染缓存")
	require.NotContains(t, second, "MUTATED")
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeENB], "TTL 内同一制式不应重复查库")

	gsm, err := lookup.LookupEnabledIndicators(context.Background(), "gsm")
	require.NoError(t, err)
	require.Contains(t, gsm, "CGSM0010001")
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeGSM], "不同制式使用独立缓存")

	repo.ids[indicator.DeviceTypeENB] = []string{"C0002"}
	now = now.Add(5*time.Minute + time.Nanosecond)
	expired, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, expired, "C0002")
	require.NotContains(t, expired, "C0001")
	require.Equal(t, 2, repo.calls[indicator.DeviceTypeENB], "TTL 过期后应重新查库")
}

func TestEnabledIndicatorLookupFailureIsNotCached(t *testing.T) {
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	repoErr := errors.New("db down")
	repo := &fakeEnabledIndicatorRepo{
		ids: map[indicator.DeviceType][]string{
			indicator.DeviceTypeENB: {"C0001"},
		},
		err: repoErr,
	}
	lookup := &enabledIndicatorLookup{
		repo:  repo,
		ttl:   5 * time.Minute,
		now:   func() time.Time { return now },
		cache: make(map[indicator.DeviceType]enabledIndicatorCacheEntry),
	}

	_, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.ErrorIs(t, err, repoErr)
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeENB])

	repo.err = nil
	got, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, got, "C0001")
	require.Equal(t, 2, repo.calls[indicator.DeviceTypeENB], "查询失败不应写入缓存")
}

func TestEnabledIndicatorLookupReloadsImmediatelyWhenCacheVersionChanges(t *testing.T) {
	repo := &fakeEnabledIndicatorRepo{ids: map[indicator.DeviceType][]string{
		indicator.DeviceTypeENB: {"K900010040"},
	}}
	version := "1"
	lookup := &enabledIndicatorLookup{
		repo: repo,
		ttl:  5 * time.Minute,
		now:  time.Now,
		readCacheVersion: func(context.Context) (string, error) {
			return version, nil
		},
		cache: make(map[indicator.DeviceType]enabledIndicatorCacheEntry),
	}

	first, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, first, "K900010040")
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeENB])

	repo.ids[indicator.DeviceTypeENB] = []string{
		"K900010040", "C000190005", "C000190007", "C000190009",
	}
	version = "2"
	second, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, second, "C000190005")
	require.Contains(t, second, "C000190007")
	require.Contains(t, second, "C000190009")
	require.Equal(t, 2, repo.calls[indicator.DeviceTypeENB],
		"cross-process cache version bump must bypass the five-minute TTL")
}

type fakeEnabledIndicatorRepo struct {
	ids   map[indicator.DeviceType][]string
	err   error
	calls map[indicator.DeviceType]int
}

func (f *fakeEnabledIndicatorRepo) ListAll(_ context.Context, dt indicator.DeviceType) ([]string, error) {
	if f.calls == nil {
		f.calls = make(map[indicator.DeviceType]int)
	}
	f.calls[dt]++
	if f.err != nil {
		return nil, f.err
	}
	return append([]string(nil), f.ids[dt]...), nil
}

func TestKnownReportKeyLookupCachesByDeviceType(t *testing.T) {
	now := time.Date(2026, 7, 31, 5, 0, 0, 0, time.UTC)
	repo := &fakeKnownReportKeyRepo{
		keys: map[indicator.DeviceType][]string{
			indicator.DeviceTypeENB: {"RRC.AttConn", "", "Cqi.00"},
		},
	}
	lookup := &knownReportKeyLookup{
		repo:  repo,
		ttl:   5 * time.Minute,
		now:   func() time.Time { return now },
		cache: make(map[indicator.DeviceType]knownReportKeyCacheEntry),
	}

	first, err := lookup.LookupKnownReportKeys(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, first, "RRC.AttConn")
	require.Contains(t, first, "Cqi.00")
	require.NotContains(t, first, "")

	second, err := lookup.LookupKnownReportKeys(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, second, "RRC.AttConn")
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeENB], "TTL 内不应重复查指标库")

	repo.keys[indicator.DeviceTypeENB] = []string{"RRC.New"}
	now = now.Add(5*time.Minute + time.Nanosecond)
	expired, err := lookup.LookupKnownReportKeys(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, expired, "RRC.New")
	require.NotContains(t, expired, "RRC.AttConn")
	require.Equal(t, 2, repo.calls[indicator.DeviceTypeENB])
}

type fakeKnownReportKeyRepo struct {
	keys  map[indicator.DeviceType][]string
	err   error
	calls map[indicator.DeviceType]int
}

func (f *fakeKnownReportKeyRepo) ListCounterReportKeys(_ context.Context, dt indicator.DeviceType) ([]string, error) {
	if f.calls == nil {
		f.calls = make(map[indicator.DeviceType]int)
	}
	f.calls[dt]++
	if f.err != nil {
		return nil, f.err
	}
	return append([]string(nil), f.keys[dt]...), nil
}

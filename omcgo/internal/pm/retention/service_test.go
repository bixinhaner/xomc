package retention

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// stubReader 实现 SysConfigReader 用于单测。
type stubReader struct {
	mu     sync.Mutex
	values map[PolicyKey]string // 模拟 sys_configs (category, key) 的 value
	types  map[PolicyKey]string // 模拟 value_type；未设置则默认 "int"
	err    error
	calls  int
}

func newStubReader(initial map[PolicyKey]string) *stubReader {
	return &stubReader{values: initial, types: map[PolicyKey]string{}}
}

func (s *stubReader) GetByKey(_ context.Context, category, key string) (*SysConfigRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	if category != Category {
		return nil, nil
	}
	v, ok := s.values[PolicyKey(key)]
	if !ok {
		return nil, nil
	}
	vt := s.types[PolicyKey(key)]
	if vt == "" {
		vt = "int"
	}
	return &SysConfigRow{Value: v, ValueType: vt}, nil
}

func (s *stubReader) set(k PolicyKey, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[k] = v
}

func TestService_Get_FallbackDefault_WhenSysConfigsMissing(t *testing.T) {
	reader := newStubReader(map[PolicyKey]string{}) // 全空
	svc := NewService(reader, nil)

	for _, k := range AllKeys() {
		require.Equal(t, DefaultDays[k], svc.Get(context.Background(), k), "expected default for %s", k)
	}
}

func TestService_Get_ReadsSysConfigs_WhenPresent(t *testing.T) {
	reader := newStubReader(map[PolicyKey]string{
		KeyRaw15MinDays: "45",
		KeyHourlyDays:   "365",
	})
	svc := NewService(reader, nil)

	require.Equal(t, 45, svc.Get(context.Background(), KeyRaw15MinDays))
	require.Equal(t, 365, svc.Get(context.Background(), KeyHourlyDays))
	// 缺失走默认
	require.Equal(t, DefaultDays[KeyDailyDays], svc.Get(context.Background(), KeyDailyDays))
}

func TestService_ReloadStrict_UsesDefaultsForMissingSiblingPolicies(t *testing.T) {
	svc := NewService(newStubReader(map[PolicyKey]string{
		KeyRaw15MinDays: "45",
	}), nil)

	values, err := svc.ReloadStrict(context.Background())

	require.NoError(t, err)
	require.Equal(t, 45, values[KeyRaw15MinDays])
	require.Equal(t, DefaultDays[KeyHourlyDays], values[KeyHourlyDays])
}

func TestService_Get_CachesAfterFirstRead(t *testing.T) {
	reader := newStubReader(map[PolicyKey]string{KeyRaw15MinDays: "60"})
	svc := NewService(reader, nil)

	require.Equal(t, 60, svc.Get(context.Background(), KeyRaw15MinDays))
	require.Equal(t, 60, svc.Get(context.Background(), KeyRaw15MinDays))
	require.Equal(t, 60, svc.Get(context.Background(), KeyRaw15MinDays))

	// 仅一次实际读
	require.Equal(t, 1, reader.calls)
}

func TestService_Reload_NotifiesListener_OnChange(t *testing.T) {
	reader := newStubReader(map[PolicyKey]string{
		KeyRaw15MinDays: "30",
		KeyHourlyDays:   "180",
	})
	svc := NewService(reader, nil)

	var (
		gotChanged    []PolicyKey
		gotCurrent    map[PolicyKey]int
		listenerCalls int
	)
	svc.RegisterListener(func(_ context.Context, current map[PolicyKey]int, changed []PolicyKey) {
		listenerCalls++
		gotCurrent = current
		gotChanged = changed
	})

	// 第一次 Reload：全部从 cache 空 → newVals，全部 changed
	require.NoError(t, svc.Reload(context.Background()))
	require.Equal(t, 1, listenerCalls)
	require.Len(t, gotChanged, len(AllKeys()))
	require.Equal(t, 30, gotCurrent[KeyRaw15MinDays])

	// 第二次 Reload：值不变 → listener 不应触发
	reader.calls = 0
	gotChanged = nil
	require.NoError(t, svc.Reload(context.Background()))
	require.Equal(t, 1, listenerCalls, "listener should not fire when no values changed")

	// 改一个值再 Reload：仅 1 changed key
	reader.set(KeyHourlyDays, "200")
	gotChanged = nil
	require.NoError(t, svc.Reload(context.Background()))
	require.Equal(t, 2, listenerCalls)
	require.Equal(t, []PolicyKey{KeyHourlyDays}, gotChanged)
	require.Equal(t, 200, svc.Get(context.Background(), KeyHourlyDays))
}

func TestService_OnSysConfigSaved_OnlyTriggersOnMatchingCategory(t *testing.T) {
	reader := newStubReader(map[PolicyKey]string{KeyRaw15MinDays: "30"})
	svc := NewService(reader, nil)

	listenerCalls := 0
	svc.RegisterListener(func(context.Context, map[PolicyKey]int, []PolicyKey) {
		listenerCalls++
	})

	// 无关 category → 不触发 Reload
	svc.OnSysConfigSaved(context.Background(), "ui_custom")
	require.Equal(t, 0, listenerCalls)

	// 匹配 category → 触发 Reload
	svc.OnSysConfigSaved(context.Background(), Category)
	require.GreaterOrEqual(t, listenerCalls, 1)
}

func TestService_readOne_RejectsInvalidValueType(t *testing.T) {
	reader := newStubReader(map[PolicyKey]string{KeyRaw15MinDays: "30"})
	reader.types[KeyRaw15MinDays] = "string" // 错误类型
	svc := NewService(reader, nil)

	// Get 应 fallback 走默认
	require.Equal(t, DefaultDays[KeyRaw15MinDays], svc.Get(context.Background(), KeyRaw15MinDays))
}

func TestService_readOne_RejectsOutOfRange(t *testing.T) {
	reader := newStubReader(map[PolicyKey]string{
		KeyRaw15MinDays: "0", // 越界（< MinRetentionDays）
	})
	svc := NewService(reader, nil)
	require.Equal(t, DefaultDays[KeyRaw15MinDays], svc.Get(context.Background(), KeyRaw15MinDays))

	reader2 := newStubReader(map[PolicyKey]string{
		KeyRaw15MinDays: strconv.Itoa(MaxRetentionDays + 1), // 越界（> MaxRetentionDays）
	})
	svc2 := NewService(reader2, nil)
	require.Equal(t, DefaultDays[KeyRaw15MinDays], svc2.Get(context.Background(), KeyRaw15MinDays))
}

func TestService_readOne_RejectsNonNumeric(t *testing.T) {
	reader := newStubReader(map[PolicyKey]string{
		KeyRaw15MinDays: "abc",
	})
	svc := NewService(reader, nil)
	require.Equal(t, DefaultDays[KeyRaw15MinDays], svc.Get(context.Background(), KeyRaw15MinDays))
}

func TestService_Get_HandlesReaderError(t *testing.T) {
	reader := newStubReader(map[PolicyKey]string{KeyRaw15MinDays: "30"})
	reader.err = errors.New("db unreachable")
	svc := NewService(reader, nil)

	require.Equal(t, DefaultDays[KeyRaw15MinDays], svc.Get(context.Background(), KeyRaw15MinDays))
}

func TestService_GetAll_ReturnsAllFiveKeys(t *testing.T) {
	reader := newStubReader(map[PolicyKey]string{KeyRaw15MinDays: "45"})
	svc := NewService(reader, nil)

	all := svc.GetAll(context.Background())
	require.Len(t, all, len(AllKeys()))
	require.Equal(t, 45, all[KeyRaw15MinDays])
	require.Equal(t, DefaultDays[KeyHourlyDays], all[KeyHourlyDays])
}

func TestService_RegisterListener_NilSafe(t *testing.T) {
	svc := NewService(newStubReader(map[PolicyKey]string{}), nil)
	svc.RegisterListener(nil) // 不应 panic
	require.Empty(t, svc.listeners)
}

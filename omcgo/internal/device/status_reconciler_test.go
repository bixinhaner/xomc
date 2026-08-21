package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock: statusReconcilerRepo（Pattern B — function fields）
// ---------------------------------------------------------------------------

type mockReconcilerRepo struct {
	findFn func(ctx context.Context, enbThresholdSec, cpeThresholdSec, upsThresholdSec, limit int) ([]*model.Device, error)
	markFn func(ctx context.Context, deviceID uuid.UUID, reason string, now time.Time) (bool, error)

	// 记录调用,用于断言。
	markCalls []markCall
	findCalls []findCall
}

type markCall struct {
	DeviceID uuid.UUID
	Reason   string
	Now      time.Time
}

type findCall struct {
	ENBThresholdSec int
	CPEThresholdSec int
	UPSThresholdSec int
	Limit           int
}

func (m *mockReconcilerRepo) FindStaleDevicesByClass(ctx context.Context, enbThresholdSec, cpeThresholdSec, upsThresholdSec, limit int) ([]*model.Device, error) {
	m.findCalls = append(m.findCalls, findCall{ENBThresholdSec: enbThresholdSec, CPEThresholdSec: cpeThresholdSec, UPSThresholdSec: upsThresholdSec, Limit: limit})
	if m.findFn == nil {
		return nil, nil
	}
	return m.findFn(ctx, enbThresholdSec, cpeThresholdSec, upsThresholdSec, limit)
}

func (m *mockReconcilerRepo) MarkOfflineWithAccounting(ctx context.Context, deviceID uuid.UUID, reason string, now time.Time) (bool, error) {
	m.markCalls = append(m.markCalls, markCall{DeviceID: deviceID, Reason: reason, Now: now})
	if m.markFn == nil {
		return true, nil
	}
	return m.markFn(ctx, deviceID, reason, now)
}

type capturingOfflineAlarmSink struct {
	alarms []*model.Alarm
	err    error
}

func (s *capturingOfflineAlarmSink) Process(_ context.Context, alarm *model.Alarm) error {
	if s.err != nil {
		return s.err
	}
	s.alarms = append(s.alarms, alarm)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestReconciler(t *testing.T, repo *mockReconcilerRepo, bus event.EventBus) (*DeviceStatusReconciler, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	r := NewDeviceStatusReconciler(rdb, repo, bus, zap.NewNop())
	return r, mr
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewDeviceStatusReconciler_Defaults(t *testing.T) {
	r, _ := newTestReconciler(t, &mockReconcilerRepo{}, nil)

	require.NotNil(t, r)
	// issue #203：checkInterval=0 表示按配置阈值动态计算（不再固定 5min）。
	assert.Equal(t, time.Duration(0), r.checkInterval)
	assert.Equal(t, 1000, r.batchSize)
	assert.Nil(t, r.thresholdLookup, "默认未注入阈值 lookup")
}

func TestRefreshHeartbeat_TTLFromInterval(t *testing.T) {
	r, mr := newTestReconciler(t, &mockReconcilerRepo{}, nil)

	r.RefreshHeartbeat(context.Background(), "SN-HB-001", 300)

	key := redisx.Keys.ACSHeartbeat("SN-HB-001")
	assert.True(t, mr.Exists(key), "heartbeat key must exist after refresh")
	assert.Equal(t, 600*time.Second, mr.TTL(key), "TTL should be 2 × inform_interval")
}

func TestRefreshHeartbeat_MinTTLClamp(t *testing.T) {
	r, mr := newTestReconciler(t, &mockReconcilerRepo{}, nil)

	// 10×2 = 20s 远小于 60s,应钳到 600s。
	r.RefreshHeartbeat(context.Background(), "SN-HB-MIN", 10)

	key := redisx.Keys.ACSHeartbeat("SN-HB-MIN")
	require.True(t, mr.Exists(key))
	assert.Equal(t, 600*time.Second, mr.TTL(key))
}

func TestRefreshHeartbeat_NilRedisIsNoOp(t *testing.T) {
	r := &DeviceStatusReconciler{logger: zap.NewNop()}
	assert.NotPanics(t, func() {
		r.RefreshHeartbeat(context.Background(), "SN-NIL", 300)
	})
}

func TestDetect_MarksStaleDevicesOffline(t *testing.T) {
	stale := &model.Device{
		ID:           uuid.New(),
		SerialNumber: "SN-STALE",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
	}
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _ int, _ int, _ int, _ int) ([]*model.Device, error) {
			return []*model.Device{stale}, nil
		},
	}

	r, _ := newTestReconciler(t, repo, nil)
	r.detect(context.Background())

	require.Len(t, repo.markCalls, 1)
	assert.Equal(t, stale.ID, repo.markCalls[0].DeviceID)
	assert.Equal(t, OfflineReasonHeartbeatTimeout, repo.markCalls[0].Reason)
}

func TestDetect_NoStaleDevicesIsNoOp(t *testing.T) {
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _ int, _ int, _ int, _ int) ([]*model.Device, error) {
			return nil, nil
		},
	}

	r, _ := newTestReconciler(t, repo, nil)
	assert.NotPanics(t, func() { r.detect(context.Background()) })
	assert.Empty(t, repo.markCalls)
}

func TestDetect_RepoErrorIsLoggedNotPanic(t *testing.T) {
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _ int, _ int, _ int, _ int) ([]*model.Device, error) {
			return nil, errors.New("db down")
		},
	}

	r, _ := newTestReconciler(t, repo, nil)
	assert.NotPanics(t, func() { r.detect(context.Background()) })
	assert.Empty(t, repo.markCalls, "should not attempt to mark when find fails")
}

func TestMarkOffline_PublishesEventOnTransition(t *testing.T) {
	stale := &model.Device{
		ID: uuid.New(), SerialNumber: "SN-EVT", Carrier: model.CarrierCMCC, Technology: model.TechLTE,
	}
	repo := &mockReconcilerRepo{
		markFn: func(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
			return true, nil // 真翻转
		},
	}

	bus := event.NewChannelEventBus(64, zap.NewNop())
	received := make(chan event.Event, 1)
	_, err := bus.QueueSubscribe(event.SubjectDeviceOffline, "test-sub",
		func(_ context.Context, evt event.Event) error {
			received <- evt
			return nil
		})
	require.NoError(t, err)

	r, _ := newTestReconciler(t, repo, bus)
	transitioned, err := r.markOffline(context.Background(), stale)
	require.NoError(t, err)
	assert.True(t, transitioned)

	select {
	case evt := <-received:
		assert.Equal(t, event.SubjectDeviceOffline, evt.Subject)
		var payload DeviceOfflineEvent
		require.NoError(t, evt.DecodePayload(&payload))
		assert.Equal(t, stale.ID, payload.DeviceID)
		assert.Equal(t, OfflineReasonHeartbeatTimeout, payload.Reason)
	case <-time.After(1 * time.Second):
		t.Fatal("expected device.offline event not received")
	}
}

func TestMarkOffline_InvalidatesDeviceCacheOnTransition(t *testing.T) {
	stale := &model.Device{
		ID: uuid.New(), SerialNumber: "SN-CACHE", Carrier: model.CarrierCMCC, Technology: model.TechLTE,
	}
	repo := &mockReconcilerRepo{
		markFn: func(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
			return true, nil
		},
	}

	r, mr := newTestReconciler(t, repo, nil)
	cache := NewDeviceCache(redis.NewClient(&redis.Options{Addr: mr.Addr()}), zap.NewNop())
	r.SetDeviceCache(cache)
	cache.Set(context.Background(), stale)
	require.True(t, mr.Exists(deviceCacheKey(stale.SerialNumber)), "precondition: cache entry must exist")

	transitioned, err := r.markOffline(context.Background(), stale)
	require.NoError(t, err)
	assert.True(t, transitioned)
	assert.False(t, mr.Exists(deviceCacheKey(stale.SerialNumber)), "offline transition must delete cached device snapshot")
}

func TestMarkOffline_RaisesDisconnectedAlarmByTechnology(t *testing.T) {
	tests := []struct {
		name       string
		tech       model.Technology
		wantID     string
		wantSource string
	}{
		{name: "LTE eNB", tech: model.TechLTE, wantID: "7", wantSource: "OMC"},
		{name: "NR gNB", tech: model.TechNR, wantID: "23", wantSource: "OMC"},
		{name: "GSM", tech: model.TechGSM, wantID: "4", wantSource: "OMC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stale := &model.Device{
				ID:           uuid.New(),
				SerialNumber: "SN-" + tt.wantID,
				Carrier:      model.CarrierCMCC,
				Technology:   tt.tech,
			}
			sink := &capturingOfflineAlarmSink{}
			repo := &mockReconcilerRepo{
				markFn: func(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
					return true, nil
				},
			}
			r, _ := newTestReconciler(t, repo, nil)
			r.SetOfflineAlarmSink(sink)

			transitioned, err := r.markOffline(context.Background(), stale)

			require.NoError(t, err)
			assert.True(t, transitioned)
			require.Len(t, sink.alarms, 1)
			alarm := sink.alarms[0]
			assert.Equal(t, tt.wantID, alarm.AlarmIdentifier)
			require.NotNil(t, alarm.AlarmSource)
			assert.Equal(t, tt.wantSource, *alarm.AlarmSource)
			assert.Equal(t, stale.ID, alarm.DeviceID)
			assert.Equal(t, stale.SerialNumber, alarm.DeviceSN)
			assert.Equal(t, model.AlarmCritical, alarm.Severity)
			require.NotNil(t, alarm.Technology)
			assert.Equal(t, string(tt.tech), *alarm.Technology)
		})
	}
}

func TestMarkOffline_DoesNotRaiseBaseStationDisconnectedAlarmForUPS(t *testing.T) {
	stale := &model.Device{
		ID:           uuid.New(),
		SerialNumber: "SN-UPS-OFFLINE",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
		ProductClass: "UPS_M3_BMU",
	}
	sink := &capturingOfflineAlarmSink{}
	repo := &mockReconcilerRepo{
		markFn: func(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
			return true, nil
		},
	}
	r, _ := newTestReconciler(t, repo, nil)
	r.SetOfflineAlarmSink(sink)

	transitioned, err := r.markOffline(context.Background(), stale)

	require.NoError(t, err)
	assert.True(t, transitioned)
	assert.Empty(t, sink.alarms, "UPS 不能因为默认 LTE technology 误报 eNB Disconnected")
}

func TestMarkOffline_DoesNotRaiseDisconnectedAlarmWhenAlreadyOffline(t *testing.T) {
	stale := &model.Device{ID: uuid.New(), SerialNumber: "SN-IDEM-ALARM", Technology: model.TechLTE}
	sink := &capturingOfflineAlarmSink{}
	repo := &mockReconcilerRepo{
		markFn: func(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
			return false, nil
		},
	}
	r, _ := newTestReconciler(t, repo, nil)
	r.SetOfflineAlarmSink(sink)

	transitioned, err := r.markOffline(context.Background(), stale)

	require.NoError(t, err)
	assert.False(t, transitioned)
	assert.Empty(t, sink.alarms)
}

func TestMarkOffline_AlarmFailureDoesNotFailOfflineTransition(t *testing.T) {
	stale := &model.Device{ID: uuid.New(), SerialNumber: "SN-ALARM-FAIL", Technology: model.TechLTE}
	sink := &capturingOfflineAlarmSink{err: errors.New("alarm store down")}
	repo := &mockReconcilerRepo{
		markFn: func(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
			return true, nil
		},
	}
	r, _ := newTestReconciler(t, repo, nil)
	r.SetOfflineAlarmSink(sink)

	transitioned, err := r.markOffline(context.Background(), stale)

	require.NoError(t, err)
	assert.True(t, transitioned)
}

func TestMarkOffline_NoEventWhenAlreadyOffline(t *testing.T) {
	stale := &model.Device{ID: uuid.New(), SerialNumber: "SN-IDEM"}
	repo := &mockReconcilerRepo{
		markFn: func(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
			return false, nil // 幂等:已离线
		},
	}

	bus := event.NewChannelEventBus(64, zap.NewNop())
	received := make(chan event.Event, 1)
	_, err := bus.QueueSubscribe(event.SubjectDeviceOffline, "test-sub-idem",
		func(_ context.Context, evt event.Event) error {
			received <- evt
			return nil
		})
	require.NoError(t, err)

	r, _ := newTestReconciler(t, repo, bus)
	transitioned, err := r.markOffline(context.Background(), stale)
	require.NoError(t, err)
	assert.False(t, transitioned)

	select {
	case <-received:
		t.Fatal("should NOT publish event when device was already offline")
	case <-time.After(100 * time.Millisecond):
		// ok - no event published
	}
}

func TestStartStop_GracefulShutdown(t *testing.T) {
	repo := &mockReconcilerRepo{}
	r, _ := newTestReconciler(t, repo, nil)
	r.SetCheckInterval(50 * time.Millisecond)

	r.Start()
	time.Sleep(120 * time.Millisecond) // 让 detect 至少跑两轮
	r.Stop()

	// Stop 后再 Stop 不应 panic。
	assert.NotPanics(t, func() { r.Stop() })
}

func TestStartStop_StopBeforeStartIsNoop(t *testing.T) {
	r := &DeviceStatusReconciler{logger: zap.NewNop()}
	assert.NotPanics(t, func() { r.Stop() })
}

func TestSetters_GuardAgainstNonPositive(t *testing.T) {
	r, _ := newTestReconciler(t, &mockReconcilerRepo{}, nil)

	// 默认 checkInterval=0（动态）。
	r.SetCheckInterval(0)
	assert.Equal(t, time.Duration(0), r.checkInterval, "0 should be ignored")

	r.SetCheckInterval(-5 * time.Second)
	assert.Equal(t, time.Duration(0), r.checkInterval, "negative should be ignored")

	r.SetCheckInterval(10 * time.Second)
	assert.Equal(t, 10*time.Second, r.checkInterval)

	r.SetBatchSize(0)
	assert.Equal(t, 1000, r.batchSize)

	r.SetBatchSize(50)
	assert.Equal(t, 50, r.batchSize)
}

// ---------------------------------------------------------------------------
// issue #203：离线阈值实时配置 + 判离线规则 + 扫描周期计算
// ---------------------------------------------------------------------------

// 判离线规则——成功路径：配置阈值传给仓库,过期设备被翻离线。
func TestDetect_AppliesConfiguredThresholds(t *testing.T) {
	stale := &model.Device{ID: uuid.New(), SerialNumber: "SN-CFG", Carrier: model.CarrierCMCC, Technology: model.TechLTE}
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, enb, cpe, ups, _ int) ([]*model.Device, error) {
			// 断言对账器把配置阈值如实下传给仓库。
			assert.Equal(t, 100, enb)
			assert.Equal(t, 600, cpe)
			assert.Equal(t, 200, ups)
			return []*model.Device{stale}, nil
		},
	}
	r, _ := newTestReconciler(t, repo, nil)
	r.SetThresholdLookup(func(_ context.Context, _, key string) (string, bool) {
		switch key {
		case offlineConfigKeyENB:
			return "100", true
		case offlineConfigKeyCPE:
			return "600", true
		case offlineConfigKeyUPS:
			return "200", true
		}
		return "", false
	})

	r.detect(context.Background())

	require.Len(t, repo.findCalls, 1)
	assert.Equal(t, 100, repo.findCalls[0].ENBThresholdSec)
	assert.Equal(t, 600, repo.findCalls[0].CPEThresholdSec)
	assert.Equal(t, 200, repo.findCalls[0].UPSThresholdSec)
	require.Len(t, repo.markCalls, 1, "过期设备应被判离线")
	assert.Equal(t, stale.ID, repo.markCalls[0].DeviceID)
}

// 判离线规则——失败路径：lookup 返回非法值/缺失时退化到默认阈值,且无设备过期则不翻离线。
func TestDetect_FallsBackToDefaultsAndNoFalsePositive(t *testing.T) {
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _, _, _, _ int) ([]*model.Device, error) {
			return nil, nil // 无过期设备
		},
	}
	r, _ := newTestReconciler(t, repo, nil)
	// 非法值（非数字）+ 缺失 key → 全退默认 600 / 600（BUG-05 修复后 ENB 默认也是 600）。
	r.SetThresholdLookup(func(_ context.Context, _, key string) (string, bool) {
		if key == offlineConfigKeyENB {
			return "not-a-number", true
		}
		return "", false // cpeTimeout 缺失
	})

	r.detect(context.Background())

	require.Len(t, repo.findCalls, 1)
	assert.Equal(t, defaultENBOfflineSec, repo.findCalls[0].ENBThresholdSec, "非法值退默认 600")
	assert.Equal(t, defaultCPEOfflineSec, repo.findCalls[0].CPEThresholdSec, "缺失退默认 600")
	assert.Equal(t, defaultUPSOfflineSec, repo.findCalls[0].UPSThresholdSec, "缺失退默认 300")
	assert.Empty(t, repo.markCalls, "无过期设备不应误判离线")
}

// 对账器读配置值——改配置后下一轮用新值（无缓存,每轮实时读）。
func TestDetect_ReadsLatestConfigEachPass(t *testing.T) {
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _, _, _, _ int) ([]*model.Device, error) { return nil, nil },
	}
	r, _ := newTestReconciler(t, repo, nil)

	enbVal := "100"
	r.SetThresholdLookup(func(_ context.Context, _, key string) (string, bool) {
		if key == offlineConfigKeyENB {
			return enbVal, true
		}
		return "600", true
	})

	r.detect(context.Background())
	require.Len(t, repo.findCalls, 1)
	assert.Equal(t, 100, repo.findCalls[0].ENBThresholdSec)

	// 模拟用户在 UI 把基站阈值改小到 30s。
	enbVal = "30"
	r.detect(context.Background())
	require.Len(t, repo.findCalls, 2)
	assert.Equal(t, 30, repo.findCalls[1].ENBThresholdSec, "下一轮应读到新配置值")
}

// 扫描周期 = min(min(阈值)/2, 60s)：阈值 100 → 50s；阈值 600 → 60s（封顶）。
func TestScanIntervalFor(t *testing.T) {
	cases := []struct {
		name string
		th   OfflineThresholds
		want time.Duration
	}{
		{"阈值100→50s", OfflineThresholds{ENBSec: 100, CPESec: 100}, 50 * time.Second},
		{"阈值600→60s封顶", OfflineThresholds{ENBSec: 600, CPESec: 600}, 60 * time.Second},
		{"取较小者的一半", OfflineThresholds{ENBSec: 100, CPESec: 600}, 50 * time.Second},
		{"UPS阈值参与最小值", OfflineThresholds{ENBSec: 600, CPESec: 600, UPSSec: 300}, 60 * time.Second},
		{"UPS较小时跟随UPS", OfflineThresholds{ENBSec: 600, CPESec: 600, UPSSec: 40}, 20 * time.Second},
		{"极小阈值钳到1s下限", OfflineThresholds{ENBSec: 1, CPESec: 1}, time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, scanIntervalFor(c.th))
		})
	}
}

// nextScanInterval：显式 SetCheckInterval 优先于动态计算。
func TestNextScanInterval_ExplicitOverridesDynamic(t *testing.T) {
	r, _ := newTestReconciler(t, &mockReconcilerRepo{}, nil)
	th := OfflineThresholds{ENBSec: 100, CPESec: 600}

	// 未设固定值：动态 = 50s。
	assert.Equal(t, 50*time.Second, r.nextScanInterval(th))

	// 设固定值后优先固定值。
	r.SetCheckInterval(5 * time.Second)
	assert.Equal(t, 5*time.Second, r.nextScanInterval(th))
}

// ---------------------------------------------------------------------------
// issue #203：离线阈值 / limit 的 guard 默认值（DB-free 纯函数）
//
// 任务点：FindStaleDevicesByClass 的 guard 默认值须被尊重（enb<=0→600,
// cpe<=0→600, ups<=0→300, limit<=0→1000）。enb/cpe/ups 的钳制在 resolveOfflineThresholds
// （offline_threshold.go）这一纯函数里发生；limit 默认值由 reconciler 的
// batchSize（默认 1000，SetBatchSize 拒非正数）经 FindStaleDevicesByClass 第三参
// 端到端传入。下方分别直测纯函数与端到端下传。
// ---------------------------------------------------------------------------

// resolveOfflineThresholds 在 lookup=nil / 缺失 / 非数字 / 非正数时一律退默认
// （enb=600, cpe=600, ups=300，BUG-05 修复后 ENB 也是 600）——guard 默认值的纯函数源头。
func TestResolveOfflineThresholds_GuardDefaults(t *testing.T) {
	tests := []struct {
		name    string
		lookup  OfflineThresholdLookup
		wantENB int
		wantCPE int
		wantUPS int
	}{
		{
			name:    "nil lookup → 全退默认 600/600/300",
			lookup:  nil,
			wantENB: defaultENBOfflineSec,
			wantCPE: defaultCPEOfflineSec,
			wantUPS: defaultUPSOfflineSec,
		},
		{
			name: "key 缺失（found=false）→ 退默认",
			lookup: func(_ context.Context, _, _ string) (string, bool) {
				return "", false
			},
			wantENB: defaultENBOfflineSec,
			wantCPE: defaultCPEOfflineSec,
			wantUPS: defaultUPSOfflineSec,
		},
		{
			name: "非数字值 → 退默认",
			lookup: func(_ context.Context, _, _ string) (string, bool) {
				return "not-a-number", true
			},
			wantENB: defaultENBOfflineSec,
			wantCPE: defaultCPEOfflineSec,
			wantUPS: defaultUPSOfflineSec,
		},
		{
			name: "0（非正数）→ 退默认",
			lookup: func(_ context.Context, _, _ string) (string, bool) {
				return "0", true
			},
			wantENB: defaultENBOfflineSec,
			wantCPE: defaultCPEOfflineSec,
			wantUPS: defaultUPSOfflineSec,
		},
		{
			name: "负数 → 退默认",
			lookup: func(_ context.Context, _, _ string) (string, bool) {
				return "-30", true
			},
			wantENB: defaultENBOfflineSec,
			wantCPE: defaultCPEOfflineSec,
			wantUPS: defaultUPSOfflineSec,
		},
		{
			name: "合法正数 → 采用配置值（非默认）",
			lookup: func(_ context.Context, _, key string) (string, bool) {
				if key == offlineConfigKeyENB {
					return "45", true
				}
				if key == offlineConfigKeyUPS {
					return "180", true
				}
				return "300", true
			},
			wantENB: 45,
			wantCPE: 300,
			wantUPS: 180,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := resolveOfflineThresholds(context.Background(), tt.lookup)
			assert.Equal(t, tt.wantENB, th.ENBSec)
			assert.Equal(t, tt.wantCPE, th.CPESec)
			assert.Equal(t, tt.wantUPS, th.UPSSec)
		})
	}
}

// 默认常量本身就是 600/600/300（BUG-05 修复后 ENB 也是 600）——FindStaleDevicesByClass 的 guard 默认值同源。
func TestOfflineDefaultConstants(t *testing.T) {
	assert.Equal(t, 600, defaultENBOfflineSec, "enb 默认阈值 600（BUG-05 修复）")
	assert.Equal(t, 600, defaultCPEOfflineSec, "cpe 默认阈值 600")
	assert.Equal(t, 300, defaultUPSOfflineSec, "ups 默认阈值 300")
}

// 端到端：未注入 lookup 时，一轮扫描下传给仓库的阈值是默认 600/600/300（BUG-05 修复后），limit 是默认
// 1000（reconciler batchSize 默认值）——三个 guard 默认值在 detect→Find 链路上兑现。
func TestDetect_GuardDefaultsHonoredEndToEnd(t *testing.T) {
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _, _, _, _ int) ([]*model.Device, error) { return nil, nil },
	}
	r, _ := newTestReconciler(t, repo, nil) // 不注入 thresholdLookup → 默认阈值

	r.detect(context.Background())

	require.Len(t, repo.findCalls, 1)
	assert.Equal(t, 600, repo.findCalls[0].ENBThresholdSec, "enb 默认 600（BUG-05 修复）")
	assert.Equal(t, 600, repo.findCalls[0].CPEThresholdSec, "cpe 默认 600")
	assert.Equal(t, 300, repo.findCalls[0].UPSThresholdSec, "ups 默认 300")
	assert.Equal(t, 1000, repo.findCalls[0].Limit, "limit 默认 1000（batchSize 默认值）")
}

// limit guard：SetBatchSize 拒绝非正数（0 / 负数），batchSize 保持 1000，下传 limit 仍 1000。
func TestDetect_BatchSizeGuardKeepsLimitDefault(t *testing.T) {
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _, _, _, _ int) ([]*model.Device, error) { return nil, nil },
	}
	r, _ := newTestReconciler(t, repo, nil)

	r.SetBatchSize(0)    // 非正数被 guard 拒绝
	r.SetBatchSize(-100) // 负数同样被拒
	r.detect(context.Background())

	require.Len(t, repo.findCalls, 1)
	assert.Equal(t, 1000, repo.findCalls[0].Limit, "非正数 batchSize 被拒，limit 保持默认 1000")

	// 合法正数生效后下传新 limit。
	r.SetBatchSize(250)
	r.detect(context.Background())
	require.Len(t, repo.findCalls, 2)
	assert.Equal(t, 250, repo.findCalls[1].Limit, "合法 batchSize 端到端下传为 limit")
}

package device

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

// TestDetect_OnlineIndexVeto_RescuesFreshDevices 验证 issue #397 根治片：
// last_inform_at 判为离线候选的设备，若 acs:online 里 score 仍在其 class 阈值内，
// 则被「免 NATS」二次确认为在线、不标离线；超阈值或不在索引中的才真标离线。
func TestDetect_OnlineIndexVeto_RescuesFreshDevices(t *testing.T) {
	enb := func(sn string) *model.Device {
		return &model.Device{ID: uuid.New(), SerialNumber: sn, ProductClass: "BaiStation_eNodeB"}
	}
	cpe := func(sn string) *model.Device {
		return &model.Device{ID: uuid.New(), SerialNumber: sn, ProductClass: "CPE-Home-Indoor"}
	}
	ups := func(sn string) *model.Device {
		return &model.Device{ID: uuid.New(), SerialNumber: sn, ProductClass: "UPS_M3_BMU"}
	}

	aFresh := enb("SN-A-fresh")   // eNB，50s 前上报（< 100s）→ 救回
	bStale := enb("SN-B-stale")   // eNB，5000s 前 → 真离线
	cCPE := cpe("SN-C-cpe")       // CPE，300s 前（> 100s 但 < 600s）→ 按 CPE 阈值救回
	dAbsent := enb("SN-D-absent") // 不在索引 → 真离线
	eUPS := ups("SN-E-ups")       // UPS，350s 前（> 300s）→ 按 UPS 阈值真离线

	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _, _, _, _ int) ([]*model.Device, error) {
			return []*model.Device{aFresh, bStale, cCPE, dAbsent, eUPS}, nil
		},
	}
	r, _ := newTestReconciler(t, repo, nil)

	ctx := context.Background()
	now := time.Now().Unix()
	require.NoError(t, r.onlineIndex.Mark(ctx, aFresh.SerialNumber, now-50))
	require.NoError(t, r.onlineIndex.Mark(ctx, bStale.SerialNumber, now-5000))
	require.NoError(t, r.onlineIndex.Mark(ctx, cCPE.SerialNumber, now-300))
	require.NoError(t, r.onlineIndex.Mark(ctx, eUPS.SerialNumber, now-350))
	// dAbsent 不写入索引（ZMSCORE 返回 0）。

	r.detect(ctx)

	// 只有 B（eNB 超阈值）+ D（不在索引）+ E（UPS 超 300s）被标离线；A、C 被在线索引救回。
	marked := map[uuid.UUID]bool{}
	for _, mc := range repo.markCalls {
		marked[mc.DeviceID] = true
	}
	require.Len(t, repo.markCalls, 3, "应只标 3 台离线（B、D、E）")
	require.True(t, marked[bStale.ID], "B（eNB 5000s）应被标离线")
	require.True(t, marked[dAbsent.ID], "D（不在索引）应被标离线")
	require.True(t, marked[eUPS.ID], "E（UPS 350s > 300s）应被标离线")
	require.False(t, marked[aFresh.ID], "A（eNB 50s）应被救回")
	require.False(t, marked[cCPE.ID], "C（CPE 300s < 600s）应被救回（按 CPE 阈值）")
}

// TestDetect_OnlineIndexVeto_DisabledByConfig 验证 sys_configs offlineZsetConfirm=false
// 时退化为旧行为（不二次确认，全部候选按 last_inform_at 标离线）。
func TestDetect_OnlineIndexVeto_DisabledByConfig(t *testing.T) {
	d1 := &model.Device{ID: uuid.New(), SerialNumber: "SN-1", ProductClass: "eNodeB"}
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _, _, _, _ int) ([]*model.Device, error) {
			return []*model.Device{d1}, nil
		},
	}
	r, _ := newTestReconciler(t, repo, nil)
	r.SetThresholdLookup(func(_ context.Context, _, key string) (string, bool) {
		if key == offlineConfigKeyZsetConfirm {
			return "false", true
		}
		return "", false
	})

	ctx := context.Background()
	// 即便 d1 在索引中很新鲜，关了开关也应照标离线。
	require.NoError(t, r.onlineIndex.Mark(ctx, d1.SerialNumber, time.Now().Unix()))

	r.detect(ctx)

	require.Len(t, repo.markCalls, 1, "关闭确认开关后，候选应照旧标离线")
	require.Equal(t, d1.ID, repo.markCalls[0].DeviceID)
}

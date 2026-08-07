package alarm

import (
	"context"
	"errors"
	"testing"

	"github.com/omcgo/omcgo/internal/alarm/definition"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// stubAlarmDefLookup 实现 alarmDefLookup，按 identifier 返回预置定义；缺失 → ErrUnknownIdentifier。
type stubAlarmDefLookup struct {
	defs map[string]definition.ResolvedDefinition
	err  error // 非 ErrUnknownIdentifier 的查询错误（测 fallback 不阻断）
}

func (s *stubAlarmDefLookup) Lookup(_ context.Context, identifier string) (*definition.ResolvedDefinition, error) {
	if s.err != nil {
		return nil, s.err
	}
	d, ok := s.defs[identifier]
	if !ok {
		return nil, definition.ErrUnknownIdentifier
	}
	return &d, nil
}

func newEnrichEngine(lookup alarmDefLookup) *FilterEngine {
	e := NewFilterEngine(&mockFilterRuleRepo{}, &mockStoreForEngine{}, nil, nil, nil, zap.NewNop())
	if lookup != nil {
		e.SetAlarmDefLookup(lookup)
	}
	return e
}

func strp(s string) *string { return &s }

func sampleLookup() *stubAlarmDefLookup {
	return &stubAlarmDefLookup{defs: map[string]definition.ResolvedDefinition{
		"ALM-1001": {
			AlarmDefinition: definition.AlarmDefinition{
				Identifier:      "ALM-1001",
				NeType:          "ENB",
				CnName:          "小区退服",
				EnName:          "Cell Out of Service",
				CnProbableCause: "传输中断",
				EnProbableCause: "Transport interrupted",
				CnSuggestion:    "检查传输链路",
				EnSuggestion:    "Check the transport link",
			},
		},
		"ALM-2002": {
			AlarmDefinition: definition.AlarmDefinition{
				Identifier: "ALM-2002",
				CnName:     "仅有中文名",
				EnName:     "", // 英文缺失，en locale 应回退中文
			},
		},
	}}
}

func TestEnrichFromLibrary_ZhLocale_UsesCnName(t *testing.T) {
	e := newEnrichEngine(sampleLookup())
	ctx := appcontext.WithLocale(context.Background(), appcontext.LocaleZH)
	a := &model.Alarm{AlarmIdentifier: "ALM-1001", Description: "device reported text"}

	e.enrichFromLibrary(ctx, a)

	assert.Equal(t, "小区退服", a.Description, "zh locale 应取 cn_name")
	require.NotNil(t, a.ProbableCause)
	assert.Equal(t, "传输中断", *a.ProbableCause, "zh locale 可能原因应取 cn_probable_cause")
	assert.Equal(t, "ENB", a.AdditionalInfo["ne_type"])
	assert.Equal(t, "小区退服", a.AdditionalInfo["alarm_name"])
	assert.Equal(t, "检查传输链路", a.AdditionalInfo["handling_suggestion"])
}

func TestEnrichFromLibrary_EnLocale_UsesEnName(t *testing.T) {
	e := newEnrichEngine(sampleLookup())
	ctx := appcontext.WithLocale(context.Background(), appcontext.LocaleEN)
	a := &model.Alarm{AlarmIdentifier: "ALM-1001"}

	e.enrichFromLibrary(ctx, a)

	assert.Equal(t, "Cell Out of Service", a.Description, "en locale 应取 en_name")
	require.NotNil(t, a.ProbableCause)
	assert.Equal(t, "Transport interrupted", *a.ProbableCause)
	assert.Equal(t, "Check the transport link", a.AdditionalInfo["handling_suggestion"])
}

func TestEnrichFromLibrary_EnLocale_FallsBackToCnWhenEnEmpty(t *testing.T) {
	e := newEnrichEngine(sampleLookup())
	ctx := appcontext.WithLocale(context.Background(), appcontext.LocaleEN)
	a := &model.Alarm{AlarmIdentifier: "ALM-2002"}

	e.enrichFromLibrary(ctx, a)

	assert.Equal(t, "仅有中文名", a.Description, "en_name 为空时 COALESCE 应回退 cn_name")
}

func TestEnrichFromLibrary_UnknownIdentifier_KeepsDeviceReportedText(t *testing.T) {
	e := newEnrichEngine(sampleLookup())
	ctx := appcontext.WithLocale(context.Background(), appcontext.LocaleEN)
	a := &model.Alarm{AlarmIdentifier: "ALM-UNKNOWN", Description: "原始设备文本"}

	e.enrichFromLibrary(ctx, a)

	assert.Equal(t, "原始设备文本", a.Description, "字典缺失该 identifier 时不得置空，保留设备上报原文")
	assert.Nil(t, a.ProbableCause, "字典缺失且设备未报 → 可能原因保持 nil")
}

func TestEnrichFromLibrary_RespectsDeviceProbableCause(t *testing.T) {
	e := newEnrichEngine(sampleLookup())
	ctx := appcontext.WithLocale(context.Background(), appcontext.LocaleZH)
	a := &model.Alarm{AlarmIdentifier: "ALM-1001", ProbableCause: strp("设备自报原因")}

	e.enrichFromLibrary(ctx, a)

	require.NotNil(t, a.ProbableCause)
	assert.Equal(t, "设备自报原因", *a.ProbableCause, "设备已上报 probable_cause 时尊重原文，不覆盖")
}

func TestEnrichFromLibrary_NilLookup_NoOp(t *testing.T) {
	e := newEnrichEngine(nil) // 未注入字典
	ctx := appcontext.WithLocale(context.Background(), appcontext.LocaleEN)
	a := &model.Alarm{AlarmIdentifier: "ALM-1001", Description: "untouched"}

	e.enrichFromLibrary(ctx, a)

	assert.Equal(t, "untouched", a.Description, "无字典查询器时静默跳过，零回归")
}

func TestEnrichFromLibrary_LookupError_DoesNotBlock(t *testing.T) {
	e := newEnrichEngine(&stubAlarmDefLookup{err: errors.New("db down")})
	ctx := appcontext.WithLocale(context.Background(), appcontext.LocaleEN)
	a := &model.Alarm{AlarmIdentifier: "ALM-1001", Description: "device text"}

	// 不应 panic；查询失败时保留原文。
	e.enrichFromLibrary(ctx, a)
	assert.Equal(t, "device text", a.Description)
}

func TestEnrichFromLibrary_DefaultLocale_TreatedAsZh(t *testing.T) {
	e := newEnrichEngine(sampleLookup())
	// context 无 locale → GetLocale 回退 zh-CN
	a := &model.Alarm{AlarmIdentifier: "ALM-1001"}

	e.enrichFromLibrary(context.Background(), a)

	assert.Equal(t, "小区退服", a.Description, "缺省 locale 应按中文处理")
}

func TestLocalizedAlarmNameExpr_LocaleColumnOrder(t *testing.T) {
	en := localizedAlarmNameExpr(appcontext.LocaleEN, "alarms_active")
	assert.Contains(t, en, "ad.en_name")
	assert.Contains(t, en, "alarms_active.description")
	// en 优先英文列
	assert.Less(t, indexOf(en, "ad.en_name"), indexOf(en, "ad.cn_name"))

	zh := localizedAlarmNameExpr(appcontext.LocaleZH, "alarms_history")
	assert.Contains(t, zh, "ad.cn_name")
	assert.Contains(t, zh, "alarms_history.description")
	// zh 优先中文列
	assert.Less(t, indexOf(zh, "ad.cn_name"), indexOf(zh, "ad.en_name"))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

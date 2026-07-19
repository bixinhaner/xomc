package parammodel

import (
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkMapping(std, priv string) ParamMapping {
	return ParamMapping{
		ID:           uuid.New(),
		StandardPath: std,
		PrivatePath:  priv,
		EntryType:    "parameter",
	}
}

func TestTranslator_ToPrivate_HitAndMiss(t *testing.T) {
	set := &MappingSet{
		ParamModelID: uuid.New(),
		Source:       MappingSourceDefault,
		Mappings: []ParamMapping{
			mkMapping("Device.DeviceInfo.SAS.CpiId", "Device.DeviceInfo.CPI_Id"),
			mkMapping("Device.Services.FAPService.{i}.LICENSE.", "Device.Services.FAPService.{i}.X_COM_LICENSE."),
		},
	}
	tr := NewTranslator(set, NewRegistryMetrics(nil), nil)

	hit := tr.ToPrivate("Device.DeviceInfo.SAS.CpiId")
	require.True(t, hit.Found)
	assert.Equal(t, "Device.DeviceInfo.CPI_Id", hit.Translated)
	require.NotNil(t, hit.Mapping)

	miss := tr.ToPrivate("Device.Unknown")
	assert.False(t, miss.Found)
	assert.Equal(t, "Device.Unknown", miss.Translated)
	assert.Nil(t, miss.Mapping)
}

func TestTranslator_ToStandard_HitAndMiss(t *testing.T) {
	set := &MappingSet{
		Source:   MappingSourceDiscovered,
		Mappings: []ParamMapping{mkMapping("Device.Foo", "Device.X_Foo")},
	}
	tr := NewTranslator(set, NewRegistryMetrics(nil), nil)

	hit := tr.ToStandard("Device.X_Foo")
	require.True(t, hit.Found)
	assert.Equal(t, "Device.Foo", hit.Translated)

	miss := tr.ToStandard("Device.X_Unknown")
	assert.False(t, miss.Found)
	assert.Equal(t, "Device.X_Unknown", miss.Translated)
}

func TestTranslator_PlaceholderMismatchSkipped(t *testing.T) {
	// standard 1 个 {i}，private 0 个 → 跳过
	set := &MappingSet{
		ParamModelID: uuid.New(),
		Source:       MappingSourceDefault,
		Mappings: []ParamMapping{
			mkMapping("Device.OK.{i}.X", "Device.Priv.{i}.X"), // ok
			mkMapping("Device.Bad.{i}.X", "Device.Priv.X"),    // 跳过
		},
	}
	tr := NewTranslator(set, NewRegistryMetrics(nil), nil)
	assert.Equal(t, 1, tr.SkippedCount())

	// 校验 OK 那条仍可翻译
	r := tr.ToPrivate("Device.OK.{i}.X")
	assert.True(t, r.Found)

	// 跳过的不可翻译
	r2 := tr.ToPrivate("Device.Bad.{i}.X")
	assert.False(t, r2.Found)

	// 跳过的条目也不能参与 partial object prefix 派生。
	r3 := tr.ToPrivate("Device.Bad.")
	assert.False(t, r3.Found)
	assert.Empty(t, tr.ToPrivateCandidates("Device.Bad."))
}

func TestTranslator_Source(t *testing.T) {
	cases := []MappingSource{MappingSourceDiscovered, MappingSourceDefault}
	for _, src := range cases {
		tr := NewTranslator(&MappingSet{Source: src}, NewRegistryMetrics(nil), nil)
		assert.Equal(t, src, tr.Source())
	}
}

func TestTranslator_Mappings_PreservesOriginalOrder(t *testing.T) {
	mappings := []ParamMapping{
		mkMapping("A", "a"),
		mkMapping("B", "b"),
		mkMapping("C", "c"),
	}
	set := &MappingSet{Source: MappingSourceDefault, Mappings: mappings}
	tr := NewTranslator(set, NewRegistryMetrics(nil), nil)
	got := tr.Mappings()
	require.Len(t, got, 3)
	assert.Equal(t, "A", got[0].StandardPath)
	assert.Equal(t, "B", got[1].StandardPath)
	assert.Equal(t, "C", got[2].StandardPath)
}

func TestTranslator_NilSet(t *testing.T) {
	tr := NewTranslator(nil, NewRegistryMetrics(nil), nil)
	r := tr.ToPrivate("anything")
	assert.False(t, r.Found)
	assert.Empty(t, tr.Mappings())
	assert.Equal(t, MappingSource(""), tr.Source())
}

func TestValidatePlaceholders(t *testing.T) {
	cases := []struct {
		std, priv string
		ok        bool
	}{
		{"A", "a", true},
		{"A.{i}", "a.{i}", true},
		{"A.{i}.B.{i}", "a.{i}.b.{i}", true},
		{"A.{i}", "a", false},
		{"A", "a.{i}", false},
		{"A.{i}.B.{i}", "a.{i}", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.ok, validatePlaceholders(c.std, c.priv), "%s vs %s", c.std, c.priv)
	}
}

func TestParamModelLabel(t *testing.T) {
	pmID := uuid.New()
	prodID := uuid.New()

	// default with paramModelID
	got := paramModelLabel(&MappingSet{Source: MappingSourceDefault, ParamModelID: pmID})
	assert.Equal(t, pmID.String(), got)

	// discovered
	got = paramModelLabel(&MappingSet{Source: MappingSourceDiscovered, ProductID: prodID, SoftwareVersion: "1.0"})
	assert.Equal(t, prodID.String()+":1.0", got)

	// fallback
	got = paramModelLabel(&MappingSet{})
	assert.Equal(t, "unknown", got)
}

// ──────────────────────────────────────────────────────────────────────
// T-0178: {i} 占位符规范化补齐(运行时 .N. 与模板 .{i}. 折叠为同一索引键)
// ──────────────────────────────────────────────────────────────────────

func TestTranslator_ToPrivate_RuntimeInstanceFoldsToTemplate(t *testing.T) {
	set := &MappingSet{
		Source: MappingSourceDefault,
		Mappings: []ParamMapping{
			// std 与 priv 模板完全一致（FaultMgmt 标准 TR-181 路径，
			// 真实 BLQ param_mappings 形态）
			mkMapping(
				"Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation",
				"Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation",
			),
			// std 与 priv 模板不同前缀,确认数字按位置回填到 priv 模板
			mkMapping(
				"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.AccessClassBarringFactor.{i}.Probability",
				"Device.X_BAICELLS_FAP.{i}.AC.AccessClassBarring.{i}.Prob",
			),
		},
	}
	tr := NewTranslator(set, NewRegistryMetrics(nil), nil)

	// case 1: 单 {i} 单数字回填
	got := tr.ToPrivate("Device.FaultMgmt.CurrentAlarm.0.AdditionalInformation")
	require.True(t, got.Found, "运行时实例号路径必须命中模板 mapping")
	assert.Equal(t,
		"Device.FaultMgmt.CurrentAlarm.0.AdditionalInformation",
		got.Translated,
		"std == priv 模板,数字回填后两侧路径相同")

	// case 2: 多 {i} 按位置(.{i}.X.{i}.) 各填各的
	got = tr.ToPrivate(
		"Device.Services.FAPService.7.CellConfig.LTE.RAN.CellRestriction.AccessClassBarringFactor.3.Probability",
	)
	require.True(t, got.Found, "多 {i} 模板必须命中")
	assert.Equal(t,
		"Device.X_BAICELLS_FAP.7.AC.AccessClassBarring.3.Prob",
		got.Translated,
		"两个 {i} 槽按出现顺序分别回填 7 / 3")

	// case 3: 模板形态请求(含字面 {i})仍走直接命中分支,不应被 normalize 影响
	got = tr.ToPrivate("Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation")
	require.True(t, got.Found, "模板形态请求直接命中")
	assert.Equal(t,
		"Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation",
		got.Translated)

	// case 4: 不存在的路径 — normalize 后也命不中 → Found=false,Translated=Original
	got = tr.ToPrivate("Device.Unknown.0.Foo")
	assert.False(t, got.Found)
	assert.Equal(t, "Device.Unknown.0.Foo", got.Translated)
}

func TestTranslator_ToStandard_RuntimeInstanceFoldsToTemplate(t *testing.T) {
	set := &MappingSet{
		Source: MappingSourceDefault,
		Mappings: []ParamMapping{
			mkMapping(
				"Device.FaultMgmt.CurrentAlarm.{i}.PerceivedSeverity",
				"X_BAI_Alarm.Cur.{i}.Severity",
			),
		},
	}
	tr := NewTranslator(set, NewRegistryMetrics(nil), nil)

	// 私网带运行时实例号 → 命中并按位置回填到 standard 模板
	got := tr.ToStandard("X_BAI_Alarm.Cur.42.Severity")
	require.True(t, got.Found)
	assert.Equal(t, "Device.FaultMgmt.CurrentAlarm.42.PerceivedSeverity", got.Translated)

	// 未匹配 — 静默 miss
	got = tr.ToStandard("Unknown.0.Path")
	assert.False(t, got.Found)
	assert.Equal(t, "Unknown.0.Path", got.Translated)
}

func TestTranslator_PartialPrefixDerivation(t *testing.T) {
	newTranslator := func(mappings ...ParamMapping) *Translator {
		return NewTranslator(
			&MappingSet{Source: MappingSourceDefault, Mappings: mappings},
			NewRegistryMetrics(nil),
			nil,
		)
	}

	baseMapping := mkMapping(
		"Device.A.{i}.B.{i}.Value",
		"InternetGatewayDevice.X.{i}.Pool.{i}.Val",
	)

	t.Run("one-level blank prefix", func(t *testing.T) {
		got := newTranslator(baseMapping).ToPrivate("Device.A.")

		require.True(t, got.Found)
		assert.Equal(t, "InternetGatewayDevice.X.", got.Translated)
		require.NotNil(t, got.Mapping)
	})

	t.Run("two-level prefix preserves outer runtime instance", func(t *testing.T) {
		got := newTranslator(baseMapping).ToPrivate("Device.A.1.B.")

		require.True(t, got.Found)
		assert.Equal(t, "InternetGatewayDevice.X.1.Pool.", got.Translated)
	})

	t.Run("template-form prior placeholder", func(t *testing.T) {
		got := newTranslator(baseMapping).ToPrivate("Device.A.{i}.B.")

		require.True(t, got.Found)
		assert.Equal(t, "InternetGatewayDevice.X.{i}.Pool.", got.Translated)
	})

	t.Run("symmetric to-standard derivation", func(t *testing.T) {
		tr := newTranslator(baseMapping)

		blank := tr.ToStandardCandidates("InternetGatewayDevice.X.")
		require.Len(t, blank, 1)
		assert.Equal(t, "Device.A.", blank[0].Translated)

		resolved := tr.ToStandardCandidates("InternetGatewayDevice.X.7.Pool.")
		require.Len(t, resolved, 1)
		assert.Equal(t, "Device.A.7.B.", resolved[0].Translated)
	})

	multiCandidateTranslator := func() *Translator {
		return newTranslator(
			mkMapping(
				"Device.A.{i}.B.{i}.Value",
				"InternetGatewayDevice.Z.{i}.Pool.{i}.Val",
			),
			mkMapping(
				"Device.A.{i}.C.{i}.Value",
				"InternetGatewayDevice.X.{i}.Group.{i}.Val",
			),
			mkMapping(
				"Device.A.{i}.D.{i}.Value",
				"InternetGatewayDevice.Z.{i}.Other.{i}.Val",
			),
		)
	}

	t.Run("multiple candidates are deduplicated and deterministic", func(t *testing.T) {
		got := multiCandidateTranslator().ToPrivateCandidates("Device.A.")

		require.Len(t, got, 2)
		assert.Equal(t, "InternetGatewayDevice.X.", got[0].Translated)
		assert.Equal(t, "InternetGatewayDevice.Z.", got[1].Translated)
		assert.True(t, got[0].Found)
		assert.True(t, got[1].Found)
	})

	t.Run("legacy method misses for multiple candidates", func(t *testing.T) {
		got := multiCandidateTranslator().ToPrivate("Device.A.")

		assert.False(t, got.Found)
		assert.Equal(t, "Device.A.", got.Translated)
		assert.Nil(t, got.Mapping)
	})

	reverseMultiCandidateTranslator := func() *Translator {
		return newTranslator(
			mkMapping(
				"Device.Z.{i}.Pool.{i}.Value",
				"InternetGatewayDevice.Shared.{i}.B.{i}.Val",
			),
			mkMapping(
				"Device.X.{i}.Group.{i}.Value",
				"InternetGatewayDevice.Shared.{i}.C.{i}.Val",
			),
			mkMapping(
				"Device.X.{i}.Other.{i}.Value",
				"InternetGatewayDevice.Shared.{i}.D.{i}.Val",
			),
		)
	}

	t.Run("reverse multiple candidates are deduplicated and deterministic", func(t *testing.T) {
		got := reverseMultiCandidateTranslator().ToStandardCandidates("InternetGatewayDevice.Shared.")

		require.Len(t, got, 2)
		assert.Equal(t, "Device.X.", got[0].Translated)
		assert.Equal(t, "Device.Z.", got[1].Translated)
		assert.True(t, got[0].Found)
		assert.True(t, got[1].Found)
	})

	t.Run("reverse legacy method misses for multiple candidates", func(t *testing.T) {
		got := reverseMultiCandidateTranslator().ToStandard("InternetGatewayDevice.Shared.")

		assert.False(t, got.Found)
		assert.Equal(t, "InternetGatewayDevice.Shared.", got.Translated)
		assert.Nil(t, got.Mapping)
	})

	t.Run("exact direct mapping takes precedence", func(t *testing.T) {
		tr := newTranslator(
			baseMapping,
			mkMapping("Device.A.", "InternetGatewayDevice.Direct."),
		)

		candidates := tr.ToPrivateCandidates("Device.A.")
		require.Len(t, candidates, 1)
		assert.Equal(t, "InternetGatewayDevice.Direct.", candidates[0].Translated)

		legacy := tr.ToPrivate("Device.A.")
		require.True(t, legacy.Found)
		assert.Equal(t, "InternetGatewayDevice.Direct.", legacy.Translated)
	})

	t.Run("reverse exact direct mapping takes precedence", func(t *testing.T) {
		tr := newTranslator(
			mkMapping(
				"Device.Derived.{i}.Value",
				"InternetGatewayDevice.Shared.{i}.Val",
			),
			mkMapping("Device.Direct.", "InternetGatewayDevice.Shared."),
		)

		candidates := tr.ToStandardCandidates("InternetGatewayDevice.Shared.")
		require.Len(t, candidates, 1)
		assert.Equal(t, "Device.Direct.", candidates[0].Translated)

		legacy := tr.ToStandard("InternetGatewayDevice.Shared.")
		require.True(t, legacy.Found)
		assert.Equal(t, "Device.Direct.", legacy.Translated)
	})

	t.Run("existing full concrete leaf translation remains unchanged", func(t *testing.T) {
		tr := newTranslator(baseMapping)

		candidates := tr.ToPrivateCandidates("Device.A.7.B.3.Value")
		require.Len(t, candidates, 1)
		assert.Equal(t, "InternetGatewayDevice.X.7.Pool.3.Val", candidates[0].Translated)

		legacy := tr.ToPrivate("Device.A.7.B.3.Value")
		require.True(t, legacy.Found)
		assert.Equal(t, "InternetGatewayDevice.X.7.Pool.3.Val", legacy.Translated)
	})
}

func TestTranslator_PartialPrefixPreservesFixedNumericSegments(t *testing.T) {
	tr := NewTranslator(
		&MappingSet{
			Source: MappingSourceDefault,
			Mappings: []ParamMapping{mkMapping(
				"Device.A.{i}.Profile.1.B.{i}.Value",
				"InternetGatewayDevice.X.{i}.Profile.1.Pool.{i}.Val",
			)},
		},
		NewRegistryMetrics(nil),
		nil,
	)

	toPrivate := tr.ToPrivate("Device.A.7.Profile.1.B.")
	require.True(t, toPrivate.Found)
	assert.Equal(t, "InternetGatewayDevice.X.7.Profile.1.Pool.", toPrivate.Translated)

	privateCandidates := tr.ToPrivateCandidates("Device.A.7.Profile.1.B.")
	require.Len(t, privateCandidates, 1)
	assert.Equal(t, "InternetGatewayDevice.X.7.Profile.1.Pool.", privateCandidates[0].Translated)

	toStandard := tr.ToStandard("InternetGatewayDevice.X.7.Profile.1.Pool.")
	require.True(t, toStandard.Found)
	assert.Equal(t, "Device.A.7.Profile.1.B.", toStandard.Translated)

	standardCandidates := tr.ToStandardCandidates("InternetGatewayDevice.X.7.Profile.1.Pool.")
	require.Len(t, standardCandidates, 1)
	assert.Equal(t, "Device.A.7.Profile.1.B.", standardCandidates[0].Translated)
}

func TestTranslator_LegacyExactFastPathDoesNotAllocateCandidateSlice(t *testing.T) {
	tr := NewTranslator(
		&MappingSet{
			Source: MappingSourceDefault,
			Mappings: []ParamMapping{
				mkMapping("Device.Exact", "InternetGatewayDevice.Exact"),
			},
		},
		nil,
		nil,
	)

	var privateResult TranslationResult
	var standardResult TranslationResult
	allocs := testing.AllocsPerRun(1000, func() {
		privateResult = tr.ToPrivate("Device.Exact")
		standardResult = tr.ToStandard("InternetGatewayDevice.Exact")
	})

	assert.Zero(t, allocs)
	assert.True(t, privateResult.Found)
	assert.True(t, standardResult.Found)
}

func TestTranslator_CandidateAndLegacyMetrics(t *testing.T) {
	metrics := NewRegistryMetrics(nil)
	tr := NewTranslator(
		&MappingSet{
			Source: MappingSourceDefault,
			Mappings: []ParamMapping{
				mkMapping("Device.A.{i}.B", "InternetGatewayDevice.X.{i}.B"),
				mkMapping("Device.A.{i}.C", "InternetGatewayDevice.Y.{i}.C"),
			},
		},
		metrics,
		nil,
	)

	require.Len(t, tr.ToPrivateCandidates("Device.A."), 2)
	assert.False(t, tr.ToPrivate("Device.A.").Found)

	assert.Equal(t, float64(1), testutil.ToFloat64(
		metrics.translateTotal.WithLabelValues("to_private", "hit"),
	))
	assert.Equal(t, float64(1), testutil.ToFloat64(
		metrics.translateTotal.WithLabelValues("to_private", "miss"),
	))
}

func TestSubstituteInstanceNumbers(t *testing.T) {
	cases := []struct {
		name            string
		instanceNumbers []string
		dst             string
		want            string
		ok              bool
	}{
		{"single_i",
			[]string{"0"}, "Device.Other.{i}.Baz",
			"Device.Other.0.Baz", true},
		{"multi_i_in_order",
			[]string{"7", "3"}, "X.{i}.Y.{i}.Z",
			"X.7.Y.3.Z", true},
		{"fixed_numeric_segment_preserved",
			[]string{"7", "3"}, "X.{i}.Profile.1.Y.{i}.Z",
			"X.7.Profile.1.Y.3.Z", true},
		{"no_instance_number_for_placeholder",
			nil, "X.{i}.Y",
			"", false},
		{"extra_instance_number",
			[]string{"5"}, "X.Y",
			"", false},
		{"empty_destination",
			[]string{"5"}, "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := substituteInstanceNumbers(c.instanceNumbers, c.dst)
			assert.Equal(t, c.ok, ok)
			assert.Equal(t, c.want, got)
		})
	}
}

package parammodel

import (
	"testing"

	"github.com/google/uuid"
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
			mkMapping("Device.Bad.{i}.X", "Device.Priv.X"),     // 跳过
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

func TestSubstituteInstanceNumbers(t *testing.T) {
	cases := []struct {
		name     string
		src, dst string
		want     string
		ok       bool
	}{
		{"single_i",
			"Device.Foo.0.Bar", "Device.Other.{i}.Baz",
			"Device.Other.0.Baz", true},
		{"multi_i_in_order",
			"A.7.B.3.C", "X.{i}.Y.{i}.Z",
			"X.7.Y.3.Z", true},
		{"src_no_digits_dst_has_i",
			"A.B", "X.{i}.Y",
			"", false},
		{"dst_no_i_src_has_digits",
			"A.5.B", "X.Y",
			"", false},
		{"empty_inputs",
			"", "X.{i}", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := substituteInstanceNumbers(c.src, c.dst)
			assert.Equal(t, c.ok, ok)
			assert.Equal(t, c.want, got)
		})
	}
}

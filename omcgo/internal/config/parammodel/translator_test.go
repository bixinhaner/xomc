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

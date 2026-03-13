package carrier

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registryMockCarrier is a minimal mock implementing the Carrier interface
// for registry unit tests.
type registryMockCarrier struct {
	code          model.CarrierCode
	technologies  []model.Technology
	ouiProductMap map[model.Technology][]OUIProductClassInfo
}

func (m *registryMockCarrier) Code() model.CarrierCode                  { return m.code }
func (m *registryMockCarrier) Name() string                             { return string(m.code) }
func (m *registryMockCarrier) SupportedTechnologies() []model.Technology { return m.technologies }
func (m *registryMockCarrier) DefaultDataModelVersions(_ model.Technology) []string { return nil }
func (m *registryMockCarrier) KnownOUIProductClasses(tech model.Technology) []OUIProductClassInfo {
	if m.ouiProductMap == nil {
		return nil
	}
	return m.ouiProductMap[tech]
}
func (m *registryMockCarrier) MapParameterToUnified(_ string) string                        { return "" }
func (m *registryMockCarrier) MapUnifiedToParameter(_ string) string                        { return "" }
func (m *registryMockCarrier) ProvisioningTemplates(_ model.Technology) []*ProvisionTemplate { return nil }
func (m *registryMockCarrier) KPIDefinitions(_ model.Technology) []*KPIDefinition            { return nil }
func (m *registryMockCarrier) AlarmSeverityMapping(_ string) model.AlarmSeverity             { return 0 }
func (m *registryMockCarrier) ValidateParameter(_ string, _ string) error                    { return nil }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)
	assert.Empty(t, r.All(), "new registry should have no carriers")
}

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	mock := &registryMockCarrier{code: "test_carrier"}
	r.Register(mock)

	got, err := r.Get("test_carrier")
	require.NoError(t, err)
	assert.Equal(t, model.CarrierCode("test_carrier"), got.Code())
}

func TestGetNotFound(t *testing.T) {
	r := NewRegistry()
	got, err := r.Get("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "carrier not found")
}

func TestMustGetPanic(t *testing.T) {
	r := NewRegistry()
	assert.Panics(t, func() {
		r.MustGet("nonexistent")
	})
}

func TestMustGetSuccess(t *testing.T) {
	r := NewRegistry()
	r.Register(&registryMockCarrier{code: "safe"})
	assert.NotPanics(t, func() {
		got := r.MustGet("safe")
		assert.Equal(t, model.CarrierCode("safe"), got.Code())
	})
}

func TestAll(t *testing.T) {
	r := NewRegistry()
	r.Register(&registryMockCarrier{code: "alpha"})
	r.Register(&registryMockCarrier{code: "beta"})

	all := r.All()
	assert.Len(t, all, 2)

	codes := make(map[model.CarrierCode]bool)
	for _, c := range all {
		codes[c.Code()] = true
	}
	assert.True(t, codes["alpha"])
	assert.True(t, codes["beta"])
}

func TestResolveByOUI_Match(t *testing.T) {
	r := NewRegistry()
	r.Register(&registryMockCarrier{
		code:         "carrier_a",
		technologies: []model.Technology{model.TechLTE},
		ouiProductMap: map[model.Technology][]OUIProductClassInfo{
			model.TechLTE: {
				{OUI: "AABBCC", ProductClass: "SmallCell"},
				{OUI: "112233", ProductClass: "Pico"},
			},
		},
	})

	got := r.ResolveByOUI("112233")
	assert.Equal(t, model.CarrierCode("carrier_a"), got)
}

func TestResolveByOUI_NoMatch(t *testing.T) {
	r := NewRegistry()
	r.Register(&registryMockCarrier{
		code:         "carrier_b",
		technologies: []model.Technology{model.TechNR},
		ouiProductMap: map[model.Technology][]OUIProductClassInfo{
			model.TechNR: {{OUI: "DDEEFF", ProductClass: "gNB"}},
		},
	})

	got := r.ResolveByOUI("FFFFFF")
	assert.Equal(t, model.CarrierCode(""), got)
}

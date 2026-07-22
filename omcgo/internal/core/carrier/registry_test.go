package carrier

import (
	"errors"
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
	// unsupportedMR lists MR types this mock carrier does NOT support. nil =
	// supports everything (default for registry tests that don't care).
	unsupportedMR map[model.MRType]bool
}

func (m *registryMockCarrier) Code() model.CarrierCode                              { return m.code }
func (m *registryMockCarrier) Name() string                                         { return string(m.code) }
func (m *registryMockCarrier) SupportedTechnologies() []model.Technology            { return m.technologies }
func (m *registryMockCarrier) DefaultDataModelVersions(_ model.Technology) []string { return nil }
func (m *registryMockCarrier) KnownOUIProductClasses(tech model.Technology) []OUIProductClassInfo {
	if m.ouiProductMap == nil {
		return nil
	}
	return m.ouiProductMap[tech]
}
func (m *registryMockCarrier) MapParameterToUnified(_ string) string { return "" }
func (m *registryMockCarrier) MapUnifiedToParameter(_ string) string { return "" }
func (m *registryMockCarrier) ProvisioningTemplates(_ model.Technology) []*ProvisionTemplate {
	return nil
}
func (m *registryMockCarrier) AlarmSeverityMapping(_ string) model.AlarmSeverity        { return 0 }
func (m *registryMockCarrier) ValidateParameter(_ string, _ string) error               { return nil }
func (m *registryMockCarrier) GetInfoParamMapping(_ model.Technology) map[string]string { return nil }
func (m *registryMockCarrier) RFControlPath(_ model.Technology) string                  { return "" }
func (m *registryMockCarrier) SupportsMRType(mrType model.MRType) bool {
	return !m.unsupportedMR[mrType]
}

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

func TestMustGetError(t *testing.T) {
	r := NewRegistry()
	_, err := r.MustGet("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "carrier not found")
}

func TestMustGetSuccess(t *testing.T) {
	r := NewRegistry()
	r.Register(&registryMockCarrier{code: "safe"})
	got, err := r.MustGet("safe")
	require.NoError(t, err)
	assert.Equal(t, model.CarrierCode("safe"), got.Code())
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

func TestResolveByIdentity_DisambiguatesSharedOUIByProductClass(t *testing.T) {
	r := NewRegistry()
	sharedOUI := "AABBCC"
	newCarrier := func(code model.CarrierCode, productClass string) *registryMockCarrier {
		return &registryMockCarrier{
			code:         code,
			technologies: []model.Technology{model.TechLTE},
			ouiProductMap: map[model.Technology][]OUIProductClassInfo{
				model.TechLTE: {{OUI: sharedOUI, ProductClass: productClass}},
			},
		}
	}

	r.Register(newCarrier(model.CarrierCMCC, "SmallCell-LTE"))
	r.Register(newCarrier(model.CarrierCTCC, "eSmallCell-LTE"))

	got, err := r.ResolveByIdentity(sharedOUI, "eSmallCell-LTE")
	require.NoError(t, err)
	assert.Equal(t, model.CarrierCTCC, got)
}

func TestResolveByIdentity_RejectsAmbiguousSharedOUIWithoutProductClass(t *testing.T) {
	r := NewRegistry()
	sharedOUI := "AABBCC"
	r.Register(&registryMockCarrier{
		code:         model.CarrierCMCC,
		technologies: []model.Technology{model.TechLTE},
		ouiProductMap: map[model.Technology][]OUIProductClassInfo{
			model.TechLTE: {{OUI: sharedOUI, ProductClass: "SmallCell-LTE"}},
		},
	})
	r.Register(&registryMockCarrier{
		code:         model.CarrierCTCC,
		technologies: []model.Technology{model.TechLTE},
		ouiProductMap: map[model.Technology][]OUIProductClassInfo{
			model.TechLTE: {{OUI: sharedOUI, ProductClass: "eSmallCell-LTE"}},
		},
	})

	got, err := r.ResolveByIdentity(sharedOUI, "")
	assert.Empty(t, got)
	assert.True(t, errors.Is(err, ErrAmbiguousCarrier))
}

func TestResolveByIdentity_UsesUniqueOUIMatchWhenProductClassIsUnknown(t *testing.T) {
	r := NewRegistry()
	r.Register(&registryMockCarrier{
		code:         model.CarrierCUCC,
		technologies: []model.Technology{model.TechNR},
		ouiProductMap: map[model.Technology][]OUIProductClassInfo{
			model.TechNR: {{OUI: "DDEEFF", ProductClass: "SmallCell-NR-CU"}},
		},
	})

	got, err := r.ResolveByIdentity("DDEEFF", "unknown-product")
	require.NoError(t, err)
	assert.Equal(t, model.CarrierCUCC, got)
}

func TestResolveByIdentity_ReturnsNoMatchForUnknownOUI(t *testing.T) {
	r := NewRegistry()
	r.Register(&registryMockCarrier{
		code:         model.CarrierCMCC,
		technologies: []model.Technology{model.TechLTE},
		ouiProductMap: map[model.Technology][]OUIProductClassInfo{
			model.TechLTE: {{OUI: "AABBCC", ProductClass: "SmallCell-LTE"}},
		},
	})

	got, err := r.ResolveByIdentity("FFFFFF", "Unknown")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// #17: DefaultCarrier 取代 InformHandler 处硬编码的 CarrierCMCC 默认值。

func TestDefaultCarrier_Empty(t *testing.T) {
	r := NewRegistry()
	assert.Equal(t, model.CarrierCode(""), r.DefaultCarrier(),
		"empty registry must return empty default so callers can detect misconfiguration")
}

func TestDefaultCarrier_PrefersCMCC(t *testing.T) {
	r := NewRegistry()
	// 注册顺序故意把 CMCC 放最后，验证默认值与注册顺序/ map 迭代序无关。
	r.Register(&registryMockCarrier{code: model.CarrierCUCC})
	r.Register(&registryMockCarrier{code: model.CarrierCTCC})
	r.Register(&registryMockCarrier{code: model.CarrierCMCC})

	assert.Equal(t, model.CarrierCMCC, r.DefaultCarrier(),
		"CMCC must win when registered (dominant deployment)")
}

func TestDefaultCarrier_DeterministicWithoutCMCC(t *testing.T) {
	r := NewRegistry()
	r.Register(&registryMockCarrier{code: model.CarrierCUCC})
	r.Register(&registryMockCarrier{code: model.CarrierCTCC})

	// 无 CMCC 时取字典序最小（ctcc < cucc），且多次调用稳定。
	first := r.DefaultCarrier()
	assert.Equal(t, model.CarrierCTCC, first)
	for i := 0; i < 5; i++ {
		assert.Equal(t, first, r.DefaultCarrier(), "default must be stable across calls")
	}
}

// #17: SupportsMRType 把 "运营商是否采集某 MR 类型" 的判定下沉到 Carrier 适配器。

func TestRegistrySupportsMRType_Delegates(t *testing.T) {
	r := NewRegistry()
	r.Register(&registryMockCarrier{
		code:          model.CarrierCUCC,
		unsupportedMR: map[model.MRType]bool{model.MRTypeMRE: true},
	})
	r.Register(&registryMockCarrier{code: model.CarrierCMCC})

	// CMCC 支持 MRE；CUCC 不支持 MRE 但支持 MRO/MRS。
	assert.True(t, r.SupportsMRType(model.CarrierCMCC, model.MRTypeMRE))
	assert.False(t, r.SupportsMRType(model.CarrierCUCC, model.MRTypeMRE))
	assert.True(t, r.SupportsMRType(model.CarrierCUCC, model.MRTypeMRO))
	assert.True(t, r.SupportsMRType(model.CarrierCUCC, model.MRTypeMRS))
}

func TestRegistrySupportsMRType_UnknownCarrierFailsClosed(t *testing.T) {
	r := NewRegistry()
	r.Register(&registryMockCarrier{code: model.CarrierCMCC})

	// 未注册的运营商 → false（fail-closed，调用方需当作配置错误处理）。
	assert.False(t, r.SupportsMRType("unknown_carrier", model.MRTypeMRO))
}

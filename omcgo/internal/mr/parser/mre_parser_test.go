package parser

import (
	"os"
	"strings"
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMREParser_Parse(t *testing.T) {
	f, err := os.Open("../../../test/fixtures/mr/sample_mre.xml")
	require.NoError(t, err)
	defer f.Close()

	p := NewMREParser()
	data, err := p.Parse(f, model.CarrierCMCC)
	require.NoError(t, err)

	assert.Equal(t, "mre", data.MRType)
	assert.Equal(t, "ENB001", data.DeviceSN)
	assert.Len(t, data.Records, 2)

	first := data.Records[0]
	assert.Contains(t, first.MeasurementData, "MR.UeCategory")
}

func TestMREParser_CUCCNotSupported(t *testing.T) {
	r := strings.NewReader(`<?xml version="1.0"?><bulkPmMrDataFile></bulkPmMrDataFile>`)
	p := NewMREParser()
	_, err := p.Parse(r, model.CarrierCUCC)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestMREParser_CMCCSupported(t *testing.T) {
	f, err := os.Open("../../../test/fixtures/mr/sample_mre.xml")
	require.NoError(t, err)
	defer f.Close()

	p := NewMREParser()
	data, err := p.Parse(f, model.CarrierCMCC)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestMREParser_CTCCSupported(t *testing.T) {
	f, err := os.Open("../../../test/fixtures/mr/sample_mre.xml")
	require.NoError(t, err)
	defer f.Close()

	p := NewMREParser()
	data, err := p.Parse(f, model.CarrierCTCC)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

// stubMRSupport lets tests drive the carrier-support decision without depending
// on the carrier package — proving the MRE parser no longer hardcodes carrier
// identity (#17).
type stubMRSupport struct {
	supported bool
	gotCode   model.CarrierCode
	gotType   model.MRType
}

func (s *stubMRSupport) SupportsMRType(code model.CarrierCode, mrType model.MRType) bool {
	s.gotCode = code
	s.gotType = mrType
	return s.supported
}

// #17: 注入的 checker 说"不支持"时，无论 carrier 是哪家都拒绝 —— 证明判定
// 已下沉到 Carrier 适配器，不再是 parser 内的 "if carrier == cucc"。
func TestMREParser_InjectedCheckerRejects(t *testing.T) {
	stub := &stubMRSupport{supported: false}
	p := NewMREParser(stub)

	// 故意用 CMCC（内置回退表里是支持的），验证决策完全交给了注入的 checker。
	_, err := p.Parse(strings.NewReader("<x/>"), model.CarrierCMCC)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
	assert.Equal(t, model.CarrierCMCC, stub.gotCode, "parser must pass carrier code through to checker")
	assert.Equal(t, model.MRTypeMRE, stub.gotType, "parser must query MRE support specifically")
}

// #17: 注入的 checker 说"支持"时，即便 carrier 是 CUCC（回退表里不支持）也照常解析 ——
// 同样证明判定来自注入的 checker，而非 parser 硬编码。
func TestMREParser_InjectedCheckerAllows(t *testing.T) {
	f, err := os.Open("../../../test/fixtures/mr/sample_mre.xml")
	require.NoError(t, err)
	defer f.Close()

	stub := &stubMRSupport{supported: true}
	p := NewMREParser(stub)

	data, err := p.Parse(f, model.CarrierCUCC)
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, model.CarrierCUCC, stub.gotCode)
}

// 回退路径（无注入 checker）仍保留历史行为：仅 CUCC 不支持 MRE。
func TestMREParser_DefaultFallbackMatchesAdapters(t *testing.T) {
	p := NewMREParser() // no checker → defaultMRESupport
	_, err := p.Parse(strings.NewReader("<x/>"), model.CarrierCUCC)
	assert.ErrorIs(t, err, ErrNotSupported)
}

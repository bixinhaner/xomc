package indicator

import (
	"testing"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeDataType(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"整数", DataTypeInt},
		{"整数n", DataTypeInt},
		{"Integer", DataTypeInt},
		{"int", DataTypeInt},
		{"实数", DataTypeReal},
		{"real", DataTypeReal},
		{"浮点数", DataTypeFloat},
		{"number", DataTypeFloat},
		{"float", DataTypeFloat},
		{" 整数 ", DataTypeInt}, // 去空白
		{"", ""},               // 空串保持空
		{"未知枚举", "未知枚举"},      // 未识别保守原样返回
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, normalizeDataType(c.in), "normalizeDataType(%q)", c.in)
	}
}

func TestDataTypeLabel_Locale(t *testing.T) {
	assert.Equal(t, "整数", dataTypeLabel(DataTypeInt, appcontext.LocaleZH))
	assert.Equal(t, "Integer", dataTypeLabel(DataTypeInt, appcontext.LocaleEN))
	assert.Equal(t, "实数", dataTypeLabel(DataTypeReal, appcontext.LocaleZH))
	assert.Equal(t, "Real", dataTypeLabel(DataTypeReal, appcontext.LocaleEN))
	assert.Equal(t, "浮点数", dataTypeLabel(DataTypeFloat, appcontext.LocaleZH))
	assert.Equal(t, "Float", dataTypeLabel(DataTypeFloat, appcontext.LocaleEN))

	// 未识别码 → 回退原码本身，保证不空白。
	assert.Equal(t, "weird", dataTypeLabel("weird", appcontext.LocaleEN))
}

func TestFillDataTypeLabel(t *testing.T) {
	code := DataTypeInt
	ind := &PerfIndicator{DataType: &code}
	fillDataTypeLabel(ind, appcontext.LocaleEN)
	if assert.NotNil(t, ind.DataTypeLabel) {
		assert.Equal(t, "Integer", *ind.DataTypeLabel)
	}

	// nil / 空 data_type → 不填 label。
	empty := &PerfIndicator{}
	fillDataTypeLabel(empty, appcontext.LocaleEN)
	assert.Nil(t, empty.DataTypeLabel)

	blank := ""
	indBlank := &PerfIndicator{DataType: &blank}
	fillDataTypeLabel(indBlank, appcontext.LocaleZH)
	assert.Nil(t, indBlank.DataTypeLabel)
}

func TestFillDataTypeLabels_Batch(t *testing.T) {
	iCode, fCode := DataTypeInt, DataTypeFloat
	items := []IndicatorListItem{
		{PerfIndicator: PerfIndicator{DataType: &iCode}},
		{PerfIndicator: PerfIndicator{DataType: &fCode}},
		{PerfIndicator: PerfIndicator{}}, // 无 data_type
	}
	fillDataTypeLabels(items, appcontext.LocaleZH)
	if assert.NotNil(t, items[0].DataTypeLabel) {
		assert.Equal(t, "整数", *items[0].DataTypeLabel)
	}
	if assert.NotNil(t, items[1].DataTypeLabel) {
		assert.Equal(t, "浮点数", *items[1].DataTypeLabel)
	}
	assert.Nil(t, items[2].DataTypeLabel)
}

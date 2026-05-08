package product

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndicatorTableByDeviceType(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"enb", "perf_indicators_enb", true},
		{"gsm", "perf_indicators_gsm", true},
		{"gnb", "perf_indicators_gnb", true},
		{"unknown", "", false},
		{"", "", false},
		{"ENB", "", false}, // 大小写敏感（与 P1-06 indicator/loader.go 保持一致）
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, ok := indicatorTableByDeviceType(c.in)
			assert.Equal(t, c.want, got)
			assert.Equal(t, c.wantOK, ok)
		})
	}
}

func TestNewPgRepository(t *testing.T) {
	// 仅校验构造器不 panic 且返回非 nil；DB 真连接走集成测试。
	r := NewPgRepository(nil)
	assert.NotNil(t, r)
	assert.Nil(t, r.pool)
}

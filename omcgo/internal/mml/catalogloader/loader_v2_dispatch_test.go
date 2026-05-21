package catalogloader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsV2CatalogFile 验证 .v2.json 后缀判定。
func TestIsV2CatalogFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/tmp/cmcc-tdlte-v2.3.v2.json", true},
		{"./mml-catalog/cmcc-tdlte-v2.3.v2.json", true},
		{"cmcc-tdlte-v2.3.json", false},
		{"x.json", false},
		{"x.v2", false},               // 没有 .json
		{"x.v2.json.bak", false},      // 后缀不是 .v2.json
		{"foo.v2.json", true},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			assert.Equal(t, c.want, isV2CatalogFile(c.path))
		})
	}
}

// TestWithV2Schema_DefaultOff 验证 NewLoader 不带 Option 时 useV2Schema=false。
func TestWithV2Schema_DefaultOff(t *testing.T) {
	l := NewLoader(nil, "/tmp/none", nil)
	assert.False(t, l.useV2Schema, "default constructor should leave v2 schema off")
}

// TestWithV2Schema_OptionEnables 验证 WithV2Schema(true) 显式翻转。
func TestWithV2Schema_OptionEnables(t *testing.T) {
	l := NewLoader(nil, "/tmp/none", nil, WithV2Schema(true))
	assert.True(t, l.useV2Schema)
}

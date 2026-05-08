package parammodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 集成测试见 P3-02（admin handler 起来后用 dockertest）；本处仅冒烟构造。
func TestNewPgRepository(t *testing.T) {
	r := NewPgRepository(nil)
	assert.NotNil(t, r)
}

func TestStrDeref(t *testing.T) {
	assert.Equal(t, "", strDeref(nil))
	v := "x"
	assert.Equal(t, "x", strDeref(&v))
}

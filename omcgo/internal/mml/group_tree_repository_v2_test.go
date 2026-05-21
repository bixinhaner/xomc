package mml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPgGroupTreeRepository_DefaultV1 验证默认构造时 v2Mode=false。
func TestPgGroupTreeRepository_DefaultV1(t *testing.T) {
	r := NewPgGroupTreeRepository(nil)
	assert.False(t, r.v2Mode, "default constructor must not enable v2 mode")
}

// TestPgGroupTreeRepository_WithV2ModeOpt 验证 Option 显式翻转。
func TestPgGroupTreeRepository_WithV2ModeOpt(t *testing.T) {
	r := NewPgGroupTreeRepository(nil, WithV2Mode(true))
	assert.True(t, r.v2Mode)
}

// TestPgGroupTreeRepository_WithV2ModeFalseOpt 显式 false（无 op）也应保持 false。
func TestPgGroupTreeRepository_WithV2ModeFalseOpt(t *testing.T) {
	r := NewPgGroupTreeRepository(nil, WithV2Mode(false))
	assert.False(t, r.v2Mode)
}

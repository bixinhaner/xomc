package acs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdmissionController(t *testing.T) {
	ac := NewAdmissionController(2)

	assert.True(t, ac.Acquire())
	assert.Equal(t, int64(1), ac.Current())

	assert.True(t, ac.Acquire())
	assert.Equal(t, int64(2), ac.Current())

	// Should be denied
	assert.False(t, ac.Acquire())
	assert.Equal(t, int64(2), ac.Current())

	ac.Release()
	assert.Equal(t, int64(1), ac.Current())

	// Now should be admitted
	assert.True(t, ac.Acquire())
}

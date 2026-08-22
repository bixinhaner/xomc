package nats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEffectiveMaxReconnectDefaultsToInfinite(t *testing.T) {
	assert.Equal(t, -1, effectiveMaxReconnect(0))
	assert.Equal(t, -1, effectiveMaxReconnect(-1))
	assert.Equal(t, 12, effectiveMaxReconnect(12))
}

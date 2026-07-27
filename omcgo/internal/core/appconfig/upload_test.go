package appconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUploadConfigEffectivePMDedupTTL(t *testing.T) {
	assert.Equal(t, 4*time.Hour, (UploadConfig{}).EffectivePMDedupTTL())
	assert.Equal(t, 6*time.Hour, (UploadConfig{PMDedupTTL: 6 * time.Hour}).EffectivePMDedupTTL())
	assert.Equal(t, 4*time.Hour, (UploadConfig{PMDedupTTL: -time.Hour}).EffectivePMDedupTTL())
}

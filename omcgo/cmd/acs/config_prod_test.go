package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProductionConfigSupportsThirtyThousandConcurrentSessions(t *testing.T) {
	raw, err := os.ReadFile("etc/config.prod.yaml")
	require.NoError(t, err)

	var cfg struct {
		Session struct {
			MaxConcurrent int64 `yaml:"max_concurrent"`
		} `yaml:"session"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &cfg))
	require.Equal(t, int64(30000), cfg.Session.MaxConcurrent)
}

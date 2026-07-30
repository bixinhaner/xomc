package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestAdmissionGateSuppressesAllConfiguredLoggerOutputs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	gate := NewAdmissionGate()
	log, err := NewLoggerWithAdmissionGate(appconfig.LogConfig{
		Level:       "info",
		Format:      "json",
		OutputPaths: []string{path},
		Rotation:    appconfig.RotationConfig{Enabled: false},
	}, gate)
	require.NoError(t, err)

	log.Info("before-block")
	gate.SetBlocked(true)
	log.Info("while-blocked")
	log.Error("error-while-blocked")
	gate.SetBlocked(false)
	log.Info("after-recovery")
	require.NoError(t, log.Sync())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "before-block")
	require.Contains(t, content, "after-recovery")
	require.NotContains(t, content, "while-blocked")
	require.NotContains(t, content, "error-while-blocked")
	// The gate suppresses the core before JSON encoding, so blocked messages
	// cannot leak through as a partial line or an empty encoded entry.
	require.Equal(t, 2, strings.Count(content, "\n"))
}

func TestAdmissionLevelEnablerStillHonorsConfiguredLevel(t *testing.T) {
	gate := NewAdmissionGate()
	enabler := AdmissionLevelEnabler(gate, levelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zapcore.WarnLevel
	}))
	require.False(t, enabler.Enabled(zapcore.InfoLevel))
	require.True(t, enabler.Enabled(zapcore.WarnLevel))
	gate.SetBlocked(true)
	require.False(t, enabler.Enabled(zapcore.ErrorLevel))
}

package main

import (
	"testing"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewWorkerParamRegistryReusesProcessRegistry(t *testing.T) {
	existing := &parammodel.Registry{}
	w := &workerInfra{ParamRegistry: existing}

	got := newWorkerParamRegistry(w, nil, zap.NewNop())

	require.Same(t, existing, got)
	require.Same(t, existing, w.ParamRegistry)
}

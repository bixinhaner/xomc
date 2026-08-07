package main

import (
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/stretchr/testify/require"
)

func TestZedSummarySnapshot_MapsTechnologyCountsAndCPEExclusion(t *testing.T) {
	input := &device.TechnologyStatusSummarySet{ByTechnology: map[model.Technology]device.TechnologyStatusSummary{
		model.TechGSM: {Total: 12, ExcludedCPE: 1},
		model.TechLTE: {Total: 81, Online: 14, Activated: 10, ExcludedCPE: 2},
		model.TechNR:  {Total: 7, Online: 5, Activated: 4, ExcludedCPE: 3},
	}}

	snapshot := zedSummarySnapshot(input, time.Date(2026, 8, 10, 3, 0, 0, 0, time.UTC), "Asia/Shanghai")

	require.Equal(t, int64(12), snapshot.ByTechnology[model.TechGSM].Total)
	require.Equal(t, int64(14), snapshot.ByTechnology[model.TechLTE].Online)
	require.Equal(t, int64(4), snapshot.ByTechnology[model.TechNR].Activated)
	require.Equal(t, int64(6), snapshot.ExcludedCPE)
}

func TestProtectZedSummaryRecipients_DeduplicatesAddresses(t *testing.T) {
	protector := &testSummaryProtector{}
	recipients, err := protectZedSummaryRecipients(protector, []string{"NOC@example.com", "noc@example.com"})

	require.NoError(t, err)
	require.Len(t, recipients, 1)
	require.Equal(t, 1, protector.protectCalls)
}

type testSummaryProtector struct {
	protectCalls int
}

func (p *testSummaryProtector) Protect(_, _ string) ([]byte, int, []byte, error) {
	p.protectCalls++
	return []byte("ciphertext"), 1, []byte("fingerprint"), nil
}

func (*testSummaryProtector) Unprotect(string, []byte, int) (string, error) {
	return "noc@example.com", nil
}

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestACSTransferProtocolDefaultsAreFoldedIntoSeedBaseline(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	sql := string(contents)

	const marker = "-- Consolidated pre-release ACS transfer protocol defaults."
	index := strings.LastIndex(sql, marker)
	require.NotEqual(t, -1, index)
	section := sql[index:]

	require.Contains(t, section, "'acs_transfer', 'protocolPolicy', 'force_http'")
	require.Contains(t, section, "'acs_transfer', 'httpsUploadBaseURL', ''")
	require.Contains(t, section, "'acs_transfer', 'httpsDownloadBaseURL', ''")
}

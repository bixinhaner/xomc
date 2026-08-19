package mml

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGSMObjectSeedProvidesInstanceAwareCommands(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	sql := string(seed)

	start := strings.Index(sql, "-- GSM writable and multi-instance operation commands.")
	require.NotEqual(t, -1, start)
	endOffset := strings.Index(sql[start:], "-- GSM single-leaf commands should be one consolidated query command.")
	require.NotEqual(t, -1, endOffset)
	block := sql[start : start+endOffset]

	require.Contains(t, block, "replace(c.command_code, 'LST ', 'MOD ')")
	for _, code := range []string{
		"LST MML350_DEVICEGSM__BTS",
		"LST MML350_DEVICEGSM__BTS_TRX",
		"LST MML350_DEVICEGSM__BTS_TRX_TS",
		"LST MML350_DEVICEGSM__CS7INSTANCE",
		"LST MML350_DEVICEGSM__CS7INSTANCE_AS",
		"LST MML350_DEVICEGSM__CS7INSTANCE_ASP",
		"LST MML350_DEVICEGSM__CS7INSTANCE_SCCPADDR",
		"LST MML350_DEVICEGSM__GLOBAL",
		"LST MML350_DEVICEGSM__HANDOVER2",
		"LST MML350_DEVICEGSM__MGW",
		"LST MML350_DEVICEGSM__MSC",
	} {
		require.Contains(t, block, "'"+code+"'", code)
	}
	require.Contains(t, block, "('ADD', 'AddObject', '新增', 'Add')")
	require.Contains(t, block, "('RMV', 'DeleteObject', '删除', 'Delete')")
}

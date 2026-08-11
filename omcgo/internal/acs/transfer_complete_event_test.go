package acs

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransferCompleteEventIncludesSessionDeviceSN(t *testing.T) {
	evt, err := newTransferCompleteEvent(tr069.TransferComplete{CommandKey: "command_key"}, "  SN-1  ")
	require.NoError(t, err)

	assert.Equal(t, "SN-1", evt.Metadata[event.MetadataDeviceSN])
	var payload tr069.TransferComplete
	require.NoError(t, evt.DecodePayload(&payload))
	assert.Equal(t, "command_key", payload.CommandKey)
}

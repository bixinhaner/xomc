package collector

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeFileMarkerLookup struct {
	parsed   bool
	err      error
	deviceSN string
	fileName string
}

func (f *fakeFileMarkerLookup) IsFileParsed(_ context.Context, deviceSN, fileName string) (bool, error) {
	f.deviceSN = deviceSN
	f.fileName = fileName
	return f.parsed, f.err
}

func duplicateFileReceivedEvent(t *testing.T) event.Event {
	t.Helper()
	evt, err := event.NewEvent(event.SubjectPMFileReceived, FileReceivedPayload{
		MinIOPath:  "pm/SN001/report.xml",
		DeviceID:   uuid.NewString(),
		DeviceSN:   "SN001",
		DeviceOUI:  "ABCDEF",
		Carrier:    "cmcc",
		Technology: "lte",
	})
	require.NoError(t, err)
	return evt
}

func TestHandleFileReceived_ParsedMarkerStillChecksSourceForChangedContent(t *testing.T) {
	lookup := &fakeFileMarkerLookup{parsed: true}
	c := &PMCollector{
		logger:           zap.NewNop(),
		fileMarkerLookup: lookup,
	}

	err := c.handleFileReceived(context.Background(), duplicateFileReceivedEvent(t))

	require.ErrorContains(t, err, "download pm file")
	require.Equal(t, "SN001", lookup.deviceSN)
	require.Equal(t, "report.xml", lookup.fileName)
}

func TestHandleFileReceived_MarkerLookupError(t *testing.T) {
	lookup := &fakeFileMarkerLookup{err: errors.New("timescaledb unavailable")}
	c := &PMCollector{
		logger:           zap.NewNop(),
		fileMarkerLookup: lookup,
	}

	err := c.handleFileReceived(context.Background(), duplicateFileReceivedEvent(t))

	require.ErrorContains(t, err, "lookup parsed pm file marker")
	require.ErrorContains(t, err, "timescaledb unavailable")
}

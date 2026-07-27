package collector

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type recordingRawArchiver struct {
	scheduled []string
}

func (a *recordingRawArchiver) Schedule(bucket, object string, _ func(context.Context, string, string, string)) {
	a.scheduled = append(a.scheduled, bucket+"/"+object)
}

func (a *recordingRawArchiver) RemoveOld(context.Context, string, string) {}

func TestIngestViaCopyPersistsDetectedGzipAndSkipsRedundantArchive(t *testing.T) {
	copyIngestor := &recordingCopyIngestor{ingested: true}
	archiver := &recordingRawArchiver{}
	c := &PMCollector{
		bucket:       "pm",
		copyIngestor: copyIngestor,
		archiver:     archiver,
		eventBus:     noopEventBus{},
		logger:       zap.NewNop(),
	}
	payload := &FileReceivedPayload{
		MinIOPath: "2026/07/27/file.xml.gz", DeviceSN: "SN-1",
		Carrier: "cmcc", Technology: "lte",
	}
	content := &PMFileContent{CollectTime: time.Now(), Counters: []model.PMCounter{}}

	err := c.ingestViaCopy(
		context.Background(), trace.SpanFromContext(context.Background()), time.Now(), time.Now(),
		128, uuid.New(), payload, content, nil, true, make([]byte, 32),
	)

	require.NoError(t, err)
	assert.True(t, copyIngestor.marker.RawCompressed)
	assert.Empty(t, archiver.scheduled)
}

func TestIngestViaCopySchedulesPlainRawFile(t *testing.T) {
	copyIngestor := &recordingCopyIngestor{ingested: true}
	archiver := &recordingRawArchiver{}
	c := &PMCollector{
		bucket:       "pm",
		copyIngestor: copyIngestor,
		archiver:     archiver,
		eventBus:     noopEventBus{},
		logger:       zap.NewNop(),
	}
	payload := &FileReceivedPayload{
		MinIOPath: "2026/07/27/file.xml", DeviceSN: "SN-1",
		Carrier: "cmcc", Technology: "lte",
	}

	err := c.ingestViaCopy(
		context.Background(), trace.SpanFromContext(context.Background()), time.Now(), time.Now(),
		128, uuid.New(), payload, &PMFileContent{CollectTime: time.Now()}, nil, false, make([]byte, 32),
	)

	require.NoError(t, err)
	assert.False(t, copyIngestor.marker.RawCompressed)
	assert.Equal(t, []string{"pm/2026/07/27/file.xml"}, archiver.scheduled)
}

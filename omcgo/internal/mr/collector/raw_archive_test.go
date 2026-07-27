package collector

import (
	"context"
	"errors"
	"testing"

	"github.com/omcgo/omcgo/internal/mr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingMRStore struct {
	mr.MRStore
	renames map[string]string
	err     error
}

func (s *recordingMRStore) MarkCompressed(_ context.Context, renames map[string]string) error {
	s.renames = renames
	return s.err
}

type recordingMRArchiver struct {
	scheduled []string
}

func (a *recordingMRArchiver) Schedule(bucket, object string, _ func(context.Context, string, string, string)) {
	a.scheduled = append(a.scheduled, bucket+"/"+object)
}

func (a *recordingMRArchiver) RemoveOld(context.Context, string, string) {}

func TestFinalizeRawArchiveMarksDetectedGzipWithoutScheduling(t *testing.T) {
	store := &recordingMRStore{}
	archiver := &recordingMRArchiver{}
	c := &MRCollector{store: store, archiver: archiver, logger: zap.NewNop()}

	err := c.finalizeRawArchive(context.Background(), "mr", "file.xml.gz", true)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"file.xml.gz": "file.xml.gz"}, store.renames)
	assert.Empty(t, archiver.scheduled)
}

func TestFinalizeRawArchiveSchedulesPlainObject(t *testing.T) {
	store := &recordingMRStore{}
	archiver := &recordingMRArchiver{}
	c := &MRCollector{store: store, archiver: archiver, logger: zap.NewNop()}

	err := c.finalizeRawArchive(context.Background(), "mr", "file.xml", false)

	require.NoError(t, err)
	assert.Nil(t, store.renames)
	assert.Equal(t, []string{"mr/file.xml"}, archiver.scheduled)
}

func TestFinalizeRawArchiveMarkerFailureDoesNotRetryParsedFile(t *testing.T) {
	store := &recordingMRStore{err: errors.New("database unavailable")}
	c := &MRCollector{store: store, logger: zap.NewNop()}

	err := c.finalizeRawArchive(context.Background(), "mr", "file.xml.gz", true)

	require.NoError(t, err)
}

package download

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingDownloadObjectClient struct {
	buckets []string
}

func (c *recordingDownloadObjectClient) GetObject(
	_ context.Context,
	bucket string,
	_ string,
	_ minio.GetObjectOptions,
) (*minio.Object, error) {
	c.buckets = append(c.buckets, bucket)
	return nil, errors.New("recording client stops after bucket capture")
}

type deadlineRecordingResponseWriter struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
}

func (w *deadlineRecordingResponseWriter) SetWriteDeadline(t time.Time) error {
	w.deadlines = append(w.deadlines, t)
	return nil
}

func TestHandler_ClearsWriteDeadlineForLargeFileDownloads(t *testing.T) {
	client := &recordingDownloadObjectClient{}
	handler := NewHandler(client, "", "", zap.NewNop())
	request := httptest.NewRequest(
		http.MethodGet,
		"/smallcell/FileDownloadService/firmware/img/package.img",
		nil,
	)
	response := &deadlineRecordingResponseWriter{ResponseRecorder: httptest.NewRecorder()}

	handler.ServeHTTP(response, request)

	require.NotEmpty(t, response.deadlines)
	require.True(t, response.deadlines[0].IsZero())
}

func TestHandler_LegacyInternalDownloadCallsMinIOWithCanonicalBucket(t *testing.T) {
	client := &recordingDownloadObjectClient{}
	handler := NewHandler(client, "", "", zap.NewNop())
	request := httptest.NewRequest(
		http.MethodGet,
		"/smallcell/FileDownloadService/config_backup/backup/SN001_CFG.xml",
		nil,
	)

	handler.ServeHTTP(httptest.NewRecorder(), request)

	require.Equal(t, []string{"config-backup"}, client.buckets)
}

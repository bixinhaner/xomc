package regularreport

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

type MinIOAttachmentStore struct {
	client *minio.Client
}

func NewMinIOAttachmentStore(client *minio.Client) *MinIOAttachmentStore {
	return &MinIOAttachmentStore{client: client}
}

func (s *MinIOAttachmentStore) Read(ctx context.Context, bucket, object string, maxBytes int64) ([]byte, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("object storage is not configured")
	}
	reader, err := s.client.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("open object %s/%s: %w", bucket, object, err)
	}
	defer reader.Close()
	stat, err := reader.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat object %s/%s: %w", bucket, object, err)
	}
	if stat.Size > maxBytes {
		return nil, fmt.Errorf("attachment exceeds %d bytes", maxBytes)
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read object %s/%s: %w", bucket, object, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("attachment exceeds %d bytes", maxBytes)
	}
	return data, nil
}

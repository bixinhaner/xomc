package pageconfig

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
)

type MinIOMRSourceStore struct {
	client *minio.Client
	bucket string
}

func NewMinIOMRSourceStore(client *minio.Client, bucket string) *MinIOMRSourceStore {
	return &MinIOMRSourceStore{
		client: client,
		bucket: strings.TrimSpace(bucket),
	}
}

func (s *MinIOMRSourceStore) Get(ctx context.Context, objectKey string) ([]byte, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("MR source MinIO client is not configured")
	}
	if s.bucket == "" {
		return nil, fmt.Errorf("MR source bucket is not configured")
	}
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return nil, fmt.Errorf("MR source object key is empty")
	}
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get MR source object: %w", err)
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read MR source object: %w", err)
	}
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("open gzip MR source object: %w", err)
		}
		defer reader.Close()
		plain, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read gzip MR source object: %w", err)
		}
		return plain, nil
	}
	return data, nil
}

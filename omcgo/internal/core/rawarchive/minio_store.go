package rawarchive

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

// minioStore 用 *minio.Client 实现 RawStore。
type minioStore struct {
	client *minio.Client
}

// NewMinIOStore 返回基于 MinIO 客户端的 RawStore。client 为 nil 时返回 nil，
// 调用方（New）据此降级为无操作 archiver。
func NewMinIOStore(client *minio.Client) RawStore {
	if client == nil {
		return nil
	}
	return &minioStore{client: client}
}

// ReadHead ranged-get 对象前 n 字节（仅探测魔数，避免整文件读）。对象短于 n 时返回实际字节。
func (s *minioStore) ReadHead(ctx context.Context, bucket, object string, n int) ([]byte, error) {
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(0, int64(n-1)); err != nil {
		return nil, fmt.Errorf("set range: %w", err)
	}
	obj, err := s.client.GetObject(ctx, bucket, object, opts)
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	buf := make([]byte, n)
	m, rerr := io.ReadFull(obj, buf)
	if rerr == io.EOF || rerr == io.ErrUnexpectedEOF {
		return buf[:m], nil // 对象短于 n 字节
	}
	if rerr != nil {
		return nil, rerr
	}
	return buf[:m], nil
}

// Get 返回对象完整字节流。
func (s *minioStore) Get(ctx context.Context, bucket, object string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Put 按原 key 覆盖写。
func (s *minioStore) Put(ctx context.Context, bucket, object string, r io.Reader, size int64, contentType, contentEncoding string) error {
	_, err := s.client.PutObject(ctx, bucket, object, r, size, minio.PutObjectOptions{
		ContentType:     contentType,
		ContentEncoding: contentEncoding,
	})
	return err
}

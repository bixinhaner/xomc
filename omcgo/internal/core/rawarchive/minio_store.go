package rawarchive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

// ErrObjectNotFound 表示对象在 MinIO 中已不存在（如被 retention/cleanup 删除，pm_files/
// mr_files 行尚存的孤儿）。compress() 据此把该次结果判为终态（outcomeNotFound），让 Sweeper
// 标记其 raw_compressed=true 移出待扫集，避免对已删对象永久重扫导致 Sweeper 在孤儿行处卡死、
// 饿死其后真正待压的文件（"保证压到"加固）。
var ErrObjectNotFound = errors.New("rawarchive: object not found")

// isNotFound 判定 MinIO 错误是否为对象/桶不存在。
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	code := minio.ToErrorResponse(err).Code
	return code == "NoSuchKey" || code == "NoSuchBucket"
}

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
		if isNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	defer obj.Close()
	buf := make([]byte, n)
	m, rerr := io.ReadFull(obj, buf)
	if rerr == io.EOF || rerr == io.ErrUnexpectedEOF {
		return buf[:m], nil // 对象短于 n 字节
	}
	if rerr != nil {
		// 对象不存在（孤儿 pm_files/mr_files 行：对象已被 retention 删，行尚存）→ 归一化为
		// ErrObjectNotFound，让 compress 判终态、Sweeper 标记并移出待扫集（不永久重扫卡死）。
		if isNotFound(rerr) {
			return nil, ErrObjectNotFound
		}
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

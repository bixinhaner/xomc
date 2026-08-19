package trace

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/omcgo/omcgo/internal/storageprotection"
)

// BulkStore 抽象 MinIO trace-bulk bucket 操作。
// 大报文（> MaxInlinePayloadBytes）GZIP 后写入；查询时按 object_key 拉回。
//
// 对象键规范：`{task_id}/{message_id}.xml.gz`
//   - task_id 前缀方便 purge 时批量删除
//   - 报文 ID 作为唯一标识
type BulkStore struct {
	client           *minio.Client
	bucket           string
	storageAdmission storageprotection.WriteAdmission
}

// NewBulkStore 构造函数。bucket 为空时 Enabled() 返 false，调用方应跳过。
func NewBulkStore(client *minio.Client, bucket string) *BulkStore {
	return &BulkStore{client: client, bucket: bucket}
}

func (b *BulkStore) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	b.storageAdmission = admission
}

// Enabled 当 MinIO 客户端 + bucket 配置齐全时返回 true。
func (b *BulkStore) Enabled() bool {
	return b != nil && b.client != nil && b.bucket != ""
}

// ObjectKey 生成对象键。
func (b *BulkStore) ObjectKey(taskID, msgID uuid.UUID) string {
	return taskID.String() + "/" + msgID.String() + ".xml.gz"
}

// Put GZIP 压缩 payload 后写入 MinIO；返回 object_key。
func (b *BulkStore) Put(ctx context.Context, taskID, msgID uuid.UUID, payload string) (string, error) {
	if !b.Enabled() {
		return "", fmt.Errorf("bulk store disabled")
	}
	if b.storageAdmission != nil {
		decision, err := b.storageAdmission.CheckPath(ctx, storageprotection.ProtectedPathIDMinIO, storageprotection.WriteScopeTrace)
		if err != nil {
			return "", fmt.Errorf("storage admission check: %w", err)
		}
		if !decision.Allowed {
			return "", fmt.Errorf("storage write protected: %s", decision.Reason)
		}
	}
	key := b.ObjectKey(taskID, msgID)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := io.WriteString(gz, payload); err != nil {
		return "", fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("gzip close: %w", err)
	}
	_, err := b.client.PutObject(ctx, b.bucket, key,
		bytes.NewReader(buf.Bytes()), int64(buf.Len()),
		minio.PutObjectOptions{ContentType: "application/gzip"})
	if err != nil {
		return "", fmt.Errorf("put object %s: %w", key, err)
	}
	return key, nil
}

// Get 拉取并 GZIP 解压。
func (b *BulkStore) Get(ctx context.Context, key string) (string, error) {
	if !b.Enabled() {
		return "", fmt.Errorf("bulk store disabled")
	}
	obj, err := b.client.GetObject(ctx, b.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("get object %s: %w", key, err)
	}
	defer obj.Close()

	gz, err := gzip.NewReader(obj)
	if err != nil {
		return "", fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	data, err := io.ReadAll(gz)
	if err != nil {
		return "", fmt.Errorf("gzip read: %w", err)
	}
	return string(data), nil
}

// DeleteByPrefix 批量删除前缀下所有对象，实现 BulkObjectDeleter 接口。
func (b *BulkStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	if !b.Enabled() {
		return nil
	}
	// 列出对象 → 批量删
	objCh := b.client.ListObjects(ctx, b.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	// 转 channel
	toDelete := make(chan minio.ObjectInfo, 32)
	go func() {
		defer close(toDelete)
		for o := range objCh {
			if o.Err != nil {
				continue
			}
			toDelete <- o
		}
	}()
	errCh := b.client.RemoveObjects(ctx, b.bucket, toDelete, minio.RemoveObjectsOptions{})
	var firstErr error
	for re := range errCh {
		if re.Err != nil && firstErr == nil {
			firstErr = fmt.Errorf("remove %s: %w", re.ObjectName, re.Err)
		}
	}
	return firstErr
}

// IsExternal 判断 message 是否为外置存储（payload_object_key 非空 + inline 为空）。
func IsExternal(m *Message) bool {
	return m != nil && m.PayloadInline == "" && strings.TrimSpace(m.PayloadObjectKey) != ""
}

// 编译期断言：BulkStore 实现 BulkObjectDeleter（供 Sweeper purge 使用）。
var _ BulkObjectDeleter = (*BulkStore)(nil)

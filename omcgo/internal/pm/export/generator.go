package export

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

// Uploader 是把 CSV 流式上传到对象存储的最小契约（便于单测 stub）。
// 真实实现由 minio.Client.PutObject 满足（size=-1 → 从 io.Reader 流式上传，不预读全量）。
type Uploader interface {
	PutObject(ctx context.Context, bucket, object string, reader io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

// GenerateResult 是一次导出生成的产物（回填导出任务表用）。
type GenerateResult struct {
	RowCount int64
	FileSize int64
}

// streamCSVToObject 把 RowSource 的全部数据点流式写成 CSV、经 io.Pipe 直传对象存储。
//
// 关键：边查边写边传——CSVWriter 写进 pipe 的 writer 端，PutObject 从 reader 端读，
// size=-1 让 SDK 流式上传，全程不把全量行 / 全量文件读进内存（设计 §5.4）。
// 取数 / 写 CSV 出错时用 CloseWithError 让 PutObject 端拿到错误并中止上传。
func streamCSVToObject(
	ctx context.Context,
	up Uploader,
	bucket, object string,
	src RowSource,
	resolver *nameResolver,
) (GenerateResult, error) {
	pr, pw := io.Pipe()

	// 写 CSV 的 goroutine：拉源 → 回填名 → 写行 → flush；任何错误 CloseWithError 传给上传端。
	var rowCount int64
	writeErrCh := make(chan error, 1)
	go func() {
		cw, err := NewCSVWriter(pw)
		if err != nil {
			pw.CloseWithError(err)
			writeErrCh <- err
			return
		}
		for {
			rows, done, err := src.Next(ctx)
			if err != nil {
				pw.CloseWithError(err)
				writeErrCh <- err
				return
			}
			if resolver != nil && len(rows) > 0 {
				resolver.resolveBatch(ctx, rows)
			}
			for i := range rows {
				if werr := cw.WriteRow(rows[i]); werr != nil {
					pw.CloseWithError(werr)
					writeErrCh <- werr
					return
				}
			}
			if done {
				break
			}
		}
		if ferr := cw.Flush(); ferr != nil {
			pw.CloseWithError(ferr)
			writeErrCh <- ferr
			return
		}
		rowCount = cw.RowCount()
		writeErrCh <- nil
		pw.Close()
	}()

	info, upErr := up.PutObject(ctx, bucket, object, pr, -1, minio.PutObjectOptions{ContentType: "text/csv; charset=utf-8"})
	writeErr := <-writeErrCh
	if writeErr != nil {
		// 取数 / 写 CSV 失败优先：上传端的 err 多半是 pipe 被 CloseWithError 的派生错误。
		return GenerateResult{}, fmt.Errorf("export generate: write csv: %w", writeErr)
	}
	if upErr != nil {
		return GenerateResult{}, fmt.Errorf("export generate: upload object: %w", upErr)
	}
	// 排空 reader 防 PutObject 提前返回后 writer 端阻塞（正常 size=-1 已读尽，防御性）。
	_, _ = io.Copy(io.Discard, pr)
	return GenerateResult{RowCount: rowCount, FileSize: info.Size}, nil
}

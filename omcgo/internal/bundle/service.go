package bundle

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// Service —— 同步流式打包。
//
// 设计极简化（推倒原异步 / MinIO / 轮询的复杂方案）：
//   - 各模块用 Register 注册 Source 闭包（"按 IDs/SNs 列出物理文件"）
//   - WriteZipTo(w) 流式拉 MinIO 对象写 zip,直接 io.Writer 出去
//   - 调用方（HTTP handler）把 http.ResponseWriter 传进来 → 浏览器看到一次下载
//
// 不需要表、不需要 presign、不需要轮询。大批量（GB 级）走 nginx proxy_read_timeout
// 调长即可。
type Service struct {
	minio   *minio.Client
	logger  *zap.Logger
	sources map[Module]Source
	mu      sync.RWMutex
}

func NewService(minioClient *minio.Client, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		minio:   minioClient,
		logger:  logger.Named("bundle"),
		sources: make(map[Module]Source, 4),
	}
}

// Register 给某模块装配 Source 闭包。
func (s *Service) Register(module Module, source Source) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sources[module] = source
}

// WriteZipTo 解析 targetIDs → 流式打 zip 到 w。
//
// 单文件 GetObject 失败时往 zip 里写 `xxx.error.txt` 占位（不要因为一个对象丢就整批 5xx,
// 用户可以打开 zip 看哪个具体丢了),其它继续。整体 error 只在 source 解析失败 / w 写
// 中断时返回。
//
// 返回 (写入的文件数, error)。0 文件 + 无 err 时 caller 应当 4xx（让用户知道没东西可下）。
func (s *Service) WriteZipTo(ctx context.Context, module Module, targetIDs []string, w io.Writer) (int, error) {
	s.mu.RLock()
	source, ok := s.sources[module]
	s.mu.RUnlock()
	if !ok {
		return 0, fmt.Errorf("no source registered for module %s: %w", module, commonerrors.ErrInvalidInput)
	}
	if len(targetIDs) == 0 {
		return 0, fmt.Errorf("target_ids empty: %w", commonerrors.ErrInvalidInput)
	}
	files, err := source(ctx, targetIDs)
	if err != nil {
		return 0, fmt.Errorf("resolve files: %w", err)
	}
	if len(files) == 0 {
		return 0, nil
	}
	flusher, _ := w.(interface{ Flush() })
	zw := zip.NewWriter(w)
	defer zw.Close()

	writtenCount := 0
	for _, f := range files {
		if cErr := s.appendOne(ctx, zw, f, flusher); cErr != nil {
			s.logger.Warn("append file to zip failed; writing .error.txt placeholder",
				zap.String("bucket", f.Bucket),
				zap.String("path", f.ObjectPath),
				zap.Error(cErr))
			placeholder, perr := zw.Create(f.EntryName + ".error.txt")
			if perr == nil {
				_, _ = placeholder.Write([]byte(cErr.Error()))
			}
			continue
		}
		writtenCount++
		// 一个 entry 结束后也 flush 一次,防止 entry trailer 攒在 zip.Writer 内部。
		if fErr := zw.Flush(); fErr != nil {
			s.logger.Warn("zip writer flush failed", zap.Error(fErr))
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
	return writtenCount, nil
}

// chunkSize 是单文件流式拷贝的批大小。每 chunk 写完会 flush 一次底层
// ResponseWriter,让浏览器 onDownloadProgress 看到连续增长。
// 256KB 平衡进度粒度(下载 1MB/s 时 4 次/s 触发,够平滑)和系统调用开销。
const chunkSize = 256 * 1024

func (s *Service) appendOne(ctx context.Context, zw *zip.Writer, f BundleFile, flusher interface{ Flush() }) error {
	obj, err := s.minio.GetObject(ctx, f.Bucket, f.ObjectPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("get object: %w", err)
	}
	defer obj.Close()
	// Method: Store(不压缩,直传字节)而不是默认 Deflate 的两个理由：
	//  1. 流式可观测性: deflate writer 内部要攒 ~32KB 才 emit 一个 block,大文件
	//     里 onDownloadProgress 看到的是大段大段跳;Store 模式 io.Copy 直接落到 socket。
	//  2. 性能 + 文件大小: 固件 IMG / NV / license 多数已是压缩过的二进制,deflate
	//     再压缩压缩率接近 0 反而费 CPU(单核 ~100MB/s 上限)。
	w, err := zw.CreateHeader(&zip.FileHeader{Name: f.EntryName, Method: zip.Store})
	if err != nil {
		return fmt.Errorf("zip create entry: %w", err)
	}
	// 手写循环 + 每 chunkSize Flush:io.Copy 不主动调底层 Flush,大文件中间几分钟
	// 不刷会让浏览器进度条卡死。手循环每 256KB 推一次,进度连续可见。
	buf := make([]byte, chunkSize)
	var written int64
	lastLog := time.Now()
	for {
		n, rerr := obj.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return fmt.Errorf("zip write: %w", werr)
			}
			written += int64(n)
			if flusher != nil {
				// zw.Flush 把 zip writer 内部 pending 字节刷到底层 ResponseWriter
				if fErr := zw.Flush(); fErr != nil {
					return fmt.Errorf("zip flush: %w", fErr)
				}
				flusher.Flush()
			}
			// 每秒打一条日志看服务端真的在持续 flush 没。如果日志数字也是大段跳,
			// 问题在 MinIO 读 / Go zip 内部;如果日志平滑但浏览器看到大段跳,
			// 问题在 nginx / 中间代理 / 浏览器 axios。
			if time.Since(lastLog) > time.Second {
				s.logger.Info("bundle stream flush",
					zap.String("entry", f.EntryName),
					zap.Int64("written_bytes", written),
				)
				lastLog = time.Now()
			}
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			if errors.Is(rerr, io.ErrUnexpectedEOF) {
				return rerr
			}
			return fmt.Errorf("minio read: %w", rerr)
		}
	}
}

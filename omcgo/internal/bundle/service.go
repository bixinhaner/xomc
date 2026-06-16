package bundle

import (
	"archive/zip"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/compress"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// MaxBatchTargets 是单次批量下载允许的目标数上限（ids / serial_numbers 合计）。
// 超过 → 400。防止超大请求拖垮 MinIO 拉取与 zip 流（#63 加固，亦缓解 DoS）。
const MaxBatchTargets = 1000

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
	minio    *minio.Client
	logger   *zap.Logger
	sources  map[Module]Source
	snReader SNVisibilityReader
	mu       sync.RWMutex
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

// SetSNVisibilityReader 注入设备序列号可见性判定器（#63 设备组可见性强制层）。
// 不注入则 WriteZipTo 退化为不过滤（dev/test），与 device/alarm nil-safe 语义一致。
func (s *Service) SetSNVisibilityReader(reader SNVisibilityReader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snReader = reader
}

// snKeyedModules 是 targetIDs 直接为 serial_number 的模块集合；这些可在调用 Source
// 之前就把入参 SN 过滤到可见集合。其余模块（firmware 全局制品；mr_files/pm_files 是
// 文件 ID）入口拿不到 SN，由 WriteZipTo 在 Source 返回后按 BundleFile.DeviceSN 后置剔除。
var snKeyedModules = map[Module]struct{}{
	ModuleConfigSnapshot: {},
	ModuleDeviceLicense:  {},
	ModuleMR:             {},
	ModulePM:             {},
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
//
// visibleGroups 是 #63 设备组可见性强制层从调用者身份解析的可见分组集合（三态契约见
// authz 包）：nil=超管不过滤，[]=无权限空集，[ids]=限定。SN-keyed 模块在调 Source 前
// 把入参 SN 收窄到可见集合；文件 ID 模块（mr_files/pm_files）由 Source 返回后按
// BundleFile.DeviceSN 后置剔除域外设备文件。firmware 镜像无 DeviceSN，是全局制品，不过滤。
func (s *Service) WriteZipTo(ctx context.Context, module Module, targetIDs []string, visibleGroups []uuid.UUID, w io.Writer) (int, error) {
	s.mu.RLock()
	source, ok := s.sources[module]
	snReader := s.snReader
	s.mu.RUnlock()
	if !ok {
		return 0, fmt.Errorf("no source registered for module %s: %w", module, commonerrors.ErrInvalidInput)
	}
	if len(targetIDs) == 0 {
		return 0, fmt.Errorf("target_ids empty: %w", commonerrors.ErrInvalidInput)
	}

	// SN-keyed 模块：调 Source 前先把 targetIDs（=SN 列表）收窄到可见集合。
	if _, snKeyed := snKeyedModules[module]; snKeyed {
		kept, fErr := filterVisibleSNs(ctx, snReader, visibleGroups, targetIDs)
		if fErr != nil {
			return 0, fmt.Errorf("filter visible serial numbers: %w", fErr)
		}
		if len(kept) == 0 {
			// 全部目标都不在可见组 → 无可下内容（caller 据 0 文件回 4xx）。
			return 0, nil
		}
		targetIDs = kept
	}

	files, err := source(ctx, targetIDs)
	if err != nil {
		return 0, fmt.Errorf("resolve files: %w", err)
	}

	// 文件 ID 模块：Source 已把每个文件的归属 SN 填到 BundleFile.DeviceSN，
	// 这里按可见性后置剔除（DeviceSN 为空的条目——如 firmware——不参与过滤直接保留）。
	if files, err = s.filterFilesByVisibleSN(ctx, snReader, visibleGroups, files); err != nil {
		return 0, fmt.Errorf("filter files by visible serial numbers: %w", err)
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

// filterFilesByVisibleSN 按 BundleFile.DeviceSN 把 Source 返回的文件剔除到可见集合。
// DeviceSN 为空的条目（如 firmware 全局制品）不参与过滤，直接保留。
// reader == nil（dev/test）或 visibleGroups == nil（超管）→ 原样返回。
func (s *Service) filterFilesByVisibleSN(ctx context.Context, reader SNVisibilityReader, visibleGroups []uuid.UUID, files []BundleFile) ([]BundleFile, error) {
	if reader == nil || visibleGroups == nil {
		return files, nil
	}
	// 收集需要判定的 SN（去重）。
	snSet := make(map[string]struct{})
	for i := range files {
		if files[i].DeviceSN != "" {
			snSet[files[i].DeviceSN] = struct{}{}
		}
	}
	if len(snSet) == 0 {
		return files, nil // 全是无归属设备的全局制品。
	}
	sns := make([]string, 0, len(snSet))
	for sn := range snSet {
		sns = append(sns, sn)
	}
	visible, err := reader.VisibleSerialNumbers(ctx, visibleGroups, sns)
	if err != nil {
		return nil, err
	}
	kept := make([]BundleFile, 0, len(files))
	for _, f := range files {
		if f.DeviceSN == "" {
			kept = append(kept, f) // 全局制品保留。
			continue
		}
		if _, ok := visible[f.DeviceSN]; ok {
			kept = append(kept, f)
		}
	}
	return kept, nil
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

	// issue #321：PM/MR 原始文件入库后被 gzip 压缩回写 MinIO（对象键 / file_name 仍是 .xml）。
	// 嗅探 gzip 魔数：压缩内容的 zip 条目名补 .gz，让用户解开外层 zip 后得到可正常解压的
	// .xml.gz；明文及非 gzip 模块（固件 / 配置 / license）原样不变。bufio 包裹后 Peek 不消耗
	// 数据，后续读取仍从头开始。
	br := bufio.NewReader(obj)
	entryName := f.EntryName
	if head, _ := br.Peek(2); compress.IsGzip(head) && !strings.HasSuffix(strings.ToLower(entryName), ".gz") {
		entryName += ".gz"
	}
	// Method: Store(不压缩,直传字节)而不是默认 Deflate 的两个理由：
	//  1. 流式可观测性: deflate writer 内部要攒 ~32KB 才 emit 一个 block,大文件
	//     里 onDownloadProgress 看到的是大段大段跳;Store 模式 io.Copy 直接落到 socket。
	//  2. 性能 + 文件大小: 固件 IMG / NV / license 多数已是压缩过的二进制,deflate
	//     再压缩压缩率接近 0 反而费 CPU(单核 ~100MB/s 上限)。
	w, err := zw.CreateHeader(&zip.FileHeader{Name: entryName, Method: zip.Store})
	if err != nil {
		return fmt.Errorf("zip create entry: %w", err)
	}
	// 手写循环 + 每 chunkSize Flush:io.Copy 不主动调底层 Flush,大文件中间几分钟
	// 不刷会让浏览器进度条卡死。手循环每 256KB 推一次,进度连续可见。
	buf := make([]byte, chunkSize)
	var written int64
	lastLog := time.Now()
	for {
		n, rerr := br.Read(buf)
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

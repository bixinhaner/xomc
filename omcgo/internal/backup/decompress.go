// Package backup — backup-side decompression (issue #61 / Epic #27).
//
// 与 compression.go 的 Compressor 对称：把存盘的压缩备份字节还原为明文。
// PromoteFromBackup 用它把备份对象解出明文配置写进 config_snapshots
// （快照桶统一存明文，见 snapshot_service.go 的 #61 注释）。
//
// 注意：ACS 下载侧已有同形实现（internal/acs/download/decompress.go），但
// 那个包 import 了 backup，本包反向 import 会形成依赖环，故此处独立维护一份。
// 两处算法集必须保持一致——新增第五种压缩算法时同步改两边。
package backup

import (
	"bytes"
	stdbzip2 "compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// stripCompressionSuffix 检测 name 末尾的压缩扩展名，命中则返回
// (format, 去掉后缀的 name, true)。ok=false 表示未压缩、原样返回。
// 后缀集合与 acs/upload 侧 Compressor.Extension() 写入的一致
// (.gz/.zst/.lz4/.bz2)。
func stripCompressionSuffix(name string) (format, stripped string, ok bool) {
	switch {
	case strings.HasSuffix(name, ".gz"):
		return "gzip", strings.TrimSuffix(name, ".gz"), true
	case strings.HasSuffix(name, ".zst"):
		return "zstd", strings.TrimSuffix(name, ".zst"), true
	case strings.HasSuffix(name, ".lz4"):
		return "lz4", strings.TrimSuffix(name, ".lz4"), true
	case strings.HasSuffix(name, ".bz2"):
		return "bzip2", strings.TrimSuffix(name, ".bz2"), true
	default:
		return "", name, false
	}
}

// decompress 把 format 格式的压缩字节还原为明文，最多读取 maxOut+1 字节——
// 超过 maxOut 即判定为异常/解压炸弹并报错，而不是无界缓冲。
func decompress(format string, data []byte, maxOut int64) ([]byte, error) {
	src := bytes.NewReader(data)
	var rc io.ReadCloser
	switch format {
	case "gzip":
		r, err := gzip.NewReader(src)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		rc = r
	case "zstd":
		r, err := zstd.NewReader(src)
		if err != nil {
			return nil, fmt.Errorf("zstd reader: %w", err)
		}
		rc = zstdReadCloser{r}
	case "lz4":
		rc = io.NopCloser(lz4.NewReader(src))
	case "bzip2":
		// stdlib compress/bzip2 只有 Reader（无 Writer），正合此处所需。
		rc = io.NopCloser(stdbzip2.NewReader(src))
	default:
		return nil, fmt.Errorf("unknown compression format %q: %w", format, ErrCompressionFormatInvalid)
	}
	defer rc.Close()

	limited := io.LimitReader(rc, maxOut+1)
	out, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("%s decompress: %w", format, err)
	}
	if int64(len(out)) > maxOut {
		return nil, fmt.Errorf("decompressed output exceeds %d bytes: %w", maxOut, ErrCompressionFormatInvalid)
	}
	return out, nil
}

// zstdReadCloser 把 klauspost zstd.Decoder.Close()（无返回值）适配成
// io.ReadCloser（要求 Close() error）。
type zstdReadCloser struct{ *zstd.Decoder }

func (z zstdReadCloser) Close() error { z.Decoder.Close(); return nil }

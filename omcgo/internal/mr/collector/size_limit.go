package collector

import "fmt"

// maxMRFileBytes 是单个 MR 文件解析的体积上限（#168 安全护栏）。
//
// MR（MRO/MRS/MRE）文件比 PM 偏大但仍有界；64 MiB 是宽松天花板，挡住超大/异常/损坏文件被单次
// 全量读进内存导致 worker OOM。配合 io.LimitReader 双重兜底（Stat 给确切大小先拒；LimitReader
// 在 Stat 不可用时截断使解析报错）。
const maxMRFileBytes int64 = 64 << 20

// ensureMRFileSize 校验文件体积未超上限；超限返回错误（交由 retry/DLQ 处理，不致 OOM）。
func ensureMRFileSize(size int64) error {
	if size > maxMRFileBytes {
		return fmt.Errorf("MR file too large: %d bytes exceeds limit %d bytes", size, maxMRFileBytes)
	}
	return nil
}

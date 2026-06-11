package collector

import "fmt"

// maxPMFileBytes 是单个 PM 文件解析的体积上限（#168 安全护栏）。
//
// PM XML 正常 KB~数 MB；64 MiB 是宽松天花板，用于挡住超大/异常/损坏文件被单次全量读进内存
// 导致 worker OOM。worker 句柄同步逐个处理文件，真正的内存背压就是「单文件体积」——本上限
// 配合 io.LimitReader 双重兜底（Stat 给确切大小先拒；LimitReader 在 Stat 不可用时截断）。
const maxPMFileBytes int64 = 64 << 20

// ensurePMFileSize 校验文件体积未超上限；超限返回错误（交由 retry/DLQ 包装器处理，不致 OOM）。
func ensurePMFileSize(size int64) error {
	if size > maxPMFileBytes {
		return fmt.Errorf("PM file too large: %d bytes exceeds limit %d bytes", size, maxPMFileBytes)
	}
	return nil
}

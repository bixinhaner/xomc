package task

import (
	"testing"
	"time"
)

// TestTaskTTLsBounded 守卫 TTL 不回归到 24h（#168：Redis 工作集增长真因之一）。
// 命令实际生命周期远短于 4h；上界放 6h 留余量但挡住 24h 回归。
func TestTaskTTLsBounded(t *testing.T) {
	const maxAllowed = 6 * time.Hour
	if taskDetailTTL <= 0 || taskDetailTTL > maxAllowed {
		t.Errorf("taskDetailTTL=%v 越界，应在 (0, %v]", taskDetailTTL, maxAllowed)
	}
	if cwmpMappingTTL <= 0 || cwmpMappingTTL > maxAllowed {
		t.Errorf("cwmpMappingTTL=%v 越界，应在 (0, %v]", cwmpMappingTTL, maxAllowed)
	}
}

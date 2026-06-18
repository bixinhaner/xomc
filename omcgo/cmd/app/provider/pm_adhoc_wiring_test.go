package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestAppAdhocRepo_HasWatermarkReaderWired 钉死 #528 P3 的装配缺口（检查方回合1阻塞项）。
//
// 建持续任务的唯一入口 = app 进程的 POST /pm/adhoc，其 repo 必须注入「上游完成水位」读取器，
// 否则新建持续任务初始游标退化为 NULL（落回 created_at），sweep 会逐 tick 补出 created_at..水位
// 之间的史前空格批量行——正是 scope 要消除的现象。检查方曾发现水位读取器被误注入到 worker
// 进程（从不建任务），app 端 repo 漏注入。
//
// 本测试直接调用生产构造函数 buildPMAdhocRepo（initPMModule 用的同一函数），pool 传 nil 不发起
// 任何 DB 调用，仅验证注入链路。若有人把 SetWatermarkReader 从 buildPMAdhocRepo 删掉/挪走，
// 此测试立即红——真正钉死生产装配路径，而非镜像复制。
func TestAppAdhocRepo_HasWatermarkReaderWired(t *testing.T) {
	repo := buildPMAdhocRepo(nil, nil, zap.NewNop())

	assert.True(t, repo.HasWatermarkReader(),
		"app 端建持续任务的 adhoc repo 必须注入水位读取器，否则初始游标退化、结果表冒史前空格（#528 P3 回归）")
}

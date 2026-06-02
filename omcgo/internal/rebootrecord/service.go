package rebootrecord

import (
	"context"

	"go.uber.org/zap"
)

// Service 统一重启记录服务：只读合并查询（普通重启 event_logs + 异常重启 station_fault_logs）。
//
// 与 eventlog / stationlog 的关系：本服务不写入，只读时 UNION 两表合成"重启记录"视图，
// 供「启动记录」页面单列表展示。写入仍分别由 device.RecordBootFromInform 分流到两表。
type Service struct {
	repo   Repository
	logger *zap.Logger
}

func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger.Named("rebootrecord")}
}

// List 统一重启记录列表（分页 + 过滤 + 按重启类型筛选）。
func (s *Service) List(ctx context.Context, f Filter) ([]*RebootRecord, int64, error) {
	return s.repo.List(ctx, f)
}

// StatByDevice 按设备聚合重启次数（总次数 + 异常次数，跟随过滤、不分页）。
func (s *Service) StatByDevice(ctx context.Context, f Filter) ([]*DeviceRebootStat, error) {
	return s.repo.StatByDevice(ctx, f)
}

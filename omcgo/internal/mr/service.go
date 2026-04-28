package mr

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/mr/parser"
	"go.uber.org/zap"
)

// ErrIndicatorNotFound 表示指标未在指标库中找到。
var ErrIndicatorNotFound = errors.New("mr indicator not found")

// IndicatorStats 描述某指标在 MR 数据中的简要统计（基于值域）。
type IndicatorStats struct {
	IndicatorCode string  `json:"indicator_code"`
	Avg           float64 `json:"avg"`
	Min           float64 `json:"min"`
	Max           float64 `json:"max"`
	P50           float64 `json:"p50"`
	P95           float64 `json:"p95"`
	SampleCount   int64   `json:"sample_count"`
}

// MRService 是 MR 模块的 thin service facade，
// 把底层 store / indicator repo / mapping repo 的常用组合操作（如
// "落库一份解析完成的 MR 文件"、"按设备分页查 MR 记录"、"读指标统计"）
// 收拢到单一调用入口，方便 handler / collector pipeline 使用，也方便测试。
//
// 设计原则（参考 internal/alarm/library_service.go）：
//   - 接口小（≤ 10 个方法），职责单一
//   - 不持有 HTTP / SOAP 上下文，只依赖 ctx + 领域参数
//   - 失败统一 fmt.Errorf 包装上下文，不裸 panic
type MRService interface {
	// IngestParsedFile 是 collector / parser 完成解析后的统一落库入口：
	// 1) 保存 MRFileInfo 元数据；2) 批量插入记录；3) 把 file 标记为 parsed。
	// 任一步失败都会返回带上下文的 error，调用方可基于返回值决定重试策略。
	IngestParsedFile(ctx context.Context, file *MRFileInfo, records []parser.MRRecord) error

	// ListFilesByDevice 列出指定设备的 MR 文件（device_id = nil 则不过滤设备）。
	ListFilesByDevice(ctx context.Context, deviceID *uuid.UUID, listReq model.ListRequest) (*model.ListResponse[MRFileInfo], error)

	// QueryRecordsByDevice 查询指定设备的 MR 记录（device_id = nil 则不过滤设备）。
	QueryRecordsByDevice(ctx context.Context, deviceID *uuid.UUID, listReq model.ListRequest) (*model.ListResponse[MRRecordEntry], error)

	// GetIndicatorStats 读取指标定义并基于其值域生成简要统计。
	// 指标不存在时返回 ErrIndicatorNotFound。
	GetIndicatorStats(ctx context.Context, code string) (*IndicatorStats, error)
}

// service 是 MRService 的默认实现。
type service struct {
	store   MRStore
	indRepo IndicatorRepository
	logger  *zap.Logger
}

// NewService 创建 MRService 实例。logger 允许传 nil（内部使用 NopLogger）。
func NewService(store MRStore, indRepo IndicatorRepository, logger *zap.Logger) MRService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{store: store, indRepo: indRepo, logger: logger}
}

// IngestParsedFile 实现 collector pipeline 的"落库 + 标记完成"组合操作。
func (s *service) IngestParsedFile(ctx context.Context, file *MRFileInfo, records []parser.MRRecord) error {
	if file == nil {
		return fmt.Errorf("ingest mr file: file is nil")
	}
	if file.ID == uuid.Nil {
		return fmt.Errorf("ingest mr file: file ID is empty")
	}

	if err := s.store.SaveFile(ctx, file); err != nil {
		return fmt.Errorf("save mr file metadata: %w", err)
	}

	if len(records) > 0 {
		if err := s.store.BatchInsertRecords(ctx, file.ID, file.DeviceID, file.MRType, records); err != nil {
			return fmt.Errorf("batch insert mr records: %w", err)
		}
	}

	if err := s.store.UpdateFileParsed(ctx, file.ID, len(records)); err != nil {
		return fmt.Errorf("update mr file parsed flag: %w", err)
	}

	s.logger.Debug("mr file ingested",
		zap.String("file_id", file.ID.String()),
		zap.String("device_id", file.DeviceID.String()),
		zap.Int("record_count", len(records)),
	)
	return nil
}

// ListFilesByDevice 是 store.ListFiles 的薄包装。
func (s *service) ListFilesByDevice(ctx context.Context, deviceID *uuid.UUID, listReq model.ListRequest) (*model.ListResponse[MRFileInfo], error) {
	filter := MRFileFilter{ListRequest: listReq, DeviceID: deviceID}
	resp, err := s.store.ListFiles(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list mr files by device: %w", err)
	}
	return resp, nil
}

// QueryRecordsByDevice 是 store.QueryRecords 的薄包装。
func (s *service) QueryRecordsByDevice(ctx context.Context, deviceID *uuid.UUID, listReq model.ListRequest) (*model.ListResponse[MRRecordEntry], error) {
	filter := MRRecordFilter{ListRequest: listReq, DeviceID: deviceID}
	resp, err := s.store.QueryRecords(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("query mr records by device: %w", err)
	}
	return resp, nil
}

// GetIndicatorStats 读取指标定义并基于值域生成简要统计。
// 当前 sample_count 固定 0，待后续接入真实时序聚合查询时填充。
func (s *service) GetIndicatorStats(ctx context.Context, code string) (*IndicatorStats, error) {
	if code == "" {
		return nil, fmt.Errorf("get mr indicator stats: code is empty")
	}
	indicator, err := s.indRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("get mr indicator by code: %w", err)
	}
	if indicator == nil {
		return nil, ErrIndicatorNotFound
	}

	var minVal, maxVal float64
	if indicator.ValueRangeMin != nil {
		minVal = *indicator.ValueRangeMin
	}
	if indicator.ValueRangeMax != nil {
		maxVal = *indicator.ValueRangeMax
	}
	avg := (minVal + maxVal) / 2

	return &IndicatorStats{
		IndicatorCode: code,
		Avg:           avg,
		Min:           minVal,
		Max:           maxVal,
		P50:           avg,
		P95:           maxVal * 0.9,
		SampleCount:   0,
	}, nil
}

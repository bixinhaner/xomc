package rawarchive

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// 补偿扫描默认参数。grace 必须 > 内联压缩典型时延，避免 Sweeper 与内联快路径在每个
// 文件上重复竞争（内联通常秒级压完并置 raw_compressed=true，grace 后才进 Sweeper 视野）。
const (
	DefaultSweepInterval = 5 * time.Minute // 扫描周期
	DefaultSweepGrace    = 5 * time.Minute // 文件入库到进入扫描视野的宽限（给内联压缩先完成）
	DefaultSweepBatch    = 500             // 单次 ListUncompressed 取多少条
	DefaultSweepConc     = 4               // 单批压缩的有界并发
	sweepMaxRounds       = 20              // 单源单 tick 最多 maxRounds 轮，防一轮卡死饿其他源/无限自旋
)

// RawFileRegistry 抽象一张文件元数据表（pm_files / mr_files）中"原始对象压缩状态"的读写。
// *pm.PgPMFileStore 与 *mr.PgMRStore 均实现之（按 minio_path 键）。
type RawFileRegistry interface {
	// ListUncompressed 返回至多 limit 个 raw_compressed=false 且 created_at < olderThan
	// （grace 截止）的 MinIO 对象键，按 created_at 升序（先压最旧的）。
	ListUncompressed(ctx context.Context, olderThan time.Time, limit int) ([]string, error)
	// MarkCompressed 把给定 MinIO 对象键对应的行标记为已压缩（raw_compressed=true）。
	MarkCompressed(ctx context.Context, objects []string) error
}

// SweepSource 把一个注册表绑定到其对象所在的 MinIO bucket。
type SweepSource struct {
	Name     string // 标签（pm / mr），仅用于日志与 metric
	Bucket   string // 对象所在 MinIO bucket
	Registry RawFileRegistry
}

// SweepMetrics 暴露补偿扫描的可观测指标。
type SweepMetrics struct {
	Scanned    *prometheus.CounterVec // source：本轮取出的待压对象数
	Compressed *prometheus.CounterVec // source：补压并标记成功数
	Errors     *prometheus.CounterVec // source：列表/标记错误次数
}

// NewSweepMetrics 注册并返回补偿扫描指标。reg 为 nil 时不注册（仅供测试）。
func NewSweepMetrics(reg prometheus.Registerer) *SweepMetrics {
	m := &SweepMetrics{
		Scanned: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_raw_archive_sweep_scanned_total",
			Help: "补偿扫描取出的待压缩原始对象数（issue #321 加固），label source 区分 pm/mr",
		}, []string{"source"}),
		Compressed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_raw_archive_sweep_compressed_total",
			Help: "补偿扫描补压并标记成功的原始对象数，label source 区分 pm/mr",
		}, []string{"source"}),
		Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_raw_archive_sweep_errors_total",
			Help: "补偿扫描的列表/标记错误次数，label source 区分 pm/mr",
		}, []string{"source"}),
	}
	if reg != nil {
		reg.MustRegister(m.Scanned, m.Compressed, m.Errors)
	}
	return m
}

// Sweeper 周期性地把内联压缩遗漏的原始对象（dropped_busy / 失败 / 崩溃前未压）补压，
// 直至其 raw_compressed 置真。它是"保证压到"的兜底（内联 Schedule 是快路径）。
//
// 调度：每 interval 跑一轮 SweepOnce；每轮先查总开关（禁用则整轮跳过，不空转），再对每个
// 源分批 ListUncompressed → 有界并发 CompressNow → 把终态对象批量 MarkCompressed，直到该
// 源排空或本轮无进展或达 sweepMaxRounds。零值不可用，须经 NewSweeper 构造。
type Sweeper struct {
	archiver *Archiver
	sources  []SweepSource
	interval time.Duration
	grace    time.Duration
	batch    int
	conc     int
	logger   *zap.Logger
	metrics  *SweepMetrics
	now      func() time.Time // 可注入时钟，便于测试 grace 截止
}

// NewSweeper 构造 Sweeper。非法/零参数回落默认值；logger 为 nil 用 no-op。
func NewSweeper(a *Archiver, sources []SweepSource, interval, grace time.Duration, batch, conc int, metrics *SweepMetrics, logger *zap.Logger) *Sweeper {
	if interval <= 0 {
		interval = DefaultSweepInterval
	}
	if grace < 0 {
		grace = DefaultSweepGrace
	}
	if batch <= 0 {
		batch = DefaultSweepBatch
	}
	if conc <= 0 {
		conc = DefaultSweepConc
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Sweeper{
		archiver: a, sources: sources, interval: interval, grace: grace,
		batch: batch, conc: conc, metrics: metrics, logger: logger, now: time.Now,
	}
}

// Run 阻塞运行扫描循环直至 ctx 取消（在 worker 里以 goroutine 启动，进程优雅关停取消 ctx）。
func (s *Sweeper) Run(ctx context.Context) {
	if s == nil || s.archiver == nil || len(s.sources) == 0 {
		return
	}
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.logger.Info("raw-archive sweeper started",
		zap.Duration("interval", s.interval), zap.Duration("grace", s.grace), zap.Int("batch", s.batch))
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SweepOnce(ctx)
		}
	}
}

// SweepOnce 跑一轮全源补偿扫描（导出供测试与手动触发）。总开关关闭时整轮跳过。
func (s *Sweeper) SweepOnce(ctx context.Context) {
	if s == nil || s.archiver == nil {
		return
	}
	if !s.archiver.Enabled(ctx) {
		return
	}
	cutoff := s.now().Add(-s.grace)
	for _, src := range s.sources {
		if ctx.Err() != nil {
			return
		}
		s.sweepSource(ctx, src, cutoff)
	}
}

func (s *Sweeper) sweepSource(ctx context.Context, src SweepSource, cutoff time.Time) {
	for round := 0; round < sweepMaxRounds; round++ {
		if ctx.Err() != nil {
			return
		}
		objs, err := src.Registry.ListUncompressed(ctx, cutoff, s.batch)
		if err != nil {
			s.logger.Warn("sweep list uncompressed failed", zap.String("source", src.Name), zap.Error(err))
			s.recordErr(src.Name)
			return
		}
		if len(objs) == 0 {
			return
		}
		s.recordScanned(src.Name, len(objs))

		done := s.compressBatch(ctx, src.Bucket, objs)
		if len(done) > 0 {
			if err := src.Registry.MarkCompressed(ctx, done); err != nil {
				s.logger.Warn("sweep mark compressed failed",
					zap.String("source", src.Name), zap.Int("count", len(done)), zap.Error(err))
				s.recordErr(src.Name)
				return
			}
			s.recordCompressed(src.Name, len(done))
		}
		// 排空（取回不足一批）或本轮零进展（整批非终态/错误）→ 收手，留待下一 tick 重试，避免自旋。
		if len(objs) < s.batch || len(done) == 0 {
			return
		}
	}
}

// compressBatch 有界并发地压缩一批对象，返回其中"终态"的对象键（可标记已压缩）。
func (s *Sweeper) compressBatch(ctx context.Context, bucket string, objs []string) []string {
	sem := make(chan struct{}, s.conc)
	var mu sync.Mutex
	var wg sync.WaitGroup
	done := make([]string, 0, len(objs))
	for _, obj := range objs {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(o string) {
			defer wg.Done()
			defer func() { <-sem }()
			if s.archiver.CompressNow(ctx, bucket, o) {
				mu.Lock()
				done = append(done, o)
				mu.Unlock()
			}
		}(obj)
	}
	wg.Wait()
	return done
}

func (s *Sweeper) recordScanned(source string, n int) {
	if s.metrics != nil {
		s.metrics.Scanned.WithLabelValues(source).Add(float64(n))
	}
}

func (s *Sweeper) recordCompressed(source string, n int) {
	if s.metrics != nil {
		s.metrics.Compressed.WithLabelValues(source).Add(float64(n))
	}
}

func (s *Sweeper) recordErr(source string) {
	if s.metrics != nil {
		s.metrics.Errors.WithLabelValues(source).Inc()
	}
}

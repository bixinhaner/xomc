package logger

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 运行期日志轮转可配（sys_configs category=log.rotation）。
//
// 背景：日志文件的「大小/个数/过期」此前仅能在各服务 YAML 的 log.rotation 块里配置，运营商
// 无法在系统配置页里调。本文件提供一个进程级原子 override：compactor goroutine 每分钟读取它来
// 决定「保留几个不压缩(keep_files) / 归档过期天数(max_age_days) / 单文件最大(max_size_mb)」，
// timed rotation goroutine 读取它来决定「定时轮转间隔(rotate_interval_minutes)」，故改完无需
// 重启即在下一轮检查（≤1 分钟）生效。
//
// 安全性：override 为 nil 或某字段 ≤0 时，对应维度回退到启动期 YAML 值——即「未配置」时行为与
// 改动前完全一致，对默认路径零影响。compactor 和 timed rotation goroutine 只读原子指针，无数据
// 竞争；max_size 的执行通过 compactor stat 活动文件后调用 lumberjack 公开的 Rotate()（线程安全），
// 不直接改 lumberjack.MaxSize 字段，避免与其内部写锁竞争。lumberjack 自身 YAML MaxSize 仍作为
// 每次写入的硬兜底。

// RotationCategory 是 sys_configs 中日志轮转配置的 category。
const RotationCategory = "log.rotation"

const (
	// KeyMaxSizeMB 单个日志文件触发切割的大小（MB）。
	KeyMaxSizeMB = "max_size_mb"
	// KeyMaxAgeDays 归档保留天数（早于此的归档删除）。
	KeyMaxAgeDays = "max_age_days"
	// KeyKeepFiles 保持不压缩的最新归档个数（≈可直接 tail 的近期文件数）。
	KeyKeepFiles = "keep_files"
	// KeyRotateIntervalMinutes 定时轮转间隔（分钟）。
	KeyRotateIntervalMinutes = "rotate_interval_minutes"
)

// RotationOverride 是运行期覆盖的轮转参数。各字段 ≤0 表示「该维度不覆盖，用启动期值」。
type RotationOverride struct {
	MaxSizeMB             int
	MaxAgeDays            int
	KeepUncompressed      int
	RotateIntervalMinutes int
}

var rotationOverride atomic.Pointer[RotationOverride]

// SetRotationOverride 设置进程级轮转覆盖（后台轮转检查下一个 tick 生效）。
func SetRotationOverride(o RotationOverride) {
	rotationOverride.Store(&o)
}

func loadRotationOverride() *RotationOverride { return rotationOverride.Load() }

// effectiveRotation 把启动期值与运行期 override 合并，返回当前生效的 keep/maxAge/maxSizeMB。
// maxSizeMB 返回 0 表示「不额外执行 size 切割」（沿用 lumberjack 自身 YAML MaxSize）。
func effectiveRotation(startupKeep int, startupMaxAge time.Duration) (keep int, maxAge time.Duration, maxSizeMB int) {
	keep, maxAge, maxSizeMB = startupKeep, startupMaxAge, 0
	if o := loadRotationOverride(); o != nil {
		if o.KeepUncompressed > 0 {
			keep = o.KeepUncompressed
		}
		if o.MaxAgeDays > 0 {
			maxAge = time.Duration(o.MaxAgeDays) * 24 * time.Hour
		}
		if o.MaxSizeMB > 0 {
			maxSizeMB = o.MaxSizeMB
		}
	}
	return keep, maxAge, maxSizeMB
}

// effectiveRotateInterval 返回当前生效的定时轮转间隔。
func effectiveRotateInterval(startupInterval time.Duration) time.Duration {
	if o := loadRotationOverride(); o != nil && o.RotateIntervalMinutes > 0 {
		return time.Duration(o.RotateIntervalMinutes) * time.Minute
	}
	return startupInterval
}

// enforceMaxSize 若活动文件 ≥ maxBytes 则用 lumberjack 的线程安全 Rotate() 切割（不改其内部字段）。
func enforceMaxSize(lj *lumberjack.Logger, path string, maxBytes int64) {
	if lj == nil || maxBytes <= 0 {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Size() >= maxBytes {
		_ = lj.Rotate()
	}
}

// ConfigLookup 读 sys_configs 单值（value, found）。各进程注入自己的 sys_configs 读适配器。
type ConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

// StartRotationConfigWatcher 启动一个轮询 goroutine，每 60s 从 sys_configs 读 log.rotation 配置
// 并更新进程级 override（后台轮转检查据此在 ≤1 分钟内生效）。lookup 为 nil 直接返回（保持启动期行为）。
// 立即先读一次再进入轮询。
func StartRotationConfigWatcher(ctx context.Context, lookup ConfigLookup, logger *zap.Logger) {
	if lookup == nil {
		return
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	apply := func() {
		SetRotationOverride(RotationOverride{
			MaxSizeMB:             readIntCfg(ctx, lookup, KeyMaxSizeMB),
			MaxAgeDays:            readIntCfg(ctx, lookup, KeyMaxAgeDays),
			KeepUncompressed:      readIntCfg(ctx, lookup, KeyKeepFiles),
			RotateIntervalMinutes: readIntCfg(ctx, lookup, KeyRotateIntervalMinutes),
		})
	}
	apply() // 启动即应用一次（早于首个 compactor tick）
	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				apply()
			}
		}
	}()
	logger.Info("log rotation config watcher started",
		zap.String("category", RotationCategory), zap.Duration("poll", 60*time.Second))
}

// readIntCfg 读一个非负整数配置；缺失/非法/≤0 返回 0（= 该维度不覆盖）。
func readIntCfg(ctx context.Context, lookup ConfigLookup, key string) int {
	v, found := lookup(ctx, RotationCategory, key)
	if !found {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

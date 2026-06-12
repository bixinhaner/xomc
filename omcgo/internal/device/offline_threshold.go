package device

import (
	"context"
	"strconv"
	"time"
)

// OfflineThresholdLookup 是离线阈值读取的唯一外部依赖：按 (category, key) 读
// sys_configs 行的 value 列。消费者驱动小接口（不引 admin 包），由 wiring 层
// 提供 admin.PgSysConfigRepository.GetByKey 的适配器。
//
// 返回 (value, true) 表示读到；(_, false) 表示 key 不存在 / 读失败 — 调用方
// 退化到 default。
type OfflineThresholdLookup func(ctx context.Context, category, key string) (value string, found bool)

// 离线阈值配置位置与键名（与前端 DeviceSettings.tsx 表单字段名一致）。
//   - category=device, key=enbTimeout → 基站类阈值（秒）
//   - category=device, key=cpeTimeout → CPE 类阈值（秒）
const (
	offlineConfigCategory = "device"
	offlineConfigKeyENB   = "enbTimeout"
	offlineConfigKeyCPE   = "cpeTimeout"
)

// 默认阈值（与 seed 迁移种子值一致：基站 100s、CPE 600s）。
const (
	defaultENBOfflineSec = 100
	defaultCPEOfflineSec = 600
)

// OfflineThresholds 是一轮扫描读到的两类离线阈值（秒）。
type OfflineThresholds struct {
	ENBSec int // 基站类阈值（秒）
	CPESec int // CPE 类阈值（秒）
}

// resolveOfflineThresholds 用注入的 lookup 读取两类阈值。
//
// lookup=nil 或读不到 / 解析失败 / 非正数 → 退化到默认值（100 / 600）。
// 每次扫描调用一次，保证 "改配置下一轮即生效"，无进程级缓存。
func resolveOfflineThresholds(ctx context.Context, lookup OfflineThresholdLookup) OfflineThresholds {
	enb := defaultENBOfflineSec
	cpe := defaultCPEOfflineSec
	if lookup != nil {
		if v, ok := lookup(ctx, offlineConfigCategory, offlineConfigKeyENB); ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				enb = n
			}
		}
		if v, ok := lookup(ctx, offlineConfigCategory, offlineConfigKeyCPE); ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				cpe = n
			}
		}
	}
	return OfflineThresholds{ENBSec: enb, CPESec: cpe}
}

// scanIntervalFor 计算扫描周期 = min(两类阈值中较小者的一半, 60s)。
//
// 例：阈值 100 → 50s；阈值 600 → 60s（300s 被 60s 封顶）。
// 周期跟随配置阈值变化，避免阈值很小时扫描太慢导致离线判定迟到。
func scanIntervalFor(th OfflineThresholds) time.Duration {
	minSec := th.ENBSec
	if th.CPESec < minSec {
		minSec = th.CPESec
	}
	if minSec <= 0 {
		minSec = defaultENBOfflineSec
	}
	half := time.Duration(minSec/2) * time.Second
	if half < time.Second {
		half = time.Second
	}
	if half > 60*time.Second {
		half = 60 * time.Second
	}
	return half
}

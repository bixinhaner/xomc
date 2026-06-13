// Package logretention 提供「审计 / 业务软件日志」DB 表的按时间保留清理。
//
// 背景：审计日志（audit_logs）、运维审计（ops_audit_logs）、登录/操作/任务日志
// （sys_login_logs / sys_oper_logs / sys_task_logs）、系统日志（system_logs）、网元报文
// 日志（ne_message_logs）、设备事件日志（event_logs）此前**无任何保留/清理**，无限增长。
// 本包按用户要求让各类日志的过期时间可配（sys_configs category=log.retention），由 worker
// 每日 cron 触发批量 DELETE 过期行。配置 TTL 缓存，热加载（改完下一个采样周期/下一次 Run 生效）。
//
// 注：基站日志（station_*_logs，#320）、TR069 报文跟踪（trace_messages，TSDB 3 天）已有各自
// 保留机制，不在本包范围内，避免重复。
package logretention

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Category 是 sys_configs 中日志保留策略的 category。
const Category = "log.retention"

// KeyEnabled 总开关键（false 时 worker 跳过整轮清理）。
const KeyEnabled = "enabled"

const (
	minDays = 1
	maxDays = 3650
	// policyTTL 配置缓存有效期；worker 每日 cron 触发，Run 时若缓存过期则重读 sys_configs。
	policyTTL = 60 * time.Second
)

// LogTable 描述一类受保留管理的日志表：表名、时间列、对应 sys_configs 键、默认保留天数。
type LogTable struct {
	// Name 是简短标识（用于日志/返回 summary）。
	Name string
	// Table 是 PostgreSQL 表名（来自固定白名单，非用户输入，可安全拼入 SQL）。
	Table string
	// TimeCol 是判定过期用的时间戳列。
	TimeCol string
	// Key 是 sys_configs 中该表保留天数的键。
	Key string
	// DefaultDays 是缺省保留天数（安全/审计类久留，高频报文类短留）。
	DefaultDays int
}

// Tables 是受保留管理的全部日志表（列名经 information_schema 现场核对，2026-06-13）。
// 表名/列名均为编译期常量白名单，绝不来自用户输入。
var Tables = []LogTable{
	{Name: "audit", Table: "audit_logs", TimeCol: "created_at", Key: "audit_days", DefaultDays: 180},
	{Name: "ops_audit", Table: "ops_audit_logs", TimeCol: "created_at", Key: "ops_audit_days", DefaultDays: 180},
	{Name: "login", Table: "sys_login_logs", TimeCol: "login_at", Key: "login_days", DefaultDays: 180},
	{Name: "oper", Table: "sys_oper_logs", TimeCol: "created_at", Key: "oper_days", DefaultDays: 180},
	{Name: "task", Table: "sys_task_logs", TimeCol: "started_at", Key: "task_days", DefaultDays: 90},
	{Name: "system", Table: "system_logs", TimeCol: "created_at", Key: "system_days", DefaultDays: 90},
	{Name: "ne_message", Table: "ne_message_logs", TimeCol: "created_at", Key: "ne_message_days", DefaultDays: 30},
	{Name: "event", Table: "event_logs", TimeCol: "occurred_at", Key: "event_days", DefaultDays: 90},
}

// ConfigLookup 读 sys_configs 单值（value, found）。
type ConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

// RetentionPolicy 提供各日志表保留天数 + 总开关，从 sys_configs 读取并 TTL 缓存。并发安全。
// 零值不可用，须经 NewRetentionPolicy 构造。
type RetentionPolicy struct {
	lookup ConfigLookup
	logger *zap.Logger

	mu       sync.Mutex
	enabled  bool
	days     map[string]int // key -> days
	loadedAt time.Time
}

// NewRetentionPolicy 构造保留策略。lookup 为 nil 时恒返回默认值（enabled=true，各表默认天数）。
func NewRetentionPolicy(lookup ConfigLookup, logger *zap.Logger) *RetentionPolicy {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RetentionPolicy{
		lookup: lookup,
		logger: logger.Named("log.retention"),
		days:   map[string]int{},
	}
}

// snapshot 返回当前生效的（enabled, key->days）。TTL 过期则锁外重读 sys_configs。
func (p *RetentionPolicy) snapshot(ctx context.Context) (bool, map[string]int) {
	p.mu.Lock()
	if !p.loadedAt.IsZero() && time.Since(p.loadedAt) < policyTTL {
		enabled, days := p.enabled, copyMap(p.days)
		p.mu.Unlock()
		return enabled, days
	}
	p.mu.Unlock()

	// 慢路径：锁外读 sys_configs（每日 cron Run 一次，成本可忽略）。
	enabled := p.readBool(ctx, KeyEnabled, true)
	days := make(map[string]int, len(Tables))
	for _, t := range Tables {
		days[t.Key] = p.readInt(ctx, t.Key, t.DefaultDays, minDays, maxDays)
	}

	p.mu.Lock()
	p.enabled, p.days, p.loadedAt = enabled, days, time.Now()
	p.mu.Unlock()
	return enabled, copyMap(days)
}

// Enabled 返回保留清理总开关是否开启。
func (p *RetentionPolicy) Enabled(ctx context.Context) bool {
	enabled, _ := p.snapshot(ctx)
	return enabled
}

// DaysFor 返回某表（按 sys_configs key）当前生效的保留天数。
func (p *RetentionPolicy) DaysFor(ctx context.Context, key string) int {
	_, days := p.snapshot(ctx)
	if d, ok := days[key]; ok {
		return d
	}
	return 0
}

func (p *RetentionPolicy) readInt(ctx context.Context, key string, def, lo, hi int) int {
	if p.lookup == nil {
		return def
	}
	v, found := p.lookup(ctx, Category, key)
	if !found {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < lo || n > hi {
		p.logger.Warn("invalid log retention value; using default",
			zap.String("key", key), zap.String("value", v), zap.Int("default", def))
		return def
	}
	return n
}

func (p *RetentionPolicy) readBool(ctx context.Context, key string, def bool) bool {
	if p.lookup == nil {
		return def
	}
	v, found := p.lookup(ctx, Category, key)
	if !found {
		return def
	}
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "true", "1":
		return true
	case "false", "0":
		return false
	default:
		return def
	}
}

func copyMap(m map[string]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

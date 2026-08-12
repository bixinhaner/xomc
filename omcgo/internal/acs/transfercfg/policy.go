package transfercfg

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

const (
	Category = "acs_transfer"
	cacheTTL = 30 * time.Second

	KeyProtocolPolicy = "protocolPolicy"

	ProtocolPolicyForceHTTP   = "force_http"
	ProtocolPolicyPreferHTTPS = "prefer_https"

	KeyUploadBaseURL      = "uploadBaseURL"
	KeyHTTPSUploadBaseURL = "httpsUploadBaseURL"
	KeyUploadPath         = "uploadPath"
	KeyUploadUsername     = "uploadUsername"
	KeyUploadPassword     = "uploadPassword"
	KeyUploadMaxFileSize  = "uploadMaxFileSize"

	KeyDownloadBaseURL      = "downloadBaseURL"
	KeyHTTPSDownloadBaseURL = "httpsDownloadBaseURL"
	KeyDownloadPath         = "downloadPath"
	KeyDownloadUsername     = "downloadUsername"
	KeyDownloadPassword     = "downloadPassword"

	// KeyMaxGlobalUpgradeConcurrency 系统级（跨任务）升级/回退设备并发上限。
	// 前端"系统设置 → ACS 传输配置"页维护；software 模块消费（全局升级闸）。
	KeyMaxGlobalUpgradeConcurrency = "maxGlobalUpgradeConcurrency"
)

type Provider interface {
	Snapshot(ctx context.Context) Snapshot
}

type SysConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

type UploadSettings struct {
	BaseURL      string
	HTTPSBaseURL string
	Path         string
	Username     string
	Password     string
	MaxFileSize  int64
}

type DownloadSettings struct {
	BaseURL      string
	HTTPSBaseURL string
	Path         string
	Username     string
	Password     string
}

type Snapshot struct {
	ProtocolPolicy string
	Upload         UploadSettings
	Download       DownloadSettings
	// MaxGlobalUpgradeConcurrency 系统级（跨任务）升级/回退设备并发上限；
	// 0 表示 sys_config 未配置，消费方应回落自己的默认值。
	MaxGlobalUpgradeConcurrency int

	expiresAt time.Time
}

type Policy struct {
	lookup   SysConfigLookup
	defaults Snapshot
	cacheMu  sync.Mutex
	cache    atomic.Pointer[Snapshot]
}

func DefaultsFromACSConfig(cfg appconfig.ACSConfig) Snapshot {
	return Snapshot{
		ProtocolPolicy: ProtocolPolicyForceHTTP,
		Upload: UploadSettings{
			BaseURL:     normalizeBaseURL(cfg.Upload.BaseURL),
			Path:        normalizePath(cfg.Upload.Path),
			Username:    cfg.Upload.Username,
			Password:    cfg.Upload.Password,
			MaxFileSize: cfg.Upload.MaxFileSize,
		},
		Download: DownloadSettings{
			BaseURL:  normalizeBaseURL(cfg.Download.BaseURL),
			Path:     normalizePath(cfg.Download.Path),
			Username: cfg.Download.Username,
			Password: cfg.Download.Password,
		},
	}
}

func NewPolicy(defaults Snapshot, lookup SysConfigLookup) *Policy {
	if strings.TrimSpace(defaults.ProtocolPolicy) == "" {
		defaults.ProtocolPolicy = ProtocolPolicyForceHTTP
	}
	defaults.Upload.BaseURL = normalizeBaseURL(defaults.Upload.BaseURL)
	defaults.Upload.HTTPSBaseURL = normalizeBaseURL(defaults.Upload.HTTPSBaseURL)
	defaults.Upload.Path = normalizePath(defaults.Upload.Path)
	defaults.Download.BaseURL = normalizeBaseURL(defaults.Download.BaseURL)
	defaults.Download.HTTPSBaseURL = normalizeBaseURL(defaults.Download.HTTPSBaseURL)
	defaults.Download.Path = normalizePath(defaults.Download.Path)
	return &Policy{lookup: lookup, defaults: defaults}
}

func (p *Policy) Snapshot(ctx context.Context) Snapshot {
	if cached := p.cache.Load(); cached != nil && time.Now().Before(cached.expiresAt) {
		return *cached
	}

	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	if cached := p.cache.Load(); cached != nil && time.Now().Before(cached.expiresAt) {
		return *cached
	}

	snap := p.defaults
	snap.expiresAt = time.Now().Add(cacheTTL)
	if p.lookup != nil {
		p.loadFromSysConfig(ctx, &snap)
	}
	p.cache.Store(&snap)
	return snap
}

func (p *Policy) InvalidateCache() {
	p.cache.Store(nil)
}

func (p *Policy) loadFromSysConfig(ctx context.Context, snap *Snapshot) {
	if value, ok := p.lookup(ctx, Category, KeyProtocolPolicy); ok {
		if trimmed, ok := optionalString(value); ok {
			snap.ProtocolPolicy = trimmed
		}
	}
	if value, ok := p.lookup(ctx, Category, KeyUploadBaseURL); ok {
		if normalized, ok := normalizedOptionalBaseURL(value); ok {
			snap.Upload.BaseURL = normalized
		}
	}
	if value, ok := p.lookup(ctx, Category, KeyHTTPSUploadBaseURL); ok {
		if normalized, ok := normalizedOptionalBaseURL(value); ok {
			snap.Upload.HTTPSBaseURL = normalized
		}
	}
	if value, ok := p.lookup(ctx, Category, KeyUploadPath); ok {
		if normalized, ok := normalizedOptionalPath(value); ok {
			snap.Upload.Path = normalized
		}
	}
	if value, ok := p.lookup(ctx, Category, KeyUploadUsername); ok {
		if trimmed, ok := optionalString(value); ok {
			snap.Upload.Username = trimmed
		}
	}
	if value, ok := p.lookup(ctx, Category, KeyUploadPassword); ok {
		if trimmed, ok := optionalString(value); ok {
			snap.Upload.Password = trimmed
		}
	}
	if value, ok := p.lookup(ctx, Category, KeyUploadMaxFileSize); ok {
		if parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil && parsed >= 0 {
			snap.Upload.MaxFileSize = parsed
		}
	}

	if value, ok := p.lookup(ctx, Category, KeyDownloadBaseURL); ok {
		if normalized, ok := normalizedOptionalBaseURL(value); ok {
			snap.Download.BaseURL = normalized
		}
	}
	if value, ok := p.lookup(ctx, Category, KeyHTTPSDownloadBaseURL); ok {
		if normalized, ok := normalizedOptionalBaseURL(value); ok {
			snap.Download.HTTPSBaseURL = normalized
		}
	}
	if value, ok := p.lookup(ctx, Category, KeyDownloadPath); ok {
		if normalized, ok := normalizedOptionalPath(value); ok {
			snap.Download.Path = normalized
		}
	}
	if value, ok := p.lookup(ctx, Category, KeyDownloadUsername); ok {
		if trimmed, ok := optionalString(value); ok {
			snap.Download.Username = trimmed
		}
	}
	if value, ok := p.lookup(ctx, Category, KeyDownloadPassword); ok {
		if trimmed, ok := optionalString(value); ok {
			snap.Download.Password = trimmed
		}
	}

	if value, ok := p.lookup(ctx, Category, KeyMaxGlobalUpgradeConcurrency); ok {
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed > 0 {
			snap.MaxGlobalUpgradeConcurrency = parsed
		}
	}
}

func optionalString(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func normalizedOptionalBaseURL(value string) (string, bool) {
	trimmed, ok := optionalString(value)
	if !ok {
		return "", false
	}
	return normalizeBaseURL(trimmed), true
}

func normalizedOptionalPath(value string) (string, bool) {
	trimmed, ok := optionalString(value)
	if !ok {
		return "", false
	}
	return normalizePath(trimmed), true
}

func normalizeBaseURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func normalizePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "/" + trimmed
}

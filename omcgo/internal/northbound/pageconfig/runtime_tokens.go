package pageconfig

import (
	"context"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
)

const defaultLocalHostTokenValue = "127.0.0.1"

func configuredLocalHostToken(configured string) string {
	return configuredLocalHostTokenWithProvider(context.Background(), configured, nil)
}

func configuredLocalHostTokenWithProvider(ctx context.Context, configured string, provider transfercfg.Provider) string {
	if host := normalizeLocalHostToken(configured); host != "" {
		return host
	}
	if host := localHostTokenFromTransferProvider(ctx, provider); host != "" {
		return host
	}
	if host := normalizeLocalHostToken(os.Getenv("OMC_PUBLIC_HOST")); host != "" {
		return host
	}
	return defaultLocalHostTokenValue
}

func localHostTokenFromTransferProvider(ctx context.Context, provider transfercfg.Provider) string {
	if provider == nil {
		return ""
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return localHostTokenFromTransferSnapshot(provider.Snapshot(ctx))
}

func localHostTokenFromTransferSnapshot(snapshot transfercfg.Snapshot) string {
	if strings.EqualFold(strings.TrimSpace(snapshot.ProtocolPolicy), transfercfg.ProtocolPolicyPreferHTTPS) {
		if host := usableTransferBaseHost(snapshot.Upload.HTTPSBaseURL); host != "" {
			return host
		}
	}
	if host := usableTransferBaseHost(snapshot.Upload.BaseURL); host != "" {
		return host
	}
	return usableTransferBaseHost(snapshot.Upload.HTTPSBaseURL)
}

func usableTransferBaseHost(value string) string {
	host := normalizeLocalHostToken(value)
	if host == "" || isLoopbackHostToken(host) {
		return ""
	}
	return host
}

func normalizeLocalHostToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "\x00") {
		return ""
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err == nil && parsed.Host != "" {
			value = parsed.Host
		}
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.Trim(strings.TrimSpace(value), "[]")
	if value == "" || strings.ContainsAny(value, `/\`) {
		return ""
	}
	return value
}

func isLoopbackHostToken(host string) bool {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(host), "[]")) {
	case "", "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

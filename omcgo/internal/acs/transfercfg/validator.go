package transfercfg

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

// ValidateBaseURL validates an optional ACS transfer base URL without changing
// the submitted value. Empty values keep the configured startup fallback.
//
// A CPE must be able to reach these endpoints. Development and test deployments
// may use local addresses, while production rejects local-only literal hosts
// before they can be saved as an apparently valid device endpoint.
func ValidateBaseURL(value string) error {
	if value == "" {
		return nil
	}
	if value != strings.TrimSpace(value) {
		return fmt.Errorf("URL must not contain leading or trailing whitespace")
	}

	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https")
	}
	if u.Host == "" || u.Hostname() == "" {
		return fmt.Errorf("URL must be absolute and include a host")
	}
	if u.User != nil {
		return fmt.Errorf("URL must not include user credentials")
	}
	if u.RawQuery != "" || u.ForceQuery {
		return fmt.Errorf("URL must not include a query string")
	}
	if u.Fragment != "" {
		return fmt.Errorf("URL must not include a fragment")
	}
	if strings.Contains(u.Path, `\`) || hasDotPathSegment(u.Path) {
		return fmt.Errorf("URL path must not contain backslashes or dot segments")
	}
	if appconfig.IsProductionEnv() && isLocalOnlyHost(u.Hostname()) {
		return fmt.Errorf("production URL host must be reachable from device networks")
	}
	return nil
}

// ValidateServicePath validates the configured path appended after a transfer
// Base URL. It must be an absolute path on the configured host, not another URL
// or a path that can escape the Base URL prefix.
func ValidateServicePath(value string) error {
	if value == "" {
		return fmt.Errorf("service path is required")
	}
	if value != strings.TrimSpace(value) {
		return fmt.Errorf("service path must not contain leading or trailing whitespace")
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return fmt.Errorf("service path must start with exactly one slash")
	}
	if strings.Contains(value, `\`) {
		return fmt.Errorf("service path must not contain backslashes")
	}
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("parse service path: %w", err)
	}
	if u.IsAbs() || u.Host != "" || u.User != nil {
		return fmt.Errorf("service path must not include a scheme or host")
	}
	if u.RawQuery != "" || u.ForceQuery {
		return fmt.Errorf("service path must not include a query string")
	}
	if u.Fragment != "" {
		return fmt.Errorf("service path must not include a fragment")
	}
	if hasDotPathSegment(u.Path) {
		return fmt.Errorf("service path must not contain dot segments")
	}
	return nil
}

func hasDotPathSegment(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func isLocalOnlyHost(host string) bool {
	normalized := strings.TrimSuffix(host, ".")
	if strings.EqualFold(normalized, "localhost") {
		return true
	}
	// net.ParseIP intentionally accepts only canonical IP literals. URL parsers
	// and some resolvers also accept legacy numeric IPv4 spellings such as 127.1
	// or 2130706433, which can resolve to loopback. Reject an all-numeric host
	// that is not a canonical literal instead of letting it bypass this guard.
	if isNumericHost(normalized) && net.ParseIP(normalized) == nil {
		return true
	}

	ip := net.ParseIP(normalized)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsPrivate()
}

func isNumericHost(host string) bool {
	if host == "" {
		return false
	}
	for _, part := range strings.Split(host, ".") {
		if part == "" {
			return false
		}
		if _, err := strconv.ParseUint(part, 10, 32); err != nil {
			return false
		}
	}
	return true
}

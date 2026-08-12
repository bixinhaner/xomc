package transfercfg

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

// ValidateConfig validates the cross-key contract for acs_transfer settings.
// Missing policy values are treated as force_http for pre-existing databases.
func ValidateConfig(values map[string]string) error {
	policy := values[KeyProtocolPolicy]
	if policy == "" {
		policy = ProtocolPolicyForceHTTP
	}
	if policy != ProtocolPolicyForceHTTP && policy != ProtocolPolicyPreferHTTPS {
		return fmt.Errorf("protocol policy must be %s or %s", ProtocolPolicyForceHTTP, ProtocolPolicyPreferHTTPS)
	}
	if err := ValidateHTTPBaseURL(values[KeyUploadBaseURL]); err != nil {
		return fmt.Errorf("HTTP upload base URL: %w", err)
	}
	if err := ValidateHTTPBaseURL(values[KeyDownloadBaseURL]); err != nil {
		return fmt.Errorf("HTTP download base URL: %w", err)
	}
	if policy == ProtocolPolicyPreferHTTPS &&
		(strings.TrimSpace(values[KeyHTTPSUploadBaseURL]) == "" ||
			strings.TrimSpace(values[KeyHTTPSDownloadBaseURL]) == "") {
		return fmt.Errorf("prefer_https requires both HTTPS upload and download base URLs")
	}
	if err := ValidateHTTPSBaseURL(values[KeyHTTPSUploadBaseURL]); err != nil {
		return fmt.Errorf("HTTPS upload base URL: %w", err)
	}
	if err := ValidateHTTPSBaseURL(values[KeyHTTPSDownloadBaseURL]); err != nil {
		return fmt.Errorf("HTTPS download base URL: %w", err)
	}
	return nil
}

// ValidateHTTPSBaseURL applies the common transfer URL checks and requires an
// HTTPS endpoint. Empty values are allowed for force_http configurations.
func ValidateHTTPSBaseURL(value string) error {
	if err := ValidateBaseURL(value); err != nil {
		return err
	}
	if value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("URL scheme must be https")
	}
	return nil
}

// ValidateHTTPBaseURL applies the common transfer URL checks and keeps the
// existing uploadBaseURL/downloadBaseURL keys reserved for HTTP endpoints.
func ValidateHTTPBaseURL(value string) error {
	if err := ValidateBaseURL(value); err != nil {
		return err
	}
	if value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}
	if u.Scheme != "http" {
		return fmt.Errorf("URL scheme must be http")
	}
	return nil
}

// ValidateBaseURL validates an optional ACS transfer base URL without changing
// the submitted value. Empty values keep the configured startup fallback.
//
// A CPE must be able to reach these endpoints. Reachability depends on the
// operator network topology, so private address ranges are valid production
// endpoints. Production only rejects literal hosts that cannot identify the
// OMC endpoint from a device network, such as loopback or unspecified addresses.
func ValidateBaseURL(value string) error {
	if value == "" {
		return nil
	}
	if value != strings.TrimSpace(value) {
		return fmt.Errorf("URL must not contain leading or trailing whitespace")
	}
	if strings.ContainsAny(value, "?#") {
		return fmt.Errorf("URL must not include a query or fragment delimiter")
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
	if err := validateURLPort(u.Host); err != nil {
		return err
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

func validateURLPort(host string) error {
	rawPort := ""
	portSpecified := false
	if strings.HasPrefix(host, "[") {
		closingBracket := strings.LastIndex(host, "]")
		if closingBracket < 0 {
			return fmt.Errorf("URL contains an invalid IPv6 host")
		}
		suffix := host[closingBracket+1:]
		if suffix == "" {
			return nil
		}
		if !strings.HasPrefix(suffix, ":") {
			return fmt.Errorf("URL contains an invalid port")
		}
		rawPort = suffix[1:]
		portSpecified = true
	} else if colon := strings.LastIndex(host, ":"); colon >= 0 {
		rawPort = host[colon+1:]
		portSpecified = true
	}
	if !portSpecified {
		return nil
	}
	if rawPort == "" {
		return fmt.Errorf("URL port must not be empty")
	}
	for _, char := range rawPort {
		if char < '0' || char > '9' {
			return fmt.Errorf("URL port must be numeric")
		}
	}
	port, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil || port == 0 {
		return fmt.Errorf("URL port must be between 1 and 65535")
	}
	return nil
}

// ValidateServicePath validates the configured path appended after a transfer
// Base URL. Empty values keep the configured startup fallback. Non-empty values
// must be absolute paths on the configured host, not another URL or a path that
// can escape the Base URL prefix.
func ValidateServicePath(value string) error {
	if value == "" {
		return nil
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
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
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

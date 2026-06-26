package admin

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

func auditClientIP(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	privateCandidate := ""
	for _, candidate := range auditIPHeaderCandidates(c) {
		if ip := normalizeAuditIP(candidate); ip != "" {
			parsed := net.ParseIP(ip)
			if parsed != nil && !isPrivateOrLocalIP(parsed) {
				return ip
			}
			if privateCandidate == "" {
				privateCandidate = ip
			}
		}
	}
	if privateCandidate != "" {
		return privateCandidate
	}

	return normalizeAuditIP(c.ClientIP())
}

func isPrivateOrLocalIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// carrier-grade NAT 100.64.0.0/10
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}
	return false
}

func auditIPHeaderCandidates(c *gin.Context) []string {
	candidates := make([]string, 0, 4)
	if forwarded := c.GetHeader("Forwarded"); forwarded != "" {
		for _, part := range strings.Split(forwarded, ",") {
			for _, attr := range strings.Split(part, ";") {
				key, value, ok := strings.Cut(strings.TrimSpace(attr), "=")
				if ok && strings.EqualFold(key, "for") {
					candidates = append(candidates, value)
				}
			}
		}
	}
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		candidates = append(candidates, strings.Split(xff, ",")...)
	}
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		candidates = append(candidates, realIP)
	}
	return candidates
}

func normalizeAuditIP(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	if value == "" || strings.EqualFold(value, "unknown") {
		return ""
	}

	if strings.HasPrefix(value, "[") {
		if end := strings.Index(value, "]"); end >= 0 {
			value = value[1:end]
		}
	} else if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}

	if ip, _, err := net.ParseCIDR(value); err == nil {
		return ip.String()
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}

	return ""
}

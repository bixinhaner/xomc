package agentruntime

import (
	"net/http"
	"path"
	"strings"

	"github.com/omcgo/omcgo/internal/agentconfig"
)

func methodAllowed(policy agentconfig.RuntimePolicy, method string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(method))
	for _, allowed := range policy.AllowedMethods {
		if strings.ToUpper(strings.TrimSpace(allowed)) == normalized {
			return true
		}
	}
	return false
}

func pathBlocked(policy agentconfig.RuntimePolicy, requestPath string) bool {
	cleanPath := cleanAPIPath(requestPath)
	for _, prefix := range policy.BlockedPathPrefixes {
		normalized := strings.TrimSpace(prefix)
		if normalized == "" {
			continue
		}
		if strings.HasSuffix(normalized, "*") {
			if strings.HasPrefix(cleanPath, strings.TrimSuffix(normalized, "*")) {
				return true
			}
			continue
		}
		if cleanPath == normalized || strings.HasPrefix(cleanPath, strings.TrimRight(normalized, "/")+"/") {
			return true
		}
	}
	return false
}

func cleanAPIPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return path.Clean(trimmed)
}

func isReadMethod(method string) bool {
	return strings.EqualFold(method, http.MethodGet) || strings.EqualFold(method, http.MethodHead)
}

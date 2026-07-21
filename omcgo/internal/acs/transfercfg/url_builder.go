package transfercfg

import (
	"fmt"
	"net/url"
	"strings"
)

// BuildURL constructs an ACS transfer endpoint without changing the Base URL
// host, port, or reverse-proxy prefix. Query values are encoded by net/url;
// objectSegments are appended as individual escaped path segments.
func BuildURL(baseURL, servicePath string, objectSegments []string, query url.Values) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return "", fmt.Errorf("transfer base URL is empty")
	}
	if err := ValidateBaseURL(baseURL); err != nil {
		return "", fmt.Errorf("validate transfer base URL: %w", err)
	}
	if err := ValidateServicePath(servicePath); err != nil {
		return "", fmt.Errorf("validate transfer service path: %w", err)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse transfer base URL: %w", err)
	}
	service, err := url.Parse(servicePath)
	if err != nil {
		return "", fmt.Errorf("parse transfer service path: %w", err)
	}

	escapedPath := strings.TrimRight(base.EscapedPath(), "/") + "/" + strings.TrimLeft(service.EscapedPath(), "/")
	for _, segment := range objectSegments {
		if segment == "" || segment == "." || segment == ".." || strings.ContainsAny(segment, `/\`) {
			return "", fmt.Errorf("invalid transfer object path segment %q", segment)
		}
		escapedPath = strings.TrimRight(escapedPath, "/") + "/" + url.PathEscape(segment)
	}

	decodedPath, err := url.PathUnescape(escapedPath)
	if err != nil {
		return "", fmt.Errorf("decode constructed transfer path: %w", err)
	}
	base.Path = decodedPath
	base.RawPath = escapedPath
	base.RawQuery = query.Encode()
	base.ForceQuery = false
	return base.String(), nil
}

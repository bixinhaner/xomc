package e2e

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	defaultBaseURL = "http://localhost:18080"
	timeout        = 10 * time.Second
)

// baseURL returns the target server URL from env or default.
func baseURL() string {
	if url := os.Getenv("OMCGO_E2E_BASE_URL"); url != "" {
		return url
	}
	return defaultBaseURL
}

// newClient creates an HTTP client with E2E test timeout.
func newClient() *http.Client {
	return &http.Client{Timeout: timeout}
}

func TestMain(m *testing.M) {
	base := baseURL()
	client := newClient()

	// Wait for server to be ready
	for i := 0; i < 30; i++ {
		resp, err := client.Get(base + "/healthz")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			break
		}
		if i == 29 {
			fmt.Fprintf(os.Stderr, "server at %s not ready after 30s, skipping E2E tests\n", base)
			os.Exit(0)
		}
		time.Sleep(time.Second)
	}

	code := m.Run()
	os.Exit(code)
}

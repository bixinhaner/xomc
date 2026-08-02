package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// internalAPIKeyPath 容器内默认 API key 文件路径,由 app 启动期签发。
// 与 internal/admin.DefaultInternalAPIKeyPath 保持一致 (单独写在这里避免拉
// admin 包的重依赖 pgxpool/casbin 等到 omcctl 二进制)。
const internalAPIKeyPath = "/var/lib/omcgo/secrets/.api-key"

// loadInternalAPIKeyFile 读 internalAPIKeyPath 并 trim。
// 文件不存在返回 "" + nil (调用方继续 fallback)。
func loadInternalAPIKeyFile() string {
	raw, err := os.ReadFile(internalAPIKeyPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

var (
	flagServer string
	flagAPIKey string
	flagOutput string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcctl",
		Short: "OMC CLI Management Tool",
		Long:  "Command-line tool for managing OMC devices, alarms, PM data, and system operations",
	}

	// --server 优先级:--server flag > OMCCTL_SERVER env > hardcoded localhost:8080
	// 镜像内(Dockerfile.worker)已设 OMCCTL_SERVER=http://app:8081,operator 进
	// worker 容器跑无需显式 --server。
	defaultServer := os.Getenv("OMCCTL_SERVER")
	if defaultServer == "" {
		defaultServer = "http://localhost:8080"
	}
	rootCmd.PersistentFlags().StringVar(&flagServer, "server", defaultServer, "OMC App server address (env: OMCCTL_SERVER)")

	// --api-key 优先级:--api-key flag > OMCCTL_API_KEY env > /var/lib/omcgo/secrets/.api-key 文件
	// 文件由 app 启动期幂等签发 (admin.EnsureInternalAPIKey),容器内零配置默认入口。
	// 详见 omcgo/CLAUDE.md §5.6。
	defaultAPIKey := os.Getenv("OMCCTL_API_KEY")
	if defaultAPIKey == "" {
		defaultAPIKey = loadInternalAPIKeyFile()
	}
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", defaultAPIKey,
		"API key (env: OMCCTL_API_KEY, fallback: /var/lib/omcgo/secrets/.api-key)")
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "table", "Output format: table or json")

	rootCmd.AddCommand(
		newDeviceCmd(),
		newAlarmCmd(),
		newPMCmd(),
		newPMRedisCmd(),
		newSystemCmd(),
		newMMLCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getClient() *OMCClient {
	return NewOMCClient(flagServer, flagAPIKey)
}

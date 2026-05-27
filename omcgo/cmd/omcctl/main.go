package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", os.Getenv("OMCCTL_API_KEY"), "API key for authentication (env: OMCCTL_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "table", "Output format: table or json")

	rootCmd.AddCommand(
		newDeviceCmd(),
		newAlarmCmd(),
		newPMCmd(),
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

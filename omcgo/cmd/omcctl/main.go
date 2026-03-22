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

	rootCmd.PersistentFlags().StringVar(&flagServer, "server", "http://localhost:8080", "OMC App server address")
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", os.Getenv("OMCCTL_API_KEY"), "API key for authentication")
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "table", "Output format: table or json")

	rootCmd.AddCommand(
		newDeviceCmd(),
		newAlarmCmd(),
		newPMCmd(),
		newSystemCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getClient() *OMCClient {
	return NewOMCClient(flagServer, flagAPIKey)
}

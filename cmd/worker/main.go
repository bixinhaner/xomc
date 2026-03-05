package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-worker",
		Short: "OMC Background Worker",
		Long:  "Background worker process for PM/MR file processing and KPI calculation",
		RunE:  runWorker,
	}

	rootCmd.Flags().String("config", "configs/worker.yaml", "configuration file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runWorker(cmd *cobra.Command, args []string) error {
	fmt.Println("omcgo-worker starting...")
	return nil
}

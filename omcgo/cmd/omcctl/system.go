package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "System management commands",
	}

	// system info
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show system information",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			resp, err := client.Get("/api/v1/system/info")
			if err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}

	// system health
	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Check system health",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			resp, err := client.Get("/healthz")
			if err != nil {
				return fmt.Errorf("health check failed: %w", err)
			}
			status := getString(resp, "status")
			if status == "ok" {
				fmt.Println("System is healthy")
			} else {
				fmt.Printf("System status: %s\n", status)
			}
			return nil
		},
	}

	cmd.AddCommand(infoCmd, healthCmd)
	return cmd
}

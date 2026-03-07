package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcctl",
		Short: "OMC CLI Management Tool",
		Long:  "Command-line tool for managing OMC devices, alarms, PM data, and system operations",
	}

	rootCmd.PersistentFlags().String("server", "http://localhost:8080", "OMC App server address")

	rootCmd.AddCommand(
		newDeviceCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newDeviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "Device management commands",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List devices",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("listing devices...")
				return nil
			},
		},
		&cobra.Command{
			Use:   "get [id]",
			Short: "Get device details",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("getting device %s...\n", args[0])
				return nil
			},
		},
	)

	return cmd
}

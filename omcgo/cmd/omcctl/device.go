package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDeviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "Device management commands",
	}

	// device list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			carrier, _ := cmd.Flags().GetString("carrier")
			status, _ := cmd.Flags().GetString("status")
			limit, _ := cmd.Flags().GetInt("limit")

			path := "/api/v1/devices?"
			if carrier != "" {
				path += "carrier=" + carrier + "&"
			}
			if status != "" {
				path += "status=" + status + "&"
			}
			if limit > 0 {
				path += fmt.Sprintf("page_size=%d&", limit)
			}

			resp, err := client.Get(path)
			if err != nil {
				return err
			}

			if flagOutput == "json" {
				printJSON(resp)
				return nil
			}

			items := extractItems(resp)
			headers := []string{"ID", "SERIAL_NUMBER", "CARRIER", "STATUS", "MANUFACTURER"}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				rows = append(rows, []string{
					getString(m, "id"),
					getString(m, "serial_number"),
					getString(m, "carrier"),
					getString(m, "status"),
					getString(m, "manufacturer"),
				})
			}
			printTable(headers, rows)
			return nil
		},
	}
	listCmd.Flags().String("carrier", "", "Filter by carrier (cmcc/ctcc/cucc)")
	listCmd.Flags().String("status", "", "Filter by status")
	listCmd.Flags().Int("limit", 20, "Number of results")

	// device get
	getCmd := &cobra.Command{
		Use:   "get [id]",
		Short: "Get device details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			resp, err := client.Get("/api/v1/devices/" + args[0])
			if err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}

	// device reboot
	rebootCmd := &cobra.Command{
		Use:   "reboot [id]",
		Short: "Reboot a device",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			resp, err := client.Post("/api/v1/devices/"+args[0]+"/reboot", nil)
			if err != nil {
				return err
			}
			fmt.Println(getString(resp, "message"))
			return nil
		},
	}

	// device delete
	deleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a device",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			if err := client.Delete("/api/v1/devices/" + args[0]); err != nil {
				return err
			}
			fmt.Println("device deleted")
			return nil
		},
	}

	// device stats
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show device statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			resp, err := client.Get("/api/v1/devices/stats")
			if err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, rebootCmd, deleteCmd, statsCmd)
	return cmd
}

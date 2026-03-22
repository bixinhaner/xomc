package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAlarmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alarm",
		Short: "Alarm management commands",
	}

	// alarm list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List alarms",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			severity, _ := cmd.Flags().GetString("severity")
			status, _ := cmd.Flags().GetString("status")
			limit, _ := cmd.Flags().GetInt("limit")

			path := "/api/v1/alarms?"
			if severity != "" {
				path += "severity=" + severity + "&"
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
			headers := []string{"ID", "DEVICE_SN", "ALARM_TYPE", "SEVERITY", "STATUS", "RAISED_AT"}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				rows = append(rows, []string{
					getString(m, "id"),
					getString(m, "device_serial_number"),
					getString(m, "alarm_type"),
					getString(m, "severity"),
					getString(m, "status"),
					getString(m, "raised_at"),
				})
			}
			printTable(headers, rows)
			return nil
		},
	}
	listCmd.Flags().String("severity", "", "Filter by severity (critical/major/minor/warning)")
	listCmd.Flags().String("status", "", "Filter by status (active/cleared/acknowledged)")
	listCmd.Flags().Int("limit", 20, "Number of results")

	// alarm ack
	ackCmd := &cobra.Command{
		Use:   "ack [id]",
		Short: "Acknowledge an alarm",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			resp, err := client.Post("/api/v1/alarms/"+args[0]+"/acknowledge", nil)
			if err != nil {
				return err
			}
			fmt.Println(getString(resp, "message"))
			return nil
		},
	}

	// alarm clear
	clearCmd := &cobra.Command{
		Use:   "clear [id]",
		Short: "Clear an alarm",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			resp, err := client.Post("/api/v1/alarms/"+args[0]+"/clear", nil)
			if err != nil {
				return err
			}
			fmt.Println(getString(resp, "message"))
			return nil
		},
	}

	// alarm stats
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show alarm statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			resp, err := client.Get("/api/v1/alarms/stats")
			if err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}

	cmd.AddCommand(listCmd, ackCmd, clearCmd, statsCmd)
	return cmd
}

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pm",
		Short: "Performance management commands",
	}

	// pm counters
	countersCmd := &cobra.Command{
		Use:   "counters",
		Short: "Query PM counters",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			deviceID, _ := cmd.Flags().GetString("device-id")
			from, _ := cmd.Flags().GetString("from")
			to, _ := cmd.Flags().GetString("to")
			limit, _ := cmd.Flags().GetInt("limit")

			path := "/api/v1/pm/counters?"
			if deviceID != "" {
				path += "device_id=" + deviceID + "&"
			}
			if from != "" {
				path += "start_time=" + from + "&"
			}
			if to != "" {
				path += "end_time=" + to + "&"
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
			headers := []string{"TIMESTAMP", "DEVICE_SN", "COUNTER_NAME", "VALUE"}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				rows = append(rows, []string{
					getString(m, "timestamp"),
					getString(m, "device_serial_number"),
					getString(m, "counter_name"),
					getString(m, "value"),
				})
			}
			printTable(headers, rows)
			return nil
		},
	}
	countersCmd.Flags().String("device-id", "", "Filter by device ID")
	countersCmd.Flags().String("from", "", "Start time (RFC3339)")
	countersCmd.Flags().String("to", "", "End time (RFC3339)")
	countersCmd.Flags().Int("limit", 20, "Number of results")

	// pm kpi
	kpiCmd := &cobra.Command{
		Use:   "kpi",
		Short: "Query KPI values",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			name, _ := cmd.Flags().GetString("name")
			from, _ := cmd.Flags().GetString("from")
			to, _ := cmd.Flags().GetString("to")
			limit, _ := cmd.Flags().GetInt("limit")

			path := "/api/v1/pm/kpi?"
			if name != "" {
				path += "name=" + name + "&"
			}
			if from != "" {
				path += "start_time=" + from + "&"
			}
			if to != "" {
				path += "end_time=" + to + "&"
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
			headers := []string{"TIMESTAMP", "KPI_NAME", "VALUE", "GRANULARITY"}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				rows = append(rows, []string{
					getString(m, "timestamp"),
					getString(m, "kpi_name"),
					getString(m, "value"),
					getString(m, "granularity"),
				})
			}
			printTable(headers, rows)
			return nil
		},
	}
	kpiCmd.Flags().String("name", "", "Filter by KPI name")
	kpiCmd.Flags().String("from", "", "Start time (RFC3339)")
	kpiCmd.Flags().String("to", "", "End time (RFC3339)")
	kpiCmd.Flags().Int("limit", 20, "Number of results")

	// pm aggregate - 手动触发 PM 聚合 recompute
	aggregateCmd := &cobra.Command{
		Use:   "aggregate",
		Short: "Manually trigger PM aggregation recompute",
		Long:  "POST /api/v1/pm/aggregation/recompute to enqueue a recompute job. Returns the job_id; use SQL to poll async_jobs status (no LIST API yet).",
		RunE: func(cmd *cobra.Command, args []string) error {
			granularity, _ := cmd.Flags().GetString("granularity")
			dimension, _ := cmd.Flags().GetString("dimension")
			start, _ := cmd.Flags().GetString("start")
			end, _ := cmd.Flags().GetString("end")

			client := getClient()
			payload := map[string]interface{}{
				"granularity": granularity,
				"dimension":   dimension,
				"start":       start,
				"end":         end,
			}
			resp, err := client.Post("/api/v1/pm/aggregation/recompute", payload)
			if err != nil {
				return err
			}

			if flagOutput == "json" {
				printJSON(resp)
				return nil
			}

			jobID := getString(resp, "job_id")
			if jobID == "" {
				jobID = getString(resp, "id")
			}
			if jobID == "" {
				printJSON(resp)
				return nil
			}
			fmt.Println("job_id:", jobID)
			fmt.Println("Status query (no UI yet):")
			fmt.Printf("  docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c \"SELECT id, status, started_at, finished_at FROM async_jobs WHERE id='%s'\"\n", jobID)
			return nil
		},
	}
	aggregateCmd.Flags().String("granularity", "", "Granularity: hourly | daily | weekly | monthly (required)")
	aggregateCmd.Flags().String("dimension", "device", "Dimension: device | device_group")
	aggregateCmd.Flags().String("start", "", "Window start (RFC3339, required)")
	aggregateCmd.Flags().String("end", "", "Window end (RFC3339, required)")
	_ = aggregateCmd.MarkFlagRequired("granularity")
	_ = aggregateCmd.MarkFlagRequired("start")
	_ = aggregateCmd.MarkFlagRequired("end")

	cmd.AddCommand(countersCmd, kpiCmd, aggregateCmd)
	return cmd
}

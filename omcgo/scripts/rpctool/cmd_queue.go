package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/task"
	"github.com/spf13/cobra"
)

const taskQueueKeyPrefix = "acs:taskq:"

func newQueueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "任务队列管理 — 查看队列、查询历史任务",
	}

	cmd.AddCommand(
		newQueueListCmd(),
		newQueuePeekCmd(),
		newQueueLenCmd(),
		newQueueClearCmd(),
		newQueueHistoryCmd(),
	)

	return cmd
}

func newQueueListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "列出 Redis 队列中所有待执行任务",
		Example: `  rpctool queue list --sn DEVICE001`,
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			key := taskQueueKeyPrefix + deviceSN

			results, err := redisClient.ZRangeWithScores(ctx, key, 0, -1).Result()
			if err != nil {
				return fmt.Errorf("查询队列: %w", err)
			}

			if len(results) == 0 {
				fmt.Printf("设备 %s 的任务队列为空\n", deviceSN)
				return nil
			}

			fmt.Printf("设备 %s 的任务队列 (%d 条):\n", deviceSN, len(results))
			fmt.Println("─────────────────────────────────────────")

			for i, r := range results {
				var t task.Task
				if err := json.Unmarshal([]byte(r.Member.(string)), &t); err != nil {
					fmt.Printf("[%d] 解析失败: %v\n", i+1, err)
					continue
				}

				fmt.Printf("[%d] 方法: %-25s 优先级: %d  ID: %s\n",
					i+1, t.Method, t.Priority, t.ID)
				fmt.Printf("    状态: %s  来源: %s  创建者: %s\n",
					t.Status, t.Source, t.CreatorID)

				if t.ExpiresAt != nil {
					fmt.Printf("    过期时间: %s\n", t.ExpiresAt.Format("2006-01-02 15:04:05"))
				}

				if len(t.Params) > 0 && string(t.Params) != "null" {
					var pretty json.RawMessage
					if json.Unmarshal(t.Params, &pretty) == nil {
						indented, _ := json.MarshalIndent(pretty, "    ", "  ")
						fmt.Printf("    参数: %s\n", string(indented))
					}
				}

				if i < len(results)-1 {
					fmt.Println()
				}
			}

			return nil
		},
	}
}

func newQueuePeekCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "peek",
		Short:   "查看队首任务（不移除）",
		Example: `  rpctool queue peek --sn DEVICE001`,
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			taskQueue := task.NewRedisTaskQueue(redisClient)

			t, err := taskQueue.Peek(ctx, deviceSN)
			if err != nil {
				return fmt.Errorf("查看队首: %w", err)
			}

			if t == nil {
				fmt.Printf("设备 %s 的任务队列为空\n", deviceSN)
				return nil
			}

			fmt.Printf("设备 %s 的队首任务:\n", deviceSN)
			prettyJSON, _ := json.MarshalIndent(t, "", "  ")
			fmt.Println(string(prettyJSON))

			return nil
		},
	}
}

func newQueueLenCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "len",
		Short:   "查看队列长度",
		Example: `  rpctool queue len --sn DEVICE001`,
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			taskQueue := task.NewRedisTaskQueue(redisClient)

			n, err := taskQueue.Len(ctx, deviceSN)
			if err != nil {
				return fmt.Errorf("查询队列长度: %w", err)
			}

			fmt.Printf("设备 %s 的任务队列长度: %d\n", deviceSN, n)
			return nil
		},
	}
}

func newQueueClearCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "clear",
		Short:   "清空任务队列",
		Example: `  rpctool queue clear --sn DEVICE001 --force`,
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			key := taskQueueKeyPrefix + deviceSN

			if !force {
				n, _ := redisClient.ZCard(ctx, key).Result()
				if n > 0 {
					fmt.Printf("设备 %s 的队列中有 %d 条任务，使用 --force 确认清空\n", deviceSN, n)
					return nil
				}
				fmt.Printf("设备 %s 的队列已为空\n", deviceSN)
				return nil
			}

			if err := redisClient.Del(ctx, key).Err(); err != nil {
				return fmt.Errorf("清空队列: %w", err)
			}

			fmt.Printf("已清空设备 %s 的任务队列\n", deviceSN)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "确认清空队列")

	return cmd
}

func newQueueHistoryCmd() *cobra.Command {
	var (
		status   string
		page     int
		pageSize int
		days     int
	)

	cmd := &cobra.Command{
		Use:   "history",
		Short: "查询数据库中的历史任务记录",
		Long: `从 PostgreSQL device_tasks 表查询历史任务。
支持按状态过滤和时间范围查询。`,
		Example: `  rpctool queue history --sn DEVICE001
  rpctool queue history --sn DEVICE001 --status completed
  rpctool queue history --sn DEVICE001 --status failed --days 7`,
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			opts := &task.TaskHistoryOptions{
				Page:     page,
				PageSize: pageSize,
			}

			if status != "" {
				opts.Status = task.TaskStatus(status)
			}

			if days > 0 {
				t := time.Now().AddDate(0, 0, -days)
				opts.Start = &t
			}

			resp, err := taskSvc.GetTaskHistory(ctx, deviceSN, opts)
			if err != nil {
				return fmt.Errorf("查询任务历史: %w", err)
			}

			if len(resp.Tasks) == 0 {
				fmt.Printf("设备 %s 无任务记录", deviceSN)
				if status != "" {
					fmt.Printf(" (状态: %s)", status)
				}
				fmt.Println()
				return nil
			}

			fmt.Printf("设备 %s 的任务历史 (共 %d 条，第 %d 页):\n", deviceSN, resp.Total, resp.Page)
			fmt.Println("─────────────────────────────────────────────────────────────")

			for i, t := range resp.Tasks {
				fmt.Printf("[%d] ID: %s\n", i+1, t.ID)
				fmt.Printf("    方法: %-20s 状态: %-12s 优先级: %d\n", t.Method, t.Status, t.Priority)
				fmt.Printf("    来源: %-10s 创建者: %s\n", t.Source, t.CreatorID)
				fmt.Printf("    创建: %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))

				if t.SentAt != nil {
					fmt.Printf("    发送: %s\n", t.SentAt.Format("2006-01-02 15:04:05"))
				}
				if t.CompletedAt != nil {
					fmt.Printf("    完成: %s\n", t.CompletedAt.Format("2006-01-02 15:04:05"))
				}
				if t.Description != "" {
					fmt.Printf("    描述: %s\n", t.Description)
				}
				if t.ErrorMessage != "" {
					fmt.Printf("    错误: [%d] %s\n", t.ErrorCode, t.ErrorMessage)
				}

				if i < len(resp.Tasks)-1 {
					fmt.Println()
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "按状态过滤 (pending/sent/completed/failed/expired/cancelled)")
	cmd.Flags().IntVar(&page, "page", 1, "页码")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "每页条数")
	cmd.Flags().IntVar(&days, "days", 0, "最近 N 天内的记录 (0=不限)")

	return cmd
}

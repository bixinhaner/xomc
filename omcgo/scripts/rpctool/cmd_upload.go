package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newUploadCmd() *cobra.Command {
	var (
		fileType string
		url      string
		username string
		password string
		delay    int
	)

	cmd := &cobra.Command{
		Use:   "upload [文件类型别名]",
		Short: "推送 Upload RPC 任务 — 让基站上传文件到指定 URL",
		Long: `创建 TR-069 Upload RPC 任务（同时写入 Redis 队列和 PostgreSQL）。
设备收到后会将指定类型的文件上传到给定 URL。

文件类型别名 (TR-069 标准):
  config       (1)  配置文件      → "1 Vendor Configuration File"
  log          (2)  运行日志      → "2 Vendor Log File"

文件类型别名 (运营商扩展):
  pm           (4)  性能管理文件
  mr           (5)  测量报告文件
  security-log      安全日志
  fault-log         故障日志
  pcap         (9)  抓包文件
  datamodel    (11) 数据模型文件`,
		Example: `  # 让设备上传运行日志
  rpctool upload log --sn DEVICE001 --url http://acs:7547/upload

  # 让设备上传配置文件（TR-069 标准 FileType "1 Vendor Configuration File"）
  rpctool upload config --sn DEVICE001 --url http://acs:7547/upload

  # 采集 PM 性能数据
  rpctool upload pm --sn DEVICE001 --url http://acs:7547/upload

  # 仅预览，不写入数据库
  rpctool upload log --sn DEVICE001 --url http://acs:7547/upload --dry-run`,
		Args:    cobra.MaximumNArgs(1),
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			ft := fileType
			if len(args) > 0 {
				ft = args[0]
			}
			if ft == "" {
				return fmt.Errorf("必须指定文件类型: 使用位置参数 (如 upload log) 或 --file-type 标志")
			}

			resolvedFT, err := resolveUploadFileType(ft)
			if err != nil {
				return err
			}

			if url == "" {
				return fmt.Errorf("必须指定 --url (文件上传目标地址)")
			}

			params := map[string]interface{}{
				"file_type":     resolvedFT,
				"url":           url,
				"username":      username,
				"password":      password,
				"delay_seconds": delay,
			}

			return createAndPrint(context.Background(), deviceSN, "Upload", params,
				fmt.Sprintf("Upload %s to %s", resolvedFT, url))
		},
	}

	cmd.Flags().StringVarP(&fileType, "file-type", "t", "", "文件类型 (别名或数字代码)")
	cmd.Flags().StringVarP(&url, "url", "u", "", "文件上传目标 URL (必填)")
	cmd.Flags().StringVar(&username, "username", "", "HTTP 认证用户名")
	cmd.Flags().StringVar(&password, "password", "", "HTTP 认证密码")
	cmd.Flags().IntVar(&delay, "delay", 0, "延迟执行秒数")

	return cmd
}

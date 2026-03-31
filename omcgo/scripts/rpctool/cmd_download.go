package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newDownloadCmd() *cobra.Command {
	var (
		fileType   string
		url        string
		username   string
		password   string
		fileSize   int64
		targetFile string
		delay      int
	)

	cmd := &cobra.Command{
		Use:   "download [文件类型别名]",
		Short: "推送 Download RPC 任务 — 让基站从指定 URL 下载文件",
		Long: `创建 TR-069 Download RPC 任务（同时写入 Redis 队列和 PostgreSQL）。
设备收到后会从给定 URL 下载指定类型的文件。

文件类型别名 (TR-069 标准):
  firmware  (1)  固件升级镜像  → "1 Firmware Upgrade Image"
  web       (2)  Web 内容      → "2 Web Content"
  config    (3)  配置文件      → "3 Vendor Configuration File"`,
		Example: `  # 让设备下载固件
  rpctool download firmware --sn DEVICE001 --url http://minio:9000/fw/v2.0.bin --file-size 10485760

  # 让设备下载配置文件
  rpctool download config --sn DEVICE001 --url http://minio:9000/configs/device.xml`,
		Args:    cobra.MaximumNArgs(1),
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			ft := fileType
			if len(args) > 0 {
				ft = args[0]
			}
			if ft == "" {
				return fmt.Errorf("必须指定文件类型: 使用位置参数 (如 download firmware) 或 --file-type 标志")
			}

			resolvedFT, err := resolveDownloadFileType(ft)
			if err != nil {
				return err
			}

			if url == "" {
				return fmt.Errorf("必须指定 --url (文件下载地址)")
			}

			params := map[string]interface{}{
				"file_type":        resolvedFT,
				"url":              url,
				"username":         username,
				"password":         password,
				"file_size":        fileSize,
				"target_file_name": targetFile,
				"delay_seconds":    delay,
			}

			return createAndPrint(context.Background(), deviceSN, "Download", params,
				fmt.Sprintf("Download %s from %s", resolvedFT, url))
		},
	}

	cmd.Flags().StringVarP(&fileType, "file-type", "t", "", "文件类型 (别名或数字代码)")
	cmd.Flags().StringVarP(&url, "url", "u", "", "文件下载地址 (必填)")
	cmd.Flags().StringVar(&username, "username", "", "HTTP 认证用户名")
	cmd.Flags().StringVar(&password, "password", "", "HTTP 认证密码")
	cmd.Flags().Int64Var(&fileSize, "file-size", 0, "文件大小 (字节)")
	cmd.Flags().StringVar(&targetFile, "target", "", "目标文件名")
	cmd.Flags().IntVar(&delay, "delay", 0, "延迟执行秒数")

	return cmd
}

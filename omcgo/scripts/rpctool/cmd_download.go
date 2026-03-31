package main

import (
	"context"
	"fmt"
	"strings"

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
		oui        string
		rawMode    int
	)

	cmd := &cobra.Command{
		Use:   "download [文件类型别名]",
		Short: "推送 Download RPC 任务 — 让基站从指定 URL 下载文件",
		Long: `创建 TR-069 Download RPC 任务（同时写入 Redis 队列和 PostgreSQL）。
设备收到后会从给定 URL 下载指定类型的文件。

文件类型别名 (TR-069 标准):
  firmware   (1)    固件升级镜像       → "1 Firmware Upgrade Image"
  web        (2)    Web 内容           → "2 Web Content"
  config     (3)    厂商配置文件       → "3 Vendor Configuration File"

文件类型别名 (厂商/运营商扩展):
  oui-config (10)   OUI 配置文件       → "10 <OUI> Configuration File" (需 --oui)
  script     (101)  脚本文件           → "101 Script File"
  startup    (103)  基站启动文件       → "103 Base Station Startup File"
  license           License 文件       → "License File"
  ssl-cert          TR069 SSL 证书     → "Tr069 Ssl Cert File"

也可直接传入完整的 FileType 字符串（含空格），如:
  rpctool download --file-type "1 Firmware Upgrade Image" --sn DEVICE001 --url ...`,
		Example: `  # 固件升级（保配置，默认）— MinIO 路径由 ACS 自动翻译为 CPE 可访问的 HTTP 地址
  rpctool download firmware --sn DEVICE001 --url firmware/v2.0.bin --file-size 52428800

  # 固件升级（不保配置，RawMode=1，升级后恢复出厂设置）
  rpctool download firmware --sn DEVICE001 --url firmware/v2.0.bin --file-size 52428800 --raw-mode 1

  # 下载配置文件
  rpctool download config --sn DEVICE001 --url config-backup/device.xml

  # 下载 OUI 配置文件（自动解析并批量导入参数）
  rpctool download oui-config --sn DEVICE001 --url config-backup/oui_config.xml --oui 48BF74

  # 下载脚本文件
  rpctool download script --sn DEVICE001 --url omc-exchange/scripts/init.sh

  # 下载 License 文件
  rpctool download license --sn DEVICE001 --url omc-exchange/license/device.lic

  # 下载 SSL 证书
  rpctool download ssl-cert --sn DEVICE001 --url omc-exchange/certs/tr069_ca.crt

  # 下载基站启动文件
  rpctool download startup --sn DEVICE001 --url omc-exchange/startup/config.bin`,
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

			var resolvedFT string
			resolvedFT, err := resolveDownloadFileType(ft)
			if err != nil {
				// Special handling for OUI config
				if strings.Contains(err.Error(), "__OUI_CONFIG__") {
					resolvedFT = resolveOUIConfigFileType(oui)
				} else {
					return err
				}
			}

			if url == "" {
				return fmt.Errorf("必须指定 --url (文件下载地���)")
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

			// RawMode: 厂商扩展参数，控制升级时是否保留配置
			if rawMode > 0 {
				params["raw_mode"] = rawMode
			}

			return createAndPrint(context.Background(), deviceSN, "Download", params,
				fmt.Sprintf("Download %s from %s", resolvedFT, url))
		},
	}

	cmd.Flags().StringVarP(&fileType, "file-type", "t", "", "文件类型 (别名、数字代码或完整 FileType 字符串)")
	cmd.Flags().StringVarP(&url, "url", "u", "", "文件下载地址 (必填)")
	cmd.Flags().StringVar(&username, "username", "", "HTTP 认证用户名")
	cmd.Flags().StringVar(&password, "password", "", "HTTP 认证密码")
	cmd.Flags().Int64Var(&fileSize, "file-size", 0, "文件大小 (字节)")
	cmd.Flags().StringVar(&targetFile, "target", "", "目标文件名")
	cmd.Flags().IntVar(&delay, "delay", 0, "延迟执行秒数")
	cmd.Flags().StringVar(&oui, "oui", defaultOUI, "OUI 标识 (仅 oui-config 类型使用)")
	cmd.Flags().IntVar(&rawMode, "raw-mode", 0, "升级模式: 0=保配置(默认), 1=不保配置(恢复出厂)")

	return cmd
}

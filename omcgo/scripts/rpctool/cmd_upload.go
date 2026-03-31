package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newUploadCmd() *cobra.Command {
	var (
		fileType string
		url      string
		username string
		password string
		delay    int
		oui      string
	)

	cmd := &cobra.Command{
		Use:   "upload [文件类型别名]",
		Short: "推送 Upload RPC 任务 — 让基站上传文件到指定 URL",
		Long: `创建 TR-069 Upload RPC 任务（同时写入 Redis 队列和 PostgreSQL）。
设备收到后会将指定类型的文件上传到给定 URL。

文件类型别名 (TR-069 标准):
  config       (1)   配置文件          → "1 Vendor Configuration File"
  log          (2)   运行日志          → "2 Vendor Log File"

文件类型别名 (运营商/厂商扩展):
  log-ext            日志文件(扩展)    → "4 Vendor Log File"
  running-log        运行日志          → "2 Vendor Log File" (同 log)
  security-log       安全日志          → "2 Vendor Security Log"
  fault-log          故障日志          → "2 Vendor Fault Log"
  pm           (4)   性能管理文件      → "4 Vendor PM File"
  mr           (5)   测量报告          → "5 Vendor MR File"
  pcap         (9)   抓包文件          → "9 Vendor PCAP"
  oui-config   (10)  OUI 配置文件      → "10 <OUI> Configuration File" (需 --oui)
  datamodel    (11)  数据模型文件      → "11 OUI Parameter Model"
  ssl-cert           TR069 SSL 证书    → "Tr069 Ssl Cert File"

也可直接传入完整的 FileType 字符串（含空格），如:
  rpctool upload --file-type "1 Vendor Configuration File" --sn DEVICE001 --url ...`,
		Example: `  # 让设备上传运行日志
  rpctool upload log --sn DEVICE001 --url http://acs:7547/upload

  # 让设备上传配置文件
  rpctool upload config --sn DEVICE001 --url http://acs:7547/upload

  # 采集 PM 性能数据
  rpctool upload pm --sn DEVICE001 --url http://acs:7547/upload

  # 采集 MR 测量报告
  rpctool upload mr --sn DEVICE001 --url http://acs:7547/upload

  # 设备抓包
  rpctool upload pcap --sn DEVICE001 --url http://acs:7547/upload

  # 上传 OUI 配置文件（导出完整配置 XML）
  rpctool upload oui-config --sn DEVICE001 --url http://acs:7547/upload --oui 48BF74

  # 上传安全日志
  rpctool upload security-log --sn DEVICE001 --url http://acs:7547/upload

  # 上传故障日志
  rpctool upload fault-log --sn DEVICE001 --url http://acs:7547/upload

  # 上传 SSL 证书
  rpctool upload ssl-cert --sn DEVICE001 --url http://acs:7547/upload

  # 上传数据模型文件
  rpctool upload datamodel --sn DEVICE001 --url http://acs:7547/upload

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

			var resolvedFT string
			resolvedFT, err := resolveUploadFileType(ft)
			if err != nil {
				// Special handling for OUI config
				if strings.Contains(err.Error(), "__OUI_CONFIG__") {
					resolvedFT = resolveOUIConfigFileType(oui)
				} else {
					return err
				}
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

	cmd.Flags().StringVarP(&fileType, "file-type", "t", "", "文件类型 (别名、数字代码或完整 FileType 字符串)")
	cmd.Flags().StringVarP(&url, "url", "u", "", "文件上传目标 URL (必填)")
	cmd.Flags().StringVar(&username, "username", "", "HTTP 认证用户名")
	cmd.Flags().StringVar(&password, "password", "", "HTTP 认证密码")
	cmd.Flags().IntVar(&delay, "delay", 0, "延迟执行秒数")
	cmd.Flags().StringVar(&oui, "oui", defaultOUI, "OUI 标识 (仅 oui-config 类型使用)")

	return cmd
}

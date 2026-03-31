package main

import (
	"context"

	"github.com/spf13/cobra"
)

func newRebootCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "reboot",
		Short:   "推送 Reboot RPC 任务 — 重启设备",
		Example: `  rpctool reboot --sn DEVICE001`,
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			return createAndPrint(context.Background(), deviceSN, "Reboot", nil, "Reboot device")
		},
	}
}

func newResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "推送 FactoryReset RPC 任务 — 恢复出厂设置",
		Long:  "创建 TR-069 FactoryReset RPC 任务。\n警告：此操作会清除设备所有配置！",
		Example: `  rpctool reset --sn DEVICE001`,
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			return createAndPrint(context.Background(), deviceSN, "FactoryReset", nil, "Factory reset device")
		},
	}
}

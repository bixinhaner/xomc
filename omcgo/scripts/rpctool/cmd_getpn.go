package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newGetPNCmd() *cobra.Command {
	var (
		path      string
		nextLevel bool
	)

	cmd := &cobra.Command{
		Use:   "getpn",
		Short: "推送 GetParameterNames RPC 任务 — 获取设备参数树结构",
		Long: `创建 TR-069 GetParameterNames RPC 任务。
设备收到后会返回参数树中匹配路径的参数名列表。

--next-level=true  只返回直接子节点
--next-level=false 返回所有子孙节点`,
		Example: `  rpctool getpn --sn DEVICE001 --path Device.
  rpctool getpn --sn DEVICE001 --path Device. --next-level=false
  rpctool getpn --sn DEVICE001 --path Device.ManagementServer.`,
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]interface{}{
				"path":       path,
				"next_level": nextLevel,
			}
			return createAndPrint(context.Background(), deviceSN, "GetParameterNames", params,
				fmt.Sprintf("GetParameterNames: %s (next_level=%v)", path, nextLevel))
		},
	}

	cmd.Flags().StringVar(&path, "path", "Device.", "参数路径 (以 '.' 结尾表示对象)")
	cmd.Flags().BoolVarP(&nextLevel, "next-level", "n", true, "是否只返回下一级")

	return cmd
}

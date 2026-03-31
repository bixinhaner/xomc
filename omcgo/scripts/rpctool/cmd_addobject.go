package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newAddObjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "addobject <对象路径>",
		Short: "推送 AddObject RPC 任务 — 创建对象实例",
		Example: `  rpctool addobject --sn DEVICE001 "Device.Services.FAPService."`,
		Args:    cobra.ExactArgs(1),
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]interface{}{
				"object_name": args[0],
			}
			return createAndPrint(context.Background(), deviceSN, "AddObject", params,
				fmt.Sprintf("AddObject: %s", args[0]))
		},
	}
}

func newDeleteObjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deleteobject <对象路径>",
		Short: "推送 DeleteObject RPC 任务 — 删除对象实例",
		Example: `  rpctool deleteobject --sn DEVICE001 "Device.Services.FAPService.1."`,
		Args:    cobra.ExactArgs(1),
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return fmt.Errorf("对象路径不能为空")
			}
			params := map[string]interface{}{
				"object_name": args[0],
			}
			return createAndPrint(context.Background(), deviceSN, "DeleteObject", params,
				fmt.Sprintf("DeleteObject: %s", args[0]))
		},
	}
}

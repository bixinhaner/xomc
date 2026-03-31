package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newGetPVCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "getpv <参数路径> [参数路径...]",
		Short: "推送 GetParameterValues RPC 任务 — 读取设备参数值",
		Long: `创建 TR-069 GetParameterValues RPC 任务。
设备收到后会返回指定路径的参数值。

路径可以是:
  - 完整参数路径: Device.DeviceInfo.Manufacturer
  - 部分路径(获取子树): Device.DeviceInfo.
  - 根路径(获取全部): Device.`,
		Example: `  rpctool getpv --sn DEVICE001 Device.DeviceInfo.
  rpctool getpv --sn DEVICE001 Device.DeviceInfo.Manufacturer Device.DeviceInfo.SoftwareVersion`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]interface{}{
				"names": args,
			}
			return createAndPrint(context.Background(), deviceSN, "GetParameterValues", params,
				fmt.Sprintf("GetParameterValues: %s", strings.Join(args, ", ")))
		},
	}
}

func newSetPVCmd() *cobra.Command {
	var paramFlags []string

	cmd := &cobra.Command{
		Use:   "setpv",
		Short: "推送 SetParameterValues RPC 任务 — 设置设备参数值",
		Long: `创建 TR-069 SetParameterValues RPC 任务。
设备收到后会设置指定参数的值。

参数格式: name=value 或 name=value:type
  - 默认类型为 xsd:string
  - 常用类型: xsd:string, xsd:unsignedInt, xsd:boolean, xsd:int, xsd:dateTime`,
		Example: `  rpctool setpv --sn DEVICE001 -p "Device.ManagementServer.PeriodicInformInterval=300:xsd:unsignedInt"
  rpctool setpv --sn DEVICE001 -p "Device.X_VENDOR.CustomField=hello"`,
		PreRunE: requireSNAndInfra,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(paramFlags) == 0 {
				return fmt.Errorf("必须指定至少一个 -p 参数，格式: name=value 或 name=value:type")
			}

			values, err := parseSetPVParams(paramFlags)
			if err != nil {
				return err
			}

			params := map[string]interface{}{
				"values": values,
			}
			return createAndPrint(context.Background(), deviceSN, "SetParameterValues", params,
				fmt.Sprintf("SetParameterValues: %d params", len(values)))
		},
	}

	cmd.Flags().StringArrayVarP(&paramFlags, "param", "p", nil, "参数 (格式: name=value 或 name=value:type，可重复)")

	return cmd
}

// parseSetPVParams parses "name=value" or "name=value:xsd:type" strings.
func parseSetPVParams(params []string) ([]map[string]string, error) {
	var result []map[string]string

	for _, p := range params {
		eqIdx := strings.IndexByte(p, '=')
		if eqIdx < 0 {
			return nil, fmt.Errorf("参数格式错误: %q (期望 name=value 或 name=value:type)", p)
		}

		name := p[:eqIdx]
		rest := p[eqIdx+1:]

		value := rest
		typ := "xsd:string"

		// Look for :xsd: or :xsi: type suffix from the right.
		for _, prefix := range []string{":xsd:", ":xsi:"} {
			idx := strings.LastIndex(rest, prefix)
			if idx >= 0 {
				value = rest[:idx]
				typ = rest[idx+1:] // skip leading ':'
				break
			}
		}

		if name == "" {
			return nil, fmt.Errorf("参数名不能为空: %q", p)
		}

		result = append(result, map[string]string{
			"name":  name,
			"value": value,
			"type":  typ,
		})
	}

	return result, nil
}

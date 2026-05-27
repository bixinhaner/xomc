package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// newSweepPathsCmd 构造 `omcctl device sweep-paths <SN>` 子命令（T-0179）。
//
// 设计：thin HTTP client，所有业务逻辑在 app server 的
// internal/devsweep 包里。CLI 负责 flag 解析 + 输出格式化 + 退出码映射。
//
// Exit codes（spec）：
//
//	0 OK（含 dry-run 全 supported 或 --apply 成功）
//	1 解析失败 / 配置错（用户错）
//	2 设备离线 / RPC 全失败
//	3 安全门 abort
//	4 DB UPDATE 失败
func newSweepPathsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sweep-paths <SN>",
		Short: "Probe a device via GPV and mark unsupported standard paths",
		Long: `Submit GPV RPCs against the device's standard path catalog (paramModel),
classify each path as supported / unsupported (CWMP 9005) / unknown, then
(with --apply) flip param_mappings.is_supported=false for the unsupported set.

This is a paramModel-level write: all devices sharing the same paramModel
are affected. Safety gates require --confirm-paramodel-wide when the
paramModel touches > 10 active devices, and --force when the unsupported
ratio exceeds 50%%.

{i} occurrences in catalog paths are replaced with instance 0 before being
sent to the CPE. Devices with no instance 0 may falsely return 9005 — prefer
running --prefix on subtrees where instance 0 exists.

Dry-run is the default; pass --apply to mutate the database.`,
		Args: cobra.ExactArgs(1),
		RunE: runSweepPaths,
	}
	cmd.Flags().Bool("apply", false, "Actually write param_mappings.is_supported=false (default dry-run)")
	cmd.Flags().String("prefix", "", "Filter candidate paths by this standardPath prefix")
	cmd.Flags().Int("batch-size", 1, "GPV batch size per task (1 recommended for accurate per-path attribution)")
	cmd.Flags().Duration("rpc-timeout", 30*time.Second, "Per-task wait timeout for terminal state")
	cmd.Flags().Float64("rpc-rate", 5.0, "Global rate limit for GPV task enqueue (tasks/second)")
	cmd.Flags().String("operator", defaultOperator(), "Operator identity, recorded in app log")
	cmd.Flags().Bool("confirm-paramodel-wide", false, "Required when paramModel affects > 10 active devices")
	cmd.Flags().Bool("force", false, "Skip the 50%% unsupported ratio safety gate")
	cmd.Flags().Bool("json", false, "Emit machine-readable JSON instead of text")
	cmd.Flags().Bool("verbose", false, "Include per-batch / per-path detail")
	return cmd
}

// defaultOperator picks an identity for the audit field: $USER or "omcctl".
func defaultOperator() string {
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return u
	}
	return "omcctl"
}

// sweepPathsRequest 与 internal/devsweep/handler.go sweepRequest 形态一致。
type sweepPathsRequest struct {
	Apply                 bool    `json:"apply"`
	Prefix                string  `json:"prefix,omitempty"`
	BatchSize             int     `json:"batch_size,omitempty"`
	RPCTimeoutSeconds     int     `json:"rpc_timeout_seconds,omitempty"`
	RPCRate               float64 `json:"rpc_rate,omitempty"`
	Operator              string  `json:"operator,omitempty"`
	ConfirmParamModelWide bool    `json:"confirm_param_model_wide,omitempty"`
	Force                 bool    `json:"force,omitempty"`
	Verbose               bool    `json:"verbose,omitempty"`
}

// sweepPathsResponse 解析 /api/v1/devices/:sn/sweep-paths 的标准信封 data 节点。
type sweepPathsResponse struct {
	DeviceSN                  string   `json:"device_sn"`
	ProductClass              string   `json:"product_class"`
	ProductID                 string   `json:"product_id"`
	ProductName               string   `json:"product_name"`
	ParamModelID              string   `json:"param_model_id"`
	ParamModelName            string   `json:"param_model_name"`
	FirmwareVersion           string   `json:"firmware_version"`
	IsOnline                  bool     `json:"is_online"`
	CandidateCount            int      `json:"candidate_count"`
	SupportedCount            int      `json:"supported_count"`
	UnsupportedCount          int      `json:"unsupported_count"`
	UnknownCount              int      `json:"unknown_count"`
	MarkedCount               int      `json:"marked_count"`
	UnsupportedPaths          []string `json:"unsupported_paths"`
	UnknownPaths              []string `json:"unknown_paths"`
	DryRun                    bool     `json:"dry_run"`
	StartedAt                 string   `json:"started_at"`
	FinishedAt                string   `json:"finished_at"`
	DurationMS                int64    `json:"duration_ms"`
	ParamModelDevicesAffected int      `json:"param_model_devices_affected"`
	Aborted                   bool     `json:"aborted"`
	ErrorCode                 string   `json:"error_code,omitempty"`
}

// runSweepPaths 是 cobra Cmd.RunE — 解析 flag、POST 调端点、渲染输出、设置 exit code。
func runSweepPaths(cmd *cobra.Command, args []string) error {
	sn := strings.TrimSpace(args[0])
	if sn == "" {
		return fmt.Errorf("device SN required")
	}

	apply, _ := cmd.Flags().GetBool("apply")
	prefix, _ := cmd.Flags().GetString("prefix")
	batchSize, _ := cmd.Flags().GetInt("batch-size")
	rpcTimeout, _ := cmd.Flags().GetDuration("rpc-timeout")
	rpcRate, _ := cmd.Flags().GetFloat64("rpc-rate")
	operator, _ := cmd.Flags().GetString("operator")
	confirmWide, _ := cmd.Flags().GetBool("confirm-paramodel-wide")
	force, _ := cmd.Flags().GetBool("force")
	wantJSON, _ := cmd.Flags().GetBool("json")
	verbose, _ := cmd.Flags().GetBool("verbose")

	req := sweepPathsRequest{
		Apply:                 apply,
		Prefix:                prefix,
		BatchSize:             batchSize,
		RPCTimeoutSeconds:     int(rpcTimeout.Seconds()),
		RPCRate:               rpcRate,
		Operator:              operator,
		ConfirmParamModelWide: confirmWide,
		Force:                 force,
		Verbose:               verbose,
	}

	client := getClient()
	resp, err := client.Post("/api/v1/devices/"+sn+"/sweep-paths", req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[sweep] request failed:", err)
		os.Exit(1)
	}

	// 解析 envelope
	raw, _ := json.Marshal(resp["data"])
	var result sweepPathsResponse
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &result); err != nil {
			fmt.Fprintln(os.Stderr, "[sweep] parse response:", err)
			os.Exit(1)
		}
	}

	if wantJSON {
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
	} else {
		renderText(result, verbose)
		// envelope msg 在 abort 路径里有真正错因
		if result.Aborted {
			if msg, _ := resp["msg"].(string); msg != "" && msg != "ok" {
				fmt.Fprintln(os.Stderr, "[sweep] reason:", msg)
			}
		}
	}

	os.Exit(exitCodeFor(result))
	return nil
}

// exitCodeFor 把 Result 映射到 spec 定义的 exit code。
func exitCodeFor(r sweepPathsResponse) int {
	if r.Aborted {
		switch r.ErrorCode {
		case "device_not_found", "product_class_empty", "orphan_product_class",
			"param_model_unset", "no_mappings":
			return 1
		case "device_offline":
			return 2
		case "unsupported_rate_high", "param_model_wide_unconfirmed":
			return 3
		}
		return 1
	}
	// 未 abort 但 unknown=candidate（全失败）也算 RPC 全失败
	if r.CandidateCount > 0 && r.UnknownCount == r.CandidateCount {
		return 2
	}
	return 0
}

// renderText 把 Result 渲染成 spec 示例样式。
func renderText(r sweepPathsResponse, verbose bool) {
	fmt.Printf("[sweep] device:        %s\n", r.DeviceSN)
	fmt.Printf("[sweep] product_class: %s\n", r.ProductClass)
	if r.ProductID != "" {
		fmt.Printf("[sweep] product:       %s (%s)\n", r.ProductName, r.ProductID)
	}
	if r.ParamModelID != "" {
		fmt.Printf("[sweep] param_model:   %s (%s)\n", r.ParamModelName, r.ParamModelID)
	}
	fmt.Printf("[sweep] firmware:      %s\n", r.FirmwareVersion)
	fmt.Printf("[sweep] online:        %v\n", r.IsOnline)

	if r.Aborted {
		fmt.Printf("\n[sweep] ABORTED: %s\n", r.ErrorCode)
		return
	}

	fmt.Println()
	fmt.Printf("[sweep] candidate paths: %d\n", r.CandidateCount)
	fmt.Println("[sweep] result:")
	fmt.Printf("  supported:   %d\n", r.SupportedCount)
	fmt.Printf("  unsupported: %d\n", r.UnsupportedCount)
	fmt.Printf("  unknown:     %d\n", r.UnknownCount)
	fmt.Printf("[sweep] paramModel devices affected: %d\n", r.ParamModelDevicesAffected)
	fmt.Printf("[sweep] elapsed: %dms\n", r.DurationMS)

	if len(r.UnsupportedPaths) > 0 {
		fmt.Println("\n[sweep] unsupported paths:")
		for _, p := range r.UnsupportedPaths {
			fmt.Printf("  - %s\n", p)
		}
	}
	if verbose && len(r.UnknownPaths) > 0 {
		fmt.Println("\n[sweep] unknown paths (skipped — fault code != 9005 or timeout):")
		for _, p := range r.UnknownPaths {
			fmt.Printf("  - %s\n", p)
		}
	}

	if r.DryRun {
		fmt.Println("\n[sweep] DRY-RUN: no DB change. Pass --apply to write param_mappings.")
	} else {
		fmt.Printf("\n[sweep] APPLIED: marked %d row(s) in param_mappings.\n", r.MarkedCount)
	}
}

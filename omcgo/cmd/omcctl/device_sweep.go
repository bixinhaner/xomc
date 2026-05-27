package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/omcgo/omcgo/internal/devsweep"
)

// newSweepPathsCmd 构造 `omcctl device sweep-paths <SN>` 子命令(T-0183 thick mode)。
//
// 设计:omcctl 进程内直连 PG + Redis,直接调 devsweep.Service.Run + Applier.Apply。
// 不再走 app HTTP 端点 — 详见 T-0183 设计 / omcgo/CLAUDE.md §5.6。
//
// 数据流:
//
//	worker config (DSN / Redis) → pgxpool + redis client
//	→ device + product + param-model Registry → Service.Run (探测)
//	→ Applier.Apply (--apply: 写 XML + UPDATE param_mappings + UPDATE
//	  discovered_param_mappings + 清 Redis L2 + bump cache_version)
//
// Exit codes (与历史一致):
//
//	0 OK
//	1 配置错 / 设备不存在
//	2 设备离线 / RPC 全失败
//	3 安全门 abort
//	4 apply 中途出错(XML 或 DB 写失败,部分结果可能已落)
func newSweepPathsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sweep-paths <SN>",
		Short: "Probe a device via GPV, write XML supported=\"false\" + sync DB",
		Long: `Submit GPV RPCs against the device's paramModel path catalog,
classify each path as supported / unsupported (CWMP 9005) / unknown.

With --apply:
  1. Write supported="false" into the paramModel XML file (preserves formatting)
  2. UPDATE param_mappings.is_supported = false for matching default rows
  3. UPDATE discovered_param_mappings.is_supported = false for matching rows
  4. Invalidate Redis L2 (parammodel:default:*, parammodel:discovered:*) +
     bump parammodel:cache_version so app instances reload

Default is dry-run (no DB / XML change). Pass --apply to mutate.

Safety gates (paramModel-wide write):
  --confirm-paramodel-wide  required when paramModel > 10 active devices
  --force                   required when unsupported ratio > 50%

Process model: omcctl runs the entire sweep INSIDE its process — no app HTTP
roundtrip. Requires DB/Redis access (worker container has both).`,
		Args: cobra.ExactArgs(1),
		RunE: runSweepPaths,
	}
	cmd.Flags().Bool("apply", false, "Write XML + DB + invalidate cache (default dry-run)")
	cmd.Flags().String("prefix", "", "Filter candidate paths by this standardPath prefix")
	cmd.Flags().Int("batch-size", 16, "GPV batch size per task")
	cmd.Flags().Duration("rpc-timeout", 30*time.Second, "Per-task wait timeout for terminal state")
	cmd.Flags().Float64("rpc-rate", 5.0, "Global rate limit for GPV task enqueue (tasks/second)")
	cmd.Flags().String("operator", defaultOperator(), "Operator identity, recorded in app log")
	cmd.Flags().Bool("confirm-paramodel-wide", false, "Required when paramModel affects > 10 active devices")
	cmd.Flags().Bool("force", false, "Skip the 50%% unsupported ratio safety gate")
	cmd.Flags().Bool("json", false, "Emit machine-readable JSON")
	cmd.Flags().Bool("verbose", false, "Include per-batch / per-path detail")
	return cmd
}

func defaultOperator() string {
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return u
	}
	return "omcctl"
}

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

	ctx := context.Background()

	deps, err := loadSweepDeps(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[sweep] init deps failed:", err)
		os.Exit(1)
	}
	defer deps.Close()

	svc, applier, err := deps.buildSweepService(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[sweep] build sweep service failed:", err)
		os.Exit(1)
	}

	opts := devsweep.Options{
		DeviceSN:              sn,
		Prefix:                prefix,
		BatchSize:             batchSize,
		RPCTimeout:            rpcTimeout,
		RPCRate:               rpcRate,
		Apply:                 apply,
		Force:                 force,
		ConfirmParamModelWide: confirmWide,
		Operator:              operator,
		Safety:                devsweep.DefaultSafetyConfig(),
	}

	result, err := svc.Run(ctx, opts)
	if err != nil && (result == nil || !result.Aborted) {
		fmt.Fprintln(os.Stderr, "[sweep] probe error:", err)
		os.Exit(1)
	}

	var applyOutcome *devsweep.ApplyOutcome
	if apply && result != nil && !result.Aborted && result.UnsupportedCount > 0 {
		applyOutcome, err = applier.Apply(ctx, result)
		if err != nil {
			fmt.Fprintln(os.Stderr, "[sweep] apply error:", err)
			renderResult(result, applyOutcome, verbose, wantJSON)
			os.Exit(4)
		}
	}

	renderResult(result, applyOutcome, verbose, wantJSON)
	os.Exit(exitCodeFor(result))
	return nil
}

func exitCodeFor(r *devsweep.Result) int {
	if r == nil {
		return 1
	}
	if r.Aborted {
		switch r.ErrorCode {
		case devsweep.ErrCodeDeviceOffline:
			return 2
		case devsweep.ErrCodeUnsupportedRateHigh, devsweep.ErrCodeParamModelWideUnconf:
			return 3
		}
		return 1
	}
	if r.CandidateCount > 0 && r.UnknownCount == r.CandidateCount {
		return 2
	}
	return 0
}

func renderResult(r *devsweep.Result, apply *devsweep.ApplyOutcome, verbose, wantJSON bool) {
	if r == nil {
		return
	}
	if wantJSON {
		out, _ := json.MarshalIndent(map[string]any{
			"result": r,
			"apply":  apply,
		}, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Printf("[sweep] device:        %s\n", r.DeviceSN)
	fmt.Printf("[sweep] product_class: %s\n", r.ProductClass)
	if r.ProductID.String() != "00000000-0000-0000-0000-000000000000" {
		fmt.Printf("[sweep] product:       %s (%s)\n", r.ProductName, r.ProductID)
	}
	if r.ParamModelID.String() != "00000000-0000-0000-0000-000000000000" {
		name := r.ParamModelName
		if name == "" {
			name = "(unknown)"
		}
		fmt.Printf("[sweep] param_model:   %s (%s)\n", name, r.ParamModelID)
	}
	fmt.Printf("[sweep] firmware:      %s\n", r.FirmwareVersion)
	fmt.Printf("[sweep] online:        %v\n", r.IsOnline)

	if r.Aborted {
		fmt.Printf("\n[sweep] ABORTED: %s\n", r.ErrorCode)
		return
	}

	fmt.Println()
	fmt.Printf("[sweep] candidate paths: %d\n", r.CandidateCount)
	fmt.Println("[sweep] probe result:")
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
		fmt.Println("\n[sweep] unknown paths:")
		for _, p := range r.UnknownPaths {
			fmt.Printf("  - %s\n", p)
		}
	}

	if r.DryRun {
		fmt.Println("\n[sweep] DRY-RUN: no XML / DB change. Pass --apply to commit.")
		return
	}
	if apply == nil {
		fmt.Println("\n[sweep] APPLY: no unsupported paths to write.")
		return
	}

	fmt.Println("\n[sweep] APPLY:")
	if apply.XML != nil {
		fmt.Printf("  XML file:            %s\n", apply.XML.FilePath)
		fmt.Printf("  XML lines modified:  %d\n", apply.XML.LinesModified)
		if len(apply.XML.PathsApplied) > 0 {
			fmt.Printf("  XML applied:         %d\n", len(apply.XML.PathsApplied))
		}
		if len(apply.XML.PathsAlreadyMarked) > 0 {
			fmt.Printf("  XML already marked:  %d\n", len(apply.XML.PathsAlreadyMarked))
		}
		if len(apply.XML.PathsNotFound) > 0 {
			fmt.Printf("  XML not found:       %d (paths missing in XML; only DB-write applied)\n",
				len(apply.XML.PathsNotFound))
			if verbose {
				for _, p := range apply.XML.PathsNotFound {
					fmt.Printf("    · %s\n", p)
				}
			}
		}
	}
	fmt.Printf("  DB (default):        %d rows\n", apply.DBUpdatedDefault)
	fmt.Printf("  DB (discovered):     %d rows\n", apply.DBUpdatedDiscovered)
	fmt.Printf("  Cache keys cleared:  %d\n", apply.CacheKeysCleared)
	fmt.Printf("  Cache version:       %d\n", apply.CacheVersionAfter)
}

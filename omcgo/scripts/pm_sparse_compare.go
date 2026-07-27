//go:build ignore

// Command pm_sparse_compare applies the offline sparse-storage acceptance gate
// to normalized logical PM JSON or CSV exports.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	pmcompare "github.com/omcgo/omcgo/internal/pm/compare"
)

func main() {
	oldPath := flag.String("old", "", "legacy normalized logical rows (.json or .csv)")
	sparsePath := flag.String("sparse", "", "sparse normalized logical rows (.json or .csv)")
	oldBytes := flag.Int64("old-bytes", -1, "legacy table+index+TOAST physical bytes")
	sparseBytes := flag.Int64("sparse-bytes", -1, "sparse table+index+TOAST physical bytes")
	flag.Parse()

	provided := make(map[string]bool)
	flag.Visit(func(item *flag.Flag) {
		provided[item.Name] = true
	})
	if *oldPath == "" || *sparsePath == "" || !provided["old-bytes"] || !provided["sparse-bytes"] {
		fmt.Fprintln(os.Stderr, "--old, --sparse, --old-bytes, and --sparse-bytes are required")
		flag.Usage()
		os.Exit(2)
	}
	oldRows, err := pmcompare.ReadRows(*oldPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read old export: %v\n", err)
		os.Exit(2)
	}
	sparseRows, err := pmcompare.ReadRows(*sparsePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read sparse export: %v\n", err)
		os.Exit(2)
	}

	result := pmcompare.Compare(oldRows, sparseRows, *oldBytes, *sparseBytes)
	fmt.Printf("logical_equal=%t\nold_bytes=%d\nsparse_bytes=%d\nsparse_ratio=%s\nphysical_valid=%t\nphysical_reason=%s\nwithin_size_target=%t\naccepted=%t\n",
		result.LogicalEqual, result.OldBytes, result.SparseBytes, result.SparseRatioText(),
		result.PhysicalValid, result.PhysicalReason, result.WithinSizeTarget, result.Accepted)
	if len(result.Mismatches) > 0 {
		encoded, err := json.MarshalIndent(result.Mismatches, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "encode mismatches: %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("mismatches=%s\n", encoded)
	}
	if !result.Accepted {
		os.Exit(1)
	}
}

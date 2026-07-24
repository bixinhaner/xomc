//go:build ignore

// Command pm_sparse_compare applies the offline sparse-storage acceptance gate
// to normalized logical PM JSON or CSV exports.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"

	pmcompare "github.com/omcgo/omcgo/internal/pm/compare"
)

func main() {
	oldPath := flag.String("old", "", "legacy normalized logical rows (.json or .csv)")
	sparsePath := flag.String("sparse", "", "sparse normalized logical rows (.json or .csv)")
	oldBytes := flag.Int64("old-bytes", -1, "legacy table+index+TOAST physical bytes")
	sparseBytes := flag.Int64("sparse-bytes", -1, "sparse table+index+TOAST physical bytes")
	flag.Parse()

	if *oldPath == "" || *sparsePath == "" || *oldBytes < 0 || *sparseBytes < 0 {
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
	ratio := fmt.Sprintf("%.6f", result.SparseRatio)
	if math.IsInf(result.SparseRatio, 1) {
		ratio = "+Inf"
	}
	fmt.Printf("logical_equal=%t\nold_bytes=%d\nsparse_bytes=%d\nsparse_ratio=%s\nwithin_size_target=%t\naccepted=%t\n",
		result.LogicalEqual, result.OldBytes, result.SparseBytes, ratio, result.WithinSizeTarget, result.Accepted)
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

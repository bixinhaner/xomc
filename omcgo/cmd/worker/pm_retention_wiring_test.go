package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_StartPMExportOnly_WiresPMRetentionCleanupExactlyOnce(t *testing.T) {
	t.Helper()

	_, testFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(filepath.Dir(testFile), "pm_streaming.go"), nil, 0)
	require.NoError(t, err)

	var calls int
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "startPMExportOnly" {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if ok && ident.Name == "startPMRetentionCleanup" {
				calls++
			}
			return true
		})
	}

	require.Equal(t, 1, calls,
		"worker shared async-job startup must wire PM retention exactly once")
}

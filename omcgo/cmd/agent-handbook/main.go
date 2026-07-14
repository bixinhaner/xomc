package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/omcgo/omcgo/internal/agentruntime/handbookgen"
)

func main() {
	var routesPath string
	var openAPIPath string
	var sourceRoot string
	var outputDir string
	var packagePath string
	var checkOnly bool
	flag.StringVar(&routesPath, "routes", "", "path to the live OMC handbook route export JSON")
	flag.StringVar(&openAPIPath, "openapi", "api/openapi/openapi.yaml", "path to the OMC OpenAPI document")
	flag.StringVar(&sourceRoot, "source-root", ".", "path to the omcgo source root")
	flag.StringVar(&outputDir, "output", "data/agent-skill/omc-operations", "path to the omc-operations Skill")
	flag.StringVar(&packagePath, "package", "internal/agentruntime/handbookasset/omc-api-handbook.tar.gz", "path to the embedded handbook data package")
	flag.BoolVar(&checkOnly, "check", false, "validate existing generated files without rewriting them")
	flag.Parse()

	if routesPath == "" {
		fail("-routes is required")
	}
	raw, err := os.ReadFile(routesPath)
	if err != nil {
		fail(fmt.Sprintf("read route export: %v", err))
	}
	routes, err := handbookgen.DecodeRouteExport(raw)
	if err != nil {
		fail(err.Error())
	}
	if checkOnly {
		if err := handbookgen.Check(outputDir, routes); err != nil {
			fail(err.Error())
		}
		if err := handbookgen.CheckPackage(outputDir, packagePath); err != nil {
			fail(err.Error())
		}
		fmt.Printf("handbook valid: %d operations, catalog %s\n", routes.TotalRoutes, routes.CatalogVersion)
		return
	}
	manifest, err := handbookgen.Generate(handbookgen.Options{
		Routes:      routes,
		OpenAPIPath: openAPIPath,
		SourceRoot:  sourceRoot,
		OutputDir:   outputDir,
	})
	if err != nil {
		fail(err.Error())
	}
	if err := handbookgen.WritePackage(outputDir, packagePath); err != nil {
		fail(err.Error())
	}
	fmt.Printf("handbook generated: %d operations, %d categories, catalog %s\n", manifest.TotalOperations, len(manifest.Categories), manifest.CatalogVersion)
}

func fail(message string) {
	_, _ = fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

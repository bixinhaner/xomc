package handbookgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	contractsDirectory = "contracts"
	contractsFile      = "handlers.json"
	contractsOpenAPI   = "openapi.yaml"
)

type contractAssets struct {
	SchemaVersion string                     `json:"schemaVersion"`
	Handlers      map[string]handlerContract `json:"handlers"`
	OpenAPI       openAPIDocument            `json:"-"`
}

type ContractOptions struct {
	SourceRoot  string
	OpenAPIPath string
	OutputDir   string
	PackagePath string
}

// SyncContracts refreshes source-derived API contracts and the deterministic
// package embedded by the app binary. Runtime route discovery remains the
// authority for which operations are published.
func SyncContracts(options ContractOptions) error {
	handlersRaw, openAPIRaw, err := generateContractFiles(options.SourceRoot, options.OpenAPIPath)
	if err != nil {
		return err
	}
	contractsDir := filepath.Join(options.OutputDir, handbookPackageRoot, contractsDirectory)
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		return fmt.Errorf("create handbook contracts directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(contractsDir, contractsFile), handlersRaw, 0o644); err != nil {
		return fmt.Errorf("write handler contracts: %w", err)
	}
	if err := os.WriteFile(filepath.Join(contractsDir, contractsOpenAPI), openAPIRaw, 0o644); err != nil {
		return fmt.Errorf("write OpenAPI contract source: %w", err)
	}
	if err := WritePackage(options.OutputDir, options.PackagePath); err != nil {
		return fmt.Errorf("write handbook package with contracts: %w", err)
	}
	return nil
}

// CheckContracts fails when committed contract assets do not match the current
// Go handlers and OpenAPI source, or when the embedded package is stale.
func CheckContracts(options ContractOptions) error {
	expectedHandlers, expectedOpenAPI, err := generateContractFiles(options.SourceRoot, options.OpenAPIPath)
	if err != nil {
		return err
	}
	contractsDir := filepath.Join(options.OutputDir, handbookPackageRoot, contractsDirectory)
	checks := []struct {
		name     string
		path     string
		expected []byte
	}{
		{name: "handler contracts", path: filepath.Join(contractsDir, contractsFile), expected: expectedHandlers},
		{name: "OpenAPI contract source", path: filepath.Join(contractsDir, contractsOpenAPI), expected: expectedOpenAPI},
	}
	for _, check := range checks {
		actual, readErr := os.ReadFile(check.path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", check.name, readErr)
		}
		if !bytes.Equal(actual, check.expected) {
			return fmt.Errorf("%s is stale; run make agent-handbook-generate", check.name)
		}
	}
	if err := CheckPackage(options.OutputDir, options.PackagePath); err != nil {
		return err
	}
	return nil
}

func generateContractFiles(sourceRoot, openAPIPath string) ([]byte, []byte, error) {
	analyzer, err := newGoSourceAnalyzer(sourceRoot)
	if err != nil {
		return nil, nil, err
	}
	assets := contractAssets{SchemaVersion: SchemaVersion, Handlers: analyzer.allHandlerContracts()}
	if len(assets.Handlers) == 0 {
		return nil, nil, fmt.Errorf("no Gin handler contracts found under %s", sourceRoot)
	}
	handlersRaw, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("encode handler contracts: %w", err)
	}
	handlersRaw = append(handlersRaw, '\n')
	openAPIRaw, err := os.ReadFile(openAPIPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read OpenAPI contract source: %w", err)
	}
	if _, err := loadOpenAPIRaw(openAPIRaw); err != nil {
		return nil, nil, err
	}
	return handlersRaw, openAPIRaw, nil
}

func loadContractAssets(files map[string][]byte) (contractAssets, error) {
	var assets contractAssets
	handlersPath := handbookPackageRoot + "/" + contractsDirectory + "/" + contractsFile
	handlersRaw := files[handlersPath]
	if len(handlersRaw) == 0 {
		return assets, nil
	}
	if err := json.Unmarshal(handlersRaw, &assets); err != nil {
		return contractAssets{}, fmt.Errorf("decode handbook handler contracts: %w", err)
	}
	if assets.SchemaVersion != SchemaVersion {
		return contractAssets{}, fmt.Errorf("handler contract schema %q is not supported", assets.SchemaVersion)
	}
	openAPIPath := handbookPackageRoot + "/" + contractsDirectory + "/" + contractsOpenAPI
	openAPI, err := loadOpenAPIRaw(files[openAPIPath])
	if err != nil {
		return contractAssets{}, err
	}
	assets.OpenAPI = openAPI
	return assets, nil
}

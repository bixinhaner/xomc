package handbookgen

import (
	"encoding/json"
	"fmt"
)

const SchemaVersion = "1.0"

type Parameter struct {
	Name        string         `json:"name" yaml:"name"`
	In          string         `json:"in" yaml:"in"`
	Required    bool           `json:"required" yaml:"required"`
	Type        string         `json:"type,omitempty" yaml:"type,omitempty"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	Default     any            `json:"default,omitempty" yaml:"default,omitempty"`
	Enum        []any          `json:"enum,omitempty" yaml:"enum,omitempty"`
	Schema      map[string]any `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type Route struct {
	OperationID string      `json:"operationId"`
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	Handler     string      `json:"handler"`
	Title       string      `json:"title"`
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Risk        string      `json:"risk"`
	Tags        []string    `json:"tags"`
	PathParams  []Parameter `json:"pathParams,omitempty"`
}

type RouteExport struct {
	SchemaVersion  string  `json:"schemaVersion"`
	CatalogVersion string  `json:"catalogVersion"`
	TotalRoutes    int     `json:"totalRoutes"`
	Routes         []Route `json:"routes"`
}

type Options struct {
	Routes      RouteExport
	OpenAPIPath string
	SourceRoot  string
	OutputDir   string
}

type CategoryManifest struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Count int    `json:"count"`
	File  string `json:"file"`
}

type Manifest struct {
	SchemaVersion   string             `json:"schemaVersion"`
	CatalogVersion  string             `json:"catalogVersion"`
	TotalOperations int                `json:"totalOperations"`
	SearchIndex     string             `json:"searchIndex"`
	Categories      []CategoryManifest `json:"categories"`
}

type OperationSummary struct {
	OperationID string   `json:"operationId"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Intents     []string `json:"intents"`
	SearchTerms []string `json:"searchTerms,omitempty"`
	Risk        string   `json:"risk"`
	Document    string   `json:"document"`
}

type RelatedOperation struct {
	OperationID string `json:"operationId"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Relation    string `json:"relation"`
}

type CategoryIndex struct {
	SchemaVersion string             `json:"schemaVersion"`
	Category      string             `json:"category"`
	Title         string             `json:"title"`
	Operations    []OperationSummary `json:"operations"`
}

type OperationDocument struct {
	SchemaVersion        string             `json:"schemaVersion"`
	OperationID          string             `json:"operationId"`
	Method               string             `json:"method"`
	Path                 string             `json:"path"`
	Handler              string             `json:"handler,omitempty"`
	Category             string             `json:"category"`
	Title                string             `json:"title"`
	Summary              string             `json:"summary"`
	Description          string             `json:"description"`
	Intents              []string           `json:"intents"`
	Tags                 []string           `json:"tags"`
	Risk                 string             `json:"risk"`
	ConfirmationRequired bool               `json:"confirmationRequired"`
	Idempotent           bool               `json:"idempotent"`
	SideEffects          string             `json:"sideEffects"`
	PathParams           []Parameter        `json:"pathParams"`
	QueryParams          []Parameter        `json:"queryParams"`
	FormParams           []Parameter        `json:"formParams"`
	RequestBody          map[string]any     `json:"requestBody,omitempty"`
	Responses            map[string]any     `json:"responses,omitempty"`
	ReferencedSchemas    map[string]any     `json:"referencedSchemas,omitempty"`
	ContractCoverage     map[string]string  `json:"contractCoverage"`
	EmptyResult          string             `json:"emptyResult,omitempty"`
	RelatedOperations    []RelatedOperation `json:"relatedOperations,omitempty"`
	Sources              []string           `json:"sources"`
}

func DecodeRouteExport(raw []byte) (RouteExport, error) {
	var envelope struct {
		Ret  *int            `json:"ret"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return RouteExport{}, fmt.Errorf("decode route export JSON: %w", err)
	}
	payload := raw
	if envelope.Ret != nil {
		if *envelope.Ret != 1 || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
			return RouteExport{}, fmt.Errorf("route export response is not successful: %s", envelope.Msg)
		}
		payload = envelope.Data
	}
	var exported RouteExport
	if err := json.Unmarshal(payload, &exported); err != nil {
		return RouteExport{}, fmt.Errorf("decode route export data: %w", err)
	}
	return exported, nil
}

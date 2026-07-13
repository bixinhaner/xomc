package handbookgen

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type openAPIDocument struct {
	root       map[string]any
	operations map[string]map[string]any
}

type openAPIOperation struct {
	Found             bool
	Title             string
	Summary           string
	Description       string
	Tags              []string
	PathParams        []Parameter
	QueryParams       []Parameter
	FormParams        []Parameter
	RequestBody       map[string]any
	Responses         map[string]any
	ReferencedSchemas map[string]any
}

var openAPIPathParameterPattern = regexp.MustCompile(`\{([^}/]+)\}`)

func loadOpenAPI(path string) (openAPIDocument, error) {
	document := openAPIDocument{operations: map[string]map[string]any{}}
	if strings.TrimSpace(path) == "" {
		return document, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return document, fmt.Errorf("read OpenAPI document: %w", err)
	}
	if err := yaml.Unmarshal(raw, &document.root); err != nil {
		return document, fmt.Errorf("decode OpenAPI document: %w", err)
	}
	paths, _ := document.root["paths"].(map[string]any)
	for rawPath, rawPathValue := range paths {
		pathItem, _ := rawPathValue.(map[string]any)
		normalized := normalizeOpenAPIPath(rawPath)
		for _, method := range []string{"get", "head", "options", "post", "put", "patch", "delete"} {
			operation, ok := pathItem[method].(map[string]any)
			if !ok {
				continue
			}
			copy := cloneMap(operation)
			if pathParameters, ok := pathItem["parameters"].([]any); ok {
				operationParameters, _ := copy["parameters"].([]any)
				copy["parameters"] = append(append([]any(nil), pathParameters...), operationParameters...)
			}
			document.operations[strings.ToUpper(method)+" "+normalized] = copy
		}
	}
	return document, nil
}

func (document openAPIDocument) operation(method, path string) openAPIOperation {
	raw, found := document.operations[strings.ToUpper(method)+" "+normalizeOpenAPIPath(path)]
	if !found {
		return openAPIOperation{}
	}
	result := openAPIOperation{
		Found:             true,
		Summary:           textValue(raw["summary"]),
		Description:       textValue(raw["description"]),
		Responses:         mapValue(raw["responses"]),
		ReferencedSchemas: map[string]any{},
	}
	result.Title = result.Summary
	result.Tags = stringSlice(raw["tags"])

	parameters, _ := raw["parameters"].([]any)
	for _, rawParameter := range parameters {
		parameterMap := resolveLocalRef(document.root, mapValue(rawParameter))
		parameter := normalizeOpenAPIParameter(parameterMap)
		switch parameter.In {
		case "path":
			result.PathParams = append(result.PathParams, parameter)
		case "query":
			result.QueryParams = append(result.QueryParams, parameter)
		case "formData", "form":
			result.FormParams = append(result.FormParams, parameter)
		}
	}

	if requestBody := resolveLocalRef(document.root, mapValue(raw["requestBody"])); len(requestBody) > 0 {
		result.RequestBody, result.FormParams = normalizeOpenAPIRequestBody(requestBody, result.FormParams)
	}
	collectReferencedSchemas(document.root, raw, result.ReferencedSchemas, map[string]bool{})
	if len(result.ReferencedSchemas) == 0 {
		result.ReferencedSchemas = nil
	}
	return result
}

func normalizeOpenAPIPath(path string) string {
	return openAPIPathParameterPattern.ReplaceAllString(path, `:$1`)
}

func normalizeOpenAPIParameter(raw map[string]any) Parameter {
	schema := mapValue(raw["schema"])
	parameter := Parameter{
		Name:        textValue(raw["name"]),
		In:          textValue(raw["in"]),
		Required:    boolValue(raw["required"]),
		Description: textValue(raw["description"]),
		Schema:      schema,
	}
	parameter.Type = textValue(schema["type"])
	parameter.Default = schema["default"]
	parameter.Enum = anySlice(schema["enum"])
	return parameter
}

func normalizeOpenAPIRequestBody(raw map[string]any, formParameters []Parameter) (map[string]any, []Parameter) {
	content := mapValue(raw["content"])
	contentType := "application/json"
	media := mapValue(content[contentType])
	if len(media) == 0 {
		keys := make([]string, 0, len(content))
		for key := range content {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) > 0 {
			contentType = keys[0]
			media = mapValue(content[contentType])
		}
	}
	schema := mapValue(media["schema"])
	requestBody := map[string]any{
		"required":    boolValue(raw["required"]),
		"contentType": contentType,
		"schema":      schema,
	}
	if example, ok := media["example"]; ok {
		requestBody["example"] = example
	}
	if examples, ok := media["examples"]; ok {
		requestBody["examples"] = examples
	}
	if strings.Contains(contentType, "form") {
		properties := mapValue(schema["properties"])
		required := stringSet(schema["required"])
		for name, rawProperty := range properties {
			property := mapValue(rawProperty)
			formParameters = append(formParameters, Parameter{
				Name:        name,
				In:          "form",
				Required:    required[name],
				Type:        textValue(property["type"]),
				Description: textValue(property["description"]),
				Schema:      property,
			})
		}
	}
	return requestBody, formParameters
}

func collectReferencedSchemas(root map[string]any, node any, output map[string]any, visiting map[string]bool) {
	switch value := node.(type) {
	case map[string]any:
		if reference := textValue(value["$ref"]); strings.HasPrefix(reference, "#/components/schemas/") {
			name := strings.TrimPrefix(reference, "#/components/schemas/")
			if !visiting[name] {
				visiting[name] = true
				schema := lookupLocalRef(root, reference)
				if len(schema) > 0 {
					output[name] = schema
					collectReferencedSchemas(root, schema, output, visiting)
				}
				delete(visiting, name)
			}
		}
		for _, child := range value {
			collectReferencedSchemas(root, child, output, visiting)
		}
	case []any:
		for _, child := range value {
			collectReferencedSchemas(root, child, output, visiting)
		}
	}
}

func resolveLocalRef(root, value map[string]any) map[string]any {
	if reference := textValue(value["$ref"]); strings.HasPrefix(reference, "#/") {
		if resolved := lookupLocalRef(root, reference); len(resolved) > 0 {
			return resolved
		}
	}
	return value
}

func lookupLocalRef(root map[string]any, reference string) map[string]any {
	current := any(root)
	for _, part := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = object[strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")]
		if !ok {
			return nil
		}
	}
	return cloneMap(mapValue(current))
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	copy := make(map[string]any, len(value))
	for key, item := range value {
		copy[key] = item
	}
	return copy
}

func mapValue(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func textValue(value any) string {
	result, _ := value.(string)
	return strings.TrimSpace(result)
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func anySlice(value any) []any {
	result, _ := value.([]any)
	return result
}

func stringSlice(value any) []string {
	raw, _ := value.([]any)
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text := textValue(item); text != "" {
			result = append(result, text)
		}
	}
	return result
}

func stringSet(value any) map[string]bool {
	result := map[string]bool{}
	for _, item := range anySlice(value) {
		if text := textValue(item); text != "" {
			result[text] = true
		}
	}
	return result
}

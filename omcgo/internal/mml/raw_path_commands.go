package mml

import (
	"fmt"
	"sort"
	"strings"
)

const (
	rawPathModeStandard = "standard"
	rawPathModePrivate  = "private"
	planOrderScale      = 1000
)

func isRawPathCommandEntry(entry map[string]interface{}) bool {
	if entry == nil {
		return false
	}
	if strings.TrimSpace(commandString(entry, "raw_path_mode")) != "" {
		return true
	}
	code := strings.ToUpper(commandString(entry, "command_code"))
	return strings.HasPrefix(code, "RAW ") || strings.HasSuffix(code, " PATH")
}

func rawPathOperation(entry map[string]interface{}) string {
	op := strings.ToUpper(strings.TrimSpace(commandString(entry, "operation_type")))
	if op == "" {
		code := strings.ToUpper(commandString(entry, "command_code"))
		switch {
		case strings.HasPrefix(code, "RAW "):
			op = strings.TrimSpace(strings.TrimPrefix(code, "RAW "))
		case strings.HasSuffix(code, " PATH"):
			op = strings.TrimSpace(strings.TrimSuffix(code, " PATH"))
		default:
			op = deriveOperationType(code)
		}
		if fields := strings.Fields(op); len(fields) > 0 {
			op = fields[0]
		}
	}
	if op == "DEL" {
		return "RMV"
	}
	return op
}

func normalizeRawPathCommandEntry(entry map[string]interface{}) ([]map[string]interface{}, error) {
	op := rawPathOperation(entry)
	paths := nonEmptyStringSlice(commandStringSlice(entry, "param_paths"))
	params := commandAnyMap(entry, "parameters")
	if params == nil {
		params = map[string]interface{}{}
	}
	pathMode := normalizeRawPathMode(commandString(entry, "raw_path_mode"))

	switch op {
	case "LST", "DSP":
		if len(paths) == 0 {
			return nil, fmt.Errorf("raw PATH %s requires at least one path", op)
		}
		refs := make([]MMLParamRef, len(paths))
		for i, path := range paths {
			refs[i] = MMLParamRef{Tr069Path: path, ValueType: "string", PathMode: pathMode}
		}
		return []map[string]interface{}{{
			"command_code":   "RAW " + op,
			"rpc_method":     "GetParameterValues",
			"operation_type": op,
			"param_paths":    paths,
			"param_refs":     refs,
			"parameters":     params,
			"raw_path_mode":  pathMode,
		}}, nil
	case "MOD":
		if len(paths) == 0 {
			return nil, fmt.Errorf("raw PATH MOD requires at least one path")
		}
		refs := make([]MMLParamRef, len(paths))
		for i, path := range paths {
			refs[i] = MMLParamRef{ParamCode: path, Tr069Path: path, ValueType: "string", IsWritable: true, PathMode: pathMode}
			if _, ok := params[path]; !ok {
				return nil, fmt.Errorf("raw PATH MOD requires value for path %q", path)
			}
		}
		return []map[string]interface{}{{
			"command_code":   "RAW MOD",
			"rpc_method":     "SetParameterValues",
			"operation_type": "MOD",
			"param_paths":    paths,
			"param_refs":     refs,
			"parameters":     params,
			"raw_path_mode":  pathMode,
		}}, nil
	case "ADD":
		if len(paths) != 1 {
			return nil, fmt.Errorf("raw PATH ADD requires exactly one object path, got %d", len(paths))
		}
		objectPath := ensureTrailingDot(paths[0])
		addCommand := map[string]interface{}{
			"command_code":   "RAW ADD",
			"rpc_method":     "AddObject",
			"operation_type": "ADD",
			"param_paths":    []string{objectPath},
			"parameters":     rawObjectParameters(objectPath, pathMode),
			"raw_path_mode":  pathMode,
		}
		valueKeys := sortedRawAddValueKeys(params)
		if len(valueKeys) == 0 {
			return []map[string]interface{}{addCommand}, nil
		}
		spvParams := make(map[string]interface{}, len(valueKeys))
		spvPaths := make([]string, 0, len(valueKeys))
		spvRefs := make([]MMLParamRef, 0, len(valueKeys))
		for _, key := range valueKeys {
			relative := strings.TrimPrefix(strings.TrimSpace(key), ".")
			path := objectPath + "{NEW}." + relative
			spvParams[key] = params[key]
			spvPaths = append(spvPaths, path)
			spvRefs = append(spvRefs, MMLParamRef{
				ParamCode:   key,
				Tr069Path:   path,
				ValueType:   "string",
				IsWritable:  true,
				IsRequired:  false,
				PrivatePath: "",
				PathMode:    pathMode,
			})
		}
		return []map[string]interface{}{
			addCommand,
			{
				"command_code":    "RAW MOD",
				"rpc_method":      "SetParameterValues",
				"operation_type":  "MOD",
				"param_paths":     spvPaths,
				"param_refs":      spvRefs,
				"parameters":      spvParams,
				"raw_path_mode":   pathMode,
				"compound_phase":  "spv_after_add",
				"compound_parent": "RAW ADD",
			},
		}, nil
	case "RMV":
		if len(paths) != 1 {
			return nil, fmt.Errorf("raw PATH RMV requires exactly one object path, got %d", len(paths))
		}
		objectPath := ensureTrailingDot(paths[0])
		return []map[string]interface{}{{
			"command_code":   "RAW RMV",
			"rpc_method":     "DeleteObject",
			"operation_type": "RMV",
			"param_paths":    []string{objectPath},
			"parameters":     rawObjectParameters(objectPath, pathMode),
			"raw_path_mode":  pathMode,
		}}, nil
	default:
		return nil, fmt.Errorf("raw PATH mode: unsupported operation_type %q", op)
	}
}

func normalizeRawPathMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case rawPathModePrivate:
		return rawPathModePrivate
	default:
		return rawPathModeStandard
	}
}

func rawObjectParameters(objectPath, pathMode string) map[string]interface{} {
	params := map[string]interface{}{"object_name": objectPath}
	if normalizeRawPathMode(pathMode) == rawPathModePrivate {
		params["path_mode"] = rawPathModePrivate
	}
	return params
}

func commandStringSlice(entry map[string]interface{}, key string) []string {
	raw, ok := entry[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func commandAnyMap(entry map[string]interface{}, key string) map[string]interface{} {
	raw, ok := entry[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for k, v := range typed {
			out[k] = v
		}
		return out
	case map[string]string:
		out := make(map[string]interface{}, len(typed))
		for k, v := range typed {
			out[k] = v
		}
		return out
	default:
		return nil
	}
}

func nonEmptyStringSlice(in []string) []string {
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func ensureTrailingDot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasSuffix(path, ".") {
		return path
	}
	return path + "."
}

func sortedRawAddValueKeys(params map[string]interface{}) []string {
	keys := make([]string, 0, len(params))
	for key := range params {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" || strings.EqualFold(trimmed, "object_name") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.TrimSpace(keys[i]) < strings.TrimSpace(keys[j])
	})
	return keys
}

func commandInt(entry map[string]interface{}, key string) int {
	if entry == nil {
		return 0
	}
	switch value := entry[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case jsonNumber:
		i, _ := value.Int64()
		return int(i)
	}
	return 0
}

type jsonNumber interface {
	Int64() (int64, error)
}

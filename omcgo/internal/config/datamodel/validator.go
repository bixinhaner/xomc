package datamodel

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParameterValidator validates parameter values against a data model.
type ParameterValidator struct {
	params        []Parameter
	objects       []ObjectInfo
	paramIndex    map[string]int    // exact path → index (non-multi-instance)
	paramRegexes  []paramRegexEntry // multi-instance param regex matchers
	objectIndex   map[string]int    // exact object path → index
	objectRegexes []objectRegexEntry // multi-instance object regex matchers
}

type paramRegexEntry struct {
	regex *regexp.Regexp
	index int
}

type objectRegexEntry struct {
	regex *regexp.Regexp
	index int
}

// ValidationError describes a parameter validation failure.
type ValidationError struct {
	Path    string `json:"path"`
	Rule    string `json:"rule"`    // exists, writable, type, constraint
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: [%s] %s", e.Path, e.Rule, e.Message)
}

// NewParameterValidator builds a validator from a DataModel.
func NewParameterValidator(dm *DataModel) (*ParameterValidator, error) {
	v := &ParameterValidator{
		paramIndex:  make(map[string]int),
		objectIndex: make(map[string]int),
	}

	// Parse parameters from JSON.
	if dm.ParameterTree != nil {
		if err := json.Unmarshal(dm.ParameterTree, &v.params); err != nil {
			return nil, fmt.Errorf("unmarshal parameter_tree: %w", err)
		}
	}

	// Parse objects from JSON.
	if dm.ObjectTree != nil {
		if err := json.Unmarshal(dm.ObjectTree, &v.objects); err != nil {
			return nil, fmt.Errorf("unmarshal object_tree: %w", err)
		}
	}

	// Build indexes.
	for i, p := range v.params {
		if ContainsPlaceholder(p.Path) {
			v.paramRegexes = append(v.paramRegexes, paramRegexEntry{
				regex: TemplateToRegex(p.Path),
				index: i,
			})
		} else {
			v.paramIndex[p.Path] = i
		}
	}

	for i, o := range v.objects {
		if ContainsPlaceholder(o.Name) {
			v.objectRegexes = append(v.objectRegexes, objectRegexEntry{
				regex: TemplateToRegex(o.Name),
				index: i,
			})
		} else {
			v.objectIndex[o.Name] = i
		}
	}

	return v, nil
}

// LookupParam finds the parameter definition for an actual device path.
func (v *ParameterValidator) LookupParam(actualPath string) *Parameter {
	if idx, ok := v.paramIndex[actualPath]; ok {
		return &v.params[idx]
	}
	for _, entry := range v.paramRegexes {
		if entry.regex.MatchString(actualPath) {
			return &v.params[entry.index]
		}
	}
	return nil
}

// LookupObject finds the object definition for an actual device object path.
func (v *ParameterValidator) LookupObject(actualPath string) *ObjectInfo {
	if idx, ok := v.objectIndex[actualPath]; ok {
		return &v.objects[idx]
	}
	for _, entry := range v.objectRegexes {
		if entry.regex.MatchString(actualPath) {
			return &v.objects[entry.index]
		}
	}
	return nil
}

// ValidateValue validates a single parameter value.
func (v *ParameterValidator) ValidateValue(path, value string) *ValidationError {
	param := v.LookupParam(path)
	if param == nil {
		return &ValidationError{Path: path, Rule: "exists", Message: "参数路径不存在于数据模型中"}
	}
	if !param.Writable {
		return &ValidationError{Path: path, Rule: "writable", Message: "参数为只读，不可修改"}
	}
	if err := validateType(param.Type, value); err != nil {
		return &ValidationError{Path: path, Rule: "type", Message: err.Error()}
	}
	if param.Constraints != nil {
		if err := validateConstraints(param.Type, param.Constraints, value); err != nil {
			return &ValidationError{Path: path, Rule: "constraint", Message: err.Error()}
		}
	}
	return nil
}

// ValidateValues validates multiple parameter values, returning all errors.
func (v *ParameterValidator) ValidateValues(params []struct{ Path, Value string }) []ValidationError {
	var errs []ValidationError
	for _, p := range params {
		if ve := v.ValidateValue(p.Path, p.Value); ve != nil {
			errs = append(errs, *ve)
		}
	}
	return errs
}

// ValidateAddObject checks whether adding an instance is allowed.
func (v *ParameterValidator) ValidateAddObject(objectPath string, currentCount int) *ValidationError {
	obj := v.LookupObject(objectPath)
	if obj == nil {
		return &ValidationError{Path: objectPath, Rule: "exists", Message: "对象路径不存在于数据模型中"}
	}
	if obj.Access != "READ_WRITE" {
		return &ValidationError{Path: objectPath, Rule: "access", Message: "对象为只读，不可添加实例"}
	}
	if obj.MaxInstances > 0 && currentCount >= obj.MaxInstances {
		return &ValidationError{
			Path:    objectPath,
			Rule:    "max_instances",
			Message: fmt.Sprintf("已达最大实例数 %d，不可继续添加", obj.MaxInstances),
		}
	}
	return nil
}

// ValidateDeleteObject checks whether deleting an instance is allowed.
func (v *ParameterValidator) ValidateDeleteObject(objectPath string, currentCount int) *ValidationError {
	obj := v.LookupObject(objectPath)
	if obj == nil {
		return &ValidationError{Path: objectPath, Rule: "exists", Message: "对象路径不存在于数据模型中"}
	}
	if obj.Access != "READ_WRITE" {
		return &ValidationError{Path: objectPath, Rule: "access", Message: "对象为只读，不可删除实例"}
	}
	minInst := GetMinInstances(*obj)
	if currentCount <= minInst {
		return &ValidationError{
			Path:    objectPath,
			Rule:    "min_instances",
			Message: fmt.Sprintf("当前实例数 %d 已达最小要求 %d，不可删除", currentCount, minInst),
		}
	}
	return nil
}

// GetParams returns the parsed parameter list.
func (v *ParameterValidator) GetParams() []Parameter {
	return v.params
}

// GetObjects returns the parsed object list.
func (v *ParameterValidator) GetObjects() []ObjectInfo {
	return v.objects
}

// validateType checks that the value is valid for the given parameter type.
func validateType(paramType, value string) error {
	switch paramType {
	case "string":
		return nil
	case "int":
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("值 %q 不是有效的整数", value)
		}
	case "unsignedInt":
		_, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("值 %q 不是有效的无符号整数", value)
		}
	case "long":
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("值 %q 不是有效的长整数", value)
		}
	case "unsignedLong":
		_, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("值 %q 不是有效的无符号长整数", value)
		}
	case "boolean":
		lower := strings.ToLower(value)
		if lower != "true" && lower != "false" && value != "0" && value != "1" {
			return fmt.Errorf("值 %q 不是有效的布尔值，应为 true/false/0/1", value)
		}
	case "dateTime":
		_, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return fmt.Errorf("值 %q 不是有效的 ISO 8601 日期时间格式", value)
		}
	case "base64":
		_, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return fmt.Errorf("值 %q 不是有效的 Base64 编码", value)
		}
	}
	return nil
}

// validateConstraints checks value against parameter constraints.
func validateConstraints(paramType string, c *Constraints, value string) error {
	if c == nil {
		return nil
	}

	// Check enum values first (applies to any type).
	if len(c.EnumValues) > 0 {
		found := false
		for _, ev := range c.EnumValues {
			if value == ev {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("值 %q 不在允许的枚举列表中: %v", value, c.EnumValues)
		}
	}

	// Pattern check.
	if c.Pattern != "" {
		matched, err := regexp.MatchString(c.Pattern, value)
		if err != nil {
			return fmt.Errorf("正则表达式 %q 无效: %w", c.Pattern, err)
		}
		if !matched {
			return fmt.Errorf("值 %q 不匹配模式 %q", value, c.Pattern)
		}
	}

	// String length checks.
	if paramType == "string" {
		if c.MaxLength > 0 && len(value) > c.MaxLength {
			return fmt.Errorf("值长度 %d 超过最大长度 %d", len(value), c.MaxLength)
		}
		if c.MinLength > 0 && len(value) < c.MinLength {
			return fmt.Errorf("值长度 %d 小于最小长度 %d", len(value), c.MinLength)
		}
	}

	// Numeric range checks.
	switch paramType {
	case "int", "long":
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			if c.MinValue != nil && v < *c.MinValue {
				return fmt.Errorf("值 %d 小于最小值 %d", v, *c.MinValue)
			}
			if c.MaxValue != nil && v > *c.MaxValue {
				return fmt.Errorf("值 %d 大于最大值 %d", v, *c.MaxValue)
			}
		}
	case "unsignedInt", "unsignedLong":
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			if c.MinValue != nil && v < *c.MinValue {
				return fmt.Errorf("值 %d 小于最小值 %d", v, *c.MinValue)
			}
			if c.MaxValue != nil && v > *c.MaxValue {
				return fmt.Errorf("值 %d 大于最大值 %d", v, *c.MaxValue)
			}
		}
	}

	return nil
}

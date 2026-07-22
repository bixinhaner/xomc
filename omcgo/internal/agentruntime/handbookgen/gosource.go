package handbookgen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type handlerContract struct {
	Found       bool           `json:"found"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	QueryParams []Parameter    `json:"queryParams,omitempty"`
	FormParams  []Parameter    `json:"formParams,omitempty"`
	RequestBody map[string]any `json:"requestBody,omitempty"`
}

type goSourceAnalyzer struct {
	root        string
	packages    map[string]*goPackageIndex
	globalTypes map[string]ast.Expr
}

type goPackageIndex struct {
	functions map[string]*ast.FuncDecl
	types     map[string]ast.Expr
}

func newGoSourceAnalyzer(root string) (*goSourceAnalyzer, error) {
	root = strings.TrimSpace(root)
	analyzer := &goSourceAnalyzer{root: root, packages: map[string]*goPackageIndex{}, globalTypes: map[string]ast.Expr{}}
	if root != "" {
		analyzer.globalTypes = indexUniqueGoTypes(root)
	}
	return analyzer, nil
}

func (analyzer *goSourceAnalyzer) analyze(handler string) handlerContract {
	packagePath, receiver, method, ok := parseHandlerIdentity(handler)
	if !ok || analyzer == nil || analyzer.root == "" {
		return handlerContract{}
	}
	index := analyzer.packageIndex(packagePath)
	if index == nil {
		return handlerContract{}
	}
	function := index.functions[receiver+"."+method]
	if function == nil || function.Body == nil {
		return handlerContract{}
	}
	return analyzeGoFunction(index, receiver, function, map[string]bool{})
}

func (analyzer *goSourceAnalyzer) allHandlerContracts() map[string]handlerContract {
	if analyzer == nil || analyzer.root == "" {
		return map[string]handlerContract{}
	}
	packagePaths := map[string]struct{}{}
	_ = filepath.WalkDir(analyzer.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != analyzer.root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "vendor" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, relativeErr := filepath.Rel(analyzer.root, filepath.Dir(path))
		if relativeErr == nil && relative != "." {
			packagePaths[filepath.ToSlash(relative)] = struct{}{}
		}
		return nil
	})

	orderedPackages := make([]string, 0, len(packagePaths))
	for packagePath := range packagePaths {
		orderedPackages = append(orderedPackages, packagePath)
	}
	sort.Strings(orderedPackages)
	contracts := make(map[string]handlerContract)
	for _, packagePath := range orderedPackages {
		index := analyzer.packageIndex(packagePath)
		if index == nil {
			continue
		}
		functionKeys := make([]string, 0, len(index.functions))
		for key := range index.functions {
			functionKeys = append(functionKeys, key)
		}
		sort.Strings(functionKeys)
		for _, key := range functionKeys {
			receiver, method, found := strings.Cut(key, ".")
			function := index.functions[key]
			if !found || receiver == "" || method == "" || !acceptsGinContext(function) {
				continue
			}
			handler := "github.com/omcgo/omcgo/" + packagePath + ".(*" + receiver + ")." + method + "-fm"
			contract := analyzeGoFunction(index, receiver, function, map[string]bool{})
			if contract.Found {
				contracts[handler] = contract
			}
		}
	}
	return contracts
}

func acceptsGinContext(function *ast.FuncDecl) bool {
	if function == nil || function.Type == nil || function.Type.Params == nil {
		return false
	}
	for _, field := range function.Type.Params.List {
		expression := field.Type
		if pointer, ok := expression.(*ast.StarExpr); ok {
			expression = pointer.X
		}
		selector, ok := expression.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil || selector.Sel.Name != "Context" {
			continue
		}
		if identifier, ok := selector.X.(*ast.Ident); ok && identifier.Name == "gin" {
			return true
		}
	}
	return false
}

func analyzeGoFunction(index *goPackageIndex, receiver string, function *ast.FuncDecl, visiting map[string]bool) handlerContract {
	key := receiver + "." + function.Name.Name
	if visiting[key] {
		return handlerContract{}
	}
	visiting[key] = true
	defer delete(visiting, key)

	contract := handlerContract{Found: true}
	if function.Doc != nil {
		raw := strings.TrimSpace(function.Doc.Text())
		contract.Summary = goDocDirective(raw, "@Summary")
		contract.Description = goDocDirective(raw, "@Description")
		if contract.Summary == "" && contract.Description == "" {
			contract.Description = raw
		}
	}

	variables := variableTypes(function.Body)
	receiverVariable := functionReceiverVariable(function)
	ast.Inspect(function.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if identifier, ok := selector.X.(*ast.Ident); ok && identifier.Name == receiverVariable {
			if helper := index.functions[receiver+"."+selector.Sel.Name]; helper != nil && helper != function {
				contract = mergeHandlerContracts(contract, analyzeGoFunction(index, receiver, helper, visiting))
			}
		}
		if parameter, ok := parameterFromContextCall(selector.Sel.Name, call.Args); ok {
			if parameter.In == "query" {
				contract.QueryParams = append(contract.QueryParams, parameter)
			} else {
				contract.FormParams = append(contract.FormParams, parameter)
			}
			return true
		}
		if len(call.Args) == 0 {
			return true
		}
		typeExpression := boundTypeExpression(call.Args[0], variables, index.types)
		if typeExpression == nil {
			return true
		}
		switch selector.Sel.Name {
		case "ShouldBindQuery":
			schema := schemaFromExpression(typeExpression, index.types, map[string]bool{})
			contract.QueryParams = append(contract.QueryParams, parametersFromSchema(schema, "query")...)
		case "ShouldBindJSON", "BindJSON", "ShouldBind", "Bind":
			contract.RequestBody = map[string]any{
				"required":    true,
				"contentType": "application/json",
				"schema":      schemaFromExpression(typeExpression, index.types, map[string]bool{}),
			}
		}
		return true
	})
	contract.QueryParams = mergeParameters(nil, contract.QueryParams)
	contract.FormParams = mergeParameters(nil, contract.FormParams)
	return contract
}

func functionReceiverVariable(function *ast.FuncDecl) string {
	if function == nil || function.Recv == nil || len(function.Recv.List) == 0 || len(function.Recv.List[0].Names) == 0 {
		return ""
	}
	return function.Recv.List[0].Names[0].Name
}

func mergeHandlerContracts(primary, secondary handlerContract) handlerContract {
	primary.Found = primary.Found || secondary.Found
	primary.QueryParams = mergeParameters(primary.QueryParams, secondary.QueryParams)
	primary.FormParams = mergeParameters(primary.FormParams, secondary.FormParams)
	if len(primary.RequestBody) == 0 {
		primary.RequestBody = secondary.RequestBody
	}
	return primary
}

func goDocDirective(comment, name string) string {
	for _, line := range strings.Split(comment, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, name) {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(line, name))
	}
	return ""
}

func (analyzer *goSourceAnalyzer) packageIndex(packagePath string) *goPackageIndex {
	if cached, ok := analyzer.packages[packagePath]; ok {
		return cached
	}
	directory := filepath.Join(analyzer.root, filepath.FromSlash(packagePath))
	packages, err := parser.ParseDir(token.NewFileSet(), directory, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		analyzer.packages[packagePath] = nil
		return nil
	}
	index := &goPackageIndex{functions: map[string]*ast.FuncDecl{}, types: map[string]ast.Expr{}}
	for name, expression := range analyzer.globalTypes {
		index.types["*."+name] = expression
	}
	for _, parsedPackage := range packages {
		for _, file := range parsedPackage.Files {
			for _, declaration := range file.Decls {
				switch typed := declaration.(type) {
				case *ast.FuncDecl:
					index.functions[functionKey(typed)] = typed
				case *ast.GenDecl:
					if typed.Tok != token.TYPE {
						continue
					}
					for _, specification := range typed.Specs {
						if typeSpec, ok := specification.(*ast.TypeSpec); ok {
							index.types[typeSpec.Name.Name] = typeSpec.Type
						}
					}
				}
			}
		}
	}
	analyzer.packages[packagePath] = index
	return index
}

func indexUniqueGoTypes(root string) map[string]ast.Expr {
	candidates := map[string][]ast.Expr{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "vendor" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil {
			return nil
		}
		for _, declaration := range file.Decls {
			generic, ok := declaration.(*ast.GenDecl)
			if !ok || generic.Tok != token.TYPE {
				continue
			}
			for _, specification := range generic.Specs {
				if typeSpec, ok := specification.(*ast.TypeSpec); ok {
					candidates[typeSpec.Name.Name] = append(candidates[typeSpec.Name.Name], typeSpec.Type)
				}
			}
		}
		return nil
	})
	result := map[string]ast.Expr{}
	for name, expressions := range candidates {
		if len(expressions) == 1 {
			result[name] = expressions[0]
		}
	}
	return result
}

func parseHandlerIdentity(handler string) (string, string, string, bool) {
	const modulePrefix = "github.com/omcgo/omcgo/"
	value := strings.TrimSuffix(strings.TrimSpace(handler), "-fm")
	if !strings.HasPrefix(value, modulePrefix) {
		return "", "", "", false
	}
	value = strings.TrimPrefix(value, modulePrefix)
	receiverAt := strings.LastIndex(value, ".(")
	methodAt := strings.LastIndex(value, ").")
	if receiverAt < 0 || methodAt < receiverAt {
		return "", "", "", false
	}
	packagePath := value[:receiverAt]
	receiver := strings.TrimPrefix(value[receiverAt+2:methodAt], "*")
	method := value[methodAt+2:]
	if packagePath == "" || receiver == "" || method == "" || strings.Contains(method, ".") {
		return "", "", "", false
	}
	return packagePath, receiver, method, true
}

func functionKey(function *ast.FuncDecl) string {
	if function == nil || function.Name == nil {
		return ""
	}
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return function.Name.Name
	}
	receiver := function.Recv.List[0].Type
	if pointer, ok := receiver.(*ast.StarExpr); ok {
		receiver = pointer.X
	}
	identifier, _ := receiver.(*ast.Ident)
	if identifier == nil {
		return function.Name.Name
	}
	return identifier.Name + "." + function.Name.Name
}

func variableTypes(body *ast.BlockStmt) map[string]ast.Expr {
	result := map[string]ast.Expr{}
	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.DeclStmt:
			declaration, ok := typed.Decl.(*ast.GenDecl)
			if !ok || declaration.Tok != token.VAR {
				return true
			}
			for _, specification := range declaration.Specs {
				valueSpec, ok := specification.(*ast.ValueSpec)
				if !ok || valueSpec.Type == nil {
					continue
				}
				for _, name := range valueSpec.Names {
					result[name.Name] = valueSpec.Type
				}
			}
		case *ast.AssignStmt:
			for index, expression := range typed.Rhs {
				literal, ok := expression.(*ast.CompositeLit)
				if !ok || index >= len(typed.Lhs) {
					continue
				}
				name, ok := typed.Lhs[index].(*ast.Ident)
				if ok {
					result[name.Name] = literal.Type
				}
			}
		}
		return true
	})
	return result
}

func parameterFromContextCall(name string, arguments []ast.Expr) (Parameter, bool) {
	if len(arguments) == 0 {
		return Parameter{}, false
	}
	parameterName, ok := stringLiteral(arguments[0])
	if !ok || parameterName == "" {
		return Parameter{}, false
	}
	parameter := Parameter{Name: parameterName, Required: false, Type: "string"}
	switch name {
	case "Query", "GetQuery", "QueryArray", "GetQueryArray":
		parameter.In = "query"
		if strings.Contains(name, "Array") {
			parameter.Type = "array"
		}
	case "DefaultQuery":
		parameter.In = "query"
		if len(arguments) > 1 {
			if value, ok := stringLiteral(arguments[1]); ok {
				parameter.Default = value
			}
		}
	case "PostForm", "GetPostForm", "PostFormArray", "GetPostFormArray":
		parameter.In = "form"
		if strings.Contains(name, "Array") {
			parameter.Type = "array"
		}
	case "DefaultPostForm":
		parameter.In = "form"
		if len(arguments) > 1 {
			if value, ok := stringLiteral(arguments[1]); ok {
				parameter.Default = value
			}
		}
	case "FormFile":
		parameter.In = "form"
		parameter.Type = "file"
	default:
		return Parameter{}, false
	}
	return parameter, true
}

func boundTypeExpression(expression ast.Expr, variables map[string]ast.Expr, types map[string]ast.Expr) ast.Expr {
	if unary, ok := expression.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expression = unary.X
	}
	switch typed := expression.(type) {
	case *ast.Ident:
		return variables[typed.Name]
	case *ast.SelectorExpr:
		base := boundTypeExpression(typed.X, variables, types)
		return structFieldType(base, typed.Sel.Name, types)
	default:
		return nil
	}
}

func structFieldType(expression ast.Expr, fieldName string, types map[string]ast.Expr) ast.Expr {
	switch typed := expression.(type) {
	case *ast.Ident:
		if definition := types[typed.Name]; definition != nil {
			return structFieldType(definition, fieldName, types)
		}
	case *ast.StarExpr:
		return structFieldType(typed.X, fieldName, types)
	case *ast.StructType:
		for _, field := range typed.Fields.List {
			for _, name := range field.Names {
				if name.Name == fieldName {
					return field.Type
				}
			}
			if len(field.Names) == 0 {
				if identifier, ok := field.Type.(*ast.Ident); ok && identifier.Name == fieldName {
					return field.Type
				}
				if selector, ok := field.Type.(*ast.SelectorExpr); ok && selector.Sel.Name == fieldName {
					return field.Type
				}
			}
		}
	}
	return nil
}

func schemaFromExpression(expression ast.Expr, types map[string]ast.Expr, visiting map[string]bool) map[string]any {
	switch typed := expression.(type) {
	case *ast.Ident:
		if primitive := primitiveSchema(typed.Name); primitive != nil {
			return primitive
		}
		if visiting[typed.Name] {
			return map[string]any{"type": "object", "goType": typed.Name}
		}
		if definition := types[typed.Name]; definition != nil {
			visiting[typed.Name] = true
			schema := schemaFromExpression(definition, types, visiting)
			delete(visiting, typed.Name)
			schema["goType"] = typed.Name
			return schema
		}
		return map[string]any{"type": "object", "goType": typed.Name}
	case *ast.StarExpr:
		schema := schemaFromExpression(typed.X, types, visiting)
		schema["nullable"] = true
		return schema
	case *ast.ArrayType:
		return map[string]any{"type": "array", "items": schemaFromExpression(typed.Elt, types, visiting)}
	case *ast.MapType:
		return map[string]any{"type": "object", "additionalProperties": schemaFromExpression(typed.Value, types, visiting)}
	case *ast.SelectorExpr:
		packageName, _ := typed.X.(*ast.Ident)
		goType := typed.Sel.Name
		if packageName != nil {
			goType = packageName.Name + "." + goType
		}
		if goType == "time.Time" {
			return map[string]any{"type": "string", "format": "date-time", "goType": goType}
		}
		if strings.HasSuffix(goType, ".UUID") {
			return map[string]any{"type": "string", "format": "uuid", "goType": goType}
		}
		if definition := types["*."+typed.Sel.Name]; definition != nil {
			schema := schemaFromExpression(definition, types, visiting)
			schema["goType"] = goType
			return schema
		}
		return map[string]any{"type": "object", "goType": goType}
	case *ast.StructType:
		return schemaFromStruct(typed, types, visiting)
	case *ast.InterfaceType:
		return map[string]any{"type": "object", "additionalProperties": true}
	default:
		return map[string]any{"type": "object"}
	}
}

func schemaFromStruct(structType *ast.StructType, types map[string]ast.Expr, visiting map[string]bool) map[string]any {
	properties := map[string]any{}
	var required []string
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			embedded := schemaFromExpression(field.Type, types, visiting)
			for name, property := range mapValue(embedded["properties"]) {
				properties[name] = property
			}
			for name := range stringSet(embedded["required"]) {
				required = append(required, name)
			}
			continue
		}
		tag := reflect.StructTag("")
		if field.Tag != nil {
			if unquoted, err := strconv.Unquote(field.Tag.Value); err == nil {
				tag = reflect.StructTag(unquoted)
			}
		}
		for _, fieldName := range field.Names {
			name := taggedFieldName(tag.Get("json"))
			if name == "" {
				name = taggedFieldName(tag.Get("form"))
			}
			if name == "-" {
				continue
			}
			if name == "" {
				name = lowerFirst(fieldName.Name)
			}
			property := schemaFromExpression(field.Type, types, visiting)
			if description := fieldComment(field); description != "" {
				property["description"] = description
			}
			validation := tag.Get("binding") + "," + tag.Get("validate")
			if strings.Contains(validation, "required") {
				required = append(required, name)
			}
			if enum := enumFromValidation(validation); len(enum) > 0 {
				property["enum"] = enum
			}
			applyValidationConstraints(property, validation)
			if value := tag.Get("default"); value != "" {
				property["default"] = value
			}
			properties[name] = property
		}
	}
	sort.Strings(required)
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func parametersFromSchema(schema map[string]any, location string) []Parameter {
	properties := mapValue(schema["properties"])
	required := stringSet(schema["required"])
	result := make([]Parameter, 0, len(properties))
	for name, rawProperty := range properties {
		property := mapValue(rawProperty)
		result = append(result, Parameter{
			Name:        name,
			In:          location,
			Required:    required[name],
			Type:        textValue(property["type"]),
			Description: textValue(property["description"]),
			Default:     property["default"],
			Enum:        anySlice(property["enum"]),
			Schema:      property,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func primitiveSchema(name string) map[string]any {
	switch name {
	case "string":
		return map[string]any{"type": "string"}
	case "bool":
		return map[string]any{"type": "boolean"}
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return map[string]any{"type": "integer", "goType": name}
	case "float32", "float64":
		return map[string]any{"type": "number", "goType": name}
	case "byte":
		return map[string]any{"type": "integer", "format": "byte"}
	case "any":
		return map[string]any{"type": "object", "additionalProperties": true}
	default:
		return nil
	}
}

func taggedFieldName(value string) string {
	if value == "" {
		return ""
	}
	return strings.Split(value, ",")[0]
}

func lowerFirst(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func fieldComment(field *ast.Field) string {
	if field.Doc != nil {
		return strings.TrimSpace(field.Doc.Text())
	}
	if field.Comment != nil {
		return strings.TrimSpace(field.Comment.Text())
	}
	return ""
}

func enumFromValidation(validation string) []any {
	index := strings.Index(validation, "oneof=")
	if index < 0 {
		return nil
	}
	value := validation[index+len("oneof="):]
	if end := strings.Index(value, ","); end >= 0 {
		value = value[:end]
	}
	fields := strings.Fields(value)
	result := make([]any, 0, len(fields))
	for _, field := range fields {
		result = append(result, field)
	}
	return result
}

func applyValidationConstraints(schema map[string]any, validation string) {
	for _, rule := range strings.Split(validation, ",") {
		key, rawValue, ok := strings.Cut(strings.TrimSpace(rule), "=")
		if !ok || rawValue == "" {
			continue
		}
		value, err := strconv.Atoi(rawValue)
		if err != nil {
			continue
		}
		typeName := textValue(schema["type"])
		switch key {
		case "min":
			switch typeName {
			case "string":
				schema["minLength"] = value
			case "array":
				schema["minItems"] = value
			default:
				schema["minimum"] = value
			}
		case "max":
			switch typeName {
			case "string":
				schema["maxLength"] = value
			case "array":
				schema["maxItems"] = value
			default:
				schema["maximum"] = value
			}
		case "len":
			switch typeName {
			case "string":
				schema["minLength"] = value
				schema["maxLength"] = value
			case "array":
				schema["minItems"] = value
				schema["maxItems"] = value
			}
		}
	}
}

func stringLiteral(expression ast.Expr) (string, bool) {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

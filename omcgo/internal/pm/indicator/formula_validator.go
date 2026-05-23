package indicator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/omcgo/omcgo/internal/pm/kpi/expr"
)

// FormulaValidator validates indicator formulas against a known ID→formula mapping.
type FormulaValidator struct {
	// idMap maps indicator ID → arithmetic formula string.
	// Counter IDs map to themselves (or empty string).
	idMap map[string]string
}

// NewFormulaValidator creates a validator with the given ID→formula mapping.
func NewFormulaValidator(idMap map[string]string) *FormulaValidator {
return &FormulaValidator{idMap: idMap}
}

// ValidationResult holds the outcome of formula validation.
type ValidationResult struct {
	IsValid     bool
	IsCounter   bool   // true if formula is a single indicator ID with no operators
	IndicatorID string // the single indicator ID if IsCounter
	ErrorMsg    string
}

// Validate validates a formula string.
// Steps:
//  1. Tokenize by operators +-*/()
//  2. Replace each token: Duration→"1", known indicator ID→"1", K90000 custom KPI→recursive expand
//  3. Recombine and parse as numeric expression via kpi.ParseFormula + Evaluate
//  4. Must reference at least 1 indicator ID (no pure constant formulas)
//  5. If formula is a single indicator ID with no operators, mark IsCounter=true
func (v *FormulaValidator) Validate(arithmetic string) ValidationResult {
	if arithmetic == "" {
		return ValidationResult{IsValid: false, ErrorMsg: "Expression is invalid"}
	}

	tokens := tokenize(arithmetic)
	var indicatorCount int
	var singleIndicatorID string
	var onlyOneToken bool

	// Count non-operator, non-number tokens to detect single-indicator case.
	nonOperatorTokens := 0
	for _, tok := range tokens {
		if !isOperator(tok) {
			nonOperatorTokens++
		}
	}
	onlyOneToken = nonOperatorTokens == 1

	var rebuilt strings.Builder
	for _, tok := range tokens {
		if isOperator(tok) || isNumber(tok) {
			rebuilt.WriteString(tok)
			continue
		}

		// Duration keyword → 1
		if tok == "Duration" {
			rebuilt.WriteString("1")
			continue
		}

		// Custom KPI containing K90000 → recursive expand (must check before idMap)
		if strings.Contains(tok, "K90000") {
			expanded, err := v.expandCustomKPI(tok, make(map[string]bool))
			if err != nil {
				return ValidationResult{IsValid: false, ErrorMsg: "Expression is invalid"}
			}
			indicatorCount++
			rebuilt.WriteString(expanded)
			continue
		}

		// Known indicator ID
		if _, ok := v.idMap[tok]; ok {
			indicatorCount++
			if onlyOneToken {
				singleIndicatorID = tok
			}
			rebuilt.WriteString("1")
			continue
		}

		// Unknown token → invalid
		return ValidationResult{IsValid: false, ErrorMsg: "Expression is invalid"}
	}

	// Must reference at least 1 indicator
	if indicatorCount == 0 {
		return ValidationResult{IsValid: false, ErrorMsg: "Expression is invalid"}
	}

	// Parse and evaluate the numeric expression
	rebuiltExpr := rebuilt.String()
	formula, err := expr.Parse(rebuiltExpr)
	if err != nil {
		return ValidationResult{IsValid: false, ErrorMsg: "Expression is invalid"}
	}
	if _, err := formula.Evaluate(map[string]float64{}); err != nil {
		// Division by zero etc. is acceptable for validation (runtime behavior).
		// The key check is that ParseFormula succeeded.
		_ = err
	}

	return ValidationResult{
		IsValid:      true,
		IsCounter:    onlyOneToken && singleIndicatorID != "",
		IndicatorID:  singleIndicatorID,
	}
}

// expandCustomKPI recursively expands a custom KPI reference.
// Returns a numeric string (e.g., "1+1") that replaces the KPI token.
func (v *FormulaValidator) expandCustomKPI(kpiID string, visited map[string]bool) (string, error) {
	if visited[kpiID] {
		return "", fmt.Errorf("circular reference: %s", kpiID)
	}
	visited[kpiID] = true

	formula, ok := v.idMap[kpiID]
	if !ok {
		return "", fmt.Errorf("unknown KPI ID: %s", kpiID)
	}

	tokens := tokenize(formula)
	var rebuilt strings.Builder
	for _, tok := range tokens {
		if isOperator(tok) || isNumber(tok) {
			rebuilt.WriteString(tok)
			continue
		}
		if tok == "Duration" {
			rebuilt.WriteString("1")
			continue
		}
		if strings.Contains(tok, "K90000") {
			expanded, err := v.expandCustomKPI(tok, visited)
			if err != nil {
				return "", err
			}
			rebuilt.WriteString(expanded)
			continue
		}
		if _, ok := v.idMap[tok]; ok {
			rebuilt.WriteString("1")
			continue
		}
		return "", fmt.Errorf("unknown token in KPI %s: %s", kpiID, tok)
	}
	return rebuilt.String(), nil
}

// tokenize splits a formula string into tokens (identifiers, numbers, operators).
var tokenRe = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*|[0-9]+\.?[0-9]*|[+\-*/()]`)

func tokenize(formula string) []string {
	return tokenRe.FindAllString(formula, -1)
}

func isOperator(tok string) bool {
	return tok == "+" || tok == "-" || tok == "*" || tok == "/" || tok == "(" || tok == ")"
}

func isNumber(tok string) bool {
	if tok == "" {
		return false
	}
	dotSeen := false
	for i, c := range tok {
		if c == '.' {
			if dotSeen || i == 0 || i == len(tok)-1 {
				return false
			}
			dotSeen = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

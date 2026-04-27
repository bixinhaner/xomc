package indicator

import (
	"testing"
)

func TestValidate_ValidKPIFormula(t *testing.T) {
	idMap := map[string]string{
		"C01001": "C01001",
		"C01002": "C01002",
	}
	v := NewFormulaValidator(idMap)

	result := v.Validate("C01001+C01002")
	if !result.IsValid {
		t.Fatalf("expected valid, got: %s", result.ErrorMsg)
	}
	if result.IsCounter {
		t.Error("should not be detected as counter")
	}
}

func TestValidate_SingleCounter(t *testing.T) {
	idMap := map[string]string{
		"C01001": "C01001",
	}
	v := NewFormulaValidator(idMap)

	result := v.Validate("C01001")
	if !result.IsValid {
		t.Fatalf("expected valid, got: %s", result.ErrorMsg)
	}
	if !result.IsCounter {
		t.Error("should be detected as counter")
	}
	if result.IndicatorID != "C01001" {
		t.Errorf("expected IndicatorID C01001, got %s", result.IndicatorID)
	}
}

func TestValidate_InvalidUnknownToken(t *testing.T) {
	idMap := map[string]string{
		"C01001": "C01001",
	}
	v := NewFormulaValidator(idMap)

	result := v.Validate("C01001+UNKNOWN_COUNTER")
	if result.IsValid {
		t.Error("should be invalid for unknown token")
	}
	if result.ErrorMsg != "Expression is invalid" {
		t.Errorf("unexpected error: %s", result.ErrorMsg)
	}
}

func TestValidate_EmptyFormula(t *testing.T) {
	v := NewFormulaValidator(nil)
	result := v.Validate("")
	if result.IsValid {
		t.Error("empty formula should be invalid")
	}
}

func TestValidate_DurationKeyword(t *testing.T) {
	idMap := map[string]string{
		"C01001": "C01001",
	}
	v := NewFormulaValidator(idMap)

	result := v.Validate("C01001/Duration")
	if !result.IsValid {
		t.Fatalf("expected valid, got: %s", result.ErrorMsg)
	}
}

func TestValidate_PureConstant(t *testing.T) {
	v := NewFormulaValidator(map[string]string{})

	result := v.Validate("1+2")
	if result.IsValid {
		t.Error("pure constant formula should be invalid")
	}
}

func TestValidate_ComplexFormula(t *testing.T) {
	idMap := map[string]string{
		"C01001": "C01001",
		"C01002": "C01002",
		"C01003": "C01003",
	}
	v := NewFormulaValidator(idMap)

	result := v.Validate("(C01001+C01002)*C01003")
	if !result.IsValid {
		t.Fatalf("expected valid, got: %s", result.ErrorMsg)
	}
}

func TestValidate_RecursiveKPIExpansion(t *testing.T) {
	idMap := map[string]string{
		"C01001":       "C01001",
		"C01002":       "C01002",
		"defaultK900000001": "C01001+C01002",
	}
	v := NewFormulaValidator(idMap)

	// Formula references a custom KPI which itself references counters
	result := v.Validate("defaultK900000001*2")
	if !result.IsValid {
		t.Fatalf("expected valid, got: %s", result.ErrorMsg)
	}
}

func TestValidate_CircularReference(t *testing.T) {
	idMap := map[string]string{
		"defaultK900000001": "defaultK900000002",
		"defaultK900000002": "defaultK900000001",
	}
	v := NewFormulaValidator(idMap)

	result := v.Validate("defaultK900000001")
	if result.IsValid {
		t.Error("circular reference should be invalid")
	}
}

func TestValidate_Subtraction(t *testing.T) {
	idMap := map[string]string{
		"C01001": "C01001",
		"C01002": "C01002",
	}
	v := NewFormulaValidator(idMap)

	result := v.Validate("C01001-C01002")
	if !result.IsValid {
		t.Fatalf("expected valid, got: %s", result.ErrorMsg)
	}
}

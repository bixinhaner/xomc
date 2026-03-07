package collector

import (
	"fmt"
	"path/filepath"
	"strings"
)

// MR type constants.
const (
	MRTypeMRO = "mro"
	MRTypeMRS = "mrs"
	MRTypeMRE = "mre"
)

// DetectMRType determines the MR type from the file name.
// Supported patterns: *_mro_*, *_mrs_*, *_mre_*, *MRO*, *MRS*, *MRE*
func DetectMRType(filename string) (string, error) {
	base := strings.ToLower(filepath.Base(filename))

	switch {
	case strings.Contains(base, "mro"):
		return MRTypeMRO, nil
	case strings.Contains(base, "mrs"):
		return MRTypeMRS, nil
	case strings.Contains(base, "mre"):
		return MRTypeMRE, nil
	default:
		return "", fmt.Errorf("unknown MR type for file: %s", filename)
	}
}

// ValidMRTypes returns all supported MR types.
func ValidMRTypes() []string {
	return []string{MRTypeMRO, MRTypeMRS, MRTypeMRE}
}

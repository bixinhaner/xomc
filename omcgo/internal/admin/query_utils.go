package admin

import "strings"

// escapeLike escapes SQL ILIKE wildcard characters (%, _, \) in user input
// to prevent unintended pattern matching in LIKE/ILIKE queries.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// ilikePattern wraps user input with wildcards and escapes special characters.
// Produces a pattern like "%user\_input%" safe for ILIKE queries.
func ilikePattern(s string) string {
	return "%" + escapeLike(s) + "%"
}

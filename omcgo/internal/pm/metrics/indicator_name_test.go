package metrics

import (
	"testing"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

func TestIndicatorDisplayNameExpr(t *testing.T) {
	const zhExpr = "COALESCE(NULLIF(cn_name, ''), en_name)"
	const enExpr = "COALESCE(NULLIF(en_name, ''), cn_name)"

	cases := []struct {
		name string
		loc  appcontext.Locale
		want string
	}{
		{"zh prefers cn_name", appcontext.LocaleZH, zhExpr},
		{"en prefers en_name", appcontext.LocaleEN, enExpr},
		{"unknown locale falls back zh direction", appcontext.Locale("fr"), zhExpr},
		{"empty locale falls back zh direction", appcontext.Locale(""), zhExpr},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IndicatorDisplayNameExpr(c.loc); got != c.want {
				t.Fatalf("IndicatorDisplayNameExpr(%q) = %q, want %q", c.loc, got, c.want)
			}
		})
	}
}

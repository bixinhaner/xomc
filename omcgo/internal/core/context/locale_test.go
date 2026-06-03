package context

import (
	"context"
	"testing"
)

func TestParseAcceptLanguage(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   Locale
	}{
		{"empty header falls back zh", "", LocaleZH},
		{"plain en", "en", LocaleEN},
		{"en-US", "en-US", LocaleEN},
		{"en with q-value list", "en-US,en;q=0.9,zh;q=0.8", LocaleEN},
		{"zh-CN", "zh-CN", LocaleZH},
		{"zh first wins over en", "zh-CN,zh;q=0.9,en;q=0.8", LocaleZH},
		{"unknown lang falls back zh", "fr-FR", LocaleZH},
		{"case insensitive EN", "EN-us", LocaleEN},
		{"leading whitespace", "  en-US ", LocaleEN},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ParseAcceptLanguage(c.header); got != c.want {
				t.Fatalf("ParseAcceptLanguage(%q) = %q, want %q", c.header, got, c.want)
			}
		})
	}
}

func TestGetLocale(t *testing.T) {
	t.Run("missing locale falls back zh", func(t *testing.T) {
		if got := GetLocale(context.Background()); got != LocaleZH {
			t.Fatalf("GetLocale(empty) = %q, want %q", got, LocaleZH)
		}
	})
	t.Run("empty-string locale falls back zh", func(t *testing.T) {
		ctx := WithLocale(context.Background(), Locale(""))
		if got := GetLocale(ctx); got != LocaleZH {
			t.Fatalf("GetLocale(empty string) = %q, want %q", got, LocaleZH)
		}
	})
	t.Run("en round-trips", func(t *testing.T) {
		ctx := WithLocale(context.Background(), LocaleEN)
		if got := GetLocale(ctx); got != LocaleEN {
			t.Fatalf("GetLocale(en) = %q, want %q", got, LocaleEN)
		}
	})
	t.Run("zh round-trips", func(t *testing.T) {
		ctx := WithLocale(context.Background(), LocaleZH)
		if got := GetLocale(ctx); got != LocaleZH {
			t.Fatalf("GetLocale(zh) = %q, want %q", got, LocaleZH)
		}
	})
}

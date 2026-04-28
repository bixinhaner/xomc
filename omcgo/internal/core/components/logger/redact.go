package logger

import (
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/redact"
)

// RedactString returns a zap.Field that masks the value when key matches
// a sensitive name (see redact.SensitiveKeys). It is a thin re-export of
// redact.String, provided here so call-sites that already import
// "internal/core/components/logger" can opt-in without an extra import.
//
// Example:
//
//	logger.Info(ctx, "login attempt",
//	    zap.String("user", user),
//	    logger.RedactString("password", pwd),
//	)
func RedactString(key, value string) zap.Field {
	return redact.String(key, value)
}

// RedactStringp is the *string variant of RedactString.
func RedactStringp(key string, value *string) zap.Field {
	return redact.Stringp(key, value)
}

// RedactAny is a redacting wrapper around zap.Any.
func RedactAny(key string, value any) zap.Field {
	return redact.Any(key, value)
}

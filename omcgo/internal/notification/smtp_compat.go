package notification

import (
	"os"
	"strconv"
	"strings"
)

// WithLegacySMTPEnvironment keeps existing OMC_SMTP_* deployments working
// until operators save the same channel through sys_config. Explicit YAML
// configuration always wins; the database remains the highest-priority source
// when DynamicEmailSender merges settings at send time.
func WithLegacySMTPEnvironment(options SMTPOptions) SMTPOptions {
	legacyHost := strings.TrimSpace(os.Getenv("OMC_SMTP_HOST"))
	if strings.TrimSpace(options.Host) != "" || legacyHost == "" {
		return options
	}
	options.Enabled = true
	options.Host = legacyHost
	options.Port = 25
	if port, err := strconv.Atoi(strings.TrimSpace(os.Getenv("OMC_SMTP_PORT"))); err == nil && port > 0 && port <= 65535 {
		options.Port = port
	}
	options.Username = strings.TrimSpace(os.Getenv("OMC_SMTP_USERNAME"))
	options.Password = os.Getenv("OMC_SMTP_PASSWORD")
	options.From = strings.TrimSpace(os.Getenv("OMC_SMTP_FROM"))
	options.AuthEnabled = options.Username != ""
	options.SecurityMode = SMTPSecurityNone
	if legacySMTPEnvTrue("OMC_SMTP_USE_TLS") {
		options.SecurityMode = SMTPSecurityImplicitTLS
	} else if legacySMTPEnvTrue("OMC_SMTP_USE_STARTTLS") {
		options.SecurityMode = SMTPSecuritySTARTTLS
	}
	return options
}

func legacySMTPEnvTrue(key string) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(key)))
	return err == nil && value
}

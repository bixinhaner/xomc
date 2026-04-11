package appconfig

import (
	"fmt"
	"strings"
)

// Validatable is implemented by config types that support self-validation.
type Validatable interface {
	Validate() error
}

// Validate checks AppConfig for common configuration errors.
// Returns a joined error listing all validation failures.
func (c *AppConfig) Validate() error {
	var errs []string

	if err := c.DB.validate("db"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.TSDB.validate("tsdb"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Redis.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.JWT.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Server.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Log.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Metrics.validate(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// Validate checks ACSConfig for common configuration errors.
func (c *ACSConfig) Validate() error {
	var errs []string

	if err := c.DB.validate("db"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Redis.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Server.validateACS(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Log.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Metrics.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if c.Session.Timeout > 0 && c.Session.MaxConcurrent <= 0 {
		errs = append(errs, "session.max_concurrent must be > 0 when session is configured")
	}
	if c.Auth.Mode != "" && c.Auth.Mode != "digest" && c.Auth.Mode != "basic" && c.Auth.Mode != "none" {
		errs = append(errs, fmt.Sprintf("auth.mode must be digest, basic, or none, got %q", c.Auth.Mode))
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// Validate checks WorkerConfig for common configuration errors.
func (c *WorkerConfig) Validate() error {
	var errs []string

	if err := c.DB.validate("db"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.TSDB.validate("tsdb"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Redis.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Log.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Metrics.validate(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func (c PostgresConfig) validate(prefix string) error {
	if c.DSN == "" {
		return fmt.Errorf("%s.dsn must not be empty", prefix)
	}
	if c.MaxConns > 0 && c.MinConns > c.MaxConns {
		return fmt.Errorf("%s.min_conns (%d) must not exceed max_conns (%d)", prefix, c.MinConns, c.MaxConns)
	}
	return nil
}

func (c RedisConfig) validate() error {
	if len(c.Addrs) == 0 {
		return fmt.Errorf("redis.addrs must not be empty")
	}
	return nil
}

func (c JWTConfig) validate() error {
	if c.Secret == "" {
		return fmt.Errorf("jwt.secret must not be empty")
	}
	if len(c.Secret) < 16 {
		return fmt.Errorf("jwt.secret must be at least 16 characters, got %d", len(c.Secret))
	}
	return nil
}

func (c AppServerConfig) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Port)
	}
	return nil
}

func (c ACSServerConfig) validateACS() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Port)
	}
	if c.TLS.Enabled {
		if c.TLSPort < 1 || c.TLSPort > 65535 {
			return fmt.Errorf("server.tls_port must be between 1 and 65535 when TLS enabled, got %d", c.TLSPort)
		}
	}
	return nil
}

func (c LogConfig) validate() error {
	switch c.Level {
	case "", "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log.level must be debug, info, warn, or error, got %q", c.Level)
	}
	switch c.Format {
	case "", "json", "console":
	default:
		return fmt.Errorf("log.format must be json or console, got %q", c.Format)
	}
	return nil
}

func (c MetricsConfig) validate() error {
	if c.Port != 0 && (c.Port < 1 || c.Port > 65535) {
		return fmt.Errorf("metrics.port must be between 1 and 65535, got %d", c.Port)
	}
	return nil
}

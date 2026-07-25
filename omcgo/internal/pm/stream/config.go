package stream

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Enabled             bool
	CloseGrace          time.Duration
	DailyCloseGrace     time.Duration
	WeeklyCloseGrace    time.Duration
	MonthlyCloseGrace   time.Duration
	OutboxBatch         int
	ConsumerConcurrency int
	FinalizeConcurrency int
	MaxEventBytes       int
	WindowTTL           time.Duration
}

func DefaultConfig() Config {
	return Config{
		Enabled:             true,
		CloseGrace:          5 * time.Minute,
		DailyCloseGrace:     15 * time.Minute,
		WeeklyCloseGrace:    30 * time.Minute,
		MonthlyCloseGrace:   30 * time.Minute,
		OutboxBatch:         100,
		ConsumerConcurrency: 8,
		FinalizeConcurrency: 4,
		MaxEventBytes:       8 << 20,
		WindowTTL:           45 * 24 * time.Hour,
	}
}

func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	cfg.Enabled = envBool("PM_AGGREGATION_ENABLED", cfg.Enabled)
	cfg.CloseGrace = envDuration("PM_AGGREGATION_CLOSE_GRACE", cfg.CloseGrace)
	cfg.DailyCloseGrace = envDuration("PM_AGGREGATION_DAILY_CLOSE_GRACE", cfg.DailyCloseGrace)
	cfg.WeeklyCloseGrace = envDuration("PM_AGGREGATION_WEEKLY_CLOSE_GRACE", cfg.WeeklyCloseGrace)
	cfg.MonthlyCloseGrace = envDuration("PM_AGGREGATION_MONTHLY_CLOSE_GRACE", cfg.MonthlyCloseGrace)
	cfg.OutboxBatch = envInt("PM_AGGREGATION_OUTBOX_BATCH", cfg.OutboxBatch)
	cfg.ConsumerConcurrency = envInt("PM_AGGREGATION_CONSUMER_CONCURRENCY", cfg.ConsumerConcurrency)
	cfg.FinalizeConcurrency = envInt("PM_AGGREGATION_FINALIZE_CONCURRENCY", cfg.FinalizeConcurrency)
	cfg.MaxEventBytes = envInt("PM_AGGREGATION_MAX_EVENT_BYTES", cfg.MaxEventBytes)
	cfg.WindowTTL = envDuration("PM_AGGREGATION_WINDOW_TTL", cfg.WindowTTL)
	return cfg
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

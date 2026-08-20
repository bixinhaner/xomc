package stream

import (
	"os"
	"strconv"
	"time"
)

const (
	minimumHourlyCloseGrace = 12 * time.Minute
	maximumCleanupBatch     = 5000
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
	OutboxRetention     time.Duration
	ReplayRetention     time.Duration
	CleanupBatch        int
	CleanupInterval     time.Duration
	CleanupMaxDuration  time.Duration
	CleanupVacuum       bool
	RedisV2WriteEnabled bool
}

func DefaultConfig() Config {
	return Config{
		Enabled:             true,
		CloseGrace:          12 * time.Minute,
		DailyCloseGrace:     15 * time.Minute,
		WeeklyCloseGrace:    30 * time.Minute,
		MonthlyCloseGrace:   30 * time.Minute,
		OutboxBatch:         100,
		ConsumerConcurrency: 8,
		FinalizeConcurrency: 32,
		MaxEventBytes:       8 << 20,
		WindowTTL:           45 * 24 * time.Hour,
		OutboxRetention:     24 * time.Hour,
		ReplayRetention:     45 * 24 * time.Hour,
		CleanupBatch:        500,
		CleanupInterval:     time.Hour,
		CleanupMaxDuration:  30 * time.Second,
		CleanupVacuum:       false,
		RedisV2WriteEnabled: true,
	}
}

func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	cfg.Enabled = envBool("PM_AGGREGATION_ENABLED", cfg.Enabled)
	cfg.CloseGrace = envDuration("PM_AGGREGATION_CLOSE_GRACE", cfg.CloseGrace)
	if cfg.CloseGrace < minimumHourlyCloseGrace {
		cfg.CloseGrace = minimumHourlyCloseGrace
	}
	cfg.DailyCloseGrace = envDuration("PM_AGGREGATION_DAILY_CLOSE_GRACE", cfg.DailyCloseGrace)
	cfg.WeeklyCloseGrace = envDuration("PM_AGGREGATION_WEEKLY_CLOSE_GRACE", cfg.WeeklyCloseGrace)
	cfg.MonthlyCloseGrace = envDuration("PM_AGGREGATION_MONTHLY_CLOSE_GRACE", cfg.MonthlyCloseGrace)
	cfg.OutboxBatch = envInt("PM_AGGREGATION_OUTBOX_BATCH", cfg.OutboxBatch)
	cfg.ConsumerConcurrency = envInt("PM_AGGREGATION_CONSUMER_CONCURRENCY", cfg.ConsumerConcurrency)
	cfg.FinalizeConcurrency = envInt("PM_AGGREGATION_FINALIZE_CONCURRENCY", cfg.FinalizeConcurrency)
	cfg.MaxEventBytes = envInt("PM_AGGREGATION_MAX_EVENT_BYTES", cfg.MaxEventBytes)
	cfg.WindowTTL = envDuration("PM_AGGREGATION_WINDOW_TTL", cfg.WindowTTL)
	cfg.OutboxRetention = boundedDuration(
		envDuration("PM_AGGREGATION_OUTBOX_RETENTION", cfg.OutboxRetention),
		time.Hour, 7*24*time.Hour,
	)
	cfg.ReplayRetention = boundedDuration(
		envDuration("PM_AGGREGATION_REPLAY_RETENTION", cfg.ReplayRetention),
		45*24*time.Hour, 90*24*time.Hour,
	)
	cfg.CleanupBatch = min(
		envInt("PM_AGGREGATION_CLEANUP_BATCH", cfg.CleanupBatch),
		maximumCleanupBatch,
	)
	cfg.CleanupInterval = boundedDuration(
		envDuration("PM_AGGREGATION_CLEANUP_INTERVAL", cfg.CleanupInterval),
		time.Minute, 24*time.Hour,
	)
	cfg.CleanupMaxDuration = boundedDuration(
		envDuration("PM_AGGREGATION_CLEANUP_MAX_DURATION", cfg.CleanupMaxDuration),
		time.Second, 5*time.Minute,
	)
	cfg.CleanupVacuum = envBool("PM_AGGREGATION_CLEANUP_VACUUM", cfg.CleanupVacuum)
	cfg.RedisV2WriteEnabled = envBool(
		"PM_AGGREGATION_REDIS_V2_WRITE_ENABLED", cfg.RedisV2WriteEnabled,
	)
	return cfg
}

func boundedDuration(value, minimum, maximum time.Duration) time.Duration {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
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

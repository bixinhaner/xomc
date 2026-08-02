package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

const defaultPMRedisTTLEpsilon = 2 * time.Second

type pmRedisDump struct {
	Key     string
	Payload []byte
	TTL     time.Duration
	Missing bool
}

type pmRedisStore interface {
	Scan(ctx context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error)
	Read(ctx context.Context, keys []string) ([]pmRedisDump, error)
	Restore(ctx context.Context, value pmRedisDump, replace bool) error
}

type pmRedisMigrationOptions struct {
	Pattern      string
	ScanCount    int64
	PipelineSize int
	DryRun       bool
	Replace      bool
	TTLEpsilon   time.Duration
}

type pmRedisMigrationResult struct {
	Scanned    int64 `json:"scanned"`
	Planned    int64 `json:"planned"`
	Copied     int64 `json:"copied"`
	Skipped    int64 `json:"skipped"`
	Conflicts  int64 `json:"conflicts"`
	Failed     int64 `json:"failed"`
	Verified   int64 `json:"verified"`
	SourceKeys int64 `json:"source_keys"`
	TargetKeys int64 `json:"target_keys"`
	DryRun     bool  `json:"dry_run"`
}

type pmRedisMigrator struct {
	source pmRedisStore
	target pmRedisStore
}

func (m *pmRedisMigrator) Migrate(ctx context.Context, opts pmRedisMigrationOptions) (pmRedisMigrationResult, error) {
	result := pmRedisMigrationResult{DryRun: opts.DryRun}
	if m.source == nil || m.target == nil {
		return result, errors.New("source and target Redis stores are required")
	}
	if opts.Pattern == "" || len(opts.Pattern) < len("pmagg:") || opts.Pattern[:len("pmagg:")] != "pmagg:" {
		return result, fmt.Errorf("unsafe pattern %q: only pmagg:* namespace may be migrated", opts.Pattern)
	}
	if opts.ScanCount <= 0 {
		return result, errors.New("scan-count must be greater than zero")
	}
	if opts.PipelineSize <= 0 {
		return result, errors.New("pipeline-size must be greater than zero")
	}
	if opts.TTLEpsilon <= 0 {
		opts.TTLEpsilon = defaultPMRedisTTLEpsilon
	}

	var cursor uint64
	for {
		keys, next, err := m.source.Scan(ctx, cursor, opts.Pattern, opts.ScanCount)
		if err != nil {
			result.Failed++
			return result, fmt.Errorf("scan source Redis: %w", err)
		}
		result.Scanned += int64(len(keys))
		for start := 0; start < len(keys); start += opts.PipelineSize {
			end := start + opts.PipelineSize
			if end > len(keys) {
				end = len(keys)
			}
			if err := m.migrateBatch(ctx, keys[start:end], opts, &result); err != nil {
				return result, err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}

	sourceKeys, err := scanPMRedisKeys(ctx, m.source, opts.Pattern, opts.ScanCount)
	if err != nil {
		result.Failed++
		return result, fmt.Errorf("count source Redis keys: %w", err)
	}
	targetKeys, err := scanPMRedisKeys(ctx, m.target, opts.Pattern, opts.ScanCount)
	if err != nil {
		result.Failed++
		return result, fmt.Errorf("count target Redis keys: %w", err)
	}
	result.SourceKeys = int64(len(sourceKeys))
	result.TargetKeys = int64(len(targetKeys))
	if opts.DryRun {
		return result, nil
	}
	if result.SourceKeys != result.TargetKeys {
		result.Failed++
		return result, fmt.Errorf("verification key count mismatch: source=%d target=%d", result.SourceKeys, result.TargetKeys)
	}
	if err := m.verify(ctx, sourceKeys, opts, &result); err != nil {
		result.Failed++
		return result, err
	}
	return result, nil
}

func (m *pmRedisMigrator) migrateBatch(ctx context.Context, keys []string, opts pmRedisMigrationOptions, result *pmRedisMigrationResult) error {
	sourceValues, err := m.source.Read(ctx, keys)
	if err != nil {
		result.Failed++
		return fmt.Errorf("read source Redis batch: %w", err)
	}
	targetValues, err := m.target.Read(ctx, keys)
	if err != nil {
		result.Failed++
		return fmt.Errorf("read target Redis batch: %w", err)
	}
	if len(sourceValues) != len(keys) || len(targetValues) != len(keys) {
		result.Failed++
		return errors.New("Redis batch result length mismatch")
	}

	for i, sourceValue := range sourceValues {
		if sourceValue.Missing {
			result.Skipped++
			continue
		}
		targetValue := targetValues[i]
		if !targetValue.Missing {
			samePayload := bytes.Equal(sourceValue.Payload, targetValue.Payload)
			sameTTL := pmRedisTTLMatches(sourceValue.TTL, targetValue.TTL, opts.TTLEpsilon)
			if samePayload && sameTTL {
				result.Skipped++
				continue
			}
			result.Conflicts++
			if !opts.Replace {
				result.Failed++
				return fmt.Errorf("target conflict for key %q (use --replace only after stopping PM writers)", sourceValue.Key)
			}
		}
		result.Planned++
		if opts.DryRun {
			continue
		}
		if err := m.target.Restore(ctx, sourceValue, !targetValue.Missing); err != nil {
			result.Failed++
			return fmt.Errorf("restore target key %q: %w", sourceValue.Key, err)
		}
		result.Copied++
	}
	return nil
}

func (m *pmRedisMigrator) verify(ctx context.Context, keys []string, opts pmRedisMigrationOptions, result *pmRedisMigrationResult) error {
	for start := 0; start < len(keys); start += opts.PipelineSize {
		end := start + opts.PipelineSize
		if end > len(keys) {
			end = len(keys)
		}
		sourceValues, err := m.source.Read(ctx, keys[start:end])
		if err != nil {
			return fmt.Errorf("verify source Redis: %w", err)
		}
		targetValues, err := m.target.Read(ctx, keys[start:end])
		if err != nil {
			return fmt.Errorf("verify target Redis: %w", err)
		}
		for i := range sourceValues {
			sourceValue, targetValue := sourceValues[i], targetValues[i]
			if sourceValue.Missing || targetValue.Missing {
				return fmt.Errorf("verification missing key %q", sourceValue.Key)
			}
			if !bytes.Equal(sourceValue.Payload, targetValue.Payload) {
				return fmt.Errorf("verification checksum mismatch for key %q", sourceValue.Key)
			}
			if !pmRedisTTLMatches(sourceValue.TTL, targetValue.TTL, opts.TTLEpsilon) {
				return fmt.Errorf("verification TTL mismatch for key %q", sourceValue.Key)
			}
			result.Verified++
		}
	}
	return nil
}

func scanPMRedisKeys(ctx context.Context, store pmRedisStore, pattern string, count int64) ([]string, error) {
	var cursor uint64
	var keys []string
	seen := make(map[string]struct{})
	for {
		batch, next, err := store.Scan(ctx, cursor, pattern, count)
		if err != nil {
			return nil, err
		}
		for _, key := range batch {
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
		cursor = next
		if cursor == 0 {
			return keys, nil
		}
	}
}

func pmRedisTTLMatches(a, b, epsilon time.Duration) bool {
	if a < 0 || b < 0 {
		return a == -1 && b == -1
	}
	delta := a - b
	if delta < 0 {
		delta = -delta
	}
	return delta <= epsilon
}

type goRedisPMStore struct {
	client redis.UniversalClient
}

func newGoRedisPMStore(client redis.UniversalClient) *goRedisPMStore {
	return &goRedisPMStore{client: client}
}

func (s *goRedisPMStore) Scan(ctx context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error) {
	return s.client.Scan(ctx, cursor, pattern, count).Result()
}

func (s *goRedisPMStore) Read(ctx context.Context, keys []string) ([]pmRedisDump, error) {
	pipe := s.client.Pipeline()
	ttlCommands := make([]*redis.DurationCmd, len(keys))
	dumpCommands := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		ttlCommands[i] = pipe.PTTL(ctx, key)
		dumpCommands[i] = pipe.Dump(ctx, key)
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	result := make([]pmRedisDump, len(keys))
	for i, key := range keys {
		result[i].Key = key
		ttl, err := ttlCommands[i].Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("PTTL key %q: %w", key, err)
		}
		payload, err := dumpCommands[i].Result()
		if errors.Is(err, redis.Nil) || ttl == -2 {
			result[i].Missing = true
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("DUMP key %q: %w", key, err)
		}
		result[i].TTL = ttl
		result[i].Payload = []byte(payload)
	}
	return result, nil
}

func (s *goRedisPMStore) Restore(ctx context.Context, value pmRedisDump, replace bool) error {
	ttl := value.TTL
	if ttl < 0 {
		ttl = 0
	}
	if replace {
		return s.client.RestoreReplace(ctx, value.Key, ttl, string(value.Payload)).Err()
	}
	return s.client.Restore(ctx, value.Key, ttl, string(value.Payload)).Err()
}

func newPMRedisCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "pm-redis", Short: "Safely migrate authoritative PM aggregation Redis state"}
	var sourceAddr, targetAddr, pattern string
	var scanCount int64
	var pipelineSize int
	var dryRun, replace bool
	migrate := &cobra.Command{
		Use:   "migrate",
		Short: "Copy pmagg:* state with DUMP/RESTORE and verify payload plus TTL",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if sourceAddr == targetAddr {
				return errors.New("source and target Redis addresses must differ")
			}
			sourceClient := redis.NewClient(&redis.Options{Addr: sourceAddr})
			targetClient := redis.NewClient(&redis.Options{Addr: targetAddr})
			defer sourceClient.Close()
			defer targetClient.Close()
			if err := sourceClient.Ping(cmd.Context()).Err(); err != nil {
				return fmt.Errorf("ping source Redis: %w", err)
			}
			if err := targetClient.Ping(cmd.Context()).Err(); err != nil {
				return fmt.Errorf("ping target Redis: %w", err)
			}
			result, err := (&pmRedisMigrator{
				source: newGoRedisPMStore(sourceClient), target: newGoRedisPMStore(targetClient),
			}).Migrate(cmd.Context(), pmRedisMigrationOptions{
				Pattern: pattern, ScanCount: scanCount, PipelineSize: pipelineSize,
				DryRun: dryRun, Replace: replace, TTLEpsilon: defaultPMRedisTTLEpsilon,
			})
			if flagOutput == "json" {
				if encodeErr := json.NewEncoder(cmd.OutOrStdout()).Encode(result); encodeErr != nil {
					return fmt.Errorf("encode migration result: %w", encodeErr)
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "scanned=%d planned=%d copied=%d skipped=%d conflicts=%d failed=%d verified=%d source_keys=%d target_keys=%d dry_run=%t\n",
					result.Scanned, result.Planned, result.Copied, result.Skipped, result.Conflicts,
					result.Failed, result.Verified, result.SourceKeys, result.TargetKeys, result.DryRun)
			}
			return err
		},
	}
	migrate.Flags().StringVar(&sourceAddr, "source", "redis-core:6379", "source Redis address")
	migrate.Flags().StringVar(&targetAddr, "target", "redis-pm:6379", "target Redis address")
	migrate.Flags().StringVar(&pattern, "pattern", "pmagg:*", "authoritative PM key pattern (must start with pmagg:)")
	migrate.Flags().Int64Var(&scanCount, "scan-count", 500, "Redis SCAN count hint")
	migrate.Flags().IntVar(&pipelineSize, "pipeline-size", 100, "PTTL/DUMP pipeline batch size")
	migrate.Flags().BoolVar(&dryRun, "dry-run", false, "inspect and report without writing target Redis")
	migrate.Flags().BoolVar(&replace, "replace", false, "replace conflicting target keys after PM writers are stopped")
	cmd.AddCommand(migrate)
	return cmd
}

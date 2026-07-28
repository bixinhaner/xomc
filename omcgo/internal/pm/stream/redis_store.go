package stream

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed accumulate.lua
var accumulateLua string

var (
	accumulateScript = redis.NewScript(accumulateLua)
	unlockScript     = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0`)
)

type AccumulateResult struct {
	Duplicate     bool
	ReceivedSlots int64
	Complete      bool
}

type Accumulator struct {
	Definition ContributionValue
	Sum        float64
	Count      int64
	Min        float64
	Max        float64
}

type WindowState struct {
	ExpectedSlots         int64
	ReceivedSlots         int64
	SourceExpectedSlots   int64
	SourceReceivedSlots   int64
	SourceIncompleteSlots int64
	Accumulators          []Accumulator
	Entities              map[string]EntityCompleteness
}

type EntityCompleteness struct {
	ReceivedSlots         int64
	SourceExpectedSlots   int64
	SourceReceivedSlots   int64
	SourceIncompleteSlots int64
}

type RedisWindowStore struct {
	client redis.UniversalClient
	ttl    time.Duration
}

func NewRedisWindowStore(client redis.UniversalClient, ttl time.Duration) *RedisWindowStore {
	if ttl <= 0 {
		ttl = 45 * 24 * time.Hour
	}
	return &RedisWindowStore{client: client, ttl: ttl}
}

func (s *RedisWindowStore) windowTTL(granularity Granularity) time.Duration {
	switch granularity {
	case GranularityHourly:
		return 4 * time.Hour
	case GranularityDaily:
		return 72 * time.Hour
	case GranularityWeekly:
		return 14 * 24 * time.Hour
	case GranularityMonthly:
		return 45 * 24 * time.Hour
	default:
		return s.ttl
	}
}

func (s *RedisWindowStore) ValidateConfiguration(ctx context.Context) error {
	if s.client == nil {
		return errors.New("PM aggregation Redis client is nil")
	}
	policy, err := s.client.ConfigGet(ctx, "maxmemory-policy").Result()
	if err != nil {
		return fmt.Errorf("read Redis maxmemory-policy: %w", err)
	}
	if policy["maxmemory-policy"] != "noeviction" {
		return fmt.Errorf("PM aggregation Redis requires maxmemory-policy=noeviction")
	}
	appendOnly, err := s.client.ConfigGet(ctx, "appendonly").Result()
	if err != nil {
		return fmt.Errorf("read Redis appendonly: %w", err)
	}
	if appendOnly["appendonly"] != "yes" {
		return fmt.Errorf("PM aggregation Redis requires appendonly=yes")
	}
	return nil
}

func (s *RedisWindowStore) Accumulate(
	ctx context.Context,
	contribution Contribution,
) (AccumulateResult, error) {
	if err := contribution.Validate(); err != nil {
		return AccumulateResult{}, err
	}
	shardCount := windowShardCount(contribution.ExpectedSlots, contribution.Key.Granularity)
	keys := redisKeys(contribution.Key, shardCount)
	args := []any{
		contribution.SourceFileID,
		contribution.DeviceID + "|" + contribution.SlotStart.UTC().Format(time.RFC3339Nano),
		contribution.ExpectedSlots,
		int64(s.windowTTL(contribution.Key.Granularity).Seconds()),
		len(contribution.Values),
		time.Now().UTC().Unix(),
		boolInt(contribution.Rollup),
		contribution.SourceExpectedSlots,
		contribution.SourceReceivedSlots,
		contribution.SourceIncompleteSlots,
		contribution.RollupChunkIndex,
		contribution.RollupChunkCount,
		shardCount,
	}
	for _, value := range contribution.Values {
		encoded, err := encodeDefinition(value)
		if err != nil {
			return AccumulateResult{}, err
		}
		definitionID, err := accumulatorDefinitionID(value)
		if err != nil {
			return AccumulateResult{}, err
		}
		entityKey := aggregationGroupKey(value)
		if entityKey == "" {
			entityKey = contribution.DeviceID
		}
		shard := windowShard(entityKey, shardCount)
		sum, count, minValue, maxValue := value.Value, int64(1), value.Value, value.Value
		if value.Composed {
			sum, count, minValue, maxValue = value.Sum, value.Count, value.Min, value.Max
		}
		args = append(args, definitionID, encoded, shard, sum, count, minValue, maxValue)
	}
	scriptKeys := []string{keys.seen, keys.slots, keys.meta, keys.entityMeta, keys.chunks}
	for shard := 0; shard < shardCount; shard++ {
		scriptKeys = append(scriptKeys, keys.acc[shard], keys.defs[shard])
	}
	raw, err := accumulateScript.Run(
		ctx, s.client,
		scriptKeys,
		args...,
	).Slice()
	if err != nil {
		return AccumulateResult{}, fmt.Errorf("accumulate PM aggregation window: %w", err)
	}
	if len(raw) != 3 {
		return AccumulateResult{}, fmt.Errorf("unexpected PM aggregation Lua result length %d", len(raw))
	}
	duplicate, err := redisInt64(raw[0])
	if err != nil {
		return AccumulateResult{}, err
	}
	received, err := redisInt64(raw[1])
	if err != nil {
		return AccumulateResult{}, err
	}
	complete, err := redisInt64(raw[2])
	if err != nil {
		return AccumulateResult{}, err
	}
	return AccumulateResult{
		Duplicate: duplicate == 1, ReceivedSlots: received, Complete: complete == 1,
	}, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (s *RedisWindowStore) Read(ctx context.Context, key WindowKey) (WindowState, error) {
	rootKeys := redisKeys(key, 1)
	meta, err := s.client.HMGet(
		ctx, rootKeys.meta,
		"expected_slots", "received_slots", "source_expected_slots",
		"source_received_slots", "source_incomplete_slots", "shard_count",
	).Result()
	if err != nil {
		return WindowState{}, fmt.Errorf("read PM aggregation Redis window metadata: %w", err)
	}
	if len(meta) != 6 || meta[0] == nil {
		return WindowState{}, redis.Nil
	}
	state := WindowState{Entities: make(map[string]EntityCompleteness)}
	state.ExpectedSlots = parseRedisInt(meta[0])
	state.ReceivedSlots = parseRedisInt(meta[1])
	state.SourceExpectedSlots = parseRedisInt(meta[2])
	state.SourceReceivedSlots = parseRedisInt(meta[3])
	state.SourceIncompleteSlots = parseRedisInt(meta[4])
	shardCount := int(parseRedisInt(meta[5]))
	if shardCount <= 0 || shardCount > maxWindowShards {
		shardCount = 1
	}
	keys := redisKeys(key, shardCount)
	var entityCursor uint64
	for {
		entityFields, next, scanErr := s.client.HScan(
			ctx, keys.entityMeta, entityCursor, "", 1000,
		).Result()
		if scanErr != nil {
			return WindowState{}, fmt.Errorf("scan PM aggregation entity completeness: %w", scanErr)
		}
		for index := 0; index+1 < len(entityFields); index += 2 {
			field, raw := entityFields[index], entityFields[index+1]
			suffixIndex := strings.LastIndexByte(field, '|')
			if suffixIndex <= 0 {
				continue
			}
			entityID, suffix := field[:suffixIndex], field[suffixIndex+1:]
			item := state.Entities[entityID]
			value, _ := strconv.ParseInt(raw, 10, 64)
			switch suffix {
			case "received":
				item.ReceivedSlots = value
			case "source_expected":
				item.SourceExpectedSlots = value
			case "source_received":
				item.SourceReceivedSlots = value
			case "source_incomplete":
				item.SourceIncompleteSlots = value
			}
			state.Entities[entityID] = item
		}
		entityCursor = next
		if entityCursor == 0 {
			break
		}
	}

	type partial struct {
		def   ContributionValue
		sum   float64
		count int64
		min   float64
		max   float64
	}
	values := make(map[string]*partial)
	for shard := 0; shard < shardCount; shard++ {
		var cursor uint64
		for {
			fields, next, scanErr := s.client.HScan(
				ctx, keys.acc[shard], cursor, "", 1000,
			).Result()
			if scanErr != nil {
				return WindowState{}, fmt.Errorf(
					"scan PM aggregation Redis window shard %d: %w", shard, scanErr,
				)
			}
			missingDefinitions := make([]string, 0)
			missingSet := make(map[string]struct{})
			for index := 0; index+1 < len(fields); index += 2 {
				field := fields[index]
				suffixIndex := strings.LastIndexByte(field, '|')
				if suffixIndex <= 0 {
					continue
				}
				base := field[:suffixIndex]
				if values[base] == nil {
					if _, exists := missingSet[base]; !exists {
						missingSet[base] = struct{}{}
						missingDefinitions = append(missingDefinitions, base)
					}
				}
			}
			if len(missingDefinitions) > 0 {
				definitions, getErr := s.client.HMGet(
					ctx, keys.defs[shard], missingDefinitions...,
				).Result()
				if getErr != nil {
					return WindowState{}, fmt.Errorf(
						"read PM aggregation accumulator definitions: %w", getErr,
					)
				}
				for index, base := range missingDefinitions {
					if definitions[index] == nil {
						return WindowState{}, fmt.Errorf(
							"PM aggregation accumulator definition %s is missing", base,
						)
					}
					encoded := fmt.Sprint(definitions[index])
					def, decodeErr := decodeDefinition(encoded)
					if decodeErr != nil {
						return WindowState{}, decodeErr
					}
					values[base] = &partial{def: def}
				}
			}
			for index := 0; index+1 < len(fields); index += 2 {
				field, raw := fields[index], fields[index+1]
				suffixIndex := strings.LastIndexByte(field, '|')
				if suffixIndex <= 0 {
					continue
				}
				base, suffix := field[:suffixIndex], field[suffixIndex+1:]
				item := values[base]
				switch suffix {
				case "sum":
					item.sum, _ = strconv.ParseFloat(raw, 64)
				case "count":
					item.count, _ = strconv.ParseInt(raw, 10, 64)
				case "min":
					item.min, _ = strconv.ParseFloat(raw, 64)
				case "max":
					item.max, _ = strconv.ParseFloat(raw, 64)
				}
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}
	for _, item := range values {
		state.Accumulators = append(state.Accumulators, Accumulator{
			Definition: item.def, Sum: item.sum, Count: item.count, Min: item.min, Max: item.max,
		})
	}
	return state, nil
}

func parseRedisInt(value any) int64 {
	switch typed := value.(type) {
	case string:
		parsed, _ := strconv.ParseInt(typed, 10, 64)
		return parsed
	case []byte:
		parsed, _ := strconv.ParseInt(string(typed), 10, 64)
		return parsed
	case int64:
		return typed
	default:
		return 0
	}
}

// Exists checks only the bounded window metadata. Recovery must not deserialize
// the accumulator payload merely to decide whether replay is necessary.
func (s *RedisWindowStore) Exists(ctx context.Context, key WindowKey) (bool, error) {
	keys := redisKeys(key, 1)
	exists, err := s.client.Exists(ctx, keys.meta).Result()
	if err != nil {
		return false, fmt.Errorf("check PM aggregation Redis window metadata: %w", err)
	}
	return exists > 0, nil
}

type Lock struct {
	client redis.UniversalClient
	key    string
	token  string
}

func (s *RedisWindowStore) TryFinalizeLock(
	ctx context.Context,
	key WindowKey,
	ttl time.Duration,
) (*Lock, error) {
	keys := redisKeys(key, 1)
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate PM aggregation lock token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	ok, err := s.client.SetNX(ctx, keys.lock, token, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("acquire PM aggregation finalize lock: %w", err)
	}
	if !ok {
		return nil, nil
	}
	return &Lock{client: s.client, key: keys.lock, token: token}, nil
}

func (l *Lock) Release(ctx context.Context) error {
	if l == nil {
		return nil
	}
	if _, err := unlockScript.Run(ctx, l.client, []string{l.key}, l.token).Result(); err != nil {
		return fmt.Errorf("release PM aggregation finalize lock: %w", err)
	}
	return nil
}

func (s *RedisWindowStore) Delete(ctx context.Context, key WindowKey) error {
	rootKeys := redisKeys(key, 1)
	shardCountValue, err := s.client.HGet(ctx, rootKeys.meta, "shard_count").Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("read PM aggregation shard count before delete: %w", err)
	}
	if shardCountValue <= 0 || shardCountValue > maxWindowShards {
		shardCountValue = 1
	}
	keys := redisKeys(key, shardCountValue)
	deleteKeys := []string{
		keys.seen, keys.slots, keys.meta, keys.entityMeta, keys.chunks, keys.lock,
	}
	deleteKeys = append(deleteKeys, keys.acc...)
	deleteKeys = append(deleteKeys, keys.defs...)
	if err := s.client.Del(ctx, deleteKeys...).Err(); err != nil {
		return fmt.Errorf("delete PM aggregation Redis window: %w", err)
	}
	return nil
}

type windowRedisKeys struct {
	seen, slots, meta, entityMeta, chunks, lock string
	acc, defs                                   []string
}

func redisKeys(key WindowKey, shardCount int) windowRedisKeys {
	if shardCount <= 0 {
		shardCount = 1
	}
	entityHash := sha256.Sum256([]byte(key.EntityKey))
	tag := fmt.Sprintf(
		"{%s:%s:%d:%s}",
		key.TaskVersionID.String(), key.Granularity, key.Start.UTC().Unix(),
		hex.EncodeToString(entityHash[:8]),
	)
	prefix := "pmagg:" + tag
	keys := windowRedisKeys{
		seen: prefix + ":seen", slots: prefix + ":slots",
		meta: prefix + ":meta", entityMeta: prefix + ":entities",
		chunks: prefix + ":chunks", lock: prefix + ":finalize-lock",
	}
	keys.acc = make([]string, shardCount)
	keys.defs = make([]string, shardCount)
	for shard := 0; shard < shardCount; shard++ {
		keys.acc[shard] = fmt.Sprintf("%s:acc:%03d", prefix, shard)
		keys.defs[shard] = fmt.Sprintf("%s:defs:%03d", prefix, shard)
	}
	return keys
}

func encodeDefinition(value ContributionValue) (string, error) {
	value.Value = 0
	value.Sum = 0
	value.Count = 0
	value.Min = 0
	value.Max = 0
	value.Composed = false
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal PM aggregation accumulator definition: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeDefinition(value string) (ContributionValue, error) {
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return ContributionValue{}, fmt.Errorf("decode PM aggregation accumulator key: %w", err)
	}
	var definition ContributionValue
	if err := json.Unmarshal(data, &definition); err != nil {
		return ContributionValue{}, fmt.Errorf("unmarshal PM aggregation accumulator definition: %w", err)
	}
	return definition, nil
}

func redisInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected Redis integer type %T", value)
	}
}

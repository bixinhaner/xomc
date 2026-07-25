package stream

import (
	"context"
	"crypto/rand"
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
	keys := redisKeys(contribution.Key)
	args := []any{
		contribution.SourceFileID,
		contribution.DeviceID + "|" + contribution.SlotStart.UTC().Format(time.RFC3339Nano),
		contribution.ExpectedSlots,
		int64(s.ttl.Seconds()),
		len(contribution.Values),
		time.Now().UTC().Unix(),
		boolInt(contribution.Rollup),
		contribution.SourceExpectedSlots,
		contribution.SourceReceivedSlots,
		contribution.SourceIncompleteSlots,
		contribution.RollupChunkIndex,
		contribution.RollupChunkCount,
	}
	for _, value := range contribution.Values {
		encoded, err := encodeDefinition(value)
		if err != nil {
			return AccumulateResult{}, err
		}
		sum, count, minValue, maxValue := value.Value, int64(1), value.Value, value.Value
		if value.Composed {
			sum, count, minValue, maxValue = value.Sum, value.Count, value.Min, value.Max
		}
		args = append(args, encoded, sum, count, minValue, maxValue)
	}
	raw, err := accumulateScript.Run(
		ctx, s.client,
		[]string{keys.seen, keys.slots, keys.meta, keys.acc, keys.chunks},
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
	keys := redisKeys(key)
	pipe := s.client.Pipeline()
	metaCmd := pipe.HGetAll(ctx, keys.meta)
	accCmd := pipe.HGetAll(ctx, keys.acc)
	if _, err := pipe.Exec(ctx); err != nil {
		return WindowState{}, fmt.Errorf("read PM aggregation Redis window: %w", err)
	}
	meta := metaCmd.Val()
	if len(meta) == 0 {
		return WindowState{}, redis.Nil
	}
	state := WindowState{}
	state.ExpectedSlots, _ = strconv.ParseInt(meta["expected_slots"], 10, 64)
	state.ReceivedSlots, _ = strconv.ParseInt(meta["received_slots"], 10, 64)
	state.SourceExpectedSlots, _ = strconv.ParseInt(meta["source_expected_slots"], 10, 64)
	state.SourceReceivedSlots, _ = strconv.ParseInt(meta["source_received_slots"], 10, 64)
	state.SourceIncompleteSlots, _ = strconv.ParseInt(meta["source_incomplete_slots"], 10, 64)

	type partial struct {
		def   ContributionValue
		sum   float64
		count int64
		min   float64
		max   float64
	}
	values := make(map[string]*partial)
	for field, raw := range accCmd.Val() {
		index := strings.LastIndexByte(field, '|')
		if index <= 0 {
			continue
		}
		base, suffix := field[:index], field[index+1:]
		item := values[base]
		if item == nil {
			def, err := decodeDefinition(base)
			if err != nil {
				return WindowState{}, err
			}
			item = &partial{def: def}
			values[base] = item
		}
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
	for _, item := range values {
		state.Accumulators = append(state.Accumulators, Accumulator{
			Definition: item.def, Sum: item.sum, Count: item.count, Min: item.min, Max: item.max,
		})
	}
	return state, nil
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
	keys := redisKeys(key)
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
	keys := redisKeys(key)
	if err := s.client.Del(ctx, keys.seen, keys.slots, keys.meta, keys.acc, keys.chunks).Err(); err != nil {
		return fmt.Errorf("delete PM aggregation Redis window: %w", err)
	}
	return nil
}

type windowRedisKeys struct {
	seen, slots, meta, acc, chunks, lock string
}

func redisKeys(key WindowKey) windowRedisKeys {
	tag := fmt.Sprintf(
		"{%s:%s:%d}",
		key.TaskVersionID.String(), key.Granularity, key.Start.UTC().Unix(),
	)
	prefix := "pmagg:" + tag
	return windowRedisKeys{
		seen: prefix + ":seen", slots: prefix + ":slots",
		meta: prefix + ":meta", acc: prefix + ":acc",
		chunks: prefix + ":chunks", lock: prefix + ":finalize-lock",
	}
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

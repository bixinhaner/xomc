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
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
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
	extendLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0`)
	initializeEmptyScript = redis.NewScript(`
if redis.call("GET", KEYS[2]) ~= ARGV[1] then
  return 0
end
if redis.call("EXISTS", KEYS[1]) == 0 then
  redis.call("HSET", KEYS[1],
    "expected_slots", 0,
    "received_slots", 0,
    "source_expected_slots", 0,
    "source_received_slots", 0,
    "source_incomplete_slots", 0,
    "shard_count", 1,
    "task_id", ARGV[3],
    "task_version_id", ARGV[4],
    "entity_key", ARGV[5],
    "granularity", ARGV[6],
    "window_start", ARGV[7],
    "window_end", ARGV[8],
    "updated_at_unix", ARGV[9])
end
redis.call("EXPIRE", KEYS[1], ARGV[2])
return 1`)
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

const (
	compactAccumulatorVersion       = "v2"
	legacyCompactAccumulatorVersion = "v1"
)

type compactAccumulator struct {
	Definition ContributionValue
	Sum        float64
	Count      int64
	Min        float64
	Max        float64
}

type accumulatorIdentity struct {
	Dimension     Dimension `json:"d"`
	DimensionKey  string    `json:"dk"`
	DimensionName string    `json:"dn,omitempty"`
	ObjectLDN     string    `json:"o,omitempty"`
	Technology    string    `json:"t,omitempty"`
}

type accumulatorMetricDefinition struct {
	MetricPath string        `json:"m"`
	MetricType string        `json:"mt"`
	Operation  AggregationOp `json:"op"`
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
	client   redis.UniversalClient
	ttl      time.Duration
	snapshot *SnapshotStore
	metrics  *Metrics

	definitionCache sync.Map
	definitionMu    sync.Mutex
	v2WriteEnabled  atomic.Bool
}

func NewRedisWindowStore(client redis.UniversalClient, ttl time.Duration) *RedisWindowStore {
	if ttl <= 0 {
		ttl = 45 * 24 * time.Hour
	}
	return &RedisWindowStore{client: client, ttl: ttl}
}

func (s *RedisWindowStore) SetSnapshot(snapshot *SnapshotStore) *RedisWindowStore {
	s.snapshot = snapshot
	return s
}

func (s *RedisWindowStore) SetMetrics(metrics *Metrics) *RedisWindowStore {
	s.metrics = metrics
	return s
}

func (s *RedisWindowStore) SetV2WriteEnabled(enabled bool) *RedisWindowStore {
	s.v2WriteEnabled.Store(enabled)
	return s
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

func (s *RedisWindowStore) definitionTTL() time.Duration {
	return max(s.ttl, 2*s.windowTTL(GranularityMonthly))
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
	return s.accumulate(ctx, contribution, "")
}

func (s *RedisWindowStore) accumulateWithLock(
	ctx context.Context,
	contribution Contribution,
	lock *Lock,
) (AccumulateResult, error) {
	if lock == nil {
		return AccumulateResult{}, errors.New("PM aggregation rebuild lock is nil")
	}
	return s.accumulate(ctx, contribution, lock.token)
}

// InitializeEmptyWithLock persists an authoritative zero-valued rebuild so
// open windows can continue receiving future rollups and published windows can
// be finalized through the same atomic replacement path as non-empty rebuilds.
func (s *RedisWindowStore) InitializeEmptyWithLock(
	ctx context.Context,
	key WindowKey,
	lock *Lock,
) error {
	if lock == nil {
		return errors.New("PM aggregation rebuild lock is nil")
	}
	keys := redisKeys(key, 1)
	if lock.key != keys.lock {
		return errors.New("PM aggregation rebuild lock belongs to another window")
	}
	result, err := initializeEmptyScript.Run(
		ctx,
		s.client,
		[]string{keys.meta, keys.lock},
		lock.token,
		int64(s.windowTTL(key.Granularity).Seconds()),
		key.TaskID.String(),
		key.TaskVersionID.String(),
		key.EntityKey,
		string(key.Granularity),
		key.Start.UTC().Format(time.RFC3339Nano),
		key.End.UTC().Format(time.RFC3339Nano),
		time.Now().UTC().Unix(),
	).Int64()
	if err != nil {
		s.recordRedisWriteError()
		return fmt.Errorf("initialize empty PM aggregation rebuild window: %w", err)
	}
	if result != 1 {
		return errors.New("PM aggregation rebuild lock ownership lost")
	}
	return nil
}

func (s *RedisWindowStore) accumulate(
	ctx context.Context,
	contribution Contribution,
	lockToken string,
) (AccumulateResult, error) {
	if err := contribution.Validate(); err != nil {
		return AccumulateResult{}, err
	}
	shardCount := windowShardCount(contribution.ExpectedSlots, contribution.Key.Granularity)
	keys := redisKeys(contribution.Key, shardCount)
	writeV2 := s.v2WriteEnabled.Load()
	versionDefinitions := redisVersionDefinitionsKey(contribution.Key.TaskVersionID)
	pendingDefinitions := make(map[string]any)
	identities := make(map[string]string)
	type encodedValue struct {
		v2ID, legacyID, legacyDefinition string
		shard                            int
		sum, min, max                    float64
		count                            int64
	}
	encodedValues := make([]encodedValue, 0, len(contribution.Values))
	deviceOUI, deviceSN := "", ""
	for _, value := range contribution.Values {
		if value.DeviceOUI != "" || value.DeviceSN != "" {
			deviceOUI, deviceSN = value.DeviceOUI, value.DeviceSN
			break
		}
	}
	fixedArgs := []any{
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
		contribution.Key.TaskID.String(),
		contribution.Key.TaskVersionID.String(),
		contribution.Key.EntityKey,
		string(contribution.Key.Granularity),
		contribution.Key.Start.UTC().Format(time.RFC3339Nano),
		contribution.Key.End.UTC().Format(time.RFC3339Nano),
		deviceOUI,
		deviceSN,
	}
	for _, value := range contribution.Values {
		legacyID, err := accumulatorDefinitionID(value)
		if err != nil {
			return AccumulateResult{}, err
		}
		identityID, identity, err := encodeAccumulatorIdentity(value)
		if err != nil {
			return AccumulateResult{}, err
		}
		legacyDefinition, err := encodeDefinition(value)
		if err != nil {
			return AccumulateResult{}, err
		}
		metricID, metricDefinition, err := encodeAccumulatorMetricDefinition(value)
		if err != nil {
			return AccumulateResult{}, err
		}
		if writeV2 {
			cacheKey := contribution.Key.TaskVersionID.String() + "|" + metricID
			if _, loaded := s.definitionCache.Load(cacheKey); !loaded {
				pendingDefinitions[metricID] = metricDefinition
			}
			identities[identityID] = identity
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
		encodedValues = append(encodedValues, encodedValue{
			v2ID: identityID + "|" + metricID, legacyID: legacyID,
			legacyDefinition: legacyDefinition, shard: shard,
			sum: sum, count: count, min: minValue, max: maxValue,
		})
	}
	args := append(fixedArgs, boolInt(writeV2), len(identities))
	identityIDs := make([]string, 0, len(identities))
	for identityID := range identities {
		identityIDs = append(identityIDs, identityID)
	}
	sort.Strings(identityIDs)
	for _, identityID := range identityIDs {
		args = append(args, identityID, identities[identityID])
	}
	for _, value := range encodedValues {
		args = append(
			args, value.v2ID, value.legacyID, value.legacyDefinition,
			value.shard, value.sum, value.count, value.min, value.max,
		)
	}
	args = append(args, lockToken)
	if writeV2 && len(pendingDefinitions) > 0 {
		if err := s.ensureVersionDefinitions(
			ctx, contribution.Key.TaskVersionID, versionDefinitions, pendingDefinitions,
		); err != nil {
			return AccumulateResult{}, err
		}
	}
	scriptKeys := []string{
		keys.seen, keys.slots, keys.meta, keys.entityMeta, keys.chunks, keys.identities,
	}
	for shard := 0; shard < shardCount; shard++ {
		scriptKeys = append(scriptKeys, keys.acc[shard], keys.defs[shard])
	}
	scriptKeys = append(scriptKeys, keys.lock)
	raw, err := accumulateScript.Run(
		ctx, s.client,
		scriptKeys,
		args...,
	).Slice()
	if err != nil {
		s.recordRedisWriteError()
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

func (s *RedisWindowStore) ensureVersionDefinitions(
	ctx context.Context,
	versionID uuid.UUID,
	redisKey string,
	definitions map[string]any,
) error {
	s.definitionMu.Lock()
	defer s.definitionMu.Unlock()
	pending := make(map[string]any, len(definitions))
	for metricID, definition := range definitions {
		cacheKey := versionID.String() + "|" + metricID
		if _, loaded := s.definitionCache.Load(cacheKey); !loaded {
			pending[metricID] = definition
		}
	}
	if len(pending) == 0 {
		return nil
	}
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, redisKey, pending)
	pipe.Expire(ctx, redisKey, s.definitionTTL())
	if _, err := pipe.Exec(ctx); err != nil {
		s.recordRedisWriteError()
		return fmt.Errorf("store PM aggregation task-version metric definitions: %w", err)
	}
	for metricID := range pending {
		s.definitionCache.Store(versionID.String()+"|"+metricID, struct{}{})
	}
	return nil
}

func (s *RedisWindowStore) recordRedisWriteError() {
	if s.metrics != nil {
		s.metrics.RedisWriteErrorsTotal.Inc()
	}
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
		"device_oui", "device_sn",
	).Result()
	if err != nil {
		return WindowState{}, fmt.Errorf("read PM aggregation Redis window metadata: %w", err)
	}
	if len(meta) != 8 || meta[0] == nil {
		return WindowState{}, redis.Nil
	}
	state := WindowState{Entities: make(map[string]EntityCompleteness)}
	state.ExpectedSlots = parseRedisInt(meta[0])
	state.ReceivedSlots = parseRedisInt(meta[1])
	state.SourceExpectedSlots = parseRedisInt(meta[2])
	state.SourceReceivedSlots = parseRedisInt(meta[3])
	state.SourceIncompleteSlots = parseRedisInt(meta[4])
	shardCount := int(parseRedisInt(meta[5]))
	deviceOUI, deviceSN := fmt.Sprint(meta[6]), fmt.Sprint(meta[7])
	if meta[6] == nil {
		deviceOUI = ""
	}
	if meta[7] == nil {
		deviceSN = ""
	}
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

	identities, err := s.readAccumulatorIdentities(ctx, keys.identities)
	if err != nil {
		return WindowState{}, err
	}
	metricDefinitions, err := s.readAccumulatorMetricDefinitions(ctx, key.TaskVersionID)
	if err != nil {
		return WindowState{}, err
	}
	values := make(map[string]*compactAccumulator)
	for shard := 0; shard < shardCount; shard++ {
		var cursor uint64
		legacyNumeric := make(map[string]*compactAccumulator)
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
				field, raw := fields[index], fields[index+1]
				if strings.HasPrefix(raw, compactAccumulatorVersion+"|") {
					definition, decodeErr := decodeV2AccumulatorDefinition(
						field, identities, metricDefinitions, deviceOUI, deviceSN,
					)
					if decodeErr != nil {
						return WindowState{}, decodeErr
					}
					sum, count, minValue, maxValue, decodeErr :=
						decodeCompactAccumulatorV2(raw)
					if decodeErr != nil {
						return WindowState{}, fmt.Errorf(
							"decode v2 PM aggregation accumulator %s: %w", field, decodeErr,
						)
					}
					mergeCompactAccumulator(values, compactAccumulator{
						Definition: definition, Sum: sum, Count: count,
						Min: minValue, Max: maxValue,
					})
					continue
				}
				if strings.HasPrefix(raw, legacyCompactAccumulatorVersion+"|") {
					item, decodeErr := decodeCompactAccumulator(raw)
					if decodeErr != nil {
						return WindowState{}, fmt.Errorf(
							"decode legacy compact PM aggregation accumulator %s: %w",
							field, decodeErr,
						)
					}
					mergeCompactAccumulator(values, item)
					continue
				}
				suffixIndex := strings.LastIndexByte(field, '|')
				if suffixIndex <= 0 {
					continue
				}
				base, suffix := field[:suffixIndex], field[suffixIndex+1:]
				if suffix != "sum" && suffix != "count" &&
					suffix != "min" && suffix != "max" {
					continue
				}
				if legacyNumeric[base] == nil {
					legacyNumeric[base] = &compactAccumulator{}
					if _, exists := missingSet[base]; !exists {
						missingSet[base] = struct{}{}
						missingDefinitions = append(missingDefinitions, base)
					}
				}
				item := legacyNumeric[base]
				switch suffix {
				case "sum":
					item.Sum, _ = strconv.ParseFloat(raw, 64)
				case "count":
					item.Count, _ = strconv.ParseInt(raw, 10, 64)
				case "min":
					item.Min, _ = strconv.ParseFloat(raw, 64)
				case "max":
					item.Max, _ = strconv.ParseFloat(raw, 64)
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
					def, decodeErr := decodeDefinition(fmt.Sprint(definitions[index]))
					if decodeErr != nil {
						return WindowState{}, decodeErr
					}
					legacyNumeric[base].Definition = def
				}
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
		for _, item := range legacyNumeric {
			mergeCompactAccumulator(values, *item)
		}
	}
	for _, item := range values {
		state.Accumulators = append(state.Accumulators, Accumulator{
			Definition: item.Definition, Sum: item.Sum, Count: item.Count,
			Min: item.Min, Max: item.Max,
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

func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	if l == nil {
		return errors.New("PM aggregation lock is nil")
	}
	result, err := extendLockScript.Run(
		ctx, l.client, []string{l.key}, l.token, ttl.Milliseconds(),
	).Int64()
	if err != nil {
		return fmt.Errorf("extend PM aggregation lock: %w", err)
	}
	if result != 1 {
		return errors.New("PM aggregation lock ownership lost")
	}
	return nil
}

func (s *RedisWindowStore) Delete(ctx context.Context, key WindowKey) error {
	return s.delete(ctx, key, true)
}

func (s *RedisWindowStore) DeleteState(ctx context.Context, key WindowKey) error {
	return s.delete(ctx, key, false)
}

func (s *RedisWindowStore) DeleteVersionDefinitions(
	ctx context.Context,
	versionIDs []uuid.UUID,
) error {
	ids := normalizeTaskVersionIDs(versionIDs)
	const batchSize = 128
	for start := 0; start < len(ids); start += batchSize {
		end := min(start+batchSize, len(ids))
		keys := make([]string, 0, end-start)
		for _, versionID := range ids[start:end] {
			keys = append(keys, redisVersionDefinitionsKey(versionID))
		}
		if err := s.client.Unlink(ctx, keys...).Err(); err != nil {
			s.recordRedisWriteError()
			return fmt.Errorf("unlink PM aggregation Redis version definitions: %w", err)
		}
	}
	return nil
}

func (s *RedisWindowStore) delete(ctx context.Context, key WindowKey, includeLock bool) error {
	return s.unlink(ctx, key, includeLock, 128)
}

func (s *RedisWindowStore) unlink(
	ctx context.Context,
	key WindowKey,
	includeLock bool,
	batchSize int,
) error {
	return s.unlinkFenced(ctx, key, includeLock, batchSize, nil)
}

func (s *RedisWindowStore) unlinkFenced(
	ctx context.Context,
	key WindowKey,
	includeLock bool,
	batchSize int,
	beforeBatch func(context.Context) error,
) error {
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
		keys.seen, keys.slots, keys.entityMeta, keys.chunks, keys.identities,
	}
	if includeLock {
		deleteKeys = append(deleteKeys, keys.lock)
	}
	deleteKeys = append(deleteKeys, keys.acc...)
	deleteKeys = append(deleteKeys, keys.defs...)
	if batchSize <= 0 {
		batchSize = 128
	}
	for start := 0; start < len(deleteKeys); start += batchSize {
		end := min(start+batchSize, len(deleteKeys))
		if beforeBatch != nil {
			if err := beforeBatch(ctx); err != nil {
				return err
			}
		}
		if err := s.client.Unlink(ctx, deleteKeys[start:end]...).Err(); err != nil {
			s.recordRedisWriteError()
			return fmt.Errorf("unlink PM aggregation Redis window: %w", err)
		}
	}
	if beforeBatch != nil {
		if err := beforeBatch(ctx); err != nil {
			return err
		}
	}
	if err := s.client.Unlink(ctx, keys.meta).Err(); err != nil {
		s.recordRedisWriteError()
		return fmt.Errorf("unlink PM aggregation Redis window metadata: %w", err)
	}
	return nil
}

type windowRedisKeys struct {
	seen, slots, meta, entityMeta, chunks, identities, lock string
	acc, defs                                               []string
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
		chunks: prefix + ":chunks", identities: prefix + ":identities",
		lock: prefix + ":finalize-lock",
	}
	keys.acc = make([]string, shardCount)
	keys.defs = make([]string, shardCount)
	for shard := 0; shard < shardCount; shard++ {
		keys.acc[shard] = fmt.Sprintf("%s:acc:%03d", prefix, shard)
		keys.defs[shard] = fmt.Sprintf("%s:defs:%03d", prefix, shard)
	}
	return keys
}

func redisVersionDefinitionsKey(versionID uuid.UUID) string {
	return fmt.Sprintf("pmagg:{%s}:definitions:v2", versionID)
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

func decodeCompactAccumulator(raw string) (compactAccumulator, error) {
	parts := strings.Split(raw, "|")
	if len(parts) != 6 {
		return compactAccumulator{}, fmt.Errorf(
			"invalid compact accumulator field count %d", len(parts),
		)
	}
	if parts[0] != legacyCompactAccumulatorVersion {
		return compactAccumulator{}, fmt.Errorf("unsupported compact accumulator version %q", parts[0])
	}
	definition, err := decodeDefinition(parts[1])
	if err != nil {
		return compactAccumulator{}, err
	}
	sum, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return compactAccumulator{}, fmt.Errorf("parse compact accumulator sum: %w", err)
	}
	count, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return compactAccumulator{}, fmt.Errorf("parse compact accumulator count: %w", err)
	}
	minValue, err := strconv.ParseFloat(parts[4], 64)
	if err != nil {
		return compactAccumulator{}, fmt.Errorf("parse compact accumulator min: %w", err)
	}
	maxValue, err := strconv.ParseFloat(parts[5], 64)
	if err != nil {
		return compactAccumulator{}, fmt.Errorf("parse compact accumulator max: %w", err)
	}
	return compactAccumulator{
		Definition: definition,
		Sum:        sum,
		Count:      count,
		Min:        minValue,
		Max:        maxValue,
	}, nil
}

func encodeCompactAccumulatorV2(sum float64, count int64, minValue, maxValue float64) string {
	return strings.Join([]string{
		compactAccumulatorVersion,
		strconv.FormatFloat(sum, 'g', 17, 64),
		strconv.FormatInt(count, 10),
		strconv.FormatFloat(minValue, 'g', 17, 64),
		strconv.FormatFloat(maxValue, 'g', 17, 64),
	}, "|")
}

func decodeCompactAccumulatorV2(raw string) (float64, int64, float64, float64, error) {
	parts := strings.Split(raw, "|")
	if len(parts) != 5 || parts[0] != compactAccumulatorVersion {
		return 0, 0, 0, 0, fmt.Errorf("invalid v2 compact accumulator")
	}
	sum, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse v2 accumulator sum: %w", err)
	}
	count, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse v2 accumulator count: %w", err)
	}
	minValue, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse v2 accumulator min: %w", err)
	}
	maxValue, err := strconv.ParseFloat(parts[4], 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse v2 accumulator max: %w", err)
	}
	return sum, count, minValue, maxValue, nil
}

func encodeAccumulatorIdentity(value ContributionValue) (string, string, error) {
	identity := accumulatorIdentity{
		Dimension: value.Dimension, DimensionKey: value.DimensionKey,
		DimensionName: value.DimensionName, ObjectLDN: value.ObjectLDN,
		Technology: value.Technology,
	}
	return encodeAccumulatorMetadata(identity, 12)
}

func encodeAccumulatorMetricDefinition(value ContributionValue) (string, string, error) {
	definition := accumulatorMetricDefinition{
		MetricPath: value.MetricPath, MetricType: value.MetricType, Operation: value.Operation,
	}
	return encodeAccumulatorMetadata(definition, 8)
}

func encodeAccumulatorMetadata(value any, idBytes int) (string, string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", "", fmt.Errorf("marshal PM aggregation compact metadata: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:idBytes]),
		base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeAccumulatorMetadata(encoded string, target any) error {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("decode PM aggregation compact metadata: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal PM aggregation compact metadata: %w", err)
	}
	return nil
}

func (s *RedisWindowStore) readAccumulatorIdentities(
	ctx context.Context,
	key string,
) (map[string]accumulatorIdentity, error) {
	result := make(map[string]accumulatorIdentity)
	var cursor uint64
	for {
		fields, next, err := s.client.HScan(ctx, key, cursor, "", 1000).Result()
		if err != nil {
			return nil, fmt.Errorf("scan PM aggregation window identities: %w", err)
		}
		for index := 0; index+1 < len(fields); index += 2 {
			var identity accumulatorIdentity
			if err := decodeAccumulatorMetadata(fields[index+1], &identity); err != nil {
				return nil, err
			}
			result[fields[index]] = identity
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return result, nil
}

func (s *RedisWindowStore) readAccumulatorMetricDefinitions(
	ctx context.Context,
	versionID uuid.UUID,
) (map[string]accumulatorMetricDefinition, error) {
	raw, err := s.client.HGetAll(ctx, redisVersionDefinitionsKey(versionID)).Result()
	if err != nil {
		return nil, fmt.Errorf("read PM aggregation task-version definitions: %w", err)
	}
	result := make(map[string]accumulatorMetricDefinition, len(raw))
	for id, encoded := range raw {
		var definition accumulatorMetricDefinition
		if err := decodeAccumulatorMetadata(encoded, &definition); err != nil {
			return nil, err
		}
		result[id] = definition
	}
	if s.snapshot == nil || s.snapshot.Current() == nil {
		return result, nil
	}
	version := s.snapshot.Current().ByVersion[versionID]
	if version == nil {
		return result, nil
	}
	for path, rule := range version.Counters {
		value := ContributionValue{
			MetricPath: path, MetricType: "counter", Operation: rule.Aggregation,
		}
		id, _, encodeErr := encodeAccumulatorMetricDefinition(value)
		if encodeErr != nil {
			return nil, encodeErr
		}
		if _, exists := result[id]; !exists {
			result[id] = accumulatorMetricDefinition{
				MetricPath: path, MetricType: "counter", Operation: rule.Aggregation,
			}
		}
	}
	return result, nil
}

func decodeV2AccumulatorDefinition(
	field string,
	identities map[string]accumulatorIdentity,
	metrics map[string]accumulatorMetricDefinition,
	deviceOUI string,
	deviceSN string,
) (ContributionValue, error) {
	parts := strings.Split(field, "|")
	if len(parts) != 2 {
		return ContributionValue{}, fmt.Errorf("invalid v2 PM accumulator identity %q", field)
	}
	identity, ok := identities[parts[0]]
	if !ok {
		return ContributionValue{}, fmt.Errorf("v2 PM accumulator identity %s is missing", parts[0])
	}
	metric, ok := metrics[parts[1]]
	if !ok {
		return ContributionValue{}, fmt.Errorf("v2 PM accumulator metric %s is missing", parts[1])
	}
	definition := ContributionValue{
		Dimension: identity.Dimension, DimensionKey: identity.DimensionKey,
		DimensionName: identity.DimensionName, ObjectLDN: identity.ObjectLDN,
		Technology: identity.Technology, MetricPath: metric.MetricPath,
		MetricType: metric.MetricType, Operation: metric.Operation,
	}
	if identity.Dimension == DimensionDevice {
		definition.DeviceOUI, definition.DeviceSN = deviceOUI, deviceSN
	}
	return definition, nil
}

func mergeCompactAccumulator(
	values map[string]*compactAccumulator,
	incoming compactAccumulator,
) {
	id, err := accumulatorDefinitionID(incoming.Definition)
	if err != nil {
		return
	}
	current := values[id]
	if current == nil {
		copy := incoming
		values[id] = &copy
		return
	}
	if incoming.Count <= 0 {
		return
	}
	if current.Count <= 0 {
		current.Min, current.Max = incoming.Min, incoming.Max
	} else {
		current.Min = min(current.Min, incoming.Min)
		current.Max = max(current.Max, incoming.Max)
	}
	current.Sum += incoming.Sum
	current.Count += incoming.Count
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

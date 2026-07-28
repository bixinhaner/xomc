package stream

import (
	"crypto/sha256"
	"encoding/hex"
	"hash/fnv"
)

const (
	targetEntitiesPerShard = int64(1024)
	maxWindowShards        = 256
)

func windowShardCount(expectedSlots int64, granularity Granularity) int {
	childWindows := int64(1)
	switch granularity {
	case GranularityHourly:
		childWindows = 4
	case GranularityDaily:
		childWindows = 24
	case GranularityWeekly:
		childWindows = 7
	case GranularityMonthly:
		childWindows = 31
	}
	entities := (expectedSlots + childWindows - 1) / childWindows
	required := (entities + targetEntitiesPerShard - 1) / targetEntitiesPerShard
	shards := 1
	for int64(shards) < required && shards < maxWindowShards {
		shards <<= 1
	}
	return shards
}

func windowShard(entityKey string, shardCount int) int {
	if shardCount <= 1 {
		return 0
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(entityKey))
	return int(hash.Sum32() % uint32(shardCount))
}

func accumulatorDefinitionID(value ContributionValue) (string, error) {
	encoded, err := encodeDefinition(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(encoded))
	return hex.EncodeToString(sum[:12]), nil
}

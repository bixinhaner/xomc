local seen = KEYS[1]
local slots = KEYS[2]
local meta = KEYS[3]
local entity_meta = KEYS[4]
local chunks = KEYS[5]
local identities = KEYS[6]

local source_file_id = ARGV[1]
local slot_id = ARGV[2]
local expected_slots = tonumber(ARGV[3])
local ttl_seconds = tonumber(ARGV[4])
local value_count = tonumber(ARGV[5])
local rollup = tonumber(ARGV[7])
local source_expected_delta = tonumber(ARGV[8])
local source_received_delta = tonumber(ARGV[9])
local source_incomplete_delta = tonumber(ARGV[10])
local chunk_index = tonumber(ARGV[11])
local chunk_count = tonumber(ARGV[12])
local shard_count = tonumber(ARGV[13])
local lock_token = ARGV[22 + value_count * 8]
local entity_id = string.match(slot_id, "^(.-)|") or slot_id
local lock_key = KEYS[7 + shard_count * 2]

local active_lock = redis.call("GET", lock_key)
if active_lock and active_lock ~= lock_token then
  return redis.error_reply("PM_AGGREGATION_WINDOW_LOCKED")
end

local function format_number(value)
  return string.format("%.17g", value)
end

local function decode_legacy_compact(value, definition_id)
  local version, definition, sum, count, min_value, max_value =
    string.match(value, "^([^|]+)|([^|]+)|([^|]+)|([^|]+)|([^|]+)|([^|]+)$")
  if version ~= "v1" or not definition or not tonumber(sum) or
      not tonumber(count) or not tonumber(min_value) or not tonumber(max_value) then
    error("invalid compact PM accumulator " .. definition_id)
  end
  return definition, tonumber(sum), tonumber(count), tonumber(min_value), tonumber(max_value)
end

local function decode_v2(value, definition_id)
  local version, sum, count, min_value, max_value =
    string.match(value, "^([^|]+)|([^|]+)|([^|]+)|([^|]+)|([^|]+)$")
  if version ~= "v2" or not tonumber(sum) or not tonumber(count) or
      not tonumber(min_value) or not tonumber(max_value) then
    error("invalid v2 compact PM accumulator " .. definition_id)
  end
  return tonumber(sum), tonumber(count), tonumber(min_value), tonumber(max_value)
end

if redis.call("SADD", seen, source_file_id) == 0 then
  local received = tonumber(redis.call("HGET", meta, "received_slots") or "0")
  return {1, received, received >= expected_slots and 1 or 0}
end

local slot_added = 0
if rollup == 1 then
  local chunk_id = slot_id .. "|" .. tostring(chunk_index)
  if redis.call("SADD", chunks, chunk_id) == 1 then
    local chunk_meta = "chunk_count|" .. slot_id
    local received_chunks = redis.call("HINCRBY", meta, chunk_meta, 1)
    if received_chunks >= chunk_count then
      slot_added = redis.call("SADD", slots, slot_id)
    end
  end
else
  slot_added = redis.call("SADD", slots, slot_id)
end
local received_slots = redis.call("SCARD", slots)
redis.call("HSET", meta,
  "expected_slots", expected_slots,
  "received_slots", received_slots,
  "shard_count", shard_count,
  "task_id", ARGV[14],
  "task_version_id", ARGV[15],
  "entity_key", ARGV[16],
  "granularity", ARGV[17],
  "window_start", ARGV[18],
  "window_end", ARGV[19],
  "updated_at_unix", ARGV[6])
if ARGV[20] ~= "" then
  redis.call("HSETNX", meta, "device_oui", ARGV[20])
end
if ARGV[21] ~= "" then
  redis.call("HSETNX", meta, "device_sn", ARGV[21])
end
if rollup == 1 then
  local source_meta = "source_seen|" .. slot_id
  if redis.call("HSETNX", meta, source_meta, 1) == 1 then
    redis.call("HINCRBY", meta, "source_expected_slots", source_expected_delta)
    redis.call("HINCRBY", meta, "source_received_slots", source_received_delta)
    redis.call("HINCRBY", meta, "source_incomplete_slots", source_incomplete_delta)
  end
else
  redis.call("HSET", meta,
    "source_expected_slots", expected_slots,
    "source_received_slots", received_slots,
    "source_incomplete_slots", 0)
end
if slot_added == 1 then
  local entity_received = redis.call("HINCRBY", entity_meta, entity_id .. "|received", 1)
  if rollup == 1 then
    redis.call("HINCRBY", entity_meta, entity_id .. "|source_expected", source_expected_delta)
    redis.call("HINCRBY", entity_meta, entity_id .. "|source_received", source_received_delta)
    redis.call("HINCRBY", entity_meta, entity_id .. "|source_incomplete", source_incomplete_delta)
  else
    redis.call("HSET", entity_meta,
      entity_id .. "|source_expected", expected_slots,
      entity_id .. "|source_received", entity_received)
  end
end

local offset = 22
for i = 1, value_count do
  local base = ARGV[offset]
  local legacy_base = ARGV[offset + 1]
  local identity = ARGV[offset + 2]
  local shard = tonumber(ARGV[offset + 3])
  local value_sum = tonumber(ARGV[offset + 4])
  local value_count_delta = tonumber(ARGV[offset + 5])
  local value_min = tonumber(ARGV[offset + 6])
  local value_max = tonumber(ARGV[offset + 7])
  local acc = KEYS[7 + shard * 2]
  local defs = KEYS[8 + shard * 2]
  local sum_key = legacy_base .. "|sum"
  local count_key = legacy_base .. "|count"
  local min_key = legacy_base .. "|min"
  local max_key = legacy_base .. "|max"
  local identity_id = string.match(base, "^([^|]+)|")

  local current = redis.call("HGET", acc, base)
  local current_sum = 0
  local current_count = 0
  local current_min = value_min
  local current_max = value_max
  if current then
    current_sum, current_count, current_min, current_max =
      decode_v2(current, base)
  end

  -- A rolling upgrade can leave v1 and v2 writers active concurrently.
  -- Fold every v1 representation into v2 on each new write. If an old
  -- writer runs afterwards, the Go reader still merges both representations.
  local legacy_compact = redis.call("HGET", acc, legacy_base)
  if legacy_compact then
    local ignored_definition
    local legacy_sum
    local legacy_count
    local legacy_min
    local legacy_max
    ignored_definition, legacy_sum, legacy_count, legacy_min, legacy_max =
      decode_legacy_compact(legacy_compact, legacy_base)
    if current_count == 0 then
      current_min = legacy_min
      current_max = legacy_max
    else
      current_min = math.min(current_min, legacy_min)
      current_max = math.max(current_max, legacy_max)
    end
    current_sum = current_sum + legacy_sum
    current_count = current_count + legacy_count
  end
  local legacy_count = tonumber(redis.call("HGET", acc, count_key) or "0")
  if legacy_count > 0 then
    local legacy_sum = tonumber(redis.call("HGET", acc, sum_key) or "0")
    local legacy_min = tonumber(redis.call("HGET", acc, min_key) or tostring(value_min))
    local legacy_max = tonumber(redis.call("HGET", acc, max_key) or tostring(value_max))
    if current_count == 0 then
      current_min = legacy_min
      current_max = legacy_max
    else
      current_min = math.min(current_min, legacy_min)
      current_max = math.max(current_max, legacy_max)
    end
    current_sum = current_sum + legacy_sum
    current_count = current_count + legacy_count
  end

  local next_sum = current_sum + value_sum
  local next_count = current_count + value_count_delta
  local next_min = current_min
  local next_max = current_max
  if value_min < next_min then
    next_min = value_min
  end
  if value_max > next_max then
    next_max = value_max
  end
  local packed = "v2|" .. format_number(next_sum) ..
    "|" .. tostring(next_count) ..
    "|" .. format_number(next_min) ..
    "|" .. format_number(next_max)
  redis.call("HSETNX", identities, identity_id, identity)
  redis.call("HSET", acc, base, packed)
  redis.call("HDEL", acc, legacy_base, sum_key, count_key, min_key, max_key)
  redis.call("HDEL", defs, legacy_base)
  offset = offset + 8
end

redis.call("EXPIRE", seen, ttl_seconds)
redis.call("EXPIRE", slots, ttl_seconds)
redis.call("EXPIRE", meta, ttl_seconds)
redis.call("EXPIRE", entity_meta, ttl_seconds)
redis.call("EXPIRE", chunks, ttl_seconds)
redis.call("EXPIRE", identities, ttl_seconds)
for shard = 0, shard_count - 1 do
  redis.call("EXPIRE", KEYS[7 + shard * 2], ttl_seconds)
  redis.call("EXPIRE", KEYS[8 + shard * 2], ttl_seconds)
end

return {0, received_slots, received_slots >= expected_slots and 1 or 0}

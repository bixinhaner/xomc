local seen = KEYS[1]
local slots = KEYS[2]
local meta = KEYS[3]
local entity_meta = KEYS[4]
local chunks = KEYS[5]

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
local entity_id = string.match(slot_id, "^(.-)|") or slot_id

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
  "updated_at_unix", ARGV[6])
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

local offset = 14
for i = 1, value_count do
  local base = ARGV[offset]
  local definition = ARGV[offset + 1]
  local shard = tonumber(ARGV[offset + 2])
  local value_sum = tonumber(ARGV[offset + 3])
  local value_count_delta = tonumber(ARGV[offset + 4])
  local value_min = tonumber(ARGV[offset + 5])
  local value_max = tonumber(ARGV[offset + 6])
  local acc = KEYS[6 + shard * 2]
  local defs = KEYS[7 + shard * 2]
  local sum_key = base .. "|sum"
  local count_key = base .. "|count"
  local min_key = base .. "|min"
  local max_key = base .. "|max"

  redis.call("HSETNX", defs, base, definition)
  redis.call("HINCRBYFLOAT", acc, sum_key, value_sum)
  redis.call("HINCRBY", acc, count_key, value_count_delta)
  local current_min = redis.call("HGET", acc, min_key)
  if not current_min or value_min < tonumber(current_min) then
    redis.call("HSET", acc, min_key, value_min)
  end
  local current_max = redis.call("HGET", acc, max_key)
  if not current_max or value_max > tonumber(current_max) then
    redis.call("HSET", acc, max_key, value_max)
  end
  offset = offset + 7
end

redis.call("EXPIRE", seen, ttl_seconds)
redis.call("EXPIRE", slots, ttl_seconds)
redis.call("EXPIRE", meta, ttl_seconds)
redis.call("EXPIRE", entity_meta, ttl_seconds)
redis.call("EXPIRE", chunks, ttl_seconds)
for shard = 0, shard_count - 1 do
  redis.call("EXPIRE", KEYS[6 + shard * 2], ttl_seconds)
  redis.call("EXPIRE", KEYS[7 + shard * 2], ttl_seconds)
end

return {0, received_slots, received_slots >= expected_slots and 1 or 0}

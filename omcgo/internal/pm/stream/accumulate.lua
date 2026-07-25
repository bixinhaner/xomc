local seen = KEYS[1]
local slots = KEYS[2]
local meta = KEYS[3]
local acc = KEYS[4]

local source_file_id = ARGV[1]
local slot_id = ARGV[2]
local expected_slots = tonumber(ARGV[3])
local ttl_seconds = tonumber(ARGV[4])
local value_count = tonumber(ARGV[5])

if redis.call("SADD", seen, source_file_id) == 0 then
  local received = tonumber(redis.call("HGET", meta, "received_slots") or "0")
  return {1, received, received >= expected_slots and 1 or 0}
end

redis.call("SADD", slots, slot_id)
local received_slots = redis.call("SCARD", slots)
redis.call("HSET", meta,
  "expected_slots", expected_slots,
  "received_slots", received_slots,
  "updated_at_unix", ARGV[6])

local offset = 7
for i = 1, value_count do
  local base = ARGV[offset]
  local value = tonumber(ARGV[offset + 1])
  local sum_key = base .. "|sum"
  local count_key = base .. "|count"
  local min_key = base .. "|min"
  local max_key = base .. "|max"

  redis.call("HINCRBYFLOAT", acc, sum_key, value)
  redis.call("HINCRBY", acc, count_key, 1)
  local current_min = redis.call("HGET", acc, min_key)
  if not current_min or value < tonumber(current_min) then
    redis.call("HSET", acc, min_key, value)
  end
  local current_max = redis.call("HGET", acc, max_key)
  if not current_max or value > tonumber(current_max) then
    redis.call("HSET", acc, max_key, value)
  end
  offset = offset + 2
end

redis.call("EXPIRE", seen, ttl_seconds)
redis.call("EXPIRE", slots, ttl_seconds)
redis.call("EXPIRE", meta, ttl_seconds)
redis.call("EXPIRE", acc, ttl_seconds)

return {0, received_slots, received_slots >= expected_slots and 1 or 0}

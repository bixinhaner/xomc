package retention

import "errors"

// ErrRetentionTooShort 保留天数小于 MinRetentionDays（默认 1 天）。
var ErrRetentionTooShort = errors.New("retention days < min (1 day)")

// ErrRetentionTooLong 保留天数大于 MaxRetentionDays（默认 3650 = 10 年）。
var ErrRetentionTooLong = errors.New("retention days > max (3650 days / 10 years)")

// ErrUnknownPolicyKey sys_configs 中查到未知的保留策略 key。
var ErrUnknownPolicyKey = errors.New("unknown pm.retention policy key")

// ErrInvalidValueType sys_configs 行的 value_type 不是 int。
var ErrInvalidValueType = errors.New("pm.retention sys_configs value_type must be int")

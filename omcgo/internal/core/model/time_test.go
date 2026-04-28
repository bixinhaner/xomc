package model

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Time_MarshalJSON_Zero(t *testing.T) {
	var v Time
	b, err := v.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `""`, string(b))
}

func Test_Time_MarshalJSON_NonZero(t *testing.T) {
	tt := time.Date(2026, 4, 24, 15, 29, 3, 0, time.Local)
	v := Time(tt)

	b, err := v.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"2026/4/24 15:29:03"`, string(b))
}

func Test_Time_UnmarshalJSON_Null(t *testing.T) {
	var v Time
	err := v.UnmarshalJSON([]byte("null"))
	require.NoError(t, err)
	assert.True(t, v.IsZero())
}

func Test_Time_UnmarshalJSON_EmptyString(t *testing.T) {
	var v Time
	err := v.UnmarshalJSON([]byte(`""`))
	require.NoError(t, err)
	assert.True(t, v.IsZero())
}

func Test_Time_UnmarshalJSON_CustomFormat(t *testing.T) {
	var v Time
	err := v.UnmarshalJSON([]byte(`"2026/4/24 15:29:03"`))
	require.NoError(t, err)
	assert.False(t, v.IsZero())

	std := v.Std()
	assert.Equal(t, 2026, std.Year())
	assert.Equal(t, time.April, std.Month())
	assert.Equal(t, 24, std.Day())
}

func Test_Time_UnmarshalJSON_RFC3339Fallback(t *testing.T) {
	var v Time
	err := v.UnmarshalJSON([]byte(`"2026-04-24T15:29:03Z"`))
	require.NoError(t, err)
	assert.False(t, v.IsZero())
}

func Test_Time_UnmarshalJSON_InvalidFormat(t *testing.T) {
	var v Time
	err := v.UnmarshalJSON([]byte(`"not-a-time"`))
	require.Error(t, err)
}

func Test_Time_RoundTrip(t *testing.T) {
	tt := time.Date(2026, 4, 24, 15, 29, 3, 0, time.Local)
	v := Time(tt)

	b, err := json.Marshal(v)
	require.NoError(t, err)

	var v2 Time
	err = json.Unmarshal(b, &v2)
	require.NoError(t, err)
	assert.Equal(t, v.Std().Format(TimeFormat), v2.Std().Format(TimeFormat))
}

func Test_Time_Scan_Nil(t *testing.T) {
	var v Time
	v = Time(time.Now())
	err := v.Scan(nil)
	require.NoError(t, err)
	assert.True(t, v.IsZero())
}

func Test_Time_Scan_TimeValue(t *testing.T) {
	now := time.Now()
	var v Time
	err := v.Scan(now)
	require.NoError(t, err)
	assert.Equal(t, now.Unix(), v.Std().Unix())
}

func Test_Time_Scan_UnsupportedType(t *testing.T) {
	var v Time
	err := v.Scan("string-value")
	require.Error(t, err)
}

func Test_Time_Value(t *testing.T) {
	now := time.Now()
	v := Time(now)

	val, err := v.Value()
	require.NoError(t, err)

	std, ok := val.(time.Time)
	require.True(t, ok)
	assert.Equal(t, now.Unix(), std.Unix())

	// Confirm Value satisfies driver.Valuer.
	var _ driver.Valuer = v
}

func Test_NowTime_NotZero(t *testing.T) {
	v := NowTime()
	assert.False(t, v.IsZero())
}

func Test_Time_IsZero_True(t *testing.T) {
	var v Time
	assert.True(t, v.IsZero())
}

func Test_Time_IsZero_False(t *testing.T) {
	v := Time(time.Now())
	assert.False(t, v.IsZero())
}

func Test_Time_Std_Reflects(t *testing.T) {
	now := time.Now()
	v := Time(now)
	assert.Equal(t, now, v.Std())
}

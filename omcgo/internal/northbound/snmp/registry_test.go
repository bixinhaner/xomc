package snmp

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTarget(name string, enabled bool) *TrapTarget {
	return &TrapTarget{
		OSSName:   name,
		Host:      "127.0.0.1",
		Port:      162,
		Version:   VersionV2c,
		Community: "public",
		Carrier:   "cmcc",
		Timeout:   2 * time.Second,
		Enabled:   enabled,
	}
}

func TestInMemoryRegistry_AddRemoveListGet(t *testing.T) {
	reg := NewInMemoryRegistry()
	assert.Empty(t, reg.List())

	added, err := reg.Add(newTestTarget("osr-cmcc-bj", true))
	require.NoError(t, err)
	require.NotEmpty(t, added.ID, "Add should auto-assign ID when empty")

	got, ok := reg.Get(added.ID)
	require.True(t, ok)
	assert.Equal(t, "osr-cmcc-bj", got.OSSName)

	// defensive copy: mutating the returned target must not leak into store
	got.OSSName = "tampered"
	roundTrip, _ := reg.Get(added.ID)
	assert.Equal(t, "osr-cmcc-bj", roundTrip.OSSName, "Get must return defensive copy")

	// list contains exactly one entry
	all := reg.List()
	assert.Len(t, all, 1)

	// remove
	assert.True(t, reg.Remove(added.ID))
	assert.False(t, reg.Remove(added.ID), "double-remove should report not-found")
	assert.Empty(t, reg.List())
}

func TestInMemoryRegistry_DuplicateGuards(t *testing.T) {
	reg := NewInMemoryRegistry()

	first, err := reg.Add(newTestTarget("osr-cmcc", true))
	require.NoError(t, err)

	// duplicate oss_name → ErrDuplicateOSSName
	_, err = reg.Add(newTestTarget("osr-cmcc", true))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateOSSName))

	// duplicate explicit ID → ErrDuplicateID
	dup := newTestTarget("osr-other", true)
	dup.ID = first.ID
	_, err = reg.Add(dup)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateID))

	// nil target → error (not a panic)
	_, err = reg.Add(nil)
	require.Error(t, err)

	// missing OSSName → error
	_, err = reg.Add(&TrapTarget{Host: "h", Port: 162, Version: VersionV2c, Community: "x"})
	require.Error(t, err)
}

func TestInMemoryRegistry_Update(t *testing.T) {
	reg := NewInMemoryRegistry()
	added, err := reg.Add(newTestTarget("osr-1", true))
	require.NoError(t, err)

	// update without ID → error
	require.Error(t, reg.Update(&TrapTarget{OSSName: "x"}))

	// update non-existent → ErrTargetNotFound
	err = reg.Update(&TrapTarget{ID: "ghost", OSSName: "x"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTargetNotFound))

	// rename works and oss_name index moves
	renamed := *added
	renamed.OSSName = "osr-1-renamed"
	require.NoError(t, reg.Update(&renamed))
	got, _ := reg.Get(added.ID)
	assert.Equal(t, "osr-1-renamed", got.OSSName)

	// rename collision is rejected
	_, err = reg.Add(newTestTarget("osr-2", true))
	require.NoError(t, err)
	collision := renamed
	collision.OSSName = "osr-2"
	err = reg.Update(&collision)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateOSSName))
}

func TestInMemoryRegistry_ListEnabled_OnlyEnabled(t *testing.T) {
	reg := NewInMemoryRegistry()
	_, _ = reg.Add(newTestTarget("a-on", true))
	_, _ = reg.Add(newTestTarget("b-off", false))
	_, _ = reg.Add(newTestTarget("c-on", true))

	enabled := reg.ListEnabled()
	require.Len(t, enabled, 2)
	// stable ordering: List is sorted by OSSName, ListEnabled inherits that.
	assert.Equal(t, "a-on", enabled[0].OSSName)
	assert.Equal(t, "c-on", enabled[1].OSSName)

	// total list still has all three
	assert.Len(t, reg.List(), 3)
}

func TestTrapTarget_String_RedactsSensitive(t *testing.T) {
	tt := &TrapTarget{
		ID:           "id-1",
		OSSName:      "osr",
		Host:         "127.0.0.1",
		Port:         162,
		Version:      VersionV3,
		Community:    "secret-community",
		AuthPassword: "auth-secret-DO-NOT-LOG",
		PrivPassword: "priv-secret-DO-NOT-LOG",
		Carrier:      "ctcc",
		Enabled:      true,
	}
	s := tt.String()
	assert.NotContains(t, s, "secret-community", "community must not appear in String()")
	assert.NotContains(t, s, "auth-secret", "auth password must not appear in String()")
	assert.NotContains(t, s, "priv-secret", "priv password must not appear in String()")
	assert.Contains(t, s, "id-1")
	assert.Contains(t, s, "osr")

	// nil receiver is safe
	var nilT *TrapTarget
	assert.Equal(t, "<nil TrapTarget>", nilT.String())
}

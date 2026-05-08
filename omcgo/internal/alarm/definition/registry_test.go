package definition

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRepo struct {
	defs    []ResolvedDefinition
	listErr error
	calls   int
}

func (f *fakeRepo) ListAll(_ context.Context) ([]ResolvedDefinition, error) {
	f.calls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]ResolvedDefinition, len(f.defs))
	copy(out, f.defs)
	return out, nil
}

func (f *fakeRepo) ListSeverityLevels(_ context.Context) ([]SeverityLevel, error) {
	return nil, nil
}

func mkDef(identifier, neType string, sev int) ResolvedDefinition {
	return ResolvedDefinition{
		AlarmDefinition: AlarmDefinition{
			ID:         uuid.New(),
			Identifier: identifier,
			NeType:     neType,
			SeverityID: uuid.New(),
			IsShow:     true,
		},
		SeverityCode: sev,
		SeverityName: "Major",
	}
}

func TestRegistry_Refresh_HappyPath(t *testing.T) {
	repo := &fakeRepo{defs: []ResolvedDefinition{
		mkDef("a.1.1", "ENB", 31002),
		mkDef("a.1.2", "ENB", 31001),
	}}
	r := NewRegistry(repo, nil, zap.NewNop())

	require.False(t, r.Loaded())
	err := r.Refresh(context.Background())
	require.NoError(t, err)
	assert.True(t, r.Loaded())
	assert.Equal(t, 2, r.Count())
	assert.Equal(t, 1, repo.calls)
}

func TestRegistry_Lookup_HitAndMiss(t *testing.T) {
	repo := &fakeRepo{defs: []ResolvedDefinition{mkDef("a.1.1", "ENB", 31002)}}
	r := NewRegistry(repo, nil, zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	got, err := r.Lookup(context.Background(), "a.1.1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "a.1.1", got.Identifier)
	assert.Equal(t, 31002, got.SeverityCode)

	got, err = r.Lookup(context.Background(), "missing")
	require.ErrorIs(t, err, ErrUnknownIdentifier)
	assert.Nil(t, got)
}

func TestRegistry_Refresh_ReplacesOldEntries(t *testing.T) {
	repo := &fakeRepo{defs: []ResolvedDefinition{mkDef("a.1.1", "ENB", 31002)}}
	r := NewRegistry(repo, nil, zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))
	assert.Equal(t, 1, r.Count())

	repo.defs = []ResolvedDefinition{mkDef("a.2.2", "GNB", 31001)}
	require.NoError(t, r.Refresh(context.Background()))
	assert.Equal(t, 1, r.Count())

	_, err := r.Lookup(context.Background(), "a.1.1")
	assert.ErrorIs(t, err, ErrUnknownIdentifier, "刷新后旧条目应失效")

	got, err := r.Lookup(context.Background(), "a.2.2")
	require.NoError(t, err)
	assert.Equal(t, "GNB", got.NeType)
}

func TestRegistry_Refresh_RepoError(t *testing.T) {
	repo := &fakeRepo{listErr: errors.New("db down")}
	r := NewRegistry(repo, nil, zap.NewNop())

	err := r.Refresh(context.Background())
	require.Error(t, err)
	assert.False(t, r.Loaded())
}

func TestRegistry_NilDefaults(t *testing.T) {
	r := NewRegistry(&fakeRepo{}, nil, nil)
	require.NoError(t, r.Refresh(context.Background()))
	assert.True(t, r.Loaded())
	assert.Equal(t, 0, r.Count())
}

func TestRegistry_Metrics_HitAndMiss(t *testing.T) {
	repo := &fakeRepo{defs: []ResolvedDefinition{mkDef("known", "ENB", 31002)}}
	r := NewRegistry(repo, nil, zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	_, _ = r.Lookup(context.Background(), "known")
	_, _ = r.Lookup(context.Background(), "unknown")

	m := r.Metrics()
	require.NotNil(t, m)
	// 间接验证 fallback 计数器可用
	m.UnknownDropped()
	m.UnknownKept()
}

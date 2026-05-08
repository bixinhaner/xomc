package dictloader

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLoader struct {
	name     string
	dir      string
	loadOnce func(context.Context) (Report, error)
	reload   func(context.Context) (Report, error)
}

func (f *fakeLoader) Name() string                                 { return f.name }
func (f *fakeLoader) Directory() string                            { return f.dir }
func (f *fakeLoader) LoadOnce(ctx context.Context) (Report, error) { return f.loadOnce(ctx) }
func (f *fakeLoader) Reload(ctx context.Context) (Report, error)   { return f.reload(ctx) }

func newFakeLoader(name string, rep Report, err error) *fakeLoader {
	return &fakeLoader{
		name:     name,
		dir:      name,
		loadOnce: func(context.Context) (Report, error) { return rep, err },
		reload:   func(context.Context) (Report, error) { return rep, err },
	}
}

func TestRegistry_Register(t *testing.T) {
	reg := NewRegistry(2, nil)
	require.NoError(t, reg.Register(newFakeLoader("a", Report{LoaderName: "a"}, nil)))
	require.NoError(t, reg.Register(newFakeLoader("b", Report{LoaderName: "b"}, nil)))

	err := reg.Register(newFakeLoader("a", Report{}, nil))
	require.ErrorIs(t, err, ErrDuplicateLoader)

	err = reg.Register(nil)
	require.Error(t, err)

	err = reg.Register(&fakeLoader{
		name:     "",
		loadOnce: func(context.Context) (Report, error) { return Report{}, nil },
		reload:   func(context.Context) (Report, error) { return Report{}, nil },
	})
	require.Error(t, err)

	assert.Equal(t, []string{"a", "b"}, reg.Names())
}

func TestRegistry_LoadAll_Success(t *testing.T) {
	reg := NewRegistry(4, nil)
	rep := Report{LoaderName: "x", FilesLoaded: 3, RowsAffected: 7}
	require.NoError(t, reg.Register(newFakeLoader("x", rep, nil)))
	require.NoError(t, reg.Register(newFakeLoader("y", Report{LoaderName: "y", FilesLoaded: 1}, nil)))

	results, err := reg.LoadAll(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, 7, results["x"].RowsAffected)

	snap := reg.Snapshot()
	assert.Equal(t, results["x"], snap["x"])
	assert.Equal(t, results["y"], snap["y"])
}

func TestRegistry_LoadAll_PartialFailure(t *testing.T) {
	reg := NewRegistry(4, nil)
	require.NoError(t, reg.Register(newFakeLoader("ok", Report{LoaderName: "ok"}, nil)))
	require.NoError(t, reg.Register(newFakeLoader("boom", Report{LoaderName: "boom"}, errors.New("disk full"))))

	results, err := reg.LoadAll(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
	// Both reports recorded even on failure.
	assert.Contains(t, results, "ok")
	assert.Contains(t, results, "boom")
}

func TestRegistry_LoadAll_Parallelism(t *testing.T) {
	reg := NewRegistry(3, nil)
	var (
		active int32
		peak   int32
	)
	slowLoader := func(name string) Loader {
		return &fakeLoader{
			name: name,
			dir:  name,
			loadOnce: func(context.Context) (Report, error) {
				cur := atomic.AddInt32(&active, 1)
				for {
					p := atomic.LoadInt32(&peak)
					if cur <= p || atomic.CompareAndSwapInt32(&peak, p, cur) {
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
				atomic.AddInt32(&active, -1)
				return Report{LoaderName: name}, nil
			},
			reload: func(context.Context) (Report, error) { return Report{LoaderName: name}, nil },
		}
	}
	for i := 0; i < 5; i++ {
		require.NoError(t, reg.Register(slowLoader(fmt.Sprintf("ld-%d", i))))
	}

	_, err := reg.LoadAll(context.Background())
	require.NoError(t, err)
	final := atomic.LoadInt32(&peak)
	assert.LessOrEqual(t, int(final), 3, "concurrency cap exceeded")
	assert.GreaterOrEqual(t, int(final), 2, "expected actual parallelism")
}

func TestRegistry_ReloadOne(t *testing.T) {
	reg := NewRegistry(2, nil)
	rep := Report{LoaderName: "x", RowsAffected: 5}
	require.NoError(t, reg.Register(newFakeLoader("x", rep, nil)))

	got, err := reg.ReloadOne(context.Background(), "x")
	require.NoError(t, err)
	assert.Equal(t, 5, got.RowsAffected)

	_, err = reg.ReloadOne(context.Background(), "missing")
	require.ErrorIs(t, err, ErrUnknownLoader)
}

func TestRegistry_ReloadOne_Failure(t *testing.T) {
	reg := NewRegistry(2, nil)
	want := errors.New("oops")
	require.NoError(t, reg.Register(newFakeLoader("x", Report{LoaderName: "x"}, want)))

	_, err := reg.ReloadOne(context.Background(), "x")
	require.Error(t, err)
	assert.ErrorIs(t, err, want)
}

func TestRegistry_NewRegistry_Defaults(t *testing.T) {
	reg := NewRegistry(0, nil)
	require.NotNil(t, reg)
	require.NoError(t, reg.Register(newFakeLoader("a", Report{}, nil)))
}

func TestRegistry_Snapshot_IsCopy(t *testing.T) {
	reg := NewRegistry(2, nil)
	require.NoError(t, reg.Register(newFakeLoader("a", Report{LoaderName: "a", RowsAffected: 1}, nil)))

	_, _ = reg.LoadAll(context.Background())
	snap := reg.Snapshot()
	snap["a"] = Report{LoaderName: "tampered"}

	again := reg.Snapshot()
	assert.Equal(t, "a", again["a"].LoaderName)
}

func TestRegistry_Concurrent_Register_LoadAll(t *testing.T) {
	// Sanity: registering and loading from multiple goroutines does not panic
	// and reports a deterministic snapshot.
	reg := NewRegistry(4, nil)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reg.Register(newFakeLoader(fmt.Sprintf("l-%d", i), Report{LoaderName: fmt.Sprintf("l-%d", i)}, nil))
		}()
	}
	wg.Wait()
	results, err := reg.LoadAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, results, 10)
}

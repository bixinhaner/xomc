package dictloader

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Loader is implemented by each dictionary domain (param-model / indicator /
// alarm-definition / product). Concrete implementations live with their
// respective business packages (T-0098 P1-06).
type Loader interface {
	// Name identifies the loader (e.g. "param-model"). Must be unique within a Registry.
	Name() string
	// Directory returns the path scanned by this loader, relative to DictLoaderConfig.XMLBaseDir.
	Directory() string
	// LoadOnce performs the initial startup load. Implementations should be
	// idempotent w.r.t. concurrent re-entry within the same process.
	LoadOnce(ctx context.Context) (Report, error)
	// Reload re-scans and applies changes. Implementations may diff against
	// their internal cache to skip unchanged files.
	Reload(ctx context.Context) (Report, error)
}

// ErrUnknownLoader is returned by Registry.ReloadOne when no loader is registered under the given name.
var ErrUnknownLoader = errors.New("unknown loader")

// ErrDuplicateLoader is returned by Registry.Register when a loader with the same Name is already registered.
var ErrDuplicateLoader = errors.New("duplicate loader")

// Registry orchestrates a set of Loaders. It enforces unique names, runs
// LoadAll with bounded parallelism, and snapshots the most recent Report
// per loader for diagnostics.
type Registry struct {
	concurrency int
	logger      *zap.Logger

	mu      sync.RWMutex
	loaders map[string]Loader
	order   []string
	reports map[string]Report
}

// NewRegistry constructs a Registry. concurrency <= 0 defaults to 4. A nil logger uses zap.NewNop.
func NewRegistry(concurrency int, logger *zap.Logger) *Registry {
	if concurrency <= 0 {
		concurrency = 4
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Registry{
		concurrency: concurrency,
		logger:      logger,
		loaders:     make(map[string]Loader),
		reports:     make(map[string]Report),
	}
}

// Register adds a loader. Returns ErrDuplicateLoader on name collision.
func (r *Registry) Register(ld Loader) error {
	if ld == nil {
		return errors.New("dictloader: nil loader")
	}
	name := ld.Name()
	if name == "" {
		return errors.New("dictloader: loader name must be non-empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.loaders[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateLoader, name)
	}
	r.loaders[name] = ld
	r.order = append(r.order, name)
	return nil
}

// Names returns registered loader names in registration order.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// LoadAll calls LoadOnce on every registered loader, bounded by concurrency.
// Every loader's Report is recorded in the snapshot regardless of error;
// the first error encountered is returned (other loaders still complete).
func (r *Registry) LoadAll(ctx context.Context) (map[string]Report, error) {
	r.mu.RLock()
	loaders := make([]Loader, 0, len(r.order))
	for _, name := range r.order {
		loaders = append(loaders, r.loaders[name])
	}
	r.mu.RUnlock()

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(r.concurrency)

	var (
		resultMu sync.Mutex
		results  = make(map[string]Report, len(loaders))
	)

	for _, ld := range loaders {
		ld := ld
		g.Go(func() error {
			r.logger.Info("dictloader: LoadOnce starting", zap.String("loader", ld.Name()))
			rep, err := ld.LoadOnce(gctx)
			resultMu.Lock()
			results[ld.Name()] = rep
			resultMu.Unlock()
			if err != nil {
				r.logger.Error("dictloader: LoadOnce failed",
					zap.String("loader", ld.Name()), zap.Error(err))
				return fmt.Errorf("loader %s: %w", ld.Name(), err)
			}
			r.logger.Info("dictloader: LoadOnce finished",
				zap.String("loader", ld.Name()),
				zap.Int("rows_affected", rep.RowsAffected),
				zap.Int("files_loaded", rep.FilesLoaded),
				zap.Duration("duration", rep.Duration))
			return nil
		})
	}

	err := g.Wait()

	r.mu.Lock()
	for name, rep := range results {
		r.reports[name] = rep
	}
	r.mu.Unlock()

	return results, err
}

// ReloadOne re-runs Reload on the named loader. Returns ErrUnknownLoader for
// unregistered names.
func (r *Registry) ReloadOne(ctx context.Context, name string) (Report, error) {
	r.mu.RLock()
	ld, ok := r.loaders[name]
	r.mu.RUnlock()
	if !ok {
		return Report{}, fmt.Errorf("%w: %s", ErrUnknownLoader, name)
	}

	r.logger.Info("dictloader: Reload starting", zap.String("loader", name))
	rep, err := ld.Reload(ctx)
	r.mu.Lock()
	r.reports[name] = rep
	r.mu.Unlock()
	if err != nil {
		r.logger.Error("dictloader: Reload failed", zap.String("loader", name), zap.Error(err))
		return rep, fmt.Errorf("loader %s: %w", name, err)
	}
	r.logger.Info("dictloader: Reload finished",
		zap.String("loader", name),
		zap.Int("rows_affected", rep.RowsAffected),
		zap.Duration("duration", rep.Duration))
	return rep, nil
}

// Snapshot returns a copy of the most recent Report for each loader.
func (r *Registry) Snapshot() map[string]Report {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]Report, len(r.reports))
	for k, v := range r.reports {
		out[k] = v
	}
	return out
}

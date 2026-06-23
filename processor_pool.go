package html

import (
	"sync"

	"github.com/cybergodev/html/internal"
)

// defaultCfg is the default configuration used by the processor pool.
// Initialized at package load time since DefaultConfig() is a pure function.
var defaultCfg = DefaultConfig()

// processorPool is a sync.Pool for Processor instances.
// Used by package-level functions to reduce allocation overhead.
var processorPool = sync.Pool{
	New: func() any {
		p, err := New(defaultCfg)
		if err != nil {
			// Fallback: return a minimally configured processor
			// This should never happen with valid defaults, but we avoid panic
			// to prevent crashing the entire application
			return &Processor{
				config: &defaultCfg,
				cache:  internal.NewCache(defaultCfg.MaxCacheEntries, defaultCfg.CacheTTL),
			}
		}
		return p
	},
}

// getPooledProcessor gets a Processor from the pool.
// Call putPooledProcessor when done to return it to the pool.
// Includes type check with fallback to prevent panic if pool is corrupted.
func getPooledProcessor() *Processor {
	v := processorPool.Get()
	p, ok := v.(*Processor)
	if !ok {
		// Pool corruption detected: create a new processor as fallback
		// This should never happen under normal circumstances, but we handle it
		// gracefully to prevent crashing the entire application
		p, _ = New(defaultCfg)
	}
	return p
}

// putPooledProcessor returns a Processor to the pool.
// The processor's statistics, audit log, and cache are reset before returning
// to prevent memory accumulation from cached results across pool uses.
func putPooledProcessor(p *Processor) {
	if p == nil {
		return
	}
	// Capture whether Close() stopped the cleanup goroutine during this use.
	// Pooled processors are never closed on the normal path, so we only restart
	// cleanup when it was actually stopped — otherwise every pooled call would
	// needlessly stop and respawn the background cleanup goroutine.
	wasClosed := p.closed.Swap(false)
	p.ResetStatistics()
	// Sink writes are synchronous, so by the time the previous user's Extract
	// returned, every audit entry was already handed to its sink. Wait() is a
	// no-op safety hook kept here to mark the "audit work for this use is done"
	// point before clearing entries and returning the processor to the pool.
	if p.audit != nil {
		p.audit.Wait()
	}
	p.ClearAuditLog()
	p.ClearCache()
	// Restart background cleanup only if Close() stopped it during this use.
	if wasClosed && p.config.CacheTTL > 0 && p.config.CacheCleanup > 0 {
		p.cache.RestartCleanup(p.config.CacheCleanup)
	}
	processorPool.Put(p)
}

// resolveConfig resolves the configuration from optional variadic Config parameter.
// Returns the resolved config, a boolean indicating whether to use the processor pool,
// and an error. When no config is provided, returns DefaultConfig() with pooled=true.
// When one config is provided, returns it with pooled=false.
// Returns ErrMultipleConfigs if more than one config is provided.
func resolveConfig(cfg ...Config) (Config, bool, error) {
	switch len(cfg) {
	case 0:
		return DefaultConfig(), true, nil
	case 1:
		return cfg[0], false, nil
	default:
		return Config{}, false, ErrMultipleConfigs
	}
}

// withProcessor executes a function with a processor.
// When pooled is true, reuses a pooled processor (DefaultConfig) for efficiency.
// When pooled is false, creates a temporary processor with the given config.
func withProcessor[T any](pooled bool, cfg Config, fn func(*Processor) (T, error)) (T, error) {
	if pooled {
		p := getPooledProcessor()
		defer putPooledProcessor(p)
		return fn(p)
	}
	p, err := New(cfg)
	if err != nil {
		var zero T
		return zero, err
	}
	defer func() { _ = p.Close() }()
	return fn(p)
}

// withProcessorBatch executes a batch function with a processor.
// On processor creation failure, returns a BatchResult with all items marked as failed.
func withProcessorBatch(pooled bool, cfg Config, itemCount int, fn func(*Processor) *BatchResult) *BatchResult {
	var p *Processor
	if pooled {
		p = getPooledProcessor()
		defer putPooledProcessor(p)
	} else {
		var err error
		p, err = New(cfg)
		if err != nil {
			errs := make([]error, itemCount)
			for i := range errs {
				errs[i] = err
			}
			return &BatchResult{
				Results: make([]*Result, itemCount),
				Errors:  errs,
				Failed:  itemCount,
			}
		}
		defer func() { _ = p.Close() }()
	}
	return fn(p)
}

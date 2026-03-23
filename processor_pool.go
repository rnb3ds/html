// Package html provides HTML content extraction with automatic encoding detection.
// This file contains the processor pool for efficient reuse of Processor instances.
package html

import (
	"sync"

	"github.com/cybergodev/html/internal"
)

// poolConfig holds the default configuration for pooled processors.
var poolConfig Config
var poolConfigOnce sync.Once

func initPoolConfig() {
	poolConfig = DefaultConfig()
}

// processorPool is a sync.Pool for Processor instances.
// Used by package-level functions to reduce allocation overhead.
var processorPool = sync.Pool{
	New: func() any {
		poolConfigOnce.Do(initPoolConfig)
		p, err := New(poolConfig)
		if err != nil {
			// Fallback: return a minimally configured processor
			// This should never happen with valid defaults, but we avoid panic
			// to prevent crashing the entire application
			return &Processor{
				config: &poolConfig,
				cache:  internal.NewCache(poolConfig.MaxCacheEntries, poolConfig.CacheTTL),
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
		poolConfigOnce.Do(initPoolConfig)
		p, _ = New(poolConfig)
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
	p.ResetStatistics()
	p.ClearAuditLog()
	p.ClearCache()
	processorPool.Put(p)
}

// resolveConfig resolves the configuration from optional variadic Config parameter.
// If no config is provided, returns DefaultConfig(). If one config is provided,
// returns that config. Returns ErrMultipleConfigs if more than one config is provided.
func resolveConfig(cfg ...Config) (Config, error) {
	switch len(cfg) {
	case 0:
		return DefaultConfig(), nil
	case 1:
		return cfg[0], nil
	default:
		return Config{}, ErrMultipleConfigs
	}
}

// getProcessorWithConfig gets or creates a Processor with the given configuration.
// For default configuration, uses the pool for efficiency.
// For custom configuration, creates a new Processor (not pooled).
func getProcessorWithConfig(cfg Config) (*Processor, error) {
	// Check if config matches default config
	poolConfigOnce.Do(initPoolConfig)
	if configEquals(cfg, poolConfig) {
		return getPooledProcessor(), nil
	}

	// Custom config: create new processor (not pooled)
	return New(cfg)
}

// putProcessorWithConfig returns a Processor, handling pooled vs non-pooled processors.
// Processors with custom config are simply closed (not returned to pool).
// Processors with default config are returned to the pool.
func putProcessorWithConfig(p *Processor, cfg Config) {
	if p == nil {
		return
	}

	// Check if config matches default config
	poolConfigOnce.Do(initPoolConfig)
	if configEquals(cfg, poolConfig) {
		putPooledProcessor(p)
	} else {
		// Custom processor: close it (best-effort, ignore error)
		_ = p.Close()
	}
}

// configEquals checks if two configs are functionally equivalent for pooling purposes.
// This compares fields that affect processor behavior, not all fields.
//
// SECURITY: When adding new fields to Config, you MUST update this function
// to ensure processors with different security-relevant settings are not pooled together.
// Failure to do so may cause processors with custom security settings to be
// incorrectly returned to the pool and reused by other callers.
func configEquals(a, b Config) bool {
	// SECURITY-CRITICAL: These fields affect security and behavior.
	// Do NOT remove any comparison without security review.
	return a.MaxInputSize == b.MaxInputSize &&
		a.MaxCacheEntries == b.MaxCacheEntries &&
		a.CacheTTL == b.CacheTTL &&
		a.CacheCleanup == b.CacheCleanup &&
		a.WorkerPoolSize == b.WorkerPoolSize &&
		a.ProcessingTimeout == b.ProcessingTimeout &&
		a.EnableSanitization == b.EnableSanitization &&
		a.MaxDepth == b.MaxDepth &&
		a.ExtractArticle == b.ExtractArticle &&
		a.PreserveImages == b.PreserveImages &&
		a.PreserveLinks == b.PreserveLinks &&
		a.PreserveVideos == b.PreserveVideos &&
		a.PreserveAudios == b.PreserveAudios &&
		a.InlineImageFormat == b.InlineImageFormat &&
		a.InlineLinkFormat == b.InlineLinkFormat &&
		a.TableFormat == b.TableFormat &&
		a.Encoding == b.Encoding &&
		a.ResolveRelativeURLs == b.ResolveRelativeURLs &&
		a.BaseURL == b.BaseURL &&
		a.IncludeImages == b.IncludeImages &&
		a.IncludeVideos == b.IncludeVideos &&
		a.IncludeAudios == b.IncludeAudios &&
		a.IncludeCSS == b.IncludeCSS &&
		a.IncludeJS == b.IncludeJS &&
		a.IncludeContentLinks == b.IncludeContentLinks &&
		a.IncludeExternalLinks == b.IncludeExternalLinks &&
		a.IncludeIcons == b.IncludeIcons &&
		a.Scorer == b.Scorer
}

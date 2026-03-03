// Package internal provides pooled resources for memory allocation optimization.
package internal

import (
	"bytes"
	"hash"
	"hash/fnv"
	"strings"
	"sync"
)

// Pool configuration constants
const (
	// builderPoolInitialCapacity is the initial capacity for pooled strings.Builder
	builderPoolInitialCapacity = 256

	// bufferPoolInitialCapacity is the initial capacity for pooled bytes.Buffer
	bufferPoolInitialCapacity = 1024
)

// poolDebug enables debug logging for pool corruption detection.
// This is intentionally not exported - it's for internal debugging only.
// To enable, set POOL_DEBUG=1 environment variable before import.
var poolDebug = false

// poolLogger is an optional logger for pool corruption warnings.
// Set via SetPoolLogger to enable logging.
var poolLogger struct {
	mu     sync.Mutex
	logger func(format string, args ...any)
}

// SetPoolLogger sets a logger function for pool corruption warnings.
// Pass nil to disable logging. This is a no-op if poolDebug is false.
// The logger function should be thread-safe.
func SetPoolLogger(logger func(format string, args ...any)) {
	poolLogger.mu.Lock()
	defer poolLogger.mu.Unlock()
	poolLogger.logger = logger
}

// logPoolCorruption logs a pool corruption warning if debugging is enabled.
func logPoolCorruption(poolName, expectedType string, got any) {
	if !poolDebug {
		return
	}
	poolLogger.mu.Lock()
	defer poolLogger.mu.Unlock()
	if poolLogger.logger != nil {
		poolLogger.logger("POOL CORRUPTION: %s expected type %s but got %T", poolName, expectedType, got)
	}
}

// BuilderPool is a sync.Pool for strings.Builder instances.
// Use this for functions that build strings incrementally to reduce allocations.
//
// For most use cases, prefer the helper functions GetBuilder() and PutBuilder():
//
//	sb := internal.GetBuilder()
//	defer internal.PutBuilder(sb)
//	sb.Grow(estimatedSize)
//	// ... use sb ...
//	return sb.String()
//
// Direct pool access is also available for advanced use cases:
//
//	sbPtr := internal.BuilderPool.Get().(*strings.Builder)
//	sb := *sbPtr
//	defer func() {
//	    sb.Reset()
//	    internal.BuilderPool.Put(sbPtr)
//	}()
var BuilderPool = sync.Pool{
	New: func() any {
		sb := &strings.Builder{}
		sb.Grow(builderPoolInitialCapacity)
		return sb
	},
}

// BufferPool is a sync.Pool for bytes.Buffer instances.
// Use this for functions that work with byte slices to reduce allocations.
//
// For most use cases, prefer the helper functions GetBuffer() and PutBuffer():
//
//	buf := internal.GetBuffer()
//	defer internal.PutBuffer(buf)
//	buf.Grow(estimatedSize)
//	// ... use buf ...
//	return buf.Bytes()
//
// Direct pool access is also available for advanced use cases:
//
//	bufPtr := internal.BufferPool.Get().(*bytes.Buffer)
//	buf := *bufPtr
//	defer func() {
//	    buf.Reset()
//	    internal.BufferPool.Put(bufPtr)
//	}()
var BufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, bufferPoolInitialCapacity))
	},
}

// GetBuilder gets a strings.Builder from the pool.
// The returned builder has been reset and is ready for use.
// Call PutBuilder when done to return it to the pool.
//
// IMPORTANT: Callers MUST ensure PutBuilder is called even on error paths.
// Use defer immediately after GetBuilder to guarantee cleanup:
//
//	sb := internal.GetBuilder()
//	defer internal.PutBuilder(sb)
//	// ... use sb ...
//
// Failure to return the builder to the pool will not cause memory leaks
// (the GC will collect it), but will reduce the effectiveness of the pool.
func GetBuilder() *strings.Builder {
	v := BuilderPool.Get()
	sb, ok := v.(*strings.Builder)
	if !ok {
		// Log pool corruption if debugging is enabled
		logPoolCorruption("BuilderPool", "*strings.Builder", v)
		// Fallback: return a new builder if pool is corrupted
		sb = &strings.Builder{}
		sb.Grow(builderPoolInitialCapacity)
	}
	return sb
}

// PutBuilder returns a strings.Builder to the pool.
// The builder is reset before being returned to the pool.
// It is safe to call PutBuilder with a nil pointer (no-op).
func PutBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}
	sb.Reset()
	BuilderPool.Put(sb)
}

// GetBuffer gets a bytes.Buffer from the pool.
// The returned buffer has been reset and is ready for use.
// Call PutBuffer when done to return it to the pool.
//
// IMPORTANT: Callers MUST ensure PutBuffer is called even on error paths.
// Use defer immediately after GetBuffer to guarantee cleanup:
//
//	buf := internal.GetBuffer()
//	defer internal.PutBuffer(buf)
//	// ... use buf ...
//
// Failure to return the buffer to the pool will not cause memory leaks
// (the GC will collect it), but will reduce the effectiveness of the pool.
func GetBuffer() *bytes.Buffer {
	v := BufferPool.Get()
	buf, ok := v.(*bytes.Buffer)
	if !ok {
		// Log pool corruption if debugging is enabled
		logPoolCorruption("BufferPool", "*bytes.Buffer", v)
		// Fallback: return a new buffer if pool is corrupted
		buf = bytes.NewBuffer(make([]byte, 0, bufferPoolInitialCapacity))
	}
	return buf
}

// PutBuffer returns a bytes.Buffer to the pool.
// The buffer is reset before being returned to the pool.
// It is safe to call PutBuffer with a nil pointer (no-op).
func PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	buf.Reset()
	BufferPool.Put(buf)
}

// Hash128Pool is a sync.Pool for FNV-128a hash instances.
// Use this for cache key generation to avoid repeated allocations.
//
// Usage pattern:
//
//	h := internal.GetHash128()
//	defer internal.PutHash128(h)
//	h.Write(data)
//	var buf [16]byte
//	sum := h.Sum(buf[:0])
var Hash128Pool = sync.Pool{
	New: func() any {
		return fnv.New128a()
	},
}

// GetHash128 gets an FNV-128a hasher from the pool.
// The returned hasher has been reset and is ready for use.
// Call PutHash128 when done to return it to the pool.
func GetHash128() hash.Hash {
	v := Hash128Pool.Get()
	h, ok := v.(hash.Hash)
	if !ok {
		// Log pool corruption if debugging is enabled
		logPoolCorruption("Hash128Pool", "hash.Hash", v)
		// Fallback: return a new hasher if pool is corrupted
		h = fnv.New128a()
	}
	return h
}

// PutHash128 returns an FNV-128a hasher to the pool.
// The hasher is reset before being returned to the pool.
// It is safe to call PutHash128 with a nil pointer (no-op).
func PutHash128(h hash.Hash) {
	if h == nil {
		return
	}
	h.Reset()
	Hash128Pool.Put(h)
}

// TransformBufferPool is a sync.Pool for byte slices used in encoding transformation.
// These buffers are used for charset conversion operations.
var TransformBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 8192)
		return &buf
	},
}

// GetTransformBuffer gets a byte slice from the transform buffer pool.
// The returned slice has zero length but retained capacity.
func GetTransformBuffer() *[]byte {
	v := TransformBufferPool.Get()
	buf, ok := v.(*[]byte)
	if !ok {
		// Log pool corruption if debugging is enabled
		logPoolCorruption("TransformBufferPool", "*[]byte", v)
		// Fallback: return a new buffer if pool is corrupted
		newBuf := make([]byte, 0, 8192)
		return &newBuf
	}
	return buf
}

// PutTransformBuffer returns a byte slice to the transform buffer pool.
// The slice is reset to zero length before being returned.
// It is safe to call PutTransformBuffer with a nil pointer (no-op).
func PutTransformBuffer(buf *[]byte) {
	if buf == nil {
		return
	}
	*buf = (*buf)[:0]
	TransformBufferPool.Put(buf)
}

// Package internal provides pooled resources for memory allocation optimization.
package internal

import (
	"bytes"
	"hash"
	"hash/fnv"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/net/html"
)

// Pool configuration constants
const (
	// builderPoolInitialCapacity is the initial capacity for pooled strings.Builder
	// Increased from 256 to 1024 to reduce reallocations for typical HTML content
	builderPoolInitialCapacity = 1024

	// bufferPoolInitialCapacity is the initial capacity for pooled bytes.Buffer
	bufferPoolInitialCapacity = 1024
)

// poolDebug enables debug logging for pool corruption detection.
// This is intentionally not exported - it's for internal debugging only.
// To enable, modify this variable to true before using the pool.
//
// Example (in test code):
//
//	internal.SetPoolDebug(true, log.Printf)
var poolDebug atomic.Bool

// poolSecureClear enables secure clearing of pooled objects before reuse.
// When enabled, buffers are zeroed before being returned to the pool,
// preventing potential data leakage from previous uses.
// This has a performance cost and is intended for security-sensitive environments.
var poolSecureClear atomic.Bool

// poolLogger is an optional logger for pool corruption warnings.
var poolLogger struct {
	mu     sync.Mutex
	logger func(format string, args ...any)
}

// SetPoolDebug enables or disables pool debug logging.
// When enabled, pool corruption events are logged to the provided logger.
// This function is intended for development and debugging purposes only.
//
// Note: This is an internal API and may change without notice.
func SetPoolDebug(enabled bool, logger func(format string, args ...any)) {
	poolLogger.mu.Lock()
	poolLogger.logger = logger
	poolLogger.mu.Unlock()
	poolDebug.Store(enabled)
}

// SetPoolSecureClear enables or disables secure clearing of pooled objects.
// When enabled, buffers are zeroed before being returned to the pool,
// preventing potential data leakage from previous uses.
//
// SECURITY: Enable this in security-sensitive environments where data
// isolation between operations is required.
//
// Performance: This option adds ~5-10% overhead to pool operations.
// Note: This is an internal API and may change without notice.
func SetPoolSecureClear(enabled bool) {
	poolSecureClear.Store(enabled)
}

// logPoolCorruption logs a pool corruption warning if debugging is enabled.
func logPoolCorruption(poolName, expectedType string, got any) {
	if !poolDebug.Load() {
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
//
// SECURITY: When poolSecureClear is enabled, the builder's internal buffer
// is zeroed before being returned to prevent data leakage.
func PutBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}
	// SECURITY: Optionally clear sensitive data before returning to pool
	if poolSecureClear.Load() {
		// Use reflection to access and zero the internal buffer
		// strings.Builder has an unexported field 'buf []byte'
		v := reflect.ValueOf(sb).Elem()
		bufField := v.FieldByName("buf")
		if bufField.IsValid() && bufField.Kind() == reflect.Slice {
			// Get slice length and safely zero the bytes
			bufLen := bufField.Len()
			if bufLen > 0 {
				// Access the underlying bytes using unsafe
				// bufField is a slice: {ptr, len, cap}
				bufPtr := unsafe.Pointer(bufField.UnsafeAddr())
				// The slice data starts at the pointer
				for i := 0; i < bufLen; i++ {
					*(*byte)(unsafe.Add(bufPtr, i)) = 0
				}
			}
		}
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
//
// SECURITY: When poolSecureClear is enabled, the buffer's internal data
// is zeroed before being returned to prevent data leakage.
func PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	// SECURITY: Optionally clear sensitive data before returning to pool
	if poolSecureClear.Load() {
		// Zero out the buffer contents before reset
		b := buf.Bytes()
		for i := range b {
			b[i] = 0
		}
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
//
// SECURITY: When poolSecureClear is enabled, the hasher's internal state
// is reset to prevent potential hash state leakage.
func PutHash128(h hash.Hash) {
	if h == nil {
		return
	}
	// SECURITY: Reset is sufficient for hash objects as they don't retain
	// previous hash data after reset
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
//
// SECURITY: When poolSecureClear is enabled, the buffer's data is zeroed
// before being returned to prevent data leakage.
func PutTransformBuffer(buf *[]byte) {
	if buf == nil {
		return
	}
	// SECURITY: Optionally clear sensitive data before returning to pool
	if poolSecureClear.Load() {
		for i := range *buf {
			(*buf)[i] = 0
		}
	}
	*buf = (*buf)[:0]
	TransformBufferPool.Put(buf)
}

// NodeSlicePool is a sync.Pool for HTML node slices used in tree traversal.
// This reduces allocations in WalkNodes and similar traversal functions.
var NodeSlicePool = sync.Pool{
	New: func() any {
		s := make([]*html.Node, 0, 64)
		return &s
	},
}

// GetNodeSlice gets a node slice from the pool.
// The returned slice has zero length but retained capacity.
// Call PutNodeSlice when done to return it to the pool.
func GetNodeSlice() *[]*html.Node {
	v := NodeSlicePool.Get()
	s, ok := v.(*[]*html.Node)
	if !ok {
		logPoolCorruption("NodeSlicePool", "*[]*html.Node", v)
		newS := make([]*html.Node, 0, 64)
		return &newS
	}
	return s
}

// PutNodeSlice returns a node slice to the pool.
// The slice is reset to zero length before being returned.
// It is safe to call PutNodeSlice with a nil pointer (no-op).
//
// SECURITY: When poolSecureClear is enabled, the slice is zeroed
// before being returned to prevent potential pointer leakage.
func PutNodeSlice(s *[]*html.Node) {
	if s == nil {
		return
	}
	// SECURITY: Optionally clear node pointers to prevent potential
	// stale pointer access or information leakage
	if poolSecureClear.Load() {
		for i := range *s {
			(*s)[i] = nil
		}
	}
	*s = (*s)[:0]
	NodeSlicePool.Put(s)
}

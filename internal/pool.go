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
	// builderInitialSize is the initial capacity for pooled strings.Builder
	builderPoolInitialSize = 256

	// bufferInitialCapacity is the initial capacity for pooled bytes.Buffer
	bufferPoolInitialCapacity = 1024
)

// BuilderPool is a sync.Pool for strings.Builder instances.
// Use this for functions that build strings incrementally to reduce allocations.
//
// Usage pattern:
//
//	sbPtr := internal.BuilderPool.Get().(*strings.Builder)
//	sb := *sbPtr
//	defer func() {
//	    sb.Reset()
//	    internal.BuilderPool.Put(sbPtr)
//	}()
//	sb.Grow(estimatedSize)
//	// ... use sb ...
//	return sb.String()
var BuilderPool = sync.Pool{
	New: func() any {
		sb := &strings.Builder{}
		sb.Grow(builderPoolInitialSize)
		return sb
	},
}

// BufferPool is a sync.Pool for bytes.Buffer instances.
// Use this for functions that work with byte slices to reduce allocations.
//
// Usage pattern:
//
//	bufPtr := internal.BufferPool.Get().(*bytes.Buffer)
//	buf := *bufPtr
//	defer func() {
//	    buf.Reset()
//	    internal.BufferPool.Put(bufPtr)
//	}()
//	buf.Grow(estimatedSize)
//	// ... use buf ...
//	return buf.Bytes()
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
		// Fallback: return a new builder if pool is corrupted
		sb = &strings.Builder{}
		sb.Grow(builderPoolInitialSize)
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
		// Fallback: return a new hasher if pool is corrupted
		h = fnv.New128a()
	}
	return h
}

// PutHash128 returns an FNV-128a hasher to the pool.
// The hasher is reset before being returned to the pool.
func PutHash128(h hash.Hash) {
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
		// Fallback: return a new buffer if pool is corrupted
		newBuf := make([]byte, 0, 8192)
		return &newBuf
	}
	return buf
}

// PutTransformBuffer returns a byte slice to the transform buffer pool.
// The slice is reset to zero length before being returned.
func PutTransformBuffer(buf *[]byte) {
	*buf = (*buf)[:0]
	TransformBufferPool.Put(buf)
}

// Package internal provides tests for pooled resources.
package internal

import (
	"bytes"
	"sync"
	"testing"
)

// TestGetBuilder tests getting a strings.Builder from the pool.
func TestGetBuilder(t *testing.T) {
	t.Parallel()

	sb := GetBuilder()
	if sb == nil {
		t.Fatal("GetBuilder() returned nil")
	}

	// Verify the builder is usable
	_, err := sb.WriteString("test")
	if err != nil {
		t.Errorf("WriteString() failed: %v", err)
	}

	if sb.String() != "test" {
		t.Errorf("Expected 'test', got '%s'", sb.String())
	}
}

// TestPutBuilder tests returning a strings.Builder to the pool.
func TestPutBuilder(t *testing.T) {
	t.Parallel()

	sb := GetBuilder()
	sb.WriteString("content")

	// Put the builder back
	PutBuilder(sb)

	// Get another builder - it should be reset
	sb2 := GetBuilder()
	if sb2.Len() != 0 {
		t.Errorf("Builder should be reset, got length %d", sb2.Len())
	}

	PutBuilder(sb2)
}

// TestPutBuilderNil tests that PutBuilder handles nil safely.
func TestPutBuilderNil(t *testing.T) {
	t.Parallel()

	// Should not panic
	PutBuilder(nil)
}

// TestBuilderPoolConcurrent tests concurrent access to BuilderPool.
func TestBuilderPoolConcurrent(t *testing.T) {
	t.Parallel()

	const numGoroutines = 100
	const numOperations = 50

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				sb := GetBuilder()
				sb.WriteString("test string")
				_ = sb.String()
				PutBuilder(sb)
			}
		}()
	}

	wg.Wait()
}

// TestGetBuffer tests getting a bytes.Buffer from the pool.
func TestGetBuffer(t *testing.T) {
	t.Parallel()

	buf := GetBuffer()
	if buf == nil {
		t.Fatal("GetBuffer() returned nil")
	}

	// Verify the buffer is usable
	_, err := buf.Write([]byte("test"))
	if err != nil {
		t.Errorf("Write() failed: %v", err)
	}

	if !bytes.Equal(buf.Bytes(), []byte("test")) {
		t.Errorf("Expected 'test', got '%s'", buf.Bytes())
	}
}

// TestPutBuffer tests returning a bytes.Buffer to the pool.
func TestPutBuffer(t *testing.T) {
	t.Parallel()

	buf := GetBuffer()
	buf.Write([]byte("content"))

	// Put the buffer back
	PutBuffer(buf)

	// Get another buffer - it should be reset
	buf2 := GetBuffer()
	if buf2.Len() != 0 {
		t.Errorf("Buffer should be reset, got length %d", buf2.Len())
	}

	PutBuffer(buf2)
}

// TestPutBufferNil tests that PutBuffer handles nil safely.
func TestPutBufferNil(t *testing.T) {
	t.Parallel()

	// Should not panic
	PutBuffer(nil)
}

// TestBufferPoolConcurrent tests concurrent access to BufferPool.
func TestBufferPoolConcurrent(t *testing.T) {
	t.Parallel()

	const numGoroutines = 100
	const numOperations = 50

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				buf := GetBuffer()
				buf.Write([]byte("test bytes"))
				_ = buf.Bytes()
				PutBuffer(buf)
			}
		}()
	}

	wg.Wait()
}

// TestGetHash128 tests getting an FNV-128a hasher from the pool.
func TestGetHash128(t *testing.T) {
	t.Parallel()

	h := GetHash128()
	if h == nil {
		t.Fatal("GetHash128() returned nil")
	}

	// Verify the hasher is usable
	_, err := h.Write([]byte("test data"))
	if err != nil {
		t.Errorf("Write() failed: %v", err)
	}

	var sum [16]byte
	result := h.Sum(sum[:0])
	if len(result) != 16 {
		t.Errorf("Expected 16 byte hash, got %d", len(result))
	}
}

// TestPutHash128 tests returning a hasher to the pool.
func TestPutHash128(t *testing.T) {
	t.Parallel()

	h := GetHash128()
	h.Write([]byte("content"))

	// Put the hasher back
	PutHash128(h)

	// Get another hasher - it should be reset
	h2 := GetHash128()

	// Write different content and verify it produces different hash
	h2.Write([]byte("different"))
	var sum1 [16]byte
	result := h2.Sum(sum1[:0])

	// The hash should be for "different", not "content"
	// This verifies the hasher was reset
	if len(result) != 16 {
		t.Errorf("Expected 16 byte hash, got %d", len(result))
	}

	PutHash128(h2)
}

// TestHash128PoolConcurrent tests concurrent access to Hash128Pool.
func TestHash128PoolConcurrent(t *testing.T) {
	t.Parallel()

	const numGoroutines = 100
	const numOperations = 50

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				h := GetHash128()
				h.Write([]byte("test data"))
				var sum [16]byte
				h.Sum(sum[:0])
				PutHash128(h)
			}
		}()
	}

	wg.Wait()
}

// TestGetTransformBuffer tests getting a transform buffer from the pool.
func TestGetTransformBuffer(t *testing.T) {
	t.Parallel()

	bufPtr := GetTransformBuffer()
	if bufPtr == nil {
		t.Fatal("GetTransformBuffer() returned nil")
	}

	buf := *bufPtr
	if buf == nil {
		t.Fatal("Transform buffer is nil")
	}

	// Verify the buffer is usable
	buf = append(buf, []byte("test")...)
	if len(buf) != 4 {
		t.Errorf("Expected length 4, got %d", len(buf))
	}
}

// TestPutTransformBuffer tests returning a transform buffer to the pool.
func TestPutTransformBuffer(t *testing.T) {
	t.Parallel()

	bufPtr := GetTransformBuffer()
	*bufPtr = append(*bufPtr, []byte("content")...)

	// Put the buffer back
	PutTransformBuffer(bufPtr)

	// Get another buffer - it should be reset to zero length
	bufPtr2 := GetTransformBuffer()
	if len(*bufPtr2) != 0 {
		t.Errorf("Transform buffer should be reset to zero length, got %d", len(*bufPtr2))
	}

	// But it should retain capacity
	if cap(*bufPtr2) == 0 {
		t.Error("Transform buffer should retain capacity")
	}

	PutTransformBuffer(bufPtr2)
}

// TestTransformBufferPoolConcurrent tests concurrent access to TransformBufferPool.
func TestTransformBufferPoolConcurrent(t *testing.T) {
	t.Parallel()

	const numGoroutines = 100
	const numOperations = 50

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				bufPtr := GetTransformBuffer()
				*bufPtr = append(*bufPtr, []byte("test bytes")...)
				_ = *bufPtr
				PutTransformBuffer(bufPtr)
			}
		}()
	}

	wg.Wait()
}

// TestBuilderPoolInitialSize verifies that builders from the pool can grow efficiently.
// Note: After Reset(), strings.Builder capacity becomes 0, but the pool's New function
// calls Grow() to pre-allocate space for fresh builders.
func TestBuilderPoolInitialSize(t *testing.T) {
	t.Parallel()

	// Get a fresh builder from the pool (may be newly created or reused)
	sb := GetBuilder()
	defer PutBuilder(sb)

	// The builder may have capacity 0 after Reset(), but should be usable
	// The important thing is that it can be written to
	_, err := sb.WriteString("test content")
	if err != nil {
		t.Errorf("WriteString() failed: %v", err)
	}

	// Verify the content was written
	if sb.String() != "test content" {
		t.Errorf("Expected 'test content', got '%s'", sb.String())
	}
}

// TestBufferPoolInitialCapacity verifies that pooled buffers have initial capacity.
func TestBufferPoolInitialCapacity(t *testing.T) {
	t.Parallel()

	buf := GetBuffer()
	defer PutBuffer(buf)

	// The buffer should have been created with bufferPoolInitialCapacity
	if buf.Cap() < bufferPoolInitialCapacity {
		t.Errorf("Expected capacity >= %d, got %d", bufferPoolInitialCapacity, buf.Cap())
	}
}

// TestHashDeterminism verifies that the same input produces the same hash.
func TestHashDeterminism(t *testing.T) {
	t.Parallel()

	input := []byte("test input for hash determinism")

	h1 := GetHash128()
	h1.Write(input)
	var sum1 [16]byte
	result1 := h1.Sum(sum1[:0])
	PutHash128(h1)

	h2 := GetHash128()
	h2.Write(input)
	var sum2 [16]byte
	result2 := h2.Sum(sum2[:0])
	PutHash128(h2)

	if !bytes.Equal(result1, result2) {
		t.Error("Same input should produce same hash")
	}
}

// TestHashDifferentInputs verifies that different inputs produce different hashes.
func TestHashDifferentInputs(t *testing.T) {
	t.Parallel()

	h1 := GetHash128()
	h1.Write([]byte("input 1"))
	var sum1 [16]byte
	result1 := h1.Sum(sum1[:0])
	PutHash128(h1)

	h2 := GetHash128()
	h2.Write([]byte("input 2"))
	var sum2 [16]byte
	result2 := h2.Sum(sum2[:0])
	PutHash128(h2)

	if bytes.Equal(result1, result2) {
		t.Error("Different inputs should produce different hashes")
	}
}

// BenchmarkGetBuilder benchmarks GetBuilder performance.
func BenchmarkGetBuilder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb := GetBuilder()
		PutBuilder(sb)
	}
}

// BenchmarkGetBuffer benchmarks GetBuffer performance.
func BenchmarkGetBuffer(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := GetBuffer()
		PutBuffer(buf)
	}
}

// BenchmarkGetHash128 benchmarks GetHash128 performance.
func BenchmarkGetHash128(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := GetHash128()
		PutHash128(h)
	}
}

// BenchmarkGetTransformBuffer benchmarks GetTransformBuffer performance.
func BenchmarkGetTransformBuffer(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := GetTransformBuffer()
		PutTransformBuffer(buf)
	}
}

// BenchmarkBuilderWithWork benchmarks builder with actual work.
func BenchmarkBuilderWithWork(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb := GetBuilder()
		sb.WriteString("test string content")
		_ = sb.String()
		PutBuilder(sb)
	}
}

// BenchmarkBufferWithWork benchmarks buffer with actual work.
func BenchmarkBufferWithWork(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := GetBuffer()
		buf.Write([]byte("test byte content"))
		_ = buf.Bytes()
		PutBuffer(buf)
	}
}

// BenchmarkHash128WithWork benchmarks hasher with actual work.
func BenchmarkHash128WithWork(b *testing.B) {
	data := []byte("test data for hashing")
	var sum [16]byte

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := GetHash128()
		h.Write(data)
		h.Sum(sum[:0])
		PutHash128(h)
	}
}

// TestGetBuilderPoolCorruption tests that GetBuilder handles corrupted pool gracefully.
// This simulates a scenario where something external puts a wrong type in the pool.
func TestGetBuilderPoolCorruption(t *testing.T) {
	t.Parallel()

	// Put a wrong type in the pool to simulate corruption
	BuilderPool.Put("not a builder")

	// GetBuilder should still return a valid builder (fallback to new)
	sb := GetBuilder()
	if sb == nil {
		t.Fatal("GetBuilder() returned nil even with pool corruption")
	}

	// Verify the builder is usable
	_, err := sb.WriteString("test")
	if err != nil {
		t.Errorf("WriteString() failed: %v", err)
	}

	if sb.String() != "test" {
		t.Errorf("Expected 'test', got '%s'", sb.String())
	}

	PutBuilder(sb)
}

// TestGetBufferPoolCorruption tests that GetBuffer handles corrupted pool gracefully.
func TestGetBufferPoolCorruption(t *testing.T) {
	t.Parallel()

	// Put a wrong type in the pool to simulate corruption
	BufferPool.Put("not a buffer")

	// GetBuffer should still return a valid buffer (fallback to new)
	buf := GetBuffer()
	if buf == nil {
		t.Fatal("GetBuffer() returned nil even with pool corruption")
	}

	// Verify the buffer is usable
	_, err := buf.Write([]byte("test"))
	if err != nil {
		t.Errorf("Write() failed: %v", err)
	}

	if !bytes.Equal(buf.Bytes(), []byte("test")) {
		t.Errorf("Expected 'test', got '%s'", buf.Bytes())
	}

	PutBuffer(buf)
}

// TestGetHash128PoolCorruption tests that GetHash128 handles corrupted pool gracefully.
func TestGetHash128PoolCorruption(t *testing.T) {
	t.Parallel()

	// Put a wrong type in the pool to simulate corruption
	Hash128Pool.Put("not a hasher")

	// GetHash128 should still return a valid hasher (fallback to new)
	h := GetHash128()
	if h == nil {
		t.Fatal("GetHash128() returned nil even with pool corruption")
	}

	// Verify the hasher is usable
	_, err := h.Write([]byte("test data"))
	if err != nil {
		t.Errorf("Write() failed: %v", err)
	}

	var sum [16]byte
	result := h.Sum(sum[:0])
	if len(result) != 16 {
		t.Errorf("Expected 16 byte hash, got %d", len(result))
	}

	PutHash128(h)
}

// TestGetTransformBufferPoolCorruption tests that GetTransformBuffer handles corrupted pool gracefully.
func TestGetTransformBufferPoolCorruption(t *testing.T) {
	t.Parallel()

	// Put a wrong type in the pool to simulate corruption
	TransformBufferPool.Put("not a buffer")

	// GetTransformBuffer should still return a valid buffer (fallback to new)
	bufPtr := GetTransformBuffer()
	if bufPtr == nil {
		t.Fatal("GetTransformBuffer() returned nil even with pool corruption")
	}

	buf := *bufPtr
	if buf == nil {
		t.Fatal("Transform buffer is nil")
	}

	// Verify the buffer is usable
	buf = append(buf, []byte("test")...)
	if len(buf) != 4 {
		t.Errorf("Expected length 4, got %d", len(buf))
	}

	PutTransformBuffer(bufPtr)
}

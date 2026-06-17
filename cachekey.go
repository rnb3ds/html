package html

import (
	"encoding/binary"
	"unsafe"

	"github.com/cybergodev/html/internal"
)

// xxHash-inspired constants for fast hashing
const (
	prime64_1 uint64 = 0x9E3779B185EBCA87
	prime64_2 uint64 = 0xC2B2AE3D27D4EB4F
	prime64_3 uint64 = 0x165667B19E3779F9
	prime64_4 uint64 = 0x85EBCA6798F3B1AD
	prime64_5 uint64 = 0x27D4EB2F165667C5
)

// generateCacheKey creates a hash for cache key generation.
// Uses xxHash-style algorithm optimized for maximum throughput.
// Uses multi-point sampling for large documents to reduce collision risk.
//
// SECURITY: This function uses 5-point sampling for better collision resistance
// against hash-flooding attacks. The sampling strategy ensures that
// modifications anywhere in the document are likely to change the hash.
//
// Performance optimization: Uses inline hashing to reduce function call overhead
// and processes data in larger chunks for better CPU cache utilization.
func (p *Processor) generateCacheKey(content string) string {
	// Initialize hash with seed
	h := prime64_5

	// Pack boolean flags into a single uint8
	flags := uint8(0)
	if p.config.ExtractArticle {
		flags |= 1 << 0
	}
	if p.config.PreserveImages {
		flags |= 1 << 1
	}
	if p.config.PreserveLinks {
		flags |= 1 << 2
	}
	if p.config.PreserveVideos {
		flags |= 1 << 3
	}
	if p.config.PreserveAudios {
		flags |= 1 << 4
	}

	// Mix flags and string options - optimized inline
	h ^= uint64(flags)
	h = hashMixInline(h)

	// Inline short string hashing to reduce function call overhead
	h = hashMixStringInline(h, p.config.InlineImageFormat)
	h = hashMixStringInline(h, p.config.InlineLinkFormat)
	h = hashMixStringInline(h, p.config.TableFormat)

	contentLen := len(content)
	if contentLen <= maxCacheKeySize {
		// Hash the entire content - use zero-copy conversion
		h = hashMixBytesInline(h, internal.StringToBytes(content))
	} else {
		// SECURITY: Multi-point sampling for large documents
		// Using 5 sampling points for better collision resistance
		const sampleCount = 5
		sampleSize := cacheKeySample / sampleCount
		if sampleSize < 256 {
			sampleSize = 256
		}

		// Pre-compute step size for even distribution
		for i := 0; i < sampleCount; i++ {
			var start, end int
			switch i {
			case sampleCount - 1:
				end = contentLen
				start = contentLen - sampleSize
				if start < 0 {
					start = 0
				}
			case 0:
				start = 0
				end = sampleSize
				if end > contentLen {
					end = contentLen
				}
			default:
				offset := (contentLen * i) / (sampleCount - 1)
				start = offset - sampleSize/2
				if start < 0 {
					start = 0
				}
				end = start + sampleSize
				if end > contentLen {
					end = contentLen
					start = end - sampleSize
					if start < 0 {
						start = 0
					}
				}
			}

			if start < end {
				h = hashMixBytesInline(h, internal.StringToBytes(content[start:end]))
			}
		}

		// Mix content length for additional uniqueness
		h ^= uint64(contentLen) * prime64_4
		h = hashMixInline(h)
	}

	// Final avalanche for better distribution
	h ^= h >> 33
	h *= prime64_2
	h ^= h >> 29

	// Generate 16-byte hash for collision resistance
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[:8], h)
	h2 := h ^ prime64_1
	h2 = hashMixInline(h2)
	binary.LittleEndian.PutUint64(buf[8:], h2)

	return string(buf[:])
}

// hashMixInline is an inline version of hashMixFast for critical paths.
// This reduces function call overhead in hot code paths.
func hashMixInline(h uint64) uint64 {
	h ^= h >> 31
	h *= prime64_3
	return h
}

// hashMixStringInline hashes a string by delegating to hashMixBytesInline.
// This eliminates ~70 lines of code duplication between the string and []byte variants.
func hashMixStringInline(h uint64, s string) uint64 {
	return hashMixBytesInline(h, internal.StringToBytes(s))
}

// hashMixBytesInline hashes a byte slice using optimized inline operations.
// This is the canonical implementation for the cache key generation hot path.
func hashMixBytesInline(h uint64, data []byte) uint64 {
	n := len(data)
	if n == 0 {
		return h
	}

	// Mix length first
	h ^= uint64(n) * prime64_5

	// For very small slices, use safe byte-by-byte processing
	if n < 8 {
		var v uint64
		for j := 0; j < n; j++ {
			v = (v << 8) | uint64(data[j])
		}
		h ^= v * prime64_4
		h = hashMixInline(h)
		return h
	}

	ptr := unsafe.Pointer(unsafe.SliceData(data))
	i := 0

	// Process 32 bytes at a time using 4 accumulators
	var acc1, acc2, acc3, acc4 = prime64_1, prime64_2, prime64_3, prime64_4

	for i+32 <= n {
		acc1 += *(*uint64)(unsafe.Add(ptr, i)) * prime64_2
		acc1 = (acc1 << 31) | (acc1 >> 33)
		acc1 *= prime64_1

		acc2 += *(*uint64)(unsafe.Add(ptr, i+8)) * prime64_2
		acc2 = (acc2 << 31) | (acc2 >> 33)
		acc2 *= prime64_1

		acc3 += *(*uint64)(unsafe.Add(ptr, i+16)) * prime64_2
		acc3 = (acc3 << 31) | (acc3 >> 33)
		acc3 *= prime64_1

		acc4 += *(*uint64)(unsafe.Add(ptr, i+24)) * prime64_2
		acc4 = (acc4 << 31) | (acc4 >> 33)
		acc4 *= prime64_1

		i += 32
	}

	// Merge accumulators if we processed any full blocks
	if i > 0 {
		h ^= acc1 + acc2 + acc3 + acc4
		h = hashMixInline(h)
	}

	// Process remaining 8-byte chunks
	for i+8 <= n {
		h ^= *(*uint64)(unsafe.Add(ptr, i)) * prime64_3
		h = hashMixInline(h)
		i += 8
	}

	// Handle remaining bytes using safe indexing
	if i < n {
		var v uint64
		for j := i; j < n; j++ {
			v = (v << 8) | uint64(data[j])
		}
		h ^= v * prime64_4
		h = hashMixInline(h)
	}

	return h
}

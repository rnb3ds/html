// Package internal provides unsafe utility functions for zero-allocation conversions.
package internal

import "unsafe"

// BytesToString converts a byte slice to string without memory allocation.
// The returned string shares memory with the input slice.
//
// WARNING: The caller must ensure the byte slice is not modified after this call.
// Modifying the slice will cause undefined behavior in the returned string.
//
// Use this only when the byte slice is guaranteed to remain unchanged,
// such as when converting read-only data or when the result has a short lifetime.
func BytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}

// StringToBytes converts a string to a byte slice without memory allocation.
// The returned slice shares memory with the original string.
//
// WARNING: The returned slice MUST NOT be modified. Go strings are immutable,
// and modifying the returned slice would violate this immutability, potentially
// causing undefined behavior in other code holding references to the string.
//
// Use this only for short-lived operations where the string is guaranteed
// to remain in scope, such as passing strings to functions that accept []byte.
func StringToBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

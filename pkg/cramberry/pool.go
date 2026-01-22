package cramberry

import (
	"sync"
)

// Size-tiered buffer pools for efficient memory reuse.
// Buffers are pooled in size classes: 64, 256, 1024, 4096, 16384, 65536 bytes.
var bufferPools = [6]sync.Pool{
	{New: func() any { return make([]byte, 0, 64) }},    // Tiny: <= 64 bytes
	{New: func() any { return make([]byte, 0, 256) }},   // Small: <= 256 bytes
	{New: func() any { return make([]byte, 0, 1024) }},  // Medium: <= 1KB
	{New: func() any { return make([]byte, 0, 4096) }},  // Large: <= 4KB
	{New: func() any { return make([]byte, 0, 16384) }}, // XLarge: <= 16KB
	{New: func() any { return make([]byte, 0, 65536) }}, // XXLarge: <= 64KB
}

// bufferSizes maps pool index to capacity.
var bufferSizes = [6]int{64, 256, 1024, 4096, 16384, 65536}

// poolIndex returns the pool index for a given size hint.
func poolIndex(size int) int {
	if size <= 64 {
		return 0
	}
	if size <= 256 {
		return 1
	}
	if size <= 1024 {
		return 2
	}
	if size <= 4096 {
		return 3
	}
	if size <= 16384 {
		return 4
	}
	if size <= 65536 {
		return 5
	}
	return -1 // Too large for pooling
}

// GetBuffer gets a buffer from the appropriate size-tiered pool.
// The buffer is reset to zero length but retains its capacity.
// Returns nil if sizeHint is larger than 64KB (too large for pooling).
func GetBuffer(sizeHint int) []byte {
	idx := poolIndex(sizeHint)
	if idx < 0 {
		return make([]byte, 0, sizeHint)
	}
	buf := bufferPools[idx].Get().([]byte)
	return buf[:0]
}

// PutBuffer returns a buffer to the appropriate size-tiered pool.
// The buffer capacity determines which pool it goes into.
// Buffers larger than 64KB are not pooled.
func PutBuffer(buf []byte) {
	c := cap(buf)
	if c > 65536 {
		return // Too large, let GC handle it
	}
	idx := poolIndex(c)
	if idx >= 0 {
		//nolint:staticcheck // SA6002: slice pooling is intentional, boxing overhead is acceptable
		bufferPools[idx].Put(buf[:0])
	}
}

// GetWriterWithHint gets a Writer with a pre-allocated buffer sized for the hint.
// The Writer should be returned with PutWriter when done.
func GetWriterWithHint(sizeHint int) *Writer {
	buf := GetBuffer(sizeHint)
	return &Writer{
		buf:  buf,
		opts: DefaultOptions,
	}
}

// BufferPoolStats returns statistics about buffer pool usage.
// This is useful for tuning and debugging.
type BufferPoolStats struct {
	SizeClasses  []int // Capacity of each size class
	TotalClasses int   // Number of size classes
}

// GetBufferPoolStats returns current buffer pool configuration.
func GetBufferPoolStats() BufferPoolStats {
	return BufferPoolStats{
		SizeClasses:  bufferSizes[:],
		TotalClasses: len(bufferSizes),
	}
}

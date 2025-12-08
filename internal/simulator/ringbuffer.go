package simulator

import (
	"sync"

	"github.com/vedranburojevic/xcw/internal/domain"
)

// RingBuffer is a thread-safe circular buffer for log entries
type RingBuffer struct {
	mu     sync.RWMutex
	buffer []domain.LogEntry
	size   int
	head   int
	count  int
}

// NewRingBuffer creates a ring buffer with the specified capacity
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 100 // Default
	}
	return &RingBuffer{
		buffer: make([]domain.LogEntry, size),
		size:   size,
	}
}

// Push adds an entry to the buffer
func (rb *RingBuffer) Push(entry domain.LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer[rb.head] = entry
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// GetAll returns all entries in order (oldest first)
func (rb *RingBuffer) GetAll() []domain.LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result := make([]domain.LogEntry, rb.count)

	if rb.count < rb.size {
		// Buffer not full, start from 0
		copy(result, rb.buffer[:rb.count])
	} else {
		// Buffer full, start from head (oldest)
		copy(result, rb.buffer[rb.head:])
		copy(result[rb.size-rb.head:], rb.buffer[:rb.head])
	}

	return result
}

// GetLast returns the last n entries (most recent)
func (rb *RingBuffer) GetLast(n int) []domain.LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if n > rb.count {
		n = rb.count
	}

	result := make([]domain.LogEntry, n)

	start := (rb.head - n + rb.size) % rb.size
	for i := 0; i < n; i++ {
		result[i] = rb.buffer[(start+i)%rb.size]
	}

	return result
}

// Count returns the number of entries in the buffer
func (rb *RingBuffer) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Clear empties the buffer
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.count = 0
}

// CountByLevel returns counts grouped by log level
func (rb *RingBuffer) CountByLevel() map[domain.LogLevel]int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	counts := make(map[domain.LogLevel]int)
	entries := rb.GetAll()
	for _, e := range entries {
		counts[e.Level]++
	}
	return counts
}

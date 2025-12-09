package simulator

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/domain"
)

func TestNewRingBuffer(t *testing.T) {
	t.Run("creates buffer with specified size", func(t *testing.T) {
		rb := NewRingBuffer(50)
		require.NotNil(t, rb)
		assert.Equal(t, 0, rb.Count())
	})

	t.Run("uses default size for zero", func(t *testing.T) {
		rb := NewRingBuffer(0)
		require.NotNil(t, rb)
		// Should use default of 100
		for i := 0; i < 150; i++ {
			rb.Push(domain.LogEntry{Message: "test"})
		}
		assert.Equal(t, 100, rb.Count())
	})

	t.Run("uses default size for negative", func(t *testing.T) {
		rb := NewRingBuffer(-5)
		require.NotNil(t, rb)
		assert.Equal(t, 0, rb.Count())
	})
}

func TestRingBufferPush(t *testing.T) {
	t.Run("adds entries", func(t *testing.T) {
		rb := NewRingBuffer(10)

		rb.Push(domain.LogEntry{Message: "first"})
		assert.Equal(t, 1, rb.Count())

		rb.Push(domain.LogEntry{Message: "second"})
		assert.Equal(t, 2, rb.Count())
	})

	t.Run("wraps around when full", func(t *testing.T) {
		rb := NewRingBuffer(3)

		rb.Push(domain.LogEntry{Message: "1"})
		rb.Push(domain.LogEntry{Message: "2"})
		rb.Push(domain.LogEntry{Message: "3"})
		assert.Equal(t, 3, rb.Count())

		rb.Push(domain.LogEntry{Message: "4"})
		assert.Equal(t, 3, rb.Count()) // Count stays at max

		// Should have entries 2, 3, 4 (oldest first)
		entries := rb.GetAll()
		assert.Len(t, entries, 3)
		assert.Equal(t, "2", entries[0].Message)
		assert.Equal(t, "3", entries[1].Message)
		assert.Equal(t, "4", entries[2].Message)
	})
}

func TestRingBufferGetAll(t *testing.T) {
	t.Run("returns empty for new buffer", func(t *testing.T) {
		rb := NewRingBuffer(10)
		entries := rb.GetAll()
		assert.Empty(t, entries)
	})

	t.Run("returns entries in order", func(t *testing.T) {
		rb := NewRingBuffer(10)
		rb.Push(domain.LogEntry{Message: "first"})
		rb.Push(domain.LogEntry{Message: "second"})
		rb.Push(domain.LogEntry{Message: "third"})

		entries := rb.GetAll()
		require.Len(t, entries, 3)
		assert.Equal(t, "first", entries[0].Message)
		assert.Equal(t, "second", entries[1].Message)
		assert.Equal(t, "third", entries[2].Message)
	})

	t.Run("preserves order after wrap", func(t *testing.T) {
		rb := NewRingBuffer(3)
		rb.Push(domain.LogEntry{Message: "1"})
		rb.Push(domain.LogEntry{Message: "2"})
		rb.Push(domain.LogEntry{Message: "3"})
		rb.Push(domain.LogEntry{Message: "4"})
		rb.Push(domain.LogEntry{Message: "5"})

		entries := rb.GetAll()
		require.Len(t, entries, 3)
		assert.Equal(t, "3", entries[0].Message) // Oldest
		assert.Equal(t, "4", entries[1].Message)
		assert.Equal(t, "5", entries[2].Message) // Newest
	})
}

func TestRingBufferGetLast(t *testing.T) {
	t.Run("returns empty for new buffer", func(t *testing.T) {
		rb := NewRingBuffer(10)
		entries := rb.GetLast(5)
		assert.Empty(t, entries)
	})

	t.Run("returns all if n > count", func(t *testing.T) {
		rb := NewRingBuffer(10)
		rb.Push(domain.LogEntry{Message: "a"})
		rb.Push(domain.LogEntry{Message: "b"})

		entries := rb.GetLast(100)
		assert.Len(t, entries, 2)
	})

	t.Run("returns last n entries", func(t *testing.T) {
		rb := NewRingBuffer(10)
		rb.Push(domain.LogEntry{Message: "a"})
		rb.Push(domain.LogEntry{Message: "b"})
		rb.Push(domain.LogEntry{Message: "c"})
		rb.Push(domain.LogEntry{Message: "d"})
		rb.Push(domain.LogEntry{Message: "e"})

		entries := rb.GetLast(3)
		require.Len(t, entries, 3)
		assert.Equal(t, "c", entries[0].Message)
		assert.Equal(t, "d", entries[1].Message)
		assert.Equal(t, "e", entries[2].Message)
	})

	t.Run("handles wrap correctly", func(t *testing.T) {
		rb := NewRingBuffer(3)
		rb.Push(domain.LogEntry{Message: "1"})
		rb.Push(domain.LogEntry{Message: "2"})
		rb.Push(domain.LogEntry{Message: "3"})
		rb.Push(domain.LogEntry{Message: "4"})

		entries := rb.GetLast(2)
		require.Len(t, entries, 2)
		assert.Equal(t, "3", entries[0].Message)
		assert.Equal(t, "4", entries[1].Message)
	})
}

func TestRingBufferClear(t *testing.T) {
	rb := NewRingBuffer(10)
	rb.Push(domain.LogEntry{Message: "test"})
	rb.Push(domain.LogEntry{Message: "test"})

	assert.Equal(t, 2, rb.Count())

	rb.Clear()
	assert.Equal(t, 0, rb.Count())
	assert.Empty(t, rb.GetAll())
}

func TestRingBufferCountByLevel(t *testing.T) {
	rb := NewRingBuffer(10)
	rb.Push(domain.LogEntry{Level: domain.LogLevelDebug})
	rb.Push(domain.LogEntry{Level: domain.LogLevelDebug})
	rb.Push(domain.LogEntry{Level: domain.LogLevelInfo})
	rb.Push(domain.LogEntry{Level: domain.LogLevelError})
	rb.Push(domain.LogEntry{Level: domain.LogLevelError})
	rb.Push(domain.LogEntry{Level: domain.LogLevelError})

	counts := rb.CountByLevel()
	assert.Equal(t, 2, counts[domain.LogLevelDebug])
	assert.Equal(t, 1, counts[domain.LogLevelInfo])
	assert.Equal(t, 3, counts[domain.LogLevelError])
	assert.Equal(t, 0, counts[domain.LogLevelFault])
}

func TestRingBufferConcurrency(t *testing.T) {
	rb := NewRingBuffer(100)
	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rb.Push(domain.LogEntry{Message: "test"})
			}
		}(i)
	}

	// Readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rb.GetAll()
				rb.GetLast(10)
				rb.Count()
			}
		}()
	}

	wg.Wait()

	// Should not panic and count should be capped at buffer size
	assert.LessOrEqual(t, rb.Count(), 100)
}

// Benchmark tests
func BenchmarkRingBufferPush(b *testing.B) {
	rb := NewRingBuffer(1000)
	entry := domain.LogEntry{Message: "benchmark entry"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Push(entry)
	}
}

func BenchmarkRingBufferGetAll(b *testing.B) {
	rb := NewRingBuffer(1000)
	for i := 0; i < 1000; i++ {
		rb.Push(domain.LogEntry{Message: "entry"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.GetAll()
	}
}

func BenchmarkRingBufferGetLast(b *testing.B) {
	rb := NewRingBuffer(1000)
	for i := 0; i < 1000; i++ {
		rb.Push(domain.LogEntry{Message: "entry"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.GetLast(100)
	}
}

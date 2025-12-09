package filter

import (
	"github.com/vburojevic/xcw/internal/domain"
)

// Filter determines if a log entry should be included
type Filter interface {
	// Match returns true if the entry passes the filter
	Match(entry *domain.LogEntry) bool
}

// Chain combines multiple filters (all must pass)
type Chain struct {
	filters []Filter
}

// NewChain creates a filter chain from multiple filters
func NewChain(filters ...Filter) *Chain {
	return &Chain{filters: filters}
}

// Match returns true only if all filters pass
func (c *Chain) Match(entry *domain.LogEntry) bool {
	for _, f := range c.filters {
		if !f.Match(entry) {
			return false
		}
	}
	return true
}

// Add appends a filter to the chain
func (c *Chain) Add(f Filter) {
	c.filters = append(c.filters, f)
}

// OrChain combines multiple filters (any must pass)
type OrChain struct {
	filters []Filter
}

// NewOrChain creates an OR filter chain
func NewOrChain(filters ...Filter) *OrChain {
	return &OrChain{filters: filters}
}

// Match returns true if any filter passes
func (c *OrChain) Match(entry *domain.LogEntry) bool {
	if len(c.filters) == 0 {
		return true
	}
	for _, f := range c.filters {
		if f.Match(entry) {
			return true
		}
	}
	return false
}

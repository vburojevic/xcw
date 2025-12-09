package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PatternStore handles persistence of known error patterns
type PatternStore struct {
	mu       sync.RWMutex
	path     string
	patterns map[string]*StoredPattern
}

// StoredPattern represents a persisted error pattern
type StoredPattern struct {
	Pattern    string    `json:"pattern"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	TotalCount int       `json:"total_count"`
}

// patternsFile is the structure stored on disk
type patternsFile struct {
	Version  int                       `json:"version"`
	Patterns map[string]*StoredPattern `json:"patterns"`
}

// NewPatternStore creates a new pattern store
// If path is empty, uses ~/.xcw/patterns.json
func NewPatternStore(path string) *PatternStore {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		path = filepath.Join(home, ".xcw", "patterns.json")
	}

	store := &PatternStore{
		path:     path,
		patterns: make(map[string]*StoredPattern),
	}

	// Load existing patterns (ignore errors, start fresh if file doesn't exist)
	store.Load()

	return store
}

// Load reads patterns from disk
func (s *PatternStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file yet, that's fine
		}
		return err
	}

	var file patternsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}

	s.patterns = file.Patterns
	if s.patterns == nil {
		s.patterns = make(map[string]*StoredPattern)
	}

	return nil
}

// Save writes patterns to disk
func (s *PatternStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file := patternsFile{
		Version:  1,
		Patterns: s.patterns,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// RecordPattern records a pattern occurrence
// Returns true if this is a new pattern, false if it was already known
func (s *PatternStore) RecordPattern(pattern string, count int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if existing, ok := s.patterns[pattern]; ok {
		existing.LastSeen = now
		existing.TotalCount += count
		return false
	}

	s.patterns[pattern] = &StoredPattern{
		Pattern:    pattern,
		FirstSeen:  now,
		LastSeen:   now,
		TotalCount: count,
	}
	return true
}

// IsKnown returns true if the pattern has been seen before
func (s *PatternStore) IsKnown(pattern string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.patterns[pattern]
	return ok
}

// GetPattern returns stored info about a pattern
func (s *PatternStore) GetPattern(pattern string) *StoredPattern {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.patterns[pattern]
}

// GetAllPatterns returns all stored patterns
func (s *PatternStore) GetAllPatterns() []*StoredPattern {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*StoredPattern, 0, len(s.patterns))
	for _, p := range s.patterns {
		result = append(result, p)
	}
	return result
}

// Count returns the number of stored patterns
func (s *PatternStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.patterns)
}

// Clear removes all stored patterns
func (s *PatternStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.patterns = make(map[string]*StoredPattern)
}

// EnhancedPatternMatch extends PatternMatch with knowledge status
type EnhancedPatternMatch struct {
	PatternMatch
	IsNew      bool       `json:"is_new"`
	FirstSeen  *time.Time `json:"first_seen,omitempty"`
	TotalCount int        `json:"total_count,omitempty"`
}

// AnnotatePatterns adds known/new status to detected patterns
func (s *PatternStore) AnnotatePatterns(patterns []PatternMatch) []EnhancedPatternMatch {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]EnhancedPatternMatch, len(patterns))
	for i, p := range patterns {
		enhanced := EnhancedPatternMatch{
			PatternMatch: p,
			IsNew:        true,
		}

		if stored, ok := s.patterns[p.Pattern]; ok {
			enhanced.IsNew = false
			enhanced.FirstSeen = &stored.FirstSeen
			enhanced.TotalCount = stored.TotalCount
		}

		result[i] = enhanced
	}
	return result
}

// RecordPatterns records multiple patterns and returns enhanced versions
func (s *PatternStore) RecordPatterns(patterns []PatternMatch) []EnhancedPatternMatch {
	result := make([]EnhancedPatternMatch, len(patterns))
	for i, p := range patterns {
		isNew := s.RecordPattern(p.Pattern, p.Count)
		stored := s.GetPattern(p.Pattern)

		enhanced := EnhancedPatternMatch{
			PatternMatch: p,
			IsNew:        isNew,
		}
		if stored != nil {
			enhanced.FirstSeen = &stored.FirstSeen
			enhanced.TotalCount = stored.TotalCount
		}
		result[i] = enhanced
	}
	return result
}

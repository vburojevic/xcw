package simulator

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vedranburojevic/xcw/internal/domain"
)

// StreamOptions configures log streaming behavior
type StreamOptions struct {
	BundleID   string         // Filter by app bundle identifier
	Subsystems []string       // Filter by subsystems
	Categories []string       // Filter by categories
	MinLevel   domain.LogLevel // Minimum log level
	Pattern    *regexp.Regexp // Regex pattern for message filtering
	BufferSize int            // Ring buffer size
}

// Streamer handles real-time log streaming from a simulator
type Streamer struct {
	manager *Manager
	parser  *Parser

	mu         sync.RWMutex
	udid       string
	opts       StreamOptions
	cmd        *exec.Cmd
	logs       chan domain.LogEntry
	errors     chan error
	running    bool
	cancelFunc context.CancelFunc
	buffer     *RingBuffer

	// Stats
	totalCount int
	errorCount int
	faultCount int
}

// NewStreamer creates a new log streamer
func NewStreamer(manager *Manager) *Streamer {
	return &Streamer{
		manager: manager,
		parser:  NewParser(),
		logs:    make(chan domain.LogEntry, 1000),
		errors:  make(chan error, 10),
	}
}

// Start begins streaming logs from the specified device
func (s *Streamer) Start(ctx context.Context, udid string, opts StreamOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("streamer already running")
	}

	s.udid = udid
	s.opts = opts

	bufSize := opts.BufferSize
	if bufSize <= 0 {
		bufSize = 100
	}
	s.buffer = NewRingBuffer(bufSize)
	s.totalCount = 0
	s.errorCount = 0
	s.faultCount = 0

	// Create cancellable context
	streamCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.running = true

	// Start streaming with auto-reconnect
	go s.streamLoop(streamCtx)

	return nil
}

// streamLoop handles reconnection logic
func (s *Streamer) streamLoop(ctx context.Context) {
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check if device is booted
		device, err := s.manager.GetDeviceInfo(ctx, s.udid)
		if err != nil {
			s.sendError(fmt.Errorf("failed to get device info: %w", err))
			time.Sleep(backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		if !device.IsBooted() {
			// Try to boot the device
			if err := s.manager.BootDevice(ctx, s.udid); err != nil {
				s.sendError(fmt.Errorf("failed to boot device: %w", err))
				time.Sleep(backoff)
				backoff = min(backoff*2, maxBackoff)
				continue
			}

			// Wait for boot
			if err := s.manager.WaitForBoot(ctx, s.udid, 60*time.Second); err != nil {
				s.sendError(fmt.Errorf("timeout waiting for boot: %w", err))
				continue
			}
		}

		// Reset backoff on successful connection
		backoff = time.Second

		// Start log stream
		err = s.runLogStream(ctx)
		if err != nil && ctx.Err() == nil {
			s.sendError(fmt.Errorf("log stream error: %w", err))
		}

		// Check if we should reconnect
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(backoff)
			backoff = min(backoff*2, maxBackoff)
		}
	}
}

// runLogStream executes the log stream command and processes output
func (s *Streamer) runLogStream(ctx context.Context) error {
	// Build command arguments
	args := []string{"simctl", "spawn", s.udid, "log", "stream", "--style", "ndjson"}

	// Add log level
	level := strings.ToLower(string(s.opts.MinLevel))
	if level == "" || level == "default" {
		level = "default"
	}
	args = append(args, "--level", level)

	// Build predicate for filtering
	predicate := s.buildPredicate()
	if predicate != "" {
		args = append(args, "--predicate", predicate)
	}

	cmd := exec.CommandContext(ctx, "xcrun", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start log stream: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	// Read and parse log lines
	scanner := bufio.NewScanner(stdout)
	// Increase buffer size for long log lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()

		entry, err := s.parser.Parse(line)
		if err != nil {
			continue // Skip unparseable lines
		}

		if entry == nil {
			continue // Skip non-log events
		}

		// Apply level filter
		if entry.Level.Priority() < s.opts.MinLevel.Priority() {
			continue
		}

		// Apply pattern filter
		if s.opts.Pattern != nil && !s.opts.Pattern.MatchString(entry.Message) {
			continue
		}

		// Update stats
		s.mu.Lock()
		s.totalCount++
		if entry.Level == domain.LogLevelError {
			s.errorCount++
		}
		if entry.Level == domain.LogLevelFault {
			s.faultCount++
		}
		s.mu.Unlock()

		// Add to ring buffer
		s.buffer.Push(*entry)

		// Send to channel
		select {
		case s.logs <- *entry:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel full, entry already in buffer
		}
	}

	return cmd.Wait()
}

// buildPredicate constructs an NSPredicate string for log filtering
func (s *Streamer) buildPredicate() string {
	var parts []string

	if s.opts.BundleID != "" {
		// Filter by subsystem (usually matches bundle ID)
		parts = append(parts, fmt.Sprintf(`subsystem BEGINSWITH "%s"`, s.opts.BundleID))
	}

	for _, sub := range s.opts.Subsystems {
		parts = append(parts, fmt.Sprintf(`subsystem == "%s"`, sub))
	}

	for _, cat := range s.opts.Categories {
		parts = append(parts, fmt.Sprintf(`category == "%s"`, cat))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " OR ")
}

func (s *Streamer) sendError(err error) {
	select {
	case s.errors <- err:
	default:
	}
}

// Stop terminates the log stream
func (s *Streamer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}

	s.running = false
	return nil
}

// Logs returns the channel of parsed log entries
func (s *Streamer) Logs() <-chan domain.LogEntry {
	return s.logs
}

// Errors returns the channel for stream errors
func (s *Streamer) Errors() <-chan error {
	return s.errors
}

// IsRunning returns whether the streamer is active
func (s *Streamer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetBufferedLogs returns recent logs from the ring buffer
func (s *Streamer) GetBufferedLogs() []domain.LogEntry {
	return s.buffer.GetAll()
}

// GetStats returns current statistics
func (s *Streamer) GetStats() (total, errors, faults int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.totalCount, s.errorCount, s.faultCount
}

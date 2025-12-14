package simulator

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
)

// StreamOptions configures log streaming behavior
type StreamOptions struct {
	BundleID          string           // Filter by app bundle identifier
	Subsystems        []string         // Filter by subsystems
	Categories        []string         // Filter by categories
	Processes         []string         // Filter by process names
	MinLevel          domain.LogLevel  // Minimum log level (inclusive)
	MaxLevel          domain.LogLevel  // Maximum log level (inclusive, empty = no max)
	Pattern           *regexp.Regexp   // Regex pattern for message filtering
	ExcludePatterns   []*regexp.Regexp // Regex patterns to exclude from messages
	ExcludeSubsystems []string         // Subsystems to exclude
	BufferSize        int              // Ring buffer size
	RawPredicate      string           // Raw NSPredicate string (overrides other filters)
	Verbose           bool             // Enable verbose diagnostics
}

// Streamer handles real-time log streaming from a simulator
type Streamer struct {
	manager *Manager
	parser  *Parser
	rng     *rand.Rand

	mu         sync.RWMutex
	udid       string
	opts       StreamOptions
	cmd        *exec.Cmd
	logs       chan domain.LogEntry
	errors     chan error
	running    bool
	cancelFunc context.CancelFunc
	buffer     *RingBuffer
	wg         sync.WaitGroup
	done       chan struct{}
	closeOnce  sync.Once

	// Stats
	totalCount  int
	errorCount  int
	faultCount  int
	dropped     int
	tsDropped   int
	chanDropped int
	reconnects  int
}

// NewStreamer creates a new log streamer
func NewStreamer(manager *Manager) *Streamer {
	return &Streamer{
		manager: manager,
		parser:  NewParser(),
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
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

	// Always count timestamp parse drops; optionally emit diagnostics in verbose mode.
	s.parser.SetTimestampErrorHandler(func(raw string, err error) {
		s.mu.Lock()
		s.tsDropped++
		drops := s.tsDropped
		verbose := s.opts.Verbose
		s.mu.Unlock()
		if verbose && drops%100 == 0 {
			s.sendError(fmt.Errorf("timestamp_parse_drop: %d failures (latest %q: %v)", drops, raw, err))
		}
	})

	bufSize := opts.BufferSize
	if bufSize <= 0 {
		bufSize = 100
	}
	s.buffer = NewRingBuffer(bufSize)
	s.totalCount = 0
	s.errorCount = 0
	s.faultCount = 0
	s.dropped = 0
	s.tsDropped = 0
	s.chanDropped = 0
	s.reconnects = 0

	// Create cancellable context
	streamCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.running = true
	s.done = make(chan struct{})

	// Start streaming with auto-reconnect
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.streamLoop(streamCtx)
		close(s.done)
	}()

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
	consecutiveFailures := 0

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
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			backoff = s.jitter(min(backoff*2, maxBackoff))
			continue
		}

		if !device.IsBooted() {
			// Try to boot the device
			if err := s.manager.BootDevice(ctx, s.udid); err != nil {
				s.sendError(fmt.Errorf("failed to boot device: %w", err))
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return
				}
				backoff = s.jitter(min(backoff*2, maxBackoff))
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
		if ctx.Err() == nil {
			if err != nil {
				consecutiveFailures++
				// Avoid noisy warnings on transient reconnects; surface errors in verbose mode,
				// and always after a few consecutive failures.
				if s.opts.Verbose || consecutiveFailures >= 3 {
					s.sendError(fmt.Errorf("log stream error: %w", err))
				}
			} else {
				consecutiveFailures = 0
			}

			s.mu.Lock()
			s.reconnects++
			s.mu.Unlock()
			s.sendError(fmt.Errorf("reconnect_notice: reconnecting log stream"))
		}

		// Check if we should reconnect
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			backoff = s.jitter(min(backoff*2, maxBackoff))
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
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start log stream: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	// Drain stderr to avoid deadlocks and surface diagnostics in verbose mode.
	stderrErrCh := make(chan error, 1)
	verbose := s.opts.Verbose
	go func() {
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 0, 64*1024), 256*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			if verbose {
				s.sendError(fmt.Errorf("xcrun_stderr: %s", line))
			}
		}
		stderrErrCh <- sc.Err()
	}()

	// Read and parse log lines
	scanner := bufio.NewScanner(stdout)
	// Increase buffer size for long log lines
	const maxLineBytes = 1024 * 1024
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

Loop:
	for scanner.Scan() {
		line := scanner.Bytes()

		entry, err := s.parser.Parse(line)
		if err != nil {
			// Track dropped lines; emit periodic diagnostics
			s.mu.Lock()
			s.dropped++
			drops := s.dropped
			s.mu.Unlock()
			if drops%500 == 0 {
				s.sendError(fmt.Errorf("parse_drop: %d lines could not be parsed", drops))
			}
			continue // Skip unparseable lines
		}

		if entry == nil {
			continue // Skip non-log events
		}

		// Apply level filter (min)
		if entry.Level.Priority() < s.opts.MinLevel.Priority() {
			continue
		}

		// Apply level filter (max) - only if MaxLevel is set
		if s.opts.MaxLevel != "" && entry.Level.Priority() > s.opts.MaxLevel.Priority() {
			continue
		}

		// Apply pattern filter
		if s.opts.Pattern != nil && !s.opts.Pattern.MatchString(entry.Message) {
			continue
		}

		// Apply exclusion pattern filters (any match excludes)
		if s.matchExcludePatterns(entry.Message) {
			continue
		}

		// Apply subsystem exclusion filter
		if len(s.opts.ExcludeSubsystems) > 0 && s.shouldExcludeSubsystem(entry.Subsystem) {
			continue
		}

		// Apply process filter
		if len(s.opts.Processes) > 0 && !matchProcess(entry.Process, s.opts.Processes) {
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
			break Loop
		default:
			s.mu.Lock()
			s.chanDropped++
			s.mu.Unlock()
		}
	}

	stdoutErr := scanner.Err()
	if stdoutErr != nil && errors.Is(stdoutErr, bufio.ErrTooLong) {
		stdoutErr = fmt.Errorf("log stream output line too long (>%d bytes): %w", maxLineBytes, stdoutErr)
	}
	if stdoutErr != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	waitErr := cmd.Wait()
	stderrErr := <-stderrErrCh

	if stdoutErr != nil {
		return stdoutErr
	}
	if stderrErr != nil && ctx.Err() == nil {
		return fmt.Errorf("log stream stderr read error: %w", stderrErr)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return waitErr
}

// buildPredicate constructs an NSPredicate string for log filtering
// Uses AND between groups (subsystem, category) for narrowing results
// Uses OR within groups for matching any of multiple values
func (s *Streamer) buildPredicate() string {
	return buildPredicate(s.opts.RawPredicate, s.opts.BundleID, s.opts.Subsystems, s.opts.Categories)
}

// shouldExcludeSubsystem checks if a subsystem should be excluded
func (s *Streamer) shouldExcludeSubsystem(subsystem string) bool {
	for _, excl := range s.opts.ExcludeSubsystems {
		// Support wildcard matching with *
		if strings.HasSuffix(excl, "*") {
			prefix := strings.TrimSuffix(excl, "*")
			if strings.HasPrefix(subsystem, prefix) {
				return true
			}
		} else if subsystem == excl {
			return true
		}
	}
	return false
}

// matchExcludePatterns checks if message matches any exclude pattern
func (s *Streamer) matchExcludePatterns(message string) bool {
	for _, p := range s.opts.ExcludePatterns {
		if p != nil && p.MatchString(message) {
			return true
		}
	}
	return false
}

func (s *Streamer) sendError(err error) {
	select {
	case s.errors <- err:
	default:
	}
}

func (s *Streamer) jitter(d time.Duration) time.Duration {
	// random between 0.5x and 1.5x using per-stream RNG
	factor := 0.5 + s.rng.Float64()
	return time.Duration(float64(d) * factor)
}

// Stop terminates the log stream
func (s *Streamer) Stop() error {
	s.mu.Lock()
	cancel := s.cancelFunc
	cmd := s.cmd
	done := s.done
	s.running = false
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	// Wait for streamLoop/runLogStream to exit
	s.wg.Wait()
	if done != nil {
		select {
		case <-done:
		default:
		}
	}

	// Close channels once to signal consumers
	s.closeOnce.Do(func() {
		close(s.logs)
		close(s.errors)
	})

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

// GetDropped returns number of dropped (unparseable) log lines
func (s *Streamer) GetDropped() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dropped
}

type StreamDiagnostics struct {
	Reconnects          int
	ParseDrops          int
	TimestampParseDrops int
	ChannelDrops        int
	Buffered            int
}

func (s *Streamer) GetDiagnostics() StreamDiagnostics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bufCount := 0
	if s.buffer != nil {
		bufCount = s.buffer.Count()
	}
	return StreamDiagnostics{
		Reconnects:          s.reconnects,
		ParseDrops:          s.dropped,
		TimestampParseDrops: s.tsDropped,
		ChannelDrops:        s.chanDropped,
		Buffered:            bufCount,
	}
}

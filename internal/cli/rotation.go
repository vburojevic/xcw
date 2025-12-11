package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

// rotation manages per-session file rotation for tail.
type rotation struct {
	pathBuilder    func(int) (string, error)
	outputFile     *os.File
	bufferedWriter *bufio.Writer
}

func newRotation(pb func(int) (string, error)) *rotation {
	return &rotation{pathBuilder: pb}
}

func (r *rotation) Open(session int) (writer *bufio.Writer, file *os.File, path string, err error) {
	if r.pathBuilder == nil {
		return nil, nil, "", nil
	}

	if r.bufferedWriter != nil {
		if err := r.bufferedWriter.Flush(); err != nil {
			return nil, nil, "", fmt.Errorf("failed to flush previous output: %w", err)
		}
	}
	if r.outputFile != nil {
		if err := r.outputFile.Close(); err != nil {
			return nil, nil, "", fmt.Errorf("failed to close previous output: %w", err)
		}
	}

	path, err = r.pathBuilder(session)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to build path: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			return nil, nil, "", fmt.Errorf("failed to create output dir: %w", mkErr)
		}
	}

	r.outputFile, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create output file: %w", err)
	}
	r.bufferedWriter = bufio.NewWriterSize(r.outputFile, 64*1024)
	return r.bufferedWriter, r.outputFile, path, nil
}

func (r *rotation) Close() error {
	if r.bufferedWriter != nil {
		if err := r.bufferedWriter.Flush(); err != nil {
			return err
		}
	}
	if r.outputFile != nil {
		if err := r.outputFile.Close(); err != nil {
			return err
		}
	}
	return nil
}

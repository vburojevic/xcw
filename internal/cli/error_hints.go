package cli

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/vburojevic/xcw/internal/simulator"
)

func hintForDeviceLookup(err error) string {
	if err == nil {
		return ""
	}

	var multi *simulator.MultipleBootedError
	if errors.As(err, &multi) {
		return "Pass --simulator to select one; try `xcw list --booted-only`"
	}

	var amb *simulator.AmbiguousDeviceError
	if errors.As(err, &amb) {
		return "Be more specific or use the full UDID; try `xcw list`"
	}

	msg := err.Error()
	if strings.Contains(msg, "no booted simulator found") {
		return "Boot a simulator in Simulator.app/Xcode, or pass --simulator; try `xcw list --booted-only`"
	}

	return ""
}

func hintForTooling(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	// Common xcrun/Xcode-select problems.
	if strings.Contains(msg, "invalid active developer path") {
		return "Xcode CLI tools not configured; run `xcode-select --install` or `sudo xcode-select -s /Applications/Xcode.app/Contents/Developer` (then `xcw doctor`)"
	}
	if strings.Contains(strings.ToLower(msg), "license") && strings.Contains(strings.ToLower(msg), "xcodebuild") {
		return "Xcode license may not be accepted; try `sudo xcodebuild -license accept` (then `xcw doctor`)"
	}

	if isCommandNotFound(err, "xcrun") {
		return "xcrun not found; install Xcode Command Line Tools with `xcode-select --install` (then `xcw doctor`)"
	}

	return ""
}

func hintForStreamOrQuery(err error) string {
	if err == nil {
		return ""
	}
	if h := hintForTooling(err); h != "" {
		return h
	}
	if h := hintForDeviceLookup(err); h != "" {
		return h
	}
	return "Run `xcw doctor` for diagnostics"
}

func hintForFilter(err error) string {
	if err == nil {
		return ""
	}
	return "If the filter contains spaces/parentheses, quote it. Example: --where '(level=Error OR level=Fault) AND message~timeout' (regex literal: message~/timeout|crash/i)"
}

func isCommandNotFound(err error, name string) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, exec.ErrNotFound) && name == "" {
		return true
	}

	var ee *exec.Error
	if errors.As(err, &ee) && strings.EqualFold(ee.Name, name) && errors.Is(ee.Err, exec.ErrNotFound) {
		return true
	}

	var pe *os.PathError
	if errors.As(err, &pe) && errors.Is(pe.Err, exec.ErrNotFound) {
		if strings.EqualFold(pe.Path, name) || strings.HasSuffix(pe.Path, string(os.PathSeparator)+name) {
			return true
		}
	}

	// Fallback to string matching for wrapped errors.
	msg := err.Error()
	if strings.Contains(msg, "executable file not found") && strings.Contains(msg, name) {
		return true
	}
	if strings.Contains(msg, "No such file or directory") && strings.Contains(msg, name) {
		return true
	}

	return false
}

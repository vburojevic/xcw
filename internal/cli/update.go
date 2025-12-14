package cli

import (
	"encoding/json"
	"fmt"

	"github.com/vburojevic/xcw/internal/output"
)

// UpdateCmd shows how to upgrade xcw
type UpdateCmd struct{}

// UpdateOutput represents the NDJSON output for update instructions
type UpdateOutput struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schemaVersion"`
	Version       string `json:"current_version"`
	Commit        string `json:"commit"`
	Homebrew      string `json:"homebrew"`
	GoInstall     string `json:"go_install"`
	ReleasesURL   string `json:"releases_url"`
}

const (
	homebrewCmd  = "brew update && brew upgrade xcw"
	goInstallCmd = "go install github.com/vburojevic/xcw/cmd/xcw@latest"
	releasesURL  = "https://github.com/vburojevic/xcw/releases"
)

// Run executes the update command
func (c *UpdateCmd) Run(globals *Globals) error {
	if globals.Format == "ndjson" {
		return c.outputNDJSON(globals)
	}
	return c.outputText(globals)
}

func (c *UpdateCmd) outputNDJSON(globals *Globals) error {
	out := UpdateOutput{
		Type:          "update",
		SchemaVersion: output.SchemaVersion,
		Version:       Version,
		Commit:        Commit,
		Homebrew:      homebrewCmd,
		GoInstall:     goInstallCmd,
		ReleasesURL:   releasesURL,
	}

	encoder := json.NewEncoder(globals.Stdout)
	return encoder.Encode(out)
}

func (c *UpdateCmd) outputText(globals *Globals) error {
	if _, err := fmt.Fprintln(globals.Stdout, "xcw update instructions"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "Current version: %s (%s)\n", Version, Commit); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout, "To upgrade via Homebrew:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  %s\n", homebrewCmd); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout, "To upgrade via Go:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  %s\n", goInstallCmd); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout, "For release notes, see:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  %s\n", releasesURL); err != nil {
		return err
	}

	return nil
}

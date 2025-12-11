package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds application configuration
type Config struct {
	// Global settings
	Format  string `mapstructure:"format"`
	Level   string `mapstructure:"level"`
	Quiet   bool   `mapstructure:"quiet"`
	Verbose bool   `mapstructure:"verbose"`

	// Default values for commands
	Defaults DefaultsConfig `mapstructure:"defaults"`
	Tail     TailConfig     `mapstructure:"tail"`
	Query    QueryConfig    `mapstructure:"query"`
	Watch    WatchConfig    `mapstructure:"watch"`
}

// DefaultsConfig holds default values for various commands
type DefaultsConfig struct {
	// Tail command defaults
	Simulator       string   `mapstructure:"simulator"`
	App             string   `mapstructure:"app"`
	BufferSize      int      `mapstructure:"buffer_size"`
	SummaryInterval string   `mapstructure:"summary_interval"`
	Heartbeat       string   `mapstructure:"heartbeat"`
	Subsystems      []string `mapstructure:"subsystems"`
	Categories      []string `mapstructure:"categories"`

	// Query command defaults
	Since string `mapstructure:"since"`
	Limit int    `mapstructure:"limit"`

	// Exclusion filters
	ExcludeSubsystems []string `mapstructure:"exclude_subsystems"`
	ExcludePattern    string   `mapstructure:"exclude_pattern"`
}

type TailConfig struct {
	Simulator       string   `mapstructure:"simulator"`
	App             string   `mapstructure:"app"`
	SummaryInterval string   `mapstructure:"summary_interval"`
	Heartbeat       string   `mapstructure:"heartbeat"`
	SessionIdle     string   `mapstructure:"session_idle"`
	Exclude         []string `mapstructure:"exclude"`
	Where           []string `mapstructure:"where"`
}

type QueryConfig struct {
	Simulator string   `mapstructure:"simulator"`
	App       string   `mapstructure:"app"`
	Since     string   `mapstructure:"since"`
	Limit     int      `mapstructure:"limit"`
	Exclude   []string `mapstructure:"exclude"`
	Where     []string `mapstructure:"where"`
}

type WatchConfig struct {
	Simulator string `mapstructure:"simulator"`
	App       string `mapstructure:"app"`
	Cooldown  string `mapstructure:"cooldown"`
}

// Default returns a Config with default values
func Default() *Config {
	return &Config{
		Format:  "ndjson",
		Level:   "debug",
		Quiet:   false,
		Verbose: false,
		Defaults: DefaultsConfig{
			Simulator:  "booted",
			BufferSize: 100,
			Since:      "5m",
			Limit:      1000,
		},
		Tail: TailConfig{
			Simulator: "booted",
		},
		Query: QueryConfig{
			Simulator: "booted",
			Since:     "5m",
			Limit:     1000,
		},
		Watch: WatchConfig{
			Simulator: "booted",
			Cooldown:  "5s",
		},
	}
}

// Load loads configuration from files and environment
// Config file search order (highest precedence first):
// 1. ./.xcw.yaml or ./.xcw.yml
// 2. ~/.xcw.yaml or ~/.xcw.yml
// 3. $XDG_CONFIG_HOME/xcw/config.yaml (or ~/.config/xcw/config.yaml)
// 4. /etc/xcw/config.yaml
func Load() (*Config, error) {
	cfg := Default()
	v := viper.New()

	// Defaults
	v.SetDefault("format", cfg.Format)
	v.SetDefault("level", cfg.Level)
	v.SetDefault("quiet", cfg.Quiet)
	v.SetDefault("verbose", cfg.Verbose)

	v.SetDefault("defaults.simulator", cfg.Defaults.Simulator)
	v.SetDefault("defaults.buffer_size", cfg.Defaults.BufferSize)
	v.SetDefault("defaults.since", cfg.Defaults.Since)
	v.SetDefault("defaults.limit", cfg.Defaults.Limit)

	v.SetDefault("tail.simulator", cfg.Tail.Simulator)
	v.SetDefault("tail.heartbeat", cfg.Tail.Heartbeat)
	v.SetDefault("tail.summary_interval", cfg.Tail.SummaryInterval)
	v.SetDefault("tail.session_idle", cfg.Tail.SessionIdle)

	v.SetDefault("query.simulator", cfg.Query.Simulator)
	v.SetDefault("query.since", cfg.Query.Since)
	v.SetDefault("query.limit", cfg.Query.Limit)

	v.SetDefault("watch.simulator", cfg.Watch.Simulator)
	v.SetDefault("watch.cooldown", cfg.Watch.Cooldown)

	// Env overrides
	v.SetEnvPrefix("XCW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	// Common env shortcuts
	_ = v.BindEnv("defaults.app", "XCW_APP")
	_ = v.BindEnv("tail.app", "XCW_APP")
	_ = v.BindEnv("query.app", "XCW_APP")
	_ = v.BindEnv("watch.app", "XCW_APP")
	_ = v.BindEnv("defaults.simulator", "XCW_SIMULATOR")
	_ = v.BindEnv("tail.simulator", "XCW_SIMULATOR")
	_ = v.BindEnv("query.simulator", "XCW_SIMULATOR")
	_ = v.BindEnv("watch.simulator", "XCW_SIMULATOR")

	// Try to find and load config file in order of precedence
	configFile := findConfigFile()
	if configFile != "" {
		v.SetConfigFile(configFile)

		if err := v.ReadInConfig(); err != nil {
			return nil, err
		}
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// findConfigFile searches for config file in standard locations
func findConfigFile() string {
	// Config file names to search for (in order)
	names := []string{".xcw.yaml", ".xcw.yml", "xcw.yaml", "xcw.yml"}

	// Get home directory
	home, homeErr := os.UserHomeDir()

	// Get config directory (XDG_CONFIG_HOME or ~/.config)
	configDir, configDirErr := os.UserConfigDir()

	// Search locations in order of precedence (highest first)
	var searchPaths []string

	// 1. Current directory
	cwd, err := os.Getwd()
	if err == nil {
		searchPaths = append(searchPaths, cwd)
	}

	// 2. Home directory
	if homeErr == nil {
		searchPaths = append(searchPaths, home)
	}

	// 3. Config directory (e.g., ~/.config/xcw/)
	if configDirErr == nil {
		searchPaths = append(searchPaths, filepath.Join(configDir, "xcw"))
	}

	// 4. System config
	searchPaths = append(searchPaths, "/etc/xcw")

	// Search for config file
	for _, dir := range searchPaths {
		for _, name := range names {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
		// Also check for config.yaml in subdirs
		path := filepath.Join(dir, "config.yaml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// LoadFromFile loads configuration from a specific file
func LoadFromFile(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := Default()
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks config values for basic correctness.
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}

	// Validate global enums
	switch strings.ToLower(c.Format) {
	case "", "ndjson", "text":
	default:
		return fmt.Errorf("invalid format: %q (expected ndjson or text)", c.Format)
	}
	switch strings.ToLower(c.Level) {
	case "", "debug", "info", "default", "error", "fault":
	default:
		return fmt.Errorf("invalid level: %q (expected debug, info, default, error, fault)", c.Level)
	}

	checkDuration := func(name, val string) error {
		if val == "" {
			return nil
		}
		if _, err := time.ParseDuration(val); err != nil {
			return fmt.Errorf("invalid duration for %s: %q (%v)", name, val, err)
		}
		return nil
	}

	if err := checkDuration("defaults.since", c.Defaults.Since); err != nil {
		return err
	}
	if err := checkDuration("tail.heartbeat", c.Tail.Heartbeat); err != nil {
		return err
	}
	if err := checkDuration("tail.summary_interval", c.Tail.SummaryInterval); err != nil {
		return err
	}
	if err := checkDuration("tail.session_idle", c.Tail.SessionIdle); err != nil {
		return err
	}
	if err := checkDuration("query.since", c.Query.Since); err != nil {
		return err
	}
	if err := checkDuration("watch.cooldown", c.Watch.Cooldown); err != nil {
		return err
	}

	if c.Defaults.BufferSize < 0 {
		return fmt.Errorf("defaults.buffer_size must be >= 0")
	}
	if c.Defaults.Limit < 0 {
		return fmt.Errorf("defaults.limit must be >= 0")
	}
	if c.Query.Limit < 0 {
		return fmt.Errorf("query.limit must be >= 0")
	}

	return nil
}

// ConfigFile returns the path to the config file that would be loaded
func ConfigFile() string {
	return findConfigFile()
}

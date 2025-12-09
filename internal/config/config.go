package config

import (
	"os"
	"path/filepath"

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

// Default returns a Config with default values
func Default() *Config {
	return &Config{
		Format:  "ndjson",
		Level:   "default",
		Quiet:   false,
		Verbose: false,
		Defaults: DefaultsConfig{
			Simulator:  "booted",
			BufferSize: 100,
			Since:      "5m",
			Limit:      1000,
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

	// Try to find and load config file in order of precedence
	configFile := findConfigFile()
	if configFile != "" {
		v := viper.New()
		v.SetConfigFile(configFile)

		if err := v.ReadInConfig(); err != nil {
			return nil, err
		}

		if err := v.Unmarshal(cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	applyEnvOverrides(cfg)

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

// applyEnvOverrides applies environment variable overrides to config
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("XCW_FORMAT"); v != "" {
		cfg.Format = v
	}
	if v := os.Getenv("XCW_LEVEL"); v != "" {
		cfg.Level = v
	}
	if v := os.Getenv("XCW_QUIET"); v == "true" || v == "1" {
		cfg.Quiet = true
	}
	if v := os.Getenv("XCW_VERBOSE"); v == "true" || v == "1" {
		cfg.Verbose = true
	}
	if v := os.Getenv("XCW_APP"); v != "" {
		cfg.Defaults.App = v
	}
	if v := os.Getenv("XCW_SIMULATOR"); v != "" {
		cfg.Defaults.Simulator = v
	}
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

	return cfg, nil
}

// ConfigFile returns the path to the config file that would be loaded
func ConfigFile() string {
	return findConfigFile()
}

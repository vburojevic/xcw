package domain

import "time"

// DeviceState represents the current state of a simulator
type DeviceState string

const (
	DeviceStateShutdown     DeviceState = "Shutdown"
	DeviceStateBooted       DeviceState = "Booted"
	DeviceStateBooting      DeviceState = "Booting"
	DeviceStateCreating     DeviceState = "Creating"
	DeviceStateShuttingDown DeviceState = "Shutting Down"
)

// Device represents an iOS Simulator device
type Device struct {
	UDID                 string      `json:"udid"`
	Name                 string      `json:"name"`
	State                DeviceState `json:"state"`
	IsAvailable          bool        `json:"isAvailable"`
	DeviceTypeIdentifier string      `json:"deviceTypeIdentifier"`
	RuntimeIdentifier    string      `json:"runtime"`
	DataPath             string      `json:"dataPath,omitempty"`
	LogPath              string      `json:"logPath,omitempty"`
	LastBootedAt         *time.Time  `json:"lastBootedAt,omitempty"`
}

// IsBooted returns true if the device is currently booted
func (d *Device) IsBooted() bool {
	return d.State == DeviceStateBooted
}

// simctlDevicesResponse matches `xcrun simctl list devices --json` output
type SimctlDevicesResponse struct {
	Devices map[string][]SimctlDevice `json:"devices"`
}

// SimctlDevice represents a device from simctl JSON output
type SimctlDevice struct {
	UDID                 string  `json:"udid"`
	Name                 string  `json:"name"`
	State                string  `json:"state"`
	IsAvailable          bool    `json:"isAvailable"`
	DeviceTypeIdentifier string  `json:"deviceTypeIdentifier"`
	DataPath             string  `json:"dataPath"`
	LogPath              string  `json:"logPath"`
	LastBootedAt         *string `json:"lastBootedAt,omitempty"`
}

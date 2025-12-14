package domain

// Discovery represents the aggregated log discovery results
type Discovery struct {
	Type          string             `json:"type"`
	SchemaVersion int                `json:"schemaVersion"`
	App           string             `json:"app,omitempty"`
	TimeRange     DiscoveryTimeRange `json:"time_range"`
	TotalCount    int                `json:"total_count"`
	Subsystems    []SubsystemInfo    `json:"subsystems"`
	Categories    []CategoryInfo     `json:"categories"`
	Processes     []ProcessInfo      `json:"processes"`
	Levels        map[string]int     `json:"levels"`
}

// DiscoveryTimeRange represents the time range of discovered logs
type DiscoveryTimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// SubsystemInfo contains aggregated subsystem statistics
type SubsystemInfo struct {
	Name   string         `json:"name"`
	Count  int            `json:"count"`
	Levels map[string]int `json:"levels"`
}

// CategoryInfo contains aggregated category statistics
type CategoryInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ProcessInfo contains aggregated process statistics
type ProcessInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

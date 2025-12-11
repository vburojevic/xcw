package cli

// TailFilterFlags groups filtering-related flags for tail while keeping flag names intact via embedding.
type TailFilterFlags struct {
	Pattern          string   `short:"p" aliases:"filter" help:"Regex pattern to filter log messages"`
	Exclude          []string `short:"x" help:"Regex pattern to exclude from log messages (can be repeated)"`
	ExcludeSubsystem []string `help:"Exclude logs from subsystem (can be repeated, supports * wildcard)"`
	MinLevel         string   `help:"Minimum log level: debug, info, default, error, fault (overrides global --level)"`
	MaxLevel         string   `help:"Maximum log level: debug, info, default, error, fault"`
	Where            []string `short:"w" help:"Field filter (e.g., 'level=error', 'message~timeout'). Operators: =, !=, ~, !~, >=, <=, ^, $"`
	Dedupe           bool     `help:"Collapse repeated identical messages"`
	DedupeWindow     string   `help:"Time window for deduplication (e.g., '5s', '1m'). Without this, only consecutive duplicates are collapsed"`
	Process          []string `help:"Filter by process name (can be repeated)"`
}

// TailOutputFlags groups output flags (files, tmux, summaries, heartbeats).
type TailOutputFlags struct {
	Output          string `short:"o" help:"Write output to explicit file path"`
	SessionDir      string `help:"Directory for session files (default: ~/.xcw/sessions)"`
	SessionPrefix   string `help:"Prefix for session filename (default: app bundle ID)"`
	Tmux            bool   `help:"Output to tmux session"`
	Session         string `help:"Custom tmux session name (default: xcw-<simulator>)"`
	SummaryInterval string `help:"Emit periodic summaries (e.g., '30s', '1m')"`
	Heartbeat       string `help:"Emit periodic heartbeat messages (e.g., '10s', '30s')"`
}

// TailAgentFlags groups agent/control flags.
type TailAgentFlags struct {
	WaitForLaunch bool   `help:"Start streaming immediately, emit 'ready' event when capture is active"`
	NoAgentHints  bool   `help:"Suppress agent_hints banners (leave off for AI agents)"`
	DryRunJSON    bool   `help:"Print resolved stream options as JSON and exit (no streaming)"`
	MaxDuration   string `help:"Stop after duration (e.g., '5m') emitting session_end (agent-safe cutoff)"`
	MaxLogs       int    `help:"Stop after N logs emitting session_end (agent-safe cutoff)"`
	SessionIdle   string `help:"Emit session boundary after idle period with no logs (e.g., '60s')"`
}

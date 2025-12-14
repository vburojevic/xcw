package cli

import (
	"fmt"
)

// CompletionCmd generates shell completions
type CompletionCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish" help:"Shell type (bash, zsh, fish)"`
}

// Run executes the completion command
func (c *CompletionCmd) Run(globals *Globals) error {
	switch c.Shell {
	case "bash":
		return c.generateBash(globals)
	case "zsh":
		return c.generateZsh(globals)
	case "fish":
		return c.generateFish(globals)
	default:
		return fmt.Errorf("unsupported shell: %s", c.Shell)
	}
}

func (c *CompletionCmd) generateBash(globals *Globals) error {
	script := `# xcw bash completion script
# Add to ~/.bashrc or ~/.bash_profile:
#   eval "$(xcw completion bash)"

_xcw_completions() {
    local cur prev words cword
    _init_completion || return

    local commands="list tail query summary watch clear schema config version completion"
    local global_flags="-f --format -l --level -q --quiet -v --verbose"

    case "${prev}" in
        xcw)
            COMPREPLY=($(compgen -W "${commands}" -- "${cur}"))
            return
            ;;
        -f|--format)
            COMPREPLY=($(compgen -W "ndjson text" -- "${cur}"))
            return
            ;;
        -l|--level)
            COMPREPLY=($(compgen -W "debug info default error fault" -- "${cur}"))
            return
            ;;
        -s|--simulator)
            # Complete with booted simulators
            local sims=$(xcrun simctl list devices booted -j 2>/dev/null | grep '"name"' | cut -d'"' -f4 | tr '\n' ' ')
            COMPREPLY=($(compgen -W "booted ${sims}" -- "${cur}"))
            return
            ;;
        completion)
            COMPREPLY=($(compgen -W "bash zsh fish" -- "${cur}"))
            return
            ;;
    esac

    case "${words[1]}" in
        tail)
            COMPREPLY=($(compgen -W "-s --simulator -a --app -p --pattern -x --exclude --exclude-subsystem --subsystem --category --buffer-size --summary-interval --heartbeat --tmux --session ${global_flags}" -- "${cur}"))
            ;;
        query)
            COMPREPLY=($(compgen -W "-s --simulator -a --app --since --until -p --pattern -x --exclude --exclude-subsystem --limit --subsystem --category --analyze ${global_flags}" -- "${cur}"))
            ;;
        list)
            COMPREPLY=($(compgen -W "-b --booted-only --runtime ${global_flags}" -- "${cur}"))
            ;;
        watch)
            COMPREPLY=($(compgen -W "-s --simulator -a --app --on-error --on-fault --on-pattern --cooldown ${global_flags}" -- "${cur}"))
            ;;
        schema)
            COMPREPLY=($(compgen -W "-t --type ${global_flags}" -- "${cur}"))
            ;;
        *)
            COMPREPLY=($(compgen -W "${commands} ${global_flags}" -- "${cur}"))
            ;;
    esac
}

complete -F _xcw_completions xcw
`
	_, err := fmt.Fprint(globals.Stdout, script)
	return err
}

func (c *CompletionCmd) generateZsh(globals *Globals) error {
	script := `#compdef xcw
# xcw zsh completion script
# Add to ~/.zshrc:
#   eval "$(xcw completion zsh)"

_xcw() {
    local -a commands
    commands=(
        'list:List available simulators'
        'tail:Stream logs from a running simulator'
        'query:Query historical logs from simulator'
        'summary:Output summary of recent logs'
        'watch:Watch logs and trigger commands on patterns'
        'clear:Clear tmux session content'
        'schema:Output JSON Schema for xcw output types'
        'config:Show or manage configuration'
        'version:Show version information'
        'completion:Generate shell completions'
    )

    local -a global_opts
    global_opts=(
        '-f[Output format]:format:(ndjson text)'
        '--format[Output format]:format:(ndjson text)'
        '-l[Minimum log level]:level:(debug info default error fault)'
        '--level[Minimum log level]:level:(debug info default error fault)'
        '-q[Suppress non-log output]'
        '--quiet[Suppress non-log output]'
        '-v[Show debug output]'
        '--verbose[Show debug output]'
    )

    _arguments -C \
        $global_opts \
        '1: :->command' \
        '*:: :->args'

    case $state in
        command)
            _describe 'command' commands
            ;;
        args)
            case $words[1] in
                tail)
                    _arguments \
                        '-s[Simulator name or UDID]:simulator:->simulators' \
                        '--simulator[Simulator name or UDID]:simulator:->simulators' \
                        '-a[App bundle identifier]:app:' \
                        '--app[App bundle identifier]:app:' \
                        '-p[Regex pattern to filter]:pattern:' \
                        '--pattern[Regex pattern to filter]:pattern:' \
                        '-x[Regex pattern to exclude]:pattern:' \
                        '--exclude[Regex pattern to exclude]:pattern:' \
                        '--heartbeat[Emit heartbeat interval]:interval:' \
                        '--tmux[Output to tmux session]' \
                        '*--subsystem[Filter by subsystem]:subsystem:' \
                        '*--category[Filter by category]:category:' \
                        $global_opts
                    ;;
                query)
                    _arguments \
                        '-s[Simulator name or UDID]:simulator:->simulators' \
                        '--simulator[Simulator name or UDID]:simulator:->simulators' \
                        '-a[App bundle identifier]:app:' \
                        '--app[App bundle identifier]:app:' \
                        '--since[How far back to query]:duration:' \
                        '--limit[Maximum logs]:limit:' \
                        '--analyze[Include analysis summary]' \
                        $global_opts
                    ;;
                list)
                    _arguments \
                        '-b[Show only booted]' \
                        '--booted-only[Show only booted]' \
                        '--runtime[Filter by iOS version]:runtime:' \
                        $global_opts
                    ;;
                completion)
                    _arguments '1:shell:(bash zsh fish)'
                    ;;
            esac

            case $state in
                simulators)
                    local -a sims
                    sims=(booted ${(f)"$(xcrun simctl list devices booted -j 2>/dev/null | grep '"name"' | cut -d'"' -f4)"})
                    _describe 'simulator' sims
                    ;;
            esac
            ;;
    esac
}

compdef _xcw xcw
`
	_, err := fmt.Fprint(globals.Stdout, script)
	return err
}

func (c *CompletionCmd) generateFish(globals *Globals) error {
	script := `# xcw fish completion script
# Add to ~/.config/fish/completions/xcw.fish

# Disable file completion by default
complete -c xcw -f

# Commands
complete -c xcw -n "__fish_use_subcommand" -a "list" -d "List available simulators"
complete -c xcw -n "__fish_use_subcommand" -a "tail" -d "Stream logs from a running simulator"
complete -c xcw -n "__fish_use_subcommand" -a "query" -d "Query historical logs from simulator"
complete -c xcw -n "__fish_use_subcommand" -a "summary" -d "Output summary of recent logs"
complete -c xcw -n "__fish_use_subcommand" -a "watch" -d "Watch logs and trigger commands on patterns"
complete -c xcw -n "__fish_use_subcommand" -a "clear" -d "Clear tmux session content"
complete -c xcw -n "__fish_use_subcommand" -a "schema" -d "Output JSON Schema for xcw output types"
complete -c xcw -n "__fish_use_subcommand" -a "config" -d "Show or manage configuration"
complete -c xcw -n "__fish_use_subcommand" -a "version" -d "Show version information"
complete -c xcw -n "__fish_use_subcommand" -a "completion" -d "Generate shell completions"

# Global flags
complete -c xcw -s f -l format -d "Output format" -xa "ndjson text"
complete -c xcw -s l -l level -d "Minimum log level" -xa "debug info default error fault"
complete -c xcw -s q -l quiet -d "Suppress non-log output"
complete -c xcw -s v -l verbose -d "Show debug output"

# Tail command
complete -c xcw -n "__fish_seen_subcommand_from tail" -s s -l simulator -d "Simulator name or UDID"
complete -c xcw -n "__fish_seen_subcommand_from tail" -s a -l app -d "App bundle identifier" -r
complete -c xcw -n "__fish_seen_subcommand_from tail" -s p -l pattern -d "Regex pattern to filter"
complete -c xcw -n "__fish_seen_subcommand_from tail" -s x -l exclude -d "Regex pattern to exclude"
complete -c xcw -n "__fish_seen_subcommand_from tail" -l heartbeat -d "Emit heartbeat interval"
complete -c xcw -n "__fish_seen_subcommand_from tail" -l tmux -d "Output to tmux session"
complete -c xcw -n "__fish_seen_subcommand_from tail" -l subsystem -d "Filter by subsystem"
complete -c xcw -n "__fish_seen_subcommand_from tail" -l category -d "Filter by category"

# Query command
complete -c xcw -n "__fish_seen_subcommand_from query" -s s -l simulator -d "Simulator name or UDID"
complete -c xcw -n "__fish_seen_subcommand_from query" -s a -l app -d "App bundle identifier" -r
complete -c xcw -n "__fish_seen_subcommand_from query" -l since -d "How far back to query"
complete -c xcw -n "__fish_seen_subcommand_from query" -l limit -d "Maximum logs"
complete -c xcw -n "__fish_seen_subcommand_from query" -l analyze -d "Include analysis summary"

# List command
complete -c xcw -n "__fish_seen_subcommand_from list" -s b -l booted-only -d "Show only booted"
complete -c xcw -n "__fish_seen_subcommand_from list" -l runtime -d "Filter by iOS version"

# Watch command
complete -c xcw -n "__fish_seen_subcommand_from watch" -s s -l simulator -d "Simulator name or UDID"
complete -c xcw -n "__fish_seen_subcommand_from watch" -s a -l app -d "App bundle identifier" -r
complete -c xcw -n "__fish_seen_subcommand_from watch" -l on-error -d "Command on error"
complete -c xcw -n "__fish_seen_subcommand_from watch" -l on-fault -d "Command on fault"
complete -c xcw -n "__fish_seen_subcommand_from watch" -l cooldown -d "Cooldown between triggers"

# Completion command
complete -c xcw -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"

# Simulator completion
complete -c xcw -n "__fish_seen_subcommand_from tail query watch; and __fish_contains_opt -s s simulator" -a "(xcrun simctl list devices booted -j 2>/dev/null | grep '\"name\"' | cut -d'\"' -f4; echo booted)"
`
	_, err := fmt.Fprint(globals.Stdout, script)
	return err
}

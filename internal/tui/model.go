package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
)

var (
	detailStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("230")).Bold(true)
)

// Model represents the TUI state
type Model struct {
	logs        []domain.LogEntry
	filteredIdx []int
	content     string
	viewport    viewport.Model
	textinput   textinput.Model
	logChan     <-chan domain.LogEntry
	errChan     <-chan error
	width       int
	height      int
	ready       bool
	searching   bool
	searchQuery string
	levelFilter domain.LogLevel
	paused      bool
	follow      bool
	showDetails bool
	stats       Stats
	appName     string
	simName     string
}

// Stats holds log statistics
type Stats struct {
	Total  int
	Errors int
	Faults int
}

// LogMsg is a message containing a new log entry
type LogMsg domain.LogEntry

// ErrMsg is a message containing an error
type ErrMsg error

// TickMsg triggers periodic updates
type TickMsg time.Time

// New creates a new TUI model
func New(appName, simName string, logChan <-chan domain.LogEntry, errChan <-chan error) Model {
	ti := textinput.New()
	ti.Placeholder = "Search logs..."
	ti.CharLimit = 100
	ti.Width = 40

		return Model{
			logs:        make([]domain.LogEntry, 0, 1000),
			filteredIdx: make([]int, 0, 1000),
			textinput:   ti,
			logChan:     logChan,
			errChan:     errChan,
			levelFilter: domain.LogLevelDebug, // Show all by default
			follow:      true,
			appName:     appName,
			simName:     simName,
		}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForLog(m.logChan),
		waitForError(m.errChan),
		tickCmd(),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.textinput.Blur()
				m.searchQuery = ""
				m.updateFilter()
			case "enter":
				m.searching = false
				m.textinput.Blur()
				m.searchQuery = m.textinput.Value()
				m.updateFilter()
			default:
				m.textinput, cmd = m.textinput.Update(msg)
				cmds = append(cmds, cmd)
			}
		} else {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "/":
				m.searching = true
				m.textinput.Focus()
				return m, textinput.Blink
			case "esc":
				if m.searchQuery != "" {
					m.searchQuery = ""
					m.textinput.SetValue("")
					m.updateFilter()
				}
				case "p", " ":
					m.paused = !m.paused
				case "f":
					m.follow = !m.follow
					if m.follow {
						m.viewport.GotoBottom()
					}
				case "d":
					m.showDetails = !m.showDetails
					m.updateFilter()
				case "c":
					m.logs = m.logs[:0]
					m.filteredIdx = m.filteredIdx[:0]
					m.stats = Stats{}
					m.content = ""
					m.updateViewport()
			case "1":
				m.levelFilter = domain.LogLevelDebug
				m.updateFilter()
			case "2":
				m.levelFilter = domain.LogLevelInfo
				m.updateFilter()
			case "3":
				m.levelFilter = domain.LogLevelDefault
				m.updateFilter()
			case "4":
				m.levelFilter = domain.LogLevelError
				m.updateFilter()
			case "5":
				m.levelFilter = domain.LogLevelFault
				m.updateFilter()
			case "g", "home":
				m.viewport.GotoTop()
			case "G", "end":
				m.viewport.GotoBottom()
			case "j", "down":
				m.viewport.LineDown(1)
			case "k", "up":
				m.viewport.LineUp(1)
			case "ctrl+d", "pgdown":
				m.viewport.HalfViewDown()
			case "ctrl+u", "pgup":
				m.viewport.HalfViewUp()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 3
		footerHeight := 2
		viewportHeight := m.height - headerHeight - footerHeight

		if !m.ready {
			m.viewport = viewport.New(m.width, viewportHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = viewportHeight
		}
		m.updateViewport()

		case LogMsg:
			if !m.paused {
				entry := domain.LogEntry(msg)
				m.logs = append(m.logs, entry)
				m.stats.Total++
				if entry.Level == domain.LogLevelError {
					m.stats.Errors++
				} else if entry.Level == domain.LogLevelFault {
					m.stats.Faults++
				}

				// Keep only last 10000 logs
				if len(m.logs) > 10000 {
					m.logs = m.logs[1000:]
					m.stats.Total = len(m.logs)
					// Recount errors/faults
					m.stats.Errors = 0
					m.stats.Faults = 0
					for _, l := range m.logs {
						if l.Level == domain.LogLevelError {
							m.stats.Errors++
						} else if l.Level == domain.LogLevelFault {
							m.stats.Faults++
						}
					}
					// Full recompute since indices shifted
					m.updateFilter()
				} else {
					// Incremental filter/update for new entry
					query := strings.ToLower(m.searchQuery)
					if m.entryMatches(entry, query) {
						m.filteredIdx = append(m.filteredIdx, len(m.logs)-1)
						line := m.formatLogLine(entry)
						if m.content == "" {
							m.content = line
						} else {
							m.content += "\n" + line
						}
						m.updateViewport()
					}
				}
			}
			cmds = append(cmds, waitForLog(m.logChan))

	case ErrMsg:
		// Handle error (could show in status bar)
		cmds = append(cmds, waitForError(m.errChan))

	case TickMsg:
		// Periodic refresh
		cmds = append(cmds, tickCmd())
	}

	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Header
	header := m.renderHeader()

	// Main content
	content := m.viewport.View()

	// Footer
	footer := m.renderFooter()

	return fmt.Sprintf("%s\n%s\n%s", header, content, footer)
}

func (m *Model) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Width(m.width)

	title := fmt.Sprintf("xcw: %s @ %s", m.appName, m.simName)
	if m.paused {
		title += " [PAUSED]"
	}
	if !m.follow {
		title += " [NO-FOLLOW]"
	}

	// Stats
	statsStr := fmt.Sprintf("Total: %d | Errors: %d | Faults: %d",
		m.stats.Total, m.stats.Errors, m.stats.Faults)

	// Level filter indicator
	levelStr := fmt.Sprintf("Level: %s+", m.levelFilter)

	header := titleStyle.Render(title)

	// Second line: stats and filter
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(m.width)

	var info string
	if m.stats.Errors > 0 || m.stats.Faults > 0 {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		faultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("201"))
		info = fmt.Sprintf("Total: %d | %s | %s | %s",
			m.stats.Total,
			errStyle.Render(fmt.Sprintf("Errors: %d", m.stats.Errors)),
			faultStyle.Render(fmt.Sprintf("Faults: %d", m.stats.Faults)),
			levelStr)
	} else {
		info = statsStr + " | " + levelStr
	}

	if m.searchQuery != "" {
		info += fmt.Sprintf(" | Search: %q", m.searchQuery)
	}

	return header + "\n" + infoStyle.Render(info)
}

func (m *Model) renderFooter() string {
	if m.searching {
		return m.textinput.View()
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(m.width)

	help := "q:quit /:search 1-5:level p:pause f:follow d:details c:clear g/G:top/bottom j/k:scroll"
	return helpStyle.Render(help)
}

func (m *Model) updateFilter() {
	m.filteredIdx = m.filteredIdx[:0]
	query := strings.ToLower(m.searchQuery)
	var b strings.Builder

	for i, log := range m.logs {
		if !m.entryMatches(log, query) {
			continue
		}
		m.filteredIdx = append(m.filteredIdx, i)
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(m.formatLogLine(log))
	}

	m.content = b.String()
	m.updateViewport()
}

func (m *Model) updateViewport() {
	if !m.ready {
		return
	}

	m.viewport.SetContent(m.content)

	// Auto-scroll to bottom when follow mode is on
	if m.follow {
		m.viewport.GotoBottom()
	}
}

// entryMatches applies current level/search filters for a single entry.
func (m *Model) entryMatches(log domain.LogEntry, query string) bool {
	if log.Level.Priority() < m.levelFilter.Priority() {
		return false
	}
	if query == "" {
		return true
	}
	msgLower := strings.ToLower(log.Message)
	processLower := strings.ToLower(log.Process)
	subsystemLower := strings.ToLower(log.Subsystem)
	return strings.Contains(msgLower, query) ||
		strings.Contains(processLower, query) ||
		strings.Contains(subsystemLower, query)
}

func (m *Model) formatLogLine(entry domain.LogEntry) string {
	// Time
	timeStr := entry.Timestamp.Format("15:04:05.000")
	timeStyle := output.Styles.Timestamp

	// Level indicator
	levelIndicator := output.LevelIndicator(string(entry.Level))

	// Process
	processStyle := output.Styles.Process
	processStr := processStyle.Render("[" + entry.Process + "]")

	// Message with level-appropriate styling
	msgStyle := output.LevelStyle(string(entry.Level))
	msg := entry.Message

	// Truncate message if too long
	maxMsgLen := m.width - 40
	if maxMsgLen < 20 {
		maxMsgLen = 20
	}
	if len(msg) > maxMsgLen {
		msg = msg[:maxMsgLen-3] + "..."
	}

	// Highlight search hits in message when searching
	if m.searchQuery != "" {
		msg = highlight(msg, m.searchQuery)
	}

	line := timeStyle.Render(timeStr) + " " + levelIndicator + " " + processStr
	if m.showDetails {
		line += " " + detailStyle.Render(fmt.Sprintf("pid:%d", entry.PID))
	}
	line += " "

	if entry.Subsystem != "" {
		subsystemStyle := output.Styles.Subsystem
		line += subsystemStyle.Render(entry.Subsystem)
		if entry.Category != "" {
			line += "/" + entry.Category
		}
		line += ": "
	}

	line += msgStyle.Render(msg)

	return line
}

func highlight(s, query string) string {
	if query == "" || s == "" {
		return s
	}
	qs := strings.ToLower(query)
	ls := strings.ToLower(s)
	var b strings.Builder
	for {
		idx := strings.Index(ls, qs)
		if idx < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:idx])
		b.WriteString(highlightStyle.Render(s[idx : idx+len(query)]))
		s = s[idx+len(query):]
		ls = ls[idx+len(query):]
	}
	return b.String()
}

// waitForLog creates a command that waits for a log entry
func waitForLog(ch <-chan domain.LogEntry) tea.Cmd {
	return func() tea.Msg {
		entry, ok := <-ch
		if !ok {
			return nil
		}
		return LogMsg(entry)
	}
}

// waitForError creates a command that waits for an error
func waitForError(ch <-chan error) tea.Cmd {
	return func() tea.Msg {
		err, ok := <-ch
		if !ok {
			return nil
		}
		return ErrMsg(err)
	}
}

// tickCmd creates a periodic tick command
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

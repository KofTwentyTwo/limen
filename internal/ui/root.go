// Package ui contains limen's Bubble Tea models.
package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/KofTwentyTwo/limen/internal/config"
	limenexec "github.com/KofTwentyTwo/limen/internal/exec"
	"github.com/KofTwentyTwo/limen/internal/format"
	"github.com/KofTwentyTwo/limen/internal/probe"
	"github.com/KofTwentyTwo/limen/internal/state"
	"github.com/KofTwentyTwo/limen/internal/ui/theme"
)

const (
	defaultWidth  = 100
	defaultHeight = 30

	tinyTerminalMessage = "terminal too small for limen UI; falling back to bare tmux new"
)

type stage int

const (
	stageHosts stage = iota
	stageSessions
	stageNewSession
)

// App is the data needed to start the TUI.
type App struct {
	Config       config.Config
	State        state.State
	ProbeResults <-chan probe.ProbeResult
	Now          func() time.Time
	Hostname     string
}

// Result is the selected command plan or quit outcome from the TUI.
type Result struct {
	Plan     limenexec.Plan
	HasPlan  bool
	ExitCode int
	Message  string
}

// Run starts the Bubble Tea program and returns the final user decision.
func Run(app App) (Result, error) {
	program := tea.NewProgram(NewModel(app), tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return Result{ExitCode: 1}, err
	}

	model, ok := finalModel.(Model)
	if !ok {
		return Result{ExitCode: 1}, fmt.Errorf("unexpected final model %T", finalModel)
	}
	return model.Result(), nil
}

// Model is the root Bubble Tea model for both picker stages.
type Model struct {
	stage         stage
	width         int
	height        int
	hosts         []hostEntry
	hostCursor    int
	sessionCursor int
	filter        string
	filtering     bool
	help          bool
	newName       string
	inputError    string
	probeResults  <-chan probe.ProbeResult
	now           func() time.Time
	result        Result
}

type hostEntry struct {
	Name         string
	DisplayName  string
	Connection   string
	Description  string
	Remote       *config.Host
	Probing      bool
	Reachable    bool
	Sessions     []probe.SessionInfo
	LastAttached time.Time
}

type sessionRow struct {
	New     bool
	Session probe.SessionInfo
}

type probeMsg probe.ProbeResult
type probesDoneMsg struct{}

// NewModel constructs the initial root model.
func NewModel(app App) Model {
	now := app.Now
	if now == nil {
		now = time.Now
	}

	hostname := app.Hostname
	if hostname == "" {
		if local, err := os.Hostname(); err == nil {
			hostname = local
		}
	}

	hosts := make([]hostEntry, 0, len(app.Config.Hosts)+1)
	hosts = append(hosts, hostEntry{
		Name:         probe.LocalhostName,
		DisplayName:  "localhost (" + shortHostname(hostname) + ")",
		Connection:   "localhost",
		Probing:      true,
		LastAttached: app.State.LastAttached[probe.LocalhostName],
	})
	for _, host := range app.Config.Hosts {
		hostCopy := host
		hosts = append(hosts, hostEntry{
			Name:         host.Name,
			DisplayName:  host.Name,
			Connection:   connectionLine(host),
			Description:  host.Description,
			Remote:       &hostCopy,
			Probing:      true,
			LastAttached: app.State.LastAttached[host.Name],
		})
	}

	probeResults := app.ProbeResults
	if probeResults == nil {
		probeResults = probe.Start(contextBackground(), app.Config.Hosts, probe.Options{})
	}

	return Model{
		width:        defaultWidth,
		height:       defaultHeight,
		hosts:        hosts,
		probeResults: probeResults,
		now:          now,
	}
}

func contextBackground() context.Context {
	return context.Background()
}

// Init starts listening for probe results.
func (m Model) Init() tea.Cmd {
	return waitForProbe(m.probeResults)
}

// Update applies keyboard, resize, and probe messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if msg.Width < 50 || msg.Height < 15 {
			m.acceptPlan(mustPlan(limenexec.LocalNew("")))
			m.result.Message = tinyTerminalMessage
			return m, tea.Quit
		}
		return m, nil
	case probeMsg:
		m.applyProbeResult(probe.ProbeResult(msg))
		m.clampCursor()
		return m, waitForProbe(m.probeResults)
	case probesDoneMsg:
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	default:
		return m, nil
	}
}

// View renders the active picker stage.
func (m Model) View() string {
	if m.width < 50 || m.height < 15 {
		return tinyTerminalMessage
	}

	var body string
	if m.width < 80 || m.height < 24 {
		body = m.singleColumnView()
	} else {
		body = m.splitView()
	}
	if m.help {
		body = body + "\n\n" + m.helpView()
	}
	return body + "\n\n" + m.footer()
}

// Result returns the final user decision.
func (m Model) Result() Result {
	return m.result
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.help {
		switch msg.String() {
		case "?", "esc":
			m.help = false
			return m, nil
		}
	}

	if m.filtering {
		return m.updateFilterKey(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		m.result.ExitCode = 130
		return m, tea.Quit
	case "q":
		m.result.ExitCode = 0
		return m, tea.Quit
	case "?":
		m.help = !m.help
		return m, nil
	case "/":
		if m.stage != stageNewSession {
			m.filtering = true
			m.filter = ""
		}
		return m, nil
	case "up", "k":
		m.moveCursor(-1)
		return m, nil
	case "down", "j":
		m.moveCursor(1)
		return m, nil
	case "home", "g":
		m.setCursor(0)
		return m, nil
	case "end", "G":
		m.setCursor(m.entryCount() - 1)
		return m, nil
	case "esc":
		return m.escape()
	case "enter":
		return m.enter()
	case "backspace":
		if m.stage == stageNewSession && len(m.newName) > 0 {
			m.newName = m.newName[:len(m.newName)-1]
			m.inputError = ""
		}
		return m, nil
	}

	if msg.Type == tea.KeyRunes && m.stage == stageNewSession {
		m.newName += string(msg.Runes)
		m.inputError = ""
	}
	return m, nil
}

func (m Model) updateFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.clampCursor()
		return m, nil
	case "enter":
		m.filtering = false
		return m.enter()
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.clampCursor()
		}
		return m, nil
	}
	if msg.Type == tea.KeyRunes {
		m.filter += string(msg.Runes)
		m.clampCursor()
	}
	return m, nil
}

func (m Model) enter() (tea.Model, tea.Cmd) {
	switch m.stage {
	case stageHosts:
		indexes := m.filteredHostIndexes()
		if len(indexes) == 0 {
			return m, nil
		}
		m.hostCursor = indexes[m.hostCursor]
		m.stage = stageSessions
		m.sessionCursor = 0
		m.filter = ""
		m.filtering = false
		return m, nil
	case stageSessions:
		rows := m.filteredSessionRows()
		if len(rows) == 0 {
			return m, nil
		}

		row := rows[m.sessionCursor]
		if row.New {
			m.stage = stageNewSession
			m.newName = ""
			m.inputError = ""
			m.filter = ""
			m.filtering = false
			return m, nil
		}
		host := m.selectedHost()
		session := row.Session
		if host.Remote == nil {
			m.acceptPlan(limenexec.LocalAttach(session.Name))
		} else {
			m.acceptPlan(limenexec.RemoteAttach(*host.Remote, session.Name))
		}
		return m, tea.Quit
	case stageNewSession:
		host := m.selectedHost()
		var plan limenexec.Plan
		var err error
		if host.Remote == nil {
			plan, err = limenexec.LocalNew(m.newName)
		} else {
			plan, err = limenexec.RemoteNew(*host.Remote, m.newName)
		}
		if err != nil {
			m.inputError = err.Error()
			return m, nil
		}
		m.acceptPlan(plan)
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m Model) escape() (tea.Model, tea.Cmd) {
	if m.stage == stageHosts {
		m.acceptPlan(mustPlan(limenexec.LocalNew("")))
		return m, tea.Quit
	}

	host := m.selectedHost()
	var plan limenexec.Plan
	if host.Remote == nil {
		plan = mustPlan(limenexec.LocalNew(""))
	} else {
		plan = mustPlan(limenexec.RemoteNew(*host.Remote, ""))
	}
	m.acceptPlan(plan)
	return m, tea.Quit
}

func (m *Model) acceptPlan(plan limenexec.Plan) {
	m.result = Result{Plan: plan, HasPlan: true, ExitCode: 0}
}

func (m *Model) applyProbeResult(result probe.ProbeResult) {
	for i := range m.hosts {
		if m.hosts[i].Name != result.HostName {
			continue
		}
		m.hosts[i].Probing = false
		m.hosts[i].Reachable = result.Reachable
		m.hosts[i].Sessions = result.Sessions
		return
	}
}

func waitForProbe(results <-chan probe.ProbeResult) tea.Cmd {
	return func() tea.Msg {
		result, ok := <-results
		if !ok {
			return probesDoneMsg{}
		}
		return probeMsg(result)
	}
}

func (m Model) splitView() string {
	leftWidth := maxInt(30, int(float64(m.width)*0.40))
	rightWidth := maxInt(30, m.width-leftWidth-4)
	height := maxInt(10, m.height-6)

	left := theme.Pane.Width(leftWidth).Height(height).Render(m.listView())
	right := theme.Pane.Width(rightWidth).Height(height).Render(m.detailView())
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func (m Model) singleColumnView() string {
	return theme.Pane.Width(maxInt(30, m.width-4)).Render(m.listView())
}

func (m Model) listView() string {
	var builder strings.Builder
	builder.WriteString(theme.Title.Render(m.leftTitle()))
	builder.WriteString("\n\n")

	switch m.stage {
	case stageHosts:
		m.writeHostRows(&builder)
	case stageSessions, stageNewSession:
		m.writeSessionRows(&builder)
	}
	return builder.String()
}

func (m Model) leftTitle() string {
	if m.stage == stageHosts {
		return "limen"
	}
	return m.selectedHost().DisplayName
}

func (m Model) writeHostRows(builder *strings.Builder) {
	indexes := m.filteredHostIndexes()
	if len(indexes) == 0 {
		builder.WriteString(theme.Dimmed.Render("no matching hosts"))
		return
	}
	for row, index := range indexes {
		host := m.hosts[index]
		cursor := " "
		if row == m.hostCursor {
			cursor = ">"
		}
		line := fmt.Sprintf("%s %s %-24s %s", cursor, statusDot(host), host.DisplayName, sessionSummary(host))
		if row == m.hostCursor {
			line = theme.Selected.Render(line)
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}
}

func (m Model) writeSessionRows(builder *strings.Builder) {
	rows := m.filteredSessionRows()
	if len(rows) == 0 {
		builder.WriteString(theme.Dimmed.Render("no matching sessions"))
		return
	}

	for row, entry := range rows {
		cursor := " "
		if row == m.sessionCursor {
			cursor = ">"
		}
		line := cursor + " " + sessionRowText(entry)
		if row == m.sessionCursor {
			line = theme.Selected.Render(line)
		}
		builder.WriteString(line)
		builder.WriteString("\n")
		if entry.New && len(rows) > 1 {
			builder.WriteString(theme.Dimmed.Render("──────────────────────────────"))
			builder.WriteString("\n")
		}
	}
}

func (m Model) detailView() string {
	if m.stage == stageNewSession {
		return m.newSessionView()
	}
	if m.stage == stageSessions {
		return m.sessionDetailView()
	}

	host := m.selectedHost()
	var builder strings.Builder
	builder.WriteString(theme.Title.Render("DETAILS"))
	builder.WriteString("\n\n")
	builder.WriteString(host.DisplayName)
	builder.WriteString("\n")
	builder.WriteString(theme.Dimmed.Render("──────────────────────────────────"))
	builder.WriteString("\n")
	builder.WriteString(host.Connection)
	builder.WriteString("\n\n")
	builder.WriteString("Status:        ")
	builder.WriteString(statusText(host))
	builder.WriteString("\n")
	builder.WriteString("Sessions:      ")
	builder.WriteString(sessionDetail(host))
	builder.WriteString("\n")
	builder.WriteString("Last attached: ")
	builder.WriteString(format.RelativeTime(host.LastAttached, m.now()))
	if host.Description != "" {
		builder.WriteString("\n\n")
		builder.WriteString(host.Description)
	}
	return builder.String()
}

func (m Model) sessionDetailView() string {
	rows := m.filteredSessionRows()
	if len(rows) == 0 {
		return theme.Title.Render("DETAILS") + "\n\n" + theme.Dimmed.Render("no matching sessions")
	}
	if rows[m.sessionCursor].New {
		return m.newSessionView()
	}

	session := rows[m.sessionCursor].Session
	var builder strings.Builder
	builder.WriteString(theme.Title.Render("DETAILS"))
	builder.WriteString("\n\n")
	builder.WriteString(session.Name)
	builder.WriteString("\n")
	builder.WriteString(theme.Dimmed.Render("──────────────────────────────────"))
	builder.WriteString("\n\n")
	builder.WriteString("Windows:    ")
	builder.WriteString(fmt.Sprintf("%d", session.Windows))
	builder.WriteString("\n")
	if !session.Created.IsZero() {
		builder.WriteString("Created:    ")
		builder.WriteString(session.Created.Format("Mon Jan 2 15:04"))
		builder.WriteString("\n")
	}
	builder.WriteString("Attached:   ")
	if session.Attached {
		builder.WriteString("★ currently attached")
	} else {
		builder.WriteString("no")
	}
	return builder.String()
}

func (m Model) newSessionView() string {
	var builder strings.Builder
	builder.WriteString(theme.Title.Render("+ New session"))
	builder.WriteString("\n\n")
	builder.WriteString("Name for the new session:\n\n")
	builder.WriteString("> ")
	builder.WriteString(m.newName)
	builder.WriteString("\n\n")
	if m.inputError != "" {
		builder.WriteString(m.inputError)
		builder.WriteString("\n")
	}
	builder.WriteString(theme.Dimmed.Render("Enter to create. Esc creates an unnamed session."))
	return builder.String()
}

func (m Model) helpView() string {
	escapeHelp := "esc skip"
	if m.stage != stageHosts {
		escapeHelp = "esc new unnamed"
	}
	return theme.Pane.Render("↑/k up   ↓/j down   enter connect   / search   " + escapeHelp + "   q quit   ctrl+c quit")
}

func (m Model) footer() string {
	if m.filtering {
		return theme.Dimmed.Render("/ " + m.filter + "   ·   enter select   ·   esc cancel filter")
	}
	if m.stage == stageNewSession {
		return theme.Dimmed.Render("enter create   ·   esc new unnamed   ·   q quit")
	}
	if m.stage == stageSessions {
		return theme.Dimmed.Render("↑↓ navigate   ·   enter connect   ·   / search   ·   ? help   ·   esc new unnamed")
	}
	return theme.Dimmed.Render("↑↓ navigate   ·   enter connect   ·   / search   ·   ? help   ·   esc skip")
}

func (m Model) selectedHost() hostEntry {
	if len(m.hosts) == 0 {
		return hostEntry{Name: probe.LocalhostName, DisplayName: "localhost", Connection: "localhost"}
	}
	if m.hostCursor < 0 || m.hostCursor >= len(m.hosts) {
		return m.hosts[0]
	}
	return m.hosts[m.hostCursor]
}

func (m Model) sessionEntries() []probe.SessionInfo {
	sessions := append([]probe.SessionInfo(nil), m.selectedHost().Sessions...)
	return sessions
}

func (m *Model) moveCursor(delta int) {
	m.setCursor(m.currentCursor() + delta)
}

func (m *Model) setCursor(value int) {
	count := m.entryCount()
	if count == 0 {
		m.hostCursor = 0
		m.sessionCursor = 0
		return
	}
	if value < 0 {
		value = 0
	}
	if value >= count {
		value = count - 1
	}
	switch m.stage {
	case stageHosts:
		m.hostCursor = value
	case stageSessions, stageNewSession:
		m.sessionCursor = value
	}
}

func (m *Model) clampCursor() {
	m.setCursor(m.currentCursor())
}

func (m Model) currentCursor() int {
	if m.stage == stageHosts {
		return m.hostCursor
	}
	return m.sessionCursor
}

func (m Model) entryCount() int {
	if m.stage == stageHosts {
		return len(m.filteredHostIndexes())
	}
	return len(m.filteredSessionRows())
}

func (m Model) filteredHostIndexes() []int {
	indexes := make([]int, 0, len(m.hosts))
	filter := strings.ToLower(m.filter)
	for i, host := range m.hosts {
		if filter == "" || strings.Contains(strings.ToLower(host.DisplayName), filter) || strings.Contains(strings.ToLower(host.Connection), filter) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (m Model) filteredSessionRows() []sessionRow {
	filter := strings.ToLower(m.filter)
	rows := make([]sessionRow, 0, len(m.sessionEntries())+1)

	if filter == "" || strings.Contains("new session", filter) {
		rows = append(rows, sessionRow{New: true})
	}
	for _, session := range m.sessionEntries() {
		if filter == "" || strings.Contains(strings.ToLower(session.Name), filter) {
			rows = append(rows, sessionRow{Session: session})
		}
	}
	return rows
}

func sessionRowText(row sessionRow) string {
	if row.New {
		return "+ New session"
	}

	marker := " "
	if row.Session.Attached {
		marker = "●"
	}
	star := ""
	if row.Session.Attached {
		star = " ★"
	}
	return fmt.Sprintf("%s %-18s %d windows%s", marker, row.Session.Name, row.Session.Windows, star)
}

func statusDot(host hostEntry) string {
	switch {
	case host.Probing:
		return theme.ProbingStyle.Render("…")
	case host.Reachable:
		return theme.OnlineStyle.Render("●")
	default:
		return theme.UnreachableStyle.Render("○")
	}
}

func statusText(host hostEntry) string {
	switch {
	case host.Probing:
		return theme.ProbingStyle.Render("… probing")
	case host.Reachable:
		return theme.OnlineStyle.Render("● online")
	default:
		return theme.UnreachableStyle.Render("○ unreachable")
	}
}

func sessionSummary(host hostEntry) string {
	if host.Probing {
		return "probing"
	}
	if !host.Reachable {
		return "unreach."
	}
	if len(host.Sessions) == 0 {
		return "—"
	}
	return fmt.Sprintf("%d sess.", len(host.Sessions))
}

func sessionDetail(host hostEntry) string {
	if host.Probing {
		return "probing"
	}
	if !host.Reachable {
		return "n/a"
	}
	if len(host.Sessions) == 0 {
		return "—"
	}
	names := make([]string, 0, len(host.Sessions))
	for _, session := range host.Sessions {
		names = append(names, session.Name)
	}
	return fmt.Sprintf("%d (%s)", len(host.Sessions), strings.Join(names, ", "))
}

func connectionLine(host config.Host) string {
	target := host.Hostname
	if host.User != "" {
		target = host.User + "@" + target
	}
	if host.Port != 0 && host.Port != 22 {
		target = fmt.Sprintf("%s:%d", target, host.Port)
	}
	return target
}

func shortHostname(hostname string) string {
	if hostname == "" {
		return "unknown"
	}
	return strings.SplitN(hostname, ".", 2)[0]
}

func mustPlan(plan limenexec.Plan, err error) limenexec.Plan {
	if err != nil {
		return limenexec.Plan{}
	}
	return plan
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

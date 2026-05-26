package ui

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/KofTwentyTwo/limen/internal/config"
	limenexec "github.com/KofTwentyTwo/limen/internal/exec"
	"github.com/KofTwentyTwo/limen/internal/probe"
	"github.com/KofTwentyTwo/limen/internal/state"
)

func TestTeatestHostSelectionAdvancesToSessionPicker(t *testing.T) {
	model := NewModel(testApp())
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(100, 30))
	defer func() {
		_ = tm.Quit()
	}()

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("localhost (renova)"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("+ New session"))
	}, teatest.WithDuration(2*time.Second))
}

func TestHostsAreSortedByDisplayName(t *testing.T) {
	model := NewModel(App{
		Config: config.Config{Hosts: []config.Host{
			{Name: "zeta", Hostname: "zeta.example.com"},
			{Name: "alpha", Hostname: "alpha.example.com"},
		}},
		ProbeResults: closedProbeResults(),
		Hostname:     "renova.local",
	})

	view := model.View()
	assertOrder(t, view, "alpha", "localhost (renova)", "zeta")
}

func TestSelectExistingRemoteSessionProducesAttachPlan(t *testing.T) {
	model := NewModel(testApp())
	model = updateModel(t, model, probeMsg(probe.ProbeResult{
		HostName:  "prod",
		Reachable: true,
		Sessions:  []probe.SessionInfo{{Name: "api", Windows: 3, Attached: true}},
	}))
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	want := limenexec.Plan{
		HostName: "prod",
		Argv:     []string{"ssh", "-t", "deploy@prod.example.com", "tmux", "attach", "-t", "api"},
	}
	if !model.result.HasPlan || !reflect.DeepEqual(model.result.Plan, want) {
		t.Fatalf("result = %#v, want plan %#v", model.result, want)
	}
}

func TestSessionsAreSortedByName(t *testing.T) {
	model := modelWithProdSessions(t,
		probe.SessionInfo{Name: "zeta", Windows: 1},
		probe.SessionInfo{Name: "api", Windows: 3},
		probe.SessionInfo{Name: "ops", Windows: 1},
	)
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	view := model.View()
	assertOrder(t, view, "+ New session", "api", "ops", "zeta")
}

func TestTypingAndTabCompleteHostSelection(t *testing.T) {
	model := NewModel(App{
		Config: config.Config{Hosts: []config.Host{
			{Name: "prod", Hostname: "prod.example.com"},
			{Name: "builder", Hostname: "builder.example.com"},
		}},
		ProbeResults: closedProbeResults(),
		Hostname:     "renova.local",
	})

	for _, r := range "pr" {
		model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = press(t, model, tea.KeyMsg{Type: tea.KeyTab})

	if !model.filtering || model.filter != "prod" {
		t.Fatalf("filtering = %v, filter = %q; want completed prod filter", model.filtering, model.filter)
	}

	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.stage != stageSessions {
		t.Fatalf("stage = %v, want sessions", model.stage)
	}
	if got := model.selectedHost().Name; got != "prod" {
		t.Fatalf("selected host = %q, want prod", got)
	}
}

func TestTypingBoundNavigationLetterStartsHostFilter(t *testing.T) {
	model := NewModel(App{
		Config: config.Config{Hosts: []config.Host{
			{Name: "gamma", Hostname: "gamma.example.com"},
			{Name: "prod", Hostname: "prod.example.com"},
		}},
		ProbeResults: closedProbeResults(),
		Hostname:     "renova.local",
	})

	model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	if !model.filtering || model.filter != "g" {
		t.Fatalf("filtering = %v, filter = %q; want g filter", model.filtering, model.filter)
	}
	if view := model.View(); strings.Contains(view, "prod") {
		t.Fatalf("View() = %q, want prod filtered out", view)
	}
}

func TestSessionFilterSelectsMatchingSession(t *testing.T) {
	model := modelWithProdSessions(t,
		probe.SessionInfo{Name: "api", Windows: 3},
		probe.SessionInfo{Name: "ops", Windows: 1},
	)
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "ops" {
		model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	want := limenexec.Plan{
		HostName: "prod",
		Argv:     []string{"ssh", "-t", "deploy@prod.example.com", "tmux", "attach", "-t", "ops"},
	}
	if !model.result.HasPlan || !reflect.DeepEqual(model.result.Plan, want) {
		t.Fatalf("result = %#v, want plan %#v", model.result, want)
	}
}

func TestTypingAndTabCompleteSessionSelection(t *testing.T) {
	model := modelWithProdSessions(t,
		probe.SessionInfo{Name: "api", Windows: 3},
		probe.SessionInfo{Name: "ops", Windows: 1},
	)
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	for _, r := range "op" {
		model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = press(t, model, tea.KeyMsg{Type: tea.KeyTab})

	if !model.filtering || model.filter != "ops" {
		t.Fatalf("filtering = %v, filter = %q; want completed ops filter", model.filtering, model.filter)
	}

	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	want := limenexec.Plan{
		HostName: "prod",
		Argv:     []string{"ssh", "-t", "deploy@prod.example.com", "tmux", "attach", "-t", "ops"},
	}
	if !model.result.HasPlan || !reflect.DeepEqual(model.result.Plan, want) {
		t.Fatalf("result = %#v, want plan %#v", model.result, want)
	}
}

func TestEscapeFromHostPickerProducesLocalNewPlan(t *testing.T) {
	model := NewModel(testApp())
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEsc})

	want := limenexec.Plan{HostName: probe.LocalhostName, Argv: []string{"tmux", "new"}}
	if !model.result.HasPlan || !reflect.DeepEqual(model.result.Plan, want) {
		t.Fatalf("result = %#v, want plan %#v", model.result, want)
	}
}

func TestEscapeFromRemoteSessionPickerProducesRemoteNewPlan(t *testing.T) {
	model := NewModel(testApp())
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEsc})

	want := limenexec.Plan{
		HostName: "prod",
		Argv:     []string{"ssh", "-t", "deploy@prod.example.com", "tmux", "new"},
	}
	if !model.result.HasPlan || !reflect.DeepEqual(model.result.Plan, want) {
		t.Fatalf("result = %#v, want plan %#v", model.result, want)
	}
}

func TestFilterSelectsMatchingHost(t *testing.T) {
	model := NewModel(testApp())
	model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "prod" {
		model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if model.stage != stageSessions {
		t.Fatalf("stage = %v, want sessions", model.stage)
	}
	if got := model.selectedHost().Name; got != "prod" {
		t.Fatalf("selected host = %q, want prod", got)
	}
}

func TestFilterCanBeCanceledAfterNoMatches(t *testing.T) {
	model := NewModel(testApp())
	model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "missing" {
		model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	if view := model.View(); !strings.Contains(view, "no matching hosts") {
		t.Fatalf("View() = %q, want no-match message", view)
	}

	model = press(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.filtering || model.filter != "" {
		t.Fatalf("filtering = %v, filter = %q; want canceled", model.filtering, model.filter)
	}
	if view := model.View(); !strings.Contains(view, "prod") {
		t.Fatalf("View() = %q, want unfiltered host list", view)
	}
}

func TestNewSessionPromptRejectsInvalidName(t *testing.T) {
	model := NewModel(testApp())
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	for _, r := range "bad:name" {
		model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if model.inputError != "name cannot contain : or ." {
		t.Fatalf("inputError = %q", model.inputError)
	}
	if model.result.HasPlan {
		t.Fatalf("result = %#v, want no plan", model.result)
	}
}

func TestNewSessionPromptCreatesTrimmedRemoteSession(t *testing.T) {
	model := NewModel(testApp())
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	for _, r := range " api " {
		model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	want := limenexec.Plan{
		HostName: "prod",
		Argv:     []string{"ssh", "-t", "deploy@prod.example.com", "tmux", "new", "-s", "api"},
	}
	if !model.result.HasPlan || !reflect.DeepEqual(model.result.Plan, want) {
		t.Fatalf("result = %#v, want plan %#v", model.result, want)
	}
}

func TestSessionPickerRendersSessionDetails(t *testing.T) {
	created := time.Date(2026, 5, 26, 13, 45, 0, 0, time.UTC)
	model := modelWithProdSessions(t, probe.SessionInfo{Name: "api", Windows: 3, Attached: true, Created: created})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})

	view := model.View()
	for _, want := range []string{"api", "Windows:    3", "Created:    Tue May 26 13:45", "currently attached"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

func TestProbeUpdatesClampSessionCursor(t *testing.T) {
	model := modelWithProdSessions(t,
		probe.SessionInfo{Name: "api", Windows: 3},
		probe.SessionInfo{Name: "ops", Windows: 1},
	)
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyEnd})
	model = updateModel(t, model, probeMsg(probe.ProbeResult{
		HostName:  "prod",
		Reachable: true,
		Sessions:  []probe.SessionInfo{{Name: "api", Windows: 3}},
	}))

	if model.sessionCursor != 1 {
		t.Fatalf("sessionCursor = %d, want clamped to 1", model.sessionCursor)
	}
	if view := model.View(); !strings.Contains(view, "api") {
		t.Fatalf("View() = %q, want remaining session", view)
	}
}

func TestHostDetailsRenderProbeStateAndStateTimestamp(t *testing.T) {
	model := modelWithProdSessions(t, probe.SessionInfo{Name: "api", Windows: 3})
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})

	view := model.View()
	for _, want := range []string{"deploy@prod.example.com", "online", "1 (api)", "1 hour ago", "Production application servers."} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

func TestUnreachableHostDetailsShowNoSessionInventory(t *testing.T) {
	model := NewModel(testApp())
	model = updateModel(t, model, probeMsg(probe.ProbeResult{HostName: "prod", Reachable: false}))
	model = press(t, model, tea.KeyMsg{Type: tea.KeyDown})

	view := model.View()
	for _, want := range []string{"unreachable", "Sessions:      n/a"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

func TestSingleColumnViewOmitsDetailsPane(t *testing.T) {
	model := updateModel(t, NewModel(testApp()), tea.WindowSizeMsg{Width: 70, Height: 20})

	view := model.View()
	if !strings.Contains(view, "limen") {
		t.Fatalf("View() = %q, want list pane", view)
	}
	if strings.Contains(view, "DETAILS") {
		t.Fatalf("View() = %q, want single-column layout without details pane", view)
	}
}

func TestHelpOverlayTogglesAndEscapeDismisses(t *testing.T) {
	model := NewModel(testApp())
	model = press(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !model.help {
		t.Fatal("help = false, want true")
	}
	if view := model.View(); !strings.Contains(view, "ctrl+c quit") {
		t.Fatalf("View() = %q, want help overlay", view)
	}

	model = press(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.help {
		t.Fatal("help = true, want false")
	}
	if model.result.HasPlan {
		t.Fatalf("result = %#v, want no plan after dismissing help", model.result)
	}
}

func TestQuitKeysReturnExitCodes(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
		want int
	}{
		{name: "q", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, want: 0},
		{name: "ctrl-c", key: tea.KeyMsg{Type: tea.KeyCtrlC}, want: 130},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := press(t, NewModel(testApp()), tt.key)
			if model.result.ExitCode != tt.want {
				t.Fatalf("exit code = %d, want %d", model.result.ExitCode, tt.want)
			}
			if model.result.HasPlan {
				t.Fatalf("result = %#v, want no plan", model.result)
			}
		})
	}
}

func TestTinyTerminalFallsBackToLocalNew(t *testing.T) {
	model := NewModel(testApp())
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 40, Height: 10})

	want := limenexec.Plan{HostName: probe.LocalhostName, Argv: []string{"tmux", "new"}}
	if !model.result.HasPlan || !reflect.DeepEqual(model.result.Plan, want) {
		t.Fatalf("result = %#v, want plan %#v", model.result, want)
	}
	if model.result.Message != tinyTerminalMessage {
		t.Fatalf("message = %q, want %q", model.result.Message, tinyTerminalMessage)
	}
	if model.View() != tinyTerminalMessage {
		t.Fatalf("View() = %q, want tiny terminal message", model.View())
	}
}

func TestWaitForProbeReturnsResultAndDoneMessages(t *testing.T) {
	probes := make(chan probe.ProbeResult, 1)
	probes <- probe.ProbeResult{HostName: "prod", Reachable: true}
	close(probes)

	got, ok := waitForProbe(probes)().(probeMsg)
	if !ok {
		t.Fatalf("first probe message type = %T, want probeMsg", got)
	}
	if got.HostName != "prod" || !got.Reachable {
		t.Fatalf("first probe message = %#v", got)
	}
	if got := waitForProbe(probes)(); got != (probesDoneMsg{}) {
		t.Fatalf("second probe message = %#v, want done", got)
	}
}

func testApp() App {
	return App{
		Config: config.Config{Hosts: []config.Host{{
			Name:        "prod",
			Hostname:    "prod.example.com",
			User:        "deploy",
			Description: "Production application servers.",
		}}},
		State: state.State{LastAttached: map[string]time.Time{
			"prod": time.Date(2026, 5, 26, 13, 45, 0, 0, time.UTC),
		}},
		ProbeResults: closedProbeResults(),
		Now:          func() time.Time { return time.Date(2026, 5, 26, 14, 45, 0, 0, time.UTC) },
		Hostname:     "renova.local",
	}
}

func closedProbeResults() <-chan probe.ProbeResult {
	probes := make(chan probe.ProbeResult)
	close(probes)
	return probes
}

func modelWithProdSessions(t *testing.T, sessions ...probe.SessionInfo) Model {
	t.Helper()
	return updateModel(t, NewModel(testApp()), probeMsg(probe.ProbeResult{
		HostName:  "prod",
		Reachable: true,
		Sessions:  sessions,
	}))
}

func assertOrder(t *testing.T, text string, values ...string) {
	t.Helper()

	last := -1
	for _, value := range values {
		index := strings.Index(text, value)
		if index == -1 {
			t.Fatalf("text missing %q:\n%s", value, text)
		}
		if index < last {
			t.Fatalf("%q appeared before prior value in:\n%s", value, text)
		}
		last = index
	}
}

func press(t *testing.T, model Model, key tea.KeyMsg) Model {
	t.Helper()
	return updateModel(t, model, key)
}

func updateModel(t *testing.T, model Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := model.Update(msg)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("updated model type = %T, want ui.Model", updated)
	}
	return next
}

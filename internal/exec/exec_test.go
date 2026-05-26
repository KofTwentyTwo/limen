package exec

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/KofTwentyTwo/limen/internal/config"
	"github.com/KofTwentyTwo/limen/internal/probe"
	"github.com/KofTwentyTwo/limen/internal/state"
)

func TestLocalAttachPlansTmuxAttach(t *testing.T) {
	got := LocalAttach("api")
	want := Plan{
		HostName: probe.LocalhostName,
		Argv:     []string{"tmux", "attach", "-t", "api"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LocalAttach() = %#v, want %#v", got, want)
	}
}

func TestLocalNewPlansUnnamedAndNamedSessions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Plan
	}{
		{
			name:  "unnamed",
			input: "  ",
			want:  Plan{HostName: probe.LocalhostName, Argv: []string{"tmux", "new"}},
		},
		{
			name:  "named",
			input: " build ",
			want:  Plan{HostName: probe.LocalhostName, Argv: []string{"tmux", "new", "-s", "build"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LocalNew(tt.input)
			if err != nil {
				t.Fatalf("LocalNew returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("LocalNew() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestRemoteAttachPlansSSHAttach(t *testing.T) {
	host := config.Host{Name: "prod", Hostname: "prod.example.com", User: "deploy", Port: 2222}

	got := RemoteAttach(host, "api")
	want := Plan{
		HostName: "prod",
		Argv: []string{
			"ssh", "-t", "-p", "2222", "deploy@prod.example.com",
			"tmux", "attach", "-t", "api",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoteAttach() = %#v, want %#v", got, want)
	}
}

func TestRemoteNewPlansSSHNew(t *testing.T) {
	host := config.Host{Name: "dev", Hostname: "dev.lan", User: "james"}

	got, err := RemoteNew(host, " work ")
	if err != nil {
		t.Fatalf("RemoteNew returned error: %v", err)
	}
	want := Plan{
		HostName: "dev",
		Argv:     []string{"ssh", "-t", "james@dev.lan", "tmux", "new", "-s", "work"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoteNew() = %#v, want %#v", got, want)
	}
}

func TestNewSessionPlannersReturnValidationErrors(t *testing.T) {
	if _, err := LocalNew("bad:name"); err == nil || err.Error() != "name cannot contain : or ." {
		t.Fatalf("LocalNew error = %v, want invalid name", err)
	}

	host := config.Host{Name: "dev", Hostname: "dev.lan"}
	if _, err := RemoteNew(host, "bad.name"); err == nil || err.Error() != "name cannot contain : or ." {
		t.Fatalf("RemoteNew error = %v, want invalid name", err)
	}
}

func TestNormalizeSessionName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantNamed bool
		wantErr   string
	}{
		{name: "empty creates unnamed", input: " \t ", wantNamed: false},
		{name: "trims whitespace", input: " Build ", wantName: "Build", wantNamed: true},
		{name: "rejects colon", input: "bad:name", wantErr: "name cannot contain : or ."},
		{name: "rejects dot", input: "bad.name", wantErr: "name cannot contain : or ."},
		{name: "rejects over cap", input: repeatRune('a', 65), wantErr: "name cannot exceed 64 characters"},
		{name: "accepts cap", input: repeatRune('a', 64), wantName: repeatRune('a', 64), wantNamed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotNamed, err := NormalizeSessionName(tt.input)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("NormalizeSessionName error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeSessionName returned error: %v", err)
			}
			if gotName != tt.wantName || gotNamed != tt.wantNamed {
				t.Fatalf("NormalizeSessionName() = (%q, %v), want (%q, %v)", gotName, gotNamed, tt.wantName, tt.wantNamed)
			}
		})
	}
}

func TestHandoffUpdatesStateBeforeExec(t *testing.T) {
	now := time.Date(2026, 5, 26, 13, 45, 0, 0, time.FixedZone("test", -5*60*60))
	var calls []string
	var saved state.State
	var execPath string
	var execArgv []string
	var execEnv []string
	var stderr bytes.Buffer

	code := Handoff(Plan{HostName: "prod", Argv: []string{"ssh", "-t", "prod", "tmux", "new"}}, Options{
		StatePath: "/tmp/state.json",
		Now:       func() time.Time { return now },
		Environ:   func() []string { return []string{"TERM=xterm-256color"} },
		Stderr:    &stderr,
		LoadState: func(path string) state.State {
			calls = append(calls, "load:"+path)
			return state.State{LastAttached: map[string]time.Time{"localhost": now.Add(-time.Hour)}}
		},
		SaveState: func(path string, st state.State) error {
			calls = append(calls, "save:"+path)
			saved = st
			return nil
		},
		Exec: func(path string, argv []string, env []string) error {
			calls = append(calls, "exec")
			execPath = path
			execArgv = append([]string(nil), argv...)
			execEnv = append([]string(nil), env...)
			return errors.New("exec unavailable")
		},
	})

	if code != 127 {
		t.Fatalf("Handoff exit code = %d, want 127", code)
	}
	wantCalls := []string{"load:/tmp/state.json", "save:/tmp/state.json", "exec"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", calls, wantCalls)
	}
	if got := saved.LastAttached["prod"]; !got.Equal(now.UTC()) {
		t.Fatalf("saved prod timestamp = %s, want %s", got, now.UTC())
	}
	if _, ok := saved.LastAttached["localhost"]; !ok {
		t.Fatal("existing localhost timestamp was not preserved")
	}
	if execPath != "ssh" || !reflect.DeepEqual(execArgv, []string{"ssh", "-t", "prod", "tmux", "new"}) {
		t.Fatalf("exec saw %q %#v", execPath, execArgv)
	}
	if !reflect.DeepEqual(execEnv, []string{"TERM=xterm-256color"}) {
		t.Fatalf("exec env = %#v", execEnv)
	}
	if got := stderr.String(); got != "limen: exec failed: exec unavailable\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestHandoffTreatsStateSaveFailureAsAdvisory(t *testing.T) {
	var calledExec bool

	code := Handoff(Plan{HostName: "prod", Argv: []string{"ssh"}}, Options{
		LoadState: func(path string) state.State {
			return state.State{}
		},
		SaveState: func(path string, st state.State) error {
			return errors.New("disk full")
		},
		Exec: func(path string, argv []string, env []string) error {
			calledExec = true
			return nil
		},
	})

	if code != 0 {
		t.Fatalf("Handoff exit code = %d, want 0", code)
	}
	if !calledExec {
		t.Fatal("exec was not called after advisory state save failure")
	}
}

func TestHandoffReportsEmptyArgvAsExecFailure(t *testing.T) {
	var stderr bytes.Buffer

	code := Handoff(Plan{HostName: "localhost"}, Options{Stderr: &stderr})

	if code != 127 {
		t.Fatalf("Handoff exit code = %d, want 127", code)
	}
	if got := stderr.String(); got != "limen: exec failed: empty argv\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func repeatRune(r rune, count int) string {
	var builder strings.Builder
	for i := 0; i < count; i++ {
		builder.WriteRune(r)
	}
	return builder.String()
}

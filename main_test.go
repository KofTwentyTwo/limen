package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	limenexec "github.com/KofTwentyTwo/limen/internal/exec"
	"github.com/KofTwentyTwo/limen/internal/ui"
)

func TestRunLoadsConfigStateAndHandsOffSelectedPlan(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "hosts.json")
	statePath := filepath.Join(dir, "state.json")

	if err := os.WriteFile(configPath, []byte(`{
		"hosts": [
			{"name": "prod", "hostname": "prod.example.com", "user": "deploy"}
		]
	}`), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	if err := os.WriteFile(statePath, []byte(`{"lastAttached":{"prod":"2026-05-26T13:45:00Z"}}`), 0o600); err != nil {
		t.Fatalf("write state fixture: %v", err)
	}

	wantPlan := limenexec.Plan{HostName: "prod", Argv: []string{"ssh", "-t", "prod.example.com", "tmux", "new"}}
	var sawHosts int
	var sawStateEntries int
	var handedPlan limenexec.Plan
	var handedOptions limenexec.Options

	code := runWithDeps([]string{"--config", configPath, "--state", statePath}, &bytes.Buffer{}, &bytes.Buffer{}, appDeps{
		runUI: func(app ui.App) (ui.Result, error) {
			sawHosts = len(app.Config.Hosts)
			sawStateEntries = len(app.State.LastAttached)
			return ui.Result{Plan: wantPlan, HasPlan: true}, nil
		},
		handoff: func(plan limenexec.Plan, opts limenexec.Options) int {
			handedPlan = plan
			handedOptions = opts
			return 0
		},
	})

	if code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	if sawHosts != 1 {
		t.Fatalf("UI saw %d hosts, want 1", sawHosts)
	}
	if sawStateEntries != 1 {
		t.Fatalf("UI saw %d state entries, want 1", sawStateEntries)
	}
	if !reflect.DeepEqual(handedPlan, wantPlan) {
		t.Fatalf("handoff plan = %#v, want %#v", handedPlan, wantPlan)
	}
	if handedOptions.StatePath != statePath {
		t.Fatalf("handoff state path = %q, want %q", handedOptions.StatePath, statePath)
	}
}

func TestRunPrintsResultMessageBeforeHandoff(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "hosts.json")
	if err := os.WriteFile(configPath, []byte(`{"hosts":[]}`), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	var stderr bytes.Buffer
	code := runWithDeps([]string{"--config", configPath, "--state", filepath.Join(t.TempDir(), "state.json")}, &bytes.Buffer{}, &stderr, appDeps{
		runUI: func(app ui.App) (ui.Result, error) {
			return ui.Result{
				Plan:    limenexec.Plan{HostName: "localhost", Argv: []string{"tmux", "new"}},
				HasPlan: true,
				Message: "terminal too small for limen UI; falling back to bare tmux new",
			}, nil
		},
		handoff: func(plan limenexec.Plan, opts limenexec.Options) int {
			return 0
		},
	})

	if code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	want := "terminal too small for limen UI; falling back to bare tmux new\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestRunWarnsForMissingConfigAndStillStartsUI(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "hosts.json")
	statePath := filepath.Join(t.TempDir(), "state.json")
	var stderr bytes.Buffer
	var sawHosts int

	code := runWithDeps([]string{"--config", configPath, "--state", statePath}, &bytes.Buffer{}, &stderr, appDeps{
		runUI: func(app ui.App) (ui.Result, error) {
			sawHosts = len(app.Config.Hosts)
			return ui.Result{ExitCode: 0}, nil
		},
		handoff: func(plan limenexec.Plan, opts limenexec.Options) int {
			t.Fatal("handoff should not be called")
			return 1
		},
	})

	if code != 0 {
		t.Fatalf("run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	wantWarning := "limen: no hosts.json found at " + configPath + "; only localhost will be available\n"
	if stderr.String() != wantWarning {
		t.Fatalf("stderr = %q, want %q", stderr.String(), wantWarning)
	}
	if sawHosts != 0 {
		t.Fatalf("UI saw %d configured hosts, want 0", sawHosts)
	}
}

func TestRunReturnsFailureForConfigErrors(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "hosts.json")
	if err := os.WriteFile(configPath, []byte(`{"hosts":[{"name":"localhost","hostname":"127.0.0.1"}]}`), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithDeps([]string{"--config", configPath, "--state", filepath.Join(t.TempDir(), "state.json")}, &stdout, &stderr, appDeps{})

	if code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	want := "limen: \"localhost\" is reserved\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestRunReturnsUsageFailureForFlagErrors(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--config"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); got == "" {
		t.Fatal("stderr is empty, want flag error")
	}
}

func TestRunReturnsUsageFailureForUnexpectedArguments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"extra"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	want := "limen: unexpected argument \"extra\"\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestRunReturnsVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	if stdout.String() != version+"\n" {
		t.Fatalf("stdout = %q, want version", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunReturnsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, "Usage of limen") {
		t.Fatalf("stderr = %q, want usage", got)
	}
}

func TestRunReturnsUIFailure(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "hosts.json")
	if err := os.WriteFile(configPath, []byte(`{"hosts":[]}`), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	var stderr bytes.Buffer
	code := runWithDeps([]string{"--config", configPath, "--state", filepath.Join(t.TempDir(), "state.json")}, &bytes.Buffer{}, &stderr, appDeps{
		runUI: func(app ui.App) (ui.Result, error) {
			return ui.Result{}, errors.New("terminal unavailable")
		},
	})

	if code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
	want := "limen: ui failed: terminal unavailable\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

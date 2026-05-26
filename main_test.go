package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunPrintsConfigAndStateAsJSON(t *testing.T) {
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

	if err := os.WriteFile(statePath, []byte(`{
		"lastAttached": {
			"prod": "2026-05-26T13:45:00Z"
		}
	}`), 0o600); err != nil {
		t.Fatalf("write state fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--config", configPath, "--state", statePath}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var got struct {
		Config struct {
			Hosts []struct {
				Name     string `json:"name"`
				Hostname string `json:"hostname"`
				User     string `json:"user,omitempty"`
			} `json:"hosts"`
		} `json:"config"`
		State struct {
			LastAttached map[string]time.Time `json:"lastAttached"`
		} `json:"state"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}

	if len(got.Config.Hosts) != 1 {
		t.Fatalf("len(hosts) = %d, want 1", len(got.Config.Hosts))
	}
	if got.Config.Hosts[0].Name != "prod" || got.Config.Hosts[0].Hostname != "prod.example.com" || got.Config.Hosts[0].User != "deploy" {
		t.Fatalf("host = %#v", got.Config.Hosts[0])
	}

	wantTime := time.Date(2026, 5, 26, 13, 45, 0, 0, time.UTC)
	if gotTime := got.State.LastAttached["prod"]; !gotTime.Equal(wantTime) {
		t.Fatalf("state timestamp = %s, want %s", gotTime, wantTime)
	}
}

func TestRunWarnsForMissingConfigAndPrintsEmptyHosts(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "hosts.json")
	statePath := filepath.Join(t.TempDir(), "state.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--config", configPath, "--state", statePath}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}

	wantWarning := "limen: no hosts.json found at " + configPath + "; only localhost will be available\n"
	if stderr.String() != wantWarning {
		t.Fatalf("stderr = %q, want %q", stderr.String(), wantWarning)
	}

	var got struct {
		Config struct {
			Hosts []struct{} `json:"hosts"`
		} `json:"config"`
		State struct {
			LastAttached map[string]time.Time `json:"lastAttached"`
		} `json:"state"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if len(got.Config.Hosts) != 0 {
		t.Fatalf("len(hosts) = %d, want 0", len(got.Config.Hosts))
	}
	if len(got.State.LastAttached) != 0 {
		t.Fatalf("len(lastAttached) = %d, want 0", len(got.State.LastAttached))
	}
}

func TestRunReturnsFailureForConfigErrors(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "hosts.json")
	if err := os.WriteFile(configPath, []byte(`{"hosts":[{"name":"localhost","hostname":"127.0.0.1"}]}`), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--config", configPath, "--state", filepath.Join(t.TempDir(), "state.json")}, &stdout, &stderr)

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

func TestRunReturnsFailureWhenJSONOutputCannotBeWritten(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "hosts.json")
	statePath := filepath.Join(t.TempDir(), "state.json")

	if err := os.WriteFile(configPath, []byte(`{"hosts":[]}`), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	var stderr bytes.Buffer
	code := run([]string{"--config", configPath, "--state", statePath}, failingWriter{}, &stderr)

	if code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}

	want := "limen: cannot write output: write failed\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

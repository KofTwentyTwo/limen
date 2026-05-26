package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsValidHostsFile(t *testing.T) {
	path := writeConfig(t, `{
		"hosts": [
			{
				"name": "prod",
				"hostname": "prod-server-01.example.com",
				"user": "deploy",
				"port": 2222,
				"description": "Production application servers."
			},
			{
				"name": "builder",
				"hostname": "builder.lan"
			}
		]
	}`)

	var stderr bytes.Buffer
	cfg, err := Load(path, &stderr)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if stderr.Len() != 0 {
		t.Fatalf("Load wrote unexpected warning: %q", stderr.String())
	}

	if len(cfg.Hosts) != 2 {
		t.Fatalf("len(cfg.Hosts) = %d, want 2", len(cfg.Hosts))
	}

	prod := cfg.Hosts[0]
	if prod.Name != "prod" ||
		prod.Hostname != "prod-server-01.example.com" ||
		prod.User != "deploy" ||
		prod.Port != 2222 ||
		prod.Description != "Production application servers." {
		t.Fatalf("first host = %#v", prod)
	}

	builder := cfg.Hosts[1]
	if builder.Name != "builder" || builder.Hostname != "builder.lan" {
		t.Fatalf("second host = %#v", builder)
	}
}

func TestLoadAllowsEmptyHostsArray(t *testing.T) {
	path := writeConfig(t, `{"hosts": []}`)

	var stderr bytes.Buffer
	cfg, err := Load(path, &stderr)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Hosts) != 0 {
		t.Fatalf("len(cfg.Hosts) = %d, want 0", len(cfg.Hosts))
	}
}

func TestLoadWithDefaultPath(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", t.TempDir())

	path := filepath.Join(configHome, "limen", "hosts.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"hosts": [{"name": "prod", "hostname": "prod.example.com"}]}`), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	var stderr bytes.Buffer
	cfg, err := Load("", &stderr)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Hosts) != 1 || cfg.Hosts[0].Name != "prod" {
		t.Fatalf("hosts = %#v, want prod host", cfg.Hosts)
	}
}

func TestLoadMissingConfigWarnsAndReturnsEmptyHosts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts.json")

	var stderr bytes.Buffer
	cfg, err := Load(path, &stderr)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Hosts) != 0 {
		t.Fatalf("len(cfg.Hosts) = %d, want 0", len(cfg.Hosts))
	}

	want := "limen: no hosts.json found at " + path + "; only localhost will be available\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestLoadMissingConfigWithNilStderrReturnsEmptyHosts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts.json")

	cfg, err := Load(path, nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Hosts) != 0 {
		t.Fatalf("len(cfg.Hosts) = %d, want 0", len(cfg.Hosts))
	}
}

func TestLoadUnreadableConfigIsFatal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts.json")

	_, err := os.Stat(filepath.Join(path, "missing-parent"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("test setup expected missing nested path, got %v", err)
	}

	var stderr bytes.Buffer
	_, err = Load(path, &stderr)
	if err != nil {
		t.Fatalf("missing file should warn, got error: %v", err)
	}

	dirAsFile := t.TempDir()
	_, err = Load(dirAsFile, &stderr)
	if err == nil {
		t.Fatal("Load returned nil error for unreadable directory path")
	}

	wantPrefix := "limen: cannot read " + dirAsFile + ": "
	if got := err.Error(); len(got) < len(wantPrefix) || got[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("error = %q, want prefix %q", got, wantPrefix)
	}
}

func TestLoadMalformedJSONIncludesPathLineColumnAndMessage(t *testing.T) {
	path := writeConfig(t, "{\n  \"hosts\": [\n")

	var stderr bytes.Buffer
	_, err := Load(path, &stderr)
	if err == nil {
		t.Fatal("Load returned nil error")
	}

	want := "limen: parse error at " + path + ":2:13: unexpected end of JSON input"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoadInvalidTopLevelShapeIncludesPathLineColumnAndMessage(t *testing.T) {
	path := writeConfig(t, `[]`)

	var stderr bytes.Buffer
	_, err := Load(path, &stderr)
	if err == nil {
		t.Fatal("Load returned nil error")
	}

	want := "limen: hosts must be an array in hosts.json"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoadRejectsMissingHostsProperty(t *testing.T) {
	path := writeConfig(t, `{}`)

	var stderr bytes.Buffer
	_, err := Load(path, &stderr)
	if err == nil {
		t.Fatal("Load returned nil error")
	}

	want := "limen: hosts must be an array in hosts.json"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoadRejectsHostsWhenNotArray(t *testing.T) {
	path := writeConfig(t, `{"hosts": {"name": "prod"}}`)

	var stderr bytes.Buffer
	_, err := Load(path, &stderr)
	if err == nil {
		t.Fatal("Load returned nil error")
	}

	want := "limen: hosts must be an array in hosts.json"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoadRejectsInvalidHostFieldTypes(t *testing.T) {
	path := writeConfig(t, `{"hosts": [{"name": "prod", "hostname": "prod.example.com", "port": "22"}]}`)

	var stderr bytes.Buffer
	_, err := Load(path, &stderr)
	if err == nil {
		t.Fatal("Load returned nil error")
	}

	want := "limen: invalid hosts.json: json: cannot unmarshal string into Go struct field Host.port of type int"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoadRejectsDuplicateHostNames(t *testing.T) {
	path := writeConfig(t, `{
		"hosts": [
			{"name": "prod", "hostname": "prod-a.example.com"},
			{"name": "prod", "hostname": "prod-b.example.com"}
		]
	}`)

	var stderr bytes.Buffer
	_, err := Load(path, &stderr)
	if err == nil {
		t.Fatal("Load returned nil error")
	}

	want := `limen: duplicate host name "prod" in hosts.json`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoadRejectsReservedLocalhostName(t *testing.T) {
	path := writeConfig(t, `{
		"hosts": [
			{"name": "localhost", "hostname": "127.0.0.1"}
		]
	}`)

	var stderr bytes.Buffer
	_, err := Load(path, &stderr)
	if err == nil {
		t.Fatal("Load returned nil error")
	}

	want := `limen: "localhost" is reserved`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoadRejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing name",
			body: `{"hosts": [{"hostname": "prod.example.com"}]}`,
			want: `limen: host #1 missing required field "name"`,
		},
		{
			name: "missing hostname",
			body: `{"hosts": [{"name": "prod"}]}`,
			want: `limen: host #1 missing required field "hostname"`,
		},
		{
			name: "blank name",
			body: `{"hosts": [{"name": " ", "hostname": "prod.example.com"}]}`,
			want: `limen: host #1 missing required field "name"`,
		},
		{
			name: "blank hostname",
			body: `{"hosts": [{"name": "prod", "hostname": "\t"}]}`,
			want: `limen: host #1 missing required field "hostname"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfig(t, tt.body)

			var stderr bytes.Buffer
			_, err := Load(path, &stderr)
			if err == nil {
				t.Fatal("Load returned nil error")
			}

			if err.Error() != tt.want {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestDefaultPathUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-config")
	t.Setenv("HOME", "/tmp/home")

	want := filepath.Join("/tmp/xdg-config", "limen", "hosts.json")
	if got := DefaultPath(); got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestDefaultPathFallsBackToHomeConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/home")

	want := filepath.Join("/tmp/home", ".config", "limen", "hosts.json")
	if got := DefaultPath(); got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "hosts.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	return path
}

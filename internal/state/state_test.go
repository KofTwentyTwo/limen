package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingFileReturnsEmptyState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	st := Load(path)

	if len(st.LastAttached) != 0 {
		t.Fatalf("len(st.LastAttached) = %d, want 0", len(st.LastAttached))
	}
}

func TestLoadMalformedFileReturnsEmptyState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"lastAttached":`), 0o600); err != nil {
		t.Fatalf("write malformed state: %v", err)
	}

	st := Load(path)

	if len(st.LastAttached) != 0 {
		t.Fatalf("len(st.LastAttached) = %d, want 0", len(st.LastAttached))
	}
}

func TestLoadInvalidTimestampReturnsEmptyState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"lastAttached":{"prod":"not-a-time"}}`), 0o600); err != nil {
		t.Fatalf("write invalid state: %v", err)
	}

	st := Load(path)

	if len(st.LastAttached) != 0 {
		t.Fatalf("len(st.LastAttached) = %d, want 0", len(st.LastAttached))
	}
}

func TestLoadWithNullLastAttachedReturnsEmptyState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"lastAttached":null}`), 0o600); err != nil {
		t.Fatalf("write null state: %v", err)
	}

	st := Load(path)

	if len(st.LastAttached) != 0 {
		t.Fatalf("len(st.LastAttached) = %d, want 0", len(st.LastAttached))
	}
}

func TestLoadWithDefaultPath(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("HOME", t.TempDir())

	path := filepath.Join(dataHome, "limen", "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create state dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"lastAttached":{"prod":"2026-05-26T13:45:00Z"}}`), 0o600); err != nil {
		t.Fatalf("write state fixture: %v", err)
	}

	st := Load("")

	want := time.Date(2026, 5, 26, 13, 45, 0, 0, time.UTC)
	if got := st.LastAttached["prod"]; !got.Equal(want) {
		t.Fatalf("prod timestamp = %s, want %s", got, want)
	}
}

func TestSaveRoundTripsState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	attachedAt := time.Date(2026, 5, 26, 13, 45, 0, 0, time.UTC)

	err := Save(path, State{
		LastAttached: map[string]time.Time{
			"localhost": attachedAt,
			"prod":      attachedAt.Add(-2 * time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	st := Load(path)
	if got := st.LastAttached["localhost"]; !got.Equal(attachedAt) {
		t.Fatalf("localhost timestamp = %s, want %s", got, attachedAt)
	}
	if got := st.LastAttached["prod"]; !got.Equal(attachedAt.Add(-2 * time.Hour)) {
		t.Fatalf("prod timestamp = %s, want %s", got, attachedAt.Add(-2*time.Hour))
	}
}

func TestSaveWithDefaultPath(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("HOME", t.TempDir())

	attachedAt := time.Date(2026, 5, 26, 13, 45, 0, 0, time.UTC)
	if err := Save("", State{LastAttached: map[string]time.Time{"prod": attachedAt}}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	path := filepath.Join(dataHome, "limen", "state.json")
	st := Load(path)
	if got := st.LastAttached["prod"]; !got.Equal(attachedAt) {
		t.Fatalf("prod timestamp = %s, want %s", got, attachedAt)
	}
}

func TestSaveReturnsErrorWhenParentPathIsFile(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parent, []byte("x"), 0o600); err != nil {
		t.Fatalf("write parent fixture: %v", err)
	}

	err := Save(filepath.Join(parent, "state.json"), State{})

	if err == nil {
		t.Fatal("Save returned nil error")
	}
}

func TestSaveReturnsErrorWhenTemporaryPathIsDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.Mkdir(filepath.Join(dir, "state.json.tmp"), 0o700); err != nil {
		t.Fatalf("create temporary directory fixture: %v", err)
	}

	err := Save(path, State{})

	if err == nil {
		t.Fatal("Save returned nil error")
	}
}

func TestSaveReturnsErrorWhenTargetPathIsDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("create target directory fixture: %v", err)
	}

	err := Save(path, State{})

	if err == nil {
		t.Fatal("Save returned nil error")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "state.json.tmp")); !os.IsNotExist(statErr) {
		t.Fatalf("temporary file cleanup error = %v, want not exist", statErr)
	}
}

func TestSaveCreatesPrivateDirectoryAndFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "limen", "state.json")

	if err := Save(path, State{}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat state dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("state dir mode = %o, want 700", got)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("state file mode = %o, want 600", got)
	}
}

func TestSaveReplacesStaleTemporaryFileWithPrivateMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	tmp := filepath.Join(dir, "state.json.tmp")
	if err := os.WriteFile(tmp, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale temporary file: %v", err)
	}

	if err := Save(path, State{}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("state file mode = %o, want 600", got)
	}
}

func TestSaveUsesTemporaryFileInSameDirectoryThenRenames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	tmp := filepath.Join(dir, "state.json.tmp")

	if err := Save(path, State{}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("temporary file still exists, err = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}

	var decoded struct {
		LastAttached map[string]time.Time `json:"lastAttached"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("state file is not valid JSON: %v", err)
	}
	if decoded.LastAttached == nil {
		t.Fatal("lastAttached decoded as nil, want empty object")
	}
}

func TestDefaultPathUsesXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")
	t.Setenv("HOME", "/tmp/home")

	want := filepath.Join("/tmp/xdg-data", "limen", "state.json")
	if got := DefaultPath(); got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestDefaultPathFallsBackToHomeLocalShare(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/tmp/home")

	want := filepath.Join("/tmp/home", ".local", "share", "limen", "state.json")
	if got := DefaultPath(); got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

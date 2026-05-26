// Package state persists advisory last-attached timestamps.
package state

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	stateDirName  = "limen"
	stateFileName = "state.json"
)

// State is the top-level state.json document.
type State struct {
	LastAttached map[string]time.Time `json:"lastAttached"`
}

// DefaultPath returns the design-specified state.json path.
func DefaultPath() string {
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(xdgDataHome, stateDirName, stateFileName)
	}

	home := os.Getenv("HOME")
	return filepath.Join(home, ".local", "share", stateDirName, stateFileName)
}

// Load reads state.json. State is advisory, so any read or decode failure
// returns an empty state instead of blocking startup.
func Load(path string) State {
	if path == "" {
		path = DefaultPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return empty()
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return empty()
	}
	if st.LastAttached == nil {
		st.LastAttached = map[string]time.Time{}
	}
	return st
}

// Save writes state.json atomically using a temporary file in the same
// directory, then renaming it into place.
func Save(path string, st State) error {
	if path == "" {
		path = DefaultPath()
	}

	if st.LastAttached == nil {
		st.LastAttached = map[string]time.Time{}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(st); err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := writePrivateFile(tmpPath, data.Bytes()); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}

	return nil
}

func writePrivateFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(path)
		return writeErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return closeErr
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = os.Remove(path)
		return err
	}

	return nil
}

func empty() State {
	return State{LastAttached: map[string]time.Time{}}
}

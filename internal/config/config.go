// Package config loads and validates limen's user-managed hosts.json file.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDirName  = "limen"
	configFileName = "hosts.json"
)

// Config is the top-level hosts.json document.
type Config struct {
	Hosts []Host `json:"hosts"`
}

// Host describes a remote SSH target declared by the user.
type Host struct {
	Name        string `json:"name"`
	Hostname    string `json:"hostname"`
	User        string `json:"user,omitempty"`
	Port        int    `json:"port,omitempty"`
	Description string `json:"description,omitempty"`
}

// DefaultPath returns the design-specified hosts.json path.
func DefaultPath() string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, configDirName, configFileName)
	}

	home := os.Getenv("HOME")
	return filepath.Join(home, ".config", configDirName, configFileName)
}

// Load reads hosts.json from path. A missing file is non-fatal and leaves only
// the implicit localhost available to later application layers.
func Load(path string, stderr io.Writer) (Config, error) {
	if path == "" {
		path = DefaultPath()
	}
	if stderr == nil {
		stderr = io.Discard
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "limen: no hosts.json found at %s; only localhost will be available\n", path)
			return Config{Hosts: []Host{}}, nil
		}
		return Config{}, fmt.Errorf("limen: cannot read %s: %v", path, err)
	}

	cfg, err := parse(data, path)
	if err != nil {
		return Config{}, err
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func parse(data []byte, path string) (Config, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		var syntaxError *json.SyntaxError
		if errors.As(err, &syntaxError) {
			return Config{}, parseError(path, data, err)
		}
		return Config{}, errors.New("limen: hosts must be an array in hosts.json")
	}

	rawHosts := document["hosts"]
	if len(rawHosts) == 0 || !bytes.HasPrefix(bytes.TrimSpace(rawHosts), []byte("[")) {
		return Config{}, errors.New("limen: hosts must be an array in hosts.json")
	}

	var hosts []Host
	if err := json.Unmarshal(rawHosts, &hosts); err != nil {
		return Config{}, fmt.Errorf("limen: invalid hosts.json: %v", err)
	}
	return Config{Hosts: hosts}, nil
}

func parseError(path string, data []byte, err error) error {
	var syntaxError *json.SyntaxError
	line, column := 1, 1
	if errors.As(err, &syntaxError) {
		line, column = lineColumn(data, syntaxError.Offset)
	} else {
		line, column = lineColumn(data, int64(len(data)))
	}

	return fmt.Errorf("limen: parse error at %s:%d:%d: %v", path, line, column, err)
}

func lineColumn(data []byte, offset int64) (int, int) {
	if offset < 1 {
		return 1, 1
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}

	line, column := 1, 1
	for i, b := range data {
		if int64(i) >= offset-1 {
			break
		}
		if b == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}

func validate(cfg Config) error {
	seenNames := make(map[string]struct{}, len(cfg.Hosts))

	for i, host := range cfg.Hosts {
		number := i + 1
		if strings.TrimSpace(host.Name) == "" {
			return fmt.Errorf(`limen: host #%d missing required field "name"`, number)
		}
		if strings.TrimSpace(host.Hostname) == "" {
			return fmt.Errorf(`limen: host #%d missing required field "hostname"`, number)
		}
		if host.Name == "localhost" {
			return errors.New(`limen: "localhost" is reserved`)
		}
		if _, exists := seenNames[host.Name]; exists {
			return fmt.Errorf(`limen: duplicate host name "%s" in hosts.json`, host.Name)
		}
		seenNames[host.Name] = struct{}{}
	}

	return nil
}

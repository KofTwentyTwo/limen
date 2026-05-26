// Package tmuxinfo parses tmux session inventory output.
package tmuxinfo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// LocalFormat is the tmux format used when probing localhost.
	LocalFormat = "#{session_name}|#{session_windows}|#{session_attached}|#{session_created}"

	// RemoteFormat is the tmux format used when probing remote hosts.
	RemoteFormat = "#{session_name}|#{session_windows}|#{session_attached}"
)

// ErrNoServer marks tmux's normal "no server running" response.
var ErrNoServer = errors.New("tmuxinfo: no tmux server running")

// SessionInfo is the tmux session shape used by probe results.
type SessionInfo struct {
	Name        string
	Windows     int
	WindowNames []string
	Attached    bool
	Created     time.Time
}

// ParseLocal parses localhost tmux ls output.
func ParseLocal(output string) ([]SessionInfo, error) {
	return parse(output, "local", 4)
}

// ParseRemote parses remote tmux ls output.
func ParseRemote(output string) ([]SessionInfo, error) {
	return parse(output, "remote", 3)
}

// IsNoServerRunning reports whether output is tmux's no-server diagnostic.
func IsNoServerRunning(output string) bool {
	return strings.Contains(output, "no server running on /tmp/tmux-")
}

func parse(output string, variant string, fieldCount int) ([]SessionInfo, error) {
	if IsNoServerRunning(output) {
		return nil, ErrNoServer
	}

	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return []SessionInfo{}, nil
	}

	lines := strings.Split(trimmed, "\n")
	sessions := make([]SessionInfo, 0, len(lines))
	for i, line := range lines {
		session, err := parseLine(line, variant, fieldCount, i+1)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func parseLine(line string, variant string, fieldCount int, row int) (SessionInfo, error) {
	fields := strings.Split(line, "|")
	if len(fields) != fieldCount {
		return SessionInfo{}, fmt.Errorf("tmuxinfo: malformed %s row %d: expected %d fields, got %d", variant, row, fieldCount, len(fields))
	}

	name := fields[0]
	if name == "" {
		return SessionInfo{}, fmt.Errorf("tmuxinfo: %s row %d has empty session name", variant, row)
	}

	windows, err := parseNonNegativeInt(fields[1], variant, row, "window count")
	if err != nil {
		return SessionInfo{}, err
	}

	attachedCount, err := parseNonNegativeInt(fields[2], variant, row, "attached count")
	if err != nil {
		return SessionInfo{}, err
	}

	session := SessionInfo{
		Name:     name,
		Windows:  windows,
		Attached: attachedCount > 0,
	}

	if fieldCount == 4 {
		created, err := parseNonNegativeTimestamp(fields[3], variant, row)
		if err != nil {
			return SessionInfo{}, err
		}
		session.Created = time.Unix(created, 0).UTC()
	}

	return session, nil
}

func parseNonNegativeInt(value string, variant string, row int, label string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("tmuxinfo: %s row %d has invalid %s %q: %w", variant, row, label, value, err)
	}
	if number < 0 {
		return 0, fmt.Errorf("tmuxinfo: %s row %d has invalid %s %q", variant, row, label, value)
	}
	return number, nil
}

func parseNonNegativeTimestamp(value string, variant string, row int) (int64, error) {
	created, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("tmuxinfo: %s row %d has invalid created timestamp %q: %w", variant, row, value, err)
	}
	if created < 0 {
		return 0, fmt.Errorf("tmuxinfo: %s row %d has invalid created timestamp %q", variant, row, value)
	}
	return created, nil
}

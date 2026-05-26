// Package exec plans and performs the final process handoff.
package exec

import (
	"errors"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/KofTwentyTwo/limen/internal/config"
	"github.com/KofTwentyTwo/limen/internal/probe"
	"github.com/KofTwentyTwo/limen/internal/ssh"
	"github.com/KofTwentyTwo/limen/internal/state"
)

const maxSessionNameLength = 64

// ErrEmptyArgv reports an invalid handoff plan.
var ErrEmptyArgv = errors.New("empty argv")

// Plan is the final command vector and the host it targets.
type Plan struct {
	HostName string
	Argv     []string
}

// SyscallExec is the injectable boundary around syscall.Exec.
type SyscallExec func(path string, argv []string, env []string) error

// Options supplies dependencies for Handoff.
type Options struct {
	StatePath string
	Now       func() time.Time
	Environ   func() []string
	Stderr    io.Writer
	LoadState func(string) state.State
	SaveState func(string, state.State) error
	Exec      SyscallExec
}

// LocalAttach plans `tmux attach -t <session>` for localhost.
func LocalAttach(session string) Plan {
	return Plan{
		HostName: probe.LocalhostName,
		Argv:     ssh.LocalTmuxArgv("attach", "-t", session),
	}
}

// LocalNew plans `tmux new`, optionally with `-s <name>`.
func LocalNew(name string) (Plan, error) {
	args, err := newSessionArgs(name)
	if err != nil {
		return Plan{}, err
	}
	return Plan{
		HostName: probe.LocalhostName,
		Argv:     ssh.LocalTmuxArgv(args...),
	}, nil
}

// RemoteAttach plans `ssh -t <host> tmux attach -t <session>`.
func RemoteAttach(host config.Host, session string) Plan {
	return Plan{
		HostName: host.Name,
		Argv:     ssh.SessionArgv(host, "attach", "-t", session),
	}
}

// RemoteNew plans `ssh -t <host> tmux new`, optionally with `-s <name>`.
func RemoteNew(host config.Host, name string) (Plan, error) {
	args, err := newSessionArgs(name)
	if err != nil {
		return Plan{}, err
	}
	return Plan{
		HostName: host.Name,
		Argv:     ssh.SessionArgv(host, args...),
	}, nil
}

// NormalizeSessionName trims and validates a name for `tmux new -s`.
// Empty input is valid and means tmux should create an unnamed session.
func NormalizeSessionName(name string) (string, bool, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", false, nil
	}
	if strings.ContainsAny(normalized, ":.") {
		return "", false, errors.New("name cannot contain : or .")
	}
	if len([]rune(normalized)) > maxSessionNameLength {
		return "", false, fmt.Errorf("name cannot exceed %d characters", maxSessionNameLength)
	}
	return normalized, true, nil
}

// Handoff records the destination host, then replaces the current process.
// State is advisory: a write failure must not strand the user before exec.
func Handoff(plan Plan, opts Options) int {
	deps := opts.withDefaults()
	if len(plan.Argv) == 0 {
		fmt.Fprintf(deps.Stderr, "limen: exec failed: %v\n", ErrEmptyArgv)
		return 127
	}

	st := deps.LoadState(deps.StatePath)
	if st.LastAttached == nil {
		st.LastAttached = map[string]time.Time{}
	}
	st.LastAttached[plan.HostName] = deps.Now().UTC()
	_ = deps.SaveState(deps.StatePath, st)

	execPath, err := osexec.LookPath(plan.Argv[0])
	if err != nil {
		fmt.Fprintf(deps.Stderr, "limen: exec failed: %v\n", err)
		return 127
	}

	if err := deps.Exec(execPath, plan.Argv, deps.Environ()); err != nil {
		fmt.Fprintf(deps.Stderr, "limen: exec failed: %v\n", err)
		return 127
	}

	return 0
}

func newSessionArgs(name string) ([]string, error) {
	normalized, named, err := NormalizeSessionName(name)
	if err != nil {
		return nil, err
	}
	if !named {
		return []string{"new"}, nil
	}
	return []string{"new", "-s", normalized}, nil
}

func (opts Options) withDefaults() Options {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Environ == nil {
		opts.Environ = os.Environ
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.LoadState == nil {
		opts.LoadState = state.Load
	}
	if opts.SaveState == nil {
		opts.SaveState = state.Save
	}
	if opts.Exec == nil {
		opts.Exec = syscall.Exec
	}
	return opts
}

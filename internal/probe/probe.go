// Package probe discovers tmux sessions on localhost and configured remotes.
package probe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/KofTwentyTwo/limen/internal/config"
	"github.com/KofTwentyTwo/limen/internal/ssh"
	"github.com/KofTwentyTwo/limen/internal/tmuxinfo"
)

const (
	// LocalhostName is the synthetic host name used for the local machine.
	LocalhostName = "localhost"

	defaultProbeTimeout = 3 * time.Second
)

// ProbeResult is one host's in-memory probe outcome.
type ProbeResult struct {
	HostName  string
	Reachable bool
	Sessions  []SessionInfo
	Error     error
	Latency   time.Duration
}

// SessionInfo aliases tmux session metadata for probe consumers.
type SessionInfo = tmuxinfo.SessionInfo

// CommandResult captures a completed tmux or ssh probe command.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// Runner executes commands for probes.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) CommandResult
}

// RunnerFunc adapts a function into a Runner.
type RunnerFunc func(ctx context.Context, name string, args ...string) CommandResult

// Run executes f.
func (f RunnerFunc) Run(ctx context.Context, name string, args ...string) CommandResult {
	return f(ctx, name, args...)
}

// Options holds probe dependencies and timing knobs.
type Options struct {
	LookPath func(string) (string, error)
	Runner   Runner
	Timeout  time.Duration
}

// Localhost probes the local tmux server.
func Localhost(ctx context.Context, opts Options) ProbeResult {
	deps := opts.withDefaults()
	start := time.Now()
	probeCtx, cancel := context.WithTimeout(ctx, deps.Timeout)
	defer cancel()

	tmuxPath, err := deps.LookPath("tmux")
	if err != nil {
		return ProbeResult{
			HostName:  LocalhostName,
			Reachable: false,
			Error:     fmt.Errorf("tmux not found: %w", err),
			Latency:   elapsed(start),
		}
	}

	result := deps.Runner.Run(probeCtx, tmuxPath, "ls", "-F", tmuxinfo.LocalFormat)
	return interpret(LocalhostName, result, tmuxinfo.ParseLocal, start)
}

// Remote probes a configured remote host through ssh.
func Remote(ctx context.Context, host config.Host, opts Options) ProbeResult {
	deps := opts.withDefaults()
	start := time.Now()

	probeCtx, cancel := context.WithTimeout(ctx, deps.Timeout)
	defer cancel()

	argv := ssh.ProbeArgv(host, tmuxinfo.RemoteFormat)
	result := deps.Runner.Run(probeCtx, argv[0], argv[1:]...)
	return interpret(host.Name, result, tmuxinfo.ParseRemote, start)
}

// Start begins one probe per host and emits results as each one completes.
func Start(ctx context.Context, hosts []config.Host, opts Options) <-chan ProbeResult {
	results := make(chan ProbeResult)

	go func() {
		defer close(results)

		var wg sync.WaitGroup
		send := func(result ProbeResult) {
			select {
			case results <- result:
			case <-ctx.Done():
			}
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			send(Localhost(ctx, opts))
		}()

		for _, host := range hosts {
			host := host
			wg.Add(1)
			go func() {
				defer wg.Done()
				send(Remote(ctx, host, opts))
			}()
		}

		wg.Wait()
	}()

	return results
}

func interpret(hostName string, result CommandResult, parse func(string) ([]tmuxinfo.SessionInfo, error), start time.Time) ProbeResult {
	combinedOutput := result.Stdout + result.Stderr

	if errors.Is(result.Err, context.DeadlineExceeded) || errors.Is(result.Err, context.Canceled) {
		return unreachable(hostName, result.Err, start)
	}

	if result.ExitCode == 1 && tmuxinfo.IsNoServerRunning(combinedOutput) {
		return reachable(hostName, []tmuxinfo.SessionInfo{}, start)
	}

	if result.Err != nil && result.ExitCode == 0 {
		return unreachable(hostName, commandError(result), start)
	}

	if result.ExitCode != 0 {
		return unreachable(hostName, commandError(result), start)
	}

	sessions, err := parse(result.Stdout)
	if err != nil {
		if errors.Is(err, tmuxinfo.ErrNoServer) {
			return reachable(hostName, []tmuxinfo.SessionInfo{}, start)
		}
		return unreachable(hostName, err, start)
	}

	return reachable(hostName, sessions, start)
}

func reachable(hostName string, sessions []tmuxinfo.SessionInfo, start time.Time) ProbeResult {
	return ProbeResult{
		HostName:  hostName,
		Reachable: true,
		Sessions:  sessions,
		Latency:   elapsed(start),
	}
}

func unreachable(hostName string, err error, start time.Time) ProbeResult {
	return ProbeResult{
		HostName:  hostName,
		Reachable: false,
		Sessions:  []tmuxinfo.SessionInfo{},
		Error:     err,
		Latency:   elapsed(start),
	}
}

func commandError(result CommandResult) error {
	message := result.Stderr
	if message == "" {
		message = result.Stdout
	}
	if message == "" && result.Err != nil {
		return result.Err
	}
	if message == "" {
		return fmt.Errorf("command exited with status %d", result.ExitCode)
	}
	return errors.New(message)
}

func elapsed(start time.Time) time.Duration {
	duration := time.Since(start)
	if duration <= 0 {
		return time.Nanosecond
	}
	return duration
}

func (opts Options) withDefaults() Options {
	if opts.Timeout == 0 {
		opts.Timeout = defaultProbeTimeout
	}
	if opts.LookPath == nil {
		opts.LookPath = exec.LookPath
	}
	if opts.Runner == nil {
		opts.Runner = execRunner{}
	}
	return opts
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) CommandResult {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}
	if err == nil {
		return result
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		result.Err = ctxErr
	}

	return result
}

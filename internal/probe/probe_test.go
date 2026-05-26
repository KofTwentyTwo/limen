package probe

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KofTwentyTwo/limen/internal/config"
	"github.com/KofTwentyTwo/limen/internal/tmuxinfo"
)

func TestLocalhostReturnsSessionsWhenTmuxSucceeds(t *testing.T) {
	runner := &recordingRunner{
		result: CommandResult{
			Stdout:   "api|3|1|1779799500\n",
			ExitCode: 0,
		},
	}

	got := Localhost(context.Background(), Options{
		LookPath: lookPath("/opt/homebrew/bin/tmux", nil),
		Runner:   runner,
	})

	if got.HostName != LocalhostName {
		t.Fatalf("HostName = %q, want %q", got.HostName, LocalhostName)
	}
	if !got.Reachable {
		t.Fatalf("Reachable = false, error = %v", got.Error)
	}
	if len(got.Sessions) != 1 || got.Sessions[0].Name != "api" || !got.Sessions[0].Attached {
		t.Fatalf("Sessions = %#v", got.Sessions)
	}
	wantArgs := []string{"ls", "-F", tmuxinfo.LocalFormat}
	if runner.name != "/opt/homebrew/bin/tmux" || !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("runner saw %q %#v, want tmux %#v", runner.name, runner.args, wantArgs)
	}
	if got.Latency <= 0 {
		t.Fatalf("Latency = %s, want positive duration", got.Latency)
	}
}

func TestLocalhostMissingTmuxIsUnreachable(t *testing.T) {
	got := Localhost(context.Background(), Options{
		LookPath: lookPath("", exec.ErrNotFound),
		Runner:   &recordingRunner{},
	})

	if got.Reachable {
		t.Fatal("Reachable = true, want false")
	}
	if got.Error == nil || !strings.Contains(got.Error.Error(), "tmux not found") {
		t.Fatalf("Error = %v, want tmux not found diagnostic", got.Error)
	}
}

func TestLocalhostNoServerOutputIsReachableWithZeroSessions(t *testing.T) {
	got := Localhost(context.Background(), Options{
		LookPath: lookPath("/usr/bin/tmux", nil),
		Runner: &recordingRunner{result: CommandResult{
			Stderr:   "no server running on /tmp/tmux-501/default\n",
			ExitCode: 1,
			Err:      errors.New("exit status 1"),
		}},
	})

	if !got.Reachable {
		t.Fatalf("Reachable = false, error = %v", got.Error)
	}
	if len(got.Sessions) != 0 {
		t.Fatalf("len(Sessions) = %d, want 0", len(got.Sessions))
	}
}

func TestLocalhostUsesPerHostTimeout(t *testing.T) {
	runner := RunnerFunc(func(ctx context.Context, name string, args ...string) CommandResult {
		<-ctx.Done()
		return CommandResult{Err: ctx.Err()}
	})

	got := Localhost(context.Background(), Options{
		LookPath: lookPath("/usr/bin/tmux", nil),
		Runner:   runner,
		Timeout:  time.Nanosecond,
	})

	if got.Reachable {
		t.Fatal("Reachable = true, want false")
	}
	if !errors.Is(got.Error, context.DeadlineExceeded) {
		t.Fatalf("Error = %v, want context deadline exceeded", got.Error)
	}
}

func TestLocalhostMalformedOutputIsUnreachable(t *testing.T) {
	got := Localhost(context.Background(), Options{
		LookPath: lookPath("/usr/bin/tmux", nil),
		Runner: &recordingRunner{result: CommandResult{
			Stdout:   "api|not-a-number|1|1779799500\n",
			ExitCode: 0,
		}},
	})

	if got.Reachable {
		t.Fatal("Reachable = true, want false")
	}
	if got.Error == nil || !strings.Contains(got.Error.Error(), "invalid window count") {
		t.Fatalf("Error = %v, want parse diagnostic", got.Error)
	}
}

func TestRemoteReturnsSessionsWhenSSHSucceeds(t *testing.T) {
	host := config.Host{Name: "prod", Hostname: "prod.example.com", User: "deploy", Port: 2222}
	runner := &recordingRunner{result: CommandResult{
		Stdout:   "api|3|1\n",
		ExitCode: 0,
	}}

	got := Remote(context.Background(), host, Options{Runner: runner})

	if got.HostName != "prod" {
		t.Fatalf("HostName = %q, want prod", got.HostName)
	}
	if !got.Reachable {
		t.Fatalf("Reachable = false, error = %v", got.Error)
	}
	if len(got.Sessions) != 1 || got.Sessions[0].Name != "api" || !got.Sessions[0].Created.IsZero() {
		t.Fatalf("Sessions = %#v", got.Sessions)
	}
	wantArgs := []string{
		"-o", "ConnectTimeout=2",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-p", "2222",
		"deploy@prod.example.com",
		"tmux", "ls", "-F", tmuxinfo.RemoteFormat,
	}
	if runner.name != "ssh" || !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("runner saw %q %#v, want ssh %#v", runner.name, runner.args, wantArgs)
	}
}

func TestRemoteNoOutputIsReachableWithZeroSessions(t *testing.T) {
	host := config.Host{Name: "prod", Hostname: "prod.example.com"}

	got := Remote(context.Background(), host, Options{
		Runner: &recordingRunner{result: CommandResult{ExitCode: 0}},
	})

	if !got.Reachable {
		t.Fatalf("Reachable = false, error = %v", got.Error)
	}
	if len(got.Sessions) != 0 {
		t.Fatalf("len(Sessions) = %d, want 0", len(got.Sessions))
	}
}

func TestRemoteNoServerOutputIsReachableWithZeroSessions(t *testing.T) {
	host := config.Host{Name: "prod", Hostname: "prod.example.com"}

	got := Remote(context.Background(), host, Options{
		Runner: &recordingRunner{result: CommandResult{
			Stderr:   "no server running on /tmp/tmux-501/default\n",
			ExitCode: 1,
			Err:      errors.New("exit status 1"),
		}},
	})

	if !got.Reachable {
		t.Fatalf("Reachable = false, error = %v", got.Error)
	}
	if len(got.Sessions) != 0 {
		t.Fatalf("len(Sessions) = %d, want 0", len(got.Sessions))
	}
}

func TestRemoteNoServerOnStdoutIsReachableWithZeroSessions(t *testing.T) {
	host := config.Host{Name: "prod", Hostname: "prod.example.com"}

	got := Remote(context.Background(), host, Options{
		Runner: &recordingRunner{result: CommandResult{
			Stdout:   "no server running on /tmp/tmux-501/default\n",
			ExitCode: 0,
		}},
	})

	if !got.Reachable {
		t.Fatalf("Reachable = false, error = %v", got.Error)
	}
	if len(got.Sessions) != 0 {
		t.Fatalf("len(Sessions) = %d, want 0", len(got.Sessions))
	}
}

func TestRemoteNonZeroExitIsUnreachable(t *testing.T) {
	host := config.Host{Name: "prod", Hostname: "prod.example.com"}

	got := Remote(context.Background(), host, Options{
		Runner: &recordingRunner{result: CommandResult{
			Stderr:   "ssh: connect to host prod.example.com port 22: Operation timed out\n",
			ExitCode: 255,
			Err:      errors.New("exit status 255"),
		}},
	})

	if got.Reachable {
		t.Fatal("Reachable = true, want false")
	}
	if got.Error == nil || !strings.Contains(got.Error.Error(), "ssh: connect to host prod.example.com port 22") {
		t.Fatalf("Error = %v, want ssh diagnostic", got.Error)
	}
	if len(got.Sessions) != 0 {
		t.Fatalf("len(Sessions) = %d, want 0", len(got.Sessions))
	}
}

func TestRemoteRunnerErrorWithoutExitCodeIsUnreachable(t *testing.T) {
	host := config.Host{Name: "prod", Hostname: "prod.example.com"}

	got := Remote(context.Background(), host, Options{
		Runner: &recordingRunner{result: CommandResult{
			Err: errors.New("exec: ssh: executable file not found in PATH"),
		}},
	})

	if got.Reachable {
		t.Fatal("Reachable = true, want false")
	}
	if got.Error == nil || !strings.Contains(got.Error.Error(), "executable file not found") {
		t.Fatalf("Error = %v, want runner diagnostic", got.Error)
	}
}

func TestRemoteUsesPerHostTimeout(t *testing.T) {
	host := config.Host{Name: "prod", Hostname: "prod.example.com"}
	runner := RunnerFunc(func(ctx context.Context, name string, args ...string) CommandResult {
		<-ctx.Done()
		return CommandResult{Err: ctx.Err()}
	})

	got := Remote(context.Background(), host, Options{
		Runner:  runner,
		Timeout: time.Nanosecond,
	})

	if got.Reachable {
		t.Fatal("Reachable = true, want false")
	}
	if !errors.Is(got.Error, context.DeadlineExceeded) {
		t.Fatalf("Error = %v, want context deadline exceeded", got.Error)
	}
}

func TestStartEmitsLocalhostAndConfiguredHostsThenCloses(t *testing.T) {
	hosts := []config.Host{
		{Name: "prod", Hostname: "prod.example.com"},
		{Name: "dev", Hostname: "dev.example.com"},
	}
	runner := RunnerFunc(func(ctx context.Context, name string, args ...string) CommandResult {
		return CommandResult{ExitCode: 0}
	})

	results := Start(context.Background(), hosts, Options{
		LookPath: lookPath("/usr/bin/tmux", nil),
		Runner:   runner,
	})

	got := map[string]ProbeResult{}
	for result := range results {
		got[result.HostName] = result
	}

	for _, name := range []string{LocalhostName, "prod", "dev"} {
		result, ok := got[name]
		if !ok {
			t.Fatalf("missing result for %s in %#v", name, got)
		}
		if !result.Reachable {
			t.Fatalf("%s Reachable = false, error = %v", name, result.Error)
		}
	}
}

func TestStartRunsRemoteProbesConcurrently(t *testing.T) {
	hosts := []config.Host{
		{Name: "prod", Hostname: "prod.example.com"},
		{Name: "dev", Hostname: "dev.example.com"},
	}
	started := make(chan struct{}, len(hosts))
	release := make(chan struct{})
	var closeRelease sync.Once
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(func() {
		closeRelease.Do(func() {
			close(release)
		})
	})

	runner := RunnerFunc(func(ctx context.Context, name string, args ...string) CommandResult {
		if name != "ssh" {
			return CommandResult{ExitCode: 0}
		}

		started <- struct{}{}
		select {
		case <-release:
			return CommandResult{ExitCode: 0}
		case <-ctx.Done():
			return CommandResult{Err: ctx.Err()}
		}
	})

	results := Start(ctx, hosts, Options{
		LookPath: lookPath("/usr/bin/tmux", nil),
		Runner:   runner,
	})

	for i := 0; i < len(hosts); i++ {
		select {
		case <-started:
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("only %d remote probes started, want %d", i, len(hosts))
		}
	}

	closeRelease.Do(func() {
		close(release)
	})

	got := 0
	for range results {
		got++
	}
	if got != len(hosts)+1 {
		t.Fatalf("results emitted = %d, want %d", got, len(hosts)+1)
	}
}

func TestExecRunnerCapturesSuccessfulCommand(t *testing.T) {
	result := execRunner{}.Run(context.Background(), os.Args[0], helperArgs("success")...)

	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "probe stdout\n" {
		t.Fatalf("Stdout = %q, want helper stdout", result.Stdout)
	}
	if result.Stderr != "probe stderr\n" {
		t.Fatalf("Stderr = %q, want helper stderr", result.Stderr)
	}
}

func TestExecRunnerCapturesExitCode(t *testing.T) {
	result := execRunner{}.Run(context.Background(), os.Args[0], helperArgs("exit-7")...)

	if result.Err == nil {
		t.Fatal("Run returned nil error")
	}
	if result.ExitCode != 7 {
		t.Fatalf("ExitCode = %d, want 7", result.ExitCode)
	}
	if result.Stderr != "probe failed\n" {
		t.Fatalf("Stderr = %q, want helper stderr", result.Stderr)
	}
}

func TestExecRunnerReportsContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()

	result := execRunner{}.Run(ctx, os.Args[0], helperArgs("sleep")...)

	if !errors.Is(result.Err, context.DeadlineExceeded) {
		t.Fatalf("Err = %v, want context deadline exceeded", result.Err)
	}
}

func TestCommandErrorUsesExitStatusWhenNoOutputExists(t *testing.T) {
	err := commandError(CommandResult{ExitCode: 255})

	if err == nil || err.Error() != "command exited with status 255" {
		t.Fatalf("commandError() = %v, want exit status diagnostic", err)
	}
}

func TestElapsedNeverReturnsNonPositiveDuration(t *testing.T) {
	got := elapsed(time.Now().Add(time.Second))

	if got <= 0 {
		t.Fatalf("elapsed() = %s, want positive duration", got)
	}
}

func TestOptionsWithDefaultsFillsDependencies(t *testing.T) {
	got := (Options{}).withDefaults()

	if got.LookPath == nil {
		t.Fatal("LookPath is nil")
	}
	if got.Runner == nil {
		t.Fatal("Runner is nil")
	}
}

type recordingRunner struct {
	name   string
	args   []string
	result CommandResult
}

func (r *recordingRunner) Run(ctx context.Context, name string, args ...string) CommandResult {
	r.name = name
	r.args = append([]string(nil), args...)
	if r.result.Err == nil && ctx.Err() != nil {
		r.result.Err = ctx.Err()
	}
	return r.result
}

func lookPath(path string, err error) func(string) (string, error) {
	return func(file string) (string, error) {
		if file != "tmux" {
			return "", errors.New("unexpected lookpath target: " + file)
		}
		return path, err
	}
}

func helperArgs(action string) []string {
	return []string{"-test.run=TestExecRunnerHelperProcess", "--", action}
}

func TestExecRunnerHelperProcess(t *testing.T) {
	action := ""
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			action = os.Args[i+1]
			break
		}
	}
	if action == "" {
		return
	}

	switch action {
	case "success":
		_, _ = os.Stdout.WriteString("probe stdout\n")
		_, _ = os.Stderr.WriteString("probe stderr\n")
		os.Exit(0)
	case "exit-7":
		_, _ = os.Stderr.WriteString("probe failed\n")
		os.Exit(7)
	case "sleep":
		time.Sleep(time.Second)
		os.Exit(0)
	default:
		t.Fatalf("unknown helper action %q", action)
	}
}

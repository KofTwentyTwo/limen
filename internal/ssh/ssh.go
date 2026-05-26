// Package ssh builds deterministic ssh command vectors.
package ssh

import (
	"strconv"

	"github.com/KofTwentyTwo/limen/internal/config"
)

// Target returns the host target accepted by ssh.
func Target(host config.Host) string {
	if host.User == "" {
		return host.Hostname
	}
	return host.User + "@" + host.Hostname
}

// SessionArgv builds the design-specified ssh argv for an interactive tmux command.
func SessionArgv(host config.Host, tmuxArgs ...string) []string {
	argv := []string{"ssh", "-t"}
	argv = appendPort(argv, host)
	argv = append(argv, Target(host), "tmux")
	argv = append(argv, tmuxArgs...)
	return argv
}

// LocalTmuxArgv builds the local tmux argv for hosts that do not need ssh.
func LocalTmuxArgv(tmuxArgs ...string) []string {
	argv := []string{"tmux"}
	argv = append(argv, tmuxArgs...)
	return argv
}

// ProbeArgv builds the non-interactive remote probe argv.
func ProbeArgv(host config.Host, format string) []string {
	argv := []string{
		"ssh",
		"-o", "ConnectTimeout=2",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
	}
	argv = appendPort(argv, host)
	argv = append(argv, Target(host), "tmux", "ls", "-F", format)
	return argv
}

func appendPort(argv []string, host config.Host) []string {
	if host.Port != 0 && host.Port != 22 {
		argv = append(argv, "-p", strconv.Itoa(host.Port))
	}
	return argv
}

package ssh

import (
	"reflect"
	"testing"

	"github.com/KofTwentyTwo/limen/internal/config"
)

func TestSessionArgvBuildsDesignExamples(t *testing.T) {
	tests := []struct {
		name string
		host config.Host
		tmux []string
		want []string
	}{
		{
			name: "hostname only attach",
			host: config.Host{Name: "prod", Hostname: "prod.example.com"},
			tmux: []string{"attach", "-t", "api"},
			want: []string{"ssh", "-t", "prod.example.com", "tmux", "attach", "-t", "api"},
		},
		{
			name: "user new",
			host: config.Host{Name: "dev", Hostname: "dev.lan", User: "james"},
			tmux: []string{"new"},
			want: []string{"ssh", "-t", "james@dev.lan", "tmux", "new"},
		},
		{
			name: "user and non-default port named new",
			host: config.Host{Name: "bldr", Hostname: "bldr.lan", User: "ci", Port: 2222},
			tmux: []string{"new", "-s", "build"},
			want: []string{"ssh", "-t", "-p", "2222", "ci@bldr.lan", "tmux", "new", "-s", "build"},
		},
		{
			name: "non-default port without user",
			host: config.Host{Name: "bldr", Hostname: "bldr.lan", Port: 2222},
			tmux: []string{"new"},
			want: []string{"ssh", "-t", "-p", "2222", "bldr.lan", "tmux", "new"},
		},
		{
			name: "default port omitted",
			host: config.Host{Name: "prod", Hostname: "prod.example.com", Port: 22},
			tmux: []string{"new"},
			want: []string{"ssh", "-t", "prod.example.com", "tmux", "new"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SessionArgv(tt.host, tt.tmux...); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SessionArgv() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestLocalTmuxArgvBuildsLocalDesignExamples(t *testing.T) {
	tests := []struct {
		name string
		tmux []string
		want []string
	}{
		{
			name: "attach",
			tmux: []string{"attach", "-t", "api"},
			want: []string{"tmux", "attach", "-t", "api"},
		},
		{
			name: "new",
			tmux: []string{"new"},
			want: []string{"tmux", "new"},
		},
		{
			name: "named new",
			tmux: []string{"new", "-s", "build"},
			want: []string{"tmux", "new", "-s", "build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LocalTmuxArgv(tt.tmux...); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("LocalTmuxArgv() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestProbeArgvBuildsRemoteProbeCommand(t *testing.T) {
	host := config.Host{Name: "prod", Hostname: "prod.example.com", User: "deploy", Port: 2222}

	got := ProbeArgv(host, "#{session_name}|#{session_windows}|#{session_attached}")
	want := []string{
		"ssh",
		"-o", "ConnectTimeout=2",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-p", "2222",
		"deploy@prod.example.com",
		"tmux", "ls", "-F", "#{session_name}|#{session_windows}|#{session_attached}",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ProbeArgv() = %#v, want %#v", got, want)
	}
}

func TestTargetBuildsUserAtHostnameWhenUserIsSet(t *testing.T) {
	host := config.Host{Name: "dev", Hostname: "dev.lan", User: "james"}

	if got := Target(host); got != "james@dev.lan" {
		t.Fatalf("Target() = %q, want james@dev.lan", got)
	}
}

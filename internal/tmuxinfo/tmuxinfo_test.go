package tmuxinfo

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseLocalReadsSessionRows(t *testing.T) {
	got, err := ParseLocal("api|3|1|1779799500\nops|1|0|1779799600\n")
	if err != nil {
		t.Fatalf("ParseLocal returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(got))
	}

	api := got[0]
	if api.Name != "api" {
		t.Fatalf("api.Name = %q, want api", api.Name)
	}
	if api.Windows != 3 {
		t.Fatalf("api.Windows = %d, want 3", api.Windows)
	}
	if !api.Attached {
		t.Fatal("api.Attached = false, want true")
	}
	wantCreated := time.Unix(1779799500, 0).UTC()
	if !api.Created.Equal(wantCreated) {
		t.Fatalf("api.Created = %s, want %s", api.Created, wantCreated)
	}

	ops := got[1]
	if ops.Name != "ops" || ops.Windows != 1 || ops.Attached || !ops.Created.Equal(time.Unix(1779799600, 0).UTC()) {
		t.Fatalf("ops = %#v", ops)
	}
}

func TestParseRemoteReadsSessionRowsWithoutCreatedTime(t *testing.T) {
	got, err := ParseRemote("api|3|2\nops|1|0\n")
	if err != nil {
		t.Fatalf("ParseRemote returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(got))
	}

	api := got[0]
	if api.Name != "api" || api.Windows != 3 || !api.Attached || !api.Created.IsZero() {
		t.Fatalf("api = %#v", api)
	}

	ops := got[1]
	if ops.Name != "ops" || ops.Windows != 1 || ops.Attached || !ops.Created.IsZero() {
		t.Fatalf("ops = %#v", ops)
	}
}

func TestParseEmptyOutputReturnsZeroSessions(t *testing.T) {
	for _, parse := range []struct {
		name string
		fn   func(string) ([]SessionInfo, error)
	}{
		{name: "local", fn: ParseLocal},
		{name: "remote", fn: ParseRemote},
	} {
		t.Run(parse.name, func(t *testing.T) {
			got, err := parse.fn("\n\t ")
			if err != nil {
				t.Fatalf("parse returned error: %v", err)
			}
			if len(got) != 0 {
				t.Fatalf("len(sessions) = %d, want 0", len(got))
			}
		})
	}
}

func TestParseRejectsMalformedRowsWithUsefulErrors(t *testing.T) {
	tests := []struct {
		name    string
		parse   func(string) ([]SessionInfo, error)
		output  string
		wantErr string
	}{
		{
			name:    "local missing created field",
			parse:   ParseLocal,
			output:  "api|3|1\n",
			wantErr: "tmuxinfo: malformed local row 1: expected 4 fields, got 3",
		},
		{
			name:    "remote extra field",
			parse:   ParseRemote,
			output:  "api|3|1|1779799500\n",
			wantErr: "tmuxinfo: malformed remote row 1: expected 3 fields, got 4",
		},
		{
			name:    "invalid window count",
			parse:   ParseRemote,
			output:  "api|many|1\n",
			wantErr: `tmuxinfo: remote row 1 has invalid window count "many"`,
		},
		{
			name:    "invalid attached count",
			parse:   ParseRemote,
			output:  "api|3|yes\n",
			wantErr: `tmuxinfo: remote row 1 has invalid attached count "yes"`,
		},
		{
			name:    "invalid created timestamp",
			parse:   ParseLocal,
			output:  "api|3|1|not-a-time\n",
			wantErr: `tmuxinfo: local row 1 has invalid created timestamp "not-a-time"`,
		},
		{
			name:    "negative created timestamp",
			parse:   ParseLocal,
			output:  "api|3|1|-1\n",
			wantErr: `tmuxinfo: local row 1 has invalid created timestamp "-1"`,
		},
		{
			name:    "empty session name",
			parse:   ParseRemote,
			output:  "|3|1\n",
			wantErr: "tmuxinfo: remote row 1 has empty session name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.parse(tt.output)
			if err == nil {
				t.Fatal("parse returned nil error")
			}
			if !strings.HasPrefix(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want prefix %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseRejectsNegativeNumbersWithoutFormattingNoise(t *testing.T) {
	_, err := ParseLocal("api|-1|0|1779799500\n")
	if err == nil {
		t.Fatal("ParseLocal returned nil error")
	}

	want := `tmuxinfo: local row 1 has invalid window count "-1"`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestIsNoServerRunningRecognizesTmuxMessage(t *testing.T) {
	if !IsNoServerRunning("no server running on /tmp/tmux-501/default\n") {
		t.Fatal("IsNoServerRunning returned false for tmux no-server message")
	}
	if IsNoServerRunning("ssh: connect to host prod port 22: Operation timed out") {
		t.Fatal("IsNoServerRunning returned true for unrelated output")
	}
}

func TestParseNoServerOutputReturnsSentinelError(t *testing.T) {
	_, err := ParseRemote("no server running on /tmp/tmux-501/default\n")
	if !errors.Is(err, ErrNoServer) {
		t.Fatalf("ParseRemote error = %v, want ErrNoServer", err)
	}
}

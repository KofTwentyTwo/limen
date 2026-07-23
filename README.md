# limen

> *Latin: "threshold". The doorsill one crosses when entering.*

A two-stage terminal launcher TUI for tmux and SSH. The first thing you see when opening a new terminal: pick a host, pick a session, get to work.

**Status**: Active implementation. The TUI launches host and session pickers. See [`docs/DESIGN.md`](docs/DESIGN.md) for the canonical architecture specification.

---

## What It Does

`limen` replaces tmux's default name-prompt-on-launch with an interactive, keyboard-driven TUI:

1. **Host Picker**: Choose `localhost` or one of your declared SSH targets. Live status indicators show reachability and active session counts.
2. **Session Picker**: Pick an existing tmux session to attach to, or spawn a new named session. Behaves consistently across local and remote targets.

Pressing `Esc` at any stage drops you into a fresh unnamed tmux session at the selected host with zero deliberation tax.

```
  ╭─ limen ─────────────────────────╮   ╭─ DETAILS ──────────────────────────╮
  │                                 │   │                                    │
  │ ● localhost (renova)   3 sess.  │   │  prod                              │
  │ ● prod                 2 sess.  │ ◀ │  ──────────────────────────────────│
  │ ● dev                  1 sess.  │   │  deploy@prod-server-01.example.com │
  │ ○ builder              unreach. │   │                                    │
  │ ● sandbox              —        │   │  Status:        ● online           │
  │                                 │   │  Sessions:      2  (api, ops)      │
  │                                 │   │  Last attached: 2 hours ago        │
  │                                 │   │                                    │
  │                                 │   │  Production application servers.   │
  │ ╰─────────────────────────────────╯   ╰────────────────────────────────────╯

   ↑↓ navigate   ·   enter connect   ·   / search   ·   ?  help   ·   esc skip
```

---

## Tech Stack & Architecture

- **Language & Runtime**: Go 1.22+
- **TUI Engine**: [Charm Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm Architecture) & [Lip Gloss](https://github.com/charmbracelet/lipgloss) styling
- **Orchestration**: `tmux` IPC & native OpenSSH background client stream
- **Configuration**: Single JSON fleet declaration at `~/.config/limen/hosts.json`

---

## Installation

### Homebrew (macOS / Linux)
```bash
brew install KofTwentyTwo/tap/limen
```

### Nix Flake Run
```bash
nix run github:KofTwentyTwo/limen
```

### Binary Releases
Pre-built binaries are available on the [Releases](https://github.com/KofTwentyTwo/limen/releases) page:

| Platform | Architecture | Binary Name |
| :--- | :--- | :--- |
| **Linux** | x86_64 | `limen-linux-amd64` |
| **Linux** | ARM64 | `limen-linux-arm64` |
| **macOS** | Intel | `limen-macos-amd64` |
| **macOS** | Apple Silicon | `limen-macos-arm64` |

---

## Local Build & Verification

### Build from Source
```bash
# Build binary
go build -ldflags="-s -w" -o bin/limen ./cmd/limen

# Or build via Nix
nix build
```

### Quality Gates & Verification
```bash
# 1. Run Unit & Integration Tests
go test -v ./...

# 2. Run Go Linter & Vet
go vet ./...
```

---

## Fleet Configuration

Declare target hosts in `~/.config/limen/hosts.json`:

```json
{
  "hosts": [
    { "name": "prod", "hostname": "prod-server-01.example.com", "user": "deploy" },
    { "name": "dev",  "hostname": "dev-box.lan", "user": "james" },
    { "name": "builder", "hostname": "builder.lan" }
  ]
}
```

`localhost` is implicit and managed automatically by `limen`. See [`docs/DESIGN.md`](docs/DESIGN.md) for full schema specifications.

---

## License

[GPL-3.0](LICENSE).

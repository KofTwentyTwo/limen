# limen

> *Latin: "threshold". The doorsill one crosses when entering.*

A terminal launcher TUI for tmux and SSH. The first thing you see when you open a new terminal: pick a host, pick a session, get to work.

**Status:** Early implementation. The binary can launch the host and session picker; see [`docs/DESIGN.md`](docs/DESIGN.md) for the canonical spec.

---

## What it does

Replaces tmux's name-prompt-on-launch with a two-stage TUI:

1. **Host picker** — choose `localhost` or one of your declared SSH targets. Live status dots show which hosts are reachable and how many sessions each has.
2. **Session picker** — pick an existing tmux session to attach to, or create a new one with a name. Works the same whether the host is local or remote.

Press `Esc` at any point and `limen` drops you into a fresh unnamed tmux session at the appropriate host. No deliberation tax.

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
  ╰─────────────────────────────────╯   ╰────────────────────────────────────╯

   ↑↓ navigate   ·   enter connect   ·   / search   ·   ?  help   ·   esc skip
```

---

## Install

Install with Nix:

```bash
nix run github:KofTwentyTwo/limen
```

Install with Homebrew after the first tagged release:

```bash
brew install KofTwentyTwo/tap/limen
```

Prebuilt binaries are published to the GitHub Releases page for:

| Platform | Architecture | Binary |
|----------|--------------|--------|
| Linux | x64 | `limen-linux-amd64` |
| Linux | ARM64 | `limen-linux-arm64` |
| macOS | Intel | `limen-macos-amd64` |
| macOS | Apple Silicon | `limen-macos-arm64` |

## Configuration

A single JSON file at `~/.config/limen/hosts.json` declares your fleet. Example:

```json
{
  "hosts": [
    { "name": "prod", "hostname": "prod-server-01.example.com", "user": "deploy" },
    { "name": "dev",  "hostname": "dev-box.lan", "user": "james" },
    { "name": "builder", "hostname": "builder.lan" }
  ]
}
```

`localhost` is implicit and always available — never declare it in this file.

See [`docs/DESIGN.md` §7.1](docs/DESIGN.md) for the full schema.

---

## Design

The full design and requirements document lives at [`docs/DESIGN.md`](docs/DESIGN.md). That document is the source of truth for what `limen` is and should be; implementation work proceeds from there.

## Status & roadmap

Implementation lands on the `develop` branch. Tagged releases publish prebuilt binaries and update the Homebrew tap.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md).

## License

[GPL-3.0](LICENSE).

## See also

- [tmux](https://github.com/tmux/tmux/wiki) — the multiplexer `limen` orchestrates
- [bubbletea](https://github.com/charmbracelet/bubbletea) — the TUI framework `limen` is built on
- [WezTerm](https://wezfurlong.org/wezterm/) — the terminal `limen` is designed to live inside

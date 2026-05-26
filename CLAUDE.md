# CLAUDE.md

Project memory for AI assistants working in this repo. Read this first.

## What is limen

`limen` is a terminal launcher TUI: pick a host, pick a tmux session, exec into it. Written in Go using bubbletea. See [`docs/DESIGN.md`](docs/DESIGN.md) for the canonical specification — that document is the source of truth for behavior, architecture, and scope.

## Current status

**Early implementation.** The repository contains:

- `docs/DESIGN.md` — full design and requirements document.
- `internal/` packages for config, state, probing, SSH, tmux parsing, exec planning, formatting, and UI.
- `main.go`, `go.mod`, and `flake.nix` for the working binary and Nix package.
- `.github/` — CI and release workflows, issue templates, PR template.

The next implementation milestones are to:

1. Harden the terminal UI with real-world tmux and SSH smoke testing.
2. Keep release automation green for tagged releases and downstream package managers.
3. Continue from `docs/DESIGN.md`; if behavior changes, update the design doc in the same PR.

## Key conventions

| Topic              | Convention                                                                |
| ------------------ | ------------------------------------------------------------------------- |
| Language           | Go 1.24+, no CGO                                                          |
| TUI framework      | `bubbletea` + `lipgloss`                                                  |
| Default branch     | `develop` (PRs target this); `main` reserved for tagged releases          |
| Commit style       | Conventional Commits (`feat:`, `fix:`, `docs:` …) — see `CONTRIBUTING.md` |
| License            | GPL-3.0                                                                   |
| Build              | Nix flake (`packages.default`) + plain `go build` for dev                 |
| Release            | GitHub Actions tag workflow, direct Go cross-builds, GitHub Releases, Homebrew tap |
| Test framework     | Standard `testing` package; `teatest` for TUI integration                 |
| Go modules         | One `internal/` per concern (config, state, probe, ssh, ui, exec, format) |

## Boundaries to respect

- **Don't drift from the design doc.** If a proposed change conflicts with `docs/DESIGN.md`, update the doc in the same PR (or in a preceding PR) — never let code and design diverge silently.
- **`internal/` packages don't import bubbletea.** Only `internal/ui/*` does. Keep the logic layer TUI-agnostic.
- **No shelling out for things Go does natively.** Exceptions: invoking `ssh` and `tmux` for the actual session work (that's the whole point).
- **`syscall.Exec`, not `os/exec`, for the final hand-off.** The user expects `limen` to disappear, not fork a child.
- **No mouse handling, no themes (yet), no auto-discovery of SSH hosts.** These are explicit non-goals — see §4 of `DESIGN.md`.

## Distribution

Targets in order of priority:

1. **Nix flake** (`nix run github:KofTwentyTwo/limen`) — primary distribution channel.
2. **Homebrew tap** at `KofTwentyTwo/homebrew-tap` — `brew install KofTwentyTwo/tap/limen`.
3. **GitHub Releases** with direct Go-built binaries for macOS/Linux × amd64/arm64.

Eventual goal: submit to `homebrew-core` once the tool stabilizes.

## When to ask the user

- License changes (currently GPL-3.0 — anything else is a deliberate decision, not a default).
- Scope additions beyond what `DESIGN.md` specifies (e.g., adding mouse support, plugins, themes).
- Renaming binaries, packages, or the project itself.
- Anything that changes the public-facing CLI surface in a breaking way.

## Helpful pointers

- Latin name etymology: `limen` = threshold / doorsill. Symbolic of the boundary the user crosses on entering work each day.
- Sister tools in the KofTwentyTwo org with similar Latin naming: `Munitor` (fortifier), `Concilium` (council), `Aestimare` (to estimate), `Claritas` (clarity), `ArmariumMinutum` (small cabinet).
- The user's nix-darwin config repo at `github.com/KofTwentyTwo/nix` is where `limen` integrates downstream (via a `home/limen/default.nix` module that pins this repo as a flake input). Changes here that affect that integration should be reflected on that side.

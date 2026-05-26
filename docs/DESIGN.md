# limen — Design & Requirements

**Status:** Draft v1
**Date:** 2026-05-26
**Author:** James Maes
**Intended audience:** an LLM or human implementing this tool from scratch in a fresh repository, with no other context.

---

## 1. Overview

`limen` is a terminal launcher TUI. It is the first thing the user sees when opening a new terminal window. It asks two questions in sequence — *which host?* then *which tmux session?* — and then `exec`s the user into the corresponding `tmux attach` or `ssh ... tmux attach` command, replacing its own process. After that, `limen` is gone; the user is in their session.

The name comes from the Latin word for *threshold* — the doorsill one crosses when entering a building. `limen` is that doorsill for the user's daily work.

A single static Go binary, distributed primarily via a Nix flake and (eventually) a Homebrew tap. Configuration is a single JSON file declaring the curated list of SSH targets. No interactive setup, no auto-discovery, no daemons.

### 1.1 What it replaces

Today, opening a new terminal window runs tmux, which prompts the user for a session name before doing anything else. That prompt is the *only* signal of intent, has no host context, and forces the user to invent a name even when they want to attach to an existing session. `limen` replaces that prompt with an opinionated two-stage TUI that handles both host selection and session selection in one cohesive experience.

---

## 2. Problem Statement

The user operates across several machines (a primary workstation plus a small fleet of always-on SSH targets) and uses tmux for session persistence on each of them. The current friction points:

1. **Every new shell forces a naming decision before showing existing sessions.** Most of the time the user wants to attach to existing work, not start fresh.
2. **There is no unified entry point for "which machine?"** The user opens a terminal, manually `ssh`s, then runs `tmux ls` to see what's there, then `tmux attach`. Three commands, three context switches.
3. **The fleet's state is invisible.** "Is `prod-server` reachable right now? Did I leave a session running on `dev-server` last week?" requires manual probing.

`limen` answers all three: it presents the fleet at a glance, lets the user pick a host with knowledge of its status, then lets them pick a session (or create one) on that host — and then gets out of the way.

---

## 3. Goals

### 3.1 Functional goals

- **Two-stage flow:** stage 1 picks a host (localhost + configured remotes); stage 2 picks a tmux session on that host.
- **Single binary.** No runtime dependencies on a particular shell, language runtime, or package manager.
- **Fast launch.** Visible UI within ~200ms of process start. Host status probing runs in parallel and populates the UI as results arrive.
- **Process replacement.** When the user makes their final selection, `limen` `exec`s the tmux/ssh command and disappears. The user does not see a "limen exiting" message and there is no shell layer between `limen` and tmux.
- **Escape semantics.** Pressing Escape at any stage means *"I'm done deliberating, take me to the sensible default for where I am"* — never back-navigation.
- **Declarative config.** Hosts are read from a single JSON file. No mutation of config from inside `limen`. Editing happens out-of-band.
- **State persistence.** Track last-attached timestamps per host so the UI can show "Last attached: 2h ago" in the details pane.

### 3.2 Experience goals

- **The first screen must feel polished.** This is the moment the user sees dozens of times a day. It must convey identity, not feel like a script.
- **Split-pane host picker** with the list on the left and a live details pane on the right showing connection info, status, last-attached time, and session inventory.
- **Color and typography matter.** A consistent theme runs through both stages. Lipgloss styles, borders, and accent colors are chosen deliberately.
- **Keyboard-only.** All actions are reachable with arrow keys, Enter, Escape, `/` (search), and `?` (help). No mouse handling.

### 3.3 Engineering goals

- **Pure Go.** No CGO. No shelling out for things Go can do natively (with the unavoidable exception of `ssh` and `tmux` for the actual session work).
- **Testable core.** Config parsing, state persistence, ssh argument construction, and time-formatting logic are unit-tested. The TUI layer is exercised via `teatest` for at least the happy paths.
- **Reproducible builds via Nix.** The flake's `packages.default` produces an identical binary on any compatible system.
- **Cross-platform.** macOS and Linux are first-class. Windows is out of scope.

---

## 4. Non-Goals

These are explicitly outside the scope of v1. Listing them prevents scope creep.

- **Not a tmux replacement.** `limen` does not multiplex, manage windows, or alter tmux behavior. It only attaches/creates.
- **Not an SSH manager.** No key management, no jump hosts (`ProxyJump`), no port forwarding configuration. `limen` reads enough host info to construct a basic `ssh user@hostname -p port` invocation; advanced SSH features should live in the user's `~/.ssh/config` and be referenced by hostname alias.
- **Not an SSH auto-discoverer.** Hosts must be declared in the config file. Reading `~/.ssh/config` to auto-populate is a deliberate non-feature: the curated list is the point.
- **Not interactive after exec.** No "return to limen when session ends" loop. Detaching drops the user to a normal shell. Opening a new terminal starts a fresh `limen`.
- **Not a daemon.** `limen` runs, picks, and exits (via exec). No background process, no IPC, no caching to disk between launches (except the small `state.json`).
- **No mouse support.** Keyboard only.
- **No theme customization in v1.** A single sensible theme ships baked in. A `--theme` flag or config field may come later.
- **No first-run wizard.** If `hosts.json` is missing, `limen` runs with only localhost available and prints a hint pointing at the expected config path.

---

## 5. User Experience

### 5.1 Top-level flow

```
   ┌─────────────────┐         ┌────────────────────┐         ┌──────────────┐
   │                 │         │                    │         │              │
   │  Terminal opens │  ───▶   │  limen: pick host  │  ───▶   │  pick session│
   │                 │         │                    │         │              │
   └─────────────────┘         └──────────┬─────────┘         └──────┬───────┘
                                          │                          │
                                       (Escape)                   (Escape)
                                          │                          │
                                          ▼                          ▼
                                ┌────────────────────┐     ┌────────────────────┐
                                │ exec tmux new      │     │ exec tmux new      │
                                │ (localhost,        │     │ (current host,     │
                                │  unnamed)          │     │  unnamed)          │
                                └────────────────────┘     └────────────────────┘
```

### 5.2 Stage 1 — Host picker

The first screen the user sees. Split-pane layout. Host list on the left, details pane on the right that updates as the highlighted entry changes.

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
  │                                 │   │  Deploys via fastlane.             │
  ╰─────────────────────────────────╯   ╰────────────────────────────────────╯

   ↑↓ navigate   ·   enter connect   ·   / search   ·   ?  help   ·   esc skip
```

**Layout requirements:**

- Left pane width: ~40% of terminal width, minimum 30 columns.
- Right pane: remainder.
- The header line above the boxes says `limen` (left) and `DETAILS` (right), styled with the accent color.
- Status dots: filled green circle (`●`) for online, hollow circle (`○`) for unreachable, dash (`—`) for "no sessions yet" indication when applicable.
- Session count column right-aligned.
- localhost is always the first entry. Its display name is `localhost (<short-hostname>)` where short-hostname comes from `hostname -s`. This makes it immediately clear which physical machine the user is sitting at.
- Hosts are listed in the order declared in the config file. No automatic reordering. (Sorting by "most recently attached" is a future feature, called out as such.)
- A footer line shows the keyboard cheat sheet, dimmed.

**Details pane content (for the highlighted host):**

- **Title line:** the host name (the same display name as in the list).
- **Connection line:** `<user>@<hostname>[:port]`. If user is unset, omit the `user@`. If port is unset or 22, omit `:port`.
- **Status:** `● online` / `○ unreachable` / `… probing` while the probe is in flight.
- **Sessions:** count and comma-separated names. If unreachable, show `n/a`. If zero, show `—`.
- **Last attached:** human-readable relative time. Examples: `just now`, `5 minutes ago`, `2 hours ago`, `yesterday`, `3 days ago`, `2 weeks ago`. If never attached, show `never`.
- **Description:** the free-form description from the config, rendered on one or more wrapped lines. Optional.

**Keybindings:**

| Key            | Action                                                       |
| -------------- | ------------------------------------------------------------ |
| `↑` / `k`      | Move selection up                                            |
| `↓` / `j`      | Move selection down                                          |
| `Home` / `g`   | Jump to first entry                                          |
| `End` / `G`    | Jump to last entry                                           |
| `Enter`        | Accept; advance to stage 2 with this host as target          |
| `/`            | Open search/filter mode (filter the list as you type)        |
| `Esc`          | If in search mode, cancel filter. Otherwise, escape semantics (see §5.5) |
| `?`            | Toggle help overlay                                          |
| `Ctrl+C` / `q` | Quit `limen` entirely without exec'ing anything. Exit code 130 / 0 respectively |

### 5.3 Stage 2 — Session picker

Same split-pane layout as stage 1, applied to sessions on the chosen host. The user has already committed to a host; this screen helps them pick or create a session there.

```
  ╭─ prod ──────────────────────────╮   ╭─ DETAILS ──────────────────────────╮
  │                                 │   │                                    │
  │ + New session                   │   │  api                               │
  │ ─────────────────────────────── │   │  ──────────────────────────────────│
  │ ● api          3 windows ★      │ ◀ │  Windows:    3                     │
  │   ops          1 window         │   │  Window 1:   editor                │
  │   scratch      2 windows        │   │  Window 2:   server                │
  │                                 │   │  Window 3:   logs                  │
  │                                 │   │                                    │
  │                                 │   │  Created:    Mon May 19 09:14      │
  │                                 │   │  Attached:   ★ currently attached  │
  ╰─────────────────────────────────╯   ╰────────────────────────────────────╯

   ↑↓ navigate   ·   enter connect   ·   / search   ·   esc new unnamed
```

**Layout requirements:**

- The left-pane header shows the host name (e.g. `prod`), not the word "limen", so the user knows where they are.
- `+ New session` is always the first entry, followed by a horizontal separator (`─` rule), followed by existing sessions in tmux's default ordering (most recently created first, matching `tmux ls`).
- `●` marks the currently-attached session if any (i.e., the one a tmux client is currently inside). On localhost, there can be one. On remotes accessed via `ssh ... tmux ls`, attached-state comes from `#{session_attached}`.
- `★` next to the window count marks the currently-attached session (redundant with the dot for users who scan the right column).

**Picking `+ New session`:**

The right pane changes to a `gum`-style prompt for the new session's name:

```
  ╭─ + New session ─────────────────╮
  │                                 │
  │  Name for the new session:      │
  │                                 │
  │  > _____________________        │
  │                                 │
  │  Enter to create. Esc creates   │
  │  an unnamed session.            │
  │                                 │
  ╰─────────────────────────────────╯
```

- Pressing Enter with a non-empty name execs `tmux new -s <name>` (or `ssh -t host tmux new -s <name>` for remote).
- Pressing Enter with an empty name does the same as Esc.
- Esc execs `tmux new` (no `-s`) → tmux auto-numbers the session.

**Picking an existing session:**

Enter execs `tmux attach -t <name>` (or `ssh -t host tmux attach -t <name>`).

**Keybindings:**

Same as stage 1, with the addition that when the highlighted entry is `+ New session`, the details pane shows the name prompt instead of session details.

### 5.4 Search / filter mode

In either stage, `/` opens an inline filter. The list narrows to entries matching the typed substring (case-insensitive, fuzzy match optional but not required for v1 — simple substring is fine). Esc cancels filter. Enter selects the top match.

While filter is active, the cheat-sheet footer changes to indicate `esc cancel filter`.

### 5.5 Escape semantics — the precise rules

Escape is the user's "skip this decision and take me to the default" shortcut. It never goes backward. The default depends on which stage the user is in.

| Where escape is pressed | Action `limen` takes                                       |
| ----------------------- | ---------------------------------------------------------- |
| Stage 1 (host picker)   | `exec tmux new` on localhost. New unnamed local session.   |
| Stage 2 (session picker), host is localhost | `exec tmux new` on localhost. New unnamed local session.   |
| Stage 2 (session picker), host is remote    | `exec ssh -t <args> tmux new` on that host. New unnamed remote session. |
| `+ New session` name prompt | Same as the stage-2 escape for that host. Unnamed session, current host. |
| Search/filter active     | Cancels the filter; user remains in the stage. Does NOT exit `limen`. A second Esc then triggers the rules above. |

`Ctrl+C` and `q` are separate from Esc — they quit `limen` entirely without exec'ing anything, returning the user to a plain shell prompt. Exit code 130 (`Ctrl+C`) or 0 (`q`).

### 5.6 Help overlay

Pressing `?` toggles a modal overlay showing all keybindings for the current stage. Press `?` again or Esc to dismiss. The overlay floats on top of the existing UI (drawn with lipgloss `Place`).

---

## 6. Architecture

### 6.1 Process model

`limen` is a short-lived foreground process:

1. Parse command-line flags (if any).
2. Read `hosts.json` config.
3. Read `state.json` (last-attached timestamps).
4. Spawn background goroutines to probe each remote host's tmux status (parallel, with timeout). Also probe localhost via local `tmux ls`. Updates flow back to the UI via a channel.
5. Render bubbletea UI (stage 1).
6. User picks a host → transition to stage 2 (without exiting bubbletea; stage 2 is a new model state).
7. On final selection (a session pick or escape), shut down bubbletea cleanly, persist updated `state.json`, then `syscall.Exec` the chosen tmux/ssh command.
8. `syscall.Exec` replaces the `limen` process image. The user is now in tmux/ssh. `limen` is gone.

The exec step is critical:

- **Use `syscall.Exec` (Unix `execve`), not `os/exec`.** `os/exec` would fork a child and keep `limen` running as the parent. We want the actual process replacement so there is no orphaned `limen` and tmux owns the terminal directly.

```go
// pseudocode
argv := []string{"/usr/local/bin/tmux", "attach", "-t", "api"}
env := os.Environ()
syscall.Exec(argv[0], argv, env)
// unreachable
```

### 6.2 Stage transitions

Stages are bubbletea sub-models:

```
   ┌─────────────────────┐    pickHost     ┌──────────────────────┐
   │   HostPickerModel   │ ─────────────▶  │  SessionPickerModel  │
   │                     │                 │                      │
   │   (stage 1)         │ ◀────────────── │   (stage 2)          │
   └─────────────────────┘   back* (not    └──────────────────────┘
                              implemented
                              in v1)
```

The root model is a discriminated union (`type RootModel struct { stage Stage; ... }`) that delegates `Update` and `View` to the active sub-model. On final selection from either sub-model, the root model captures the exec arguments and shuts down bubbletea via `tea.Quit`, then main() performs the exec.

**No back-navigation in v1.** If the user wants to revise a host choice after entering stage 2, they Ctrl+C and re-run `limen`. This keeps escape semantics clean.

### 6.3 Component breakdown

```
internal/
├── config/         # Parse hosts.json. Validate. Expose Host structs.
├── state/          # Load/save state.json. Time-since helpers.
├── probe/          # Concurrent host status probing.
├── tmuxinfo/       # Parse `tmux ls -F ...` output. Local + remote variants.
├── ssh/            # Build ssh argument vectors from Host config.
├── exec/           # syscall.Exec wrapper. Builds final argv.
├── format/         # Relative-time strings ("2h ago"), session-info strings.
├── ui/
│   ├── theme/      # Lipgloss styles, colors, borders.
│   ├── hostpicker/ # Bubbletea model for stage 1.
│   ├── sessionpicker/ # Bubbletea model for stage 2.
│   ├── components/ # Reused: list, detail pane, help overlay, prompt.
│   └── root.go     # Root model dispatching to sub-models.
└── version/        # Version stamp injected at build time.
main.go             # Wires everything together.
```

Each `internal/*` package has a focused responsibility and a small API. The UI packages are the only ones that import bubbletea. Logic packages (config, state, probe, ssh, exec, format) are TUI-agnostic and unit-testable.

---

## 7. Data Model

### 7.1 Configuration: `hosts.json`

**Location:** `$XDG_CONFIG_HOME/limen/hosts.json`, falling back to `$HOME/.config/limen/hosts.json`.

**Schema:**

```json
{
  "hosts": [
    {
      "name": "prod",
      "hostname": "prod-server-01.example.com",
      "user": "deploy",
      "port": 22,
      "description": "Production application servers. Deploys via fastlane."
    },
    {
      "name": "dev",
      "hostname": "dev-box.lan",
      "user": "james",
      "description": "Long-running dev box. Always has a tmux session running."
    },
    {
      "name": "builder",
      "hostname": "builder.lan"
    }
  ]
}
```

**Fields:**

| Field         | Required | Type   | Notes                                                            |
| ------------- | -------- | ------ | ---------------------------------------------------------------- |
| `name`        | yes      | string | Display name shown in the list. Must be unique. Lowercase preferred but not enforced. |
| `hostname`    | yes      | string | The actual SSH target — may be a DNS name, IP, or an alias in `~/.ssh/config`. |
| `user`        | no       | string | If unset, ssh inherits whatever the local user is or whatever `~/.ssh/config` specifies. |
| `port`        | no       | int    | If unset or 22, omitted from the ssh command.                    |
| `description` | no       | string | Free text shown in the details pane. Markdown not supported.     |

**Validation rules:**

- `hosts` must be an array (may be empty — `limen` still works with just localhost).
- Each entry must have non-empty `name` and `hostname`.
- Duplicate `name` values cause a fatal startup error with a clear message: `limen: duplicate host name "<name>" in hosts.json`.
- The reserved name `localhost` must not appear in the config; if it does, fatal error: `limen: "localhost" is reserved and cannot be declared in hosts.json`.

**Behavior when config is missing:** Print a hint to stderr (`limen: no hosts.json found at <path>; only localhost will be available`) and run with an empty host list. Stage 1 then shows just localhost.

**Behavior when config is malformed:** Fatal error with the parse error message and the file path. Do not attempt recovery.

### 7.2 State: `state.json`

**Location:** `$XDG_DATA_HOME/limen/state.json`, falling back to `$HOME/.local/share/limen/state.json`.

**Schema:**

```json
{
  "lastAttached": {
    "localhost": "2026-05-26T13:45:00Z",
    "prod": "2026-05-26T11:30:00Z",
    "dev": "2026-05-19T09:14:00Z"
  }
}
```

**Behavior:**

- `lastAttached` maps a host name (from config, or the literal string `localhost`) to an RFC 3339 timestamp.
- Entries are written when `limen` execs a tmux/ssh command for that host — i.e., the act of "I'm going there now" is what counts, not whether the user later detaches.
- Entries for hosts no longer in the config are NOT pruned automatically (cheap to keep, useful if the user re-adds a host later).
- On read, the file is treated as advisory: malformed/missing → empty map, no error. Never block startup on state issues.
- Writes are atomic: write to `state.json.tmp` in the same directory, then `os.Rename`.
- Directory is created with `os.MkdirAll(..., 0700)` if missing.
- File mode `0600`.

### 7.3 In-memory probe results

These are not persisted. Each `limen` launch re-probes.

```go
type ProbeResult struct {
    HostName    string         // matches config name, or "localhost"
    Reachable   bool
    Sessions    []SessionInfo  // empty if unreachable or no sessions
    Error       error          // for diagnostics, not displayed to user
    Latency     time.Duration  // how long the probe took
}

type SessionInfo struct {
    Name        string
    Windows     int
    WindowNames []string       // best-effort; may be empty for remote
    Attached    bool
    Created     time.Time      // best-effort; may be zero for remote
}
```

---

## 8. Behavior Specification

### 8.1 Probing hosts

On startup, `limen` spawns one goroutine per non-localhost host plus one for localhost itself.

**Localhost probe:**

```
/usr/local/bin/tmux ls -F '#{session_name}|#{session_windows}|#{session_attached}|#{session_created}'
```

Locate tmux via `exec.LookPath("tmux")`. If not found, mark localhost as unreachable with an explanatory error.

**Remote host probe:**

```
ssh -o ConnectTimeout=2 -o BatchMode=yes -o StrictHostKeyChecking=accept-new \
    [-p <port>] [<user>@]<hostname> \
    tmux ls -F '#{session_name}|#{session_windows}|#{session_attached}'
```

- `BatchMode=yes` ensures ssh doesn't prompt for passwords/passphrases. If keys aren't set up, the probe fails gracefully rather than blocking.
- `ConnectTimeout=2` gives the host 2 seconds to respond to TCP SYN. Past that, mark unreachable.
- Overall per-host probe timeout: 3 seconds. Implement via `context.WithTimeout` and `exec.CommandContext`.
- Exit code 0 with no output: host reachable, no sessions (`Sessions: []`).
- Exit code 1 with output `no server running on /tmp/tmux-...`: reachable, no server, treat as zero sessions.
- Other non-zero exits: mark unreachable.

**Concurrency:**

- Use a `sync.WaitGroup` or `errgroup` to coordinate.
- Results flow back to the UI via a `chan ProbeResult`.
- The UI subscribes to the channel as a `tea.Cmd` and re-renders as each result arrives.
- The UI is fully interactive *before* probes complete. Users can navigate while probes are still in flight — the dots simply update from `…` to `●`/`○`.

**Connection multiplexing:**

`limen` does not configure ssh's `ControlMaster`. Users who want sub-second probes should configure it themselves in `~/.ssh/config`:

```
Host *
  ControlMaster auto
  ControlPath ~/.ssh/cm-%r@%h:%p
  ControlPersist 600
```

The design doc should mention this in the README as a performance tip.

### 8.2 Constructing SSH command vectors

Given a `Host` config entry, build the ssh argv as follows:

```
[ssh, -t]
  + (if Host.Port != 0 && Host.Port != 22) [-p <port>]
  + [<target>]   where target = user@hostname if user is set, else hostname
  + [tmux, <subcommand>, ...]
```

The `-t` flag forces pseudo-tty allocation, required for tmux to run interactively over ssh.

Examples:

| Host config                                                | Subcommand              | Resulting argv                                                                  |
| ---------------------------------------------------------- | ----------------------- | ------------------------------------------------------------------------------- |
| `{name: prod, hostname: prod.example.com}`                 | `attach -t api`         | `["ssh", "-t", "prod.example.com", "tmux", "attach", "-t", "api"]`              |
| `{name: dev, hostname: dev.lan, user: james}`              | `new`                   | `["ssh", "-t", "james@dev.lan", "tmux", "new"]`                                 |
| `{name: bldr, hostname: bldr.lan, user: ci, port: 2222}`   | `new -s build`          | `["ssh", "-t", "-p", "2222", "ci@bldr.lan", "tmux", "new", "-s", "build"]`      |

For localhost, no ssh wrapping:

| Subcommand              | Resulting argv                              |
| ----------------------- | ------------------------------------------- |
| `attach -t api`         | `["tmux", "attach", "-t", "api"]`           |
| `new`                   | `["tmux", "new"]`                           |
| `new -s build`          | `["tmux", "new", "-s", "build"]`            |

### 8.3 Localhost display name

The display name for localhost is dynamic: `localhost (<short-hostname>)` where `<short-hostname>` is the output of:

```go
hostname, err := os.Hostname()
short := strings.SplitN(hostname, ".", 2)[0]
```

So on a machine whose hostname is `renova.local`, localhost displays as `localhost (renova)`. This makes the multi-machine experience legible: at a glance the user sees which physical machine they're sitting at.

### 8.4 New-session name validation

For named sessions, the user types a name into the prompt. tmux session names cannot contain `:` or `.` and cannot be empty. `limen` enforces:

- Strip leading/trailing whitespace.
- Reject empty (treat as "create unnamed").
- Reject names containing `:` or `.` — show inline error `name cannot contain : or .` and let user keep typing.
- Names are case-sensitive (tmux is case-sensitive).
- Length cap: 64 characters (advisory; tmux itself has no hard limit but very long names are unwieldy).
- No collision check with existing sessions — tmux will return an error if the name is taken, which `limen` will surface (see §9.3).

### 8.5 Time formatting

The "Last attached" string in the details pane uses these brackets:

| Time delta            | Format                  |
| --------------------- | ----------------------- |
| < 60 seconds          | `just now`              |
| < 60 minutes          | `<N> minutes ago`       |
| < 24 hours            | `<N> hours ago`         |
| < 48 hours            | `yesterday`             |
| < 7 days              | `<N> days ago`          |
| < 4 weeks             | `<N> weeks ago`         |
| ≥ 4 weeks             | `on <Mon Jan 2>`        |
| timestamp absent      | `never`                 |

Singular vs plural is handled correctly (`1 minute ago`, `2 minutes ago`).

### 8.6 Theme

A single theme ships in v1. Colors are defined as lipgloss style values in `internal/ui/theme/theme.go`. The intent:

- **Accent color:** a single distinctive color used for the header titles, the selected-row background, the help-key labels. Default: a warm cyan (`#5fd7ff`) — but the implementer should choose something that reads well on both dark and light terminals.
- **Status colors:** green (`#5fdf87`) for `● online`, dim gray (`#6c6c6c`) for `○ unreachable`, soft blue (`#87afdf`) for the "probing" ellipsis.
- **Borders:** rounded corners (lipgloss `RoundedBorder`) with the accent color dimmed.
- **Cheat-sheet footer:** dimmed gray, separator dots `·` between actions.
- **Background:** terminal default — do not paint any solid background fills.

Theming is intentionally minimal. A future version may load themes from disk.

### 8.7 Logging

`limen` logs to stderr at level `WARN` by default. A `--verbose` flag bumps to `INFO`, `-vv` to `DEBUG`. Logs include probe timings, ssh argv, exec target. Logs are written *before* the UI starts and *after* it exits — never during, to avoid corrupting the rendered TUI.

A `--log-file <path>` flag redirects logs to a file, useful for debugging without polluting stderr (which the user sees as messy output when the UI exits).

### 8.8 Command-line flags

```
limen [--config <path>] [--state <path>] [--verbose|-v] [--log-file <path>] [--version] [--help]
```

| Flag             | Purpose                                                                |
| ---------------- | ---------------------------------------------------------------------- |
| `--config <p>`   | Override config file path. Default per §7.1.                           |
| `--state <p>`    | Override state file path. Default per §7.2.                            |
| `--verbose, -v`  | Increase log verbosity. Repeatable.                                    |
| `--log-file <p>` | Redirect logs to file.                                                 |
| `--version`      | Print version (injected via ldflags at build time) and exit 0.         |
| `--help, -h`     | Print usage and exit 0.                                                |

No positional arguments are accepted in v1.

---

## 9. Error Handling

The cardinal rule: **`limen` should never strand the user in a broken state.** Every error path either degrades gracefully or surfaces clearly and exits.

### 9.1 Config errors

| Condition                          | Behavior                                                                                    |
| ---------------------------------- | ------------------------------------------------------------------------------------------- |
| `hosts.json` missing               | Warn to stderr, run with empty host list. localhost still works.                            |
| `hosts.json` unreadable (perms)    | Fatal: `limen: cannot read <path>: <reason>`. Exit 1.                                       |
| `hosts.json` malformed JSON        | Fatal: `limen: parse error at <path>:<line>:<col>: <msg>`. Exit 1.                          |
| Duplicate host name                | Fatal: `limen: duplicate host name "<name>" in hosts.json`. Exit 1.                         |
| Reserved name `localhost` declared | Fatal: `limen: "localhost" is reserved`. Exit 1.                                            |
| Missing required field             | Fatal: `limen: host #N missing required field "<name>"`. Exit 1.                            |

### 9.2 Probe errors

These are non-fatal by definition. The host appears in the list with `○ unreachable` status. The details pane shows `Sessions: n/a`. Hitting Enter on an unreachable host *still attempts the connection* — the probe may have failed for a transient reason (e.g. ssh agent not loaded), but the actual exec may succeed. ssh's own error output will reach the user if it fails after the exec.

### 9.3 Exec errors

After bubbletea exits and `limen` calls `syscall.Exec`, the only failure mode is `Exec` itself returning (which means the kernel rejected the call — typically because the binary at `argv[0]` doesn't exist or isn't executable).

Handle as follows:

```go
if err := syscall.Exec(argv[0], argv, env); err != nil {
    fmt.Fprintf(os.Stderr, "limen: exec failed: %v\n", err)
    os.Exit(127)
}
```

The user sees a clear error on stderr and lands at their shell prompt. They can re-run `limen` after fixing the issue (e.g. install tmux, restore ssh keys).

ssh and tmux themselves may exit non-zero *after* the exec — at that point `limen` is gone and those exit codes propagate to the shell. That's the correct behavior. The user sees `ssh` or `tmux`'s own error messages directly.

### 9.4 Terminal too small

Below 80×24, the UI degrades to a single-column layout (no details pane). Below 50×15, `limen` prints `terminal too small for limen UI; falling back to bare tmux new` and execs `tmux new`. This makes split panes (e.g. in nested tmux) at least usable.

---

## 10. Build & Distribution

### 10.1 Repository layout

```
limen/
├── .github/
│   └── workflows/
│       ├── ci.yml         # go test + go vet on push/PR
│       └── release.yml    # tag → goreleaser → binaries + checksums
├── docs/
│   ├── DESIGN.md          # This document (or a slightly edited copy)
│   └── README.md          # User-facing docs
├── internal/              # See §6.3 for breakdown
├── flake.nix              # Nix flake exposing packages.default
├── flake.lock
├── go.mod
├── go.sum
├── main.go
├── .goreleaser.yaml       # Cross-compile darwin + linux, amd64 + arm64
├── LICENSE                # GPL-3.0 (matches KofTwentyTwo OSS standard)
└── README.md              # Repo top-level readme
```

### 10.2 Nix flake

```nix
{
  description = "limen — terminal launcher TUI";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages.default = pkgs.buildGoModule {
          pname = "limen";
          version = "0.1.0";  # bumped per release
          src = ./.;
          vendorHash = "sha256-...";  # filled in by nix build complaining first run
          ldflags = [
            "-s" "-w"
            "-X main.version=0.1.0"
          ];
          meta = {
            description = "Terminal launcher TUI for tmux + ssh";
            homepage = "https://github.com/<owner>/limen";
            license = pkgs.lib.licenses.mit;
            mainProgram = "limen";
          };
        };
        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go pkgs.gopls pkgs.golangci-lint ];
        };
      });
}
```

Consumers in nix-darwin / NixOS configs:

```nix
inputs.limen.url = "github:<owner>/limen";
# ...
home.packages = [ inputs.limen.packages.${system}.default ];
```

### 10.3 Homebrew

Path A — **personal tap (preferred for initial release):**

The user creates `github.com/<owner>/homebrew-tap` with a `Formula/limen.rb` that fetches the GitHub release tarball and `go build`s, OR (better) fetches the pre-built binary published by goreleaser. Users install via `brew install <owner>/tap/limen`.

Path B — **homebrew-core (later):**

Once the tool has some traction, submit a formula to `Homebrew/homebrew-core`. Requires:
- An open-source license (GPL-3.0 in this case; Homebrew accepts most OSI-approved licenses).
- A `homepage` URL.
- A stable, versioned release.
- A test block in the formula that confirms `limen --version` works.

Both paths are downstream of the Go build. The Go binary is the unit of distribution.

### 10.4 Release process

```
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions runs `release.yml`:
1. Checks out at the tag.
2. Runs `goreleaser release --clean`.
3. goreleaser cross-compiles for `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`.
4. Uploads release artifacts (tarballs + `checksums.txt`) to the GitHub release page.
5. (Optional) Updates the homebrew tap formula via a goreleaser hook.

---

## 11. Testing

### 11.1 Unit tests (required)

- `internal/config`: valid configs, malformed JSON, duplicate names, reserved name, missing fields.
- `internal/state`: load missing → empty, load malformed → empty, save round-trips, atomic write.
- `internal/ssh`: argv construction for every combination of user/port/none.
- `internal/format`: every bracket of relative time, plural/singular correctness.
- `internal/tmuxinfo`: parse `tmux ls -F` output formats, including edge cases (zero sessions, "no server running" message).

### 11.2 Integration tests (recommended)

- TUI tests using `github.com/charmbracelet/x/exp/teatest` — drive the bubbletea models with simulated keypresses, assert on view output.
- Specifically: stage 1 → stage 2 → exec target. Don't actually exec; verify the chosen argv is correct.

### 11.3 Manual smoke

- Open with no `hosts.json`: only localhost shows.
- Open with hosts: probe results trickle in.
- Pick localhost, attach to existing session: tmux takes over the terminal.
- Pick remote, attach to existing remote session: ssh+tmux takes over.
- Pick + New session on local: prompted for name, tmux new -s runs.
- Pick + New session on remote: same, but via ssh -t.
- Escape from each stage: correct default action.
- Ctrl+C from each stage: clean exit, shell prompt.

### 11.4 Non-goals for testing

- No automated test that actually exec's tmux/ssh and asserts on the outcome. That's an integration test that requires real infrastructure and is not worth the maintenance burden in v1.

---

## 12. Performance Targets

| Metric                                       | Target            | Hard ceiling |
| -------------------------------------------- | ----------------- | ------------ |
| Time from process start to first frame       | < 100 ms          | 200 ms       |
| Time to first probe result (with ControlMaster) | < 100 ms       | 500 ms       |
| Time to first probe result (cold ssh)        | < 1500 ms         | 3000 ms      |
| Binary size (stripped)                       | < 12 MB           | 20 MB        |
| Memory footprint at idle                     | < 20 MB           | 50 MB        |

If `limen` blocks the UI from rendering on a slow probe, that's a bug. Probes must always be off the UI thread.

---

## 13. Open Questions

These are deliberate punts — decisions to make during implementation or in a v2.

1. **Theme customization.** Should v1 ship a `--theme dark|light|none` flag, or wait for v2? Recommendation: wait.
2. **Sorting by recency.** Should "most recently attached" auto-bubble to the top of the list? Recommendation: no, keep config order in v1. Add `--sort recency` as a v2 feature.
3. **`prefix + s`-style live preview.** Could the details pane show a tiny screenshot of the session's current pane? Recommendation: out of scope. Way too much complexity for marginal value.
4. **Auto-detach handling.** If a session has an existing attached client, `tmux attach -t name` would steal it. Should `limen` use `attach -d` to force-detach the other client? Recommendation: default to attach (shared session), with a future `--detach-others` flag.
5. **What if `~/.ssh/config` has the host?** Users may want `limen` to honor `~/.ssh/config` Host blocks rather than re-specify user/port. This is already implicit — if the hostname in `hosts.json` matches a Host alias, ssh resolves it. The `user` and `port` in `hosts.json` would override the ssh_config values. Document this in the README.
6. **Plugin or hook system.** Out of scope for v1. The tool is small and opinionated; if extensibility becomes a real need it can be added without breaking compatibility.

---

## 14. Glossary

- **Stage 1 / host picker:** the first screen, where the user picks where to go.
- **Stage 2 / session picker:** the second screen, where the user picks (or creates) a tmux session on the chosen host.
- **Probe:** the act of running `ssh host tmux ls` to discover sessions and reachability.
- **Exec:** specifically `syscall.Exec`, replacing the `limen` process with tmux or ssh.
- **State:** the small JSON file at `~/.local/share/limen/state.json` tracking last-attached timestamps.
- **Config:** the user-managed JSON file at `~/.config/limen/hosts.json` declaring SSH targets.
- **Reachable / unreachable:** whether a probe succeeded within its timeout.
- **Attached / unattached:** whether a tmux session currently has a client connected.

---

## 15. Acceptance Criteria

A v1 release is complete when:

1. `limen` can be installed via `nix run github:<owner>/limen` and produces a working binary.
2. With no `hosts.json` present, opening `limen` shows localhost and lets the user attach to an existing local tmux session or create a new one.
3. With a `hosts.json` declaring at least one remote host that is reachable, the host appears in the list with status `●`, sessions are listed in the details pane, and selecting the host advances to stage 2 with that host's sessions.
4. Selecting a session on either localhost or a remote execs into tmux and the user's terminal is now running inside that tmux session.
5. Escape from any stage takes the user to a fresh unnamed tmux session at the appropriate host.
6. Ctrl+C from any stage returns the user to a plain shell prompt with no orphaned processes.
7. `limen --version` prints the version.
8. `limen --help` prints usage.
9. The README explains setup, including ssh ControlMaster configuration for performance.
10. Unit tests pass; CI is green.

---

*End of document.*

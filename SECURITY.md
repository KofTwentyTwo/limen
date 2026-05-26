# Security Policy

## Supported versions

`limen` is in pre-1.0 development. Until a tagged 1.0 release, only the tip of the `develop` branch receives security fixes. Once 1.0 ships, the latest minor release receives fixes.

## Reporting a vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Instead, report privately by either of:

- **GitHub Security Advisory:** open a private advisory at <https://github.com/KofTwentyTwo/limen/security/advisories/new>.
- **Email:** [james@koftwentytwo.com](mailto:james@koftwentytwo.com).

Please include:

- A description of the issue and its impact.
- Steps to reproduce, or a proof-of-concept if you have one.
- The version of `limen` (or commit SHA) where you observed the issue.
- Your environment: OS, terminal emulator, Go version if relevant.

You can expect:

- An acknowledgement within 72 hours.
- A status update within 7 days, including a planned fix timeline or a rationale for closing the report.
- Credit in the release notes when the fix ships, if you'd like.

## Scope

`limen` is a CLI tool that reads JSON from local disk and shells out to `ssh` and `tmux`. The primary attack surface to consider:

- **`hosts.json` parsing.** Malformed input must fail safely, not exec unexpected commands.
- **SSH command construction.** Values from `hosts.json` end up as arguments to `ssh`. Verify nothing in the construction path allows argument injection (e.g. a hostname like `--option=...`).
- **The `tmux ls` output parser.** Output is treated as data; nothing should be evaluated as a command.

Out of scope:

- Vulnerabilities in `ssh`, `tmux`, the user's `~/.ssh/config`, or the bubbletea / lipgloss libraries themselves. Report those upstream.
- Bugs that require an attacker to already have write access to the user's `~/.config/limen/` directory — at that point they own the user account.

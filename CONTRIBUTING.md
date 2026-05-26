# Contributing to limen

Thanks for your interest. `limen` is in early design phase; this document captures the conventions we'll hold to as code starts landing.

## Source of truth

[`docs/DESIGN.md`](docs/DESIGN.md) is the canonical specification. If the design doc and the code disagree, that's a bug — open an issue, don't quietly drift one from the other.

If a PR proposes behavior that isn't in the design doc, the PR should either:
1. Update the design doc in the same PR, or
2. Be split: a design-doc PR first, an implementation PR after.

## Branch model

- `develop` is the default branch and the integration branch. PRs target `develop`.
- `main` is reserved for tagged release commits.
- Feature branches: `feature/<short-slug>`. Bug fixes: `fix/<short-slug>`.

## Commit messages

Conventional Commits style:

```
<type>(<scope>): <subject>

<body>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `build`.
Scopes for `limen` typically map to internal packages: `config`, `state`, `probe`, `ssh`, `ui`, `exec`, `flake`, `ci`, etc.

Examples:
- `feat(ui): add help overlay toggled by ?`
- `fix(probe): handle "no server running" stderr from tmux ls`
- `docs(design): clarify escape semantics in session picker`

## Code style

- Go: standard `gofmt`. `golangci-lint` runs in CI (will be added once Go code lands).
- Markdown: 100-column soft wrap; tables and code blocks may exceed.
- Comments: explain *why*, not *what*. Code shows what.
- Local tool dependencies are listed in [`docs/DEV-TOOLS.md`](docs/DEV-TOOLS.md).

## Tests

- All logic packages (`internal/config`, `internal/state`, `internal/ssh`, etc.) require unit tests.
- TUI packages use `teatest` for happy-path coverage of the bubbletea models.
- Run locally: `go test -race ./...`

## Pull requests

- Keep PRs focused. One feature, one fix, one refactor — not all three.
- Include a description that says what the change does and why. If a screenshot or asciinema illustrates the UX, attach it.
- The PR template will prompt for the standard fields.
- CI must be green before merge.

## Issues

- Bugs: use the bug-report template. Include OS, terminal, Go version (or `limen --version`), and minimal repro steps.
- Features: use the feature-request template. Frame the problem you're solving, not just the solution you have in mind.

## Releases

Tags trigger GitHub Actions, which runs the Go checks, builds macOS/Linux binaries for amd64/arm64, publishes GitHub Release assets, and updates the `KofTwentyTwo/homebrew-tap` formula. See [`docs/DESIGN.md` §10.4](docs/DESIGN.md).

Release prerequisites:
- `KofTwentyTwo/homebrew-tap` must exist.
- The `limen` repository must have a `HOMEBREW_TAP_TOKEN` Actions secret with contents read/write access to `KofTwentyTwo/homebrew-tap`.
- The release tag must use the `vMAJOR.MINOR.PATCH` form, for example `v0.1.0`.

## Code of conduct

See [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md). The short version: be respectful. Disagree about the code, not about the people.

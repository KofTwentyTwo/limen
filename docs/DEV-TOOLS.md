# Development Tools

This repo is intentionally light on local tooling. The following packages cover
normal development, release verification, and runtime smoke testing.

## Home Manager Package Set

```nix
home.packages = with pkgs; [
  go_1_24
  gopls
  gotools

  git
  gh
  ripgrep

  nix
  actionlint
  shellcheck

  ruby
  tmux
  openssh
];
```

## Why Each Tool Is Here

| Tool | Use |
|------|-----|
| `go_1_24` | Builds and tests the Go module declared in `go.mod`. |
| `gopls` | Go language-server support for editors. |
| `gotools` | Standard Go helper tools used by editors and local workflows. |
| `git` | Source control, branches, tags, and tap repo updates. |
| `gh` | GitHub issue, PR, repository, and release workflows. |
| `ripgrep` | Fast repo search while developing and reviewing. |
| `nix` | Flake checks, package builds, and downstream Nix integration testing. |
| `actionlint` | Static validation for GitHub Actions workflows. |
| `shellcheck` | Shell validation used directly and by `actionlint`. |
| `ruby` | Local syntax checks for the Homebrew formula. |
| `tmux` | Runtime dependency for local session creation and attachment. |
| `openssh` | Runtime dependency for remote host connections. |

# Summary

<!-- One or two sentences on what this PR changes and why. -->

## Scope

- [ ] This PR matches a behavior described in [`docs/DESIGN.md`](../docs/DESIGN.md).
- [ ] If the PR changes intended behavior, the design doc is updated in this PR (or a preceding one).
- [ ] If the PR adds a non-trivial dependency, the rationale is in the summary above.

## Changes

<!-- Bullet list of the meaningful changes. Group by package or area if it spans several. -->

-
-

## How to test

<!-- Commands a reviewer can run locally to verify. -->

```sh
go test -race ./...
go build ./...
./limen --version
```

## Screenshots / asciinema

<!-- If the change is user-visible (TUI / output), attach a still or recording. Skip for pure refactors. -->

## Checklist

- [ ] Tests added or updated, where appropriate.
- [ ] `go vet ./...` passes.
- [ ] No new TODOs that aren't tracked in an issue.
- [ ] Commit messages follow Conventional Commits (`feat:`, `fix:`, `docs:` …).

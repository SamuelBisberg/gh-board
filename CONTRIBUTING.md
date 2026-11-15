# Contributing to gh-board

Thanks for taking the time to contribute!

## Prerequisites

- [Go](https://go.dev) 1.25+ (see `go.mod`)
- [git](https://git-scm.com)
- [GitHub CLI](https://cli.github.com) (`gh`), to run the extension locally

## Getting started

```sh
git clone https://github.com/SamuelBisberg/gh-board.git
cd gh-board
go build ./...
```

Install it as a local `gh` extension to try your changes end-to-end:

```sh
gh extension install .
gh board
```

> [!TIP]
> After changing code, `gh extension remove board && gh extension install .` picks up a fresh build.

## Making changes

```sh
go build ./...
go vet ./...
gofmt -l .          # should print nothing
go test -race ./...
```

All four must pass before opening a PR - CI runs the same checks.

### Project layout

| Package      | Responsibility                                                         |
| ------------ | ---------------------------------------------------------------------- |
| `cmd/`       | The `gh board` command tree ([cobra](https://github.com/spf13/cobra))   |
| `pkg/board`  | Statuses, PR-state-to-Status mapping, Card operations, `board.yml`      |
| `pkg/gh`     | Wrapper around the `gh` binary; implements `--dry-run` for GitHub calls |
| `pkg/git`    | Wrapper around the system `git` binary                                 |
| `pkg/tmpl`   | Embedded issue/PR templates and placeholder rendering                  |
| `pkg/config` | Persisted settings (`~/.config/gh/board.json`)                         |
| `pkg/tui`    | Terminal styling, spinner, and interactive prompts and lists           |

### Tests

Command tests live in `cmd/` and run the `runX` functions against real, disposable git repositories (with bare "remotes") and a fake `gh`: a tiny program built once from `cmd/testdata/fakegh` answers from canned rules and logs every call (see `cmd/main_test.go`). No network access. Follow that pattern for new behavior: assert on the `gh`/`git` calls and resulting state, not on internals.

> [!WARNING]
> Try the extension from a throwaway repo and a test Board, not from a checkout of this repo: `gh board new` and `gh board push` create and push real branches and PRs. Use `--dry-run` to preview.

## Submitting a pull request

1. Fork the repo and create a branch off `main`.
2. Make your change, with tests.
3. Run the checks above.
4. Open a PR - the template will guide you through the rest.

## Reporting bugs / requesting features

Please use the issue templates; they ask for just enough context to act on a report quickly.

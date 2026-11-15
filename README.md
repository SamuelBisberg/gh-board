<div align="center">

# gh-board

*One card per unit of work, on a GitHub Project board, from a Todo issue to a merged PR*

</div>

`gh-board` is a [GitHub CLI](https://cli.github.com) extension that keeps a GitHub Project **Board** in step with the work you do. Capture work as an issue, start it, push, open it for review, and its card moves through the columns by itself. The pull request is the source of truth; the Board only mirrors it. Terms in **bold** are defined in [GLOSSARY.md](GLOSSARY.md).

## Install

```sh
gh extension install SamuelBisberg/gh-board
gh auth refresh -s project -s read:org   # scopes the extension needs
gh board init
```

`gh board init` uses your active `gh` account (or one of your organisations, or `--owner`), lets you pick one of its projects or create a new one titled "@you's Work", checks it against [board.yml](pkg/board/board.yml) and writes the config. It uses your existing `gh auth login` session and never stores a token. Upgrade with `gh extension upgrade board`.

## The daily flow

```console
$ gh board todo "Upgrade filtering"     # capture it: issue -> Todo
$ gh board new --issue 12                # start: branch + draft PR -> In Progress
  ... commit ...
$ gh board push                          # push; the board stays in step
$ gh board open -r ana                   # ready for review -> Needs Review, ana requested
$ gh board review bob                    # ask someone else later
$ gh board sync                          # reviews and merges move the card
```

Every command that changes GitHub accepts `--dry-run`, which prints the planned `gh`/`git` calls and changes nothing.

## Columns

| Status | What puts a card here | Card is |
|---|---|---|
| **Todo** | An open issue you created with `gh board todo`, with no PR yet | Issue |
| **In Progress** | A draft PR | PR |
| **Needs Review** | A ready PR with no review decision | PR |
| **Changes Requested** | A ready PR with `CHANGES_REQUESTED` | PR |
| **Approved & Merged** | An approved or merged PR | PR |
| **Closed** | Closed without merging; set by the Board's own *Item closed* workflow, never by the tool | Issue or PR |

Rules: one card per unit of work (the PR's card replaces the issue's); only cards you authored are ever changed; `sync` overwrites manual drags; merged PRs are never newly tracked; nothing is added from assignments.

## Commands

| Command | What it does |
|---|---|
| `gh board` | Interactive list of your Todo issues (`--all` for every repo); `enter` starts work, `o` opens in the browser |
| `todo <title> [-b body] [-R owner/repo]` | Create an issue assigned to you and add it as Todo |
| `new [branch] [base] [-t title] [--issue N]` | Branch from `origin/<base>`, push, open a draft PR, add it as In Progress; with `--issue`, link the branch, add `Closes #N` and drop the issue's card |
| `push [git push args]` | Push (default `-u origin HEAD`), open a draft PR if none, re-derive the card's Status |
| `open [-t title] [-r a,b] [--no-reviewers]` | Open or ready the PR as Needs Review and request reviewers |
| `review [name ...]` | Request reviewers on the current branch's open PR |
| `team add\|remove\|list\|default` | Manage the teammates you can request as reviewers |
| `sync [--current]` | Re-derive every Status from its PR and reconcile issue cards |
| `init [--owner O] [--project N]` | Pick your Board, check it against `board.yml`, write config |
| `config list\|get\|set\|templates` | View or change settings and templates |

Reviewers for `open` come from `-r`, else `--no-reviewers`, else `reviewers.default`, else an interactive picker over your teammates. A plain `open` on an already-ready PR never re-notifies anybody.

## Templates

New issues and PRs get a body from the first match of: the repo's own template (`.github/PULL_REQUEST_TEMPLATE.md` etc.), your override in `~/.config/gh/board/templates/`, then the embedded default. `{{title}}`, `{{branch}}`, `{{base}}`, `{{repo}}` and `{{issue}}` are replaced. `gh board config templates` shows the active source; `--init` writes the defaults for you to edit.

## Configuration

Stored in `~/.config/gh/board.json` (or your platform's config dir).

| Key | Default | |
|---|---|---|
| `board.owner` | *(set by `init`)* | User or organisation that owns the Board; `init` defaults to your active `gh` account |
| `board.project` | *(set by `init`)* | The Board's project number |
| `todo.repo` | *(current repo)* | Where `todo` creates issues outside a repo |
| `branch.prefix` | `task` | Prefix for branches generated from an issue |
| `reviewers.default` | *(empty)* | Teammate aliases requested on every `open` |
| `ui.color` | `auto` | `auto`, `always` or `never` |
| `team` | *(empty)* | Edit with `gh board team` |

## Board setup

The Board needs a single-select **Status** field with the six options above, in that order, a board-layout view grouped by Status, and the built-in workflows listed in [board.yml](pkg/board/board.yml) (notably *Item added to project* **off**). `gh board init` creates a missing project and Status field, never edits an existing one, and prints manual steps for anything else it finds wrong.

## How it works

Everything is GitHub: cards are the issues and PRs themselves, managed through `gh project`. There is no server, database or local state beyond the config file. Status is derived from the PR's draft flag, state and review decision, so running `sync` repeatedly gives the same result.

## Known gaps

- Not yet exercised against live GitHub; the test suite uses a fake `gh`.
- Workflow `expected_action` values in `board.yml` are inferred and should be verified once in the Board's settings.
- `sync` costs about one API call per card.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

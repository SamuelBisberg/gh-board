# gh-board

A [GitHub CLI](https://cli.github.com) extension that keeps a GitHub Project board in step with your work. Capture an issue, start it, push, open it for review: the card moves by itself. The PR is the source of truth; the board only mirrors it.

## Install

```sh
gh extension install SamuelBisberg/gh-board
gh auth refresh -s project -s read:org
gh board init        # uses your active gh account; pick or create a board
```

## Daily flow

```console
$ gh board todo "Upgrade filtering"   # issue -> Todo
$ gh board new --issue 12             # branch + draft PR -> In Progress (issue card replaced)
$ gh board push                       # push; card stays in step
$ gh board open -r ana                # ready for review -> Needs Review, ana requested
$ gh board sync                       # reviews and merges move the cards
```

Add `--dry-run` to any command to print the planned `gh`/`git` calls without changing anything.

## Columns

| Status | A card is here when |
|---|---|
| Todo | an issue you created with `todo` has no PR yet |
| In Progress | its PR is a draft |
| Needs Review | its PR is ready, no review decision |
| Changes Requested | a reviewer requested changes |
| Approved & Merged | its PR is approved or merged |
| Closed | closed unmerged (set by the board's own workflow, never by the tool) |

Only cards for things you authored are changed. `sync` overwrites manual drags: move a card by changing the PR.

## Commands

```sh
gh board                              # pick a Todo issue and start it (--all: every repo)
gh board todo "title" [-b body] [-R owner/repo]
gh board new [branch] [base] [-t title] [--issue N]
gh board push [--issue N]             # opens a draft PR if there is none
gh board open [-r ana,bob | --no-reviewers] [--issue N]
gh board review bob                   # request reviewers later
gh board team add ana-lopez-1234 ana  # alias for -r ana
gh board team default ana             # requested on every `open`
gh board sync [--current]
gh board init [--owner O] [--project N]
gh board config list|get|set|templates
```

`new --issue 12` names the branch `task/<issue-title>` and adds `Closes #12` to the PR. `open` requests reviewers from `-r`, else `reviewers.default`, else a picker; it never re-notifies on an already-ready PR.

## Templates

Issue and PR bodies come from the first match of: the repo's own template, your override in `~/.config/gh/board/templates/`, then the embedded default. Placeholders: `{{title}}` `{{branch}}` `{{base}}` `{{repo}}` `{{issue}}` (a line using `{{issue}}` is dropped when there is no issue).

```sh
gh board config templates --init      # write the defaults to edit
```

## Configuration

`~/.config/gh/board.json`, edited with `gh board config set <key> <value>`:

| Key | Default |
|---|---|
| `board.owner`, `board.project` | set by `init` (your account by default) |
| `todo.repo` | current repo |
| `branch.prefix` | `task` |
| `reviewers.default` | none |
| `ui.color` | `auto` |

## Board requirements

The board needs a **Status** field with the six options above, in order, and a board view grouped by Status. `init` creates a missing project and Status field but never edits an existing one.

GitHub has no API to change project workflows, so the tool checks them instead. Before every command, *Item closed*, *Pull request merged*, *Code changes requested* and *Code review approved* must be **on**, and *Item added to project* **off**. If not, the command stops and tells you which to fix at `https://github.com/<users|orgs>/<owner>/projects/<n>/workflows`. The full definition is in [board.yml](pkg/board/board.yml); terms are in [GLOSSARY.md](GLOSSARY.md).

## Status

Tested against a fake `gh`; not yet run against live GitHub. See [CONTRIBUTING.md](CONTRIBUTING.md) to help.

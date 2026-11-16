# gh-board

A [GitHub CLI](https://cli.github.com) extension that keeps a GitHub Project board in step with your work. Capture an issue, start it, push, open it for review: the card moves by itself. The PR is the source of truth; the board only mirrors it.

## Install

```sh
gh extension install SamuelBisberg/gh-board
gh auth refresh -s project -s read:org
gh board init        # uses your active gh account; name your board (reused if it exists)
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
gh board browse [-p]                  # open the board on github.com (-p: print the URL)
gh board init [--owner O] [--title "Board name"]
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

## Board setup

`gh board init` asks what to name your board (default `@<you>'s Work`). A board with that exact title is reused after a confirmation, otherwise it is created; `--title` does the same without prompting. Then it sets the board up for you: it creates the Status field (or rewrites its options to the six above, keeping the IDs of options that already exist so cards stay put) and creates the `PRs` board view. Re-running it on a correct board changes nothing.

The one thing GitHub exposes read-only is project workflows. `init` lists any of *Item closed*, *Pull request merged*, *Code changes requested*, *Code review approved*, *Auto-close issue*, *Auto-archive items* (on) and *Item added to project* (off) that differ, with the link to flip them. In a terminal, `init` offers to walk you through them: it opens each workflow's page in your browser, tells you what to click, then re-reads the board to confirm the change took (skip any step you like). Piped or scripted, it just prints the list. They are optional: they only make cards move the instant a review happens, and `gh board sync` re-derives every Status anyway. The full definition is in [board.yml](pkg/board/board.yml); terms are in [GLOSSARY.md](GLOSSARY.md).

## Status

Tested against a fake `gh`; not yet run against live GitHub. See [CONTRIBUTING.md](CONTRIBUTING.md) to help.

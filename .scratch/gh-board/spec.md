Status: implemented (see README.md; not yet exercised against live GitHub)

# Spec: gh-board, a GitHub CLI extension that keeps a Work Board in step with your PRs

Source material: `tmp/README.md`, `tmp/GLOSSARY.md`, `tmp/board.yml` (the design). Terms in **bold** follow `tmp/GLOSSARY.md`; when the implementation lands, that glossary moves to the repo root `GLOSSARY.md`.

## Problem Statement

I track the work I own across an organisation's repositories on a GitHub Project **Board**, one **Card** per unit of work. Today I keep it current with ad-hoc bash functions (`pr_new`, `pr_open`, `pr_push`, `pr_sync`) in my shell profile. They have no issue capture, no reviewer support, no templates, no reconciliation of issues against PRs, and they are not shareable or testable. Cards drift out of step with reality, and I sometimes drag them by hand.

## Solution

A compiled `gh` extension, `gh board`, built on the existing Go skeleton in this repo (cobra commands, `pkg/config`, `pkg/git`, `pkg/tui`). I capture work as an issue (**Todo**), **Start work** on it, push, open it for review and the **Card** moves through the Board's columns by itself. The Pull Request or Issue is the **Source of truth**; the Board only mirrors it, and `gh board sync` re-derives every **Status** from GitHub.

## User Stories

1. As a developer, I want `gh board todo "title"` to create an issue assigned to me and put it on the Board as Todo, so that capturing work takes one command.
2. As a developer, I want the issue body to come from the repo's own issue template, else my override, else an embedded default, so that issues are consistent without per-repo files.
3. As a developer, I want `-b body` on `todo` to use my explicit body as given, so that I can skip templating.
4. As a developer, I want `todo` to use the current repo, else `todo.repo` from config, else `-R owner/repo`, so that I can capture work from anywhere.
5. As a developer, I want `todo` to print the issue URL, so that I can open or share it.
6. As a developer, I want `todo` to set no labels or milestone, so that it stays minimal.
7. As a developer, I want `gh board new <branch> [base]` to fetch, branch from `origin/<base>` (default: repo default branch), and carry over uncommitted changes, so that I can start where I am.
8. As a developer, I want `new` to make an empty commit `chore: start <branch>`, so that GitHub accepts a PR with no real commits yet.
9. As a developer, I want `new` to push and open a draft PR assigned to me, so that my work is visible immediately.
10. As a developer, I want the PR title derived from the branch name (dashes to spaces) with `-t` to override, so that titles need no typing.
11. As a developer, I want the PR body from the repo PR template, else my override, else the embedded default, so that PRs follow convention.
12. As a developer, I want a new draft PR added to the Board as In Progress, so that the Board shows what I am working on.
13. As a developer, I want `gh board new --issue N` to link the branch to the issue, append `Closes #N` to the PR body (unless already present) and remove the issue's Card, so that there is one Card per unit of work.
14. As a developer, I want `new --issue N` with no branch name to generate `<branch.prefix>/<slug-of-issue-title>`, so that I never invent branch names.
15. As a developer, I want `gh board push [git push args]` to push (default `-u origin HEAD`) and open a draft PR if the branch has none, so that one command keeps GitHub current.
16. As a developer, I want `push` to set the Card's Status from the PR's current state, so that the Board stays in step.
17. As a developer, I want `push` to refuse on the default branch, so that I never open a PR from it.
18. As a developer, I want `gh board open` to push and open a ready (non-draft) PR as Needs Review, so that review starts in one step.
19. As a developer, I want `open` on an existing draft PR to mark it ready and leave my edited body alone, so that I do not lose edits.
20. As a developer, I want `open` on an already-ready PR to only re-derive its Status, so that it is safe to repeat.
21. As a developer, I want `open` to refuse when there are no commits ahead of base, when the PR is closed or merged, or on the default branch, so that I get a clear error instead of a broken PR.
22. As a developer, I want `open -r name,name` to request those **Teammates** (alias or plain login) as **Reviewers**, replacing the defaults, so that I choose per PR.
23. As a developer, I want `open --no-reviewers` to request nobody, so that I can open quietly.
24. As a developer, I want `open` to fall back to `reviewers.default`, then to an interactive picker (space toggles, enter confirms, esc skips), so that reviewers are always a conscious choice.
25. As a developer, I want a plain `open` on an already-ready PR to never re-request reviewers, so that nobody is re-notified.
26. As a developer, I want the PR author never requested as a reviewer, so that GitHub does not error.
27. As a developer, I want `gh board review [name ...]` to request reviewers on the current branch's open PR after the fact, so that I can add people later.
28. As a developer, I want `review` to refuse on drafts (pointing me at `open`) and on closed or merged PRs, so that misuse is explained.
29. As a developer, I want `review` to skip Teammates who already have a pending request and not change the Card's Status, so that it is safe and side-effect free.
30. As a developer, I want `gh board team add <login> [alias]` to verify the login exists and belongs to the Board's organisation, so that typos are caught.
31. As a developer, I want aliases unique and case-insensitive, defaulting to the login, so that `-r ana` just works.
32. As a developer, I want `team remove <name>` (alias or login) to also drop the Teammate from `reviewers.default`, so that no dangling defaults remain.
33. As a developer, I want `team list` to show alias, login and whether each is a default reviewer, so that I can see my setup.
34. As a developer, I want `team default [name ...]` to set `reviewers.default`, and print it when given no names, so that defaults are scriptable.
35. As a developer, I want a bare `gh board team` to open an interactive list with `d` (remove, confirmed) and `*` (toggle default), so that I can manage Teammates visually.
36. As a developer, I want `gh board sync` to add every local branch's open or draft PR of mine to the Board and set its Status, so that nothing I work on is missing.
37. As a developer, I want `sync --current` to limit step one to the current branch, so that I can sync quickly.
38. As a developer, I want sync to re-derive the Status of every PR on the Board that I authored, in any repo, so that the Board is correct everywhere.
39. As a developer, I want sync never to touch Cards for items other people authored (not **Mine**), so that Teammates' Cards are safe.
40. As a developer, I want sync to **Reconcile** issues: remove an Issue's Card when a Mine PR closes it, and restore it to Todo when that PR was closed unmerged while the Issue is still open with no other open PR, so that Todo is accurate.
41. As a developer, I want sync to edit only Cards whose Status differs and print a table of PR, old status and result (plus skipped branches without a PR), so that I see what changed.
42. As a developer, I want sync never to newly track already-merged PRs, only refresh merged ones already on the Board, so that history does not flood the Board.
43. As a developer, I want sync and every other command to never write or overwrite Closed, leaving Closed Cards alone, so that the Board's own workflow owns that column.
44. As a developer, I want Status derived purely from `isDraft`, `state` and `reviewDecision` (draft: In Progress; ready with none or `REVIEW_REQUIRED`: Needs Review; `CHANGES_REQUESTED`: Changes Requested; `APPROVED` or merged: Approved & Merged), so that repeated runs give identical results.
45. As a developer, I want adding an already-tracked PR to return the existing Card, so that `new`, `push`, `open` and `sync` are idempotent.
46. As a developer, I want bare `gh board` to open an interactive list of my Todo issues for the current repo (`--all` for the whole Board), so that I can pick what to start.
47. As a developer, I want `enter` in that list to start work on the issue (`new --issue N`), `o` to open it in the browser and `esc` to quit, with `j`/`k` also moving, so that the list is fast to use.
48. As a developer, I want `enter` on an issue from another repo to print `cd ~/work/<repo>` and the command to run, because an extension cannot change my shell's directory, so that I am not left confused.
49. As a developer, I want `--dry-run` on every command that changes GitHub to print the planned `gh`/`git` calls and change nothing, so that I can preview safely.
50. As a developer, I want `gh board init` to list the projects I can see, let me pick one (default org `AWS-PLAT-LZ-SAFE`, project 8), and write the config, so that setup is a one-time wizard.
51. As a developer, I want `init` to create a missing project and a missing Status field (six options in order), so that a fresh Board works.
52. As a developer, I want `init` never to edit an existing Status field, so that Cards stay attached to their option IDs.
53. As a developer, I want `init` to check the view exists and each built-in workflow is enabled or disabled as `board.yml` lists (notably **Item added to project** off), and on mismatch print the manual steps and exit non-zero, so that I know exactly what to fix in the UI.
54. As a developer, I want `init` to use my existing `gh auth login` session (needs `project` and `read:org` scopes) and never store a token, so that credentials stay with `gh`.
55. As a developer, I want `gh board config list|get|set` for `board.org`, `board.project`, `todo.repo`, `branch.prefix`, `team`, `reviewers.default` and `ui.color`, so that settings are inspectable and editable.
56. As a developer, I want `gh board config templates` to print the active source of each template and `--init` to write the embedded defaults to `~/.config/gh/board/templates/`, so that I can customise them.
57. As a developer, I want template placeholders `{{title}}`, `{{branch}}`, `{{base}}`, `{{repo}}`, `{{issue}}` replaced and unknown ones left as written, so that templates are predictable.
58. As a developer, I want the repo's own PR template (`.github/PULL_REQUEST_TEMPLATE.md` and the other standard locations) and `.github/ISSUE_TEMPLATE.md` honoured, and issue-template directories and YAML forms ignored, so that non-interactive creation always works.
59. As a developer, I want the extension to refuse clearly when `gh` is not authenticated or lacks scopes, so that failures are understandable.
60. As a maintainer, I want the extension released via the existing `.github/workflows/release.yml` flow and installed with `gh extension install SamuelBisberg/gh-board`, so that it ships like `gh-ver`.

## Implementation Decisions

- Build on the existing skeleton: cobra root with subcommands, the JSON config in the user config dir, the git helper package, the TUI package (spinner, select, styles). Rename the module to `gh-board`, replace the version-bump behaviour of the root command with the interactive Todo list, and replace the `gh-ver` README.
- Modules to add or extend:
  - **Board**: loads the embedded Board definition (`board.yml`); resolves project ID, Status field ID and the six option IDs once per run; adds a PR or issue to the Board idempotently; sets or removes a Card; lists Board items.
  - **Status derivation**: a pure mapping from PR state, draft flag and review decision to a Status, or "ignored" for closed-unmerged. Closed is never produced.
  - **GitHub client**: a thin wrapper that shells out to `gh` (`gh project`, `gh issue`, `gh pr`, `gh api graphql`), returning typed results. All GitHub access goes through it. Parses the PR URL from `gh pr create` stdout (no `--json` on `gh` 2.46).
  - **Templates**: embedded `issue.md` and `pr.md` (Go `embed`), resolution order (explicit body, repo template, user override, embedded default), placeholder rendering, and the `Closes #N` rule.
  - **Teammates**: alias/login store inside config, org-membership validation, default reviewers, alias resolution.
  - **Reconcile and sync**: local-branch discovery, Board refresh restricted to Mine, issue reconciliation, diff-only edits, result table.
  - **Commands**: `todo`, `new`, `push`, `open`, `review`, `team`, `sync`, `config` (with `templates`), `init`, and the root interactive list. Each has a `runX(...)` entry function.
- Config schema keys: `board.org`, `board.project`, `todo.repo`, `branch.prefix` (default `task`), `team` (list of alias/login), `reviewers.default` (alias list), `ui.color` (`auto|always|never`). The six Status names are fixed.
- Mine is the author of the issue or PR compared with the logged-in `gh` user; because the Board item list lacks authors, each Board PR is looked up individually.
- Reviewers are plain review requests (`gh pr create --reviewer`, or `gh pr edit --add-reviewer` after `gh pr ready`); the extension never writes the Board's Reviewers field.
- No server, database, daemon or local state beyond config; every Status is re-derived from GitHub.
- `init` creates only the project and Status field; it never edits an existing field, creates views or configures workflows, and reports mismatches with manual steps.
- Cross-org PRs are trackable. Nothing is added to Todo automatically from assignments.
- `--dry-run` is implemented once, at the GitHub client and git helper boundary, so that every mutating command honours it uniformly.

## Testing Decisions

- A good test exercises external behaviour only: given a repo state, GitHub state and flags, what ends up on the Board, in git and on stdout. Tests never assert internal call structure beyond the observable `gh`/`git` effects.
- There is exactly one seam: the cmd-level `runX(...)` functions, run against real temporary git repos with a bare `origin`, and a fake `gh` executable placed on `PATH` that serves canned JSON/GraphQL responses and records the calls it receives. Status derivation, templates, Reconcile, teammate resolution and dry-run are all covered through this seam, not as separate unit-test seams.
- Prior art: `cmd/root_test.go` (`setupRepo`, `runGit`, `withAuth`), `cmd/common_test.go` (`isolateGH`), `cmd/init_test.go`, `cmd/config_test.go`, and `pkg/git` test helpers (`newTestRepo`, `newBareRemote`).
- The fake `gh` is a new shared test helper, reused by every command's tests.
- Cover at least: each row of the Status table, idempotent re-adds, `new --issue` Card replacement, `open` reviewer precedence, no re-notify on ready PRs, Reconcile both directions, Mine filtering, Closed never written, merged PRs not newly tracked, template precedence, `Closes #N` dedupe, and `--dry-run` making no mutations.

## Out of Scope

- Creating Board views or configuring built-in workflows through the API (not supported by GitHub); `init` only checks and instructs.
- Editing an existing Status field's options.
- Writing the Closed Status or the Reviewers field.
- GitHub teams as reviewers (individual users only).
- Labels, milestones and issue-template forms or directories.
- Batching Board refresh into a single GraphQL query (noted as a later optimisation).
- Retiring the old bash functions in `~/.bashrc`; they stay until the extension is adopted.
- Verifying the Changes Requested and approved-while-open transitions against the real Board (noted as a known gap).

## Further Notes

- The workflow `expected_action` values in `board.yml` are inferred from workflow names, since the API only reports enabled or disabled; verify them once in the Board's Workflows settings.
- Syncing overwrites manual Status edits to Cards you authored; move a Card by changing the PR, not by dragging it.
- The design files currently live in `./tmp`; promote `GLOSSARY.md` to the repo root and fold `README.md` and `board.yml` into the shipped repo as part of the first slice.

## Comments

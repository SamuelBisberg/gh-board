Status: implemented (branch board-naming; interactive prompts not exercised against live GitHub)

# Spec: Name the Board at `init`, and reuse an existing Board by its name

## Problem Statement

When I run `gh board init` I can't choose what my **Board** is called. The tool creates it with a fixed title from `board.yml`, and I only get a "Which board?" picker that lists existing boards by number. Reusing an existing Board means scanning numbers and titles, and creating a new one gives me no say over its name. I want to name the Board myself, with a sensible default, and have `init` recognise a Board I already named.

## Solution

`gh board init` asks one question, "Board name?", pre-filled with the default title `@<login>'s Work`. The name is the Board's identity:

- If a Board with that name already exists under the owner, `init` asks whether to use it.
- If none exists, `init` creates it.

The "Which board?" picker goes away. In scripts, `--title "X"` does the same without prompting. `--project` is removed because the title replaces it.

## User Stories

1. As a developer running `gh board init` interactively, I want to be asked what to name my Board, so that it isn't named for me.
2. As a developer, I want the name pre-filled with `@<login>'s Work`, so that pressing Enter gives a sensible Board name.
3. As a developer, I want to edit the pre-filled name freely, so that I can use any name I like.
4. As a developer, I want an empty answer to fall back to the default name, so that a stray Enter never creates an unnamed Board.
5. As a developer, I want leading and trailing whitespace in the name ignored, so that typos don't create lookalike Boards.
6. As a developer, I want Esc to abort `init` without creating or changing anything, so that I can back out safely.
7. As a developer, I want `init` to notice when a Board with the name I typed already exists, so that I don't create a duplicate.
8. As a developer, I want to be asked "Use existing board #N?" in that case, so that I confirm before reusing a Board.
9. As a developer, I want to be asked for a different name if I decline to reuse the existing Board, so that I can still create a new Board.
10. As a developer, I want `init` never to create a second Board with an existing title, so that a name always identifies one Board.
11. As a developer with two Boards that share a title, I want a small picker of the matches (`#4`, `#9`), so that I can say which one I mean.
12. As a developer who already ran `init`, I want the name field pre-filled with my configured Board's current title, so that re-running `init` to re-check the Board setup is just Enter.
13. As a developer who renamed my Board on GitHub, I want re-running `init` to show the renamed title, so that the tool never works from a stale name.
14. As a developer, I want the Board name matched exactly (case-sensitive, after trimming), so that matching is predictable.
15. As a developer, I want `--owner` to still choose whose Board it is, so that I can use an organisation's Board.
16. As a script author, I want `gh board init --title "X"` to reuse the Board titled "X" if it exists and create it otherwise, so that setup is idempotent.
17. As a script author, I want `--title` to skip every prompt and confirmation, so that it runs unattended.
18. As a script author, I want `init` with no flags and no terminal to use the configured Board, or the default title if none is configured, so that it keeps working in CI.
19. As a script author, I want `init` to fail with a clear error when several Boards share the requested title, so that it never guesses.
20. As a developer, I want the error to tell me to rename one of the duplicate Boards, so that I know how to fix it.
21. As a developer, I want `--project` removed, so that there is one way to choose a Board by name.
22. As a developer, I want `gh board config set board.project N` to keep working, so that I can still point at a Board by number in an edge case.
23. As a developer, I want the Board's title not stored in my config, so that it never disagrees with GitHub.
24. As a developer, I want the Status field and board view set up as before on whichever Board is chosen or created, so that naming doesn't change the rest of `init`.
25. As a developer, I want `init` to print which Board it created or reused, with its name and number, so that I can see what happened.
26. As a developer, I want the README and `board.yml` comments to describe the new flow, so that the docs match the tool.

## Implementation Decisions

- The default title lives in the Board definition (`board.yml`) as `@{{login}}'s Work`, with `{{login}}` replaced by the active gh account. It is kept as it is and acts as the pre-filled default.
- `init` resolves the Board in one step from a **requested title**:
  - Interactive: the answer to the "Board name?" prompt.
  - `--title`: the flag value.
  - Non-interactive with no `--title`: the configured Board if one is set, else the default title.
- Resolution against the owner's existing projects:
  - No match: create a Board with that title.
  - One match: reuse it. Interactive asks "Use existing board #N?" first, and a "no" returns to the name prompt. `--title` reuses without asking.
  - Several matches: interactive shows a picker of the matches. Non-interactive fails with an error telling the user to rename one.
- Matching is exact and case-sensitive on the trimmed title. Input is trimmed, and empty input becomes the default title.
- The prompt is pre-filled with the configured Board's current title (looked up by `board.project`) when one is set. Otherwise it is pre-filled with the default title.
- The configured Board is identified by number, as today. Changing owner still clears the stored project number.
- The owner is chosen exactly as before (`--owner`, else configured, else interactive pick, else the active account).
- The "Which board?" picker and the "Create a new board" option are removed. `--project` is removed. `board.project` stays as a stored value and stays settable through `config set`.
- The title is never persisted to the config file.
- Everything after the Board is chosen or created is unchanged: reading the live Board, saving config, Status field and board view setup, and workflow advice.
- The prompt is a new text-input helper in the terminal UI package, alongside the existing pickers and confirm.
- No glossary change and no ADR.

## Testing Decisions

- A good test checks external behaviour only: which `gh` calls `init` makes, the config it saves, the output, and the error returned. It does not check internal helpers.
- The seam is the existing one: `runInit` against the fake `gh`. It runs without a terminal, so these cases are tested through the `--title` and no-flag non-interactive paths:
  - `--title` with no existing match creates a Board with that exact title and saves its number.
  - `--title` with one existing match creates nothing and saves that Board's number.
  - `--title` with several matches returns the rename error and saves nothing.
  - No `--title`, none configured: creates a Board with the default `@<login>'s Work` title.
  - No `--title`, a Board configured: reuses it and creates nothing.
  - Surrounding whitespace in `--title` is trimmed before matching.
  - Matching is case-sensitive.
- The interactive prompt, the confirm, and the duplicate picker are thin wrappers over the resolution logic and are not unit tested.
- The existing init tests pass a project number to `runInit`. They are updated to the new signature and to choosing the Board by title.
- Prior art: `TestInitCreatesABoardForTheActiveAccount`, `TestInitFixesTheBoardAutomatically` and `TestInitLeavesAMatchingBoardAlone` in the command tests, which use the fake `gh` rules.

## Out of Scope

- Renaming an existing Board from `gh board`.
- Case-insensitive or fuzzy matching of titles.
- Storing the Board title in config.
- Changes to the Status field, board view, or workflow advice.
- Changes to the Glossary or any ADR.

## Further Notes

- The `@` in the default title is deliberate and stays.
- Because the title is the Board's identity at `init`, renaming a Board on GitHub means the next `init` shows the new name pre-filled. Day-to-day commands keep working because they use the stored owner and project number.

# Work Board

A personal tracker for the work you own across an organisation's repositories, shown as a GitHub Project board with one column per stage of work.

## Language

**Board**:
The organisation-level GitHub Project that shows all tracked work, one column per Status.
_Avoid_: Project (ambiguous with a repo or product), kanban

**Card**:
One unit of work on the Board: either a Todo issue or a Pull Request. A unit of work has at most one Card.
_Avoid_: Item, ticket, task

**Status**:
The Board column a Card sits in. Exactly one of: Todo, In Progress, Needs Review, Changes Requested, Approved & Merged, Closed. The tool sets the first five; Closed is set only by the Board's own workflow.
_Avoid_: State, stage, column (column is only the visual form of a Status)

**Mine**:
A Card whose issue or Pull Request you authored. Only Mine cards are ever changed by the tooling.
_Avoid_: Owned, assigned

## Work

**Todo**:
Work you intend to get to, captured as an open Issue that has no Pull Request yet.
_Avoid_: Backlog, idea, draft item

**Issue**:
A GitHub issue created for a piece of Todo work, assigned to you, living in a repository.
_Avoid_: Ticket, story

**Start work**:
Begin a Todo Issue by creating a branch linked to it and opening its Draft Pull Request; the Pull Request replaces the Issue's Card.
_Avoid_: Pick up, claim

**Draft PR**:
A Pull Request not yet ready for review. Its Status is In Progress.
_Avoid_: WIP PR

**Ready PR**:
A non-draft open Pull Request. Its Status is Needs Review, Changes Requested or Approved & Merged depending on its review decision.
_Avoid_: Open PR (drafts are open too)

**Template**:
The body text the tooling gives a new Issue or Pull Request, taken from the repository's own template if it has one, else the user's override, else the one embedded in the extension.
_Avoid_: Boilerplate, skeleton

## People

**Teammate**:
A GitHub user you have saved, with a short alias, as someone you can ask to review your Pull Requests.
_Avoid_: Contact, collaborator, member

**Reviewer**:
A Teammate or other GitHub user who has been requested to review a Pull Request.
_Avoid_: Approver

## Tracking

**Track**:
Add a Pull Request to the Board and set its Status from its current state.
_Avoid_: Register, import

**Sync**:
Re-derive the Status of every Mine Card from its Pull Request and reconcile Issue Cards against their Pull Requests.
_Avoid_: Refresh, update

**Reconcile**:
Remove an Issue's Card once a Mine Pull Request closes that Issue, and restore it to Todo if that Pull Request is closed unmerged while the Issue is still open.
_Avoid_: Dedupe, clean up

**Source of truth**:
The Pull Request or Issue itself; the Board only mirrors it, so manual Status edits are overwritten by Sync.
_Avoid_: Master, canonical copy

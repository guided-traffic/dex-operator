# ADR 0001: Documentation Has Five Homes, Tickets Are Numbered Work Lists That Get Archived, and an Open Security Finding Is Embargoed

## Status

Accepted. Date: 2026-10-07. The owner asked for the documentation of this repository to be built
like the `docs/` tree of a sibling project; this record adopts that project's rules for this
repository.

**Implemented** in the working tree on 2026-10-07: the five directories exist with their rules
pages; `SECURITY_ARCHITECTURE.md` was split into one page per perspective under
[docs/security/](../security/README.md) and deleted; `DEVELOPER.md` was split into
[docs/developer/](../developer/README.md) and deleted; [SECURITY.md](../../SECURITY.md) carries the
reporting route; [docs/tickets/README.md](../tickets/README.md) carries the ticket rules and the
work lists written before that date were numbered and archived or kept open by them;
[`.gitignore`](../../.gitignore) carries `docs/tickets/**/local_*`.

## Context

The repository had three root documents — `README.md`, `DEVELOPER.md`,
`SECURITY_ARCHITECTURE.md` — and a handful of unnumbered work lists, some at the root and ignored
only by one person's global excludes file, some under `docs/tickets/` and untracked. The work
lists carried decisions that were nowhere else (why connectors get their own allowlist, why CORS
origins are derived and not listed), `CLAUDE.md` pointed at a section of one of them, and the
security document ended in a checkbox list that mixed operator advice with open gaps. A single
security document cannot be finished and so cannot be checked; a work list that carries the only
copy of a decision cannot be closed.

The repository is public. A work list that describes an unfixed attack path in detail must not
reach it before the fix does.

## Decision

**D1 — A statement has exactly one home.**

| Kind | Home |
|---|---|
| A decision — what the operator does and why, what was rejected | an [ADR](README.md) |
| How the code works — a subsystem, an invariant, the contributor workflow | [docs/developer/](../developer/README.md); there is no `DEVELOPER.md` at the root |
| What somebody running the operator needs — installation, upgrading, runtime behaviour | [docs/operations/](../operations/README.md) |
| The threat model, and the gap each mechanism leaves | [docs/security/](../security/README.md) — one page per perspective, each closing with `## What this does not cover`; there is no `SECURITY_ARCHITECTURE.md`. Reporting a vulnerability is [SECURITY.md](../../SECURITY.md) |
| Work still outstanding | a [ticket](../tickets/README.md), archived when the work lands |

**D2 — `README.md` is the front page and carries the reference.** The naming conventions, the
fast start and the fully populated example of every custom resource are in the README and nowhere
else; a page under `docs/operations/` explains a behaviour without restating the reference.

**D3 — A ticket is a work list and nothing else.** `docs/tickets/NNN-<slug>.md`, closed by moving
it to `docs/tickets/archive/` when the work lands. **The extraction is the close**: the decision
goes into an ADR, the operator-facing consequence into the README or `docs/operations/`, the
security-relevant one into `docs/security/`, the contributor knowledge into `docs/developer/` — an
archived ticket is history, never the source of a current rule.

**D4 — A number is never reused,** not an embargoed ticket's, not a merged ticket's. The numbering
command in the rules page reads deleted files from git history for that reason.

**D5 — A finding goes into an existing ticket first.** The open ticket of the same subject
collects it; a new ticket only when none fits.

**D6 — A ticket carries no history.** Current state, required changes, open questions with an
answer line, nothing struck through, nothing dated. A changed fact is rewritten.

**D7 — An open security finding is embargoed.** A ticket with `security: live` or
`security: boundary` whose finding is not fixed keeps the `local_` prefix and stays untracked
through the repository's own `.gitignore` line; no tracked file, commit message or pull request
carries its details or its file name. The embargo ends at the fix, not at `state: done`; a
dropped or risk-accepted finding is published only on the owner's explicit, dated acceptance.

**D8 — Nothing outside `docs/tickets/` cites a ticket.** Not by number, label, file name or path;
cite the ADR instead. A ticket may cite an ADR; an ADR does not cite a ticket.

**D9 — A security page is one perspective, and it ends with its limits.** One page per
perspective under `docs/security/`, every page closing with `## What this does not cover`; an
open gap carries a stable `H-<n>` identifier in its heading and lives in the page whose mechanism
has it, never in a central list. No checkbox lists on these pages: an operator's mitigation is
written into the gap it mitigates.

**D10 — English, everywhere.** Code, comments, commit messages, documentation. Conversation with
the owner may be German; the repository is not.

## Consequences

- Five directories and three root documents to keep in step: whoever changes behaviour updates
  the page that describes it in the same change, or the page is wrong.
- The extraction discipline makes closing a ticket slower than deleting it. That is the cost of
  archived tickets never being cited.
- The embargo puts security tickets outside git until fixed; a lost laptop loses them. The
  alternative is publishing an attack path in a public repository.
- A link that pointed at `DEVELOPER.md` or `SECURITY_ARCHITECTURE.md` from outside this
  repository breaks; the files are gone, not redirected.

## Alternatives Considered

- **Keep the three root documents** (`README.md` / `DEVELOPER.md` / `SECURITY_ARCHITECTURE.md`).
  The security document had grown into one page that mixed trust boundaries, secret flow,
  tenancy, RBAC and a hardening checklist; the sibling project left the same layout because a
  document nobody finishes is a document nobody checks. Lost.
- **Tickets as GitHub issues.** Would publish every analysis at once, embargoed findings
  included, and split the work lists from the repository the rules live in. Lost.
- **Leave the work lists at the root, ignored by a global excludes file.** Depends on one
  person's machine; a clone without that file would track them on the first `git add .`. Lost;
  D7 puts the ignore rule into the repository.

## Residual risks

- D8 has no automated check. A `git grep -n -E 'T[0-9]+\b|docs/tickets/[0-9]'` outside
  `docs/tickets/` is the manual one.
- Not verified: that `git check-ignore` reports the `docs/tickets/**/local_*` rule for a `local_`
  path under `archive/` in a clone without the owner's global excludes file — the owner's global
  file ignores `local_*` as well, so this machine cannot tell the two rules apart for that path.

## References

- [docs/tickets/README.md](../tickets/README.md) — the ticket rules, frontmatter and body skeleton
- [docs/security/README.md](../security/README.md) — the form of a security page
- [docs/developer/README.md](../developer/README.md), [docs/operations/README.md](../operations/README.md)
- [`.gitignore`](../../.gitignore), [SECURITY.md](../../SECURITY.md)

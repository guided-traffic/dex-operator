# Tickets

One file per ticket, kept next to the code it changes and tracked with it — except while it is
embargoed, see [below](#an-open-security-finding-is-embargoed). **A ticket is a work list.** It
holds what has to be done, what was verified and how, and what was deliberately left out — and
when the work lands it is **moved to [archive/](archive/)**. Nothing outside this directory may
reference a ticket; decisions live in [docs/adr/](../adr/) and may be referenced from anywhere.
The rule and the reasoning behind it are
[ADR 0001](../adr/0001-documentation-has-five-homes-and-tickets-are-work-lists-that-get-archived.md).

Before archiving a ticket, move anything durable out of it — the decision into an
[ADR](../adr/), the operator-visible consequence into [README.md](../../README.md) or
[docs/operations/](../operations/), the security-relevant one into [docs/security/](../security/),
the contributor-facing one into [docs/developer/](../developer/) — then `git grep` its number and
its T-label and clear whatever is left. **The extraction is the close** — archiving is only what
happens to the file afterwards, and an archived file is history, never a source of a current rule.

This page carries the rules only. There is no index table here and no board: the tickets' own
frontmatter is the index ([below](#there-is-no-index-table-and-no-board)).

## Naming and numbering

`NNN-<kebab-slug>.md`, three digits, the slug naming the outcome or the defect. A new ticket
takes the next free number — the highest existing number plus one — and its `id:` is `T` and
that number (ticket 001 is `T1`). **A number is never reused**, including the number of an
embargoed ticket and of a ticket that was merged into another. Moving a ticket into `archive/`
changes its directory, not its name.

The highest existing number, archived and embargoed tickets included — the `local_` prefix is
stripped before sorting — printed from the repository root:

```
{ ls docs/tickets docs/tickets/archive; git log --all --format= --name-only -- docs/tickets | xargs -n1 basename; } \
  | sed -nE 's/^(local_)?([0-9]{3})-.*/\2/p' | sort -n | tail -1
```

The `git log` half counts the numbers of merged tickets, whose files are deleted. Run it in the
owner's working tree: `local_` files are gitignored, so a fresh clone cannot see an embargoed
ticket's number and would hand it out a second time.

Numbering starts on 2026-10-07. The work lists written before that date — Markdown files without
a number at the repository root and in this directory — were numbered in the order their work
was done, given a frontmatter, and archived or kept open by the same rules as every later ticket.
There are no legacy labels.

## An open security finding is embargoed

The repository is public. A ticket is **embargoed** while its frontmatter says `security: live`
or `security: boundary` and a live or boundary finding in it is not fixed; `state: done` alone
does not end it. **The embargo ends when the finding is fixed, not when the ticket closes**: a
ticket can reach `done` with a live or boundary risk accepted rather than fixed. A `dropped` live
or boundary finding is unfixed by definition, so it stays embargoed unless the owner explicitly
accepts publishing it; that acceptance is recorded as the frontmatter field
`publication-accepted:` with its date before the file is renamed. While a ticket is embargoed:

- **Its file carries the `local_` prefix**: `local_NNN-<kebab-slug>.md`. The repository's own
  [`.gitignore`](../../.gitignore) carries `docs/tickets/**/local_*`, so the file stays untracked
  in every clone, in `docs/tickets/` and in `archive/` alike, independent of anyone's global
  excludes file. Neither rule stops `git add -f`.
- **No tracked file carries its exploit details** — not its threat line, its mechanism, its
  attack path, and not its file name, whose slug states the finding. For the same reason
  neither does a commit message or a pull request. A tracked file may name the embargoed
  ticket's id, severity, security class, effort and state, and puts in place of the details:
  "embargoed security finding, open - details in its own ticket file until it is fixed".
- **It still takes its number**, so the tracked numbering has gaps where embargoed tickets sit.

When the `shipped:` line of a `done` ticket names the fix of every live or boundary item of the
ticket, the file is renamed without the prefix and is tracked from then on like any other
ticket. A ticket that reaches `done` or `dropped` with a live or boundary item whose risk was
accepted rather than fixed keeps the prefix until the owner's explicit, dated publishing
acceptance. A `hardening` finding is not embargoed.

The `local_` prefix inside this directory means *embargoed* and nothing else. A hand-off note for
a change in another repository is not a ticket of this one and does not live here.

## Nothing outside this directory references a ticket

No file outside `docs/tickets/` — not `README.md`, not a page under `docs/`, not an ADR, not
`CLAUDE.md`, not a code comment — and no commit message or pull request cites a ticket: not its
number, its T-label, its file name or its path. Cite the ADR that holds the rule instead. A
ticket may cite an ADR; an ADR does not cite a ticket.

## There is no index table and no board

The frontmatter `state:` line of each ticket is the index:

```
grep -rH '^state:' --include='[0-9]*.md' --include='local_[0-9]*.md' docs/tickets
```

The `--include` patterns keep this page out of the result, because the frontmatter example
below starts its lines with `state:` as well. An embargoed ticket shows up in the grep only on
a machine that holds its file.

## The filing rule

**A finding goes into an existing ticket first.** Before anything new is opened, the finding is
assigned to the open ticket whose subject it shares - the same component, mechanism or kind of
change - and becomes part of that ticket's current state and required changes. A ticket collects
similar findings. Only when no open ticket fits is a new ticket opened. A collecting ticket is
still one subject: findings that share only the event or the analysis that found them are not
collected in one file. A row somewhere, or a finding that lives only in a report, is never
enough.

**An `## Open questions` section is expected wherever a decision is open**: one subsection per
question, understandable on its own, with the sensible options, the recommended one marked and
the mark justified, and an `**Answer:**` line that the owner fills in. Decisions are put to the
owner one at a time. An adversarial pass is warranted for severity high+ or security
live/boundary.

## Frontmatter

Copy the skeleton below into `NNN-<kebab-slug>.md` (`local_NNN-<kebab-slug>.md` while
[embargoed](#an-open-security-finding-is-embargoed)). Every field is mandatory unless marked
optional.

```yaml
---
id: T1
title: a rotated Secret does not reach dex and every reconcile restarts dex
state: filed          # filed -> analysed -> decided -> in-progress -> done | dropped
severity: high        # critical | high | medium | low | cosmetic  (impact if never fixed)
security: none        # live | boundary | hardening | none         (see scale below)
threat:               # required unless security: none
urgency: now          # derived via the rules below, never gut feeling
effort: M             # XS | S | M | L
blocked-by: decision  # optional: decision | human | product | release | adr-NNNN | T<NN>
filed-from:           # optional: the analysis or event that produced it
opened: 2026-10-07
decided:              # date, when state reaches decided
done:                 # date, when state reaches done
shipped:              # optional, set with done: one line naming what shipped (for an embargoed ticket: the fix of every live or boundary item)
dropped-reason:       # optional, set with dropped: one line
publication-accepted: # optional, embargoed tickets only: date of the owner's explicit acceptance to publish an unfixed finding
---
```

### Security scale (threat-model based, not high/low)

| Class | Meaning | The `threat:` line must name |
|---|---|---|
| `live` | weakens a guarantee the repo already gives, no hostile principal needed | the guarantee and who relies on it |
| `boundary` | real trust-boundary hole, needs a hostile principal | principal + verb + target (e.g. "a principal who may create connectors in a namespace admitted for static clients issues identities every client of the installation accepts") |
| `hardening` | defense-in-depth, no concrete attack path today | what it would additionally cover |
| `none` | not security | — |

A blanket statement ("could be a security risk") does not fill the `threat:` field.

### Urgency derivation — apply top-down, first match wins

| Rule | Urgency |
|---|---|
| Defect in an **unreleased feature on the current branch**, or a **measured-false statement in tracked files** | `now` |
| Gates the release, or is gated on it | `release` |
| severity ≥ medium **and** trigger live | `next` |
| Everything with a decided fix or a cheap known fix | `later` |
| Needs a product call, a human re-decision, or a threat-model escalation nobody has accepted | `icebox` |

Urgency is recomputed when facts change (a dormant trigger goes live, a severity gets
re-measured); the comment of the `urgency:` line names the rule that matched.

## Body skeleton

**A ticket shows the current state and nothing else.** There is no History section. When a
fact, a question or an answer changes, the text is rewritten to the new state: nothing is struck
through, nothing is annotated with a date, and no account of how the code or the ticket evolved
is kept - git keeps that for a tracked ticket. Done work items are removed. The content must be
graspable in a few minutes.

```markdown
## Current state

What the code, configuration or documentation does today and why that is a problem, with
file:line links. Impact in a few lines: who hits it and what breaks; if security != none, the
principal, the action and the target. Only the facts the change or a decision needs.

## Required changes

The target state: what changes where (files, functions, docs, ADRs) and the tests that prove it
("merged" is not verification). Split into "Independent of the open questions" and "Depends on
the answers" when both exist.

## Open questions            <- wherever a decision is open

### Q1: <the question in one sentence>

Context in one to three sentences. The sensible options, each with its cost or consequence; the
recommended one marked and justified.

**Answer:** _open_

## Not verified              <- only if something unknown still matters

What is not known, and what would settle it.

## Related                   <- optional

One line per related ticket.
```

## Index maintenance

- State change → the frontmatter `state:` and its date field, in the same commit-of-work.
- `done` → `state: done`, `done:` stamped, and the one-line `shipped:` field. Then the
  extraction above, then the move to [archive/](archive/) — and, for an embargoed ticket, the
  rename without the `local_` prefix only when the `shipped:` line names the fix of every live or
  boundary item; otherwise it keeps the prefix until the owner's explicit, dated publishing
  acceptance (`publication-accepted:`, [above](#an-open-security-finding-is-embargoed)).
- `dropped` → `state: dropped` and the one-line `dropped-reason:` field, then the move to
  [archive/](archive/). An embargoed ticket keeps its `local_` prefix and moves to
  [archive/](archive/) with it; the repository ignore line keeps it untracked there.
- Merging tickets (the [filing rule](#the-filing-rule)) keeps the number and the `id:` of the
  ticket that takes the others in; the merged files are deleted, not archived, and every reference
  to their ids is pointed at the surviving ticket.

# Architecture Decision Records

Every durable decision of this project lives here, one file per decision family. An ADR
records **what was decided, why, what was rejected, and what it costs** — so a later change
can argue with the decision instead of rediscovering it.

## Format

Filename: `NNNN-kebab-case-title.md`, numbered in the order they were written.

Sections, in this order:

| Section | Content |
|---|---|
| `# ADR NNNN: Title` | The decision as a title, not a topic |
| `## Status` | `Accepted`, or `Accepted, amended <date> (…)`, plus `Date:` and what is actually implemented versus open |
| `## Context` | The forces and the concrete failure that made the decision necessary |
| `## Decision` | `D1 … Dn`, each a rule that holds going forward, in present tense |
| `## Consequences` | What this costs, including the parts nobody likes |
| `## Alternatives Considered` | Each option and why it lost |
| `## Residual risks` | Accepted risks, open items, and what was **not** verified |
| `## References` | Relative links to the code and to sibling ADRs |

Ground rules: English only; every claim verified against the code, with unverified statements
marked as such; identifiers (`functions`, fields, constants) quoted exactly so the ADR stays
checkable against the tree. An ADR here may link into the code, and it cites no ticket — the rule
is [ADR 0001](0001-documentation-has-five-homes-and-tickets-are-work-lists-that-get-archived.md).

## Keeping them current

**An ADR is part of the code, not a historical note.** When a decision changes, the ADR is
updated in the same change — the `Decision` section states the new rule, the `Status` records
the amendment with its date, and the superseded rule is marked in place rather than deleted.
A reader must never find the old rule stated as current.

**A decision that changes an existing record is an amendment of that record, never a new
one.** No ADR amends, supersedes, overrides or invalidates another; when an answer touches
several records, each of them is amended in place and says why. A new ADR is written only for
a decision no existing record covers.

The record's row in the index below changes in the same change as its `Status`.

## How a decision gets here

An open decision lives in a ticket's `## Open questions` section
([docs/tickets/README.md](../tickets/README.md)), is put to the owner one at a time with its
options and a recommendation, and **an answered question becomes an amendment of the record it
changes, or a new ADR when no record covers it, in the same session**.

ADR 0002 to ADR 0007 were written on 2026-10-07 from decisions taken earlier and recorded until
then in the work lists of the features, in `CLAUDE.md` and in the root documents
`DEVELOPER.md` and `SECURITY_ARCHITECTURE.md`; each record's `Status` names the release that
built it.

## Index

Every record here is **Accepted**. The *State* column is the coarse build state as of
2026-10-07: **Implemented**, **Partly built** or **Not built**. The record's own `Status`
section is the authority; this column is a reading aid.

| ADR | Decision | State |
|---|---|---|
| [0001](0001-documentation-has-five-homes-and-tickets-are-work-lists-that-get-archived.md) | A statement has one home — ADR, `docs/developer/`, `docs/operations/`, `docs/security/`, ticket — with the reference in `README.md`; no `DEVELOPER.md` or `SECURITY_ARCHITECTURE.md` at the root; tickets are numbered work lists that get archived; an open security finding is embargoed | Implemented |
| [0002](0002-custom-resources-carry-no-secret-values-credentials-reach-dex-through-an-env-secret.md) | Custom resources carry only references to Secrets in their own namespace; credentials reach dex through the env Secret, never through `config.yaml`; file material is a mount path; the operator never deletes a Secret | Partly built: the file mounts are reported, not created |
| [0003](0003-the-render-is-deterministic-and-change-driven-no-periodic-requeue.md) | The render is deterministic and change-driven: children sorted, the config Secret compared as parsed YAML, referenced Secrets watched with deletions ignored, a rollout restart only on a config change, no periodic requeue | Implemented |
| [0004](0004-a-static-client-is-confidential-through-a-secret-or-public-through-an-inline-id-validated-by-cel.md) | A static client is confidential through `secretRef` or public through an inline `clientID`; the conditional rules are CEL on the spec, there is no admission webhook | Implemented |
| [0005](0005-a-static-client-opts-in-to-derive-its-cors-origins-from-its-own-https-redirect-uris.md) | `spec.cors` derives the https origins of the client's own `redirectURIs` into `web.allowedOrigins`, appended sorted after the authored list | Implemented |
| [0006](0006-two-namespace-allowlists-one-per-child-category-exact-names-only.md) | Two allowlists on the installation — `allowedNamespaces` for static clients (empty denies), `allowedConnectorNamespaces` for connectors (omitted = own namespace) — exact names only, enforced at build time and in the child status | Implemented |
| [0007](0007-crds-ship-through-a-helm-hook-job-not-the-crds-directory.md) | The chart applies its CRDs in a pre-install/pre-upgrade hook Job, not through Helm's `crds/` directory | Implemented |

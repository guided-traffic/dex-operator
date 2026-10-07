# Operating the dex-operator

For the people who install the operator and keep a Dex installation running with it.
[README.md](../../README.md) is the short version — what the operator is, the fast start, the
naming conventions, and the fully populated reference of every custom resource. A page here
explains a behaviour; it never restates the reference, which lives in the README and nowhere else
([ADR 0001](../adr/0001-documentation-has-five-homes-and-tickets-are-work-lists-that-get-archived.md) D2).

| Page | Read it when |
|---|---|
| [runtime.md](runtime.md) | You want to know when a change reaches Dex: what triggers a render, when Dex restarts and when it does not, why a rotated secret needs a restart, mounting file material, reading the `Ready` conditions, Secrets left behind |
| [upgrade.md](upgrade.md) | You upgrade the operator: what the CRD hook does, where a release's behaviour changes are written, what to check afterwards |

## The other documentation

| Where | What |
|---|---|
| [README.md](../../README.md) | What the operator is, the fast start, the naming conventions, the complete reference |
| [docs/security/](../security/README.md) | Trust boundaries, where the credentials live, and what each mechanism leaves open — with what an operator can do about each gap. Reporting a vulnerability is [SECURITY.md](../../SECURITY.md) |
| [docs/adr/](../adr/README.md) | Why the operator behaves the way it does, and what was rejected |
| [docs/developer/](../developer/README.md) | Changing the code |

# Conventions

## Code

- Go 1.27 (`go 1.27.1` in [go.mod](../../go.mod)), controller-runtime.
- Cyclomatic complexity below 15 per function (`make cyclo`); refactor rather than raise it.
- Errors wrap with `%w`; a build error names the resource and the Secret reference it failed on,
  never a Secret's value.
- Never edit generated files — `zz_generated.deepcopy.go`, `config/crd/bases/`,
  `config/rbac/role.yaml`, `deploy/helm/dex-operator/files/crds/` — change the source types or
  markers and run `make generate-all`. The chart's `templates/clusterrole.yaml` is not generated
  and is edited with the markers ([helm-chart.md](helm-chart.md)).
- The builder stays free of Kubernetes client code; everything it needs from the cluster comes in
  through `Input` ([builder.md](builder.md)).
- A change to the rendering keeps existing output byte-identical unless it means to change it
  ([architecture.md](architecture.md#determinism-matters)).
- A conditional validation rule is a CEL marker; there is no admission webhook.

## Tests

New features and bug fixes come with tests; reconciliation logic is the coverage priority. Every
test command goes through `make` ([build-test-lint.md](build-test-lint.md)).

## Commits

Conventional Commits — semantic-release reads them ([ci-and-release.md](ci-and-release.md)). A
behaviour change an operator must act on goes into the commit body. No commit message, pull
request or file outside `docs/tickets/` cites a ticket; cite the ADR
([ADR 0001](../adr/0001-documentation-has-five-homes-and-tickets-are-work-lists-that-get-archived.md) D8).

## Documentation

- English, in code, comments, commits and documentation.
- A statement has one home: decision → [ADR](../adr/README.md); how the code works → this
  directory; running the operator → [docs/operations/](../operations/README.md); threat and gap →
  [docs/security/](../security/README.md); the fields of every custom resource → the reference in
  [README.md](../../README.md); outstanding work → a [ticket](../tickets/README.md).
- Whoever changes behaviour updates the page that describes it in the same change.
- Every documented fact is verified against the code; values shown in the README are marked
  `# default` (only where one exists) or `# example`.

## Security-relevant changes

A change to what a tenant can contribute, to a credential's path, to an RBAC rule or to a
validation rule updates the page of [docs/security/](../security/README.md) whose mechanism it
touches, including its `## What this does not cover`. An open finding whose ticket is embargoed is
not described in a tracked file before the fix ([docs/tickets/README.md](../tickets/README.md)).

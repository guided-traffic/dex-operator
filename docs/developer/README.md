# Developer documentation

The contributor entry point and the overviews for people changing this code. The detail is in the
code; what lives here is the shape of things — how the pieces fit, which invariants hold, why a
design that looks odd is the way it is, and the workflow around it.

**What belongs here:** anything a future developer needs before touching a subsystem, that the
code cannot state on its own, and everything about contributing: the layout, the build and test
matrix, continuous integration and the release, the checklists, the conventions.

**What does not:** decisions (those are [ADRs](../adr/README.md)), work lists (those are
[tickets](../tickets/README.md), archived when the work lands), what somebody running the operator
needs (that is [docs/operations/](../operations/README.md), with the reference in
[README.md](../../README.md)), and the security design (that is
[docs/security/](../security/README.md)).
[ADR 0001](../adr/0001-documentation-has-five-homes-and-tickets-are-work-lists-that-get-archived.md)
is the rule that separates those homes, and why there is no `DEVELOPER.md` at the root.

A page here **may and should** point at files and functions. That is the point of it. It also
means it goes stale when the tree moves, so whoever moves the tree updates the page in the same
change.

## What has to be in your head first

- **Dex is not installed by this operator.** The official Dex Helm chart runs Dex; the operator
  writes two Secrets into the installation's namespace — the config Secret with `config.yaml` and
  the env Secret with every credential — and, optionally, patches Dex's Deployment to restart it.
- **One controller owns the output.** The `DexInstallationReconciler` collects, builds and writes;
  the seventeen child reconcilers only maintain their own `Ready` condition. Every change of a
  child or a referenced Secret is funnelled back to the installation
  ([architecture.md](architecture.md)).
- **The builder is pure.** `internal/builder` knows no Kubernetes client; Secrets come in through
  a resolver function, so the whole render is unit-testable ([builder.md](builder.md)).
- **No secret value in a custom resource, none in `config.yaml`.** Specs carry `{name, key}`
  references resolved in their own namespace; values go to the env Secret, files to a mount path
  ([ADR 0002](../adr/0002-custom-resources-carry-no-secret-values-credentials-reach-dex-through-an-env-secret.md)).
- **The render is deterministic.** Children sorted, the config compared as parsed YAML — otherwise
  every reconcile restarts Dex. A builder change keeps existing output byte-identical
  ([ADR 0003](../adr/0003-the-render-is-deterministic-and-change-driven-no-periodic-requeue.md)).
- **Two allowlists, two categories.** `allowedNamespaces` admits static clients (empty denies
  all), `allowedConnectorNamespaces` connectors (omitted admits only the installation's own
  namespace) ([ADR 0006](../adr/0006-two-namespace-allowlists-one-per-child-category-exact-names-only.md)).
- **No webhook.** Conditional validation is CEL on the spec; everything that needs another object
  is found at build time ([docs/security/validation.md](../security/validation.md)).
- **`make` is the entry point**, from the repository root. CI runs Makefile targets; so do you
  ([build-test-lint.md](build-test-lint.md)).
- **Generated files are never edited.** Change `api/v1` or the RBAC markers and run
  `make generate-all`; the chart's ClusterRole is the one hand-kept copy.
- **A statement has one home**, and the page that describes behaviour changes with it
  ([conventions.md](conventions.md#documentation)). English only.

| Page | Read it when |
|---|---|
| [repository-layout.md](repository-layout.md) | You are new and want the tree, and which paths are generated |
| [package-map.md](package-map.md) | You are looking for where something lives and what each file is responsible for |
| [architecture.md](architecture.md) | You want the picture: the two controller kinds, the watches, the reconcile, determinism, the `ChildObject` interface, what runs where |
| [builder.md](builder.md) | You touch the render: `Build`, its input and output, the naming conventions, static clients, derived CORS origins |
| [helm-chart.md](helm-chart.md) | You touch the chart: the CRD hook, the templates, which files are generated |
| [build-test-lint.md](build-test-lint.md) | You want to build, generate, run, test or lint anything |
| [testing.md](testing.md) | You add a test, choose a tier, or a suite fails and you need to know what it is for |
| [ci-and-release.md](ci-and-release.md) | You touch a workflow, Renovate or the release |
| [adding-things.md](adding-things.md) | You add a connector type, a CRD field, an RBAC permission or a CI job |
| [conventions.md](conventions.md) | You write code, a commit, documentation or anything security-relevant |

## Core flows, one fact each

| Flow | The fact | Where |
|---|---|---|
| Startup | The manager sets up the `DexInstallationReconciler` first — it registers the field indexes — then one generic reconciler per child kind; leader election is on in the chart, metrics off | [architecture.md](architecture.md#what-runs-where), [cmd/main.go](../../cmd/main.go) |
| An installation reconcile | List the children by index, filter by the two allowlists, sort, build, write the config Secret (compared as YAML) and the env Secret (compared as bytes), restart Dex if the config changed, write the status | [architecture.md](architecture.md#the-reconcile-of-an-installation) |
| A child reconcile | The installation exists and admits the child's namespace → `Ready=True`; otherwise `Ready=False` and a requeue after five minutes; nothing is built | [architecture.md](architecture.md#child-errors) |
| A child changes | Its watch maps it to its `installationRef`, and the installation renders again | [architecture.md](architecture.md#how-a-change-reaches-the-render) |
| A Secret changes | The Secret index maps it to every installation whose children reference it; a deletion maps to nothing | [architecture.md](architecture.md#how-a-change-reaches-the-render) |
| An installation's spec changes | It renders again, and every child reconciler re-evaluates the children that reference it | [architecture.md](architecture.md#how-a-change-reaches-the-render) |
| A credential is rendered | Resolved in the child's namespace, written to the env Secret as `<TYPE>_<ID>_<FIELD>`, referenced as `$VAR` (`secretEnv` for a static client) | [builder.md](builder.md#conventions-encoded-here) |
| A file is needed | A fixed path in the config and an entry in `MountedSecrets`; nothing mounts it | [builder.md](builder.md#one-entry-point-no-cluster) |
| A CRD changes | `make generate-all` regenerates it and copies it into the chart; the chart's pre-upgrade hook applies it | [helm-chart.md](helm-chart.md) |
| A release | Conventional Commits on `main` → semantic-release tags → the image and the chart are published | [ci-and-release.md](ci-and-release.md) |

# Build, generate, test, lint

**`make` is the entry point** — the targets match CI exactly. Do not run `go test`, `golangci-lint`
or the generators directly. `make help` lists every target. What each test tier is for is
[testing.md](testing.md).

| Target | What it does |
|---|---|
| `make build` / `make run` | build `bin/manager` / run the manager locally against the current kubeconfig (both run `fmt` and `vet` first) |
| `make generate-all` | controller-gen: DeepCopy, CRDs and RBAC, then sync the CRDs into the Helm chart. Run after **any** `api/v1` change and after any RBAC marker change |
| `make manifests` / `make generate` | the two halves of `generate-all`: CRDs + RBAC + Helm sync / DeepCopy only |
| `make helm-sync-crds` | copy `config/crd/bases/*.yaml` into `deploy/helm/dex-operator/files/crds/` |
| `make lint` | `go vet`, `gofmt -l`, `golangci-lint` |
| `make lint-fix` | `golangci-lint` with fixes |
| `make cyclo` / `make cyclo-report` | the cyclomatic complexity gate — every function **< 15** — / the full report including tests |
| `make test` | fmt + vet + every Go test under envtest, profile `cover.out` |
| `make test-unit` / `make test-unit-coverage` | the `-short` run: unit tests only, no build tag |
| `make test-integration` / `make test-integration-coverage` | the envtest suite in `test/integration/` (build tag `integration`); a real API server, CEL enforced |
| `make test-e2e` | the e2e suite against a running Kind cluster with the operator installed (build tag `e2e`) |
| `make test-e2e-helm` | meant for a Helm migration e2e test (`TestE2E_Migrate`, tags `e2e,e2e_helm`); that test does not exist, so the target runs nothing ([testing.md](testing.md#e2e-kind)) |
| `make e2e-local` | the whole loop: Kind cluster, image build, kubectl image pull, both loaded into Kind, Helm install with [test/e2e/helm-values.yaml](../../test/e2e/helm-values.yaml), `test-e2e`, teardown |
| `make kind-create` / `kind-delete` / `kind-load` | the Kind cluster `dex-operator-test` by hand |
| `make gosec` / `make vuln` | security static analysis / govulncheck |
| `make docker-build` / `make docker-buildx` | image build (runs `generate-all` first) / multi-arch build and push; `IMG=` overrides the tag |
| `make install` / `make deploy` (and `uninstall` / `undeploy`) | the kustomize path in `config/` against the current kubeconfig — not the chart |
| `make coverage-merge` / `make coverage-json` | merge the unit and integration profiles / write the README badge JSON |

## Coverage

Coverage profiles exclude generated code (`zz_generated.*`, see `COVERAGE_EXCLUDE_RE` in the
[Makefile](../../Makefile)). The filter runs directly after every profile-producing test target,
so local reports, the CI merge step and the badge all measure hand-written code only.

## Before reporting a change done

`make lint`, `make cyclo` and the tiers the change touches — at least `make test-unit` and
`make test-integration` for anything in `internal/` or `api/`. A change to `api/v1` or to the RBAC
markers without `make generate-all` leaves the CRDs, the DeepCopy code and the chart behind.

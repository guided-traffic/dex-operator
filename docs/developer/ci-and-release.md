# CI and release

What runs on a pull request, on `main` and on a published release, and how the version is
derived. Every workflow runs on self-hosted runners.

## Test and Release — [release.yml](../../.github/workflows/release.yml)

On pull requests to `main`, pushes to `main` and `workflow_dispatch`:

| Job | What it runs |
|---|---|
| `linter` | `make lint` |
| `cyclomatic-complexity` | `make cyclo` |
| `gosec` | `make gosec` |
| `govulncheck` | `make vuln` |
| `malware-scan`, `container-malware-scan` | a malware scan of the source and of the built image |
| `unit-tests` | `make test-unit-coverage` |
| `integration-tests` | `make test-integration-coverage` |
| `coverage-report` | merges both profiles; on a pull request comments the coverage against `main`'s badge, on `main` creates `.github/badges/coverage.json` as an artifact, which `semantic-release` commits |
| `e2e-tests` | a Kind cluster, the image built and loaded with the kubectl image of the CRD hook, the chart installed, `make test-e2e` |
| `semantic-release` | on a push to `main` only, after every job above |

A new job goes into the `needs:` list of `semantic-release` in the same change, or a release can
be cut without it.

There is one pipeline per pull request and per branch (`concurrency` in the workflow). A new push to
a pull request — a Renovate rebase, for instance — cancels the run for the old head, whose checks
no longer count. On `main` a running pipeline finishes; of the pushes queued behind it only the
newest runs, since `semantic-release` publishes only when its commit is still the head of `main`.

## Versioning

[semantic-release](https://semantic-release.gitbook.io/) ([.releaserc.json](../../.releaserc.json),
`main` the one release branch) derives the version from **Conventional Commits**, tags, creates the
GitHub release with generated notes and commits the coverage badge as
`chore(release): <version> [skip ci]`. `feat:` is a minor, `fix:` a patch, `feat!:` or a
`BREAKING CHANGE:` footer a major. Commit messages therefore follow Conventional Commits; a
behaviour change an operator must act on goes into the commit body, so it reaches the release
notes.

## Release Docker & Helm — [build.yml](../../.github/workflows/build.yml)

On `release: published`:

- builds the multi-arch image and pushes it to Docker Hub as `guidedtraffic/dex-operator`, uploads
  an SBOM to the release, runs a Docker Scout scan;
- sets the chart's `version` from the tag, packages the chart, attaches only the current chart
  `.tgz` to the release, and publishes it to the Helm repository at
  `https://guided-traffic.github.io/dex-operator` (the `gh-pages` branch).

## Renovate — [renovate.yml](../../.github/workflows/renovate.yml), [renovate.json](../../renovate.json)

Automated dependency updates on a schedule. `conventional-changelog-conventionalcommits` is held at
v9 (major updates disabled in `renovate.json`): with v10,
`@semantic-release/release-notes-generator` silently drops all commits and the release notes come
out empty.

**Indirect Go modules** (`// indirect` in `go.mod`) get minor and patch updates. Their majors stay
disabled, because a transitive module's next major is a different module path that
controller-runtime and the Kubernetes modules still pin to the old one. Their digest updates stay
disabled too: a pseudo-version is a commit a parent module pinned, not a release, and the
default-branch head it would move to can break the parent's release — `k8s.io/kube-openapi`'s head
requires `sigs.k8s.io/structured-merge-diff/v7`, while `k8s.io/apimachinery` v0.37 is built on v6.
Such a module moves when its parent does, through `go mod tidy` in the parent's update. The gomod
manager extracts indirect requirements disabled, and the enable rule is evaluated before the lookup,
where no update type exists yet. That rule therefore must not carry `matchUpdateTypes` or any other
post-lookup matcher: it would never match, and no indirect module would be looked up at all — that
is how `golang.org/x/net` stayed on a vulnerable release. A second rule after it takes the majors
and digests back out.

**The Go version group** (`renovate/go-version`) carries the toolchain (`go.mod`, `Containerfile`,
`GO_VERSION` in [release.yml](../../.github/workflows/release.yml)), `golang.org/x/*`, and the minor
and patch updates of golangci-lint and gosec — the `# renovate:` annotated `*_VERSION` variables in
the [Makefile](../../Makefile). Both analysers read the compiler's export data through
`golang.org/x/tools`, and a toolchain release can raise its version beyond what the pinned release
decodes: Go 1.27.2 writes version 5, and golangci-lint v2.13.1 and gosec v2.29.0 failed `typecheck`
on every package. In one branch, the toolchain update stays red until the analyser release that
reads it is out, then Renovate adds that release to the branch and the pipeline reruns. In separate
branches the analyser update would merge alone and leave the toolchain PR red on its old run, since
Renovate rebases only on a conflict (`rebaseWhen: conflicted`). While gosec has no release on a new
enough `golang.org/x/tools`, `GOSEC_VERSION` can be a master pseudo-version; Renovate replaces it
with the next release.

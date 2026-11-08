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
| `release-toolchain` | on a pull request only: the [toolchain action](../../.github/actions/semantic-release-toolchain/action.yml), then `npx semantic-release --dry-run` |
| `semantic-release` | on a push to `main` only, after every job above except `release-toolchain` |

A new job goes into the `needs:` list of `semantic-release` in the same change, or a release can
be cut without it. A job that runs on pull requests only stays out of it: a skipped job in `needs:`
skips the release.

`release-toolchain` is the one check that runs Node and the semantic-release plugins before a
merge; without it a pull request that changes `.nvmrc`, `package.json` or the toolchain action is
first run on `main`. semantic-release loads its plugins and then ends a pull-request run before the
first plugin step, `--dry-run` would stop it before publishing in any case, and the job's token can
only read. It is not a required check of the `main` ruleset, which is why Renovate merges npm
updates itself (below).

Both Node jobs prepare the toolchain through one composite action,
[semantic-release-toolchain](../../.github/actions/semantic-release-toolchain/action.yml), so the
pull-request check cannot drift from the release it vouches for: it installs `libatomic1` if the
runner lacks it, Node from [.nvmrc](../../.nvmrc), the dependencies with `npm install
--ignore-scripts` (none has an install script, and in the release job the checkout holds a token
that can push), and runs `npm audit signatures`.

**The runners** are the stock `actions-runner` image, which carries little beyond the docker CLI;
a job installs what it needs itself. Node 26 links `libatomic.so.1`, which the image does not ship:
the loader exits with code 127 and `actions/setup-node` reports only "Node may not be installed".
Node used to be `lts/*`, so when 26 became the LTS line every release from 2026-11-02 on failed in
`Setup Node.js` while all checks stayed green. Since then `.nvmrc` pins the major, and a new one
arrives as a Renovate pull request that `release-toolchain` runs first.

There is one pipeline per pull request and per branch (`concurrency` in the workflow). A new push to
a pull request — a Renovate rebase, for instance — cancels the run for the old head, whose checks
no longer count. On `main` a running pipeline finishes; of the pushes queued behind it only the
newest runs, since `semantic-release` publishes only when its commit is still the head of `main`.

## Retry Renovate CI — [renovate-retry.yml](../../.github/workflows/renovate-retry.yml)

On the completion of a Test and Release run. A Renovate pull request whose pipeline failed would stay
red for good: GitHub's auto-merge waits, and Renovate pushes again only on a conflict
(`rebaseWhen: conflicted`). A runner that loses its connection or a flaky test parks the update. The
workflow reruns the failed jobs once — only for the first attempt of a pull-request run on a
`renovate/` branch of this repository, and only while that run is still for the branch head, since
a rerun joins the pull request's concurrency group and would cancel the run for a newer head. A
second failure stays red for a human.

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

Automated dependency updates daily, after every push to `main` and on `workflow_dispatch`. Minor
and patch updates merge without a human once their checks are green; majors wait for review, except
for GitHub Actions. Most updates use GitHub's auto-merge, which waits for the required checks of the
`main` ruleset.

**npm** (semantic-release and its plugins) minor and patch updates are merged by Renovate itself
(`platformAutomerge: false`), at its next run after every check is green: the check that exercises
them, `release-toolchain`, is not required, so GitHub's auto-merge would not wait for it.
`conventional-changelog-conventionalcommits` is held at v9 (major updates disabled in
`renovate.json`): with v10, `@semantic-release/release-notes-generator` silently drops all commits
and the release notes come out empty.

**Node** (`.nvmrc`) majors wait for review; `release-toolchain` shows on the pull request whether the
new major starts on the runners.

**Commit types**: a dependency update is a `fix`, so it cuts a patch release. GitHub Actions, npm and
Node are `chore`: they run in CI only and reach neither the image nor the chart.

**trivy-action** is pinned to a commit with its version as a comment, and the Trivy binary to a
release through the action's `version` input. The `container-malware-scan` job logs in to Docker Hub
before Trivy runs, so a branch ref (it was `master`) would run whatever lands there next with the push
credential on the runner. Upstream raises the binary's default on `master` without cutting an action
release, hence the separate pin. Renovate's github-actions manager updates both.

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

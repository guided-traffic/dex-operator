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

---
id: T8
title: make test-e2e-helm runs a Helm migration test that does not exist
state: filed
severity: low
security: none
threat:
urgency: now          # rule 1: a measured-false statement in tracked files (the target's help text)
effort: XS
blocked-by: decision
filed-from: the verification of the test matrix while moving DEVELOPER.md to docs/developer/
opened: 2026-10-07
decided:
done:
shipped:
dropped-reason:
publication-accepted:
---

## Current state

[Makefile:113-116](../../Makefile#L113-L116): `test-e2e-helm` ("Run Helm migration E2E test
(requires running Kind cluster with operator)") runs
`go test -tags=e2e,e2e_helm -run TestE2E_Migrate ./test/e2e/...`. No file in the tree carries the
`e2e_helm` build tag and nothing defines `TestE2E_Migrate` (`grep -rn 'e2e_helm\|TestE2E_Migrate'`
finds only the Makefile). `go test -run` with no matching test passes, so the target reports
success while testing nothing. No workflow calls it.

Impact: whoever runs it believes a Helm migration was tested. The removed `DEVELOPER.md` said the
e2e suite covers "the CRD upgrade hook and Helm migration"; the CRD hook runs in every e2e install,
a migration test does not exist.

## Required changes

Depends on Q1. Either way, [docs/developer/testing.md](../developer/testing.md) and
[build-test-lint.md](../developer/build-test-lint.md) follow in the same change.

## Open questions

### Q1: Write the migration test or remove the target?

What "Helm migration" meant is not recorded — most likely an upgrade from the last released chart
to the working tree with existing custom resources, checking that the CRD hook upgrades the CRDs
and the rendered config stays unchanged. Options: **remove the target** (recommended: one line,
the docs then say what is true, and the upgrade path is partly held by
`render_compat_test.go` already); or **write `TestE2E_Migrate`** behind the `e2e_helm` tag
(a real upgrade test — install the released chart, create resources, upgrade to the tree, compare
the config Secret — at the cost of a released chart download in the test and a CI job to run it).

**Answer:** _open_

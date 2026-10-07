# Security policy

## Reporting a vulnerability

Report privately. Please do **not** open a public issue for a finding that lets someone register a
client or an identity source they should not, read credentials, act under another identity, or
disrupt the single sign-on of an installation.

- Report through **GitHub private vulnerability reporting** on
  <https://github.com/guided-traffic/dex-operator> (Security → Report a vulnerability):
  <https://github.com/guided-traffic/dex-operator/security/advisories/new>.
- Or report to the maintainer organisation, <https://github.com/guided-traffic>.

> **Gap, stated plainly:** this repository publishes no contact address and no response time or
> disclosure window — stated rather than invented. Whether private vulnerability reporting is
> switched on for this repository is not verified here. If the form is not offered to you, the
> maintainer organisation is the remaining route.

Include the operator version (the image tag or the chart version), the Dex version, and the
relevant parts of the `DexInstallation` and the children involved. Leave out client secrets,
connector credentials and the contents of the generated env Secret.

## What is already known

The security design, including the gaps it does not close, is documented under
[docs/security/](docs/security/README.md). Anything written there is known — a report that adds a
working exploit, a wider consequence, or a case the analysis missed is still valuable. An open gap
there carries an `H-<n>` identifier in its heading; naming it in a report saves a round trip.
Anything that is **not** written there is what we most want to hear about.

## Supported versions

Releases are cut from `main` only: semantic-release is configured with `main` as its one release
branch ([.releaserc.json](.releaserc.json)), and there is no maintenance branch. A fix lands on
`main` and ships in the next release; there is no backport to an earlier release. Only the latest
release is supported.

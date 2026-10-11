# Adding things

Ordered checklists. Each step names the file; [package-map.md](package-map.md) says what the file
is for. Every list ends with `make lint`, `make cyclo` and the test tiers of
[build-test-lint.md](build-test-lint.md), and with the documentation the change touches.

## A new connector type

1. **API type** — `api/v1/dex<type>connector_types.go`: spec struct (with `InstallationRef`,
   `ID`, `DisplayName`), status embedding `CommonStatus`, kubebuilder markers (`shortName`,
   print columns), `init()` scheme registration.
2. **ChildObject** — the three methods in [api/v1/connector_helpers.go](../../api/v1/connector_helpers.go);
   `GetReferencedSecretNames` must list every Secret the spec can reference (it drives rotation
   reactivity, [architecture.md](architecture.md#the-childobject-interface)).
3. **Builder** — the slice in `ConnectorSet` ([builder.go](../../internal/builder/builder.go)), a
   `build<Type>Connector` with the `connectorBuildFunc` signature, and one `addConnectorUnits`
   line with the kind name in `connectorUnits` ([connectors.go](../../internal/builder/connectors.go))
   or `oauthConnectorUnits` ([connectors_oauth.go](../../internal/builder/connectors_oauth.go)) —
   its position there is its render position. Credentials go to env through `resolveEnvSecret`
   with `connectorEnvBase(<type>, id)` and a field, file material through `mountCertFile` or
   `mountSecretAsFile` ([builder.md](builder.md#conventions-encoded-here)). The kind then takes
   part in the ID contest and in skip-and-report without further code.
4. **Collection** — a collector method and the `ConnectorSet` field in
   [collect.go](../../internal/controller/collect.go).
5. **Wiring** — the type lists in `registerIndexers` and `childWatchSources`
   ([dexinstallation_controller.go](../../internal/controller/dexinstallation_controller.go)),
   `lookupChildSecretRefs` ([secret_watch.go](../../internal/controller/secret_watch.go)), the
   reconciler list in [connector_controller.go](../../internal/controller/connector_controller.go).
   The kind is a connector, so `checkChildNamespace` governs it by `allowedConnectorNamespaces`
   without a change.
6. **RBAC** — the markers in [rbac.go](../../internal/controller/rbac.go) **and** the kind and its
   `/status` in the chart's hand-written
   [clusterrole.yaml](../../deploy/helm/dex-operator/templates/clusterrole.yaml)
   ([helm-chart.md](helm-chart.md#generated-vs-hand-written)).
7. **Generate** — `make generate-all` (DeepCopy, CRDs, `config/rbac/role.yaml`, the chart's CRD
   copy).
8. **Tests** — a builder unit test (config rendering, env vars, mounts) and an integration test
   (reconcile, namespace allowlist, status) ([testing.md](testing.md)).
9. **Documentation** — the connector's fully populated example in the README's reference, its env
   var and mount path in the README's naming conventions, and
   [docs/security/connectors.md](../security/connectors.md) if the kind's trust differs from the
   others (an upstream the author does not control, headers, a switch rather than a connector).

## A field on an existing CRD

1. The field with its doc comment and markers in `api/v1/<kind>_types.go` — the doc comment is the
   field's description in the CRD and in `kubectl explain`. A conditional rule is a CEL
   `XValidation` on the spec struct; there is no webhook.
2. `make generate-all`.
3. The builder renders it ([builder.md](builder.md)); a credential is a `SecretKeyRef`, never a
   value ([ADR 0002](../adr/0002-custom-resources-carry-no-secret-values-credentials-reach-dex-through-an-env-secret.md)),
   and a new referenced Secret is added to `GetReferencedSecretNames`.
4. Existing output stays byte-identical when the field is unset — a unit test with the field
   unset, and [render_compat_test.go](../../internal/controller/render_compat_test.go) as the
   pattern when the field changes what existing objects render.
5. CEL rules get envtest cases in `test/integration/`: the accepted and the rejected objects.
6. The README's fully populated example of the kind, with `# default` or `# example` on the value
   and a security note where the field has one.

## An RBAC permission

1. The marker in [rbac.go](../../internal/controller/rbac.go).
2. The same rule in the chart's [clusterrole.yaml](../../deploy/helm/dex-operator/templates/clusterrole.yaml).
3. `make generate-all` for `config/rbac/role.yaml`.
4. [docs/security/privilege-footprint.md](../security/privilege-footprint.md): the table row and
   what the permission means if the ServiceAccount is compromised.
5. `make e2e-local` — envtest runs no RBAC; only the e2e suite runs under the chart's ClusterRole.

## A CI job

The job in [release.yml](../../.github/workflows/release.yml), its name in the `needs:` list of
`semantic-release`, and a row in [ci-and-release.md](ci-and-release.md).

# Runtime behaviour

When a change reaches Dex, and what the operator does not do for you. The fields named here are
in the README's reference; the security consequences are in
[docs/security/rotation.md](../security/rotation.md) and
[docs/security/secret-flow.md](../security/secret-flow.md).

## What triggers a render

The operator renders an installation again — and rewrites its config and env Secrets if the
result differs — when:

- the `DexInstallation` changes;
- a connector or `DexStaticClient` that references it is created, changed or deleted;
- a Secret that one of those children references is created or changed.

A **deleted** Secret triggers nothing: during a rotation by delete and recreate, the new Secret's
creation triggers the render. Nothing renders on a timer
([ADR 0003](../adr/0003-the-render-is-deterministic-and-change-driven-no-periodic-requeue.md)).

Both Secrets are written into the installation's namespace under `configSecretName` and
`envSecretName`, labelled `app.kubernetes.io/managed-by: dex-operator` and
`dex.gtrfc.com/installation: <name>`.

## When Dex restarts

With `rolloutRestart.enabled: true` and `rolloutRestart.deploymentName` naming the Dex Deployment
in the installation's namespace, the operator restarts Dex — by setting the pod-template
annotation `kubectl.kubernetes.io/restartedAt` — when the rendered **config** changes in content.
A reformatted but equal config does not count. Without `rolloutRestart`, the operator never
restarts Dex: every change waits for a restart you do.

## A rotation needs a restart

A change that touches only the env Secret — a rotated client secret, a rotated connector secret,
a rotated storage password — does **not** restart Dex, even with `rolloutRestart` enabled: the
config stays byte-identical. Dex reads its environment when its pods start, so until they restart
Dex keeps the old value — and keeps accepting an old client secret
([docs/security/rotation.md](../security/rotation.md#h-17)). After a rotation that must take
effect now:

```bash
kubectl -n dex rollout restart deployment/dex     # example namespace and Deployment name
```

## File material is yours to mount

Connector certificates, the Google service-account JSON and storage TLS material are rendered as
file paths into `config.yaml`, but the operator mounts nothing
([docs/security/secret-flow.md](../security/secret-flow.md#h-13)). The paths are fixed:

| Material | Path in the Dex pod |
|---|---|
| a connector certificate or key (LDAP client cert/key, SAML `ca`, a connector's root CA) | `/etc/dex/certs/<connector-id>-<field>.pem` |
| a Google connector's service account | `/etc/dex/secrets/<connector-id>-service-account.json` |
| storage TLS (postgres, etcd, mysql) | `/etc/dex/certs/<storage>-ca.pem`, `<storage>-client-cert.pem`, `<storage>-client-key.pem` |

Add the volumes and mounts in the Dex chart's values (its `volumes` and `volumeMounts` — keys of
the Dex chart, not verified in this repository). A pod mounts only Secrets of its own namespace:
material referenced by a connector in another namespace has to exist as a Secret in the Dex
namespace too. A missing file does not show on any `Ready` condition; Dex fails at start or when
the connector is used, so check Dex's log after adding a file-based connector. The LDAP root CA
(`rootCARef`) is the exception: it is inlined and needs no mount.

## Reading the conditions

- **`DexInstallation` `Ready`** — `True` when the last render was built and both Secrets were
  written, even if some children were left out of it. `False` carries the error that stopped the
  render: a storage credential that cannot be resolved, or the connector guard — connectors were
  collected, none of them renders and no `DexLocalConnector` keeps Dex startable, so the last
  written config stays; the message names every connector and why it was left out.
  `status.connectorCount` and `status.staticClientCount` count what was rendered.
- **`DexInstallation` `ChildrenRejected`** — `True` while `status.rejectedChildren` lists children
  that pass the allowlists but are not in the config: `DuplicateID` when their client or connector
  ID is claimed by more than one child (or is `local` while a `DexLocalConnector` renders),
  `BuildFailed` when their own build failed — a missing Secret or key, an empty client ID, both env
  var keys taken. Find every claimant of a contested ID by its `id` in that list. Which claimant
  wins is the README's `installationRef` rule.
- **`DexInstallation` `TrustedPeersDropped`** — `True` while `status.droppedTrustedPeers` lists
  `trustedPeers` entries left out because no static client in the trusting client's namespace
  holds the peer ID.
- **A child's `Ready`** — `False` when its installation does not exist, does not admit its
  namespace, or left it out of the rendered config. The messages name the reason:
  `referenced DexInstallation <ns>/<name> not found`,
  `namespace "<ns>" is not in DexInstallation <ns>/<name> allowedNamespaces` for a static client,
  `... allowedConnectorNamespaces` for a connector, with `(omitted: only "<ns>" is allowed)` when
  that field is not set; with reason `DuplicateID`
  `client ID "<id>" is claimed by more than one DexStaticClient of DexInstallation <ns>/<name>`
  (for a connector `connector ID "<id>" is claimed by more than one connector ...`); with reason
  `BuildFailed` the child's own build error. Such a child is checked again every five minutes and
  whenever the installation's spec, its rejected children or its dropped peers change.
- **A static client's `TrustedPeersDropped`** — `True` with the peer IDs of its own
  `trustedPeers` that were left out; the client itself renders and stays `Ready=True`.
- **A static client's `status.clientID`** (print column `Client ID`) — the ID last resolved for
  it, written by the installation reconciler. While the client's Secret cannot be resolved it keeps
  claiming that ID, so the gap frees nothing for another namespace.

The operator records no Kubernetes events; the conditions, the installation's status lists and the
operator's log are the record.

## Secrets left behind

The operator never deletes a Secret. After changing `configSecretName` or `envSecretName`, the old
Secrets stay — the old env Secret with every credential it held
([docs/security/privilege-footprint.md](../security/privilege-footprint.md#h-16)). Delete them by
hand:

```bash
kubectl -n dex get secrets -l dex.gtrfc.com/installation=main   # example namespace and installation
```

Deleting a `DexInstallation` leaves its Secrets as well; they carry no owner reference.

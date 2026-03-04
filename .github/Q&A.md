# Dex Operator — Q&A / Design-Entscheidungen

## Q1: API Group & Version
**Frage:** Welche API-Gruppe und Version sollen die CRDs verwenden?  
Beispiel: `dex.guidedtraffic.com/v1alpha1` oder `dex.example.com/v1alpha1`?  
**Antwort:** `dex.gtrfc.com/v1`

## Q2: CRD-Granularität für Connectors
**Frage:** Soll jeder Connector-Typ eine eigene CRD bekommen (z.B. `DexLDAPConnector`, `DexGitHubConnector`, `DexOIDCConnector`, …) oder soll es eine einzelne `DexConnector` CRD mit einem `type`-Feld und typ-spezifischer Config geben?  
- **Option A:** Eine CRD pro Connector-Typ → stärkere Typisierung, bessere Validierung per OpenAPI-Schema, aber viele CRDs  
- **Option B:** Eine generische `DexConnector` CRD → weniger CRDs, aber schwächere Validierung  
**Antwort:** Option A — Eine eigene CRD pro Connector-Typ für maximale Typisierung und Validierung.

## Q3: Scope der DexInstallation CRD
**Frage:** Soll `DexInstallation` namespace-scoped oder cluster-scoped sein? Dex läuft typischerweise in einem bestimmten Namespace (z.B. `dex-system`).  
**Antwort:** Namespace-scoped. Die `DexInstallation` lebt im Namespace der Dex-Instanz. Clients und Connectors in anderen Namespaces referenzieren die Installation per Name + Namespace. Die `DexInstallation` CRD enthält eine Namespace-Whitelist (`allowedNamespaces`), die festlegt, welche Namespaces Clients/Connectors für diese Instanz erstellen dürfen. Eine Wildcard `"*"` erlaubt alle Namespaces.

## Q4: Cross-Namespace Discovery
**Frage:** Static Clients und Connectors können in anderen Namespaces leben (z.B. Grafana im `monitoring` Namespace). Wie soll der Operator diese finden?  
- **Option A:** Jede Connector/Client-Ressource referenziert die DexInstallation per Name/Namespace  
- **Option B:** DexInstallation definiert, welche Namespaces beobachtet werden (oder alle)  
- **Option C:** Label-basierte Selektion  
**Antwort:** Beantwortet durch Q3 — Kombination aus A und B: Jeder Client/Connector referenziert die DexInstallation per Name+Namespace. Die DexInstallation hat eine `allowedNamespaces`-Whitelist (mit `"*"` für alle), die als Autorisierung dient. Der Operator watched alle Namespaces und validiert bei Reconciliation, ob der Namespace des Clients/Connectors in der Whitelist steht.

## Q5: Client-Secret-Generierung
**Frage:** Soll der Operator Client-Secrets automatisch generieren, oder muss der User sie selbst bereitstellen?  
Wenn auto-generiert: Sollen sie rotierbar sein?  
**Antwort:** Der User erstellt das Secret selbst und referenziert es in der Static-Client CR per `secretRef` (Secret im gleichen Namespace) mit Angabe der Keys für `client-id` und `client-secret`. Der Operator liest die Werte daraus und übernimmt sie in die Dex-Konfiguration. Andere Konfigurationswerte (redirectURIs, allowedScopes, trustedPeers etc.) stehen direkt in der CR. Der Operator generiert keine Secrets automatisch.

## Q6: Mehrere Dex-Instanzen
**Frage:** Soll der Operator mehrere `DexInstallation` Ressourcen im selben Cluster unterstützen?  
**Antwort:** Ja, mehrere DexInstallation-Ressourcen in verschiedenen Namespaces werden unterstützt.

## Q7: Umfang der DexInstallation Config
**Frage:** Welche Dex-Konfiguration gehört in die `DexInstallation` CRD? Nur der Verweis auf das Target-Secret, oder auch:  
- Issuer URL  
- Storage Backend (kubernetes, postgres, sqlite, etc.)  
- Token/Key Expiry  
- Web-Konfiguration (HTTP/HTTPS, TLS)  
- CORS  
- gRPC API Konfiguration  
- Logger Konfiguration  
**Antwort:** Die komplette Dex-Konfiguration wird in der `DexInstallation` CRD abgebildet — Issuer, Storage, Expiry, Web, CORS, gRPC, Logger, etc. Die CRD ist die vollständige deklarative Repräsentation der globalen Dex-Config.

## Q8: Output-Secret für Clients
**Frage:** Welche Felder soll das Secret enthalten, das in den Namespace des Clients gelegt wird?  
Vorschlag: `client-id`, `client-secret`, `auth-url` (authorize endpoint), `token-url`, `api-url` (userinfo), `issuer-url`  
Weitere Felder?  
**Antwort:** Kein Output-Secret. Der User bringt sein eigenes Secret mit (enthält client-id und client-secret). Der Operator liest daraus und überträgt die Werte in die Dex-Konfiguration. Endpoints wie auth-url, token-url etc. sind über die Issuer-URL ableitbar und müssen vom User selbst konfiguriert werden.

## Q9: Static Passwords
**Frage:** Soll der Operator auch Static Passwords verwalten (z.B. für lokale Dev/Test-User)?  
**Antwort:** Nein, Static Passwords werden nicht unterstützt.

## Q10: Dex-Version / Kompatibilität
**Frage:** Welche Dex-Version(en) sollen unterstützt werden? Soll der Operator mit einem bestimmten Helm-Chart-Release getestet werden?  
**Antwort:** Immer die aktuellste Version von Dex und dem Dex Helm Chart als Referenz. Keine explizite Rückwärtskompatibilität.

## Q11: Helm-Chart-Integration
**Frage:** Der Operator verwaltet die zwei Secrets (global config + env secrets). Das Dex Helm Chart muss diese Secrets referenzieren. Soll der Operator:  
- **Option A:** Nur die Secrets erzeugen und der User konfiguriert das Helm Chart manuell darauf  
- **Option B:** Die Secret-Namen in der `DexInstallation` CRD konfigurierbar machen  
- **Option C:** Ein festes Naming-Schema verwenden (z.B. `<installation-name>-config`, `<installation-name>-env`)  
**Antwort:** Option B — Die Secret-Namen werden in der `DexInstallation` CR konfiguriert (Pflichtfelder). Der Operator erstellt die Secrets im Namespace der DexInstallation. Der User gibt dann dieselben Namen im Dex Helm Chart an.

## Q12: Reconciliation & Restart
**Frage:** Wenn sich die Dex-Konfiguration ändert (neuer Client, neuer Connector), muss Dex neu gestartet werden, da es die Config nur beim Start liest. Soll der Operator:  
- **Option A:** Automatisch einen Rollout-Restart des Dex-Deployments auslösen  
- **Option B:** Nur die Secrets aktualisieren und den Restart dem User/Helm überlassen  
- **Option C:** Eine Annotation auf dem Deployment bumpen (z.B. `checksum/config`), sodass der Rollout automatisch passiert  
**Antwort:** Standard ist Option B — nur die Secrets werden aktualisiert. In der `DexInstallation` CR gibt es aber ein optionales Feld, um automatischen Restart zu aktivieren. Wenn aktiviert, triggert der Operator einen Rollout-Restart des Dex-Deployments bei Config-Änderungen. Der Deployment-Name muss dann ebenfalls in der CR konfiguriert werden (z.B. `deploymentName: dex`).

## Q13: Repository & Container Registry
**Frage:** Unter welcher GitHub-Organisation und welcher Container-Registry soll der Operator veröffentlicht werden?  
(Aktuell steht noch `guidedtraffic/valkey-operator` in der Makefile/package.json)  
**Antwort:** `guidedtraffic/dex-operator`

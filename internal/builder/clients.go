/*
Copyright 2025 Guided Traffic GmbH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package builder

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/types"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// clientUnits returns one render unit per static client in input order.
// Each unit claims its client ID: spec.clientID for a secretless client, the
// client-id from the referenced Secret for a confidential one.  When the
// Secret cannot be resolved, the client claims status.clientID, the ID last
// resolved for it, so a gap in its Secret frees nothing.
func clientUnits(ctx context.Context, clients []dexv1.DexStaticClient, sr SecretResolver) []*renderUnit {
	units := make([]*renderUnit, 0, len(clients))
	for i := range clients {
		c := &clients[i]
		u := &renderUnit{
			kind:      kindStaticClient,
			namespace: c.Namespace,
			name:      c.Name,
			created:   c.CreationTimestamp,
		}
		u.id, u.claimErr = resolveClientID(ctx, c, sr)
		u.resolved = u.claimErr == nil
		if !u.resolved {
			u.id = c.Status.ClientID
		}
		u.build = func(ctx context.Context, env *childEnv) (unitResult, error) {
			sc, err := buildOneStaticClient(ctx, c, u.id, sr, env)
			return unitResult{client: sc}, err
		}
		units = append(units, u)
	}
	return units
}

// resolveClientID returns the client's ID from its spec or its Secret.
func resolveClientID(ctx context.Context, c *dexv1.DexStaticClient, sr SecretResolver) (string, error) {
	// Secretless public client: the id is set inline.  CRD-level CEL
	// validation guarantees that clientID is set whenever secretRef is
	// absent.
	if c.Spec.SecretRef == nil {
		if c.Spec.ClientID == "" {
			return "", errors.New("clientID is empty")
		}
		return c.Spec.ClientID, nil
	}

	clientID, err := resolveSecret(ctx, c.Namespace, dexv1.SecretKeyRef{
		Name: c.Spec.SecretRef.Name,
		Key:  secretRefKey(c.Spec.SecretRef.ClientIDKey, "client-id"),
	}, sr)
	if err != nil {
		return "", fmt.Errorf("client-id: %w", err)
	}
	if clientID == "" {
		return "", fmt.Errorf("client-id: key %q in secret %s/%s is empty",
			secretRefKey(c.Spec.SecretRef.ClientIDKey, "client-id"), c.Namespace, c.Spec.SecretRef.Name)
	}
	return clientID, nil
}

// secretRefKey returns key, or def when key is empty.  The defaults are also
// set by kubebuilder defaults on the CRD.
func secretRefKey(key, def string) string {
	if key == "" {
		return def
	}
	return key
}

// buildOneStaticClient converts one DexStaticClient with the resolved
// clientID into a Dex StaticClient config entry.  For clients with a
// secretRef the client-secret is stored in env and referenced via the
// secretEnv field (bare env var name, no $-prefix).  Secretless public
// clients contribute nothing to env.  TrustedPeers are copied unfiltered;
// [filterTrustedPeers] narrows them once every client's ID is known.
func buildOneStaticClient(
	ctx context.Context,
	c *dexv1.DexStaticClient,
	clientID string,
	sr SecretResolver,
	env *childEnv,
) (StaticClient, error) {
	sc := StaticClient{
		ID:           clientID,
		Name:         c.Spec.DisplayName,
		RedirectURIs: c.Spec.RedirectURIs,
		Public:       c.Spec.Public,
	}
	if len(c.Spec.TrustedPeers) > 0 {
		sc.TrustedPeers = c.Spec.TrustedPeers
	}

	if c.Spec.SecretRef == nil {
		return sc, nil
	}

	// Resolve the client-secret into the env Secret; reference it via
	// the secretEnv field (bare env var name, no $-prefix).
	csVal, err := resolveSecret(ctx, c.Namespace, dexv1.SecretKeyRef{
		Name: c.Spec.SecretRef.Name,
		Key:  secretRefKey(c.Spec.SecretRef.ClientSecretKey, "client-secret"),
	}, sr)
	if err != nil {
		return StaticClient{}, fmt.Errorf("client-secret: %w", err)
	}
	sc.SecretEnv, err = env.set(c.Name, "CLIENT_SECRET", csVal)
	if err != nil {
		return StaticClient{}, err
	}

	return sc, nil
}

// filterTrustedPeers narrows the trustedPeers of every rendered client to
// the IDs held by a static client in that client's own namespace, and
// returns the entries it left out, sorted by namespace, name and peer ID.
//
// An ID is held by the client the contest rule leaves it to (see [contest]),
// whether that client renders or failed to build, so a gap in a peer's
// Secret does not rewrite the trusting client's entry.  Holders and the
// filter come from the same render.
func filterTrustedPeers(units []*renderUnit) []dexv1.DroppedTrustedPeer {
	holders := make(map[string]string, len(units)) // client ID → namespace
	for _, u := range units {
		if u.wins {
			holders[u.id] = u.namespace
		}
	}

	var dropped []dexv1.DroppedTrustedPeer
	for _, u := range units {
		if !u.rendered || len(u.result.client.TrustedPeers) == 0 {
			continue
		}
		var kept []string
		for _, peer := range u.result.client.TrustedPeers {
			if ns, held := holders[peer]; held && ns == u.namespace {
				kept = append(kept, peer)
				continue
			}
			dropped = append(dropped, dexv1.DroppedTrustedPeer{Namespace: u.namespace, Name: u.name, PeerID: peer})
		}
		u.result.client.TrustedPeers = kept
	}

	slices.SortFunc(dropped, func(a, b dexv1.DroppedTrustedPeer) int {
		return cmp.Or(
			cmp.Compare(a.Namespace, b.Namespace),
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.PeerID, b.PeerID),
		)
	})
	return slices.Compact(dropped)
}

// renderedClients returns the config entries of the rendered clients and
// their source objects, both in input order.
func renderedClients(units []*renderUnit, src []dexv1.DexStaticClient) ([]StaticClient, []dexv1.DexStaticClient) {
	entries := make([]StaticClient, 0, len(units))
	var objs []dexv1.DexStaticClient
	for i, u := range units {
		if !u.rendered {
			continue
		}
		entries = append(entries, u.result.client)
		objs = append(objs, src[i])
	}
	return entries, objs
}

// resolvedClientIDs returns the ID resolved for every client whose ID came
// from its spec or its Secret, rendered or not.
func resolvedClientIDs(units []*renderUnit) map[types.NamespacedName]string {
	ids := make(map[types.NamespacedName]string, len(units))
	for _, u := range units {
		if u.resolved {
			ids[types.NamespacedName{Namespace: u.namespace, Name: u.name}] = u.id
		}
	}
	return ids
}

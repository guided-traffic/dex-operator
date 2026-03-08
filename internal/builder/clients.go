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
	"context"
	"fmt"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// buildStaticClients converts each DexStaticClient into a Dex StaticClient
// config entry.  The client-id is resolved from the referenced Secret and
// embedded directly; the client-secret is stored in envs and referenced via
// the secretEnv field (bare env var name, no $-prefix).
func buildStaticClients(
	ctx context.Context,
	clients []dexv1.DexStaticClient,
	sr SecretResolver,
	envs map[string][]byte,
) ([]StaticClient, error) {
	usedEnvKeys := make(map[string]string, len(clients)) // envKey → resource name
	result := make([]StaticClient, 0, len(clients))
	for i := range clients {
		sc, err := buildOneStaticClient(ctx, &clients[i], sr, envs, usedEnvKeys)
		if err != nil {
			return nil, fmt.Errorf("static client %q: %w", clients[i].Name, err)
		}
		result = append(result, sc)
	}
	return result, nil
}

func buildOneStaticClient(
	ctx context.Context,
	c *dexv1.DexStaticClient,
	sr SecretResolver,
	envs map[string][]byte,
	usedEnvKeys map[string]string,
) (StaticClient, error) {
	// Default key names defined by kubebuilder defaults on the CRD.
	clientIDKey := c.Spec.SecretRef.ClientIDKey
	if clientIDKey == "" {
		clientIDKey = "client-id"
	}
	clientSecretKey := c.Spec.SecretRef.ClientSecretKey
	if clientSecretKey == "" {
		clientSecretKey = "client-secret"
	}

	// Resolve the actual client-id value (embedded inline in the config).
	clientID, err := resolveSecret(ctx, c.Namespace, dexv1.SecretKeyRef{
		Name: c.Spec.SecretRef.Name,
		Key:  clientIDKey,
	}, sr)
	if err != nil {
		return StaticClient{}, fmt.Errorf("client-id: %w", err)
	}

	// Build the env var key and check for collisions.
	csEnvKey := clientEnvKey(c.Name, "CLIENT_SECRET")
	resourceFQN := c.Namespace + "/" + c.Name
	if prev, ok := usedEnvKeys[csEnvKey]; ok {
		return StaticClient{}, fmt.Errorf(
			"env var name collision: %q is already used by %q (current: %q)",
			csEnvKey, prev, resourceFQN,
		)
	}
	usedEnvKeys[csEnvKey] = resourceFQN

	// Resolve the client-secret into the env Secret; reference it via
	// the secretEnv field (bare env var name, no $-prefix).
	csVal, err := resolveSecret(ctx, c.Namespace, dexv1.SecretKeyRef{
		Name: c.Spec.SecretRef.Name,
		Key:  clientSecretKey,
	}, sr)
	if err != nil {
		return StaticClient{}, fmt.Errorf("client-secret: %w", err)
	}
	envs[csEnvKey] = []byte(csVal)

	sc := StaticClient{
		ID:           clientID,
		SecretEnv:    csEnvKey,
		Name:         c.Spec.DisplayName,
		RedirectURIs: c.Spec.RedirectURIs,
		Public:       c.Spec.Public,
	}
	if len(c.Spec.AllowedScopes) > 0 {
		sc.AllowedScopes = c.Spec.AllowedScopes
	}
	if len(c.Spec.TrustedPeers) > 0 {
		sc.TrustedPeers = c.Spec.TrustedPeers
	}

	return sc, nil
}

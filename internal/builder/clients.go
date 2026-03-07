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
// a $ENV_VAR substitution string.
func buildStaticClients(
	ctx context.Context,
	clients []dexv1.DexStaticClient,
	sr SecretResolver,
	envs map[string][]byte,
) ([]StaticClient, error) {
	result := make([]StaticClient, 0, len(clients))
	for i := range clients {
		sc, err := buildOneStaticClient(ctx, &clients[i], sr, envs)
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

	// Resolve the client-secret into the env Secret so the container can pick
	// it up via envFrom; reference it with $VAR in the config YAML.
	csEnvKey := clientEnvKey(c.Name, "CLIENT_SECRET")
	csRef, err := resolveEnvSecret(ctx, c.Namespace, dexv1.SecretKeyRef{
		Name: c.Spec.SecretRef.Name,
		Key:  clientSecretKey,
	}, csEnvKey, sr, envs)
	if err != nil {
		return StaticClient{}, fmt.Errorf("client-secret: %w", err)
	}

	displayName := c.Spec.DisplayName

	sc := StaticClient{
		ID:           clientID,
		Secret:       csRef,
		Name:         displayName,
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

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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

// sanitizeEnvKey converts any string into a valid, uppercase environment
// variable name by replacing non-alphanumeric runes with underscores.
func sanitizeEnvKey(s string) string {
	upper := strings.ToUpper(s)
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, upper)
}

// Env var key bases.  A credential's plain key is <BASE>_<FIELD>:
//   - connector: <CONNECTOR_TYPE>_<CONNECTOR_ID>_<FIELD>
//   - static client: <RESOURCE_NAME>_<FIELD>
//   - storage: STORAGE_<FIELD>
const storageEnvBase = "STORAGE"

// connectorEnvBase returns the env var key base of a connector credential.
func connectorEnvBase(connType, connID string) string {
	return connType + "_" + connID
}

// childEnv assigns the env var keys of one child of a render and collects
// their values.
//
// Keys are assigned in the order in which children are built (see
// [byEnvPriority]): a child whose plain key <BASE>_<FIELD> is already taken
// by an earlier child gets its fallback key <BASE>_<HASH>_<FIELD>, the hash
// identifying the child.  The keys of a child become visible to later
// children only on [childEnv.commit], so a child that fails to build holds
// no key.
type childEnv struct {
	taken   map[string][]byte // keys committed by earlier children; nil for a dry run
	hash    string
	pending map[string][]byte
}

// newChildEnv returns the env of the child kind/namespace/name.  taken is
// the env of the render, which [childEnv.commit] extends; a nil taken makes
// a dry run in which every plain key is free and nothing is committed.
func newChildEnv(taken map[string][]byte, kind, namespace, name string) *childEnv {
	return &childEnv{
		taken:   taken,
		hash:    childHash(kind, namespace, name),
		pending: make(map[string][]byte),
	}
}

// childHash returns 8 hex characters of SHA-256 over kind/namespace/name.
func childHash(kind, namespace, name string) string {
	sum := sha256.Sum256([]byte(kind + "/" + namespace + "/" + name))
	return hex.EncodeToString(sum[:4])
}

// set stores value under the child's key for base and field and returns the
// key: the plain key <BASE>_<FIELD> when it is free, else the fallback key
// <BASE>_<HASH>_<FIELD>.  It fails when both are taken.  The message names
// no other child, because it ends up in this child's status.
func (e *childEnv) set(base, field, value string) (string, error) {
	plain := sanitizeEnvKey(base + "_" + field)
	if e.free(plain) {
		e.pending[plain] = []byte(value)
		return plain, nil
	}
	fallback := sanitizeEnvKey(base + "_" + e.hash + "_" + field)
	if e.free(fallback) {
		e.pending[fallback] = []byte(value)
		return fallback, nil
	}
	return "", fmt.Errorf("env var names %q and %q are both taken by other children of the installation", plain, fallback)
}

func (e *childEnv) free(key string) bool {
	if _, ok := e.pending[key]; ok {
		return false
	}
	_, ok := e.taken[key]
	return !ok
}

// commit makes the child's keys part of the render's env.
func (e *childEnv) commit() {
	for k, v := range e.pending {
		e.taken[k] = v
	}
}

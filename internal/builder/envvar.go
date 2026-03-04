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

// connectorEnvKey returns a deterministic env var name for a connector
// credential.  Format: <CONNECTOR_TYPE>_<CONNECTOR_ID>_<FIELD>
func connectorEnvKey(connType, connID, field string) string {
	return sanitizeEnvKey(connType + "_" + connID + "_" + field)
}

// clientEnvKey returns a deterministic env var name for a static client
// credential.  Format: <RESOURCE_NAME>_<FIELD>
func clientEnvKey(resourceName, field string) string {
	return sanitizeEnvKey(resourceName + "_" + field)
}

// storageEnvKey returns a deterministic env var name for a storage
// credential.  Format: STORAGE_<FIELD>
func storageEnvKey(field string) string {
	return sanitizeEnvKey("STORAGE_" + field)
}

// envRef stores a resolved value in envs and returns the "$KEY" reference.
func envRef(key string, value string, envs map[string][]byte) string {
	envs[key] = []byte(value)
	return "$" + key
}

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

// Package builder assembles the Dex configuration YAML and the companion
// environment-variable Secret from a DexInstallation, its Connectors and its
// StaticClients.
//
// Call [Build] with an [Input] that holds a populated [ConnectorSet] and a
// [SecretResolver] to obtain the ready-to-write [Output].
package builder

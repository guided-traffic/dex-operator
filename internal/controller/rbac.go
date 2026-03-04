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

// Package controller contains additional RBAC markers for the child-resource
// controllers.  Placing them in a dedicated file keeps the controller files
// focused on reconciliation logic.
package controller

// Connector and static-client child controllers need read+status-update access.
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexinstallations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexinstallations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexinstallations/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexldapconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexldapconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexgithubconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexgithubconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexsamlconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexsamlconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexgitlabconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexgitlabconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexoidcconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexoidcconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexoauth2connectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexoauth2connectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexgoogleconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexgoogleconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexlinkedinconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexlinkedinconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexmicrosoftconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexmicrosoftconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexauthproxyconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexauthproxyconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexbitbucketconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexbitbucketconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexlocalconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexlocalconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexopenshiftconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexopenshiftconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexatlassiancrowdconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexatlassiancrowdconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexgiteaconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexgiteaconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexkeystoneconnectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexkeystoneconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexstaticclients,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=dex.gtrfc.com,resources=dexstaticclients/status,verbs=get;update;patch

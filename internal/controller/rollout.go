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

package controller

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

const restartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"

// triggerRolloutRestart patches the template annotation of the Dex Deployment
// to force a rolling restart.  It is a no-op when rollout restart is not
// configured or not enabled.
func triggerRolloutRestart(
	ctx context.Context,
	c client.Client,
	installation *dexv1.DexInstallation,
) error {
	if !rolloutEnabled(installation) {
		return nil
	}

	var deploy appsv1.Deployment
	key := types.NamespacedName{
		Name:      installation.Spec.RolloutRestart.DeploymentName,
		Namespace: installation.Namespace,
	}
	if err := c.Get(ctx, key, &deploy); err != nil {
		return err
	}

	patch := client.MergeFrom(deploy.DeepCopy())
	if deploy.Spec.Template.Annotations == nil {
		deploy.Spec.Template.Annotations = make(map[string]string)
	}
	deploy.Spec.Template.Annotations[restartedAtAnnotation] = time.Now().UTC().Format(time.RFC3339)
	return c.Patch(ctx, &deploy, patch)
}

func rolloutEnabled(inst *dexv1.DexInstallation) bool {
	return inst.Spec.RolloutRestart != nil &&
		inst.Spec.RolloutRestart.Enabled &&
		inst.Spec.RolloutRestart.DeploymentName != ""
}

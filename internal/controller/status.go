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
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	intbuilder "github.com/guided-traffic/dex-operator/internal/builder"
)

// setReadyCondition sets the Ready condition on a CommonStatus.
// If err is nil the condition is set to True; otherwise it is set to False
// with err.Error() as the message, and with the reason of a [configError]
// that carries one.
func setReadyCondition(status *dexv1.CommonStatus, generation int64, err error) {
	cond := metav1.Condition{
		Type:               dexv1.ConditionTypeReady,
		ObservedGeneration: generation,
	}

	var ce *configError
	switch {
	case err == nil:
		cond.Status = metav1.ConditionTrue
		cond.Reason = "Reconciled"
		cond.Message = ""
	case errors.As(err, &ce) && ce.reason != "":
		cond.Status = metav1.ConditionFalse
		cond.Reason = ce.reason
		cond.Message = err.Error()
	default:
		cond.Status = metav1.ConditionFalse
		cond.Reason = "ReconcileError"
		cond.Message = err.Error()
	}

	setOrReplaceCondition(status, cond)
	status.ObservedGeneration = generation
}

// setOrReplaceCondition upserts a condition into the conditions slice.
func setOrReplaceCondition(status *dexv1.CommonStatus, cond metav1.Condition) {
	now := metav1.Now()
	for i, c := range status.Conditions {
		if c.Type != cond.Type {
			continue
		}
		if c.Status != cond.Status {
			cond.LastTransitionTime = now
		} else {
			cond.LastTransitionTime = c.LastTransitionTime
		}
		status.Conditions[i] = cond
		return
	}
	cond.LastTransitionTime = now
	status.Conditions = append(status.Conditions, cond)
}

// setChildReport records which children of the installation did not make it
// into the rendered config, and which trustedPeers entries were left out,
// together with the ChildrenRejected and TrustedPeersDropped conditions.
func setChildReport(status *dexv1.DexInstallationStatus, generation int64, out intbuilder.Output) {
	status.RejectedChildren = out.Rejected
	status.DroppedTrustedPeers = out.DroppedTrustedPeers

	setOrReplaceCondition(&status.CommonStatus, countCondition(
		dexv1.ConditionTypeChildrenRejected, generation, len(out.Rejected),
		"rejected children: %d, see status.rejectedChildren", "NoneRejected"))
	setOrReplaceCondition(&status.CommonStatus, countCondition(
		dexv1.ConditionTypeTrustedPeersDropped, generation, len(out.DroppedTrustedPeers),
		"dropped trustedPeers entries: %d, see status.droppedTrustedPeers", "NoneDropped"))
}

// countCondition returns condType, True with format filled in by n while n
// is positive, otherwise False with noneReason.
func countCondition(condType string, generation int64, n int, format, noneReason string) metav1.Condition {
	cond := metav1.Condition{
		Type:               condType,
		ObservedGeneration: generation,
		Status:             metav1.ConditionFalse,
		Reason:             noneReason,
	}
	if n > 0 {
		cond.Status = metav1.ConditionTrue
		cond.Reason = condType
		cond.Message = fmt.Sprintf(format, n)
	}
	return cond
}

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// setReadyCondition sets the Ready condition on a CommonStatus.
// If err is nil the condition is set to True; otherwise it is set to False
// with err.Error() as the message.
func setReadyCondition(status *dexv1.CommonStatus, generation int64, err error) {
	cond := metav1.Condition{
		Type:               dexv1.ConditionTypeReady,
		ObservedGeneration: generation,
	}

	if err == nil {
		cond.Status = metav1.ConditionTrue
		cond.Reason = "Reconciled"
		cond.Message = ""
	} else {
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

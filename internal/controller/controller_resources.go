/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"

	netv1 "github.com/gradiant/open5gs-operator/api/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func generateConfigMapHash(cm *corev1.ConfigMap) (string, error) {
	data, err := json.Marshal(cm.Data)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func setOwnerReference(open5gs *netv1.Open5GS, obj client.Object, scheme *runtime.Scheme) error {
	return ctrl.SetControllerReference(open5gs, obj, scheme)
}

func hasOwnerReference(obj client.Object, owner *netv1.Open5GS) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.UID == owner.UID {
			return true
		}
	}
	return false
}

func configMapEqual(cm1, cm2 *corev1.ConfigMap) bool {
	return reflect.DeepEqual(cm1.Data, cm2.Data) &&
		reflect.DeepEqual(cm1.Labels, cm2.Labels) &&
		reflect.DeepEqual(cm1.Annotations, cm2.Annotations)
}

func deploymentEqual(d1, d2 *appsv1.Deployment) bool {
	if d1 == nil || d2 == nil {
		return false
	}

	if !reflect.DeepEqual(d1.Labels, d2.Labels) {
		return false
	}

	if d1.Spec.Template.Spec.ServiceAccountName != d2.Spec.Template.Spec.ServiceAccountName {
		return false
	}

	if len(d1.Spec.Template.Spec.Containers) != len(d2.Spec.Template.Spec.Containers) {
		return false
	}

	for i := range d1.Spec.Template.Spec.Containers {
		if d1.Spec.Template.Spec.Containers[i].Image != d2.Spec.Template.Spec.Containers[i].Image {
			return false
		}
	}

	return true
}

func pvcEqual(pvc1, pvc2 *corev1.PersistentVolumeClaim) bool {
	return pvc1.Spec.Resources.Requests[corev1.ResourceStorage] == pvc2.Spec.Resources.Requests[corev1.ResourceStorage] &&
		pvc1.Spec.AccessModes[0] == pvc2.Spec.AccessModes[0]
}

func serviceEqual(s1, s2 *corev1.Service) bool {
	return reflect.DeepEqual(s1.Spec.Selector, s2.Spec.Selector) &&
		reflect.DeepEqual(s1.Labels, s2.Labels) &&
		reflect.DeepEqual(s1.Annotations, s2.Annotations) &&
		s1.Spec.Type == s2.Spec.Type
}

func serviceMonitorEqual(sm1, sm2 *monitoringv1.ServiceMonitor) bool {
	return reflect.DeepEqual(sm1.Spec.Selector, sm2.Spec.Selector) &&
		reflect.DeepEqual(sm1.Labels, sm2.Labels) &&
		reflect.DeepEqual(sm1.Annotations, sm2.Annotations)
}

func serviceAccountEqual(sa1, sa2 *corev1.ServiceAccount) bool {
	return reflect.DeepEqual(sa1.Labels, sa2.Labels) &&
		reflect.DeepEqual(sa1.Annotations, sa2.Annotations)
}

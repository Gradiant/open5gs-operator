/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"slices"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const deploymentIntentAnnotation = "open5gs/last-applied-deployment"

type managedDeployment struct {
	Metadata metav1.ObjectMeta `json:"metadata"`
	Spec     struct {
		Strategy appsv1.DeploymentStrategy `json:"strategy"`
		Template corev1.PodTemplateSpec    `json:"template"`
	} `json:"spec"`
}

func deploymentIntent(dep *appsv1.Deployment) managedDeployment {
	intent := managedDeployment{Metadata: metav1.ObjectMeta{Labels: dep.Labels}}
	intent.Spec.Strategy = dep.Spec.Strategy
	intent.Spec.Template = dep.Spec.Template
	return intent
}

func recordDeploymentIntent(dep *appsv1.Deployment) error {
	intent, err := json.Marshal(deploymentIntent(dep))
	if err != nil {
		return err
	}
	if dep.Annotations == nil {
		dep.Annotations = map[string]string{}
	}
	dep.Annotations[deploymentIntentAnnotation] = string(intent)
	return nil
}

func applyDeployment(ctx context.Context, c client.Client, live, desired *appsv1.Deployment) error {
	original := []byte(live.Annotations[deploymentIntentAnnotation])
	if len(original) == 0 {
		// Older operators did not record ownership. Take over the workload
		// fields they generated and leave added containers and volumes alone.
		old := deploymentIntent(live.DeepCopy())
		pod := &old.Spec.Template.Spec
		want := desired.Spec.Template.Spec
		pod.Containers = slices.DeleteFunc(pod.Containers, func(container corev1.Container) bool {
			return !slices.ContainsFunc(want.Containers, func(c corev1.Container) bool { return c.Name == container.Name })
		})
		pod.InitContainers = slices.DeleteFunc(pod.InitContainers, func(container corev1.Container) bool {
			return !(container.Name == "init" && live.Labels["app.kubernetes.io/name"] == "webui") &&
				!slices.ContainsFunc(want.InitContainers, func(c corev1.Container) bool { return c.Name == container.Name })
		})
		managedVolumes := []string{"config", "entrypoint", "tun-device", "populate", "datadir", "common-scripts"}
		pod.Volumes = slices.DeleteFunc(pod.Volumes, func(volume corev1.Volume) bool {
			return !slices.Contains(managedVolumes, volume.Name) &&
				!slices.ContainsFunc(want.Volumes, func(v corev1.Volume) bool { return v.Name == volume.Name })
		})
		old.Metadata.Labels = desired.Labels
		old.Spec.Template.ObjectMeta = metav1.ObjectMeta{Labels: desired.Spec.Template.Labels}
		podSpec := corev1.PodSpec{
			Containers: pod.Containers, InitContainers: pod.InitContainers, Volumes: pod.Volumes,
			ServiceAccountName: pod.ServiceAccountName, SecurityContext: pod.SecurityContext,
		}
		old.Spec.Template.Spec = podSpec
		var err error
		original, err = json.Marshal(old)
		if err != nil {
			return err
		}
	}

	intent := deploymentIntent(desired)
	intent.Metadata.Annotations = maps.Clone(desired.Annotations)
	modified, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	current, err := json.Marshal(live)
	if err != nil {
		return err
	}
	meta, err := strategicpatch.NewPatchMetaFromStruct(appsv1.Deployment{})
	if err != nil {
		return err
	}
	patch, err := strategicpatch.CreateThreeWayMergePatch(original, modified, current, meta, true)
	if err != nil {
		return err
	}
	// Strategic patches may contain list-order directives even when the
	// live object already has the requested order.
	patched, err := strategicpatch.StrategicMergePatch(current, patch, appsv1.Deployment{})
	if err != nil {
		return err
	}
	if bytes.Equal(current, patched) {
		return nil
	}
	var result appsv1.Deployment
	if err := json.Unmarshal(patched, &result); err != nil {
		return err
	}
	normalized, err := json.Marshal(&result)
	if err != nil {
		return err
	}
	if bytes.Equal(current, normalized) {
		return nil
	}
	return c.Patch(ctx, live, client.RawPatch(types.StrategicMergePatchType, patch))
}

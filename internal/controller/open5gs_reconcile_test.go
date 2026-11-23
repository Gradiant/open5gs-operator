/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */
package controller

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	netv1 "github.com/gradiant/open5gs-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileExistingUPF(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	for _, register := range []func(*runtime.Scheme) error{netv1.AddToScheme, appsv1.AddToScheme, corev1.AddToScheme} {
		if err := register(scheme); err != nil {
			t.Fatal(err)
		}
	}
	owner := &netv1.Open5GS{ObjectMeta: metav1.ObjectMeta{Name: "core", Namespace: "default", UID: "core-uid"}}
	before := CreateUPFDeployment("default", "core", "old", nil, false, "", nil, false)
	if err := setOwnerReference(owner, before, scheme); err != nil {
		t.Fatal(err)
	}
	before.Spec.Template.Annotations = map[string]string{"integration.example/setting": "keep"}
	before.Spec.Template.Spec.Containers = append(before.Spec.Template.Spec.Containers, corev1.Container{Name: "sidecar", Image: "keep"})
	r := &Open5GSReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(before).Build(), Scheme: scheme}
	desired := CreateUPFDeployment("default", "core", "new", nil, false, "", map[string]string{"open5gs.example/test": "first"}, true)
	reconcile := func(desired *appsv1.Deployment) *appsv1.Deployment {
		t.Helper()
		if err := reconcileDeployment(ctx, r, owner, desired, "config", "UPF", logr.Discard()); err != nil {
			t.Fatal(err)
		}
		got := &appsv1.Deployment{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(before), got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	got := reconcile(desired)
	if got.Spec.Selector == nil || got.Spec.Selector.MatchLabels["app.kubernetes.io/name"] != "upf" {
		t.Fatal("the immutable deployment selector was lost")
	}
	pod := got.Spec.Template.Spec
	if pod.Containers[0].Image != "new" || *pod.Containers[0].SecurityContext.Privileged || pod.InitContainers[0].Image != "new" {
		t.Fatal("existing UPF did not apply the desired workload")
	}
	if len(pod.Containers) != 2 || pod.Containers[1].Name != "sidecar" || got.Spec.Template.Annotations["integration.example/setting"] != "keep" {
		t.Fatal("existing integration fields were lost")
	}
	if again := reconcile(desired); again.ResourceVersion != got.ResourceVersion {
		t.Fatal("reconciling unchanged state wrote to the API")
	}
	legacy := CreateUPFDeployment("default", "core", "new", nil, false, "", nil, false)
	got = reconcile(legacy)
	if len(got.Spec.Template.Spec.SecurityContext.Sysctls) != 0 || len(got.Spec.Template.Spec.Containers[0].Command) != 0 {
		t.Fatal("fields from the previous profile were retained")
	}
	if _, ok := got.Spec.Template.Annotations["open5gs.example/test"]; ok {
		t.Fatal("obsolete annotation was retained")
	}
}

func TestMongoDBStorageSelection(t *testing.T) {
	const legacyImage = "bitnamilegacy/mongodb:8.0.13-debian-12-r0"
	core := &netv1.Open5GS{ObjectMeta: metav1.ObjectMeta{Name: "core", Namespace: "default", UID: "core-uid"}}
	scheme := runtime.NewScheme()
	for _, register := range []func(*runtime.Scheme) error{netv1.AddToScheme, appsv1.AddToScheme, corev1.AddToScheme} {
		if err := register(scheme); err != nil {
			t.Fatal(err)
		}
	}
	legacy := CreateMongoDBDeployment(core.Namespace, core.Name, legacyImage, nil, "")
	if err := setOwnerReference(core, legacy, scheme); err != nil {
		t.Fatal(err)
	}
	oldClaim := CreateMongoDBPVC(core.Namespace, core.Name)
	restoredClaim := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "restored", Namespace: core.Namespace}}
	for _, tc := range []struct {
		name, image, claim, want string
		objects                  []client.Object
		wantError                bool
	}{
		{name: "new instance", want: defaultMongoDBImage},
		{name: "keep pinned legacy image", objects: []client.Object{legacy, oldClaim}, want: legacyImage},
		{name: "keep an untagged Bitnami image on the legacy layout", image: "bitnami/mongodb", objects: []client.Object{oldClaim}, want: "bitnami/mongodb"},
		{name: "do not reuse old volume implicitly", image: defaultMongoDBImage, objects: []client.Object{oldClaim}, wantError: true},
		{name: "do not reuse old volume by name", image: defaultMongoDBImage, claim: oldClaim.Name, objects: []client.Object{oldClaim}, wantError: true},
		{name: "reject a missing migration volume", image: defaultMongoDBImage, claim: restoredClaim.Name, objects: []client.Object{oldClaim}, wantError: true},
		{name: "use staged volume", image: defaultMongoDBImage, claim: restoredClaim.Name, objects: []client.Object{legacy, oldClaim, restoredClaim}, want: defaultMongoDBImage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			instance := core.DeepCopy()
			instance.Spec.MongoDBVersion = tc.image
			instance.Spec.MongoDBVolumeClaim = tc.claim
			r := &Open5GSReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.objects...).Build(), Scheme: scheme}
			err := r.selectMongoDBStorage(context.Background(), instance)
			if tc.wantError {
				if err == nil {
					t.Fatal("an unsafe MongoDB cutover was accepted")
				}
				return
			}
			if err != nil || instance.Spec.MongoDBVersion != tc.want {
				t.Fatalf("wanted image %q, got %q (%v)", tc.want, instance.Spec.MongoDBVersion, err)
			}
		})
	}
}

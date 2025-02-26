/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	netv1 "github.com/gradiant/open5gs-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Open5GSUserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=net.gradiant.org,resources=open5gsusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=net.gradiant.org,resources=open5gsusers/status,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=net.gradiant.org,resources=open5gsusers/finalizers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list
//+kubebuilder:rbac:groups=net.gradiant.org,resources=open5gses,verbs=get;list

const (
	Open5GSUserFinalizer = "finalizer.open5gsuser.net.gradiant.org/user"
)

func (r *Open5GSUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	user := &netv1.Open5GSUser{}
	err := r.Get(ctx, req.NamespacedName, user)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get the associated Open5GS instance
	open5gsName := user.Spec.Open5GS.Name
	var open5gs netv1.Open5GS
	if err := r.Get(ctx, client.ObjectKey{Name: open5gsName, Namespace: user.Namespace}, &open5gs); err != nil {
		logger.Error(err, "Failed to get Open5GS instance", "Open5GS", open5gsName)
	}

	// Check if the user is being deleted
	if user.ObjectMeta.DeletionTimestamp.IsZero() {
		// Add finalizer if not present
		if !containsString(user.ObjectMeta.Finalizers, Open5GSUserFinalizer) {
			user.ObjectMeta.Finalizers = append(user.ObjectMeta.Finalizers, Open5GSUserFinalizer)
			if err := r.Update(ctx, user); err != nil {
				logger.Error(err, "Failed to add finalizer to Open5GSUser")
				return ctrl.Result{RequeueAfter: 10 * time.Second}, err
			}
		}
	} else {
		// Handle deletion
		if containsString(user.ObjectMeta.Finalizers, Open5GSUserFinalizer) {
			if err := r.deleteSubscriber(ctx, user, &open5gs, logger); err != nil {
				logger.Error(err, "Failed to delete subscriber from MongoDB", "Open5GS", open5gsName)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, err
			}
			user.ObjectMeta.Finalizers = removeString(user.ObjectMeta.Finalizers, Open5GSUserFinalizer)
			if err := r.Update(ctx, user); err != nil {
				logger.Error(err, "Failed to remove finalizer from Open5GSUser")
				return ctrl.Result{RequeueAfter: 10 * time.Second}, err
			}
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if err := r.reconcileSubscriber(ctx, *user, &open5gs, logger); err != nil {
		logger.Error(err, "Failed to reconcile subscriber in MongoDB", "Open5GS", open5gsName)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *Open5GSUserReconciler) reconcileSubscriber(ctx context.Context, user netv1.Open5GSUser, open5gs *netv1.Open5GS, logger logr.Logger) error {
	serviceName := fmt.Sprintf("%s-mongodb", strings.ToLower(open5gs.Name))
	ipService, err := r.GetServiceIp(ctx, serviceName, open5gs.Namespace)
	if err != nil {
		logger.Info("MongoDB service not found. Skipping reconciliation.", "service", serviceName)
		return nil
	}

	mongoURI := fmt.Sprintf("mongodb://%s:27017", ipService)

	err = addOrUpdateSubscriber(user, mongoURI, logger)
	if err != nil {
		logger.Error(err, "Failed to add or update subscriber", "IMSI", user.Spec.IMSI)
		return err
	}

	return nil
}

func (r *Open5GSUserReconciler) deleteSubscriber(ctx context.Context, user *netv1.Open5GSUser, open5gs *netv1.Open5GS, logger logr.Logger) error {
	serviceName := fmt.Sprintf("%s-mongodb", strings.ToLower(open5gs.Name))
	ipService, err := r.GetServiceIp(ctx, serviceName, open5gs.Namespace)
	if err != nil {
		logger.Info("MongoDB service not found. Skipping deletion.", "service", serviceName)
		return nil
	}

	mongoURI := fmt.Sprintf("mongodb://%s:27017", ipService)

	err = deleteSubscriberMongo(*user, mongoURI)
	if err != nil {
		logger.Error(err, "Failed to delete subscriber", "IMSI", user.Spec.IMSI)
		return err
	}

	logger.Info("Subscriber deleted from MongoDB", "IMSI", user.Spec.IMSI)

	return nil
}

func (r *Open5GSUserReconciler) GetServiceIp(ctx context.Context, serviceName string, namespace string) (string, error) {
	var service corev1.Service
	namespacedName := client.ObjectKey{Name: serviceName, Namespace: namespace}
	if err := r.Get(ctx, namespacedName, &service); err != nil {
		return "", err
	}
	return service.Spec.ClusterIP, nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return
}

func (r *Open5GSUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netv1.Open5GSUser{}).
		Complete(r)
}

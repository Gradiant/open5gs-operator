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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	netv1 "github.com/gradiant/open5gs-operator/api/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type Open5GSReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=net.gradiant.org,resources=open5gses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=net.gradiant.org,resources=open5gses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=net.gradiant.org,resources=open5gses/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete

func (r *Open5GSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	open5gs := &netv1.Open5GS{}
	err := r.Get(ctx, req.NamespacedName, open5gs)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	setDefaultValues(open5gs)

	if *open5gs.Spec.AMF.Enabled {
		if err := r.reconcileAMF(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "AMF", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.AUSF.Enabled {
		if err := r.reconcileAUSF(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "AUSF", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.BSF.Enabled {
		if err := r.reconcileBSF(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "BSF", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.NRF.Enabled {
		if err := r.reconcileNRF(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "NRF", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.NSSF.Enabled {
		if err := r.reconcileNSSF(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "NSSF", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.SMF.Enabled {
		if err := r.reconcileSMF(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "SMF", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.PCF.Enabled {
		if err := r.reconcilePCF(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "PCF", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.SCP.Enabled {
		if err := r.reconcileSCP(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "SCP", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.UDM.Enabled {
		if err := r.reconcileUDM(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "UDM", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.UDR.Enabled {
		if err := r.reconcileUDR(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "UDR", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.UPF.Enabled {
		if err := r.reconcileUPF(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "UPF", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	if *open5gs.Spec.WebUI.Enabled {
		if err := r.reconcileWebUI(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "WebUI", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}

	if *open5gs.Spec.MongoDB.Enabled {
		if err := r.reconcileMongoDB(ctx, req, open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteComponentResources(ctx, req, "MongoDB", open5gs, logger); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *Open5GSReconciler) reconcileComponent(ctx context.Context, open5gs *netv1.Open5GS, componentName string, logger logr.Logger, args ...interface{}) error {
	var configMap *corev1.ConfigMap
	var deployment *appsv1.Deployment
	var services []*corev1.Service
	var serviceMonitor *monitoringv1.ServiceMonitor
	var pvc *corev1.PersistentVolumeClaim
	var serviceAccount *corev1.ServiceAccount
	for _, arg := range args {
		switch v := arg.(type) {
		case *corev1.ConfigMap:
			configMap = v
		case *appsv1.Deployment:
			deployment = v
		case []*corev1.Service:
			services = v
		case *monitoringv1.ServiceMonitor:
			if available, err := isServiceMonitorCRDAvailable(r); err == nil && available {
				serviceMonitor = v
			}
		case *corev1.PersistentVolumeClaim:
			pvc = v
		case *corev1.ServiceAccount:
			serviceAccount = v
		default:
			return fmt.Errorf("unknown argument type %T", v)
		}

	}

	if configMap != nil {
		if err := ctrl.SetControllerReference(open5gs, configMap, r.Scheme); err != nil {
			return err
		}
		configMapHash, err := reconcileConfigMap(ctx, r, open5gs, configMap, componentName, logger)
		if err != nil {
			return err
		}
		if deployment != nil {
			if err := ctrl.SetControllerReference(open5gs, deployment, r.Scheme); err != nil {
				return err
			}
			if err := reconcileDeployment(ctx, r, open5gs, deployment, configMapHash, componentName, logger); err != nil {
				return err
			}
		}
	}

	existingServices := &corev1.ServiceList{}
	listOpts := []client.ListOption{
		client.InNamespace(open5gs.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance": open5gs.Name,
			"app.kubernetes.io/name":     strings.ToLower(componentName),
		}),
	}
	if err := r.Client.List(ctx, existingServices, listOpts...); err != nil {
		return err
	}

	desiredServiceNames := make(map[string]bool)
	for _, service := range services {
		desiredServiceNames[service.Name] = true
		if err := reconcileService(ctx, r, open5gs, service, componentName, logger); err != nil {
			return err
		}
	}

	for _, existingService := range existingServices.Items {
		if !desiredServiceNames[existingService.Name] {
			if err := r.Client.Delete(ctx, &existingService); err != nil {
				logger.Error(err, "Error deleting the Service", "component", componentName, "service", existingService.Name)
				return err
			}
			logger.Info("Service deleted", "component", componentName, "service", existingService.Name)
		}
	}
	for _, service := range services {
		if err := ctrl.SetControllerReference(open5gs, service, r.Scheme); err != nil {
			return err
		}
		if err := reconcileService(ctx, r, open5gs, service, componentName, logger); err != nil {
			return err
		}
	}
	if serviceMonitor != nil {
		if err := ctrl.SetControllerReference(open5gs, serviceMonitor, r.Scheme); err != nil {
			return err
		}
		if err := reconcileServiceMonitor(ctx, r, open5gs, serviceMonitor, componentName, logger); err != nil {
			return err
		}
	} else {
		if available, err := isServiceMonitorCRDAvailable(r); err == nil && available {
			existingServiceMonitors := &monitoringv1.ServiceMonitorList{}
			listOpts := []client.ListOption{
				client.InNamespace(open5gs.Namespace),
				client.MatchingLabels(map[string]string{
					"app.kubernetes.io/instance": open5gs.Name,
					"app.kubernetes.io/name":     strings.ToLower(componentName),
				}),
			}
			if err := r.Client.List(ctx, existingServiceMonitors, listOpts...); err != nil {
				return err
			}
			for _, existingServiceMonitor := range existingServiceMonitors.Items {
				if err := r.Client.Delete(ctx, client.Object(existingServiceMonitor)); err != nil {
					logger.Error(err, "Error deleting the ServiceMonitor", "component", componentName, "serviceMonitor", existingServiceMonitor.Name)
					return err
				}
				logger.Info("ServiceMonitor deleted", "component", componentName, "serviceMonitor", existingServiceMonitor.Name)
			}
		}
	}

	if pvc != nil {
		if err := ctrl.SetControllerReference(open5gs, pvc, r.Scheme); err != nil {
			return err
		}
		if err := reconcilePVC(ctx, r, open5gs, pvc, componentName, logger); err != nil {
			return err
		}
	}

	if serviceAccount != nil {
		if err := ctrl.SetControllerReference(open5gs, serviceAccount, r.Scheme); err != nil {
			return err
		}
		if err := reconcileServiceAccount(ctx, r, open5gs, serviceAccount, componentName, logger); err != nil {
			return err
		}
	} else {
		existingServiceAccounts := &corev1.ServiceAccountList{}
		listOpts := []client.ListOption{
			client.InNamespace(open5gs.Namespace),
			client.MatchingLabels(map[string]string{
				"app.kubernetes.io/instance": open5gs.Name,
				"app.kubernetes.io/name":     strings.ToLower(componentName),
			}),
		}
		if err := r.Client.List(ctx, existingServiceAccounts, listOpts...); err != nil {
			return err
		}
		for _, existingServiceAccount := range existingServiceAccounts.Items {
			if err := r.Client.Delete(ctx, client.Object(&existingServiceAccount)); err != nil {
				logger.Error(err, "Error deleting the ServiceAccount", "component", componentName, "serviceAccount", existingServiceAccount.Name)
				return err
			}
			logger.Info("ServiceAccount deleted", "component", componentName, "serviceAccount", existingServiceAccount.Name)
		}
	}

	return nil
}

func (r *Open5GSReconciler) reconcileAMF(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "AMF"
	configMap := CreateAMFConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration, *open5gs.Spec.AMF.Metrics)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 7777,
			Name:          "sbi",
			Protocol:      corev1.ProtocolTCP,
		},
		{
			ContainerPort: 38412,
			Name:          "ngap",
			Protocol:      corev1.ProtocolSCTP,
		},
	}
	if *open5gs.Spec.AMF.Metrics {
		ports = append(ports, corev1.ContainerPort{
			ContainerPort: 9090,
			Name:          "metrics",
			Protocol:      corev1.ProtocolTCP,
		})
	}
	envVars := []corev1.EnvVar{}
	ngapService := netv1.Open5GSService{Name: "ngap"}
	if len(open5gs.Spec.AMF.Service) > 0 {
		for _, service := range open5gs.Spec.AMF.Service {
			if service.Name == "ngap" {
				ngapService = service
				break
			}
		}
	}
	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "sbi", 7777, "TCP"),
		CreateService(req.Namespace, open5gs.Name, componentName, "ngap", 38412, "SCTP", ngapService),
	}
	if *open5gs.Spec.AMF.Metrics {
		services = append(services, CreateService(req.Namespace, open5gs.Name, componentName, "metrics", 9090, "TCP"))
	}

	var serviceMonitor *monitoringv1.ServiceMonitor
	if *open5gs.Spec.AMF.ServiceMonitor {
		serviceMonitor = CreateServiceMonitor(req.Namespace, open5gs.Name, strings.ToLower(componentName))
	}
	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if *open5gs.Spec.AMF.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateDeployment(req.Namespace, open5gs.Name, componentName, open5gs.Spec.Open5GSImage, open5gs.Name+"-amf", "open5gs-amfd", ports, envVars, serviceAccountName)

	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceMonitor, serviceAccount)
}

func (r *Open5GSReconciler) reconcileAUSF(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "AUSF"
	configMap := CreateAUSFConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 7777,
			Name:          "sbi",
			Protocol:      corev1.ProtocolTCP,
		},
	}

	envVars := []corev1.EnvVar{}

	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "sbi", 7777, "TCP"),
	}
	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if *open5gs.Spec.AUSF.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateDeployment(req.Namespace, open5gs.Name, componentName, open5gs.Spec.Open5GSImage, open5gs.Name+"-ausf", "open5gs-ausfd", ports, envVars, serviceAccountName)

	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceAccount)
}

func (r *Open5GSReconciler) reconcileBSF(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "BSF"
	configMap := CreateBSFConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 7777,
			Name:          "sbi",
			Protocol:      corev1.ProtocolTCP,
		},
	}

	envVars := []corev1.EnvVar{}

	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "sbi", 7777, "TCP"),
	}
	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if *open5gs.Spec.BSF.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)

		serviceAccountName = serviceAccount.Name

	}
	deployment := CreateDeployment(req.Namespace, open5gs.Name, componentName, open5gs.Spec.Open5GSImage, open5gs.Name+"-bsf", "open5gs-bsfd", ports, envVars, serviceAccountName)

	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceAccount)
}

func (r *Open5GSReconciler) reconcileMongoDB(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "MongoDB"
	configMap := CreateMongoDBConfigMap(req.Namespace, open5gs.Name)
	pvc := CreateMongoDBPVC(req.Namespace, open5gs.Name)
	envVars := []corev1.EnvVar{
		{Name: "BITNAMI_DEBUG", Value: "false"},
		{Name: "ALLOW_EMPTY_PASSWORD", Value: "yes"},
		{Name: "MONGODB_SYSTEM_LOG_VERBOSITY", Value: "0"},
		{Name: "MONGODB_DISABLE_SYSTEM_LOG", Value: "no"},
		{Name: "MONGODB_DISABLE_JAVASCRIPT", Value: "no"},
		{Name: "MONGODB_ENABLE_JOURNAL", Value: "yes"},
		{Name: "MONGODB_PORT_NUMBER", Value: "27017"},
		{Name: "MONGODB_ENABLE_IPV6", Value: "no"},
		{Name: "MONGODB_ENABLE_DIRECTORY_PER_DB", Value: "no"},
	}
	services := []*corev1.Service{
		CreateMongoDBService(req.Namespace, open5gs.Name),
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if *open5gs.Spec.MongoDB.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name

	}
	deployment := CreateMongoDBDeployment(req.Namespace, open5gs.Name, open5gs.Spec.MongoDBVersion, envVars, serviceAccountName)
	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	}
	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, pvc, serviceAccount)
}

func (r *Open5GSReconciler) reconcileNRF(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "NRF"
	configMap := CreateNRFConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 7777,
			Name:          "sbi",
			Protocol:      corev1.ProtocolTCP,
		},
	}

	envVars := []corev1.EnvVar{}

	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "sbi", 7777, "TCP"),
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if *open5gs.Spec.NRF.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateDeployment(req.Namespace, open5gs.Name, componentName, open5gs.Spec.Open5GSImage, open5gs.Name+"-nrf", "open5gs-nrfd", ports, envVars, serviceAccountName)

	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceAccount)
}
func (r *Open5GSReconciler) reconcileNSSF(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "NSSF"
	configMap := CreateNSSFConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 7777,
			Name:          "sbi",
			Protocol:      corev1.ProtocolTCP,
		},
	}

	envVars := []corev1.EnvVar{}

	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "sbi", 7777, "TCP"),
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if open5gs.Spec.NSSF.ServiceAccount != nil && *open5gs.Spec.NSSF.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateDeployment(req.Namespace, open5gs.Name, componentName, open5gs.Spec.Open5GSImage, open5gs.Name+"-nssf", "open5gs-nssfd", ports, envVars, serviceAccountName)

	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceAccount)
}

func (r *Open5GSReconciler) reconcilePCF(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "PCF"
	configMap := CreatePCFConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration, *open5gs.Spec.PCF.Metrics)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 7777,
			Name:          "sbi",
			Protocol:      corev1.ProtocolTCP,
		},
	}
	if *open5gs.Spec.PCF.Metrics {
		ports = append(ports, corev1.ContainerPort{
			ContainerPort: 9090,
			Name:          "metrics",
			Protocol:      corev1.ProtocolTCP,
		})
	}

	envVars := []corev1.EnvVar{
		{
			Name:  "DB_URI",
			Value: "mongodb://" + open5gs.Name + "-mongodb/open5gs",
		},
	}

	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "sbi", 7777, "TCP"),
	}
	if *open5gs.Spec.PCF.Metrics {
		services = append(services, CreateService(req.Namespace, open5gs.Name, componentName, "metrics", 9090, "TCP"))
	}

	var serviceMonitor *monitoringv1.ServiceMonitor
	if open5gs.Spec.PCF.ServiceMonitor != nil && *open5gs.Spec.PCF.ServiceMonitor {
		serviceMonitor = CreateServiceMonitor(req.Namespace, open5gs.Name, strings.ToLower(componentName))
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if open5gs.Spec.PCF.ServiceAccount != nil && *open5gs.Spec.PCF.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateDeployment(req.Namespace, open5gs.Name, componentName, open5gs.Spec.Open5GSImage, open5gs.Name+"-pcf", "open5gs-pcfd", ports, envVars, serviceAccountName)

	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceMonitor, serviceAccount)
}

func (r *Open5GSReconciler) reconcileSCP(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "SCP"
	configMap := CreateSCPConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 7777,
			Name:          "sbi",
			Protocol:      corev1.ProtocolTCP,
		},
	}

	envVars := []corev1.EnvVar{
		{
			Name:  "DB_URI",
			Value: "mongodb://" + open5gs.Name + "-mongodb/open5gs",
		},
	}

	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "sbi", 7777, "TCP"),
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if open5gs.Spec.SCP.ServiceAccount != nil && *open5gs.Spec.SCP.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateDeployment(req.Namespace, open5gs.Name, componentName, open5gs.Spec.Open5GSImage, open5gs.Name+"-scp", "open5gs-scpd", ports, envVars, serviceAccountName)

	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceAccount)
}

func (r *Open5GSReconciler) reconcileSMF(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "SMF"
	configMap := CreateSMFConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration, *open5gs.Spec.SMF.Metrics)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 7777,
			Name:          "sbi",
			Protocol:      corev1.ProtocolTCP,
		},
		{
			ContainerPort: 2123,
			Name:          "gtpc",
			Protocol:      corev1.ProtocolUDP,
		},
		{
			ContainerPort: 2152,
			Name:          "gtpu",
			Protocol:      corev1.ProtocolUDP,
		},
		{
			ContainerPort: 8805,
			Name:          "pfcp",
			Protocol:      corev1.ProtocolUDP,
		},
	}
	if *open5gs.Spec.SMF.Metrics {
		ports = append(ports, corev1.ContainerPort{
			ContainerPort: 9090,
			Name:          "metrics",
			Protocol:      corev1.ProtocolTCP,
		})
	}

	envVars := []corev1.EnvVar{}
	pfcpService := netv1.Open5GSService{Name: "pfcp"}
	if len(open5gs.Spec.SMF.Service) > 0 {
		for _, service := range open5gs.Spec.SMF.Service {
			if service.Name == "pfcp" {
				pfcpService = service
				break
			}
		}
	}
	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "sbi", 7777, "TCP"),
		CreateService(req.Namespace, open5gs.Name, componentName, "gtpc", 2123, "UDP"),
		CreateService(req.Namespace, open5gs.Name, componentName, "gtpu", 2152, "UDP"),
		CreateService(req.Namespace, open5gs.Name, componentName, "pfcp", 8805, "UDP", pfcpService),
	}
	if *open5gs.Spec.SMF.Metrics {
		services = append(services, CreateService(req.Namespace, open5gs.Name, componentName, "metrics", 9090, "TCP"))
	}

	var serviceMonitor *monitoringv1.ServiceMonitor
	if open5gs.Spec.SMF.ServiceMonitor != nil && *open5gs.Spec.SMF.ServiceMonitor {
		serviceMonitor = CreateServiceMonitor(req.Namespace, open5gs.Name, strings.ToLower(componentName))
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if open5gs.Spec.SMF.ServiceAccount != nil && *open5gs.Spec.SMF.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateDeployment(req.Namespace, open5gs.Name, componentName, open5gs.Spec.Open5GSImage, open5gs.Name+"-smf", "open5gs-smfd", ports, envVars, serviceAccountName)

	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceMonitor, serviceAccount)
}

func (r *Open5GSReconciler) reconcileUDM(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "UDM"
	configMap := CreateUDMConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 7777,
			Name:          "sbi",
			Protocol:      corev1.ProtocolTCP,
		},
	}

	envVars := []corev1.EnvVar{}

	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "sbi", 7777, "TCP"),
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if open5gs.Spec.UDM.ServiceAccount != nil && *open5gs.Spec.UDM.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateDeployment(req.Namespace, open5gs.Name, componentName, open5gs.Spec.Open5GSImage, open5gs.Name+"-udm", "open5gs-udmd", ports, envVars, serviceAccountName)

	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceAccount)
}

func (r *Open5GSReconciler) reconcileUDR(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "UDR"
	configMap := CreateUDRConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 7777,
			Name:          "sbi",
			Protocol:      corev1.ProtocolTCP,
		},
	}

	envVars := []corev1.EnvVar{
		{
			Name:  "DB_URI",
			Value: "mongodb://" + open5gs.Name + "-mongodb/open5gs",
		},
	}

	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "sbi", 7777, "TCP"),
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if open5gs.Spec.UDR.ServiceAccount != nil && *open5gs.Spec.UDR.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateDeployment(req.Namespace, open5gs.Name, componentName, open5gs.Spec.Open5GSImage, open5gs.Name+"-udr", "open5gs-udrd", ports, envVars, serviceAccountName)
	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceAccount)
}

func (r *Open5GSReconciler) reconcileUPF(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "UPF"
	configMap := CreateUPFConfigMap(req.Namespace, open5gs.Name, open5gs.Spec.Configuration, *open5gs.Spec.UPF.Metrics)
	entrypointConfigMap := CreateUPFEntrypointConfigMap(req.Namespace, open5gs.Name)

	envVars := []corev1.EnvVar{}
	pfcpService := netv1.Open5GSService{Name: "pfcp"}
	gtpuService := netv1.Open5GSService{Name: "gtpu"}
	if len(open5gs.Spec.UPF.Service) > 0 {
		for _, service := range open5gs.Spec.UPF.Service {
			if service.Name == "pfcp" {
				pfcpService = service
			}
			if service.Name == "gtpu" {
				gtpuService = service
			}
		}
	}
	services := []*corev1.Service{
		CreateService(req.Namespace, open5gs.Name, componentName, "pfcp", 8805, "UDP", pfcpService),
		CreateService(req.Namespace, open5gs.Name, componentName, "gtpu", 2152, "UDP", gtpuService),
	}
	if *open5gs.Spec.UPF.Metrics {
		services = append(services, CreateService(req.Namespace, open5gs.Name, componentName, "metrics", 9090, "TCP"))
	}

	if err := ctrl.SetControllerReference(open5gs, entrypointConfigMap, r.Scheme); err != nil {
		return err
	}
	reconcileConfigMap(ctx, r, open5gs, entrypointConfigMap, componentName, logger)

	var serviceMonitor *monitoringv1.ServiceMonitor
	if *open5gs.Spec.UPF.ServiceMonitor {
		serviceMonitor = CreateServiceMonitor(req.Namespace, open5gs.Name, strings.ToLower(componentName))
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if open5gs.Spec.UPF.ServiceAccount != nil && *open5gs.Spec.UPF.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateUPFDeployment(req.Namespace, open5gs.Name, open5gs.Spec.Open5GSImage, envVars, *open5gs.Spec.UPF.Metrics, serviceAccountName)
	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceMonitor, serviceAccount)
}

func (r *Open5GSReconciler) reconcileWebUI(ctx context.Context, req ctrl.Request, open5gs *netv1.Open5GS, logger logr.Logger) error {
	componentName := "WebUI"
	configMap := CreateWebUIConfigMap(req.Namespace, open5gs.Name)

	envVars := []corev1.EnvVar{
		{
			Name:  "DB_URI",
			Value: "mongodb://" + open5gs.Name + "-mongodb/open5gs",
		},
	}

	service := CreateService(req.Namespace, open5gs.Name, componentName, "http", 9999, "TCP")
	services := []*corev1.Service{service}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := ""
	if open5gs.Spec.WebUI.ServiceAccount != nil && *open5gs.Spec.WebUI.ServiceAccount {
		serviceAccount = CreateServiceAccount(req.Namespace, open5gs.Name, componentName)
		serviceAccountName = serviceAccount.Name
	}
	deployment := CreateWebUIDeployment(req.Namespace, open5gs.Name, open5gs.Spec.WebUIImage, envVars, serviceAccountName)
	return r.reconcileComponent(ctx, open5gs, componentName, logger, configMap, deployment, services, serviceAccount)
}

// This function deletes all the resources related to a component (with the OwnerReference set to the Open5GS CR)
func (r *Open5GSReconciler) deleteComponentResources(ctx context.Context, req ctrl.Request, componentName string, open5gs *netv1.Open5GS, logger logr.Logger) error {
	configMap := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: open5gs.Name + "-" + strings.ToLower(componentName), Namespace: req.Namespace}, configMap)
	if err == nil {
		if hasOwnerReference(configMap, open5gs) {
			if err := r.Client.Delete(ctx, configMap); err != nil {
				logger.Error(err, "Error deleting the ConfigMap", "component", componentName)
				return err
			}
			logger.Info("ConfigMap deleted", "component", componentName)
		}
	} else if !errors.IsNotFound(err) {
		logger.Error(err, "Error obtaining the ConfigMap", "component", componentName)
		return err
	}

	deployment := &appsv1.Deployment{}
	err = r.Client.Get(ctx, client.ObjectKey{Name: open5gs.Name + "-" + strings.ToLower(componentName), Namespace: req.Namespace}, deployment)
	if err == nil {
		if hasOwnerReference(deployment, open5gs) {
			if err := r.Client.Delete(ctx, deployment); err != nil {
				logger.Error(err, "Error deleting the Deployment", "component", componentName)
				return err
			}
			logger.Info("Deployment deleted", "component", componentName)
		}
	} else if !errors.IsNotFound(err) {
		logger.Error(err, "Error obtaining the Deployment", "component", componentName)
		return err
	}

	serviceList := &corev1.ServiceList{}
	listOpts := []client.ListOption{
		client.InNamespace(req.Namespace),
	}
	if err := r.Client.List(ctx, serviceList, listOpts...); err != nil {
		logger.Error(err, "Error al listar los Services", "component", componentName)
		return err
	}
	for _, service := range serviceList.Items {
		if strings.HasPrefix(service.Name, open5gs.Name+"-"+strings.ToLower(componentName)) && hasOwnerReference(&service, open5gs) {
			if err := r.Client.Delete(ctx, &service); err != nil {
				logger.Error(err, "Error deleting the Service", "component", componentName, "service", service.Name)
				return err
			}
			logger.Info("Service deleted", "component", componentName, "service", service.Name)
		}
	}
	if available, err := isServiceMonitorCRDAvailable(r); err == nil && available {

		serviceMonitor := &monitoringv1.ServiceMonitor{}
		err = r.Client.Get(ctx, client.ObjectKey{Name: open5gs.Name + "-" + strings.ToLower(componentName), Namespace: req.Namespace}, serviceMonitor)
		if err == nil {
			if hasOwnerReference(serviceMonitor, open5gs) {
				if err := r.Client.Delete(ctx, serviceMonitor); err != nil {
					logger.Error(err, "Error deleting the ServiceMonitor", "component", componentName)
					return err
				}
				logger.Info("ServiceMonitor deleted", "component", componentName)
			}
		} else if !errors.IsNotFound(err) {
			logger.Error(err, "Error obtaining the ServiceMonitor", "component", componentName)
			return err
		}
	}

	pvc := &corev1.PersistentVolumeClaim{}
	err = r.Client.Get(ctx, client.ObjectKey{Name: open5gs.Name + "-" + strings.ToLower(componentName), Namespace: req.Namespace}, pvc)
	if err == nil {
		if hasOwnerReference(pvc, open5gs) {
			if err := r.Client.Delete(ctx, pvc); err != nil {
				logger.Error(err, "Error deleting the PVC", "component", componentName)
				return err
			}
			logger.Info("PVC deleted", "component", componentName)
		}
	} else if !errors.IsNotFound(err) {
		logger.Error(err, "Error obtaining the PVC", "component", componentName)
		return err
	}

	serviceAccount := &corev1.ServiceAccount{}
	err = r.Client.Get(ctx, client.ObjectKey{Name: open5gs.Name + "-" + strings.ToLower(componentName), Namespace: req.Namespace}, serviceAccount)
	if err == nil {
		if hasOwnerReference(serviceAccount, open5gs) {
			if err := r.Client.Delete(ctx, serviceAccount); err != nil {
				logger.Error(err, "Error deleting the ServiceAccount", "component", componentName)
				return err
			}
			logger.Info("ServiceAccount deleted", "component", componentName)
		}
	} else if !errors.IsNotFound(err) {
		logger.Error(err, "Error obtaining the ServiceAccount", "component", componentName)
		return err
	}
	return nil
}

func reconcileConfigMap(ctx context.Context, r *Open5GSReconciler, open5gs *netv1.Open5GS, configMap *corev1.ConfigMap, componentName string, logger logr.Logger) (string, error) {
	if err := setOwnerReference(open5gs, configMap, r.Scheme); err != nil {
		return "", err
	}

	configMapHash, err := generateConfigMapHash(configMap)
	if err != nil {
		logger.Error(err, "Error al generar el hash del ConfigMap", "component", componentName)
		return "", err
	}

	foundConfigMap := &corev1.ConfigMap{}
	err = r.Client.Get(ctx, client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, foundConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.Client.Create(ctx, configMap); err != nil {
				logger.Error(err, "Failed to create ConfigMap", "component", componentName)
				return "", err
			}
			logger.Info("ConfigMap created", "component", componentName)
		} else {
			logger.Error(err, "Error obtaining the ConfigMap", "component", componentName)
			return "", err
		}
	} else {
		if !hasOwnerReference(foundConfigMap, open5gs) {
			return configMapHash, nil
		}

		if !configMapEqual(configMap, foundConfigMap) {
			foundConfigMap.Data = configMap.Data
			if err := r.Client.Update(ctx, foundConfigMap); err != nil {
				logger.Error(err, "Failed to update the ConfigMap", "component", componentName)
				return "", err
			}
			logger.Info("ConfigMap updated", "component", componentName)
		}
	}
	return configMapHash, nil
}

func reconcileDeployment(ctx context.Context, r *Open5GSReconciler, open5gs *netv1.Open5GS, deployment *appsv1.Deployment, configMapHash string, componentName string, logger logr.Logger) error {
	if err := setOwnerReference(open5gs, deployment, r.Scheme); err != nil {
		return err
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["open5gs/configmap-hash"] = configMapHash

	foundDeployment := &appsv1.Deployment{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.Client.Create(ctx, deployment); err != nil {
				logger.Error(err, "Failed to create Deployment", "component", componentName)
				return err
			}
			logger.Info("Deployment created", "component", componentName)
		} else {
			logger.Error(err, "Error obtaining the Deployment", "component", componentName)
			return err
		}
	} else {
		if !hasOwnerReference(foundDeployment, open5gs) {
			return nil
		}

		if !deploymentEqual(deployment, foundDeployment) || foundDeployment.Spec.Template.Annotations["open5gs/configmap-hash"] != configMapHash {
			foundDeployment.Spec = deployment.Spec
			foundDeployment.Spec.Template.Annotations["open5gs/configmap-hash"] = configMapHash
			if err := r.Client.Update(ctx, foundDeployment); err != nil {
				logger.Error(err, "Failed to update the Deployment", "component", componentName)
				return err
			}
			logger.Info("Deployment updated", "component", componentName)
		}
	}
	return nil
}

func reconcilePVC(ctx context.Context, r *Open5GSReconciler, open5gs *netv1.Open5GS, pvc *corev1.PersistentVolumeClaim, componentName string, logger logr.Logger) error {
	foundPVC := &corev1.PersistentVolumeClaim{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: pvc.Name, Namespace: pvc.Namespace}, foundPVC)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.Client.Create(ctx, pvc); err != nil {
				logger.Error(err, "Failed to create PVC", "component", componentName)
				return err
			}
			logger.Info("PVC created", "component", componentName)
		} else {
			logger.Error(err, "Error obtaining the PVC", "component", componentName)
			return err
		}
	} else {
		if !hasOwnerReference(foundPVC, open5gs) {
			return nil
		}

		if !pvcEqual(pvc, foundPVC) {
			foundPVC.Spec = pvc.Spec
			if err := r.Client.Update(ctx, foundPVC); err != nil {
				logger.Error(err, "Failed to update the PVC", "component", componentName)
				return err
			}
			logger.Info("PVC updated", "component", componentName)
		}
	}
	return nil
}

func reconcileService(ctx context.Context, r *Open5GSReconciler, open5gs *netv1.Open5GS, service *corev1.Service, componentName string, logger logr.Logger) error {
	if err := setOwnerReference(open5gs, service, r.Scheme); err != nil {
		return err
	}

	foundService := &corev1.Service{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.Client.Create(ctx, service); err != nil {
				logger.Error(err, "Failed to create Service", "component", componentName)
				return err
			}
			logger.Info("Service created", "component", componentName)
		} else {
			logger.Error(err, "Error obtaining the Service", "component", componentName)
			return err
		}
	} else {
		if !hasOwnerReference(foundService, open5gs) {
			return nil
		}

		if !serviceEqual(service, foundService) {
			foundService.Spec = service.Spec
			if err := r.Client.Update(ctx, foundService); err != nil {
				logger.Error(err, "Failed to update the Service", "component", componentName)
				return err
			}
			logger.Info("Service updated", "component", componentName)
		}
	}
	return nil
}

func reconcileServiceMonitor(ctx context.Context, r *Open5GSReconciler, open5gs *netv1.Open5GS, serviceMonitor *monitoringv1.ServiceMonitor, componentName string, logger logr.Logger) error {
	available, err := isServiceMonitorCRDAvailable(r)
	if err != nil {
		return err
	}
	if !available {
		return nil
	}
	if err := setOwnerReference(open5gs, serviceMonitor, r.Scheme); err != nil {
		return err
	}
	foundServiceMonitor := &monitoringv1.ServiceMonitor{}
	err = r.Client.Get(ctx, client.ObjectKey{Name: serviceMonitor.Name, Namespace: serviceMonitor.Namespace}, foundServiceMonitor)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.Client.Create(ctx, serviceMonitor); err != nil {
				logger.Error(err, "Failed to create ServiceMonitor", "component", componentName)
				return err
			}
			logger.Info("ServiceMonitor created", "component", componentName)
		} else {
			logger.Error(err, "Error obtaining the ServiceMonitor", "component", componentName)
			return err
		}
	} else {
		if !hasOwnerReference(foundServiceMonitor, open5gs) {
			return nil
		}

		if !serviceMonitorEqual(serviceMonitor, foundServiceMonitor) {
			foundServiceMonitor.Spec = serviceMonitor.Spec
			if err := r.Client.Update(ctx, foundServiceMonitor); err != nil {
				logger.Error(err, "Failed to update the ServiceMonitor", "component", componentName)
				return err
			}
			logger.Info("ServiceMonitor updated", "component", componentName)
		}
	}
	return nil

}

func reconcileServiceAccount(ctx context.Context, r *Open5GSReconciler, open5gs *netv1.Open5GS, serviceAccount *corev1.ServiceAccount, componentName string, logger logr.Logger) error {
	if err := setOwnerReference(open5gs, serviceAccount, r.Scheme); err != nil {
		return err
	}

	foundServiceAccount := &corev1.ServiceAccount{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: serviceAccount.Name, Namespace: serviceAccount.Namespace}, foundServiceAccount)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.Client.Create(ctx, serviceAccount); err != nil {
				logger.Error(err, "Failed to create ServiceAccount", "component", componentName)
				return err
			}
			logger.Info("ServiceAccount created", "component", componentName)
		} else {
			logger.Error(err, "Error obtaining the ServiceAccount", "component", componentName)
			return err
		}
	} else {
		if !hasOwnerReference(foundServiceAccount, open5gs) {
			return nil
		}

		if !serviceAccountEqual(serviceAccount, foundServiceAccount) {
			foundServiceAccount.Annotations = serviceAccount.Annotations
			foundServiceAccount.Labels = serviceAccount.Labels
			if err := r.Client.Update(ctx, foundServiceAccount); err != nil {
				logger.Error(err, "Failed to update the ServiceAccount", "component", componentName)
				return err
			}
			logger.Info("ServiceAccount updated", "component", componentName)
		}
	}
	return nil
}

func setDefaultValues(open5gs *netv1.Open5GS) {
	if open5gs.Spec.AMF.Enabled == nil {
		defaultAMFEnabled := true
		open5gs.Spec.AMF.Enabled = &defaultAMFEnabled
	}
	if open5gs.Spec.AMF.Metrics == nil {
		defaultAMFMetrics := true
		open5gs.Spec.AMF.Metrics = &defaultAMFMetrics
	}
	if open5gs.Spec.AMF.ServiceMonitor == nil {
		defaultAMFServiceMonitor := true
		open5gs.Spec.AMF.ServiceMonitor = &defaultAMFServiceMonitor
	}
	if open5gs.Spec.AUSF.Enabled == nil {
		defaultAUSFEnabled := true
		open5gs.Spec.AUSF.Enabled = &defaultAUSFEnabled
	}
	if open5gs.Spec.BSF.Enabled == nil {
		defaultBSFEnabled := true
		open5gs.Spec.BSF.Enabled = &defaultBSFEnabled
	}
	if open5gs.Spec.NRF.Enabled == nil {
		defaultNRFEnabled := true
		open5gs.Spec.NRF.Enabled = &defaultNRFEnabled
	}
	if open5gs.Spec.NSSF.Enabled == nil {
		defaultNSSFEnabled := true
		open5gs.Spec.NSSF.Enabled = &defaultNSSFEnabled
	}
	if open5gs.Spec.SMF.Enabled == nil {
		defaultSMFEnabled := true
		open5gs.Spec.SMF.Enabled = &defaultSMFEnabled
	}
	if open5gs.Spec.SMF.Metrics == nil {
		defaultSMFMetrics := true
		open5gs.Spec.SMF.Metrics = &defaultSMFMetrics
	}
	if open5gs.Spec.SMF.ServiceMonitor == nil {
		defaultSMFServiceMonitor := true
		open5gs.Spec.SMF.ServiceMonitor = &defaultSMFServiceMonitor
	}
	if open5gs.Spec.PCF.Enabled == nil {
		defaultPCFEnabled := true
		open5gs.Spec.PCF.Enabled = &defaultPCFEnabled
	}
	if open5gs.Spec.PCF.Metrics == nil {
		defaultPCFMetrics := true
		open5gs.Spec.PCF.Metrics = &defaultPCFMetrics
	}
	if open5gs.Spec.PCF.ServiceMonitor == nil {
		defaultPCFServiceMonitor := true
		open5gs.Spec.PCF.ServiceMonitor = &defaultPCFServiceMonitor
	}
	if open5gs.Spec.SCP.Enabled == nil {
		defaultSCPEnabled := true
		open5gs.Spec.SCP.Enabled = &defaultSCPEnabled
	}
	if open5gs.Spec.UDM.Enabled == nil {
		defaultUDMEnabled := true
		open5gs.Spec.UDM.Enabled = &defaultUDMEnabled
	}
	if open5gs.Spec.UDR.Enabled == nil {
		defaultUDREnabled := true
		open5gs.Spec.UDR.Enabled = &defaultUDREnabled
	}
	if open5gs.Spec.UPF.Enabled == nil {
		defaultUPFEnabled := true
		open5gs.Spec.UPF.Enabled = &defaultUPFEnabled
	}
	if open5gs.Spec.UPF.Metrics == nil {
		defaultUPFMetrics := true
		open5gs.Spec.UPF.Metrics = &defaultUPFMetrics
	}
	if open5gs.Spec.UPF.ServiceMonitor == nil {
		defaultUPFServiceMonitor := true
		open5gs.Spec.UPF.ServiceMonitor = &defaultUPFServiceMonitor
	}
	if open5gs.Spec.WebUI.Enabled == nil {
		defaultWebUIEnabled := false
		open5gs.Spec.WebUI.Enabled = &defaultWebUIEnabled
	}
	if open5gs.Spec.MongoDB.Enabled == nil {
		defaultMongoDBEnabled := true
		open5gs.Spec.MongoDB.Enabled = &defaultMongoDBEnabled
	}
	if open5gs.Spec.Configuration.MCC == "" {
		open5gs.Spec.Configuration.MCC = "999"
	}
	if open5gs.Spec.Configuration.MNC == "" {
		open5gs.Spec.Configuration.MNC = "70"
	}
	if open5gs.Spec.Configuration.Region == "" {
		open5gs.Spec.Configuration.Region = "2"
	}
	if open5gs.Spec.Configuration.Set == "" {
		open5gs.Spec.Configuration.Set = "1"
	}
	if open5gs.Spec.Configuration.TAC == "" {
		open5gs.Spec.Configuration.TAC = "0001"
	}
	if open5gs.Spec.Configuration.Slices == nil {
		open5gs.Spec.Configuration.Slices = []netv1.Open5GSSlice{}
		slice := netv1.Open5GSSlice{
			SST: "1",
			SD:  "0xffffff",
		}
		open5gs.Spec.Configuration.Slices = append(open5gs.Spec.Configuration.Slices, slice)
	}
	if open5gs.Spec.Open5GSImage == "" {
		open5gs.Spec.Open5GSImage = "docker.io/gradiant/open5gs:2.7.2"
	}
	if open5gs.Spec.WebUIImage == "" {
		open5gs.Spec.WebUIImage = "docker.io/gradiant/open5gs-webui:2.7.2"
	}
	if open5gs.Spec.MongoDBVersion == "" {
		open5gs.Spec.MongoDBVersion = "5.0.10-debian-11-r3"
	}
	if open5gs.Spec.AMF.ServiceAccount == nil {
		defaultAMFServiceAccount := false
		open5gs.Spec.AMF.ServiceAccount = &defaultAMFServiceAccount
	}
	if open5gs.Spec.AUSF.ServiceAccount == nil {
		defaultAUSFServiceAccount := false
		open5gs.Spec.AUSF.ServiceAccount = &defaultAUSFServiceAccount
	}
	if open5gs.Spec.BSF.ServiceAccount == nil {
		defaultBSFServiceAccount := false
		open5gs.Spec.BSF.ServiceAccount = &defaultBSFServiceAccount
	}
	if open5gs.Spec.MongoDB.ServiceAccount == nil {
		defaultMongoDBServiceAccount := false
		open5gs.Spec.MongoDB.ServiceAccount = &defaultMongoDBServiceAccount
	}
	if open5gs.Spec.NRF.ServiceAccount == nil {
		defaultNRFServiceAccount := false
		open5gs.Spec.NRF.ServiceAccount = &defaultNRFServiceAccount
	}
	if open5gs.Spec.NSSF.ServiceAccount == nil {
		defaultNSSFServiceAccount := false
		open5gs.Spec.NSSF.ServiceAccount = &defaultNSSFServiceAccount
	}
	if open5gs.Spec.PCF.ServiceAccount == nil {
		defaultPCFServiceAccount := false
		open5gs.Spec.PCF.ServiceAccount = &defaultPCFServiceAccount
	}
	if open5gs.Spec.SCP.ServiceAccount == nil {
		defaultSCPServiceAccount := false
		open5gs.Spec.SCP.ServiceAccount = &defaultSCPServiceAccount
	}
	if open5gs.Spec.SMF.ServiceAccount == nil {
		defaultSMFServiceAccount := false
		open5gs.Spec.SMF.ServiceAccount = &defaultSMFServiceAccount
	}
	if open5gs.Spec.UDM.ServiceAccount == nil {
		defaultUDMServiceAccount := false
		open5gs.Spec.UDM.ServiceAccount = &defaultUDMServiceAccount
	}
	if open5gs.Spec.UDR.ServiceAccount == nil {
		defaultUDRServiceAccount := false
		open5gs.Spec.UDR.ServiceAccount = &defaultUDRServiceAccount
	}
	if open5gs.Spec.UPF.ServiceAccount == nil {
		defaultUPFServiceAccount := false
		open5gs.Spec.UPF.ServiceAccount = &defaultUPFServiceAccount
	}
	if open5gs.Spec.WebUI.ServiceAccount == nil {
		defaultWebUIServiceAccount := false
		open5gs.Spec.WebUI.ServiceAccount = &defaultWebUIServiceAccount
	}

}

func (r *Open5GSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netv1.Open5GS{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				log.FromContext(context.Background()).Info("Open5GS '"+e.Object.GetName()+"' has been completely deleted", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
				return false
			},
		}).
		Complete(r)
}

func isServiceMonitorCRDAvailable(r *Open5GSReconciler) (bool, error) {
	gvr := schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "servicemonitors",
	}

	var list unstructured.UnstructuredList
	list.SetGroupVersionKind(gvr.GroupVersion().WithKind("ServiceMonitorList"))

	err := r.Client.List(context.TODO(), &list)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

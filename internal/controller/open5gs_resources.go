/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package controller

import (
	"strings"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	netv1 "github.com/gradiant/open5gs-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func CreateService(namespace, open5gsName, functionName string, protocol string, port int32, l4_protocol corev1.Protocol, args ...interface{}) *corev1.Service {
	labels := map[string]string{
		"app.kubernetes.io/instance": open5gsName,
		"app.kubernetes.io/name":     strings.ToLower(functionName),
	}
	if protocol == "metrics" {
		labels["app.kubernetes.io/component"] = "metrics"
	}
	var service netv1.Open5GSService
	for _, arg := range args {
		switch v := arg.(type) {
		case netv1.Open5GSService:
			service = v
		}
	}
	serviceType := corev1.ServiceTypeClusterIP
	if service.ServiceType == "NodePort" {
		serviceType = corev1.ServiceTypeNodePort
	}
	if service.ServiceType == "LoadBalancer" {
		serviceType = corev1.ServiceTypeLoadBalancer
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-" + strings.ToLower(functionName) + "-" + protocol,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower(functionName),
			},
			Ports: []corev1.ServicePort{
				{
					Name:       protocol,
					Port:       port,
					Protocol:   l4_protocol,
					TargetPort: intstr.FromString(protocol),
				},
			},
			PublishNotReadyAddresses: true,
		},
	}
}

func CreateDeployment(namespace, open5gsName, componentName, image, configMapName, containerCommand string, ports []corev1.ContainerPort, envVars []corev1.EnvVar, serviceAccountName string) *appsv1.Deployment {
	if serviceAccountName == "" {
		serviceAccountName = "default"
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-" + strings.ToLower(componentName),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower(componentName),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance": open5gsName,
					"app.kubernetes.io/name":     strings.ToLower(componentName),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/instance": open5gsName,
						"app.kubernetes.io/name":     strings.ToLower(componentName),
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					Containers: []corev1.Container{
						{
							Name:  open5gsName + "-" + strings.ToLower(componentName),
							Image: image,
							Args:  []string{containerCommand},
							Ports: ports,
							Env:   envVars,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/opt/open5gs/etc/open5gs/" + strings.ToLower(componentName) + ".yaml",
									SubPath:   strings.ToLower(componentName) + ".yaml",
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								FailureThreshold:    5,
								SuccessThreshold:    1,
								TimeoutSeconds:      5,
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromString("sbi"),
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
								FailureThreshold:    5,
								SuccessThreshold:    1,
								TimeoutSeconds:      1,
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromString("sbi"),
									},
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot: boolPtr(true),
								RunAsUser:    int64Ptr(1001),
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: int64Ptr(1001),
					},
				},
			},
		},
	}
}

func CreateAMFConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration, metrics bool) *corev1.ConfigMap {
	slicesConfig := ""
	metricsConfig := `
  `
	for _, slice := range configuration.Slices {
		slicesConfig += `
      - sd: "` + slice.SD + `"
        sst: ` + slice.SST
	}

	if metrics {
		metricsConfig = `
  metrics:
   server:
   - dev: eth0
     port: 9090
  `
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-amf",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("amf"),
			},
		},
		Data: map[string]string{
			"amf.yaml": `
logger:
  level: info
global:
amf:
  sbi:
    server:
    - dev: eth0
      port: 7777
    client:
      scp:
      - uri: http://` + open5gsName + `-scp-sbi:7777
  ngap:
    server:
    - dev: eth0` +
				metricsConfig +
				`guami:
    -
      amf_id:
        region: ` + configuration.Region + `
        set: ` + configuration.Set + `
      plmn_id:
        mcc: "` + configuration.MCC + `"
        mnc: "` + configuration.MNC + `"
  tai:
    -
      plmn_id:
        mcc: "` + configuration.MCC + `"
        mnc: "` + configuration.MNC + `"
      tac:
      - ` + configuration.TAC + `
  plmn_support:
    -
      plmn_id:
        mcc: "` + configuration.MCC + `"
        mnc: "` + configuration.MNC + `"
      s_nssai:` + slicesConfig + `
  security:
    integrity_order: [NIA2, NIA1, NIA0]
    ciphering_order: [NEA0, NEA1, NEA2]
  network_name:
    full: Gradiant
  amf_name: ` + open5gsName + `-amf
  time:
    t3512:
      value: 540
`,
		},
	}
}

func CreateAUSFConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-ausf",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("ausf"),
			},
		},
		Data: map[string]string{
			"ausf.yaml": `
logger:
  level: info
global:
ausf:
  sbi:
    server:
    - dev: eth0
      port: 7777
    client:
      scp:
      - uri: http://` + open5gsName + `-scp-sbi:7777
`,
		},
	}
}

func CreateBSFConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-bsf",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("bsf"),
			},
		},
		Data: map[string]string{
			"bsf.yaml": `
logger:
  level: info
global:
bsf:
  sbi:
    server:
    - dev: eth0
      port: 7777
    client:
      scp:
      - uri: http://` + open5gsName + `-scp-sbi:7777
`,
		},
	}
}

func CreateNRFConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-nrf",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("nrf"),
			},
		},
		Data: map[string]string{
			"nrf.yaml": `
logger:
  level: info
global:
nrf:
  serving:
    -
      plmn_id:
        mcc: ` + configuration.MCC + `
        mnc: ` + configuration.MNC + `
  sbi:
    server:
    - dev: eth0
      port: 7777
`,
		},
	}
}

func CreateNSSFConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration) *corev1.ConfigMap {
	slicesConfig := ""
	for _, slice := range configuration.Slices {
		slicesConfig += `
        - uri: http://` + open5gsName + `-nrf-sbi:7777
          s_nssai:
            sst: "` + slice.SST + `"
            sd: "` + slice.SD + `"`
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-nssf",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("nssf"),
			},
		},
		Data: map[string]string{
			"nssf.yaml": `
logger:
  level: info
global:
nssf:
  sbi:
    server:
    - dev: eth0
      port: 7777
    client:
      scp:
      - uri: http://` + open5gsName + `-scp-sbi:7777
      nsi:` + slicesConfig + `
`,
		},
	}
}

func CreateSMFConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration, metrics bool) *corev1.ConfigMap {
	metricsConfig := `
  `
	if metrics {
		metricsConfig = `
  metrics:
   server:
   - dev: eth0
     port: 9090`
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-smf",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("smf"),
			},
		},
		Data: map[string]string{
			"smf.yaml": `
logger:
  level: info
global:
smf:
  sbi:
    server:
    - dev: eth0
      port: 7777
    client:
      scp:
      - uri: http://` + open5gsName + `-scp-sbi:7777
  pfcp:
    server:
    - dev: eth0
    client:
      upf:
      - address: ` + open5gsName + `-upf-pfcp` +
				metricsConfig + `
  gtpc:
    server:
    - dev: eth0
  gtpu:
    server:
    - dev: eth0
  session:
    -
      dnn: internet
      gateway: 10.45.0.1
      subnet: 10.45.0.0/16
  dns:
    -
      8.8.8.8
    -
      8.8.4.4
    -
      2001:4860:4860::8888
    -
      2001:4860:4860::8844
  mtu: 1400
  ctf:
    enabled: auto
`,
		},
	}
}

func CreatePCFConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration, metrics bool) *corev1.ConfigMap {
	var metricsConfig string
	if metrics {
		metricsConfig = `
  metrics:
   server:
   - dev: eth0
     port: 9090`
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-pcf",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("pcf"),
			},
		},
		Data: map[string]string{
			"pcf.yaml": `
logger:
  level: info
global:
pcf:
  sbi:
    server:
    - dev: eth0
      port: 7777
    client:
      scp:
      - uri: http://` + open5gsName + `-scp-sbi:7777` +
				metricsConfig,
		},
	}
}

func CreateSCPConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-scp",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("scp"),
			},
		},
		Data: map[string]string{
			"scp.yaml": `
logger:
  level: info
global:
scp:
  sbi:
    server:
    - dev: eth0
      port: 7777
    client:
      nrf:
      - uri: http://` + open5gsName + `-nrf-sbi:7777
`,
		},
	}
}

func CreateUDMConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-udm",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("udm"),
			},
		},
		Data: map[string]string{
			"udm.yaml": `
logger:
  level: info
global:
udm:
  hnet:
  - id: 1
    scheme: 1
    key: /opt/open5gs/etc/open5gs/hnet/curve25519-1.key
  - id: 2
    scheme: 2
    key: /opt/open5gs/etc/open5gs/hnet/secp256r1-2.key
  - id: 3
    scheme: 1
    key: /opt/open5gs/etc/open5gs/hnet/curve25519-3.key
  - id: 4
    scheme: 2
    key: /opt/open5gs/etc/open5gs/hnet/secp256r1-4.key
  - id: 5
    scheme: 1
    key: /opt/open5gs/etc/open5gs/hnet/curve25519-5.key
  - id: 6
    scheme: 2
    key: /opt/open5gs/etc/open5gs/hnet/secp256r1-6.key
  sbi:
    server:
    - dev: eth0
      port: 7777
    client:
      scp:
      - uri: http://` + open5gsName + `-scp-sbi:7777
`,
		},
	}
}

func CreateUDRConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-udr",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("udr"),
			},
		},
		Data: map[string]string{
			"udr.yaml": `
logger:
  level: info
global:
udr:
  sbi:
    server:
    - dev: eth0
      port: 7777
    client:
      scp:
      - uri: http://` + open5gsName + `-scp-sbi:7777
`,
		},
	}
}

func CreateUPFConfigMap(namespace, open5gsName string, configuration netv1.Open5GSConfiguration, metrics bool, gtpuDev string) *corev1.ConfigMap {
	metricsConfig := `
  `
	if metrics {
		metricsConfig = `
  metrics:
   server:
   - dev: eth0
     port: 9090`
	}
	if gtpuDev == "" {
		gtpuDev = "eth0"
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-upf",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("upf"),
			},
		},
		Data: map[string]string{
			"upf.yaml": `
logger:
  level: info
global:
upf:
  pfcp:
    server:
    - dev: eth0
    client:
  gtpu:
    server:
    - dev: ` + gtpuDev + metricsConfig + `
  session:
    -
      dev: ogstun
      dnn: internet
      gateway: 10.45.0.1
      subnet: 10.45.0.0/16
`,
		},
	}
}

func CreateUPFEntrypointConfigMap(namespace, open5gsName string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-upf-entrypoint",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("upf"),
			},
		},
		Data: map[string]string{
			"k8s-entrypoint.sh": `
#!/bin/bash
set -e

echo "Executing k8s customized entrypoint.sh"
echo "Creating net device ogstun"
if grep "ogstun" /proc/net/dev > /dev/null; then
    echo "Warning: Net device ogstun already exists! may you need to set createDev: false";
    exit 1
fi

ip tuntap add name ogstun mode tun
ip link set ogstun up
echo "Setting IP 10.45.0.1 to device ogstun"
ip addr add 10.45.0.1/16 dev ogstun;
sysctl -w net.ipv4.ip_forward=1;
echo "Enable NAT for 10.45.0.0/16 and device ogstun"
iptables -t nat -A POSTROUTING -s 10.45.0.0/16 ! -o ogstun -j MASQUERADE;

$@
`,
		},
	}
}

func CreateUPFDeployment(namespace, open5gsName, image string, envVars []corev1.EnvVar, metrics bool, serviceAccountName string, deploymentAnnotations map[string]string) *appsv1.Deployment {
	var ports []corev1.ContainerPort
	if metrics {
		ports = []corev1.ContainerPort{
			{
				ContainerPort: 8805,
				Name:          "pfcp",
				Protocol:      corev1.ProtocolUDP,
			},
			{
				ContainerPort: 2152,
				Name:          "gtpu",
				Protocol:      corev1.ProtocolUDP,
			},
			{
				ContainerPort: 9090,
				Name:          "metrics",
				Protocol:      corev1.ProtocolTCP,
			},
		}
	} else {
		ports = []corev1.ContainerPort{
			{
				ContainerPort: 8805,
				Name:          "pfcp",
				Protocol:      corev1.ProtocolUDP,
			},
			{
				ContainerPort: 2152,
				Name:          "gtpu",
				Protocol:      corev1.ProtocolUDP,
			},
		}
	}
	if serviceAccountName == "" {
		serviceAccountName = "default"
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-upf",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("upf"),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance": open5gsName,
					"app.kubernetes.io/name":     "upf",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/instance": open5gsName,
						"app.kubernetes.io/name":     "upf",
					},
					Annotations: deploymentAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "open5gs-upf",
							Image: image,
							Args:  []string{"open5gs-upfd"},
							Ports: ports,
							Env:   envVars,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/opt/open5gs/etc/open5gs/upf.yaml",
									SubPath:   "upf.yaml",
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"NET_ADMIN",
									},
								},
								Privileged:   boolPtr(true),
								RunAsNonRoot: boolPtr(false),
								RunAsUser:    int64Ptr(0),
								RunAsGroup:   int64Ptr(0),
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: open5gsName + "-upf",
									},
									DefaultMode: int32Ptr(420),
								},
							},
						},
						{
							Name: "entrypoint",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: open5gsName + "-upf-entrypoint",
									},
									DefaultMode: int32Ptr(511),
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: int64Ptr(1001),
					},
					InitContainers: []corev1.Container{
						{
							Name:  "tun-create",
							Image: image,
							Command: []string{
								"/bin/bash",
								"-c",
								"/k8s-entrypoint.sh",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "entrypoint",
									MountPath: "/k8s-entrypoint.sh",
									SubPath:   "k8s-entrypoint.sh",
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"NET_ADMIN",
									},
								},
								Privileged:   boolPtr(true),
								RunAsNonRoot: boolPtr(false),
								RunAsUser:    int64Ptr(0),
								RunAsGroup:   int64Ptr(0),
							},
						},
					},
				},
			},
		},
	}
}

func CreateWebUIConfigMap(namespace, open5gsName string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-webui",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("webui"),
			},
		},
		Data: map[string]string{
			"add_admin.sh": `#!/bin/bash

set -e

echo "add admin user with password 1423 if no users"

cat << EOF > /tmp/account.js
db = db.getSiblingDB('open5gs')
cursor = db.accounts.find()
if ( cursor.count() == 0 ) {
    db.accounts.insert({ salt: 'f5c15fa72622d62b6b790aa8569b9339729801ab8bda5d13997b5db6bfc1d997', hash: '402223057db5194899d2e082aeb0802f6794622e1cbc47529c419e5a603f2cc592074b4f3323b239ffa594c8b756d5c70a4e1f6ecd3f9f0d2d7328c4cf8b1b766514effff0350a90b89e21eac54cd4497a169c0c7554a0e2cd9b672e5414c323f76b8559bc768cba11cad2ea3ae704fb36abc8abc2619231ff84ded60063c6e1554a9777a4a464ef9cfdfa90ecfdacc9844e0e3b2f91b59d9ff024aec4ea1f51b703a31cda9afb1cc2c719a09cee4f9852ba3cf9f07159b1ccf8133924f74df770b1a391c19e8d67ffdcbbef4084a3277e93f55ac60d80338172b2a7b3f29cfe8a36738681794f7ccbe9bc98f8cdeded02f8a4cd0d4b54e1d6ba3d11792ee0ae8801213691848e9c5338e39485816bb0f734b775ac89f454ef90992003511aa8cceed58a3ac2c3814f14afaaed39cbaf4e2719d7213f81665564eec02f60ede838212555873ef742f6666cc66883dcb8281715d5c762fb236d72b770257e7e8d86c122bb69028a34cf1ed93bb973b440fa89a23604cd3fefe85fbd7f55c9b71acf6ad167228c79513f5cfe899a2e2cc498feb6d2d2f07354a17ba74cecfbda3e87d57b147e17dcc7f4c52b802a8e77f28d255a6712dcdc1519e6ac9ec593270bfcf4c395e2531a271a841b1adefb8516a07136b0de47c7fd534601b16f0f7a98f1dbd31795feb97da59e1d23c08461cf37d6f2877d0f2e437f07e25015960f63', username: 'admin', roles: [ 'admin' ], "__v" : 0})
}
EOF

mongosh $DB_URI /tmp/account.js
rm -f /tmp/account.js
`,
		},
	}
}

func CreateWebUIDeployment(namespace, open5gsName, image string, envVars []corev1.EnvVar, serviceAccountName string, mongoDBVersion string) *appsv1.Deployment {
	if serviceAccountName == "" {
		serviceAccountName = "default"
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-webui",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("webui"),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance": open5gsName,
					"app.kubernetes.io/name":     "webui",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/instance": open5gsName,
						"app.kubernetes.io/name":     "webui",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "open5gs-webui",
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9999,
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: envVars,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "populate",
									MountPath: "/opt/open5gs/etc/open5gs/",
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 600,
								PeriodSeconds:       10,
								FailureThreshold:    5,
								SuccessThreshold:    1,
								TimeoutSeconds:      5,
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromString("http"),
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: 30,
								PeriodSeconds:       5,
								FailureThreshold:    5,
								SuccessThreshold:    1,
								TimeoutSeconds:      1,
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromString("http"),
									},
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot: boolPtr(true),
								RunAsUser:    int64Ptr(999),
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  "init",
							Image: mongoDBVersion,
							Command: []string{
								"/bin/bash",
								"/add_admin.sh",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "DB_URI",
									Value: "mongodb://" + open5gsName + "-mongodb/open5gs",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "populate",
									MountPath: "/add_admin.sh",
									SubPath:   "add_admin.sh",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "populate",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: open5gsName + "-webui",
									},
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: int64Ptr(999),
					},
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
		},
	}
}

func CreateMongoDBConfigMap(namespace, open5gsName string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-mongodb-common-scripts",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("mongodb"),
			},
		},
		Data: map[string]string{
			"ping-mongodb.sh": `
#!/bin/bash
mongosh  $TLS_OPTIONS --port $MONGODB_PORT_NUMBER --eval "db.adminCommand('ping')"
`,
			"readiness-probe.sh": `
#!/bin/bash
# Run the proper check depending on the version
[[ $(mongod -version | grep "db version") =~ ([0-9]+\.[0-9]+\.[0-9]+) ]] && VERSION=${BASH_REMATCH[1]}
. /opt/bitnami/scripts/libversion.sh
VERSION_MAJOR="$(get_sematic_version "$VERSION" 1)"
VERSION_MINOR="$(get_sematic_version "$VERSION" 2)"
VERSION_PATCH="$(get_sematic_version "$VERSION" 3)"
if [[ ( "$VERSION_MAJOR" -ge 5 ) || ( "$VERSION_MAJOR" -ge 4 && "$VERSION_MINOR" -ge 4 && "$VERSION_PATCH" -ge 2 ) ]]; then
    mongosh $TLS_OPTIONS --port $MONGODB_PORT_NUMBER --eval 'db.hello().isWritablePrimary || db.hello().secondary' | grep -q 'true'
else
    mongosh  $TLS_OPTIONS --port $MONGODB_PORT_NUMBER --eval 'db.isMaster().ismaster || db.isMaster().secondary' | grep -q 'true'
fi
`,
			"startup-probe.sh": `
#!/bin/bash
mongosh  $TLS_OPTIONS --port $MONGODB_PORT_NUMBER --eval 'db.hello().isWritablePrimary || db.hello().secondary' | grep -q 'true'
`,
		},
	}
}

func CreateMongoDBService(namespace, open5gsName string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-mongodb",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("mongodb"),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/component": "mongodb",
				"app.kubernetes.io/instance":  open5gsName,
				"app.kubernetes.io/name":      "mongodb",
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "mongodb",
					Port:       27017,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("mongodb"),
				},
			},
			PublishNotReadyAddresses: true,
		},
	}
}

func CreateMongoDBPVC(namespace, open5gsName string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-mongodb",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("mongodb"),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("8Gi"),
				},
			},
		},
	}
}

func CreateMongoDBDeployment(namespace, open5gsName, image string, envVars []corev1.EnvVar, serviceAccountName string) *appsv1.Deployment {
	if serviceAccountName == "" {
		serviceAccountName = "default"
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-mongodb",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower("mongodb"),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": "mongodb",
					"app.kubernetes.io/instance":  open5gsName,
					"app.kubernetes.io/name":      "mongodb",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/component": "mongodb",
						"app.kubernetes.io/instance":  open5gsName,
						"app.kubernetes.io/name":      "mongodb",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "mongodb",
							Image: image,
							Env:   envVars,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 27017,
									Name:          "mongodb",
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									MountPath: "/bitnami/mongodb",
								},
								{
									Name:      "common-scripts",
									MountPath: "/bitnami/scripts",
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/bash",
											"-c",
											"/bitnami/scripts/ping-mongodb.sh",
										},
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       20,
								FailureThreshold:    6,
								TimeoutSeconds:      10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/bash",
											"-c",
											"/bitnami/scripts/readiness-probe.sh",
										},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
								FailureThreshold:    6,
								TimeoutSeconds:      5,
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot: boolPtr(true),
								RunAsUser:    int64Ptr(999),
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "datadir",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: open5gsName + "-mongodb",
								},
							},
						},
						{
							Name: "common-scripts",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: open5gsName + "-mongodb-common-scripts",
									},
									DefaultMode: int32Ptr(360),
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: int64Ptr(999),
					},
				},
			},
		},
	}
}

func CreateServiceMonitor(namespace, open5gsName string, function string) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-" + function,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/component": "metrics",
				"app.kubernetes.io/instance":  open5gsName,
				"app.kubernetes.io/name":      function,
				"release":                     "prometheus",
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": "metrics",
					"app.kubernetes.io/instance":  open5gsName,
					"app.kubernetes.io/name":      function,
				},
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{namespace},
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "metrics",
				},
			},
		},
	}
}

func CreateServiceAccount(namespace, open5gsName, componentName string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      open5gsName + "-" + strings.ToLower(componentName),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance": open5gsName,
				"app.kubernetes.io/name":     strings.ToLower(componentName),
			},
		},
		AutomountServiceAccountToken: boolPtr(true),
	}
}

func int32Ptr(i int32) *int32 { return &i }
func int64Ptr(i int64) *int64 { return &i }
func boolPtr(b bool) *bool    { return &b }

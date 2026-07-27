/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package controller

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestCreateUPFDeploymentDefaultUnchanged(t *testing.T) {
	dep := CreateUPFDeployment("default", "test", "docker.io/gradiant/open5gs:2.7.5", nil, true, "", nil, false)

	main := dep.Spec.Template.Spec.Containers[0]
	if main.SecurityContext.Privileged == nil || !*main.SecurityContext.Privileged {
		t.Error("expected main container Privileged=true when unprivileged=false")
	}
	if main.SecurityContext.RunAsUser == nil || *main.SecurityContext.RunAsUser != 0 {
		t.Error("expected main container RunAsUser=0 when unprivileged=false")
	}

	if len(main.Command) != 0 {
		t.Error("expected no Command override on main container when unprivileged=false")
	}
	if len(main.Args) != 1 || main.Args[0] != "open5gs-upfd" {
		t.Error("expected main container Args to be unchanged (open5gs-upfd) when unprivileged=false")
	}

	init := dep.Spec.Template.Spec.InitContainers[0]
	if init.SecurityContext.Privileged == nil || !*init.SecurityContext.Privileged {
		t.Error("expected init container Privileged=true when unprivileged=false")
	}

	for _, v := range dep.Spec.Template.Spec.Volumes {
		if v.Name == "tun-device" {
			t.Error("did not expect a tun-device volume when unprivileged=false")
		}
	}
	if dep.Spec.Template.Spec.SecurityContext.Sysctls != nil {
		t.Error("did not expect pod-level sysctls when unprivileged=false")
	}
}

func TestCreateUPFDeploymentUnprivileged(t *testing.T) {
	dep := CreateUPFDeployment("default", "test", "docker.io/gradiant/open5gs:2.7.5", nil, true, "", nil, true)

	main := dep.Spec.Template.Spec.Containers[0]
	if main.SecurityContext.Privileged == nil || *main.SecurityContext.Privileged {
		t.Error("expected main container Privileged=false when unprivileged=true")
	}
	if main.SecurityContext.RunAsNonRoot == nil || !*main.SecurityContext.RunAsNonRoot {
		t.Error("expected main container RunAsNonRoot=true when unprivileged=true")
	}
	if main.SecurityContext.RunAsUser == nil || *main.SecurityContext.RunAsUser != 1001 {
		t.Error("expected main container RunAsUser=1001 when unprivileged=true")
	}
	if len(main.SecurityContext.Capabilities.Add) != 0 {
		t.Error("expected main container to add zero capabilities when unprivileged=true")
	}
	if len(main.SecurityContext.Capabilities.Drop) != 1 || main.SecurityContext.Capabilities.Drop[0] != "ALL" {
		t.Error("expected main container to drop ALL capabilities when unprivileged=true")
	}
	if main.SecurityContext.SELinuxOptions != nil {
		t.Error("expected no explicit seLinuxOptions on main container - left to the bound SCC's own default")
	}
	if len(main.Command) != 1 || main.Command[0] != "/opt/open5gs/bin/open5gs-upfd" {
		t.Error("expected main container Command to bypass the image's own entrypoint wrapper when unprivileged=true")
	}
	if len(main.Args) != 2 || main.Args[0] != "-c" || main.Args[1] != "/opt/open5gs/etc/open5gs/upf.yaml" {
		t.Error("expected main container Args to pass the config path directly when unprivileged=true")
	}

	init := dep.Spec.Template.Spec.InitContainers[0]
	if init.SecurityContext.Privileged == nil || *init.SecurityContext.Privileged {
		t.Error("expected init container Privileged=false when unprivileged=true")
	}
	if init.SecurityContext.RunAsUser == nil || *init.SecurityContext.RunAsUser != 0 {
		t.Error("expected init container RunAsUser=0 (root) when unprivileged=true")
	}
	if len(init.SecurityContext.Capabilities.Add) != 1 || init.SecurityContext.Capabilities.Add[0] != "NET_ADMIN" {
		t.Error("expected init container to add only NET_ADMIN when unprivileged=true")
	}
	if init.SecurityContext.SELinuxOptions != nil {
		t.Error("expected no explicit seLinuxOptions on init container - left to the bound SCC's own default")
	}

	foundTunVolume := false
	for _, v := range dep.Spec.Template.Spec.Volumes {
		if v.Name == "tun-device" {
			foundTunVolume = true
			if v.HostPath == nil || v.HostPath.Path != "/dev/net/tun" {
				t.Error("expected tun-device volume to be a hostPath mount of /dev/net/tun")
			}
			if v.HostPath.Type == nil || *v.HostPath.Type != corev1.HostPathCharDev {
				t.Error("expected tun-device hostPath type to be CharDevice")
			}
		}
	}
	if !foundTunVolume {
		t.Error("expected a tun-device volume when unprivileged=true")
	}

	foundMainMount := false
	for _, m := range main.VolumeMounts {
		if m.Name == "tun-device" && m.MountPath == "/dev/net/tun" {
			foundMainMount = true
		}
	}
	if !foundMainMount {
		t.Error("expected main container to mount tun-device at /dev/net/tun")
	}

	sysctls := dep.Spec.Template.Spec.SecurityContext.Sysctls
	if len(sysctls) != 1 || sysctls[0].Name != "net.ipv4.ip_forward" || sysctls[0].Value != "1" {
		t.Error("expected pod-level sysctls to set net.ipv4.ip_forward=1")
	}
}

func TestCreateUPFEntrypointConfigMapUnprivileged(t *testing.T) {
	defaultCM := CreateUPFEntrypointConfigMap("default", "test", false)
	unprivCM := CreateUPFEntrypointConfigMap("default", "test", true)

	defaultScript := defaultCM.Data["k8s-entrypoint.sh"]
	unprivScript := unprivCM.Data["k8s-entrypoint.sh"]

	if !strings.Contains(defaultScript, "sysctl -w net.ipv4.ip_forward=1") {
		t.Error("expected default script to set ip_forward via sysctl -w in-container")
	}
	if strings.Contains(unprivScript, "sysctl -w") {
		t.Error("unprivileged script must not attempt sysctl -w (blocked by read-only /proc/sys)")
	}
	if !strings.Contains(unprivScript, "ip tuntap add name ogstun mode tun user 1001") {
		t.Error("expected unprivileged script to create a persistent tun device owned by uid 1001")
	}
	if !strings.Contains(unprivScript, "iptables -t nat -A POSTROUTING") {
		t.Error("expected unprivileged script to still add the NAT rule")
	}
}

/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package controller

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	netv1 "github.com/gradiant/open5gs-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	defaultOpen5GSImage = "docker.io/gradiant/open5gs:2.8.0"
	defaultWebUIImage   = "docker.io/gradiant/open5gs-webui:2.8.0"
	defaultMongoDBImage = "docker.io/library/mongo:8.0.34"
	legacyMongoDBImage  = "bitnami/mongodb:latest"
)

func runtimeSeccomp() *corev1.SeccompProfile {
	return &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}
}

func workloadSecurityContext(uid int64) *corev1.SecurityContext {
	return &corev1.SecurityContext{
		RunAsUser: int64Ptr(uid), RunAsGroup: int64Ptr(uid), RunAsNonRoot: boolPtr(true),
		AllowPrivilegeEscalation: boolPtr(false), SeccompProfile: runtimeSeccomp(),
		Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
	}
}

func coreImageUID(image string) int64 {
	if image == defaultOpen5GSImage {
		return 999
	}
	return 1001
}

func workloadSpec(spec netv1.Open5GSSpec, component string) netv1.Open5GSFunction {
	return map[string]netv1.Open5GSFunction{
		"AMF": spec.AMF, "AUSF": spec.AUSF, "BSF": spec.BSF, "MongoDB": spec.MongoDB,
		"NRF": spec.NRF, "NSSF": spec.NSSF, "PCF": spec.PCF, "SCP": spec.SCP,
		"SMF": spec.SMF, "UDM": spec.UDM, "UDR": spec.UDR, "UPF": spec.UPF, "WebUI": spec.WebUI,
	}[component]
}

func setWorkloadDefaults(pod *corev1.PodSpec, component string, spec netv1.Open5GSFunction) {
	main := &pod.Containers[0]
	main.Resources = workloadResources(component)
	if spec.Resources != nil {
		main.Resources = corev1.ResourceRequirements{Requests: spec.Resources.Requests.DeepCopy(), Limits: spec.Resources.Limits.DeepCopy()}
	}
	for i := range pod.InitContainers {
		pod.InitContainers[i].Resources = workloadResources("init")
	}
	if component == "UPF" {
		main.ReadinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{
				Command: []string{"sh", "-c", "awk " + pfcpSocketCheck + " /proc/net/udp /proc/net/udp6"},
			}},
			PeriodSeconds: 5, TimeoutSeconds: 2, FailureThreshold: 3,
		}
		main.LivenessProbe = main.ReadinessProbe.DeepCopy()
		main.LivenessProbe.PeriodSeconds = 10
	}
	if main.ReadinessProbe != nil {
		main.StartupProbe = main.ReadinessProbe.DeepCopy()
		main.StartupProbe.InitialDelaySeconds = 0
		main.StartupProbe.PeriodSeconds = 5
		main.StartupProbe.FailureThreshold = 60
		main.StartupProbe.SuccessThreshold = 1
	}
}

const pfcpSocketCheck = `'$2 ~ /:2265$/ { found=1 } END { exit !found }'`

func workloadResources(component string) corev1.ResourceRequirements {
	cpu, memory, limit := "50m", "64Mi", "512Mi"
	switch component {
	case "MongoDB":
		cpu, memory, limit = "100m", "256Mi", "1Gi"
	case "WebUI":
		memory, limit = "128Mi", "1Gi"
	}
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu), corev1.ResourceMemory: resource.MustParse(memory)},
		Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse(limit)},
	}
}

func waitForMongoContainer(name, image string) corev1.Container {
	return corev1.Container{
		Name: "wait-for-mongodb", Image: image,
		SecurityContext: workloadSecurityContext(999),
		Command: []string{"bash", "-ec", `deadline=$((SECONDS+300))
while (( SECONDS < deadline )); do
    remaining=$((deadline-SECONDS))
    (( remaining > 10 )) && remaining=10
    if timeout -k 1 "${remaining}s" mongosh "$DB_URI" --quiet --norc --eval 'if (db.adminCommand({ping: 1}).ok !== 1) quit(1)' >/dev/null 2>&1; then
        exit 0
    fi
    sleep 2
done
echo "Timed out waiting for MongoDB" >&2
exit 1`},
		Env: []corev1.EnvVar{{Name: "DB_URI", Value: fmt.Sprintf("mongodb://%s-mongodb/open5gs", name)}},
	}
}

func isLegacyMongoDBImage(image string) bool {
	name, _, _ := strings.Cut(image, "@")
	if colon := strings.LastIndexByte(name, ':'); colon > strings.LastIndexByte(name, '/') {
		name = name[:colon]
	}
	return strings.HasSuffix("/"+name, "/bitnami/mongodb") ||
		strings.HasSuffix("/"+name, "/bitnamilegacy/mongodb")
}

// Existing Bitnami data is kept on its original claim. Switching layouts
// requires a separate PVC populated by the administrator before cutover.
func (r *Open5GSReconciler) selectMongoDBStorage(ctx context.Context, core *netv1.Open5GS) error {
	spec := &core.Spec
	legacyClaim := core.Name + "-mongodb"
	key := client.ObjectKey{Namespace: core.Namespace, Name: legacyClaim}
	err := r.Get(ctx, key, &corev1.PersistentVolumeClaim{})
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	legacyExists := err == nil
	if spec.MongoDBVersion == "" {
		spec.MongoDBVersion = defaultMongoDBImage
		if legacyExists && spec.MongoDBVolumeClaim == "" {
			spec.MongoDBVersion = legacyMongoDBImage
			deployment := &appsv1.Deployment{}
			if err := r.Get(ctx, key, deployment); err == nil && hasOwnerReference(deployment, core) {
				for _, c := range deployment.Spec.Template.Spec.Containers {
					if c.Name == "mongodb" && isLegacyMongoDBImage(c.Image) {
						spec.MongoDBVersion = c.Image
					}
				}
			} else if client.IgnoreNotFound(err) != nil {
				return err
			}
		}
	}
	if isLegacyMongoDBImage(spec.MongoDBVersion) {
		if spec.MongoDBVolumeClaim != "" {
			return fmt.Errorf("mongoDBVolumeClaim requires the official MongoDB layout")
		}
		return nil
	}
	if spec.MongoDBVolumeClaim == legacyClaim || (legacyExists && spec.MongoDBVolumeClaim == "") {
		return fmt.Errorf("MongoDB migration requires a separate restored PVC in spec.mongoDBVolumeClaim")
	}
	if spec.MongoDBVolumeClaim != "" {
		key.Name = spec.MongoDBVolumeClaim
		claim := &corev1.PersistentVolumeClaim{}
		if err := r.Get(ctx, key, claim); err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("MongoDB destination PVC %q does not exist", key.Name)
			}
			return err
		}
		if !claim.DeletionTimestamp.IsZero() {
			return fmt.Errorf("MongoDB destination PVC %q is being deleted", key.Name)
		}
	}
	return nil
}

func configureOfficialMongoDB(deployment *appsv1.Deployment, claimName string) {
	pod := &deployment.Spec.Template.Spec
	pod.SecurityContext.FSGroupChangePolicy = new(corev1.PodFSGroupChangePolicy)
	*pod.SecurityContext.FSGroupChangePolicy = corev1.FSGroupChangeOnRootMismatch
	main := &pod.Containers[0]
	main.Env = nil
	main.Command = []string{"mongod"}
	main.Args = []string{"--bind_ip_all", "--dbpath", "/data/db"}
	main.SecurityContext = workloadSecurityContext(999)
	main.VolumeMounts = []corev1.VolumeMount{{Name: "datadir", MountPath: "/data/db"}}
	pod.Volumes = []corev1.Volume{{Name: "datadir", VolumeSource: corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
	}}}
	main.ReadinessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{
			Command: []string{"mongosh", "--quiet", "--norc", "--eval", `if (db.adminCommand({ping: 1}).ok !== 1) quit(1)`},
		}},
		PeriodSeconds: 5, TimeoutSeconds: 5, FailureThreshold: 3,
	}
	main.LivenessProbe = main.ReadinessProbe.DeepCopy()
	main.LivenessProbe.PeriodSeconds = 10
}

const webUIBootstrap = `
const mongoose = require("mongoose");
const Account = require("./server/models/account");

(async () => {
    try {
        await mongoose.connect(process.env.DB_URI, { useNewUrlParser: true, useUnifiedTopology: true, serverSelectionTimeoutMS: 300000 });
        if (await Account.exists({ roles: "admin" })) return;
        const username = process.env.WEBUI_ADMIN_USERNAME;
        const password = process.env.WEBUI_ADMIN_PASSWORD;
        if (!username || !password || username.trim() !== username || password.length < 12) {
            console.error("Configure spec.webuiAdminSecret with a username and a password of at least 12 characters");
            process.exitCode = 1;
            return;
        }
        try {
            await Account.register(new Account({ username, roles: ["admin"] }), password);
        } catch (error) {
            if (!(await Account.exists({ roles: "admin" }))) throw error;
        }
    } catch {
        console.error("WebUI bootstrap failed; check MongoDB availability and admin configuration");
        process.exitCode = 1;
    } finally {
        await mongoose.disconnect();
    }
})();
`

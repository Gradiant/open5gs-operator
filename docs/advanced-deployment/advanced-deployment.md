---
title: Advanced Open5GS deployment
---

# Configure Open5GS workloads

This example enables WebUI, creates two slices, and updates CPU and memory on a running instance. Use a separate namespace so it can run alongside the [basic example](../basic-deployment/basic-deployment.md).

Download [the initial manifest](open5gs-advanced-deployment.yaml) and [the modified manifest](open5gs-advanced-deployment-modified.yaml). Both use the same Open5GS resource and Secret name.

### Initial Open5GS instance

```yaml
{% include_relative open5gs-advanced-deployment.yaml %}
```

## Enable WebUI

Create an administrator Secret before starting WebUI on a fresh database. It needs `username` and `password` keys. Use a unique password of at least 12 characters and keep it available for signing in.

```bash
kubectl create namespace open5gs-advanced
read -r -s -p "WebUI password: " WEBUI_PASSWORD
printf "\n"
printf "%s" "$WEBUI_PASSWORD" |
  kubectl -n open5gs-advanced create secret generic webui-admin \
    --from-literal=username=open5gs-admin --from-file=password=/dev/stdin
unset WEBUI_PASSWORD

kubectl -n open5gs-advanced apply -f open5gs-advanced-deployment.yaml
```

Wait for the WebUI Deployment to appear, then wait for it to be ready and forward its port.

```bash
kubectl -n open5gs-advanced rollout status deployment/open5gs-sample-webui --timeout=330s
kubectl -n open5gs-advanced port-forward deployment/open5gs-sample-webui 9999:9999
```

Open <http://localhost:9999> and sign in with the new account. On an existing database with an administrator, bootstrap keeps the existing accounts. Rotate an old default password through WebUI.

## Update a running deployment

In another terminal, apply the modified manifest. It changes UPF CPU and memory settings and switches AUSF to the namespace default ServiceAccount.

### Modified Open5GS instance

```yaml
{% include_relative open5gs-advanced-deployment-modified.yaml %}
```

```bash
kubectl -n open5gs-advanced apply -f open5gs-advanced-deployment-modified.yaml
kubectl -n open5gs-advanced wait --for=jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}'=75m \
  deployment/open5gs-sample-upf --timeout=90s
kubectl -n open5gs-advanced rollout status deployment/open5gs-sample-upf --timeout=330s
kubectl -n open5gs-advanced wait --for=jsonpath='{.spec.template.spec.serviceAccountName}'=default \
  deployment/open5gs-sample-ausf --timeout=90s
kubectl -n open5gs-advanced get deployment open5gs-sample-ausf \
  -o jsonpath='{.spec.template.spec.automountServiceAccountToken}{"\n"}'
```

The last command prints `false`. By default, workloads use one dedicated ServiceAccount per component and omit its API token. With `serviceAccount: false`, the Pod uses the namespace default account with token mounting also disabled. Fields under `spec.<component>.resources` replace the default requests and limits for that component's main container.

## Configure the UPF without privileged mode

Set `spec.upf.unprivileged: true` on a cluster configured for that mode. The UPF process runs as UID 1001 with all capabilities dropped. An init container with `NET_ADMIN` prepares the tunnel and NAT. Nodes need `/dev/net/tun` and the kubelet must allow the `net.ipv4.ip_forward` sysctl. OpenShift also requires a compatible Security Context Constraint. This mode does not satisfy Pod Security Restricted.

The [UERANSIM demo](../complete-demo-ueransim/complete-demo-ueransim.md) includes the Kind setup and tests HTTP traffic through the UPF.

## Other configuration

- `spec.open5gsImage` and `spec.webuiImage` override the default Gradiant 2.8.0 images.
- `spec.mongoDBVersion` overrides the MongoDB image. The default for fresh installations is `docker.io/library/mongo:8.0.34`. Follow [MongoDB migration](../mongodb-migration/mongodb-migration.md) for a Bitnami deployment with data.
- `spec.<component>.metrics` and `serviceMonitor` enable metrics and ServiceMonitor creation. Install the ServiceMonitor CRD before enabling it.
- `spec.configuration.slices` sets the supported slices. The operator applies changes to the affected Pods, which can cause an interruption. The [two-slice example](../complete-demo-ueransim/complete-demo-ueransim.md#extension-add-a-second-slice) verifies a second slice after updating this field.

To update a managed subscriber, apply [the original user](open5gs-user.yaml) and then [the updated user](open5gs-user-modified.yaml) in the same namespace. This changes the subscriber from SST 1 and SD `111111` to SST 2 and SD `222222`. Both examples use the `internet` DNN.

### Original subscriber

```yaml
{% include_relative open5gs-user.yaml %}
```

### Modified subscriber

```yaml
{% include_relative open5gs-user-modified.yaml %}
```

```bash
kubectl -n open5gs-advanced apply -f open5gs-user.yaml
kubectl -n open5gs-advanced apply -f open5gs-user-modified.yaml
kubectl -n open5gs-system logs deployment/open5gs-operator-controller-manager --tail=50
```

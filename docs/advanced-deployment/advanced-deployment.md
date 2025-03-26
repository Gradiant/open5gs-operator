# Advanced Deployment of Open5GS on Kubernetes

This tutorial provides an in-depth guide on configuring the `Open5GS` Custom Resource Definition (CRD) used by the Kubernetes operator to deploy and manage Open5GS. It covers configurable fields, default values, and customization options tailored to specific requirements. Additionally, it includes an example of modifying the manifest and analyzing its impact on the deployment. ServiceMonitors are is used in this tutorial so if the ServiceMonitor CRD is not installed in your cluster, you can ignore the references to them.


---

## 1. Complete Open5GS CRD Manifest

The following manifest defines a comprehensive instance of Open5GS, incorporating all components and default configurations:

```yaml
apiVersion: net.gradiant.org/v1
kind: Open5GS
metadata:
  labels:
    app.kubernetes.io/name: open5gs-operator
    app.kubernetes.io/managed-by: kustomize
  name: open5gs-sample
spec:
  open5gsImage: "docker.io/gradiant/open5gs:2.7.4"
  webuiImage: "docker.io/gradiant/open5gs-webui:2.7.4"
  mongoDBVersion: "5.0.10-debian-11-r3"
  amf:
    enabled: true
    serviceAccount: true
    metrics: true
    serviceMonitor: true
    service:
      - name: ngap
        serviceType: ClusterIP
  ausf:
    enabled: true
    serviceAccount: true
  bsf:
    enabled: true
    serviceAccount: true
  mongoDB:
    enabled: true
    serviceAccount: true
  nrf:
    enabled: true
    serviceAccount: true
  nssf:
    enabled: true
    serviceAccount: true
  pcf:
    enabled: true
    metrics: true
    serviceMonitor: true
    serviceAccount: true
  smf:
    enabled: true
    metrics: true
    serviceMonitor: true
    serviceAccount: true
    service:
      - name: pfcp
        serviceType: ClusterIP
  upf:
    enabled: true
    metrics: true
    serviceMonitor: true
    serviceAccount: true
    service:
      - name: pfcp
        serviceType: ClusterIP
      - name: gtpu
        serviceType: ClusterIP
  webui:
    enabled: true
    serviceAccount: true
  configuration:
    mcc: "999"
    mnc: "70"
    tac: "0001"
    region: "2"
    set: "1"
    slices:
      - sst: "1"
        sd: "0x111111"
      - sst: "2"
        sd: "0x222222"
```
<a href="https://gradiant.github.io/open5gs-operator/docs/advanced-deployment/open5gs-advanced-deployment.yaml" class="download-button" download>Download open5gs-advanced-deployment.yaml</a>

Apply the manifest to the cluster:
```bash
kubectl apply -f open5gs-advanced-deployment.yaml
```

### 1.1 Explanation of Configurable Fields

- **`open5gsImage` & `webuiImage`**: Define the Open5GS and WebUI image versions. Defaults to `docker.io/gradiant/open5gs:2.7.4`.
- **`mongoDBVersion`**: Specifies the MongoDB version used. Default is `5.0.10-debian-11-r3`.
- **`enabled`**: Activates or deactivates an Open5GS component. Essential components are enabled by default.
- **`serviceAccount`**: Determines whether a component has a dedicated Kubernetes Service Account.
- **`metrics` & `serviceMonitor`**: Enable Prometheus monitoring. Defaults: `metrics` enabled for AMF, PCF, SMF, and UPF; `serviceMonitor` disabled.
- **`serviceType`**: Defines Kubernetes service exposure (`ClusterIP`, `NodePort`, `LoadBalancer`). Default: `ClusterIP`.
- **`slices`**: Defines network slices (`sst` and `sd`) for user segmentation. If omitted, Open5GS uses a default slice.



## 2. Applying the Manifest with Modifications

We will modify some fields to observe their impact on the deployment:

```yaml
apiVersion: net.gradiant.org/v1
kind: Open5GS
metadata:
  labels:
    app.kubernetes.io/name: open5gs-operator
    app.kubernetes.io/managed-by: kustomize
  name: open5gs-sample
spec:
  open5gsImage: "docker.io/gradiant/open5gs:2.7.4"
  webuiImage: "docker.io/gradiant/open5gs-webui:2.7.0"
  mongoDBVersion: "5.0.10-debian-11-r3"
  amf:
    enabled: true
    serviceAccount: true
    metrics: true
    serviceMonitor: false
    service:
      - name: ngap
        serviceType: ClusterIP
  ausf:
    enabled: true
    serviceAccount: false
  bsf:
    enabled: false
  mongoDB:
    enabled: true
    serviceAccount: true
  nrf:
    enabled: true
    serviceAccount: true
  nssf:
    enabled: true
    serviceAccount: true
  pcf:
    enabled: true
    metrics: true
    serviceMonitor: true
    serviceAccount: true
  smf:
    enabled: true
    metrics: false
    serviceMonitor: true
    serviceAccount: true
    service:
      - name: pfcp
        serviceType: LoadBalancer
  upf:
    enabled: true
    metrics: true
    serviceMonitor: true
    serviceAccount: true
    service:
      - name: pfcp
        serviceType: NodePort
      - name: gtpu
        serviceType: ClusterIP
  webui:
    enabled: true
    serviceAccount: true
  configuration:
    mcc: "999"
    mnc: "70"
    tac: "0001"
    slices:
      - sst: "1"
        sd: "0x111111"
      - sst: "2"
        sd: "0x222222"
```
<a href="https://gradiant.github.io/open5gs-operator/docs/advanced-deployment/open5gs-advanced-deployment-modified.yaml" class="download-button" download>Download open5gs-advanced-deployment-modified.yaml</a>

Apply the manifest to the cluster:
```bash
kubectl apply -f open5gs-advanced-deployment-modified.yaml
```


### 2.1 Changes Made

- **Changed WebUI image version to 2.7.0**.
- **Disabled ServiceAccount for AUSF**.
- **Disabled ServiceMonitor for AMF**.
- **Disabled BSF**.
- **Disabled metrics for SMF**.
- **Changed SMF service type to LoadBalancer**.
- **Exposed pfcp in UPF as NodePort**.
- **Ommited `region` and `set` fields** in the configuration (set to default values).

As the slice configuration was not modified (only some values were omitted), the default slices will be used thus no changes will be observed in the deployment. If the slice configuration was modified, the changes would be reflected in the deployment by modifiying the NSSF and AMF ConfigMaps and restarting the pods in order to apply the changes.

## 3. Verifying Changes

To observe the effects of the changes in the cluster:

- Watch pods restart after applying deployment changes
```bash
kubectl get pods -w
```

- Check service type changes (exposing LoadBalancer and NodePort) and disabled metric services
```bash
kubectl get services
```

- Verify that service accounts have been removed or modified
```bash
kubectl get serviceaccounts
```

- Check the WebUI version
```bash
kubectl describe deployment open5gs-sample-webui | grep Image
```

- Check the applied ServiceMonitors
```bash
kubectl get servicemonitors
```

---

## 4. Managing Open5GS Users
- **Managed vs. Unmanaged Users**:
  - **Managed Users**: Defined in an `Open5GSUser` CRD. The operator ensures consistency between desired and actual configurations.
  - **Unmanaged Users**: Users manually added via WebUI or external scripts. The operator does not modify them.

### 4.1 Understanding User Fields

- **`imsi`**: International Mobile Subscriber Identity, a unique identifier for the user.
- **`key`**: Authentication key used in the AKA (Authentication and Key Agreement) process.
- **`opc`**: Operator key for authentication.
- **`sd`**: Slice Differentiator, used in network slicing.
- **`sst`**: Slice Service Type, defining the service category.
- **`apn`**: Access Point Name, specifies the network to which the UE connects.
- **`open5gs`**: Specifies the Open5GS instance the user belongs to. Name is needed but namespace is optional (defaults to actual namespace).

### 4.2 Adding a New User
Create a new `Open5GSUser` CRD with the following manifest:
```yaml
apiVersion: net.gradiant.org/v1
kind: Open5GSUser
metadata:
  name: open5gsuser-sample
spec:
  imsi: "999700000000001"
  key: "465B5CE8B199B49FAA5F0A2EE238A6BC"
  opc: "E8ED289DEBA952E4283B54E88E6183CA"
  sd: "111111"
  sst: "1"
  apn: "internet"
  open5gs:
    name: "open5gs-sample"
```
<a href="https://gradiant.github.io/open5gs-operator/docs/advanced-deployment/open5gs-user.yaml" class="download-button" download>Download open5gs-user.yaml</a>

Apply the manifest:
```bash
kubectl apply -f open5gs-user.yaml
```

### 4.3 Modifying an Existing User
To update user values, modify the `sd`, `sst`, or `apn` fields in the CRD:
```yaml
spec:
  sd: "222222"
  sst: "2"
  apn: "enterprise"
```

The new manifest should look like this:
```yaml
apiVersion: net.gradiant.org/v1
kind: Open5GSUser
metadata:
  name: open5gsuser-sample
spec:
  imsi: "999700000000001"
  key: "465B5CE8B199B49FAA5F0A2EE238A6BC"
  opc: "E8ED289DEBA952E4283B54E88E6183CA"
  sd: "222222"
  sst: "2"
  apn: "enterprise"
  open5gs:
    name: "open5gs-sample"
```
<a href="https://gradiant.github.io/open5gs-operator/docs/advanced-deployment/open5gs-user-modified.yaml" class="download-button" download>Download open5gs-user-modified.yaml</a>


Apply the changes:
```bash
kubectl apply -f open5gs-user-modified.yaml
```

---

## 5 Checking Operator Logs
To verify the operator's actions, check the logs for any errors or warnings:
```bash
kubectl logs deployments/open5gs-operator-controller-manager 
```

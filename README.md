# Open5GS Kubernetes Operator

## What is it?

The **Open5GS Kubernetes Operator** is a custom Kubernetes operator designed to automate the deployment, configuration, and lifecycle management of Open5GS and its subscribers in a declarative way. It uses two CRDs (Custom Resource Definitions): one for managing Open5GS deployments and another for managing Open5GS users, allowing efficient and automated core and subscriber management.

## Features

### Open5GS Deployment and Reconfiguration

The operator automates the deployment and reconfiguration of Open5GS instances in a Kubernetes cluster. It allows you to define the desired state of an Open5GS deployment in a declarative way, and the operator will ensure that the actual state of the deployment matches the desired state. This includes enabling/disabling components, configuring network slices, and defining the parameters of the Open5GS deployment. Any drift between the actual and desired state will be detected and corrected automatically by the operator. Note that the operator will restart the neccesary pods to apply the changes and that may cause a service interruption.

### Multi-Namespace Support

The operator handles multiple Open5GS deployments across different Kubernetes namespaces, ensuring resource isolation. It can also manage several Open5GS deployments within the same namespace, allowing independent management of each Open5GS instance.

### Open5GS Users Management

The operator provides full management of Open5GS subscribers, including configuration of network slices and the target Open5GS deployment to which they should be assigned. It distinguishes between **Managed Users** and **Unmanaged Users**:

- **Managed Users**: These are users whose IMSI is defined in a CR (Custom Resource). The operator controls their configuration, and any discrepancy between the actual state and the desired state in the CR will be detected as drift and corrected automatically, ensuring the configuration always aligns with the declarative source of truth.
  
- **Unmanaged Users**: These users are not controlled by the operator and are created externally (e.g., via scripts that directly modify the database or the Open5GS WebUI). Unmanaged users will not be altered by the operator, allowing compatibility with external tools and temporary deployments that don't need strict management by the operator.

## Development Requirements

- **Operator SDK**: OperatorSDK 1.37.0 version
- **Go**: go 1.23.3 version

## How to Install

To install by using Helm, you can use the Helm chart provided in the `charts` directory or the open5gs-operator-1.0.0.tgz file. The chart is also available in the Gradiant Charts repository.
```bash
helm install open5gs-operator oci://registry-1.docker.io/gradiantcharts/open5gs-operator --version 1.0.0
```

To uninstall the operator, run:
```bash
helm uninstall open5gs-operator
```

For the other installation options, you can follow [this guide](https://gradiant.github.io/open5gs-operator/docs/installation-options/installation-options.html).

## How to Use

### Create an Open5GS Deployment

1. Create a deployment configuration file for Open5GS. Here’s a basic example (the configuration missing will be set to default values):

    ``` yaml
    apiVersion: net.gradiant.org/v1
    kind: Open5GS
    metadata:
        name: open5gs-sample
        namespace: default
    spec:
        configuration:
            slices:
              - sst: "1"
                sd: "0x111111"
    ```

2. Apply the deployment file:

   ```bash
   kubectl apply -f open5gs-deployment.yaml
   ```

### Create Open5GS Users

1. Create a configuration file for the users you want to add. Here’s an example:

    ```yaml
    apiVersion: net.gradiant.org/v1
    kind: Open5GSUser
    metadata:
        name: open5gsuser-sample
        namespace: default
    spec:
        imsi: "999700000000001"
        key: "465B5CE8B199B49FAA5F0A2EE238A6BC"
        opc: "E8ED289DEBA952E4283B54E88E6183CA"
        sd: "111111"
        sst: "1"
        apn: "internet"
        open5gs:
            name: "open5gs-sample"
            namespace: "default"
    ```

    - The `apn`, `sst`, and `sd` fields are optional. If they are not provided in the configuration, default values will be used by the system.
    - The `open5gs` field must contain the `name` and `namespace` of the Open5GS deployment to which the user will be assigned.

2. Apply the user configuration:

   ```bash
   kubectl apply -f open5gsuser-1.yaml
   ```

For more information on how to use the operator and more advanced configurations, please refer to the [Documentation](https://gradiant.github.io/open5gs-operator/).

## Demo
A complete demo with UERANSIM is available at [this link](https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/complete-demo-ueransim.html).

## Notes
1. The operator will restart the necessary pods to apply the changes and that may cause a service interruption. For example, if the `amf.metrics` parameter is changed, the operator will update the AMF deployment to apply the changes, resulting in a service interruption.
2. By default, all core components are enabled (except for the WebUI). To disable a component, set the `enabled` field to `false` in the CR.
3. By default, service accounts are not created. To create a service account, set the `serviceAccount` field to `true` in the CR for the desired component.
4. By default, components with metrics support have metrics enabled (AMF, PCF, UPF). To disable metrics, set the `metrics` field to `false` in the CR for the desired component.
5. By default, components use the `ClusterIP` service type. To change the service type, set the `serviceType` field to `LoadBalancer` or `NodePort` in the CR for the desired component. This option is currently available for the `amf-ngap`, `smf-pfcp`, `upf-pfcp`, and `upf-gtpu` services.
6. The `open5gsImage` field in the CR specifies the version of the Open5GS images. If not specified, the operator defaults to version `docker.io/gradiant/open5gs:2.7.2`.
7. The `webuiImage` field in the CR specifies the version of the Open5GS WebUI image. If not specified, the operator defaults to version `docker.io/gradiant/open5gs-webui:2.7.2`.
8. The `mongoDBVersion` field in the CR specifies the version of the MongoDB image. If not specified, the operator defaults to version `5.0.10-debian-11-r3`.
9. Components with metric support can generate a `ServiceMonitor` CR to expose metrics to Prometheus. However, ensure that the `ServiceMonitor` CRD is installed in the cluster; otherwise, the operator will encounter an error and fail to create the resource. To create a ServiceMonitor, set the `serviceMonitor` field to `true` in the CR for the desired component.


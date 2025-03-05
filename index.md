---
title: "Open5GS Kubernetes Operator"
---

# Open5GS Kubernetes Operator

The **Open5GS Kubernetes Operator** is a custom Kubernetes operator designed to automate the deployment, configuration, and lifecycle management of Open5GS and its subscribers in a declarative manner. It uses two Custom Resource Definitions (CRDs): one to manage Open5GS deployments and another for user management.

## What does it do?

### Deployment and reconfiguration
The operator automates the deployment and reconfiguration of Open5GS instances in a Kubernetes cluster. It allows you to define the desired state of an Open5GS deployment in a declarative way, and the operator will ensure that the actual state of the deployment matches the desired state. This includes enabling/disabling components, configuring network slices, and defining the parameters of the Open5GS deployment. Any drift between the actual and desired state will be detected and corrected automatically by the operator. Note that the operator will restart the neccesary pods to apply the changes and that may cause a service interruption.

## Multi-namespace support 
The operator handles multiple Open5GS deployments across different Kubernetes namespaces, ensuring resource isolation. It can also manage several Open5GS deployments within the same namespace, allowing independent management of each Open5GS instance.

## User management
The operator provides full management of Open5GS subscribers, including configuration of network slices and the target Open5GS deployment to which they should be assigned. It distinguishes between **Managed Users** and **Unmanaged Users**:

- **Managed Users**: These are users whose IMSI is defined in a CR (Custom Resource). The operator controls their configuration, and any discrepancy between the actual state and the desired state in the CR will be detected as drift and corrected automatically, ensuring the configuration always aligns with the declarative source of truth.
  
- **Unmanaged Users**: These users are not controlled by the operator and are created externally (e.g., via scripts that directly modify the database or the Open5GS WebUI). Unmanaged users will not be altered by the operator, allowing compatibility with external tools and temporary deployments that don't need strict management by the operator.

## Documentation Sections
- [Installation Options](docs/installation-options/installation-options.md)  
  A guide about the different installation methods available, including using Helm and direct Kubernetes manifests.
- [Basic Open5GS Deployment](docs/basic-deployment/basic-deployment.md)  
  A guide to setting up a basic deployment of Open5GS using the operator.
- [Advanced Open5GS Deployment](docs/advanced-deployment/advanced-deployment.md)  
  A guide to setting up an advanced deployment of Open5GS using the operator, including more advanced configuration options.
- [Complete Demo with UERANSIM](docs/complete-demo-ueransim/complete-demo-ueransim.md)  
  A comprehensive demonstration of deploying the operator with a UERANSIM setup for testing connectivity.

---
## How to Install

### Option 1: Installation using Helm (recommended)
```bash
helm install open5gs-operator oci://registry-1.docker.io/gradiantcharts/open5gs-operator --version 1.0.0
```

#### Uninstall with Helm

Delete all the Open5GSUser and Open5GS resources and run:

```bash
helm uninstall open5gs-operator 
   ```

### Option 2: Installation without Helm

If you prefer not to use Helm, you can apply the Kubernetes manifests directly or use the Makefile to install de CRD and deploy the operator.

```bash
make deploy IMG=gradiant/open5gs-operator:1.0.0
```

#### Uninstall without Helm

Delete all the Open5GSUser and Open5GS resources and run:

```bash
make undeploy
```

### Option 3: Run locally (outside the cluster)

This option is only recommended for development

```bash
make install run
```
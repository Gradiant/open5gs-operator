## How to Install

### Option 1: Installation using Helm (recommended)
```bash
helm install open5gs-operator oci://registry-1.docker.io/gradiantcharts/open5gs-operator --version 1.1.0
```

New deployments use MongoDB 8.0.34. Check the [MongoDB kernel compatibility notes](https://www.mongodb.com/docs/v8.0/release-notes/8.0/) before choosing a cluster. For an existing Bitnami installation, back up the database and follow the [MongoDB migration guide](https://github.com/Gradiant/open5gs-operator/blob/112885c18336dcb6d025021fce9c73784ffd881f/README.md#mongodb-migration) before changing the image.

#### Uninstall with Helm

Delete all the Open5GSUser and Open5GS resources and run:

```bash
helm uninstall open5gs-operator 
   ```

### Option 2: Installation without Helm

If you prefer not to use Helm, you can apply the Kubernetes manifests directly or use the Makefile to install de CRD and deploy the operator.

```bash
make deploy IMG=gradiant/open5gs-operator:1.1.0
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
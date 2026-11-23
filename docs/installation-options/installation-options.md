---
title: Installation and upgrade
---

# Install Open5GS Operator 1.1.0

Use a Kubernetes cluster with a StorageClass for the MongoDB PVC. Check that the nodes can run MongoDB 8.0.34. Some Linux kernels prevent MongoDB from starting even inside a container. See the [MongoDB compatibility notes](https://www.mongodb.com/docs/v8.0/release-notes/8.0/) and the [Kind tutorial](../complete-demo-ueransim/complete-demo-ueransim.md) for a tested setup.

## Install with Helm

```bash
helm upgrade --install open5gs-operator oci://registry-1.docker.io/gradiantcharts/open5gs-operator \
  --version 1.1.0 --namespace open5gs-system --create-namespace --wait --timeout 3m
kubectl -n open5gs-system rollout status deployment/open5gs-operator-controller-manager
kubectl get crd open5gses.net.gradiant.org open5gsusers.net.gradiant.org
```

Continue with [Basic deployment](../basic-deployment/basic-deployment.md).

## Upgrade from 1.0.7

Before upgrading, back up the Open5GS database and record the running MongoDB image and PVC. If it uses Bitnami, pin a legacy image that is still available in the CR before the Helm upgrade. Use the same release name and namespace as your existing installation. The new defaults can restart workloads.

Fresh Open5GS deployments use official MongoDB 8.0.34. Follow [MongoDB migration](../mongodb-migration/mongodb-migration.md) to switch an existing Bitnami deployment to a separate, restored PVC. For WebUI on an empty database, first create the bootstrap Secret described under [Advanced deployment](../advanced-deployment/advanced-deployment.md).

## Install from source

Use a checkout of [the source for 1.1.0](https://github.com/Gradiant/open5gs-operator/tree/112885c18336dcb6d025021fce9c73784ffd881f).

```bash
make deploy IMG=gradiant/open5gs-operator:1.1.0
```

For local controller development, use `make install run` with the intended kubeconfig.

## Uninstall

Back up the database before deleting an `Open5GS` resource. Its owned PVC may be removed. Delete subscriber CRs while MongoDB is reachable, then remove any lab Open5GS instances and uninstall the operator from its installation namespace.

```bash
helm uninstall open5gs-operator --namespace open5gs-system
```

Use `make undeploy` for an installation made with the Makefile.

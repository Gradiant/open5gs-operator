---
title: Basic Open5GS deployment
---

# Create a basic Open5GS deployment

After [installing the operator](../installation-options/installation-options.md), use the example manifests to create an Open5GS instance and one subscriber. The instance uses Open5GS 2.8.0 and official MongoDB 8.0.34. You can enable WebUI in the [advanced deployment](../advanced-deployment/advanced-deployment.md).

Download [open5gs-basic-deployment.yaml](open5gs-basic-deployment.yaml) and [open5gs-user.yaml](open5gs-user.yaml) into your working directory.

```bash
kubectl create namespace open5gs-basic
kubectl -n open5gs-basic apply -f open5gs-basic-deployment.yaml
```

Wait for the Deployments to appear, then check that the core Pods are ready and the MongoDB claim is bound.

```bash
kubectl -n open5gs-basic get deployments,pods,pvc
kubectl -n open5gs-basic rollout status deployment/open5gs-sample-mongodb --timeout=330s
kubectl -n open5gs-basic rollout status deployment/open5gs-sample-upf --timeout=330s
```

Register the subscriber in the same namespace. The IMSI and authentication keys are public lab values.

```bash
kubectl -n open5gs-basic apply -f open5gs-user.yaml
kubectl -n open5gs-basic exec deployment/open5gs-sample-mongodb -- \
  mongosh open5gs --quiet --eval 'db.subscribers.countDocuments({imsi: "999700000000001"})'
```

The count should reach `1` after reconciliation. For a registration and data path test, continue with the [UERANSIM demo](../complete-demo-ueransim/complete-demo-ueransim.md).

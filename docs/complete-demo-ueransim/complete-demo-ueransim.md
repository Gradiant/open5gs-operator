---
title: Complete demo with UERANSIM
---

# Test Open5GS 1.1.0 with UERANSIM

In this lab, two independent UEs register on the same slice, establish PDU sessions, and each send an HTTP request through its own tunnel. The second part adds another slice and repeats the traffic check. The subscriber keys are public test values for a disposable cluster.

## Create a Kind cluster and install the operator

Run the lab on a Linux host with Docker, Kind, kubectl, Helm, and `/dev/net/tun`. Check the host kernel against the [MongoDB compatibility notes](https://www.mongodb.com/docs/v8.0/release-notes/8.0/). Kind nodes share the host kernel. The UPF mode used here requires the forwarding sysctl enabled in [kind-config.yaml](kind-config.yaml).

Download that file, [the first Open5GS manifest](open5gs-one-slice.yaml), [the subscriber manifest](open5gs-users.yaml), and [the gNB settings](gnb-1-values.yaml) into your working directory. Run the commands in Bash.

```bash
kind create cluster --name open5gs-demo --image kindest/node:v1.35.0 --config kind-config.yaml --wait 120s
kubectl config current-context
```

Confirm that the context is `kind-open5gs-demo`, then follow [Install with Helm](../installation-options/installation-options.md#install-with-helm). That uses the public operator chart `1.1.0`.

## Deploy the core and register subscribers

```bash
NS=open5gs-demo
k() { kubectl -n "$NS" "$@"; }
kubectl create namespace "$NS"
k apply -f open5gs-one-slice.yaml
```

Wait for the Deployments to appear. MongoDB should have a bound PVC, and the UPF logs should show `PFCP associated` before you start UERANSIM.

```bash
k get deployments,pods,pvc
k rollout status deployment/open5gs-sample-mongodb --timeout=330s
k rollout status deployment/open5gs-sample-upf --timeout=330s
k logs deployment/open5gs-sample-upf -c open5gs-upf --tail=50

k apply -f open5gs-users.yaml
k exec deployment/open5gs-sample-mongodb -- mongosh open5gs --quiet --eval 'db.subscribers.countDocuments()'
```

Wait for the count to reach `4` before starting the gNB and the first two UEs. The two remaining subscribers are used in the extension.

## Run the gNB and test each UE

```bash
helm upgrade --install ueransim-gnb oci://registry-1.docker.io/gradiant/ueransim-gnb \
  --version 0.2.6 --namespace "$NS" --wait --timeout 3m --values gnb-1-values.yaml

start_ue() {
  local number=$1 sst=$2 sd=$3
  helm upgrade --install "simulated-ue-$number" oci://registry-1.docker.io/gradiant/ueransim-ues \
    --version 0.1.2 --namespace "$NS" --wait --timeout 3m \
    --set gnb.hostname=ueransim-gnb --set image.pullPolicy=IfNotPresent \
    --set-string "initialMSISDN=000000000$number" --set-string mcc=999 --set-string mnc=70 \
    --set "sst=$sst" --set-string "sd=0x$sd" \
    --set "apnList[0].type=IPv4" --set "apnList[0].apn=internet" \
    --set "apnList[0].slice.sst=$sst" --set-string "apnList[0].slice.sd=0x$sd" \
    --set key=465B5CE8B199B49FAA5F0A2EE238A6BC --set op=E8ED289DEBA952E4283B54E88E6183CA \
    --set opType=OPC
}

start_ue 1 1 111111
start_ue 2 1 111111
k create deployment data-target --image=python:3.10-slim -- python -m http.server 8080
k rollout status deployment/data-target --timeout=120s
target=$(k get pod -l app=data-target -o jsonpath='{.items[0].status.podIP}')
```

Each UE runs in its own Pod and has its own `uesimtun0`. The HTTP target sits inside the cluster so the test does not depend on Internet access from the core.

```bash
check_ue() {
  local number=$1 deployment attempt
  deployment=$(k get deployment -l "app.kubernetes.io/instance=simulated-ue-$number" -o name)
  for attempt in {1..30}; do
    if k exec "$deployment" -- curl --ipv4 --noproxy "*" --interface uesimtun0 \
      --fail --silent --show-error --max-time 5 --output /dev/null \
      --write-out "UE $number: %{local_ip} returned HTTP %{http_code}\n" "http://$target:8080/"; then
      return 0
    fi
    sleep 2
  done
  k logs "$deployment" --tail=40
  return 1
}

check_ue 1
check_ue 2
```

Each command should print HTTP `200` and a different address from the `10.45.0.0/16` UE pool.

## Extension: add a second slice

Download [the two-slice manifest](open5gs-one-slice-uncommented.yaml) and [the second gNB configuration](gnb-2-values.yaml). The operator updates the AMF and NSSF configurations when you apply the manifest. This phase switches the same simulated gNB to the second slice and brings up two new UEs. Restart the gNB after upgrading its Helm release so it loads the updated configuration and reconnects to the AMF.

```bash
helm uninstall simulated-ue-1 simulated-ue-2 --namespace "$NS"
old_amf_hash=$(k get deployment open5gs-sample-amf -o jsonpath='{.spec.template.metadata.annotations.open5gs/configmap-hash}')
old_nssf_hash=$(k get deployment open5gs-sample-nssf -o jsonpath='{.spec.template.metadata.annotations.open5gs/configmap-hash}')
k apply -f open5gs-one-slice-uncommented.yaml

wait_for_slice_rollout() {
  local component=$1 old_hash=$2 new_hash attempt
  for attempt in {1..45}; do
    new_hash=$(k get deployment "open5gs-sample-$component" -o jsonpath='{.spec.template.metadata.annotations.open5gs/configmap-hash}')
    if [[ -n "$new_hash" && "$new_hash" != "$old_hash" ]]; then
      k rollout status "deployment/open5gs-sample-$component" --timeout=330s
      return
    fi
    sleep 2
  done
  echo "The $component configuration did not change" >&2
  return 1
}

wait_for_slice_rollout amf "$old_amf_hash"
wait_for_slice_rollout nssf "$old_nssf_hash"
helm upgrade ueransim-gnb oci://registry-1.docker.io/gradiant/ueransim-gnb \
  --version 0.2.6 --namespace "$NS" --wait --timeout 3m --values gnb-2-values.yaml
old_gnb=$(k get pods -l app.kubernetes.io/instance=ueransim-gnb -o jsonpath='{.items[0].metadata.name}')
k rollout restart deployment/ueransim-gnb
k rollout status deployment/ueransim-gnb --timeout=180s
k wait --for=delete "pod/$old_gnb" --timeout=120s

start_ue 3 2 222222
start_ue 4 2 222222
check_ue 3
check_ue 4
```

The second pair should also return HTTP `200`, this time with subscribers configured for SST 2 and SD `222222`.

## Clean up the lab

If you created the disposable `open5gs-demo` Kind cluster for this tutorial, delete it when finished. This removes all lab data.

```bash
kind delete cluster --name open5gs-demo
```

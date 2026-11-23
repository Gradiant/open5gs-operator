#!/usr/bin/env bash
set -Eeuo pipefail

namespace=${1:-open5gs-kind-smoke}
k() { kubectl --namespace "$namespace" "$@"; }
wait_for_deployment() {
    local attempt
    for ((attempt=0; attempt<90; attempt++)); do
        if k get "$1" >/dev/null 2>&1; then
            return 0
        fi
        sleep 2
    done
    printf 'Deployment %s was not created\n' "$1" >&2
    return 1
}

wait_for_pfcp() {
    local attempt pod
    for ((attempt=0; attempt<45; attempt++)); do
        pod=$(k get pods -l app.kubernetes.io/name=upf,app.kubernetes.io/instance=core \
            -o go-template='{{range .items}}{{if not .metadata.deletionTimestamp}}{{.metadata.name}}{{end}}{{end}}')
        if [[ -n "$pod" ]] && k logs "$pod" -c open5gs-upf --tail=100 | grep 'PFCP associated' >/dev/null; then
            return 0
        fi
        sleep 2
    done
    echo "UPF did not associate with SMF over PFCP" >&2
    return 1
}

kubectl create namespace "$namespace"
trap 'k get pods,pvc; k get events --sort-by=.lastTimestamp | tail -25' ERR

# Disposable credentials and public UERANSIM subscriber keys.
k create secret generic webui-admin --from-literal=username=kind-admin \
    --from-literal="password=$(openssl rand -hex 16)"
k apply -f - <<'YAML'
apiVersion: net.gradiant.org/v1
kind: Open5GS
metadata:
  name: core
spec:
  webuiAdminSecret: webui-admin
  webui: {enabled: true}
  configuration:
    slices:
      - sst: "1"
        sd: "111111"
YAML

components=(amf ausf bsf mongodb nrf nssf pcf scp smf udm udr upf webui)
deployments=()
for component in "${components[@]}"; do
    deployments+=("deployment/core-$component")
done
for deployment in "${deployments[@]}"; do
    wait_for_deployment "$deployment"
    k rollout status "$deployment" --timeout=330s
    [[ "$(k get "$deployment" -o jsonpath="{.spec.template.spec.automountServiceAccountToken}")" == false ]]
    [[ "$(k get "$deployment" -o jsonpath="{.spec.template.spec.serviceAccountName}")" == "${deployment#deployment/}" ]]
    [[ "$(k get "sa/${deployment#deployment/}" -o jsonpath="{.automountServiceAccountToken}")" == false ]]
done
[[ "$(k get deployment/core-mongodb -o jsonpath="{.spec.template.spec.containers[0].image}")" == docker.io/library/mongo:8.0.34 ]]
[[ "$(k get deployment/core-upf -o jsonpath="{.spec.template.spec.containers[0].command[0]}")" == /usr/bin/tini ]]
k exec deploy/core-mongodb -- mongosh open5gs --quiet --eval \
    'quit(db.accounts.countDocuments({username: "kind-admin", roles: "admin"}) === 1 && db.accounts.countDocuments({username: "admin"}) === 0 ? 0 : 1)'

k patch open5gs core --type=merge -p '{"spec":{"upf":{"unprivileged":true,"metrics":false,"resources":{"requests":{"cpu":"75m","memory":"64Mi"},"limits":{"cpu":"1","memory":"512Mi"}}}}}'
k wait --for=jsonpath='{.spec.template.spec.containers[0].securityContext.privileged}'=false deployment/core-upf --timeout=90s
k rollout status deployment/core-upf --timeout=330s
[[ "$(k get deployment/core-upf -o jsonpath="{.spec.template.spec.containers[0].resources.requests.cpu}")" == 75m ]]
[[ "$(k get deployment/core-upf -o jsonpath="{.spec.template.spec.initContainers[0].securityContext.capabilities.add[0]}")" == NET_ADMIN ]]

wait_for_pfcp

for index in 1 2; do
    k apply -f - <<YAML
apiVersion: net.gradiant.org/v1
kind: Open5GSUser
metadata:
  name: ue-$index
spec:
  imsi: "99970000000000$index"
  key: "465B5CE8B199B49FAA5F0A2EE238A6BC"
  opc: "E8ED289DEBA952E4283B54E88E6183CA"
  sd: "111111"
  sst: "1"
  apn: "internet"
  open5gs:
    name: core
    namespace: $namespace
YAML
done
for ((attempt=0; attempt<30; attempt++)); do
    if k exec deploy/core-mongodb -- mongosh open5gs --quiet --eval 'quit(db.subscribers.countDocuments() === 2 ? 0 : 1)'; then
        break
    fi
    sleep 2
done
[[ "$attempt" -lt 30 ]]

helm upgrade --install ueransim-gnb oci://registry-1.docker.io/gradiant/ueransim-gnb \
    --version 0.2.6 --namespace "$namespace" --timeout 180s --wait \
    --set amf.hostname=core-amf-ngap --set-string mcc=999 --set-string mnc=70 \
    --set image.pullPolicy=IfNotPresent --set sst=1 --set-string sd=0x111111 --set-string tac=0001 \
    --set ues.enabled=false
for index in 1 2; do
    helm upgrade --install "simulated-ue-$index" oci://registry-1.docker.io/gradiant/ueransim-ues \
        --version 0.1.2 --namespace "$namespace" --timeout 180s --wait \
        --set gnb.hostname=ueransim-gnb --set image.pullPolicy=IfNotPresent \
        --set-string "initialMSISDN=000000000$index" --set-string mcc=999 --set-string mnc=70 \
        --set sst=1 --set-string sd=0x111111 --set key=465B5CE8B199B49FAA5F0A2EE238A6BC \
        --set op=E8ED289DEBA952E4283B54E88E6183CA --set opType=OPC
done
k create deployment data-target --image=python:3.10-slim -- python -m http.server 8080
k rollout status deployment/data-target --timeout=120s
target=$(k get pod -l app=data-target -o jsonpath="{.items[0].status.podIP}")

check_traffic() {
    local index attempt deployment
    for index in 1 2; do
        deployment=$(k get deployment -l "app.kubernetes.io/instance=simulated-ue-$index" -o name)
        for ((attempt=0; attempt<30; attempt++)); do
            if k exec "$deployment" -- curl --ipv4 --noproxy "*" --interface uesimtun0 \
                --fail --silent --show-error --max-time 5 --output /dev/null \
                --write-out "UE $index reached data target from %{local_ip} (HTTP %{http_code})\n" \
                "http://$target:8080/"; then
                break
            fi
            sleep 2
        done
        [[ "$attempt" -lt 30 ]]
    done
}

check_traffic

k rollout restart deployment/core-mongodb
k rollout status deployment/core-mongodb --timeout=330s
k exec deploy/core-mongodb -- mongosh open5gs --quiet --eval \
    'quit(db.subscribers.countDocuments() === 2 && db.accounts.countDocuments({username: "kind-admin"}) === 1 ? 0 : 1)'

k rollout restart deployment/core-upf
k rollout status deployment/core-upf --timeout=330s
wait_for_pfcp
k rollout restart deployment/ueransim-gnb
k rollout status deployment/ueransim-gnb --timeout=180s
for index in 1 2; do
    deployment=$(k get deployment -l "app.kubernetes.io/instance=simulated-ue-$index" -o name)
    k rollout restart "$deployment"
    k rollout status "$deployment" --timeout=180s
done
check_traffic

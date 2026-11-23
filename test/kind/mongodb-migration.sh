#!/usr/bin/env bash
set -euo pipefail

namespace=${1:-open5gs-kind-migration}
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

kubectl create namespace "$namespace"
trap 'k get pods,pvc; k get events --sort-by=.lastTimestamp | tail -25' ERR
k create secret generic webui-admin --from-literal=username=migration-admin \
    --from-literal="password=$(openssl rand -hex 16)"

k apply -f - <<'YAML'
apiVersion: net.gradiant.org/v1
kind: Open5GS
metadata:
  name: core
spec:
  mongoDBVersion: bitnamilegacy/mongodb:8.0.13-debian-12-r0
  webuiAdminSecret: webui-admin
  webui: {enabled: true}
  amf: {enabled: false}
  ausf: {enabled: false}
  bsf: {enabled: false}
  nrf: {enabled: false}
  nssf: {enabled: false}
  pcf: {enabled: false}
  scp: {enabled: false}
  smf: {enabled: false}
  udm: {enabled: false}
  udr: {enabled: false}
  upf: {enabled: false}
YAML
wait_for_deployment deployment/core-mongodb
wait_for_deployment deployment/core-webui
k rollout status deployment/core-mongodb --timeout=330s
k rollout status deployment/core-webui --timeout=330s
k exec deployment/core-mongodb -- mongosh open5gs --quiet --eval 'db.subscribers.insertOne({imsi: "999700000000001"})'

# Dropping the explicit Bitnami image must keep the legacy deployment intact.
k patch open5gs core --type=merge -p '{"spec":{"mongoDBVersion":null}}'
k apply -f - <<'YAML'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: restored
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 1Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: restore-target
spec:
  replicas: 1
  selector:
    matchLabels:
      app: restore-target
  template:
    metadata:
      labels:
        app: restore-target
    spec:
      securityContext:
        runAsUser: 999
        runAsGroup: 999
        fsGroup: 999
      containers:
      - name: mongodb
        image: mongo:8.0.34
        args: ["mongod", "--bind_ip_all"]
        readinessProbe:
          exec:
            command: ["mongosh", "--quiet", "--eval", "db.adminCommand({ping: 1})"]
        volumeMounts:
        - name: restored
          mountPath: /data/db
      volumes:
      - name: restored
        persistentVolumeClaim:
          claimName: restored
YAML
k rollout status deployment/restore-target --timeout=330s
[[ "$(k get deployment/core-mongodb -o jsonpath="{.spec.template.spec.containers[0].image}")" == bitnamilegacy/mongodb:8.0.13-debian-12-r0 ]]
k exec deployment/core-mongodb -- mongodump --db open5gs --archive |
    k exec -i deployment/restore-target -- mongorestore --archive --nsInclude="open5gs.*"
k scale deployment/restore-target --replicas=0
k wait --for=delete pod -l app=restore-target --timeout=120s

k patch open5gs core --type=merge -p '{"spec":{"mongoDBVersion":"docker.io/library/mongo:8.0.34","mongoDBVolumeClaim":"restored"}}'
k wait --for=jsonpath='{.spec.template.spec.volumes[0].persistentVolumeClaim.claimName}'=restored deployment/core-mongodb --timeout=90s
k rollout status deployment/core-mongodb --timeout=330s
k exec deployment/core-mongodb -- mongosh open5gs --quiet --eval \
    'quit(db.subscribers.countDocuments({imsi: "999700000000001"}) === 1 && db.accounts.countDocuments({username: "migration-admin", roles: "admin"}) === 1 ? 0 : 1)'
k get pvc core-mongodb restored

k patch open5gs core --type=merge -p '{"spec":{"mongoDBVersion":"bitnamilegacy/mongodb:8.0.13-debian-12-r0","mongoDBVolumeClaim":null}}'
k wait --for=jsonpath='{.spec.template.spec.volumes[0].persistentVolumeClaim.claimName}'=core-mongodb deployment/core-mongodb --timeout=90s
k rollout status deployment/core-mongodb --timeout=330s
k exec deployment/core-mongodb -- mongosh open5gs --quiet --eval \
    'quit(db.subscribers.countDocuments({imsi: "999700000000001"}) === 1 && db.accounts.countDocuments({username: "migration-admin", roles: "admin"}) === 1 ? 0 : 1)'

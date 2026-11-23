---
title: MongoDB migration
---

# Move an existing Open5GS database from Bitnami to official MongoDB

Open5GS Operator 1.1.0 uses official MongoDB 8.0.34 for new deployments. Existing Bitnami deployments use a different on-disk layout. Move the database with a logical backup and restore into a **separate PVC**. Keep the old PVC so you can roll back.

Plan a maintenance window and pause writers while you copy and verify the data, including subscriber reconciliation, WebUI access and live core traffic. Keep a record of how to pause and resume those writers in your installation. Check the source version and [MongoDB kernel compatibility](https://www.mongodb.com/docs/v8.0/release-notes/8.0/) before choosing the target cluster. The lab flow was tested with Bitnami 8.0.13 as the source. For a different source version, verify the supported dump and restore path.

## Record the source and take a backup

Set the namespace and instance name for the existing deployment. Check the image and claim that the operator currently uses, confirm the source image is available, and record a pinned tag or digest for rollback.

```bash
NS=open5gs
CORE=open5gs-sample
kubectl -n "$NS" get deployment "$CORE-mongodb" \
  -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}{.spec.template.spec.volumes[0].persistentVolumeClaim.claimName}{"\n"}'

umask 077
kubectl -n "$NS" exec "deployment/$CORE-mongodb" -c mongodb -- \
  mongodump --db open5gs --archive > open5gs.archive
test -s open5gs.archive
```

If your database requires authentication, use your approved access method. Keep the archive private because it contains subscriber keys and account hashes.

## Restore into a separate PVC

Download [mongodb-restore.yaml](mongodb-restore.yaml). It starts a temporary MongoDB instance without a Service and mounts `open5gs-mongodb-restored` at `/data/db`. Choose a claim name and storage size appropriate for your deployment. Verify that the namespace has a StorageClass and that these object names are available before you apply the file.

### Temporary MongoDB and restore PVC

```yaml
{% include_relative mongodb-restore.yaml %}
```

```bash
kubectl -n "$NS" apply -f mongodb-restore.yaml
kubectl -n "$NS" rollout status deployment/mongodb-restore --timeout=330s
kubectl -n "$NS" exec -i deployment/mongodb-restore -- \
  mongorestore --archive --nsInclude='open5gs.*' < open5gs.archive
```

Compare the collection counts on the source and destination. Also check representative subscribers and accounts without printing authentication data.

```bash
counts='printjson({subscribers: db.subscribers.countDocuments(), accounts: db.accounts.countDocuments()})'
kubectl -n "$NS" exec "deployment/$CORE-mongodb" -c mongodb -- mongosh open5gs --quiet --eval "$counts"
kubectl -n "$NS" exec deployment/mongodb-restore -- mongosh open5gs --quiet --eval "$counts"
kubectl -n "$NS" scale deployment/mongodb-restore --replicas=0
kubectl -n "$NS" wait --for=delete pod -l app=mongodb-restore --timeout=120s
```

Wait for the temporary MongoDB Pod to release the PVC before switching the operator to it.

## Switch and verify

Update the following fields in your source of truth for the Open5GS CR. Apply that manifest, or use this patch for a manually managed deployment. The operator must be running to reconcile the change.

```bash
kubectl -n "$NS" patch open5gs "$CORE" --type=merge \
  -p '{"spec":{"mongoDBVersion":"docker.io/library/mongo:8.0.34","mongoDBVolumeClaim":"open5gs-mongodb-restored"}}'
kubectl -n "$NS" wait --for=jsonpath='{.spec.template.spec.volumes[0].persistentVolumeClaim.claimName}'=open5gs-mongodb-restored \
  "deployment/$CORE-mongodb" --timeout=90s
kubectl -n "$NS" rollout status "deployment/$CORE-mongodb" --timeout=330s
kubectl -n "$NS" exec "deployment/$CORE-mongodb" -c mongodb -- mongosh open5gs --quiet --eval "$counts"
```

Check subscriber and WebUI account counts, sign in with a known account if WebUI is enabled, and validate a known subscriber session before opening the deployment to normal traffic. Retain both PVCs and the backup until the migration is accepted.

## Roll back

Pause writes again. If any writes reached the new database, take a fresh backup and decide how to preserve them before rolling back. The legacy PVC only contains data up to the first cutover.

Restore the recorded Bitnami image in `spec.mongoDBVersion` and clear `spec.mongoDBVolumeClaim`. Wait until the Deployment references the original PVC and is ready, then repeat the subscriber and WebUI checks before resuming writes.

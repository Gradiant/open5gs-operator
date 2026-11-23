# Complete Demo with UERANSIM

In this demo it is shown how to deploy an Open5GS instance and create several users. Also, it is demonstrated how the changes in the network slices are managed by the operator and a connectivity test is performed to verify that the users are assigned to the correct slice.

The commands use the `default` namespace. The operator is deployed using the following command:
```bash
helm install open5gs-operator oci://registry-1.docker.io/gradiantcharts/open5gs-operator --version 1.1.0 --wait --timeout 3m
```
To observe the logs of the operator:
```bash
kubectl logs deployment/open5gs-operator-controller-manager --tail=50
```
For a fresh database, create the WebUI administrator Secret once in the same namespace. Use a password of at least 12 characters.

```bash
read -r -s -p "WebUI password: " WEBUI_PASSWORD
printf '\n'
printf '%s' "$WEBUI_PASSWORD" | kubectl create secret generic open5gs-webui-admin \
  --from-literal=username=open5gs-admin --from-file=password=/dev/stdin
unset WEBUI_PASSWORD
```

We create an instance of Open5GS with a configured slice:
```bash
kubectl apply -f https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/open5gs-one-slice.yaml
```
Once the Deployments appear, wait for MongoDB and the UPF before registering the users:
```bash
kubectl -n default rollout status deployment/open5gs-sample-mongodb --timeout=330s
kubectl -n default rollout status deployment/open5gs-sample-upf --timeout=330s
```
Four users are registered, two for each slice:
```bash
kubectl apply -f https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/open5gs-users.yaml
```
UERANSIM is deployed with two GNBs, each one with two users:
```bash
helm install ueransim-gnb-1 oci://registry-1.docker.io/gradiant/ueransim-gnb --version 0.2.6 --wait --timeout 3m --values https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/gnb-1-values.yaml

helm install ueransim-gnb-2 oci://registry-1.docker.io/gradiant/ueransim-gnb --version 0.2.6 --wait --timeout 3m --values https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/gnb-2-values.yaml
```

It is observed that only the users of the first slice are connected. This is because we only enabled the first slice in the Open5GS instance:
```bash
kubectl logs deployment/open5gs-sample-amf -n default
kubectl logs deployment/ueransim-gnb-1-ues -n default
kubectl logs deployment/ueransim-gnb-2-ues -n default

```
We can also check connectivity. Use an HTTP target inside the cluster so this test does not depend on external DNS or Internet access:
```bash
kubectl -n default create deployment docs-data-target --image=python:3.10-slim -- python -m http.server 8080
kubectl -n default rollout status deployment/docs-data-target --timeout=120s
TARGET=$(kubectl -n default get pod -l app=docs-data-target -o jsonpath='{.items[0].status.podIP}')
```
The first two commands should return HTTP 200. The last two should fail because the second slice is not yet enabled:
```bash
kubectl -n default exec deployment/ueransim-gnb-1-ues -- curl -fsS --noproxy "*" --interface uesimtun0 --max-time 5 -o /dev/null -w "HTTP %{http_code}\n" "http://$TARGET:8080/"
kubectl -n default exec deployment/ueransim-gnb-1-ues -- curl -fsS --noproxy "*" --interface uesimtun1 --max-time 5 -o /dev/null -w "HTTP %{http_code}\n" "http://$TARGET:8080/"
kubectl -n default exec deployment/ueransim-gnb-2-ues -- curl -fsS --noproxy "*" --interface uesimtun0 --max-time 5 -o /dev/null -w "HTTP %{http_code}\n" "http://$TARGET:8080/"
kubectl -n default exec deployment/ueransim-gnb-2-ues -- curl -fsS --noproxy "*" --interface uesimtun1 --max-time 5 -o /dev/null -w "HTTP %{http_code}\n" "http://$TARGET:8080/"
```
UERANSIM in undeployed:
```bash
helm uninstall ueransim-gnb-1 ueransim-gnb-2 --wait --timeout 120s
```
Now you must uncomment the second slice in the Open5GS CR `https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/open5gs-one-slice.yaml` and apply it. 

This will enable the new slice and the operator will apply the new configuration and restart the neccesary pods. You can apply the uncommented file with the following command:
```bash
kubectl apply -f https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/open5gs-one-slice-uncommented.yaml
```
Wait for the new AMF and NSSF Pods to appear and be ready before deploying UERANSIM again:
```bash
helm install ueransim-gnb-1 oci://registry-1.docker.io/gradiant/ueransim-gnb --version 0.2.6 --wait --timeout 3m --values https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/gnb-1-values.yaml

helm install ueransim-gnb-2 oci://registry-1.docker.io/gradiant/ueransim-gnb --version 0.2.6 --wait --timeout 3m --values https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/gnb-2-values.yaml
```
Once both gNBs are up, restart the UE Deployments so they resolve the new gNB Pod addresses:
```bash
kubectl -n default rollout restart deployment/ueransim-gnb-1-ues deployment/ueransim-gnb-2-ues
kubectl -n default rollout status deployment/ueransim-gnb-1-ues --timeout=120s
kubectl -n default rollout status deployment/ueransim-gnb-2-ues --timeout=120s
```
We observe that the users of both slices are connected:
```bash
kubectl logs deployment/open5gs-sample-amf -n default
kubectl logs deployment/ueransim-gnb-1-ues -n default
kubectl logs deployment/ueransim-gnb-2-ues -n default
```
We can also check the connectivity of the users. All four commands should now return HTTP 200:
```bash
kubectl -n default exec deployment/ueransim-gnb-1-ues -- curl -fsS --noproxy "*" --interface uesimtun0 --max-time 5 -o /dev/null -w "HTTP %{http_code}\n" "http://$TARGET:8080/"
kubectl -n default exec deployment/ueransim-gnb-1-ues -- curl -fsS --noproxy "*" --interface uesimtun1 --max-time 5 -o /dev/null -w "HTTP %{http_code}\n" "http://$TARGET:8080/"
kubectl -n default exec deployment/ueransim-gnb-2-ues -- curl -fsS --noproxy "*" --interface uesimtun0 --max-time 5 -o /dev/null -w "HTTP %{http_code}\n" "http://$TARGET:8080/"
kubectl -n default exec deployment/ueransim-gnb-2-ues -- curl -fsS --noproxy "*" --interface uesimtun1 --max-time 5 -o /dev/null -w "HTTP %{http_code}\n" "http://$TARGET:8080/"
```

Uninstall everything:
```bash
kubectl -n default delete deployment docs-data-target
helm uninstall ueransim-gnb-1
helm uninstall ueransim-gnb-2
kubectl delete open5gsusers.net.gradiant.org open5gsuser-sample-1
kubectl delete open5gsusers.net.gradiant.org open5gsuser-sample-2
kubectl delete open5gsusers.net.gradiant.org open5gsuser-sample-3
kubectl delete open5gsusers.net.gradiant.org open5gsuser-sample-4 
kubectl delete open5gs open5gs-sample
helm uninstall open5gs-operator
```
# Complete Demo with UERANSIM

In this demo it is shown how to deploy an Open5GS instance and create several users. Also, it is demonstrated how the changes in the network slices are managed by the operator and a connectivity test is performed to verify that the users are assigned to the correct slice.

The operator is deployed using the following command:
```bash
helm install open5gs-operator oci://registry-1.docker.io/gradiantcharts/open5gs-operator --version 1.0.0
```
To observe the logs of the operator:
```bash
kubectl logs deployment/open5gs-operator-controller-manager -n open5gs-operator-system -f
```
We create an instance of Open5GS with a configured slice:
```bash
kubectl apply -f https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/open5gs-one-slice.yaml
```
Four users are registered, two for each slice:
```bash
kubectl apply -f https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/open5gs-users.yaml
```
UERANSIM is deployed with two GNBs, each one with two users:
```bash
helm install ueransim-gnb-1 oci://registry-1.docker.io/gradiant/ueransim-gnb --version 0.2.6 --values https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/gnb-1-values.yaml

helm install ueransim-gnb-2 oci://registry-1.docker.io/gradiant/ueransim-gnb --version 0.2.6 --values https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/gnb-2-values.yaml
```

It is observed that only the users of the first slice are connected. This is because we only enabled the first slice in the Open5GS instance:
```bash
kubectl logs deployment/open5gs-sample-amf -n default
kubectl logs deployment/ueransim-gnb-1-ues -n default
kubectl logs deployment/ueransim-gnb-2-ues -n default

```
We can also check the connectivity of the users:
```bash
kubectl -n default exec -ti deployment/ueransim-gnb-1-ues -- ping -I uesimtun0 gradiant.org
kubectl -n default exec -ti deployment/ueransim-gnb-1-ues -- ping -I uesimtun1 gradiant.org
kubectl -n default exec -ti deployment/ueransim-gnb-2-ues -- ping -I uesimtun0 gradiant.org
kubectl -n default exec -ti deployment/ueransim-gnb-2-ues -- ping -I uesimtun1 gradiant.org
```
UERANSIM in undeployed:
```bash
helm uninstall ueransim-gnb-1
helm uninstall ueransim-gnb-2
```
Now you must uncomment the second slice in the Open5GS CR `https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/open5gs-one-slice.yaml` and apply it. 

This will enable the new slice and the operator will apply the new configuration and restart the neccesary pods. You can apply the uncommented file with the following command:
```bash
kubectl apply -f https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/open5gs-one-slice-uncommented.yaml
```
Wait until AMF and NSSF are `Running` again and deploy UERANSIM:
```bash
helm install ueransim-gnb-1 oci://registry-1.docker.io/gradiant/ueransim-gnb --version 0.2.6 --values https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/gnb-1-values.yaml

helm install ueransim-gnb-2 oci://registry-1.docker.io/gradiant/ueransim-gnb --version 0.2.6 --values https://gradiant.github.io/open5gs-operator/docs/complete-demo-ueransim/gnb-2-values.yaml
```
We observe that the users of both slices are connected:
```bash
kubectl logs deployment/open5gs-sample-amf -n default
kubectl logs deployment/ueransim-gnb-1-ues -n default
kubectl logs deployment/ueransim-gnb-2-ues -n default
```
We can also check the connectivity of the users:
```bash
kubectl -n default exec -ti deployment/ueransim-gnb-1-ues -- ping -I uesimtun0 gradiant.org
kubectl -n default exec -ti deployment/ueransim-gnb-1-ues -- ping -I uesimtun1 gradiant.org
kubectl -n default exec -ti deployment/ueransim-gnb-2-ues -- ping -I uesimtun0 gradiant.org
kubectl -n default exec -ti deployment/ueransim-gnb-2-ues -- ping -I uesimtun1 gradiant.org
```

Uninstall everything:
```bash
helm uninstall ueransim-gnb-1
helm uninstall ueransim-gnb-2
kubectl delete -f config/samples/net_v1_open5gsuser-1-4.yaml
kubectl delete -f config/samples/net_v1_open5gs.yaml
helm uninstall open5gs-operator
```
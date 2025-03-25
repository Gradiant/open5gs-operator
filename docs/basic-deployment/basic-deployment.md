# Create a Basic Open5GS Deployment

1. Create a deployment configuration file for Open5GS. Here’s a basic example (the missing configuration will be set to default values):

    ``` yaml
apiVersion: net.gradiant.org/v1
kind: Open5GS
metadata:
    name: open5gs-sample
spec:
    configuration:
        slices:
          - sst: "1"
            sd: "0x111111"
    ```
    <a href="https://gradiant.github.io/open5gs-operator/docs/basic-deployment/open5gs-basic-deployment.yaml" class="download-button" download>Download open5gs-basic-deployment.yaml</a>

    - The `slices` field is optional. If it is not provided in the configuration, default values will be used by the operator.

2. Apply the deployment file:

   ``` bash
   kubectl apply -f open5gs-basic-deployment.yaml
   ```

### Create Open5GS Users

1. Create a configuration file for the users you want to add. Here’s an example:

    ```yaml
apiVersion: net.gradiant.org/v1
kind: Open5GSUser
metadata:
    name: open5gsuser-sample
spec:
    imsi: "999700000000001"
    key: "465B5CE8B199B49FAA5F0A2EE238A6BC"
    opc: "E8ED289DEBA952E4283B54E88E6183CA"
    sd: "111111"
    sst: "1"
    apn: "internet"
    open5gs:
        name: "open5gs-sample"
        namespace: "default"
    ```
    <a href="https://gradiant.github.io/open5gs-operator/docs/basic-deployment/open5gs-user.yaml" class="download-button" download>Download open5gs-user.yaml</a>

    - The `apn`, `sst`, and `sd` fields are optional. If they are not provided in the configuration, default values will be used by the system.
    - The `open5gs` field must contain the `name` and `namespace` of the Open5GS deployment to which the user will be assigned.

2. Apply the user configuration:

   ``` bash
   kubectl apply -f open5gs-user.yaml
   ```

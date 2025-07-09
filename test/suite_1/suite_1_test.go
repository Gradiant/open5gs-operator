/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package suite_1

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gradiant/open5gs-operator/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const namespace = "open5gs-operator-system-test"
const namespace2 = "open5gs-operator-system-test-2"

var _ = Describe("controller", Ordered, func() {
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)
		By("creating manager namespace2")
		cmd = exec.Command("kubectl", "create", "ns", namespace2)
		_, _ = utils.Run(cmd)
	})

	AfterAll(func() {
		By("removing manager namespace")
		cmd := exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace2)
		_, _ = utils.Run(cmd)
	})

	Context("Operator", func() {
		It("should run successfully", func() {
			var controllerPodName string
			var err error

			var projectimage = "gradiant/open5gs-operator:1.0.4"

			// By("building the manager(Operator) image")
			// cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
			// _, err = utils.Run(cmd)
			// ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// By("loading the the manager(Operator) image on Kind")
			// err = utils.LoadImageToKindClusterWithName(projectimage)
			// ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("installing CRDs")
			cmd := exec.Command("make", "install")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploynamespace", fmt.Sprintf("IMG=%s", projectimage), fmt.Sprintf("NAMESPACE=%s", namespace))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())
		})

		It("should deploy Open5GS instance successfully", func() {
			By("applying Open5GS manifest")
			cmd := exec.Command("kubectl", "apply", "-f", "test/suite_1/samples/net_v1_open5gs-test-1-1.yaml", "-n", namespace)
			_, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the Open5GS instance is created")
			verifyOpen5GSInstance := func() error {
				cmd := exec.Command("kubectl", "get", "open5gses.net.gradiant.org", "-n", namespace)
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "open5gs-test") {
					return fmt.Errorf("Open5GS instance not found")
				}
				return nil
			}
			EventuallyWithOffset(1, verifyOpen5GSInstance, time.Minute, time.Second).Should(Succeed())
		})

		It("should create and deploy all necessary deployments", func() {
			By("validating that all deployments are created")
			verifyDeployments := func() error {
				cmd := exec.Command("kubectl", "get", "deployments", "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				deployments := strings.Fields(string(output))
				expectedDeployments := []string{"open5gs-test-amf", "open5gs-test-ausf", "open5gs-test-bsf", "open5gs-test-mongodb", "open5gs-test-nrf", "open5gs-test-nssf", "open5gs-test-pcf", "open5gs-test-smf", "open5gs-test-udm", "open5gs-test-udr", "open5gs-test-upf", "open5gs-test-webui"}
				for _, deployment := range expectedDeployments {
					if !contains(deployments, deployment) {
						return fmt.Errorf("deployment %s not found", deployment)
					}
				}
				return nil
			}
			EventuallyWithOffset(1, verifyDeployments, time.Minute, time.Second).Should(Succeed())

			By("validating that all deployments have the necessary replicas available")
			verifyDeploymentsAvailable := func() error {
				expectedDeployments := []string{"open5gs-test-amf", "open5gs-test-ausf", "open5gs-test-bsf", "open5gs-test-mongodb", "open5gs-test-nrf", "open5gs-test-nssf", "open5gs-test-pcf", "open5gs-test-smf", "open5gs-test-udm", "open5gs-test-udr", "open5gs-test-upf", "open5gs-test-webui"}
				for _, deployment := range expectedDeployments {
					cmd := exec.Command("kubectl", "get", "deployment", deployment, "-n", namespace, "-o", "jsonpath={.status.availableReplicas}")
					output, err := utils.Run(cmd)
					ExpectWithOffset(2, err).NotTo(HaveOccurred())
					availableReplicas := strings.TrimSpace(string(output))
					if availableReplicas == "0" || availableReplicas == "" {
						return fmt.Errorf("deployment %s does not have available replicas", deployment)
					}
				}
				return nil
			}
			EventuallyWithOffset(1, verifyDeploymentsAvailable, 5*time.Minute, time.Second).Should(Succeed())
		})

		It("should add the new slice restarting the necessary pods", func() {
			By("applying the new Open5GS manifest")
			cmd := exec.Command("kubectl", "apply", "-f", "test/suite_1/samples/net_v1_open5gs-test-1-2.yaml", "-n", namespace)
			_, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("waiting for the open5gs-test-amf deployment to finish rollout")
			cmd = exec.Command("kubectl", "rollout", "status", "deployments/open5gs-test-amf", "-n", namespace)
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("waiting for the open5gs-test-nssf deployment to finish rollout")
			cmd = exec.Command("kubectl", "rollout", "status", "deployments/open5gs-test-nssf", "-n", namespace)
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the new deployments are running successfully")
			deploymentsToCheck := []string{"open5gs-test-amf", "open5gs-test-nssf"}
			for _, deployment := range deploymentsToCheck {
				cmd = exec.Command("kubectl", "get", "deployment", deployment, "-n", namespace, "-o", "jsonpath={.status.availableReplicas}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				availableReplicas := strings.TrimSpace(string(output))
				if availableReplicas == "0" || availableReplicas == "" {
					Fail(fmt.Sprintf("Deployment %s has no available replicas", deployment))
				}
			}
			By("validating that the NSSF ConfigMap contains the correct NSI values")
			verifyConfigMap := func() error {
				cmd := exec.Command("kubectl", "get", "cm", "open5gs-test-nssf", "-n", namespace, "-o", "yaml")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "sst: \"1\"\n                sd: \"0x111111\"") || !strings.Contains(string(output), "sst: \"2\"\n                sd: \"0x222222\"") {
					return fmt.Errorf("NSI values not found in ConfigMap")
				}
				return nil
			}
			EventuallyWithOffset(1, verifyConfigMap, time.Minute, time.Second).Should(Succeed())
			By("validating that the AMF ConfigMap contains the correct values")
			verifyAMFConfigMap := func() error {
				cmd := exec.Command("kubectl", "get", "cm", "open5gs-test-amf", "-n", namespace, "-o", "yaml")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "- sd: \"0x111111\"") {
					return fmt.Errorf("AMF ConfigMap \"- sd: \"0x111111\" \"values not found")
				}
				if !strings.Contains(string(output), "- sd: \"0x222222\"") {
					return fmt.Errorf("AMF ConfigMap \"- sd: \"0x222222\" \"values not found")
				}
				if !strings.Contains(string(output), "mcc: \"999\"") {
					return fmt.Errorf("AMF ConfigMap \"mcc: \"999\"\" values not found")
				}
				if !strings.Contains(string(output), "mnc: \"70\"") {
					return fmt.Errorf("AMF ConfigMap \"mnc: \"70\"\" values not found")
				}
				if !strings.Contains(string(output), "sst: 1") {
					return fmt.Errorf("AMF ConfigMap \"sst: 1\" values not found")
				}
				if !strings.Contains(string(output), "sst: 2") {
					return fmt.Errorf("AMF ConfigMap \"sst: 2\" values not found")
				}
				return nil
			}
			EventuallyWithOffset(1, verifyAMFConfigMap, time.Minute, time.Second).Should(Succeed())
		})

		It("should apply the Open5GSUser manifest and verify in MongoDB", func() {
			By("applying the Open5GSUser manifest")
			cmd := exec.Command("kubectl", "apply", "-f", "test/suite_1/samples/net_v1_open5gsuser-test-1-4.yaml", "-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Open5GSUser entries in MongoDB")
			usersToCheck := []struct {
				Name      string
				Namespace string
				IMSI      string
			}{
				{"open5gsuser-test-1", namespace, "999700000000001"},
				{"open5gsuser-test-2", namespace, "999700000000002"},
				{"open5gsuser-test-3", namespace, "999700000000003"},
				{"open5gsuser-test-4", namespace, "999700000000004"},
			}

			for _, user := range usersToCheck {
				By(fmt.Sprintf("checking MongoDB entry for user %s", user.Name))
				cmd = exec.Command(
					"kubectl", "exec", "deployment/open5gs-test-mongodb", "-n", namespace,
					"--", "sh", "-c",
					fmt.Sprintf(`mongosh open5gs --eval 'db.subscribers.find({"imsi": "%s"}).pretty()'`, user.IMSI),
				)
				output, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to query MongoDB for user %s: %s", user.Name, err))
				Expect(string(output)).To(ContainSubstring(user.IMSI), fmt.Sprintf("IMSI %s not found in MongoDB output", user.IMSI))
			}
		})

		It("should delete the Open5GSUser manifest and verify in MongoDB", func() {
			By("deleting the Open5GSUser manifest")
			cmd := exec.Command("kubectl", "delete", "-f", "test/suite_1/samples/net_v1_open5gsuser-test-1-4.yaml", "-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Open5GSUser entries are removed from MongoDB")
			usersToCheck := []struct {
				Name      string
				Namespace string
				IMSI      string
			}{
				{"open5gsuser-test-1", namespace, "999700000000001"},
				{"open5gsuser-test-2", namespace, "999700000000002"},
				{"open5gsuser-test-3", namespace, "999700000000003"},
				{"open5gsuser-test-4", namespace, "999700000000004"},
			}

			for _, user := range usersToCheck {
				By(fmt.Sprintf("checking MongoDB entry for user %s", user.Name))
				cmd = exec.Command(
					"kubectl", "exec", "deployment/open5gs-test-mongodb", "-n", namespace,
					"--", "sh", "-c",
					fmt.Sprintf(`mongosh open5gs --eval 'db.subscribers.find({"imsi": "%s"}).pretty()'`, user.IMSI),
				)
				output, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to query MongoDB for user %s: %s", user.Name, err))
				Expect(string(output)).NotTo(ContainSubstring(user.IMSI), fmt.Sprintf("IMSI %s found in MongoDB output", user.IMSI))
			}
		})

		It("should deploy and manage Open5GS second instance successfully", func() {
			By("applying Open5GS manifest")
			cmd := exec.Command("kubectl", "apply", "-f", "test/suite_1/samples/net_v1_open5gs-test-1-3.yaml", "-n", namespace)
			_, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the Open5GS instance is created")
			verifyOpen5GSInstance := func() error {
				cmd := exec.Command("kubectl", "get", "open5gses.net.gradiant.org", "-n", namespace)
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "open5gs-test-2") {
					return fmt.Errorf("Open5GS instance not found")
				}
				return nil
			}
			EventuallyWithOffset(1, verifyOpen5GSInstance, time.Minute, time.Second).Should(Succeed())

			By("validating that all deployments are created")
			verifyDeployments := func() error {
				cmd := exec.Command("kubectl", "get", "deployments", "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				deployments := strings.Fields(string(output))
				expectedDeployments := []string{"open5gs-test-2-amf", "open5gs-test-2-ausf", "open5gs-test-2-bsf", "open5gs-test-2-mongodb", "open5gs-test-2-nrf", "open5gs-test-2-nssf", "open5gs-test-2-pcf", "open5gs-test-2-smf", "open5gs-test-2-udm", "open5gs-test-2-udr", "open5gs-test-2-upf", "open5gs-test-2-webui"}
				for _, deployment := range expectedDeployments {
					if !contains(deployments, deployment) {
						return fmt.Errorf("deployment %s not found", deployment)
					}
				}
				return nil
			}
			EventuallyWithOffset(1, verifyDeployments, time.Minute, time.Second).Should(Succeed())

			By("validating that all deployments have the necessary replicas available")
			verifyDeploymentsAvailable := func() error {
				expectedDeployments := []string{"open5gs-test-2-amf", "open5gs-test-2-ausf", "open5gs-test-2-bsf", "open5gs-test-2-mongodb", "open5gs-test-2-nrf", "open5gs-test-2-nssf", "open5gs-test-2-pcf", "open5gs-test-2-smf", "open5gs-test-2-udm", "open5gs-test-2-udr", "open5gs-test-2-upf", "open5gs-test-2-webui"}
				for _, deployment := range expectedDeployments {
					cmd := exec.Command("kubectl", "get", "deployment", deployment, "-n", namespace, "-o", "jsonpath={.status.availableReplicas}")
					output, err := utils.Run(cmd)
					ExpectWithOffset(2, err).NotTo(HaveOccurred())
					availableReplicas := strings.TrimSpace(string(output))
					if availableReplicas == "0" || availableReplicas == "" {
						return fmt.Errorf("deployment %s has no available replicas", deployment)
					}
				}
				return nil
			}
			EventuallyWithOffset(1, verifyDeploymentsAvailable, 5*time.Minute, time.Second).Should(Succeed())

			By("applying the Open5GSUser manifest")
			cmd = exec.Command("kubectl", "apply", "-f", "test/suite_1/samples/net_v1_open5gsuser-test-2-4.yaml", "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Open5GSUser entries in MongoDB")
			usersToCheck := []struct {
				Name      string
				Namespace string
				IMSI      string
			}{
				{"open5gsuser-test-1", namespace, "999700000000001"},
				{"open5gsuser-test-2", namespace, "999700000000002"},
				{"open5gsuser-test-3", namespace, "999700000000003"},
				{"open5gsuser-test-4", namespace, "999700000000004"},
			}

			for _, user := range usersToCheck {
				By(fmt.Sprintf("checking MongoDB entry for user %s", user.Name))
				cmd = exec.Command(
					"kubectl", "exec", "deployment/open5gs-test-2-mongodb", "-n", namespace,
					"--", "sh", "-c",
					fmt.Sprintf(`mongosh open5gs --eval 'db.subscribers.find({"imsi": "%s"}).pretty()'`, user.IMSI),
				)
				output, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to query MongoDB for user %s: %s", user.Name, err))
				Expect(string(output)).To(ContainSubstring(user.IMSI), fmt.Sprintf("IMSI %s not found in MongoDB output", user.IMSI))
			}

			By("deleting the Open5GSUser manifest")
			cmd = exec.Command("kubectl", "delete", "-f", "test/suite_1/samples/net_v1_open5gsuser-test-2-4.yaml", "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Open5GSUser entries are removed from MongoDB")
			for _, user := range usersToCheck {
				By(fmt.Sprintf("checking MongoDB entry for user %s", user.Name))
				cmd = exec.Command(
					"kubectl", "exec", "deployment/open5gs-test-mongodb", "-n", namespace,
					"--", "sh", "-c",
					fmt.Sprintf(`mongosh open5gs --eval 'db.subscribers.find({"imsi": "%s"}).pretty()'`, user.IMSI),
				)
				output, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to query MongoDB for user %s: %s", user.Name, err))
				Expect(string(output)).NotTo(ContainSubstring(user.IMSI), fmt.Sprintf("IMSI %s found in MongoDB output", user.IMSI))
			}
		})

		It("should deploy and manage Open5GS third instance in another namespace successfully", func() {
			By("applying Open5GS manifest")
			cmd := exec.Command("kubectl", "apply", "-f", "test/suite_1/samples/net_v1_open5gs-test-1-4.yaml", "-n", namespace2)
			_, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the Open5GS instance is created")
			verifyOpen5GSInstance := func() error {
				cmd := exec.Command("kubectl", "get", "open5gses.net.gradiant.org", "-n", namespace2)
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "open5gs-test-2") {
					return fmt.Errorf("Open5GS instance not found")
				}
				return nil
			}
			EventuallyWithOffset(1, verifyOpen5GSInstance, time.Minute, time.Second).Should(Succeed())

			By("validating that all deployments are created")
			verifyDeployments := func() error {
				cmd := exec.Command("kubectl", "get", "deployments", "-n", namespace2, "-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				deployments := strings.Fields(string(output))
				expectedDeployments := []string{"open5gs-test-2-amf", "open5gs-test-2-ausf", "open5gs-test-2-bsf", "open5gs-test-2-mongodb", "open5gs-test-2-nrf", "open5gs-test-2-nssf", "open5gs-test-2-pcf", "open5gs-test-2-smf", "open5gs-test-2-udm", "open5gs-test-2-udr", "open5gs-test-2-upf", "open5gs-test-2-webui"}
				for _, deployment := range expectedDeployments {
					if !contains(deployments, deployment) {
						return fmt.Errorf("deployment %s not found", deployment)
					}
				}
				return nil
			}
			EventuallyWithOffset(1, verifyDeployments, time.Minute, time.Second).Should(Succeed())

			By("validating that all deployments have the necessary replicas available")
			verifyDeploymentsAvailable := func() error {
				expectedDeployments := []string{"open5gs-test-2-amf", "open5gs-test-2-ausf", "open5gs-test-2-bsf", "open5gs-test-2-mongodb", "open5gs-test-2-nrf", "open5gs-test-2-nssf", "open5gs-test-2-pcf", "open5gs-test-2-smf", "open5gs-test-2-udm", "open5gs-test-2-udr", "open5gs-test-2-upf", "open5gs-test-2-webui"}
				for _, deployment := range expectedDeployments {
					cmd := exec.Command("kubectl", "get", "deployment", deployment, "-n", namespace2, "-o", "jsonpath={.status.availableReplicas}")
					output, err := utils.Run(cmd)
					ExpectWithOffset(2, err).NotTo(HaveOccurred())
					availableReplicas := strings.TrimSpace(string(output))
					if availableReplicas == "0" || availableReplicas == "" {
						return fmt.Errorf("deployment %s has no available replicas", deployment)
					}
				}
				return nil
			}
			EventuallyWithOffset(1, verifyDeploymentsAvailable, 5*time.Minute, time.Second).Should(Succeed())

			By("applying the Open5GSUser manifest")
			cmd = exec.Command("kubectl", "apply", "-f", "test/suite_1/samples/net_v1_open5gsuser-test-3-4.yaml", "-n", namespace2)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Open5GSUser entries in MongoDB")
			usersToCheck := []struct {
				Name      string
				Namespace string
				IMSI      string
			}{
				{"open5gsuser-test-1", namespace2, "999700000000001"},
				{"open5gsuser-test-2", namespace2, "999700000000002"},
				{"open5gsuser-test-3", namespace2, "999700000000003"},
				{"open5gsuser-test-4", namespace2, "999700000000004"},
			}

			for _, user := range usersToCheck {
				By(fmt.Sprintf("checking MongoDB entry for user %s", user.Name))
				cmd = exec.Command(
					"kubectl", "exec", "deployment/open5gs-test-2-mongodb", "-n", namespace2,
					"--", "sh", "-c",
					fmt.Sprintf(`mongosh open5gs --eval 'db.subscribers.find({"imsi": "%s"}).pretty()'`, user.IMSI),
				)
				output, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to query MongoDB for user %s: %s", user.Name, err))
				Expect(string(output)).To(ContainSubstring(user.IMSI), fmt.Sprintf("IMSI %s not found in MongoDB output", user.IMSI))
			}

			By("deleting the Open5GSUser manifest")
			cmd = exec.Command("kubectl", "delete", "-f", "test/suite_1/samples/net_v1_open5gsuser-test-3-4.yaml", "-n", namespace2)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Open5GSUser entries are removed from MongoDB")
			for _, user := range usersToCheck {
				By(fmt.Sprintf("checking MongoDB entry for user %s", user.Name))
				cmd = exec.Command(
					"kubectl", "exec", "deployment/open5gs-test-2-mongodb", "-n", namespace2,
					"--", "sh", "-c",
					fmt.Sprintf(`mongosh open5gs --eval 'db.subscribers.find({"imsi": "%s"}).pretty()'`, user.IMSI),
				)
				output, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to query MongoDB for user %s: %s", user.Name, err))
				Expect(string(output)).NotTo(ContainSubstring(user.IMSI), fmt.Sprintf("IMSI %s found in MongoDB output", user.IMSI))
			}
		})
		It("should deploy and verify metrics, service accounts, and service monitors", func() {
			By("applying Open5GS manifest with metrics enabled")
			cmd := exec.Command("kubectl", "apply", "-f", "test/suite_1/samples/net_v1_open5gs-test-1-5.yaml", "-n", namespace)
			_, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the Open5GS instance is created")
			verifyOpen5GSInstance := func() error {
				cmd := exec.Command("kubectl", "get", "open5gses.net.gradiant.org", "-n", namespace)
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "open5gs-test") {
					return fmt.Errorf("Open5GS instance not found")
				}
				return nil
			}
			EventuallyWithOffset(1, verifyOpen5GSInstance, time.Minute, time.Second).Should(Succeed())

			By("validating that all deployments are created")
			verifyDeployments := func() error {
				cmd := exec.Command("kubectl", "get", "deployments", "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				deployments := strings.Fields(string(output))
				expectedDeployments := []string{"open5gs-test-5-amf", "open5gs-test-5-ausf", "open5gs-test-5-bsf", "open5gs-test-5-mongodb", "open5gs-test-5-nrf", "open5gs-test-5-nssf", "open5gs-test-5-pcf", "open5gs-test-5-smf", "open5gs-test-5-udm", "open5gs-test-5-udr", "open5gs-test-5-upf", "open5gs-test-5-webui"}
				for _, deployment := range expectedDeployments {
					if !contains(deployments, deployment) {
						return fmt.Errorf("deployment %s not found", deployment)
					}
				}
				return nil
			}
			EventuallyWithOffset(1, verifyDeployments, time.Minute, time.Second).Should(Succeed())

			By("validating that all service accounts are created")
			verifyServiceAccounts := func() error {
				cmd := exec.Command("kubectl", "get", "serviceaccounts", "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				serviceAccounts := strings.Fields(string(output))
				expectedServiceAccounts := []string{"open5gs-test-5-amf", "open5gs-test-5-ausf", "open5gs-test-5-bsf", "open5gs-test-5-mongodb", "open5gs-test-5-nrf", "open5gs-test-5-nssf", "open5gs-test-5-pcf", "open5gs-test-5-smf", "open5gs-test-5-udm", "open5gs-test-5-udr", "open5gs-test-5-upf", "open5gs-test-5-webui"}
				for _, serviceAccount := range expectedServiceAccounts {
					if !contains(serviceAccounts, serviceAccount) {
						return fmt.Errorf("service account %s not found", serviceAccount)
					}
				}
				return nil
			}
			EventuallyWithOffset(1, verifyServiceAccounts, time.Minute, time.Second).Should(Succeed())

			By("validating that all service monitors are created")
			verifyServiceMonitors := func() error {
				cmd := exec.Command("kubectl", "get", "servicemonitors", "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				serviceMonitors := strings.Fields(string(output))
				expectedServiceMonitors := []string{"open5gs-test-5-amf", "open5gs-test-5-pcf", "open5gs-test-5-smf", "open5gs-test-5-upf"}
				for _, serviceMonitor := range expectedServiceMonitors {
					if !contains(serviceMonitors, serviceMonitor) {
						return fmt.Errorf("service monitor %s not found", serviceMonitor)
					}
				}
				return nil
			}
			EventuallyWithOffset(1, verifyServiceMonitors, time.Minute, time.Second).Should(Succeed())
		})
	})
})

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

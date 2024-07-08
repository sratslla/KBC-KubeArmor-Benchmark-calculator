package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the benchmark process and apply all the relevant resources.",
	Long:  `Start the benchmark process and apply all the relevant resources.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("start called")
		if isKubernetesClusterRunning() {
			fmt.Println("Kubernetes cluster is running")
		} else {
			fmt.Println("Kubernetes cluster is not running or accessible")
		}
		REPO_URL := "https://raw.githubusercontent.com/sratslla/KBC-KubeArmor-Benchmark-calculator/main"
		YAML_FILE_PATH := "manifests/kubernetes-manifests.yaml"
		YAML_FILE_PATH2 := "manifests/loadgenerator_ui.yaml"
		YAML_FILE_PATH3 := "manifests/kube-static-metrics.yaml"
		YAML_FILE_PATH4 := "manifests/prometheusComponent.yaml"
		err := applyManifestFromGitHub(REPO_URL, YAML_FILE_PATH)
		if err != nil {
			fmt.Println("Error applying manifest:", err)
			os.Exit(1)
		}
		err2 := applyManifestFromGitHub(REPO_URL, YAML_FILE_PATH2)
		if err2 != nil {
			fmt.Println("Error applying manifest:", err2)
			os.Exit(1)
		}
		err3 := applyManifestFromGitHub(REPO_URL, YAML_FILE_PATH3)
		if err3 != nil {
			fmt.Println("Error applying manifest:", err3)
			os.Exit(1)
		}
		err4 := applyManifestFromGitHub(REPO_URL, YAML_FILE_PATH4)
		if err4 != nil {
			fmt.Println("Error applying manifest:", err4)
			os.Exit(1)
		}

		autoscaleDeployment("cartservice", 50, 2, 400)
		autoscaleDeployment("currencyservice", 50, 2, 400)
		autoscaleDeployment("emailservice", 50, 2, 400)
		autoscaleDeployment("checkoutservice", 50, 2, 400)
		autoscaleDeployment("frontend", 50, 5, 400)
		autoscaleDeployment("paymentservice", 50, 2, 400)
		autoscaleDeployment("productcatalogservice", 50, 2, 400)
		autoscaleDeployment("recommendationservice", 50, 2, 400)
		autoscaleDeployment("redis-cart", 50, 1, 400)
		autoscaleDeployment("shippingservice", 50, 2, 400)
		autoscaleDeployment("adservice", 50, 1, 400)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func isKubernetesClusterRunning() bool {
	cmd := exec.Command("kubectl", "cluster-info")

	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return false
	}
	// fmt.Println(cmd, output.String())
	return true
}

func applyManifestFromGitHub(repoURL, yamlFilePath string) error {
	cmd := exec.Command("kubectl", "apply", "-f", fmt.Sprintf("%s/%s", repoURL, yamlFilePath))
	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error applying manifest: %v\n%s", err, output.String())
	}
	fmt.Println("Manifest applied successfully.", output.String())
	return nil
}

func autoscaleDeployment(deploymentName string, cpuPercent, minReplicas, maxReplicas int) {
	cmd := exec.Command("kubectl", "autoscale", "deployment", deploymentName,
		fmt.Sprintf("--cpu-percent=%d", cpuPercent),
		fmt.Sprintf("--min=%d", minReplicas),
		fmt.Sprintf("--max=%d", maxReplicas))

	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error autoscaling deployment %s: %v\n%s", deploymentName, err, output.String())
	} else {
		fmt.Printf("Deployment %s autoscaled successfully.\n", deploymentName)
	}
}

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the benchmark process",
	Long:  `Start the benchmark process`,
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
		// ticker := time.NewTicker(5 * time.Second)
		// defer ticker.Stop()

		// done := make(chan bool)
		// condition := false

		time.Sleep(200 * time.Second)

		externalIP, err := getExternalIP()
		if err != nil {
			fmt.Println("Error getting external IP:", err)
			return
		}

		prometheusURL := fmt.Sprintf("http://%s:30000", externalIP)
		promClient, err := NewPrometheusClient(prometheusURL)
		if err != nil {
			fmt.Println("Error creating Prometheus client:", err)
			return
		}

		query := `sum(rate(container_cpu_usage_seconds_total{pod=~"frontend-.*", container = "", namespace="default"}[1m]))`
		cpuResult, err := QueryPrometheus(promClient, query)
		if err != nil {
			fmt.Println("Error querying Prometheus for CPU metrics:", err)
			return
		}
		fmt.Println("CPU Query result:", cpuResult)

		locustQuery := `locust_users{job="locust"}`
		locustResult, err := QueryPrometheus(promClient, locustQuery)
		if err != nil {
			fmt.Println("Error querying Prometheus for Locust metrics:", err)
			return
		}
		fmt.Println("Locust Query result:", locustResult)
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

func NewPrometheusClient(prometheusURL string) (v1.API, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(client), nil
}

func QueryPrometheus(api v1.API, query string) (model.Value, error) {
	result, warnings, err := api.Query(context.Background(), query, time.Now())
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		fmt.Println("Warnings received during query execution:", warnings)
	}
	return result, nil
}

func getExternalIP() (string, error) {
	cmd := exec.Command("kubectl", "get", "nodes", "-o", "json")

	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error getting nodes: %v", err)
	}

	var nodes struct {
		Items []struct {
			Status struct {
				Addresses []struct {
					Type    string `json:"type"`
					Address string `json:"address"`
				} `json:"addresses"`
			} `json:"status"`
		} `json:"items"`
	}

	err = json.Unmarshal(output.Bytes(), &nodes)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling nodes JSON: %v", err)
	}

	for _, node := range nodes.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == "ExternalIP" {
				return address.Address, nil
			}
		}
	}

	return "", fmt.Errorf("no external IP found")
}

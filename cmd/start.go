package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the benchmark process and apply all the relevant resources.",
	Long:  `Start the benchmark process and apply all the relevant resources.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("start called")
		// Check if cluster is running then apply manifest files and start autoscalling
		if isKubernetesClusterRunning() {
			fmt.Println("Kubernetes cluster is running ")
		} else {
			fmt.Println("Kubernetes cluster is not running or accessible")
		}
		REPO_URL := "https://raw.githubusercontent.com/sratslla/KBC-KubeArmor-Benchmark-calculator/main/manifests"
		manifestPaths := []string{
			"kubernetes-manifests.yaml",
			"kube-static-metrics.yaml",
			"prometheusComponent.yaml",
		}
		for _, manifestmanifestPath := range manifestPaths {
			err := applyManifestFromGitHub(REPO_URL, manifestmanifestPath)
			if err != nil {
				fmt.Println("Error applying manifest:", err)
				os.Exit(1)
			}
		}

		// TODO - optimize it using a Loop
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

		// TODO - Automatically locust start using flag

		// Start the benchmark
		time.Sleep(1 * time.Minute)

		// getting externalIP of a node
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

		locustQuery := `locust_users{job="locust"}`

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		// Check if locust users have reached required amount every 10 sec.
		for range ticker.C {
			locustResult, err := QueryPrometheus(promClient, locustQuery)
			if err != nil {
				fmt.Println("Error querying Prometheus for Locust metrics:", err)
				return
			}

			locustUsers := 0
			if locustResult.Type() == model.ValVector {
				vector := locustResult.(model.Vector)
				for _, sample := range vector {
					locustUsers = int(sample.Value)
					fmt.Printf("locustUsers %v", locustUsers)
				}
			}

			if locustUsers >= 300 {
				fmt.Println("locust users reached 300. data will be fetched now to calculate avg benchmark.")
				break
			}

			fmt.Printf("\rWaiting for locust_users to reach 300\n")
		}

		// waiting 1 min for resources to stabalization and 10 mins for calculating avg
		time.Sleep(11 * time.Minute)

		calculateBenchMark(promClient)

		deployments := []string{
			"cartservice",
			"currencyservice",
			"emailservice",
			"checkoutservice",
			"frontend",
			"paymentservice",
			"productcatalogservice",
			"recommendationservice",
			"redis-cart",
			"shippingservice",
			"adservice",
		}

		replicasMap := make(map[string]int)

		for _, deployment := range deployments {
			replicas, err := getCurrentReplicas(deployment)
			if err != nil {
				fmt.Printf("Error getting current replicas %s : %v\n", deployment, err)
				continue
			}
			replicasMap[deployment] = replicas
		}

		for deployment, replicas := range replicasMap {
			fmt.Printf("%s: %d replicas\n", deployment, replicas)
		}

		for _, deployment := range deployments {
			err := deleteHPA(deployment)
			if err != nil {
				fmt.Printf("Error deleting HPA for %s: %v\n", deployment, err)
				continue
			}
		}

		for deployment, replicas := range replicasMap {
			err := scaleDeployment(deployment, replicas)
			if err != nil {
				fmt.Printf("Error svaling deployment %s to %d replicas: %v\n", deployment, replicas, err)
				continue
			}
		}

		err = installKubearmor()
		if err != nil {
			fmt.Printf("Error installing Kubearmor: %v\n", err)
			return
		}

		err = runKarmorInstall()
		if err != nil {
			fmt.Printf("Error running karmor install: %v\n", err)
			return
		}

		err = configureKubearmorRelay()
		if err != nil {
			fmt.Printf("Error configuring kubearmor relay: %v\n", err)
			return
		}

		fmt.Println("exec3 called")
		time.Sleep(5 * time.Minute)
		calculateBenchMark(promClient)

		changeVisiblity("process")
		time.Sleep(5 * time.Minute)
		calculateBenchMark(promClient)

		changeVisiblity("process, file")
		time.Sleep(5 * time.Minute)
		calculateBenchMark(promClient)

		changeVisiblity("process, network")
		time.Sleep(5 * time.Minute)
		calculateBenchMark(promClient)

		changeVisiblity("process, network, file")
		time.Sleep(5 * time.Minute)
		calculateBenchMark(promClient)
		changeVisiblity("none")

		// Apply Policies and check
		// Process Policy
		err = applyManifestFromGitHub(REPO_URL, "policy-process.yaml")
		if err != nil {
			fmt.Println("Error applying manifest:", err)
		}
		time.Sleep(5 * time.Minute)
		calculateBenchMark(promClient)

		// Process and Network Policy
		err = applyManifestFromGitHub(REPO_URL, "policy-file.yaml")
		if err != nil {
			fmt.Println("Error applying manifest:", err)
		}
		time.Sleep(5 * time.Minute)
		calculateBenchMark(promClient)

		// Process, Network and File Policy
		// err = applyManifestFromGitHub(REPO_URL, "policy-process.yaml")
		// if err != nil {
		// 	fmt.Println("Error applying manifest:", err)
		// }
		// calculateBenchMark(promClient)

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

// Make a Prometheus Client
func NewPrometheusClient(prometheusURL string) (v1.API, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(client), nil
}

// Query using PromQL
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

// Get the externalIP of a node as prometheus is running on "externalIP:30000"
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

func calculateBenchMark(promClient v1.API) {
	CPUQueries := map[string]string{
		"Frontend":              `sum(rate(container_cpu_usage_seconds_total{pod=~"frontend-.*", container="", namespace="default"}[5m])) * 1000`,
		"Adservice":             `sum(rate(container_cpu_usage_seconds_total{pod=~"adservice-.*", container="", namespace="default"}[5m])) * 1000`,
		"Cartservice":           `sum(rate(container_cpu_usage_seconds_total{pod=~"cartservice-.*", container="", namespace="default"}[5m])) * 1000`,
		"Checkoutservice":       `sum(rate(container_cpu_usage_seconds_total{pod=~"checkoutservice-.*", container="", namespace="default"}[5m])) * 1000`,
		"Currencyservice":       `sum(rate(container_cpu_usage_seconds_total{pod=~"currencyservice-.*", container="", namespace="default"}[5m])) * 1000`,
		"Emailservice":          `sum(rate(container_cpu_usage_seconds_total{pod=~"emailservice-.*", container="", namespace="default"}[5m])) * 1000`,
		"Loadgenerator":         `sum(rate(container_cpu_usage_seconds_total{pod=~"loadgenerator-.*", container="", namespace="default"}[5m])) * 1000`,
		"Paymentservice":        `sum(rate(container_cpu_usage_seconds_total{pod=~"paymentservice-.*", container="", namespace="default"}[5m])) * 1000`,
		"Productcatalogservice": `sum(rate(container_cpu_usage_seconds_total{pod=~"productcatalogservice-.*", container="", namespace="default"}[5m])) * 1000`,
		"Recommendationservice": `sum(rate(container_cpu_usage_seconds_total{pod=~"recommendationservice-.*", container="", namespace="default"}[5m])) * 1000`,
		"Redis":                 `sum(rate(container_cpu_usage_seconds_total{pod=~"redis-.*", container="", namespace="default"}[5m])) * 1000`,
		"Shippingservice":       `sum(rate(container_cpu_usage_seconds_total{pod=~"shippingservice-.*", container="", namespace="default"}[5m])) * 1000`,
	}
	MemoryQueries := map[string]string{
		"Frontend":              `sum(container_memory_usage_bytes{pod=~"frontend-.*", namespace="default"}) / 1024 / 1024`,
		"Adservice":             `sum(container_memory_usage_bytes{pod=~"adservice-.*", namespace="default"}) / 1024 / 1024`,
		"Cartservice":           `sum(container_memory_usage_bytes{pod=~"cartservice-.*", namespace="default"}) / 1024 / 1024`,
		"Checkoutservice":       `sum(container_memory_usage_bytes{pod=~"checkoutservice-.*", namespace="default"}) / 1024 / 1024`,
		"Currencyservice":       `sum(container_memory_usage_bytes{pod=~"currencyservice-.*", namespace="default"}) / 1024 / 1024`,
		"Emailservice":          `sum(container_memory_usage_bytes{pod=~"emailservice-.*", namespace="default"}) / 1024 / 1024`,
		"Loadgenerator":         `sum(container_memory_usage_bytes{pod=~"loadgenerator-.*", namespace="default"}) / 1024 / 1024`,
		"Paymentservice":        `sum(container_memory_usage_bytes{pod=~"paymentservice-.*", namespace="default"}) / 1024 / 1024`,
		"Productcatalogservice": `sum(container_memory_usage_bytes{pod=~"productcatalogservice-.*", namespace="default"}) / 1024 / 1024`,
		"Recommendationservice": `sum(container_memory_usage_bytes{pod=~"recommendationservice-.*", namespace="default"}) / 1024 / 1024`,
		"Redis":                 `sum(container_memory_usage_bytes{pod=~"redis-.*", namespace="default"}) / 1024 / 1024`,
		"Shippingservice":       `sum(container_memory_usage_bytes{pod=~"shippingservice-.*", namespace="default"}) / 1024 / 1024`,
	}

	fmt.Printf("CPU Usage \n")
	for serviceName, query := range CPUQueries {
		result, err := QueryPrometheus(promClient, query)
		if err != nil {
			fmt.Printf("Error querying Prometheus for CPU metrics (%s): %v\n", serviceName, err)
			return
		}
		fmt.Printf("%s CPU usage: %v\n", serviceName, result)
	}
	fmt.Printf("Memory \n")
	for serviceName, query := range MemoryQueries {
		result, err := QueryPrometheus(promClient, query)
		if err != nil {
			fmt.Printf("Error querying Prometheus for Memory metrics (%s): %v\n", serviceName, err)
			return
		}
		fmt.Printf("%s Memory usage: %v\n", serviceName, result)
	}

	locustThroughput := `avg_over_time(locust_requests_current_rps{job="locust", name="Aggregated"}[5m])`
	locustThroughputResult, err := QueryPrometheus(promClient, locustThroughput)
	if err != nil {
		fmt.Println("Error querying Prometheus for Locust metrics:", err)
		return
	}
	re := regexp.MustCompile(`=>\s+([0-9.]+)\s+@`)
	match := re.FindStringSubmatch(locustThroughputResult.String())
	if len(match) > 1 {
		value := match[1]
		fmt.Println("Locust Throughput Average for last 10mins:", value)
	} else {
		fmt.Println("Error: Could not parse the result")
	}
	fmt.Println("=========================================")
}

func getCurrentReplicas(deploymentName string) (int, error) {
	cmd := exec.Command("kubectl", "get", "hpa", deploymentName, "-o", "jsonpath={.status.currentReplicas}")
	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("error getting the current replicas %V \n%s", err, output.String())
	}
	var replicas int
	err = json.Unmarshal(output.Bytes(), &replicas)
	if err != nil {
		return 0, fmt.Errorf("error unmarshalling current replicas: %v", err)
	}
	return replicas, nil
}

func scaleDeployment(deploymentName string, replicas int) error {
	cmd := exec.Command("kubectl", "scale", "deployment", deploymentName, fmt.Sprintf("--replicas=%d", replicas))
	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error scaling deployment: %v\n%s", err, output.String())
	}
	fmt.Printf("Deployment %s scaled to %d replicas successfully.\n", deploymentName, replicas)
	return nil
}

func deleteHPA(deploymentName string) error {
	cmd := exec.Command("kubectl", "delete", "hpa", deploymentName)
	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error deleting HPA: %v\n%s", err, output.String())
	}
	fmt.Printf("HPA for %s deleted successfully.\n", deploymentName)
	return nil
}

func installKubearmor() error {
	cmd := exec.Command("sh", "-c", "curl -sfL http://get.kubearmor.io/ | sudo sh -s -- -b /usr/local/bin")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error installing Kubearmor: %v\n%s", err, output.String())
	}
	fmt.Println("Kubearmor installed successfully.")
	return nil
}

func runKarmorInstall() error {
	cmd := exec.Command("karmor", "install")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running karmor install: %v\n%s", err, output.String())
	}
	fmt.Println("karmor installed successfully.")
	return nil
}

func configureKubearmorRelay() error {
	// Command to patch the kubearmor-relay deployment
	patch := `{"spec": {"template": {"spec": {"tolerations": [{"key": "color", "operator": "Equal", "value": "blue", "effect": "NoSchedule"}], "nodeSelector": {"nodetype": "node1"}}}}}`
	cmd := exec.Command("kubectl", "patch", "deploy", "kubearmor-relay", "-n", "kubearmor", "--patch", patch)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error configuring kubearmor relay: %v\n%s", err, output.String())
	}
	fmt.Println("kubearmor-relay configured successfully.")
	return nil
}

func changeVisiblity(visiblityMode string) error {
	cmd := exec.Command("kubectl", "annotate", "ns", "default", fmt.Sprintf("kubearmor-visibility=%s", visiblityMode), "--overwrite")
	var output bytes.Buffer
	cmd.Stderr = &output
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error annotating namespace with %s: %v\n%s", visiblityMode, err, output.String())
	}
	fmt.Printf("Namespace annotated with visibility type %s successfully.\n", visiblityMode)
	return nil
}

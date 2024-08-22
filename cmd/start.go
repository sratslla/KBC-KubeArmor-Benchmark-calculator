package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/spf13/cobra"
)

type CaseEnum string

const (
	WithoutKubeArmor        CaseEnum = "WithoutKubeArmor"
	WithKubeArmorPolicy     CaseEnum = "WithKubeArmorPolicy"
	WithKubeArmorVisibility CaseEnum = "WithKubeArmorVisibility"
)

type ResourceUsage struct {
	Name   string
	CPU    float32
	Memory float32
}

// WK WOKP WOKV
type SingleCaseReport struct {
	Case                    CaseEnum // Case type: WithoutKubeArmor, WithKubeArmor, WithKubeArmorPolicy, WithKubeArmorVisibility
	MetricName              string   // Metric type: policy type, visibility type, none
	Users                   int32
	KubearmorResourceUsages []ResourceUsage
	Throughput              float32
	PercentageDrop          float32
	ResourceUsages          []ResourceUsage // List of resource usages
}

type FinalReport struct {
	Reports []SingleCaseReport
}

var finalReport FinalReport

var defaultThroughput float32

var defaultUsers int32 = 600

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the benchmark process and apply all the relevant  resources.",
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
			"loadgenerator_ui.yaml",
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

		// TODO - Automatically locust start using flag - DONE

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

			if locustUsers >= int(defaultUsers) {
				fmt.Println("locust users reached 300. data will be fetched now to calculate avg benchmark.")
				break
			}

			fmt.Printf("\rWaiting for locust_users to reach 300\n")
		}

		// waiting 1 min for resources to stabalization and 10 mins for calculating avg
		time.Sleep(11 * time.Minute)

		calculateBenchMark(promClient, WithoutKubeArmor, "")

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
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorVisibility, "none")

		changeVisiblity("process")
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorVisibility, "process")

		changeVisiblity("process, file")
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorVisibility, "process & file")

		changeVisiblity("process, network")
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorVisibility, "process & network")

		changeVisiblity("process, network, file")
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorVisibility, "process, network & file")
		changeVisiblity("none")

		// Apply Policies and check

		// Process Policy.
		err = applyManifestFromGitHub(REPO_URL, "policy-process.yaml")
		if err != nil {
			fmt.Println("Error applying manifest:", err)
		}
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorPolicy, "process")

		// Process and File Policy.
		err = applyManifestFromGitHub(REPO_URL, "policy-file.yaml")
		if err != nil {
			fmt.Println("Error applying manifest:", err)
		}
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorPolicy, "process & file")

		// Process, File and Network.
		err = applyManifestFromGitHub(REPO_URL, "policy-network.yaml")
		if err != nil {
			fmt.Println("Error applying manifest:", err)
		}
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorPolicy, "process, file and network")

		// print final report
		printFinalReport(finalReport)

		// Write the data to markdown file.
		templateContent, err := ioutil.ReadFile("report_template.md")
		if err != nil {
			fmt.Println("Error reading template file:", err)
			return
		}

		// Create a new template and parse the Markdown template content
		tmpl, err := template.New("markdown").Parse(string(templateContent))
		if err != nil {
			fmt.Println("Error parsing template:", err)
			return
		}

		// Create a file to write the Markdown content
		file, err := os.Create("final_report.md")
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer file.Close()

		// Execute the template and write the content to the file
		err = tmpl.Execute(file, finalReport)
		if err != nil {
			fmt.Println("Error executing template:", err)
			return
		}

		fmt.Println("Markdown file created successfully!")

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
		fmt.Println("error applying manifest", output.String())
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

func calculateBenchMark(promClient v1.API, scenario CaseEnum, Metric string) {
	// TODO - Use fmt.Sprintf and add the service name from a variable.
	queries := map[string]map[string]string{
		"FRONTEND": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"frontend-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"frontend-.*", namespace="default"}) / 1024 / 1024`,
		},
		"AD": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"adservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"adservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"CART": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"cartservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"cartservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"CHECKOUT": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"checkoutservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"checkoutservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"CURRENCY": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"currencyservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"currencyservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"EMAIL": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"emailservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"emailservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"LOAD": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"loadgenerator-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"loadgenerator-.*", namespace="default"}) / 1024 / 1024`,
		},
		"PAYMENT": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"paymentservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"paymentservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"PRODUCT": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"productcatalogservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"productcatalogservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"RECOMMENDATION": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"recommendationservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"recommendationservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"REDIS": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"redis-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"redis-.*", namespace="default"}) / 1024 / 1024`,
		},
		"SHIPPING": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"shippingservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"shippingservice-.*", namespace="default"}) / 1024 / 1024`,
		},
	}

	var resourceUsages []ResourceUsage

	for serviceName, queryMap := range queries {
		cpuQuery := queryMap["cpu"]
		memoryQuery := queryMap["memory"]

		cpuTempResult, err := QueryPrometheus(promClient, cpuQuery)
		if err != nil {
			fmt.Printf("Error querying Prometheus for CPU metrics (%s): %v\n", serviceName, err)
			continue
		}
		memoryTempResult, err := QueryPrometheus(promClient, memoryQuery)
		if err != nil {
			fmt.Printf("Error querying Prometheus for Memory metrics (%s): %v\n", serviceName, err)
			continue
		}
		cpuResult, _ := parseUsage(cpuTempResult)
		memoryResult, _ := parseUsage(memoryTempResult)

		resourceUsage := ResourceUsage{
			Name:   serviceName,
			CPU:    cpuResult,
			Memory: memoryResult,
		}
		resourceUsages = append(resourceUsages, resourceUsage)
	}

	locustThroughputQuery := `avg_over_time(locust_requests_current_rps{job="locust", name="Aggregated"}[5m])`
	locustThroughput, err := QueryPrometheus(promClient, locustThroughputQuery)
	if err != nil {
		fmt.Println("Error querying Prometheus for Locust metrics:", err)
		return
	}
	locustThroughputResult, _ := parseUsage(locustThroughput)

	kubearmorQueries := map[string]map[string]string{
		"KUBEARMOR": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"kubearmor-bpf-containerd-.*", container="", namespace="kubearmor"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"kubearmor-bpf-containerd-.*", namespace="kubearmor"}) / 1024 / 1024`,
		},
		"KUBEARMOR-RELAY": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"kubearmor-relay-.*", container="", namespace="kubearmor"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"kubearmor-relay-.*", namespace="kubearmor"}) / 1024 / 1024`,
		},
	}

	var KubearmorResourceUsages []ResourceUsage
	for serviceName, queryMap := range kubearmorQueries {
		cpuQuery := queryMap["cpu"]
		memoryQuery := queryMap["memory"]

		cpuTempResult, err := QueryPrometheus(promClient, cpuQuery)
		if err != nil {
			fmt.Printf("Error querying Prometheus for CPU metrics (%s): %v\n", serviceName, err)
			continue
		}
		memoryTempResult, err := QueryPrometheus(promClient, memoryQuery)
		if err != nil {
			fmt.Printf("Error querying Prometheus for Memory metrics (%s): %v\n", serviceName, err)
			continue
		}
		cpuResult, _ := parseUsage(cpuTempResult)
		memoryResult, _ := parseUsage(memoryTempResult)

		resourceUsage := ResourceUsage{
			Name:   serviceName,
			CPU:    cpuResult,
			Memory: memoryResult,
		}
		KubearmorResourceUsages = append(KubearmorResourceUsages, resourceUsage)
	}

	if scenario == "WithoutKubeArmor" {
		defaultThroughput = locustThroughputResult
	}
	partialReport := SingleCaseReport{
		Case:                    scenario,
		MetricName:              Metric,
		Users:                   defaultUsers,
		KubearmorResourceUsages: KubearmorResourceUsages,
		Throughput:              locustThroughputResult,
		PercentageDrop:          ((defaultThroughput - locustThroughputResult) / defaultThroughput) * 100,
		ResourceUsages:          resourceUsages,
	}
	finalReport.Reports = append(finalReport.Reports, partialReport)
}

func parseUsage(result model.Value) (float32, error) {
	re := regexp.MustCompile(`=>\s+([0-9.]+)\s+@`)
	matches := re.FindStringSubmatch(result.String())
	if len(matches) > 1 {
		// Convert the extracted string to float32
		value, err := strconv.ParseFloat(matches[1], 32)
		if err != nil {
			return 0, fmt.Errorf("unable to convert string to float32: %v", err)
		}
		return float32(value), nil
	} else {
		return 0, fmt.Errorf("unable to parse result: %s", result)
	}
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
func printFinalReport(report FinalReport) {
	fmt.Println("Final Report:")
	for _, caseReport := range report.Reports {
		fmt.Printf("\nCase: %s\n", caseReport.Case)
		fmt.Printf("Metric Name: %s\n", caseReport.MetricName)
		fmt.Printf("Users: %d\n", caseReport.Users)
		for _, usage := range caseReport.KubearmorResourceUsages {
			fmt.Printf("  Service: %s\n", usage.Name)
			fmt.Printf("    CPU: %.2f\n", usage.CPU)
			fmt.Printf("    Memory: %.2f MB\n", usage.Memory)
		}
		fmt.Printf("Throughput: %.2f\n", caseReport.Throughput)
		fmt.Printf("Percentage Drop: %.2f%%\n", caseReport.PercentageDrop)
		fmt.Println("Resource Usages:")
		for _, usage := range caseReport.ResourceUsages {
			fmt.Printf("  Service: %s\n", usage.Name)
			fmt.Printf("    CPU: %.2f\n", usage.CPU)
			fmt.Printf("    Memory: %.2f MB\n", usage.Memory)
		}
	}
}

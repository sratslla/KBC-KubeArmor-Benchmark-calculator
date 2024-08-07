/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// exec3Cmd represents the exec3 command
var exec3Cmd = &cobra.Command{
	Use:   "exec3",
	Short: "Here we will test the benchmark on different visiblities",
	Long:  `Here we will test the benchmark on different visiblities i.e none, process, process+file, process+network, process+network+file`,
	Run: func(cmd *cobra.Command, args []string) {

		// REPO_URL := "https://raw.githubusercontent.com/sratslla/KBC-KubeArmor-Benchmark-calculator/main"
		// manifestmanifestPath := "manifests/policy-file.yaml"

		fmt.Println("exec3 called")
		// time.Sleep(3 * time.Minute)
		// calculateBenchMark()
		// fmt.Println("=========================================")
		// changeVisiblity("process")
		// time.Sleep(3 * time.Minute)
		// calculateBenchMark()
		// fmt.Println("=========================================")
		// changeVisiblity("process, file")
		// time.Sleep(3 * time.Minute)
		// calculateBenchMark()
		// fmt.Println("=========================================")
		// changeVisiblity("process, network")
		// time.Sleep(3 * time.Minute)
		// calculateBenchMark()
		// fmt.Println("=========================================")
		// changeVisiblity("process, network, file")
		// time.Sleep(3 * time.Minute)
		// calculateBenchMark()
		// fmt.Println("=========================================")
		// changeVisiblity("none")

		// Calculating Benchmark on different Policies.
		// changeVisiblity("none")
		// fmt.Printf("no checking for different policy")
		// err := applyManifestFromGitHub(REPO_URL, manifestmanifestPath)
		// if err != nil {
		// 	fmt.Println("Error applying manifest:", err)
		// 	os.Exit(1)
		// }
		// calculateBenchMark()
	},
}

func init() {
	rootCmd.AddCommand(exec3Cmd)
}

// func changeVisiblity(visiblityMode string) error {
// 	cmd := exec.Command("kubectl", "annotate", "ns", "default", fmt.Sprintf("kubearmor-visibility=%s", visiblityMode), "--overwrite")
// 	var output bytes.Buffer
// 	cmd.Stderr = &output
// 	cmd.Stdout = &output
// 	err := cmd.Run()
// 	if err != nil {
// 		return fmt.Errorf("error annotating namespace with %s: %v\n%s", visiblityMode, err, output.String())
// 	}
// 	fmt.Printf("Namespace annotated with visibility type %s successfully.\n", visiblityMode)
// 	return nil
// }

// func calculateBenchMark() {
// 	externalIP, err := getExternalIP()
// 	if err != nil {
// 		fmt.Println("Error getting external IP:", err)
// 		return
// 	}
// 	prometheusURL := fmt.Sprintf("http://%s:30000", externalIP)
// 	promClient, err := NewPrometheusClient(prometheusURL)
// 	if err != nil {
// 		fmt.Println("Error creating Prometheus client:", err)
// 		return
// 	}
// 	CPUQueries := map[string]string{
// 		"Frontend":              `sum(rate(container_cpu_usage_seconds_total{pod=~"frontend-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Adservice":             `sum(rate(container_cpu_usage_seconds_total{pod=~"adservice-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Cartservice":           `sum(rate(container_cpu_usage_seconds_total{pod=~"cartservice-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Checkoutservice":       `sum(rate(container_cpu_usage_seconds_total{pod=~"checkoutservice-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Currencyservice":       `sum(rate(container_cpu_usage_seconds_total{pod=~"currencyservice-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Emailservice":          `sum(rate(container_cpu_usage_seconds_total{pod=~"emailservice-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Loadgenerator":         `sum(rate(container_cpu_usage_seconds_total{pod=~"loadgenerator-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Paymentservice":        `sum(rate(container_cpu_usage_seconds_total{pod=~"paymentservice-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Productcatalogservice": `sum(rate(container_cpu_usage_seconds_total{pod=~"productcatalogservice-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Recommendationservice": `sum(rate(container_cpu_usage_seconds_total{pod=~"recommendationservice-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Redis":                 `sum(rate(container_cpu_usage_seconds_total{pod=~"redis-.*", container="", namespace="default"}[3m])) * 1000`,
// 		"Shippingservice":       `sum(rate(container_cpu_usage_seconds_total{pod=~"shippingservice-.*", container="", namespace="default"}[3m])) * 1000`,
// 	}
// 	MemoryQueries := map[string]string{
// 		"Frontend":              `sum(container_memory_usage_bytes{pod=~"frontend-.*", namespace="default"}) / 1024 / 1024`,
// 		"Adservice":             `sum(container_memory_usage_bytes{pod=~"adservice-.*", namespace="default"}) / 1024 / 1024`,
// 		"Cartservice":           `sum(container_memory_usage_bytes{pod=~"cartservice-.*", namespace="default"}) / 1024 / 1024`,
// 		"Checkoutservice":       `sum(container_memory_usage_bytes{pod=~"checkoutservice-.*", namespace="default"}) / 1024 / 1024`,
// 		"Currencyservice":       `sum(container_memory_usage_bytes{pod=~"currencyservice-.*", namespace="default"}) / 1024 / 1024`,
// 		"Emailservice":          `sum(container_memory_usage_bytes{pod=~"emailservice-.*", namespace="default"}) / 1024 / 1024`,
// 		"Loadgenerator":         `sum(container_memory_usage_bytes{pod=~"loadgenerator-.*", namespace="default"}) / 1024 / 1024`,
// 		"Paymentservice":        `sum(container_memory_usage_bytes{pod=~"paymentservice-.*", namespace="default"}) / 1024 / 1024`,
// 		"Productcatalogservice": `sum(container_memory_usage_bytes{pod=~"productcatalogservice-.*", namespace="default"}) / 1024 / 1024`,
// 		"Recommendationservice": `sum(container_memory_usage_bytes{pod=~"recommendationservice-.*", namespace="default"}) / 1024 / 1024`,
// 		"Redis":                 `sum(container_memory_usage_bytes{pod=~"redis-.*", namespace="default"}) / 1024 / 1024`,
// 		"Shippingservice":       `sum(container_memory_usage_bytes{pod=~"shippingservice-.*", namespace="default"}) / 1024 / 1024`,
// 	}

// 	fmt.Printf("CPU Usage \n")
// 	for serviceName, query := range CPUQueries {
// 		result, err := QueryPrometheus(promClient, query)
// 		if err != nil {
// 			fmt.Printf("Error querying Prometheus for CPU metrics (%s): %v\n", serviceName, err)
// 			return
// 		}
// 		fmt.Printf("%s CPU usage: %v\n", serviceName, result)
// 	}
// 	fmt.Printf("Memory \n")
// 	for serviceName, query := range MemoryQueries {
// 		result, err := QueryPrometheus(promClient, query)
// 		if err != nil {
// 			fmt.Printf("Error querying Prometheus for Memory metrics (%s): %v\n", serviceName, err)
// 			return
// 		}
// 		fmt.Printf("%s Memory usage: %v\n", serviceName, result)
// 	}

// 	locustThroughput := `avg_over_time(locust_requests_current_rps{job="locust", name="Aggregated"}[3m])`
// 	locustThroughputResult, err := QueryPrometheus(promClient, locustThroughput)
// 	if err != nil {
// 		fmt.Println("Error querying Prometheus for Locust metrics:", err)
// 		return
// 	}
// 	re := regexp.MustCompile(`=>\s+([0-9.]+)\s+@`)
// 	match := re.FindStringSubmatch(locustThroughputResult.String())
// 	if len(match) > 1 {
// 		value := match[1]
// 		fmt.Println("Locust Throughput Average for last 10mins:", value)
// 	} else {
// 		fmt.Println("Error: Could not parse the result")
// 	}
// }

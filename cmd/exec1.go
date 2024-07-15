/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/spf13/cobra"
)

// exec1Cmd represents the exec1 command
var exec1Cmd = &cobra.Command{
	Use:   "exec1",
	Short: "A brief description of your command",
	Long:  `This will check when the users become 400 and after that this will bring us the throughput, cpu and memory.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("exec1 called")

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

			if locustUsers >= 400 {
				fmt.Println("locust users reached 400. data will be fetched now to calculate avg benchmark.")
				break
			}

			fmt.Printf("\rWaiting for locust_users to reach 400\n")
		}

		time.Sleep(11 * time.Minute)

		// query of cpu of each microservice pod
		CPUQueries := map[string]string{
			"Frontend":              `sum(rate(container_cpu_usage_seconds_total{pod=~"frontend-.*", container="", namespace="default"}[10m]))`,
			"Adservice":             `sum(rate(container_cpu_usage_seconds_total{pod=~"adservice-.*", container="", namespace="default"}[10m]))`,
			"Cartservice":           `sum(rate(container_cpu_usage_seconds_total{pod=~"cartservice-.*", container="", namespace="default"}[10m]))`,
			"Checkoutservice":       `sum(rate(container_cpu_usage_seconds_total{pod=~"checkoutservice-.*", container="", namespace="default"}[10m]))`,
			"Currencyservice":       `sum(rate(container_cpu_usage_seconds_total{pod=~"currencyservice-.*", container="", namespace="default"}[10m]))`,
			"Emailservice":          `sum(rate(container_cpu_usage_seconds_total{pod=~"emailservice-.*", container="", namespace="default"}[10m]))`,
			"Loadgenerator":         `sum(rate(container_cpu_usage_seconds_total{pod=~"loadgenerator-.*", container="", namespace="default"}[10m]))`,
			"Paymentservice":        `sum(rate(container_cpu_usage_seconds_total{pod=~"paymentservice-.*", container="", namespace="default"}[10m]))`,
			"Productcatalogservice": `sum(rate(container_cpu_usage_seconds_total{pod=~"productcatalogservice-.*", container="", namespace="default"}[10m]))`,
			"Recommendationservice": `sum(rate(container_cpu_usage_seconds_total{pod=~"recommendationservice-.*", container="", namespace="default"}[10m]))`,
			"Redis":                 `sum(rate(container_cpu_usage_seconds_total{pod=~"redis-.*", container="", namespace="default"}[10m]))`,
			"Shippingservice":       `sum(rate(container_cpu_usage_seconds_total{pod=~"shippingservice-.*", container="", namespace="default"}[10m]))`,
		}
		MemoryQueries := map[string]string{
			"Frontend":              `sum(container_memory_usage_bytes{pod=~"frontend-.*", namespace="default"})`,
			"Adservice":             `sum(container_memory_usage_bytes{pod=~"adservice-.*", namespace="default"})`,
			"Cartservice":           `sum(container_memory_usage_bytes{pod=~"cartservice-.*", namespace="default"})`,
			"Checkoutservice":       `sum(container_memory_usage_bytes{pod=~"checkoutservice-.*", namespace="default"})`,
			"Currencyservice":       `sum(container_memory_usage_bytes{pod=~"currencyservice-.*", namespace="default"})`,
			"Emailservice":          `sum(container_memory_usage_bytes{pod=~"emailservice-.*", namespace="default"})`,
			"Loadgenerator":         `sum(container_memory_usage_bytes{pod=~"loadgenerator-.*", namespace="default"})`,
			"Paymentservice":        `sum(container_memory_usage_bytes{pod=~"paymentservice-.*", namespace="default"})`,
			"Productcatalogservice": `sum(container_memory_usage_bytes{pod=~"productcatalogservice-.*", namespace="default"})`,
			"Recommendationservice": `sum(container_memory_usage_bytes{pod=~"recommendationservice-.*", namespace="default"})`,
			"Redis":                 `sum(container_memory_usage_bytes{pod=~"redis-.*", namespace="default"})`,
			"Shippingservice":       `sum(container_memory_usage_bytes{pod=~"shippingservice-.*", namespace="default"})`,
		}

		for serviceName, query := range CPUQueries {
			result, err := QueryPrometheus(promClient, query)
			if err != nil {
				fmt.Printf("Error querying Prometheus for CPU metrics (%s): %v\n", serviceName, err)
				return
			}
			fmt.Printf("%s CPU usage: %v\n", serviceName, result)
		}
		for serviceName, query := range MemoryQueries {
			result, err := QueryPrometheus(promClient, query)
			if err != nil {
				fmt.Printf("Error querying Prometheus for Memory metrics (%s): %v\n", serviceName, err)
				return
			}
			fmt.Printf("%s Memory usage: %v\n", serviceName, result)
		}
		// cpuResults := make([]float64, 0, 10)

		// cpuTicker := time.NewTicker(1 * time.Minute)
		// defer cpuTicker.Stop()

		// for i := 0; i < 10; i++ {
		// 	<-cpuTicker.C

		// 	cpuResult, err := QueryPrometheus(promClient, cpuQuery)
		// 	if err != nil {
		// 		fmt.Println("Error querying Prometheus for CPU metrics:", err)
		// 		return
		// 	}

		// 	if cpuResult.Type() == model.ValVector {
		// 		vector := cpuResult.(model.Vector)
		// 		for _, sample := range vector {
		// 			cpuResults = append(cpuResults, float64(sample.Value))
		// 		}
		// 	}
		// }
		// var sum float64
		// for _, value := range cpuResults {
		// 	sum += value
		// }
		// avgCPUUsage := sum / float64(len(cpuResults))

		// fmt.Printf("\nAverage CPU usage over the last 10 minutes: %f\n", avgCPUUsage)

		locustThroughput := `avg_over_time(locust_requests_current_rps{job="locust", name="Aggregated"}[10m])`
		locustThroughputResult, err := QueryPrometheus(promClient, locustThroughput)
		if err != nil {
			fmt.Println("Error querying Prometheus for Locust metrics:", err)
			return
		}
		fmt.Println("Locust Throughput Average for last 10mins:", locustThroughputResult)
	},
}

func init() {
	rootCmd.AddCommand(exec1Cmd)
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

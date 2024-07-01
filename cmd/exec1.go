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
	Long:  `This will check when the users become 1000 and after that this will bring us the throughput, cpu and memory.`,
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

			// Parse locust users count from the query result
			locustUsers := 0
			if locustResult.Type() == model.ValVector {
				vector := locustResult.(model.Vector)
				for _, sample := range vector {
					locustUsers = int(sample.Value)
					fmt.Printf("locustUsers %v", locustUsers)
				}
			}

			if locustUsers >= 100 {
				fmt.Println("locust users reached. data will be fetched now to calculate avg benchmark.")
				break
			}

			// Display loading animation
			fmt.Printf("\rWaiting for locust_users to reach 1000 ")
		}

		time.Sleep(2 * time.Minute)

		// Now we will query the CPU usage every minute for the next 10 minutes
		cpuQuery := `sum(rate(container_cpu_usage_seconds_total{pod=~"frontend-.*", container = "", namespace="default"}[1m]))`
		cpuResults := make([]float64, 0, 10)

		cpuTicker := time.NewTicker(1 * time.Minute)
		defer cpuTicker.Stop()

		for i := 0; i < 10; i++ {
			<-cpuTicker.C

			cpuResult, err := QueryPrometheus(promClient, cpuQuery)
			if err != nil {
				fmt.Println("Error querying Prometheus for CPU metrics:", err)
				return
			}

			// Parse CPU usage from the query result
			if cpuResult.Type() == model.ValVector {
				vector := cpuResult.(model.Vector)
				for _, sample := range vector {
					cpuResults = append(cpuResults, float64(sample.Value))
				}
			}
		}
		fmt.Println(cpuResults)
		// Calculate the average CPU usage over the 10 minutes
		var sum float64
		for _, value := range cpuResults {
			sum += value
		}
		avgCPUUsage := sum / float64(len(cpuResults))

		fmt.Printf("\nAverage CPU usage over the last 10 minutes: %f\n", avgCPUUsage)

		locustResult, err := QueryPrometheus(promClient, locustQuery)
		if err != nil {
			fmt.Println("Error querying Prometheus for Locust metrics:", err)
			return
		}
		fmt.Println("Locust Query result:", locustResult)
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

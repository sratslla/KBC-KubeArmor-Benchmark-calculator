/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

// exec2Cmd represents the exec2 command
var exec2Cmd = &cobra.Command{
	Use:   "exec2",
	Short: "A brief description of your command",
	Long:  `This will keep the replicaset constant and install kubearmor.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("exec2 called")

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
	},
}

func init() {
	rootCmd.AddCommand(exec2Cmd)
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

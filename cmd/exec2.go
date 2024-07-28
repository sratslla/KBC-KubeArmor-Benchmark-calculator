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
	Long:  `This will keep the replicaset constant, remove the hpa and install kubearmor.`,
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

		err := installKubearmor()
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

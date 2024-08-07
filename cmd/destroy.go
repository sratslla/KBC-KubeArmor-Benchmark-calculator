/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Delete the Kubernetes Resources",
	Long:  "Delete the Kubernetes Resources",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("destroy called")
		REPO_URL := "https://raw.githubusercontent.com/sratslla/KBC-KubeArmor-Benchmark-calculator/main"
		YAML_FILE_PATH := "manifests/kubernetes-manifests.yaml"
		YAML_FILE_PATH2 := "manifests/loadgenerator_ui.yaml"
		YAML_FILE_PATH3 := "manifests/kube-static-metrics.yaml"
		YAML_FILE_PATH4 := "manifests/prometheusComponent.yaml"
		err := destroyManifestFromGitHub(REPO_URL, YAML_FILE_PATH)
		if err != nil {
			fmt.Println("Error deleting manifest:", err)
			os.Exit(1)
		}
		err2 := destroyManifestFromGitHub(REPO_URL, YAML_FILE_PATH2)
		if err2 != nil {
			fmt.Println("Error deleting manifest:", err)
			os.Exit(1)
		}
		err3 := destroyManifestFromGitHub(REPO_URL, YAML_FILE_PATH3)
		if err3 != nil {
			fmt.Println("Error deleting manifest:", err)
			os.Exit(1)
		}
		err4 := destroyManifestFromGitHub(REPO_URL, YAML_FILE_PATH4)
		if err4 != nil {
			fmt.Println("Error deleting manifest:", err)
			os.Exit(1)
		}
		cmdhpa := exec.Command("kubectl", "delete", "hpa", "--all")
		output, err := cmdhpa.CombinedOutput()
		if err != nil {
			fmt.Errorf("error applying manifest: %v\n%s", err, output)
		}
		fmt.Println("Resources Deleted Successfully.")
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
func destroyManifestFromGitHub(repoURL, yamlFilePath string) error {
	cmd := exec.Command("kubectl", "delete", "-f", fmt.Sprintf("%s/%s", repoURL, yamlFilePath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error applying manifest: %v\n%s", err, output)
	}
	fmt.Println("Resources Deleted Successfully.")
	return nil
}

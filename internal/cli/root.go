// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package cli

import (
	"github.com/spf13/cobra"
)

var (
	kubeconfig   string
	manifestsDir string
	configsDir   string
)

var rootCmd = &cobra.Command{
	Use:   "kbc",
	Short: "KubeArmor Benchmark Calculator",
	Long:  `kbc automates KubeArmor performance benchmarking against the Online Boutique workload using Locust and Prometheus.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig (default: in-cluster or KUBECONFIG/~/.kube/config)")
	rootCmd.PersistentFlags().StringVar(&manifestsDir, "manifests-dir", "deploy/manifests", "Directory containing Kubernetes manifests")
	rootCmd.PersistentFlags().StringVar(&configsDir, "configs-dir", "configs", "Directory containing profiles and scenarios")

	rootCmd.AddCommand(newSetupCmd())
	rootCmd.AddCommand(newEnsureCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newDestroyCmd())
}

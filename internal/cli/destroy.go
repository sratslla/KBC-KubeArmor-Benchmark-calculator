// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/cluster"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/ensure"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/workload"
)

func newDestroyCmd() *cobra.Command {
	var (
		namespace   string
		uninstallKA bool
	)
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Tear down workload HPAs/manifests and optionally uninstall KubeArmor",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			clients, err := cluster.New(kubeconfig)
			if err != nil {
				return err
			}
			wm := &workload.Manager{Clients: clients, ManifestsDir: manifestsDir}
			_ = wm.DeleteAllPolicies(ctx, namespace)
			_ = wm.DeleteAllHPAs(ctx, namespace)
			if err := wm.DeleteCore(ctx); err != nil {
				return err
			}
			_ = clients.Clientset.AppsV1().Deployments(namespace).Delete(ctx, "loadgenerator", metav1.DeleteOptions{})
			_ = clients.Clientset.CoreV1().Services(namespace).Delete(ctx, "loadgenerator-service", metav1.DeleteOptions{})

			if uninstallKA {
				em := &ensure.Manager{Clients: clients}
				if err := em.Uninstall(ctx); err != nil {
					fmt.Printf("warning uninstalling kubearmor: %v\n", err)
				}
			}
			fmt.Println("destroy complete")
			return nil
		},
	}
	cmd.Flags().StringVar(&namespace, "namespace", "default", "Workload namespace")
	cmd.Flags().BoolVar(&uninstallKA, "uninstall-ka", false, "Also helm uninstall kubearmor-operator")
	return cmd
}

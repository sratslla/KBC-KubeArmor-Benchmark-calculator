// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/cluster"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/workload"
)

func newSetupCmd() *cobra.Command {
	var (
		cpuPercent int32
		users      int32
		spawnRate  int32
		debugUI    bool
		namespace  string
	)
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Apply boutique workload, observability stack, HPAs, and headless Locust",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			clients, err := cluster.New(kubeconfig)
			if err != nil {
				return err
			}
			wm := &workload.Manager{Clients: clients, ManifestsDir: manifestsDir}
			if err := wm.ApplyCore(ctx); err != nil {
				return err
			}
			if err := wm.CreateHPAs(ctx, namespace, cpuPercent); err != nil {
				return err
			}
			if err := wm.EnsureLocust(ctx, workload.LocustOptions{
				Users: users, SpawnRate: spawnRate, Namespace: namespace, DebugUI: debugUI,
			}); err != nil {
				return err
			}
			names := make([]string, 0, len(workload.GuideDeployments))
			for _, d := range workload.GuideDeployments {
				names = append(names, d.Name)
			}
			fmt.Println("waiting for deployments to become available...")
			if err := wm.WaitDeploymentsReady(ctx, namespace, names, 15*time.Minute); err != nil {
				return fmt.Errorf("wait deployments: %w", err)
			}
			fmt.Println("setup complete")
			return nil
		},
	}
	cmd.Flags().Int32Var(&cpuPercent, "cpu-percent", 50, "HPA CPU target percent")
	cmd.Flags().Int32Var(&users, "users", 2000, "Locust virtual users")
	cmd.Flags().Int32Var(&spawnRate, "spawn-rate", 1, "Locust spawn rate")
	cmd.Flags().BoolVar(&debugUI, "debug-ui", false, "Expose Locust UI NodePort 30001")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "Workload namespace")
	return cmd
}

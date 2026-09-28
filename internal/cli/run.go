// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/cluster"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/ensure"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/metrics"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/report"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/scenario"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/workload"
)

func newRunCmd() *cobra.Command {
	var (
		profileName    string
		fromScenario   string
		prometheusURL  string
		checkpointPath string
		outDir         string
		skipSetup      bool
		skipFreeze     bool
		namespace      string
	)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the benchmark profile (setup check, baseline, ensure, matrix, report)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			profilePath := filepath.Join(configsDir, "profiles", profileName+".yaml")
			prof, err := scenario.LoadProfile(profilePath)
			if err != nil {
				return fmt.Errorf("load profile: %w", err)
			}
			if namespace != "" {
				prof.Namespace = namespace
			}

			clients, err := cluster.New(kubeconfig)
			if err != nil {
				return err
			}
			wm := &workload.Manager{Clients: clients, ManifestsDir: manifestsDir}
			em := &ensure.Manager{Clients: clients}

			if !skipSetup {
				fmt.Println("applying core manifests + HPA + locust...")
				if err := wm.ApplyCore(ctx); err != nil {
					return err
				}
				if err := wm.CreateHPAs(ctx, prof.Namespace, prof.CPUPercent); err != nil {
					return err
				}
				if err := wm.EnsureLocust(ctx, workload.LocustOptions{
					Users: prof.Users, SpawnRate: prof.SpawnRate, Namespace: prof.Namespace,
					RunTime: "180m",
				}); err != nil {
					return err
				}
			}

			promURL, err := wm.ResolvePrometheusURL(ctx, prometheusURL)
			if err != nil {
				return err
			}
			fmt.Printf("prometheus: %s\n", promURL)
			mc, err := metrics.NewClient(promURL)
			if err != nil {
				return err
			}

			fmt.Println("waiting for locust users...")
			if err := scenario.WaitLocustUsers(ctx, mc, prof.Users, 90*time.Minute); err != nil {
				return fmt.Errorf("wait locust users: %w", err)
			}

			if !skipFreeze {
				fmt.Println("freezing replicas after HPA stability...")
				if _, err := wm.FreezeReplicas(ctx, prof.Namespace); err != nil {
					return err
				}
			}

			if checkpointPath == "" {
				checkpointPath = filepath.Join(outDir, "checkpoint.json")
			}

			runner := &scenario.Runner{
				Profile:        prof,
				Workload:       wm,
				Ensure:         em,
				Metrics:        mc,
				Queries:        metrics.DefaultQueries(),
				CheckpointPath: checkpointPath,
				FromScenario:   fromScenario,
			}

			results, err := runner.Run(ctx)
			if err != nil {
				_ = report.WriteJSON(filepath.Join(outDir, "final_report.json"), results)
				return err
			}

			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return err
			}
			mdPath := filepath.Join(outDir, "final_report.md")
			jsonPath := filepath.Join(outDir, "final_report.json")
			if err := report.WriteMarkdown(mdPath, results); err != nil {
				return err
			}
			if err := report.WriteJSON(jsonPath, results); err != nil {
				return err
			}
			fmt.Printf("wrote %s and %s\n", mdPath, jsonPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "release", "Profile name under configs/profiles/")
	cmd.Flags().StringVar(&fromScenario, "from-scenario", "", "Resume from scenario id")
	cmd.Flags().StringVar(&prometheusURL, "prometheus-url", "", "Prometheus base URL (auto-detected if empty)")
	cmd.Flags().StringVar(&checkpointPath, "checkpoint", "", "Checkpoint JSON path")
	cmd.Flags().StringVar(&outDir, "out", ".", "Output directory for reports")
	cmd.Flags().BoolVar(&skipSetup, "skip-setup", false, "Skip manifest/HPA/Locust apply")
	cmd.Flags().BoolVar(&skipFreeze, "skip-freeze", false, "Skip HPA freeze step")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Override profile namespace")
	return cmd
}

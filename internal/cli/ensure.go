// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/cluster"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/ensure"
)

func newEnsureCmd() *cobra.Command {
	var (
		lsmOrder   string
		visibility string
	)
	cmd := &cobra.Command{
		Use:   "ensure",
		Short: "Detect KubeArmor and install via Helm Operator + KubeArmorConfig if missing",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			clients, err := cluster.New(kubeconfig)
			if err != nil {
				return err
			}
			em := &ensure.Manager{Clients: clients}
			st, err := em.Detect(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("status: operator=%v config=%v daemon=%v enforcer=%q\n",
				st.OperatorReady, st.ConfigExists, st.DaemonReady, st.Enforcer)
			if err := em.EnsureInstalled(ctx, lsmOrder, visibility); err != nil {
				return err
			}
			fmt.Println("ensure complete")
			return nil
		},
	}
	cmd.Flags().StringVar(&lsmOrder, "lsm-order", "bpf,apparmor,selinux", "LSM preference order passed to KubeArmor")
	cmd.Flags().StringVar(&visibility, "default-visibility", "process,network", "KubeArmorConfig defaultVisibility")
	return cmd
}

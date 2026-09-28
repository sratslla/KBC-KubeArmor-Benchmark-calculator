// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package cluster

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Clients holds shared Kubernetes API clients.
type Clients struct {
	Config    *rest.Config
	Clientset kubernetes.Interface
	Dynamic   dynamic.Interface
}

// New builds clients from kubeconfig path (empty = in-cluster / default kubeconfig).
func New(kubeconfig string) (*Clients, error) {
	cfg, err := restConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}
	return &Clients{Config: cfg, Clientset: cs, Dynamic: dyn}, nil
}

func restConfig(kubeconfig string) (*rest.Config, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig %q: %w", kubeconfig, err)
	}
	return cfg, nil
}

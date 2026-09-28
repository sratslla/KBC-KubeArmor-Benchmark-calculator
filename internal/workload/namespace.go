// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package workload

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/waitutil"
)

// AnnotateNamespace sets an annotation on a namespace (merge patch).
func (m *Manager) AnnotateNamespace(ctx context.Context, name, key, value string) error {
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{key: value},
		},
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = m.Clients.Clientset.CoreV1().Namespaces().Patch(ctx, name, types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("annotate ns %s: %w", name, err)
	}
	fmt.Printf("namespace %s: %s=%s\n", name, key, value)
	return nil
}

// SetVisibility sets kubearmor-visibility on the namespace.
func (m *Manager) SetVisibility(ctx context.Context, namespace, visibility string) error {
	return m.AnnotateNamespace(ctx, namespace, "kubearmor-visibility", visibility)
}

// WaitDeploymentsReady waits until listed deployments have Available replicas.
func (m *Manager) WaitDeploymentsReady(ctx context.Context, namespace string, names []string, timeout time.Duration) error {
	return waitutil.Until(ctx, 5*time.Second, timeout, func(ctx context.Context) (bool, error) {
		for _, name := range names {
			d, err := m.Clients.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if d.Status.AvailableReplicas < 1 {
				return false, nil
			}
		}
		return true, nil
	})
}

// ExternalNodeIP returns the first ExternalIP among nodes.
func (m *Manager) ExternalNodeIP(ctx context.Context) (string, error) {
	nodes, err := m.Clients.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, n := range nodes.Items {
		for _, a := range n.Status.Addresses {
			if a.Type == corev1.NodeExternalIP && a.Address != "" {
				return a.Address, nil
			}
		}
	}
	// fallback InternalIP for kind/minikube
	for _, n := range nodes.Items {
		for _, a := range n.Status.Addresses {
			if a.Type == corev1.NodeInternalIP && a.Address != "" {
				return a.Address, nil
			}
		}
	}
	return "", fmt.Errorf("no node IP found")
}

// ResolvePrometheusURL picks in-cluster service URL or NodePort via external IP.
func (m *Manager) ResolvePrometheusURL(ctx context.Context, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	// Prefer in-cluster DNS when running in-cluster (no kubeconfig host typically is kubernetes).
	_, err := m.Clients.Clientset.CoreV1().Services("monitoring").Get(ctx, "prometheus-service", metav1.GetOptions{})
	if err == nil {
		// Try cluster DNS first; out-of-cluster callers should use NodePort.
		ip, ipErr := m.ExternalNodeIP(ctx)
		if ipErr == nil {
			return fmt.Sprintf("http://%s:30000", ip), nil
		}
		return "http://prometheus-service.monitoring.svc:8080", nil
	}
	ip, err := m.ExternalNodeIP(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:30000", ip), nil
}

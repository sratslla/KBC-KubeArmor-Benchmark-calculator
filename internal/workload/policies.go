// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package workload

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/cluster"
)

// PolicyFiles maps short names to manifest filenames.
var PolicyFiles = map[string]string{
	"process": "policy-process.yaml",
	"file":    "policy-file.yaml",
	"network": "policy-network.yaml",
}

// ApplyPolicies applies named policy manifests (process, file, network).
func (m *Manager) ApplyPolicies(ctx context.Context, names []string) error {
	for _, n := range names {
		file, ok := PolicyFiles[n]
		if !ok {
			return fmt.Errorf("unknown policy %q", n)
		}
		if err := m.ApplyFile(ctx, file); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAllPolicies removes all KubeArmorPolicies in default namespace.
func (m *Manager) DeleteAllPolicies(ctx context.Context, namespace string) error {
	list, err := m.Clients.Dynamic.Resource(cluster.GVRKubeArmorPolicy).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	for _, item := range list.Items {
		err := m.Clients.Dynamic.Resource(cluster.GVRKubeArmorPolicy).Namespace(namespace).Delete(ctx, item.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		fmt.Printf("deleted KubeArmorPolicy %s\n", item.GetName())
	}
	return nil
}

// ApplyPolicyYAML applies raw YAML bytes as policies (helper for tests).
func (m *Manager) ApplyPolicyYAML(ctx context.Context, path string) error {
	if _, err := os.Stat(filepath.Clean(path)); err != nil {
		return err
	}
	return m.ApplyFile(ctx, filepath.Base(path))
}

// GetUnstructured is a small helper for dynamic gets.
func GetUnstructured(obj *unstructured.Unstructured) *unstructured.Unstructured { return obj }

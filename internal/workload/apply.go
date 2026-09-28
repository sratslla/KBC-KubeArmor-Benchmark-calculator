// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package workload

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/cluster"
)

var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

// Manager applies and manages boutique/benchmark workloads.
type Manager struct {
	Clients      *cluster.Clients
	ManifestsDir string
}

// CoreManifests are applied during setup.
var CoreManifests = []string{
	"kubernetes-manifests.yaml",
	"kube-static-metrics.yaml",
	"prometheusComponent.yaml",
}

// ApplyFile applies all objects in a multi-doc YAML file.
func (m *Manager) ApplyFile(ctx context.Context, name string) error {
	path := filepath.Join(m.ManifestsDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return m.applyYAML(ctx, data)
}

// ApplyCore applies boutique + observability manifests.
func (m *Manager) ApplyCore(ctx context.Context) error {
	for _, f := range CoreManifests {
		fmt.Printf("applying %s\n", f)
		if err := m.ApplyFile(ctx, f); err != nil {
			return err
		}
	}
	return nil
}

// DeleteCore deletes core manifests.
func (m *Manager) DeleteCore(ctx context.Context) error {
	for i := len(CoreManifests) - 1; i >= 0; i-- {
		f := CoreManifests[i]
		fmt.Printf("deleting %s\n", f)
		if err := m.DeleteFile(ctx, f); err != nil {
			fmt.Printf("warning deleting %s: %v\n", f, err)
		}
	}
	return nil
}

// DeleteFile deletes objects from a multi-doc YAML.
func (m *Manager) DeleteFile(ctx context.Context, name string) error {
	path := filepath.Join(m.ManifestsDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return m.deleteYAML(ctx, data)
}

func (m *Manager) applyYAML(ctx context.Context, data []byte) error {
	mapper, err := m.restMapper()
	if err != nil {
		return err
	}
	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		doc, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}
		obj := &unstructured.Unstructured{}
		_, gvk, err := decUnstructured.Decode(doc, nil, obj)
		if err != nil {
			return fmt.Errorf("decode manifest: %w", err)
		}
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("rest mapping for %s: %w", gvk.String(), err)
		}
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == "namespace" {
			ns := obj.GetNamespace()
			if ns == "" {
				ns = "default"
				obj.SetNamespace(ns)
			}
			dr = m.Clients.Dynamic.Resource(mapping.Resource).Namespace(ns)
		} else {
			dr = m.Clients.Dynamic.Resource(mapping.Resource)
		}
		_, err = dr.Get(ctx, obj.GetName(), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = dr.Create(ctx, obj, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("create %s/%s: %w", gvk.Kind, obj.GetName(), err)
			}
			fmt.Printf("  created %s/%s\n", gvk.Kind, obj.GetName())
			continue
		}
		if err != nil {
			return err
		}
		existing, err := dr.Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		obj.SetResourceVersion(existing.GetResourceVersion())
		_, err = dr.Update(ctx, obj, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("update %s/%s: %w", gvk.Kind, obj.GetName(), err)
		}
		fmt.Printf("  applied %s/%s\n", gvk.Kind, obj.GetName())
	}
	return nil
}

func (m *Manager) deleteYAML(ctx context.Context, data []byte) error {
	mapper, err := m.restMapper()
	if err != nil {
		return err
	}
	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		doc, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}
		obj := &unstructured.Unstructured{}
		_, gvk, err := decUnstructured.Decode(doc, nil, obj)
		if err != nil {
			continue
		}
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			continue
		}
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == "namespace" {
			ns := obj.GetNamespace()
			if ns == "" {
				ns = "default"
			}
			dr = m.Clients.Dynamic.Resource(mapping.Resource).Namespace(ns)
		} else {
			dr = m.Clients.Dynamic.Resource(mapping.Resource)
		}
		err = dr.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (m *Manager) restMapper() (*restmapper.DeferredDiscoveryRESTMapper, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(m.Clients.Config)
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc)), nil
}

// EnsureNamespace creates ns if missing.
func (m *Manager) EnsureNamespace(ctx context.Context, name string) error {
	_, err := m.Clients.Clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	_, err = m.Clients.Clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}, metav1.CreateOptions{})
	return err
}

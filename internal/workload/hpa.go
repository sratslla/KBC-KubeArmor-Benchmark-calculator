// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package workload

import (
	"context"
	"fmt"
	"time"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/waitutil"
)

// DeploymentScale describes HPA bounds for a deployment.
type DeploymentScale struct {
	Name        string
	MinReplicas int32
	MaxReplicas int32
}

// GuideDeployments matches the Benchmarking Guide HPA settings.
var GuideDeployments = []DeploymentScale{
	{Name: "cartservice", MinReplicas: 2, MaxReplicas: 400},
	{Name: "currencyservice", MinReplicas: 2, MaxReplicas: 400},
	{Name: "emailservice", MinReplicas: 2, MaxReplicas: 400},
	{Name: "checkoutservice", MinReplicas: 2, MaxReplicas: 400},
	{Name: "frontend", MinReplicas: 5, MaxReplicas: 400},
	{Name: "paymentservice", MinReplicas: 2, MaxReplicas: 400},
	{Name: "productcatalogservice", MinReplicas: 2, MaxReplicas: 400},
	{Name: "recommendationservice", MinReplicas: 2, MaxReplicas: 400},
	{Name: "redis-cart", MinReplicas: 1, MaxReplicas: 400},
	{Name: "shippingservice", MinReplicas: 2, MaxReplicas: 400},
	{Name: "adservice", MinReplicas: 1, MaxReplicas: 400},
}

// CreateHPAs creates HPAs at the given CPU target percentage.
func (m *Manager) CreateHPAs(ctx context.Context, namespace string, cpuPercent int32) error {
	for _, d := range GuideDeployments {
		hpa := &autoscalingv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: d.Name, Namespace: namespace},
			Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       d.Name,
				},
				MinReplicas:                    &d.MinReplicas,
				MaxReplicas:                    d.MaxReplicas,
				TargetCPUUtilizationPercentage: &cpuPercent,
			},
		}
		_, err := m.Clients.Clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Create(ctx, hpa, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			existing, getErr := m.Clients.Clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(ctx, d.Name, metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}
			hpa.ResourceVersion = existing.ResourceVersion
			_, err = m.Clients.Clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Update(ctx, hpa, metav1.UpdateOptions{})
		}
		if err != nil {
			return fmt.Errorf("hpa %s: %w", d.Name, err)
		}
		fmt.Printf("HPA ready for %s (cpu=%d%% min=%d max=%d)\n", d.Name, cpuPercent, d.MinReplicas, d.MaxReplicas)
	}
	return nil
}

// FreezeReplicas waits until HPA currentReplicas are stable, deletes HPAs, and scales deployments to frozen counts.
func (m *Manager) FreezeReplicas(ctx context.Context, namespace string) (map[string]int32, error) {
	replicas := make(map[string]int32)
	err := waitutil.StableUntil(ctx, 15*time.Second, 30*time.Minute, 3, func(ctx context.Context) (bool, error) {
		changed := false
		for _, d := range GuideDeployments {
			hpa, err := m.Clients.Clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(ctx, d.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			cur := hpa.Status.CurrentReplicas
			if prev, ok := replicas[d.Name]; !ok || prev != cur {
				changed = true
			}
			replicas[d.Name] = cur
		}
		if changed {
			fmt.Printf("waiting for HPA stability: %v\n", replicas)
		}
		return !changed, nil
	})
	if err != nil {
		return nil, fmt.Errorf("wait hpa stable: %w", err)
	}

	for name, count := range replicas {
		fmt.Printf("freezing %s at %d replicas\n", name, count)
		err := m.Clients.Clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("delete hpa %s: %w", name, err)
		}
		scale, err := m.Clients.Clientset.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		scale.Spec.Replicas = count
		_, err = m.Clients.Clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
		if err != nil {
			return nil, fmt.Errorf("scale %s: %w", name, err)
		}
	}
	return replicas, nil
}

// DeleteAllHPAs removes every HPA in the namespace.
func (m *Manager) DeleteAllHPAs(ctx context.Context, namespace string) error {
	return m.Clients.Clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
}

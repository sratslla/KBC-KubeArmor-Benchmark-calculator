// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package workload

import (
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// LocustOptions configures the headless Locust deployment.
type LocustOptions struct {
	Users     int32
	SpawnRate int32
	RunTime   string // e.g. "90m"
	Image     string
	DebugUI   bool
	Namespace string
}

// EnsureLocust deploys/updates a headless Locust loadgenerator on the tainted node.
func (m *Manager) EnsureLocust(ctx context.Context, opt LocustOptions) error {
	if opt.Namespace == "" {
		opt.Namespace = "default"
	}
	if opt.Image == "" {
		opt.Image = "sratslla/locust"
	}
	if opt.SpawnRate == 0 {
		opt.SpawnRate = 1
	}
	if opt.RunTime == "" {
		opt.RunTime = "120m"
	}

	args := []string{
		"--host=http://frontend:80",
		"--headless",
		fmt.Sprintf("-u=%d", opt.Users),
		fmt.Sprintf("-r=%d", opt.SpawnRate),
		fmt.Sprintf("--run-time=%s", opt.RunTime),
		"--autostart",
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loadgenerator",
			Namespace: opt.Namespace,
			Labels:    map[string]string{"app": "loadgenerator"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](1),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "loadgenerator"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "loadgenerator"},
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   "8089",
					},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: ptr.To[int64](5),
					RestartPolicy:                 corev1.RestartPolicyAlways,
					NodeSelector:                  map[string]string{"nodetype": "node1"},
					Tolerations: []corev1.Toleration{{
						Key:      "color",
						Operator: corev1.TolerationOpEqual,
						Value:    "blue",
						Effect:   corev1.TaintEffectNoSchedule,
					}},
					Containers: []corev1.Container{{
						Name:            "main",
						Image:           opt.Image,
						ImagePullPolicy: corev1.PullAlways,
						Command:         []string{"locust"},
						Args:            args,
						Env: []corev1.EnvVar{
							{Name: "FRONTEND_ADDR", Value: "frontend:80"},
							{Name: "USERS", Value: strconv.Itoa(int(opt.Users))},
						},
						Ports: []corev1.ContainerPort{{ContainerPort: 8089}},
					}},
				},
			},
		},
	}

	apps := m.Clients.Clientset.AppsV1().Deployments(opt.Namespace)
	existing, err := apps.Get(ctx, "loadgenerator", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = apps.Create(ctx, dep, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create loadgenerator: %w", err)
		}
		fmt.Println("created headless loadgenerator")
	} else if err != nil {
		return err
	} else {
		dep.ResourceVersion = existing.ResourceVersion
		_, err = apps.Update(ctx, dep, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("update loadgenerator: %w", err)
		}
		fmt.Println("updated headless loadgenerator")
	}

	if opt.DebugUI {
		return m.ensureLocustUI(ctx, opt.Namespace)
	}
	return nil
}

func (m *Manager) ensureLocustUI(ctx context.Context, namespace string) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "loadgenerator-service", Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "loadgenerator"},
			Type:     corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       8089,
				TargetPort: intstr.FromInt(8089),
				NodePort:   30001,
			}},
		},
	}
	_, err := m.Clients.Clientset.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

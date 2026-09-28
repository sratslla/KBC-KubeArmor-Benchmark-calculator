// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package ensure

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/cluster"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/waitutil"
)

const (
	kaNamespace    = "kubearmor"
	configName     = "kubearmorconfig-default"
	operatorDeploy = "kubearmor-operator"
	relayDeploy    = "kubearmor-relay"
)

// Manager installs and configures KubeArmor via Helm Operator + KubeArmorConfig.
type Manager struct {
	Clients *cluster.Clients
}

// Status summarizes whether KubeArmor is present and ready.
type Status struct {
	OperatorReady bool
	DaemonReady   bool
	ConfigExists  bool
	Enforcer      string
}

// Detect reports current KubeArmor installation state.
func (m *Manager) Detect(ctx context.Context) (Status, error) {
	var s Status
	_, err := m.Clients.Clientset.AppsV1().Deployments(kaNamespace).Get(ctx, operatorDeploy, metav1.GetOptions{})
	if err == nil {
		s.OperatorReady = true
	} else if !apierrors.IsNotFound(err) {
		// namespace may not exist
		if !apierrors.IsNotFound(err) && !strings.Contains(err.Error(), "not found") {
			// continue with empty status for missing ns
		}
	}

	_, err = m.Clients.Dynamic.Resource(cluster.GVRKubeArmorConfig).Namespace(kaNamespace).Get(ctx, configName, metav1.GetOptions{})
	if err == nil {
		s.ConfigExists = true
	}

	pods, err := m.Clients.Clientset.CoreV1().Pods(kaNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "kubearmor-app=kubearmor",
	})
	if err == nil && len(pods.Items) > 0 {
		ready := 0
		for _, p := range pods.Items {
			if podReady(p) {
				ready++
			}
		}
		s.DaemonReady = ready > 0
	}

	s.Enforcer = m.detectEnforcer(ctx)
	return s, nil
}

func podReady(p corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (m *Manager) detectEnforcer(ctx context.Context) string {
	pods, err := m.Clients.Clientset.CoreV1().Pods(kaNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "kubearmor-app=kubearmor",
	})
	if err != nil || len(pods.Items) == 0 {
		return ""
	}
	for _, p := range pods.Items {
		if v, ok := p.Labels["kubearmor.io/enforcer"]; ok {
			return v
		}
		for _, c := range p.Spec.Containers {
			for _, a := range c.Args {
				if strings.HasPrefix(a, "-lsm=") || strings.HasPrefix(a, "--lsm=") {
					return strings.TrimPrefix(strings.TrimPrefix(a, "--lsm="), "-lsm=")
				}
			}
		}
	}
	return ""
}

// EnsureInstalled detects KubeArmor and installs via Helm + Config if missing.
func (m *Manager) EnsureInstalled(ctx context.Context, lsmOrder string, defaultVisibility string) error {
	st, err := m.Detect(ctx)
	if err != nil {
		return err
	}
	if !st.OperatorReady {
		fmt.Println("KubeArmor operator not found; installing via Helm...")
		if err := m.helmInstallOperator(ctx); err != nil {
			return err
		}
	} else {
		fmt.Println("KubeArmor operator present")
	}

	if err := m.applyKubeArmorConfig(ctx, lsmOrder, defaultVisibility); err != nil {
		return err
	}
	if err := m.PinRelay(ctx); err != nil {
		return err
	}
	return m.WaitReady(ctx, 10*time.Minute)
}

func (m *Manager) helmInstallOperator(ctx context.Context) error {
	steps := [][]string{
		{"helm", "repo", "add", "kubearmor", "https://kubearmor.github.io/charts"},
		{"helm", "repo", "update", "kubearmor"},
		{"helm", "upgrade", "--install", "kubearmor-operator", "kubearmor/kubearmor-operator",
			"-n", kaNamespace, "--create-namespace", "--wait", "--timeout", "5m"},
	}
	for _, args := range steps {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		fmt.Print(string(out))
		if err != nil {
			// repo add may fail if already exists
			if strings.Contains(string(out), "already exists") {
				continue
			}
			return fmt.Errorf("%v: %w\n%s", args, err, out)
		}
	}
	return nil
}

func (m *Manager) applyKubeArmorConfig(ctx context.Context, lsmOrder, defaultVisibility string) error {
	if defaultVisibility == "" {
		defaultVisibility = "process,network"
	}
	if lsmOrder == "" {
		lsmOrder = "bpf,apparmor,selinux"
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "operator.kubearmor.com/v1",
			"kind":       "KubeArmorConfig",
			"metadata": map[string]interface{}{
				"name":      configName,
				"namespace": kaNamespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/name": "kubearmorconfig",
				},
			},
			"spec": map[string]interface{}{
				"defaultCapabilitiesPosture": "audit",
				"defaultFilePosture":         "audit",
				"defaultNetworkPosture":      "audit",
				"defaultVisibility":          defaultVisibility,
				"enableStdOutLogs":           false,
				"enableStdOutAlerts":         false,
				"enableStdOutMsgs":           false,
				"seccompEnabled":             false,
				"kubearmorImage": map[string]interface{}{
					"image":           "kubearmor/kubearmor:stable",
					"imagePullPolicy": "Always",
					"args":            []interface{}{fmt.Sprintf("-lsm=%s", lsmOrder)},
				},
				"kubearmorInitImage": map[string]interface{}{
					"image":           "kubearmor/kubearmor-init:stable",
					"imagePullPolicy": "Always",
				},
				"kubearmorRelayImage": map[string]interface{}{
					"image":           "kubearmor/kubearmor-relay-server",
					"imagePullPolicy": "Always",
				},
				"kubearmorControllerImage": map[string]interface{}{
					"image":           "kubearmor/kubearmor-controller",
					"imagePullPolicy": "Always",
				},
			},
		},
	}

	dr := m.Clients.Dynamic.Resource(cluster.GVRKubeArmorConfig).Namespace(kaNamespace)
	existing, err := dr.Get(ctx, configName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = dr.Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create KubeArmorConfig: %w", err)
		}
		fmt.Println("created KubeArmorConfig")
		return nil
	}
	if err != nil {
		return err
	}
	obj.SetResourceVersion(existing.GetResourceVersion())
	_, err = dr.Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update KubeArmorConfig: %w", err)
	}
	fmt.Printf("updated KubeArmorConfig (lsm=%s visibility=%s)\n", lsmOrder, defaultVisibility)
	return nil
}

// ConfigureLSM updates KubeArmorConfig LSM preference and waits for rollout.
func (m *Manager) ConfigureLSM(ctx context.Context, lsmOrder, defaultVisibility string) error {
	if err := m.applyKubeArmorConfig(ctx, lsmOrder, defaultVisibility); err != nil {
		return err
	}
	// trigger daemon restart observation
	return m.WaitReady(ctx, 8*time.Minute)
}

// PinRelay patches kubearmor-relay onto the tainted Locust node.
func (m *Manager) PinRelay(ctx context.Context) error {
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"nodeSelector": map[string]string{"nodetype": "node1"},
					"tolerations": []map[string]string{{
						"key": "color", "operator": "Equal", "value": "blue", "effect": "NoSchedule",
					}},
				},
			},
		},
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	err = waitutil.Until(ctx, 5*time.Second, 5*time.Minute, func(ctx context.Context) (bool, error) {
		_, err := m.Clients.Clientset.AppsV1().Deployments(kaNamespace).Get(ctx, relayDeploy, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return fmt.Errorf("wait for relay deployment: %w", err)
	}

	_, err = m.Clients.Clientset.AppsV1().Deployments(kaNamespace).Patch(ctx, relayDeploy, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("pin relay: %w", err)
	}
	fmt.Println("pinned kubearmor-relay to nodetype=node1")
	return nil
}

// WaitReady waits until at least one KubeArmor daemon pod is Ready.
func (m *Manager) WaitReady(ctx context.Context, timeout time.Duration) error {
	return waitutil.Until(ctx, 5*time.Second, timeout, func(ctx context.Context) (bool, error) {
		st, err := m.Detect(ctx)
		if err != nil {
			return false, err
		}
		if st.DaemonReady {
			fmt.Printf("KubeArmor ready (enforcer hint=%q)\n", st.Enforcer)
			return true, nil
		}
		return false, nil
	})
}

// VerifyEnforcer returns true if the active enforcer matches want (bpf|apparmor substring).
func (m *Manager) VerifyEnforcer(ctx context.Context, want string) (bool, string) {
	got := m.detectEnforcer(ctx)
	if got == "" {
		// also check daemonset pods node labels / args
		dsList, err := m.Clients.Clientset.AppsV1().DaemonSets(kaNamespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, ds := range dsList.Items {
				if strings.Contains(strings.ToLower(ds.Name), want) {
					return true, ds.Name
				}
				if enforcerFromDS(&ds, want) {
					return true, ds.Name
				}
			}
		}
		return false, got
	}
	return strings.Contains(strings.ToLower(got), strings.ToLower(want)), got
}

func enforcerFromDS(ds *appsv1.DaemonSet, want string) bool {
	sel := ds.Spec.Template.Spec.NodeSelector
	if v, ok := sel["kubearmor.io/enforcer"]; ok && strings.Contains(strings.ToLower(v), want) {
		return true
	}
	for _, c := range ds.Spec.Template.Spec.Containers {
		for _, a := range c.Args {
			if strings.Contains(strings.ToLower(a), want) {
				return true
			}
		}
	}
	return false
}

// Uninstall removes the Helm release (best-effort).
func (m *Manager) Uninstall(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "helm", "uninstall", "kubearmor-operator", "-n", kaNamespace)
	out, err := cmd.CombinedOutput()
	fmt.Print(string(out))
	return err
}

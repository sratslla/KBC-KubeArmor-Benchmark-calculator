// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package cluster

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GVRKubeArmorPolicy is security.kubearmor.com/v1 KubeArmorPolicy.
	GVRKubeArmorPolicy = schema.GroupVersionResource{
		Group:    "security.kubearmor.com",
		Version:  "v1",
		Resource: "kubearmorpolicies",
	}
	// GVRKubeArmorConfig is operator.kubearmor.com/v1 KubeArmorConfig.
	GVRKubeArmorConfig = schema.GroupVersionResource{
		Group:    "operator.kubearmor.com",
		Version:  "v1",
		Resource: "kubearmorconfigs",
	}
)

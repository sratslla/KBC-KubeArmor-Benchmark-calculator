// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package scenario

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadReleaseProfile(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "profiles", "release.yaml")
	p, err := LoadProfile(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.Users != 2000 {
		t.Fatalf("users=%d", p.Users)
	}
	if p.Stabilize != 12*time.Minute {
		t.Fatalf("stabilize=%s", p.Stabilize)
	}
	if p.Samples != 5 {
		t.Fatalf("samples=%d", p.Samples)
	}
	if len(p.Scenarios) < 15 {
		t.Fatalf("expected full matrix, got %d scenarios", len(p.Scenarios))
	}
	if p.Scenarios[0].ID != "baseline" {
		t.Fatalf("first scenario=%s", p.Scenarios[0].ID)
	}
}

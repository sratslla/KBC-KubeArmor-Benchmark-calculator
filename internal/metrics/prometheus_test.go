// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package metrics

import "testing"

func TestAggregateSnapshots(t *testing.T) {
	samples := []Snapshot{
		{Throughput: 100, FrontendCPU: 10},
		{Throughput: 200, FrontendCPU: 20},
	}
	agg := AggregateSnapshots(samples)
	if agg.Count != 2 {
		t.Fatalf("count=%d", agg.Count)
	}
	if agg.Mean.Throughput != 150 {
		t.Fatalf("mean throughput=%v", agg.Mean.Throughput)
	}
	if agg.Mean.FrontendCPU != 15 {
		t.Fatalf("mean frontend cpu=%v", agg.Mean.FrontendCPU)
	}
	if agg.Stdev.Throughput <= 0 {
		t.Fatalf("expected positive stdev, got %v", agg.Stdev.Throughput)
	}
}

func TestAggregateEmpty(t *testing.T) {
	agg := AggregateSnapshots(nil)
	if agg.Count != 0 {
		t.Fatalf("count=%d", agg.Count)
	}
}

func TestAggregateSingle(t *testing.T) {
	agg := AggregateSnapshots([]Snapshot{{Throughput: 42}})
	if agg.Mean.Throughput != 42 {
		t.Fatalf("mean=%v", agg.Mean.Throughput)
	}
	if agg.Stdev.Throughput != 0 {
		t.Fatalf("stdev=%v", agg.Stdev.Throughput)
	}
}

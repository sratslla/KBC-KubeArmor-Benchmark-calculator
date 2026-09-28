// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package metrics

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Client wraps the Prometheus HTTP API.
type Client struct {
	api v1.API
}

// NewClient creates a Prometheus API client for address (e.g. http://host:30000).
func NewClient(address string) (*Client, error) {
	c, err := api.NewClient(api.Config{Address: address})
	if err != nil {
		return nil, err
	}
	return &Client{api: v1.NewAPI(c)}, nil
}

// QueryInstant runs an instant PromQL query and returns the first vector sample value.
func (c *Client) QueryInstant(ctx context.Context, query string) (float64, error) {
	result, warnings, err := c.api.Query(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	if len(warnings) > 0 {
		fmt.Printf("prometheus warnings: %v\n", warnings)
	}
	return vectorValue(result)
}

func vectorValue(result model.Value) (float64, error) {
	switch v := result.(type) {
	case model.Vector:
		if len(v) == 0 {
			return 0, nil
		}
		return float64(v[0].Value), nil
	case *model.Scalar:
		return float64(v.Value), nil
	default:
		return 0, fmt.Errorf("unexpected prometheus result type %s", result.Type())
	}
}

// Snapshot is one metrics sample.
type Snapshot struct {
	Throughput       float64
	FailedRequests   float64
	KubeArmorCPU     float64
	KubeArmorMemory  float64
	RelayCPU         float64
	RelayMemory      float64
	FrontendCPU      float64
	FrontendReplicas float64
	CartCPU          float64
	CartReplicas     float64
	CurrencyCPU      float64
	CurrencyReplicas float64
	LocustUsers      float64
}

// Aggregate is the mean of N snapshots.
type Aggregate struct {
	Mean   Snapshot
	Stdev  Snapshot
	Count  int
	Samples []Snapshot
}

// Queries holds PromQL used for benchmarking.
type Queries struct {
	LocustUsers      string
	Throughput       string
	FailedRequests   string
	KubeArmorCPU     string
	KubeArmorMemory  string
	RelayCPU         string
	RelayMemory      string
	FrontendCPU      string
	FrontendReplicas string
	CartCPU          string
	CartReplicas     string
	CurrencyCPU      string
	CurrencyReplicas string
}

// DefaultQueries returns Guide-aligned PromQL (5m rate windows).
func DefaultQueries() Queries {
	return Queries{
		LocustUsers:    `locust_users{job="locust"}`,
		Throughput:     `avg_over_time(locust_requests_current_rps{job="locust", name="Aggregated"}[1m])`,
		FailedRequests: `locust_requests_num_failures{job="locust", name="Aggregated"}`,
		KubeArmorCPU: `sum(rate(container_cpu_usage_seconds_total{pod=~"kubearmor-.*", container!="", container!="POD", namespace="kubearmor"}[1m])) * 1000` +
			` / count(count by (pod) (container_cpu_usage_seconds_total{pod=~"kubearmor-.*", namespace="kubearmor", container!="", container!="POD"}))`,
		KubeArmorMemory: `sum(container_memory_usage_bytes{pod=~"kubearmor-.*", container!="", container!="POD", namespace="kubearmor"}) / 1024 / 1024` +
			` / count(count by (pod) (container_memory_usage_bytes{pod=~"kubearmor-.*", namespace="kubearmor", container!="", container!="POD"}))`,
		RelayCPU:         `sum(rate(container_cpu_usage_seconds_total{pod=~"kubearmor-relay-.*", container!="", container!="POD", namespace="kubearmor"}[1m])) * 1000`,
		RelayMemory:      `sum(container_memory_usage_bytes{pod=~"kubearmor-relay-.*", container!="", container!="POD", namespace="kubearmor"}) / 1024 / 1024`,
		FrontendCPU:      `sum(rate(container_cpu_usage_seconds_total{pod=~"frontend-.*", container="", namespace="default"}[1m])) * 1000`,
		FrontendReplicas: `kube_deployment_status_replicas{deployment="frontend", namespace="default"}`,
		CartCPU:          `sum(rate(container_cpu_usage_seconds_total{pod=~"cartservice-.*", container="", namespace="default"}[1m])) * 1000`,
		CartReplicas:     `kube_deployment_status_replicas{deployment="cartservice", namespace="default"}`,
		CurrencyCPU:      `sum(rate(container_cpu_usage_seconds_total{pod=~"currencyservice-.*", container="", namespace="default"}[1m])) * 1000`,
		CurrencyReplicas: `kube_deployment_status_replicas{deployment="currencyservice", namespace="default"}`,
	}
}

// Capture takes a single Snapshot.
func (c *Client) Capture(ctx context.Context, q Queries) (Snapshot, error) {
	var s Snapshot
	var err error
	get := func(query string) (float64, error) {
		if query == "" {
			return 0, nil
		}
		return c.QueryInstant(ctx, query)
	}
	if s.LocustUsers, err = get(q.LocustUsers); err != nil {
		return s, fmt.Errorf("locust users: %w", err)
	}
	if s.Throughput, err = get(q.Throughput); err != nil {
		return s, fmt.Errorf("throughput: %w", err)
	}
	if s.FailedRequests, err = get(q.FailedRequests); err != nil {
		return s, fmt.Errorf("failed requests: %w", err)
	}
	if s.KubeArmorCPU, err = get(q.KubeArmorCPU); err != nil {
		return s, fmt.Errorf("kubearmor cpu: %w", err)
	}
	if s.KubeArmorMemory, err = get(q.KubeArmorMemory); err != nil {
		return s, fmt.Errorf("kubearmor memory: %w", err)
	}
	if s.RelayCPU, err = get(q.RelayCPU); err != nil {
		return s, fmt.Errorf("relay cpu: %w", err)
	}
	if s.RelayMemory, err = get(q.RelayMemory); err != nil {
		return s, fmt.Errorf("relay memory: %w", err)
	}
	if s.FrontendCPU, err = get(q.FrontendCPU); err != nil {
		return s, fmt.Errorf("frontend cpu: %w", err)
	}
	if s.FrontendReplicas, err = get(q.FrontendReplicas); err != nil {
		return s, fmt.Errorf("frontend replicas: %w", err)
	}
	if s.CartCPU, err = get(q.CartCPU); err != nil {
		return s, fmt.Errorf("cart cpu: %w", err)
	}
	if s.CartReplicas, err = get(q.CartReplicas); err != nil {
		return s, fmt.Errorf("cart replicas: %w", err)
	}
	if s.CurrencyCPU, err = get(q.CurrencyCPU); err != nil {
		return s, fmt.Errorf("currency cpu: %w", err)
	}
	if s.CurrencyReplicas, err = get(q.CurrencyReplicas); err != nil {
		return s, fmt.Errorf("currency replicas: %w", err)
	}
	return s, nil
}

// SampleWindow soaks then captures n samples at interval and returns their aggregate.
func (c *Client) SampleWindow(ctx context.Context, q Queries, n int, interval time.Duration) (Aggregate, error) {
	if n < 1 {
		n = 1
	}
	samples := make([]Snapshot, 0, n)
	for i := 0; i < n; i++ {
		s, err := c.Capture(ctx, q)
		if err != nil {
			return Aggregate{}, err
		}
		samples = append(samples, s)
		if i < n-1 {
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return Aggregate{}, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return AggregateSnapshots(samples), nil
}

// AggregateSnapshots computes mean and population stdev.
func AggregateSnapshots(samples []Snapshot) Aggregate {
	agg := Aggregate{Count: len(samples), Samples: samples}
	if len(samples) == 0 {
		return agg
	}
	var sum Snapshot
	for _, s := range samples {
		sum = addSnap(sum, s)
	}
	n := float64(len(samples))
	agg.Mean = scaleSnap(sum, 1/n)
	if len(samples) == 1 {
		return agg
	}
	var sq Snapshot
	for _, s := range samples {
		d := subSnap(s, agg.Mean)
		sq = addSnap(sq, mulSnap(d, d))
	}
	agg.Stdev = sqrtSnap(scaleSnap(sq, 1/n))
	return agg
}

func addSnap(a, b Snapshot) Snapshot {
	return Snapshot{
		Throughput: a.Throughput + b.Throughput, FailedRequests: a.FailedRequests + b.FailedRequests,
		KubeArmorCPU: a.KubeArmorCPU + b.KubeArmorCPU, KubeArmorMemory: a.KubeArmorMemory + b.KubeArmorMemory,
		RelayCPU: a.RelayCPU + b.RelayCPU, RelayMemory: a.RelayMemory + b.RelayMemory,
		FrontendCPU: a.FrontendCPU + b.FrontendCPU, FrontendReplicas: a.FrontendReplicas + b.FrontendReplicas,
		CartCPU: a.CartCPU + b.CartCPU, CartReplicas: a.CartReplicas + b.CartReplicas,
		CurrencyCPU: a.CurrencyCPU + b.CurrencyCPU, CurrencyReplicas: a.CurrencyReplicas + b.CurrencyReplicas,
		LocustUsers: a.LocustUsers + b.LocustUsers,
	}
}

func subSnap(a, b Snapshot) Snapshot {
	return Snapshot{
		Throughput: a.Throughput - b.Throughput, FailedRequests: a.FailedRequests - b.FailedRequests,
		KubeArmorCPU: a.KubeArmorCPU - b.KubeArmorCPU, KubeArmorMemory: a.KubeArmorMemory - b.KubeArmorMemory,
		RelayCPU: a.RelayCPU - b.RelayCPU, RelayMemory: a.RelayMemory - b.RelayMemory,
		FrontendCPU: a.FrontendCPU - b.FrontendCPU, FrontendReplicas: a.FrontendReplicas - b.FrontendReplicas,
		CartCPU: a.CartCPU - b.CartCPU, CartReplicas: a.CartReplicas - b.CartReplicas,
		CurrencyCPU: a.CurrencyCPU - b.CurrencyCPU, CurrencyReplicas: a.CurrencyReplicas - b.CurrencyReplicas,
		LocustUsers: a.LocustUsers - b.LocustUsers,
	}
}

func mulSnap(a, b Snapshot) Snapshot {
	return Snapshot{
		Throughput: a.Throughput * b.Throughput, FailedRequests: a.FailedRequests * b.FailedRequests,
		KubeArmorCPU: a.KubeArmorCPU * b.KubeArmorCPU, KubeArmorMemory: a.KubeArmorMemory * b.KubeArmorMemory,
		RelayCPU: a.RelayCPU * b.RelayCPU, RelayMemory: a.RelayMemory * b.RelayMemory,
		FrontendCPU: a.FrontendCPU * b.FrontendCPU, FrontendReplicas: a.FrontendReplicas * b.FrontendReplicas,
		CartCPU: a.CartCPU * b.CartCPU, CartReplicas: a.CartReplicas * b.CartReplicas,
		CurrencyCPU: a.CurrencyCPU * b.CurrencyCPU, CurrencyReplicas: a.CurrencyReplicas * b.CurrencyReplicas,
		LocustUsers: a.LocustUsers * b.LocustUsers,
	}
}

func scaleSnap(a Snapshot, f float64) Snapshot {
	return Snapshot{
		Throughput: a.Throughput * f, FailedRequests: a.FailedRequests * f,
		KubeArmorCPU: a.KubeArmorCPU * f, KubeArmorMemory: a.KubeArmorMemory * f,
		RelayCPU: a.RelayCPU * f, RelayMemory: a.RelayMemory * f,
		FrontendCPU: a.FrontendCPU * f, FrontendReplicas: a.FrontendReplicas * f,
		CartCPU: a.CartCPU * f, CartReplicas: a.CartReplicas * f,
		CurrencyCPU: a.CurrencyCPU * f, CurrencyReplicas: a.CurrencyReplicas * f,
		LocustUsers: a.LocustUsers * f,
	}
}

func sqrtSnap(a Snapshot) Snapshot {
	return Snapshot{
		Throughput: math.Sqrt(a.Throughput), FailedRequests: math.Sqrt(a.FailedRequests),
		KubeArmorCPU: math.Sqrt(a.KubeArmorCPU), KubeArmorMemory: math.Sqrt(a.KubeArmorMemory),
		RelayCPU: math.Sqrt(a.RelayCPU), RelayMemory: math.Sqrt(a.RelayMemory),
		FrontendCPU: math.Sqrt(a.FrontendCPU), FrontendReplicas: math.Sqrt(a.FrontendReplicas),
		CartCPU: math.Sqrt(a.CartCPU), CartReplicas: math.Sqrt(a.CartReplicas),
		CurrencyCPU: math.Sqrt(a.CurrencyCPU), CurrencyReplicas: math.Sqrt(a.CurrencyReplicas),
		LocustUsers: math.Sqrt(a.LocustUsers),
	}
}

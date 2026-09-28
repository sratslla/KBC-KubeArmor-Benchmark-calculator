// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/ensure"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/metrics"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/waitutil"
	"github.com/sratslla/KBC-KubeArmor-Benchmark-calculator/internal/workload"
)

// Profile is the top-level release configuration.
type Profile struct {
	Name            string        `yaml:"name"`
	Users           int32         `yaml:"users"`
	SpawnRate       int32         `yaml:"spawn_rate"`
	CPUPercent      int32         `yaml:"cpu_percent"`
	StabilizeRaw    string        `yaml:"stabilize"`
	SampleEveryRaw  string        `yaml:"sample_every"`
	Stabilize       time.Duration `yaml:"-"`
	Samples         int           `yaml:"samples"`
	SampleEvery     time.Duration `yaml:"-"`
	Namespace       string        `yaml:"namespace"`
	Scenarios       []Scenario    `yaml:"scenarios"`
}

// Scenario describes one measurement case.
type Scenario struct {
	ID                string   `yaml:"id"`
	Section           string   `yaml:"section"`
	Description       string   `yaml:"description"`
	KubeArmor         string   `yaml:"kubearmor"` // off | on
	LSMOrder          string   `yaml:"lsm_order"`
	DefaultVisibility string   `yaml:"default_visibility"`
	Visibility        string   `yaml:"visibility"`
	Policies          []string `yaml:"policies"`
	RequireEnforcer   string   `yaml:"require_enforcer"` // bpf | apparmor
	ClearPolicies     bool     `yaml:"clear_policies"`
}

// Result is one scenario outcome.
type Result struct {
	ID              string            `json:"id"`
	Section         string            `json:"section"`
	Description     string            `json:"description"`
	Skipped         bool              `json:"skipped,omitempty"`
	SkipReason      string            `json:"skip_reason,omitempty"`
	Users           int32             `json:"users"`
	Aggregate       metrics.Aggregate `json:"aggregate"`
	PercentageDrop  float64           `json:"percentage_drop"`
	BaselineRPS     float64           `json:"baseline_rps"`
}

// Checkpoint persists progress for resume.
type Checkpoint struct {
	BaselineRPS float64  `json:"baseline_rps"`
	Completed   []string `json:"completed"`
	Results     []Result `json:"results"`
}

// Runner executes a profile.
type Runner struct {
	Profile      Profile
	Workload     *workload.Manager
	Ensure       *ensure.Manager
	Metrics      *metrics.Client
	Queries      metrics.Queries
	CheckpointPath string
	FromScenario string
}

// LoadProfile reads a YAML profile.
func LoadProfile(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return Profile{}, err
	}
	if p.Namespace == "" {
		p.Namespace = "default"
	}
	if p.Users == 0 {
		p.Users = 2000
	}
	if p.SpawnRate == 0 {
		p.SpawnRate = 1
	}
	if p.CPUPercent == 0 {
		p.CPUPercent = 50
	}
	if p.Samples == 0 {
		p.Samples = 5
	}
	var errDur error
	if p.StabilizeRaw == "" {
		p.Stabilize = 12 * time.Minute
	} else {
		p.Stabilize, errDur = time.ParseDuration(p.StabilizeRaw)
		if errDur != nil {
			return Profile{}, fmt.Errorf("stabilize: %w", errDur)
		}
	}
	if p.SampleEveryRaw == "" {
		p.SampleEvery = time.Minute
	} else {
		p.SampleEvery, errDur = time.ParseDuration(p.SampleEveryRaw)
		if errDur != nil {
			return Profile{}, fmt.Errorf("sample_every: %w", errDur)
		}
	}
	return p, nil
}

// Run executes scenarios, optionally resuming from FromScenario.
func (r *Runner) Run(ctx context.Context) ([]Result, error) {
	cp := r.loadCheckpoint()
	skipUntil := r.FromScenario != ""
	var results []Result
	if len(cp.Results) > 0 && r.FromScenario != "" {
		results = append(results, cp.Results...)
	}
	baselineRPS := cp.BaselineRPS

	completed := map[string]bool{}
	for _, id := range cp.Completed {
		completed[id] = true
	}

	for _, sc := range r.Profile.Scenarios {
		if skipUntil {
			if sc.ID == r.FromScenario {
				skipUntil = false
			} else {
				continue
			}
		}
		if completed[sc.ID] && r.FromScenario == "" {
			continue
		}

		fmt.Printf("\n=== scenario %s (%s) ===\n", sc.ID, sc.Section)
		res, err := r.runOne(ctx, sc, baselineRPS)
		if err != nil {
			return results, fmt.Errorf("scenario %s: %w", sc.ID, err)
		}
		if sc.KubeArmor == "off" && !res.Skipped {
			baselineRPS = res.Aggregate.Mean.Throughput
			res.BaselineRPS = baselineRPS
			res.PercentageDrop = 0
		} else if baselineRPS > 0 && !res.Skipped {
			res.BaselineRPS = baselineRPS
			res.PercentageDrop = ((baselineRPS - res.Aggregate.Mean.Throughput) / baselineRPS) * 100
		}
		results = append(results, res)
		completed[sc.ID] = true
		cp = Checkpoint{BaselineRPS: baselineRPS, Completed: keys(completed), Results: results}
		if err := r.saveCheckpoint(cp); err != nil {
			fmt.Printf("warning: checkpoint save: %v\n", err)
		}
	}
	return results, nil
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			out = append(out, k)
		}
	}
	return out
}

func (r *Runner) runOne(ctx context.Context, sc Scenario, baselineRPS float64) (Result, error) {
	res := Result{ID: sc.ID, Section: sc.Section, Description: sc.Description, Users: r.Profile.Users, BaselineRPS: baselineRPS}

	if sc.ClearPolicies {
		_ = r.Workload.DeleteAllPolicies(ctx, r.Profile.Namespace)
	}

	switch sc.KubeArmor {
	case "off":
		// baseline: assume KA not installed yet
	case "on", "":
		lsm := sc.LSMOrder
		if lsm == "" {
			lsm = "bpf,apparmor,selinux"
		}
		vis := sc.DefaultVisibility
		if vis == "" {
			vis = "process,network"
		}
		if err := r.Ensure.EnsureInstalled(ctx, lsm, vis); err != nil {
			return res, err
		}
		if sc.RequireEnforcer != "" {
			ok, got := r.Ensure.VerifyEnforcer(ctx, sc.RequireEnforcer)
			if !ok {
				res.Skipped = true
				res.SkipReason = fmt.Sprintf("required enforcer %q not available (got %q)", sc.RequireEnforcer, got)
				fmt.Println(res.SkipReason)
				return res, nil
			}
		}
	}

	if sc.Visibility != "" {
		if err := r.Workload.SetVisibility(ctx, r.Profile.Namespace, sc.Visibility); err != nil {
			return res, err
		}
	}

	if len(sc.Policies) > 0 {
		if err := r.Workload.ApplyPolicies(ctx, sc.Policies); err != nil {
			return res, err
		}
	}

	fmt.Printf("soaking %s for stability...\n", r.Profile.Stabilize)
	if err := waitutil.Soak(ctx, r.Profile.Stabilize); err != nil {
		return res, err
	}

	fmt.Printf("sampling %d times every %s...\n", r.Profile.Samples, r.Profile.SampleEvery)
	agg, err := r.Metrics.SampleWindow(ctx, r.Queries, r.Profile.Samples, r.Profile.SampleEvery)
	if err != nil {
		return res, err
	}
	res.Aggregate = agg
	return res, nil
}

func (r *Runner) loadCheckpoint() Checkpoint {
	if r.CheckpointPath == "" {
		return Checkpoint{}
	}
	data, err := os.ReadFile(r.CheckpointPath)
	if err != nil {
		return Checkpoint{}
	}
	var cp Checkpoint
	_ = json.Unmarshal(data, &cp)
	return cp
}

func (r *Runner) saveCheckpoint(cp Checkpoint) error {
	if r.CheckpointPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(r.CheckpointPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.CheckpointPath, data, 0o644)
}

// WaitLocustUsers blocks until locust_users >= target.
func WaitLocustUsers(ctx context.Context, c *metrics.Client, target int32, timeout time.Duration) error {
	q := metrics.DefaultQueries().LocustUsers
	return waitutil.Until(ctx, 10*time.Second, timeout, func(ctx context.Context) (bool, error) {
		v, err := c.QueryInstant(ctx, q)
		if err != nil {
			fmt.Printf("waiting locust users: query err: %v\n", err)
			return false, nil
		}
		fmt.Printf("locust users=%.0f (target=%d)\n", v, target)
		return v >= float64(target), nil
	})
}
